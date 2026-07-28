package main

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// baseline_identity_dispatch_e2e_test.go (ISSUE-046 Phase 3) — the acceptance-
// critical integration-gap closer (CLM-009, ISSUE-045 lesson: records fed straight
// into EnrichViolationIdentity can hide a dispatch-shaped bug). These drive the
// REAL findings dispatch (dispatchPackEngines -> runFindingsEngine -> real SARIF
// parse -> the gate.Violation bridge that now canonicalizes File) over the SAME
// fixture file under BOTH scope shapes and assert one identity.
//
// The tool executable is stood in by scopePathShapingRunner — the standard
// cmd/backstop dispatch-e2e pattern (fixtureRunner et al.). It is NOT a stub of the
// dispatch code under test: runFindingsEngine's scope-branch arg shaping, the real
// SARIF parse, and the real bridge all execute. The runner FAITHFULLY reproduces
// the ONLY behavior this bug turns on — semgrep echoes the path form of the target
// it is pointed at (a directory arg yields the "./"-prefixed walk form; an explicit
// repo-relative file arg yields that exact string) while its content-based
// partialFingerprints stay identical regardless of scope.

// scopePathShapingRunner emits semgrep-style SARIF whose artifactLocation.uri
// mirrors the LAST argument (the scan target) runFindingsEngine appends: an
// ABSOLUTE target (the full-scope projectRoot directory arg) yields the "./"-form
// a directory walk produces; a relative target (the diff-scope explicit file arg)
// is echoed verbatim. The partialFingerprints are content-derived and IDENTICAL
// across both invocations — the faithful model of semgrep's position-independent
// fingerprints.
type scopePathShapingRunner struct {
	relFile string
	calls   [][]string
}

func (r *scopePathShapingRunner) sarifFor(args []string) []byte {
	uri := r.relFile
	if len(args) > 0 {
		target := args[len(args)-1]
		if filepath.IsAbs(target) {
			// Full-scope directory walk: semgrep reports the "./"-prefixed walk form.
			uri = "./" + r.relFile
		} else {
			// Diff-scope explicit file arg: semgrep echoes the given path verbatim.
			uri = target
		}
	}
	// One finding; partialFingerprints are content-based and scope-independent.
	return []byte(fmt.Sprintf(`{"version":"2.1.0","runs":[{"results":[{"ruleId":"semgrep-no-eval","level":"error","message":{"text":"eval usage is forbidden"},"partialFingerprints":{"primaryLocationLineHash":"stable-content-hash-v1"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":%q},"region":{"startLine":7,"snippet":{"text":"eval(x)"}}}}]}]}]}`, uri))
}

func (r *scopePathShapingRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	return r.sarifFor(args), nil
}

func (r *scopePathShapingRunner) RunStdout(_ context.Context, _ string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	return r.sarifFor(args), nil
}

// semgrepDispatchManifest builds an in-memory manifest with a single semgrep rule
// pointing at the engine-dispatch fixture pack's real semgrep/no-eval.yml, so the
// findings dispatch runs through the REAL semgrep EngineBinding (rule-fed, no
// convert).
func semgrepDispatchManifest() *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: "test-org/engine-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "semgrep-no-eval", Engine: "semgrep", RulePath: "semgrep/no-eval.yml", Standard: "x"},
		}}},
	}
}

// dispatchWithTarget runs the real findings dispatch for the semgrep fixture pack
// under the given scope and returns the (single) resulting violation plus the
// target arg the engine was actually pointed at (the last arg runFindingsEngine
// appended). Both returns are meaningful, so no call site discards.
func dispatchWithTarget(t *testing.T, projectRoot, relFile string, scope *gate.GateScope) (gate.Violation, string) {
	t.Helper()
	runner := &scopePathShapingRunner{relFile: relFile}
	violations, err := dispatchPackEngines(
		[]*pack.Manifest{semgrepDispatchManifest()},
		engineDispatchPacksDir(t),
		projectRoot,
		scope,
		runner,
	)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 finding through real dispatch, got %d: %#v", len(violations), violations)
	}
	return violations[0], lastArg(runner.calls)
}

// TestBaselineIdentity_RealDispatch_FullScopeVsDiffScope_ByteIdentical (CLM-009,
// CLM-002): the same finding on the same file, produced by a full-scope directory
// invocation (semgrep gets the projectRoot dir arg -> "./"-form path) and a
// diff-scope explicit-file invocation (semgrep gets the repo-relative file arg),
// must yield BYTE-IDENTICAL EnrichViolationIdentity IdentityHash through the REAL
// dispatch path. FAILS before Phase 1 (raw File differs by scope); passes after.
func TestBaselineIdentity_RealDispatch_FullScopeVsDiffScope_ByteIdentical(t *testing.T) {
	projectRoot := t.TempDir()
	relFile := "pkg/pack/manifest.go"

	fullV, fullTarget := dispatchWithTarget(t, projectRoot, relFile, nil)

	diffScope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{relFile})
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	diffV, diffTarget := dispatchWithTarget(t, projectRoot, relFile, diffScope)

	// Prove the two invocations really were shaped differently (the confound is real,
	// not accidentally identical inputs): full scope points at the ABSOLUTE root dir,
	// diff scope at the explicit relative file.
	if !filepath.IsAbs(fullTarget) {
		t.Fatalf("full-scope invocation must target the projectRoot directory (abs), got %q", fullTarget)
	}
	if diffTarget != relFile {
		t.Fatalf("diff-scope invocation must target the explicit repo-relative file %q, got %q", relFile, diffTarget)
	}
	// And the raw SARIF paths really did differ by scope form.
	if fullV.File != diffV.File {
		t.Fatalf("expected the bridge to canonicalize both scope path forms to one File, got full=%q diff=%q", fullV.File, diffV.File)
	}

	fullID := gate.EnrichViolationIdentity(fullV)
	diffID := gate.EnrichViolationIdentity(diffV)
	if fullID.IdentityHash != diffID.IdentityHash {
		t.Fatalf("real-dispatch identity differs by scope shape:\n full  %s (File %q)\n diff  %s (File %q)", fullID.IdentityHash, fullV.File, diffID.IdentityHash, diffV.File)
	}
}

// TestSarifFingerprint_StableAcrossScopeInvocations (CLM-007): the finding's
// RegionHash (sarifFingerprint over partialFingerprints) is byte-identical across a
// full-scope and a diff-scope invocation of the same rule against the same file —
// the empirical verification the issue demands. If this FAILS, RegionHash is itself
// scope-unstable: that is a distinct follow-up OUT of this plan's File-normalization
// scope (do NOT expand into a fingerprint-scheme rewrite) — STOP and report it.
func TestSarifFingerprint_StableAcrossScopeInvocations(t *testing.T) {
	projectRoot := t.TempDir()
	relFile := "pkg/pack/manifest.go"

	fullV, fullTarget := dispatchWithTarget(t, projectRoot, relFile, nil)
	diffScope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{relFile})
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	diffV, diffTarget := dispatchWithTarget(t, projectRoot, relFile, diffScope)
	if fullTarget == "" || diffTarget == "" {
		t.Fatalf("both invocations must point the engine at a target; got full=%q diff=%q", fullTarget, diffTarget)
	}

	if fullV.RegionHash == "" {
		t.Fatalf("expected a non-empty RegionHash from the semgrep partialFingerprints, got empty")
	}
	if fullV.RegionHash != diffV.RegionHash {
		t.Fatalf("RegionHash is SCOPE-UNSTABLE across invocation shapes: full=%q diff=%q — this is a distinct fingerprint-scheme defect beyond ISSUE-046 File normalization; STOP and file a follow-up rather than rewriting the fingerprint here", fullV.RegionHash, diffV.RegionHash)
	}
}

// TestBaselineCycle_GenerateThenGate_TouchedFileRevoked (CLM-003 / ISSUE-050): the
// real-dispatch generate->gate cycle. Generate a baseline from a full-scope run,
// then CompareBaseline a diff/file-scope run of the same tree against it. The gated
// file IS explicitly touched, so under the strict file-level ratchet (ISSUE-050)
// the pre-existing finding's grandfather is REVOKED and it surfaces as exactly one
// NewViolation through the REAL dispatch path — the end-to-end proof the ratchet
// fires, not just in a hand-aligned unit. Identity stability across scope forms
// (ISSUE-046) is still pinned by FixedViolations == 0: a stable identity means the
// baseline finding MATCHED the current one (not orphaned), whereas a path-form
// mismatch would surface it as both net-new AND fixed. (The pure canonicalization
// guarantee is separately covered by
// TestBaselineIdentity_RealDispatch_FullScopeVsDiffScope_ByteIdentical.)
func TestBaselineCycle_GenerateThenGate_TouchedFileRevoked(t *testing.T) {
	projectRoot := t.TempDir()
	relFile := "pkg/pack/manifest.go"

	// baseline generate (full scope).
	fullV, fullTarget := dispatchWithTarget(t, projectRoot, relFile, nil)
	baseline := &gate.BaselineArtifact{
		SchemaVersion: gate.BaselineSchemaV1,
		Violations:    []gate.Violation{gate.EnrichViolationIdentity(fullV)},
	}

	// gate (diff/file scope) over the same tree — the gated file is touched.
	diffScope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{relFile})
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	diffV, diffTarget := dispatchWithTarget(t, projectRoot, relFile, diffScope)
	if !filepath.IsAbs(fullTarget) || diffTarget != relFile {
		t.Fatalf("cycle must exercise both scope shapes: full=%q (want abs) diff=%q (want %q)", fullTarget, diffTarget, relFile)
	}

	comparison := gate.CompareBaseline([]gate.Violation{diffV}, baseline, gate.BaselineCompareOptions{Scope: diffScope})
	if len(comparison.NewViolations) != 1 {
		t.Fatalf("touched-file pre-existing finding must be revoked as exactly one NEW across the generate->gate cycle: got %d %#v", len(comparison.NewViolations), comparison.NewViolations)
	}
	if len(comparison.FixedViolations) != 0 {
		t.Fatalf("stable identity must NOT orphan the finding as fixed (a path-form mismatch would): got %d %#v", len(comparison.FixedViolations), comparison.FixedViolations)
	}
}

func lastArg(calls [][]string) string {
	if len(calls) == 0 {
		return ""
	}
	last := calls[len(calls)-1]
	if len(last) == 0 {
		return ""
	}
	return last[len(last)-1]
}
