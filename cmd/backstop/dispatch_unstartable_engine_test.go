package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ISSUE-112 CLM-004/005/006 — A PROCESS THAT NEVER STARTED IS NOT A CLEAN SCAN.
//
// The gate dispatch path captured the run error from RunStdout and consulted it
// ONLY under binding.CrashGuard, which every rule-fed findings engine (semgrep,
// ast-grep) leaves false. So a binary that could not be exec'd produced empty
// stdout, the LENIENT SARIF parser read zero findings, and the step reported a
// clean pass. runCoverageEngine was worse in form: it discarded its run error with
// a literal `_ = runErr` on both branches.
//
// "Never started" is TWO Go error shapes, not one, and both are reachable here:
//
//   - `*exec.Error` — exec.Command consults LookPath, and therefore can produce
//     this, ONLY for a BARE command name with no path separator.
//   - `*fs.PathError` with Op == "fork/exec" — a PATH-FUL command never goes
//     through LookPath at all; when its exec fails the error wraps the syscall
//     errno instead.
//
// Path-ful engine commands are real, not hypothetical: the coverage PRODUCER
// branch is always path-ful (filepath.Join under packRoot) and is already
// os.Stat-guarded, so it can NEVER yield an *exec.Error. A narrow *exec.Error-only
// check on that branch would be greenable only by a stub runner returning a
// synthetic error — i.e. vacuously. Every test below therefore drives a REAL
// check.ExecCommandRunner against a REAL failed exec.
//
// SHARED BINDING CONSTRAINTS, each with a silent misattributed failure mode:
//   - NIL Provision, so the allowlist trust gate returns early and the RUN is what
//     fails. (This is correct HERE because these tests call the dispatch functions
//     directly and never pass through provisionEngines.)
//   - NO stdout_artifact. A binary that never ran ALWAYS leaves a declared artifact
//     missing, which hard-errors for its own unrelated reason — the red would read
//     "artifact not produced" rather than "expected an error, got nil", and would
//     keep firing after the fix.
//   - NO absent convert. A declared-but-missing convert hard-errors "missing convert
//     script" before the never-started predicate is consulted.

// unstartableFindingsPack writes a minimal findings pack (pack.yml is implied by
// the in-memory manifest; only the rule file must exist on disk for input
// gathering) and returns the packs dir plus the manifest. command supplies argv[0]
// — the whole variable under test.
func unstartableFindingsPack(t *testing.T, command string, crashGuard bool) (packsDir string, m *pack.Manifest) {
	t.Helper()
	packsDir = t.TempDir()
	packRoot := filepath.Join(packsDir, "acme", "findings")
	mkDirAll(t, filepath.Join(packRoot, "rules"))
	writeFileStr(t, filepath.Join(packRoot, "rules", "r.yml"), "rules: []\n")

	return packsDir, &pack.Manifest{
		NormalizedName: "acme/findings",
		Engines: map[string]pack.EngineSpec{
			"acme-findings": {Binding: engine.EngineBinding{
				Command:   command,
				InputMode: engine.InputModeRuleFlags,
				InputFlag: "--config",
				ScopeKind: engine.ScopeKindFileArgs,
				Category:  engine.EngineCategoryOpinion,
				// nil Provision (trust gate exempt) + no Convert + no StdoutArtifact.
				CrashGuard: crashGuard,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "acme-findings", RulePath: "rules/r.yml", Standard: "x"},
		}}},
	}
}

// writeNonExecutableScript writes a real file with NO exec bit (0o644). Exec'ing a
// path-ful reference to it fails at fork/exec with an *fs.PathError — exactly what
// a pack whose script lost its exec bit in transit produces.
func writeNonExecutableScript(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho should-never-run\n"), 0o644); err != nil {
		t.Fatalf("write non-executable script: %v", err)
	}
	// Guard the fixture's own premise: if this ever became executable the test
	// would silently stop testing the fork/exec shape.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode().Perm()&0o111 != 0 {
		t.Fatalf("fixture invariant: %s must NOT be executable, got mode %v", path, info.Mode().Perm())
	}
	return path
}

// TestDispatch_UnstartableFindingsEngineFailsLoud proves a findings engine whose
// PROCESS NEVER STARTED fails loud rather than reading as a finding-free scan
// (CLM-004), across BOTH real never-started shapes. A test covering only one shape
// would leave half the defect class uncovered — and which half depends on whether
// the pack's command happens to carry a path separator.
func TestDispatch_UnstartableFindingsEngineFailsLoud(t *testing.T) {
	// A fixed nonsense name with NO path separator: exec.Command routes it through
	// LookPath, which misses, yielding *exec.Error. PATH is never mutated and no
	// real tool's absence is assumed.
	const bareAbsent = "backstop-absent-engine-112"

	t.Run("bare absent name (LookPath miss)", func(t *testing.T) {
		packsDir, m := unstartableFindingsPack(t, bareAbsent+" --sarif", false)
		projectRoot := t.TempDir()
		runner := &check.ExecCommandRunner{Dir: projectRoot}

		violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
		assertNeverStartedRefusal(t, err, violations, "acme/findings", bareAbsent)
	})

	t.Run("path-ful non-executable file (fork/exec)", func(t *testing.T) {
		scriptDir := t.TempDir()
		scriptPath := writeNonExecutableScript(t, scriptDir, "engine.sh")
		// An ABSOLUTE command path, mirroring a pack whose binding names a script by
		// path. This never reaches LookPath, so it cannot produce *exec.Error.
		packsDir, m := unstartableFindingsPack(t, scriptPath+" --sarif", false)
		projectRoot := t.TempDir()
		runner := &check.ExecCommandRunner{Dir: projectRoot}

		violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
		assertNeverStartedRefusal(t, err, violations, "acme/findings", scriptPath)
	})
}

// assertNeverStartedRefusal is the shared acceptance shape: a *check.ConfigError
// (the gate's exit-2 class) naming the pack and the declared command, and NO
// violations that could be mistaken for a clean scan. It asserts the error CLASS
// via errors.As rather than matching a Go type name in a string.
func assertNeverStartedRefusal(t *testing.T, err error, violations []gate.Violation, wantPack, wantCommand string) {
	t.Helper()
	if err == nil {
		t.Fatalf("an engine whose process never started must fail loud, got nil with %d violations — that nil IS the vacuous green ISSUE-112 reports", len(violations))
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("a never-started run must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), wantPack) {
		t.Errorf("the refusal must name the pack %q, got: %v", wantPack, err)
	}
	if !strings.Contains(err.Error(), wantCommand) {
		t.Errorf("the refusal must name the declared command %q, got: %v", wantCommand, err)
	}
	if len(violations) != 0 {
		t.Errorf("a refused run must yield NO violations that could read as a completed scan, got %d: %#v", len(violations), violations)
	}
}

// TestDispatch_UnstartableEngineFailsLoudWithoutCrashGuard is ISSUE-112's exact
// shape (CLM-004): the refusal must be INDEPENDENT of CrashGuard. Every rule-fed
// findings engine — semgrep and ast-grep included — leaves CrashGuard false, which
// is precisely why the pre-existing CrashGuard-gated handling never fired for them.
// This assertion is what stops the fix from being folded into the CrashGuard branch.
func TestDispatch_UnstartableEngineFailsLoudWithoutCrashGuard(t *testing.T) {
	const bareAbsent = "backstop-absent-engine-112"
	packsDir, m := unstartableFindingsPack(t, bareAbsent+" --sarif", false)

	// Pin the premise explicitly: if this binding ever gained CrashGuard the test
	// would pass for the pre-existing reason instead of the new one.
	if m.Engines["acme-findings"].Binding.CrashGuard {
		t.Fatal("fixture invariant: CrashGuard must be FALSE — the point is that the refusal does not depend on it")
	}

	projectRoot := t.TempDir()
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	assertNeverStartedRefusal(t, err, violations, "acme/findings", bareAbsent)
}

// TestDispatch_NonZeroExitWithSarifStillReportsFindings is CLM-006, the
// anti-over-correction: an engine that STARTS, exits NON-ZERO, and emits SARIF
// still reports its findings and returns no error. A rule-fed findings engine
// exits non-zero precisely WHEN it has findings, so keying the refusal on
// `runErr != nil` would red the gate on every real finding.
//
// It is the deliberate twin of the path-ful subtest above: the SAME shape of
// command (an absolute path to a script the test wrote), opposite verdict, and the
// ONLY difference is the exec bit. That pairing is what proves the predicate
// discriminates started-vs-never-started rather than path-ful-vs-bare.
func TestDispatch_NonZeroExitWithSarifStillReportsFindings(t *testing.T) {
	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "engine.sh")
	sarif := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"acme"}},"results":[` +
		`{"ruleId":"acme-rule","level":"error","message":{"text":"a real finding"},` +
		`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"main.go"},"region":{"startLine":3}}}]}]}]}`
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\ncat <<'J'\n"+sarif+"\nJ\nexit 1\n"), 0o755); err != nil { // #nosec G306 — the exec bit IS the fixture
		t.Fatalf("write executable script: %v", err)
	}

	packsDir, m := unstartableFindingsPack(t, scriptPath, false)
	projectRoot := t.TempDir()
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("an engine that STARTED and exited non-zero with SARIF must still report findings, got error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected the one finding the engine emitted, got %d: %#v", len(violations), violations)
	}
	if violations[0].Message != "a real finding" {
		t.Errorf("the engine's finding must survive intact, got %q", violations[0].Message)
	}
}

// unstartableCoveragePack builds a coverage pack for the plain-command branch. It
// DECLARES a convert that EXISTS and emits an empty record array: without one,
// runCoverageEngine's own "declares no convert" broken-pack error would fire and
// the pre-fix run would produce a loud error for an unrelated reason, making the
// red unfalsifiable. With it, the pre-fix run reaches the parser, reads zero
// records, and returns cleanly — the vacuous green this test exists to kill.
func unstartableCoveragePack(t *testing.T, command, producerRel string) (packsDir string, m *pack.Manifest) {
	t.Helper()
	packsDir = t.TempDir()
	packRoot := filepath.Join(packsDir, "acme", "coverage")
	mkDirAll(t, filepath.Join(packRoot, "scripts"))
	// 0o755 deliberately: the shared writeFileStr helper writes 0o644, and a convert
	// with no exec bit fails inside the sandbox for its own reason — which would
	// misattribute this test's red away from the never-started refusal under test.
	if err := os.WriteFile(filepath.Join(packRoot, "scripts", "convert.sh"),
		[]byte("#!/bin/sh\ncat >/dev/null\necho '[]'\n"), 0o755); err != nil { // #nosec G306 — the exec bit is required for the convert to run
		t.Fatalf("write convert script: %v", err)
	}

	return packsDir, &pack.Manifest{
		NormalizedName: "acme/coverage",
		Engines: map[string]pack.EngineSpec{
			"acme-cov": {Binding: engine.EngineBinding{
				Command:   command,
				Producer:  producerRel,
				Convert:   "scripts/convert.sh",
				InputMode: engine.InputModeNone,
				ScopeKind: engine.ScopeKindProjectWide,
				GateType:  engine.GateTypeCoverage,
				// nil Provision (trust gate exempt) + no StdoutArtifact.
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "cov", Engine: "acme-cov", Standard: "x"},
		}}},
	}
}

// TestDispatch_UnstartableCoverageEngineFailsLoud is CLM-005 on the PLAIN-COMMAND
// branch: an unstartable coverage tool must not yield an empty record set that
// reads as coverage-clean. Uses the bare-absent shape.
func TestDispatch_UnstartableCoverageEngineFailsLoud(t *testing.T) {
	const bareAbsent = "backstop-absent-coverage-112"
	packsDir, m := unstartableCoveragePack(t, bareAbsent+" --cover", "")
	projectRoot := t.TempDir()
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	records, err := dispatchPackCoverage([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err == nil {
		t.Fatalf("an unstartable coverage engine must fail loud, got nil with %d records — zero records from a tool that never ran reads as coverage-clean", len(records))
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("a never-started coverage run must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "acme/coverage") {
		t.Errorf("the refusal must name the pack, got: %v", err)
	}
	if !strings.Contains(err.Error(), bareAbsent) {
		t.Errorf("the refusal must name the declared command %q, got: %v", bareAbsent, err)
	}
	if len(records) != 0 {
		t.Errorf("a refused coverage run must yield NO records, got %d", len(records))
	}
}

// TestDispatch_UnstartableCoverageProducerFailsLoud is CLM-005 on the declared
// PRODUCER branch — and the reason the predicate cannot be *exec.Error-only.
// producerPath is filepath.Join(packRoot, …) and is already os.Stat-guarded, so the
// script MUST exist on disk (a missing producer is a different, already-loud error)
// and must fail AT EXEC. A real run here can never produce an *exec.Error;
// asserting that type would be unimplementable without a stub runner.
func TestDispatch_UnstartableCoverageProducerFailsLoud(t *testing.T) {
	packsDir, m := unstartableCoveragePack(t, "must-not-run plain-command", "scripts/produce.sh")
	packRoot := filepath.Join(packsDir, "acme", "coverage")
	// Present on disk (clears the os.Stat guard) but NOT executable, so it fails at
	// fork/exec with an *fs.PathError.
	producerPath := writeNonExecutableScript(t, filepath.Join(packRoot, "scripts"), "produce.sh")

	projectRoot := t.TempDir()
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	records, err := dispatchPackCoverage([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err == nil {
		t.Fatalf("an unstartable coverage PRODUCER must fail loud, got nil with %d records", len(records))
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("a never-started producer must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "acme/coverage") {
		t.Errorf("the refusal must name the pack, got: %v", err)
	}
	if !strings.Contains(err.Error(), producerPath) {
		t.Errorf("the refusal must name the producer that could not start (%q), got: %v", producerPath, err)
	}
	if len(records) != 0 {
		t.Errorf("a refused producer run must yield NO records, got %d", len(records))
	}
}
