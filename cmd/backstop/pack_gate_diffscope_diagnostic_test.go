package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// diagInitCommitRepo initializes a git repo in dir and commits every file already
// on disk, so a subsequently-planted untracked file becomes the SOLE entry the
// diff-scope resolver appends via `git ls-files --others`. It is a diagnostic
// scaffold: it does not touch any production file.
func diagInitCommitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
		{"add", "-A"},
		{"commit", "-m", "baseline"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// diagLintSarif returns a SARIF log carrying TWO findings from one project-wide
// rule id, one pinned to fileA and one to fileB — modeling golangci reporting
// findings across the whole repo (both an in-scope changed file and an
// out-of-scope untouched file) in a single `./...` run.
func diagLintSarif(fileA, fileB string) []byte {
	result := func(f string) string {
		return `{"ruleId":"unused","message":{"text":"declared but not used in ` + f + `"},` +
			`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"` + f + `"}}}]}`
	}
	return []byte(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"golangci-lint","rules":[{"id":"unused"}]}},"results":[` +
		result(fileA) + `,` + result(fileB) + `]}]}`)
}

// TestDiag_PackEngines_DispatchOutputRetainsOutOfScopeNonExemptViolation is the
// ISSUE-070 diagnostic probe (TASK-001, CLM-002/CLM-007). It pins the EXACT root
// cause WITH LOGGED EVIDENCE before the fix, and — because the fix lives in the
// STEP (packValidatorStep), not in dispatch — it stays green after the fix.
//
// It drives the REAL dispatch path (dispatchPackEngines) for a project-wide,
// NON-exempt engine binding shaped like golangci (ScopeKindProjectWide,
// ProjectTarget "./...", ExemptFromScopeFilter==false) whose canned SARIF reports
// findings on BOTH an in-scope changed file AND an out-of-scope untouched file,
// plus one finding from an EXEMPT (go-build-shaped, ExemptFromScopeFilter==true)
// binding on an out-of-scope file.
//
// The durable root-cause facts it asserts:
//   - the raw dispatch output CONTAINS the out-of-scope non-exempt violation —
//     proving the leak originates from the ABSENT step-level filter, not dispatch
//     or a wrong ProjectWide stamp (CLM-007);
//   - the out-of-scope non-exempt violation's stamped File is already repo-relative
//     and scope.Contains returns FALSE for it while TRUE for the in-scope file —
//     the normalization hypothesis is a red herring: once the filter is CALLED,
//     membership resolves correctly (CLM-002 evidence);
//   - the out-of-scope EXEMPT violation carries ProjectWide==true, ruling out the
//     "wrongly stamped ProjectWide" hypothesis (non-exempt engines stamp false,
//     exempt ones true).
func TestDiag_PackEngines_DispatchOutputRetainsOutOfScopeNonExemptViolation(t *testing.T) {
	projectRoot := t.TempDir()
	inScope := filepath.Join("pkg", "changed", "changed.go")
	outScope := filepath.Join("pkg", "untouched", "untouched.go")
	exemptOut := filepath.Join("pkg", "broken", "build_break.go")

	// Commit both source files as the baseline, then MODIFY the in-scope file so it
	// is the only entry a diff-scope run reports.
	for _, f := range []string{inScope, outScope} {
		abs := filepath.Join(projectRoot, f)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("package p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diagInitCommitRepo(t, projectRoot)
	if err := os.WriteFile(filepath.Join(projectRoot, inScope), []byte("package p\n\nfunc Changed() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A genuine GateScopeModeDiff scope: only the modified in-scope file is present.
	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope diff: %v", err)
	}
	if scope.Mode != gate.GateScopeModeDiff {
		t.Fatalf("expected a diff-mode scope, got %q", scope.Mode)
	}
	if !scope.Contains(inScope) {
		t.Fatalf("the modified in-scope file %q must be in the diff scope; got %#v", inScope, scope.Files)
	}
	if scope.Contains(outScope) {
		t.Fatalf("the untouched file %q must NOT be in the diff scope; got %#v", outScope, scope.Files)
	}

	// Install a golangci-shaped NON-exempt project-wide engine and a go-build-shaped
	// EXEMPT project-wide engine. ProjectWide is stamped PER-VIOLATION from each
	// binding's ExemptFromScopeFilter, so the two engines partition the exempt axis.
	installExemptRegistry(t, map[string]engine.EngineBinding{
		"diag-lint":  exemptBinding(engine.GateTypeLint, false), // ExemptFromScopeFilter=false -> ProjectWide=false
		"diag-build": exemptBinding(engine.GateTypeBuild, true),  // ExemptFromScopeFilter=true  -> ProjectWide=true
	})

	// Dispatch the non-exempt lint engine: its SARIF reports on BOTH the in-scope
	// changed file and the out-of-scope untouched file (one `./...` run).
	lintManifests, lintPacksDir := exemptManifest(t, "diag-lint")
	lintRunner := &matrixRunner{sarif: diagLintSarif(inScope, outScope)}
	lintViolations, err := dispatchPackEngines(lintManifests, lintPacksDir, projectRoot, scope, lintRunner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (lint): %v", err)
	}

	// Dispatch the exempt build engine: its SARIF reports on an out-of-scope file.
	buildManifests, buildPacksDir := exemptManifest(t, "diag-build")
	buildRunner := &matrixRunner{sarif: sarifForFile(exemptOut, "go-build", "undefined: Frobnicate")}
	buildViolations, err := dispatchPackEngines(buildManifests, buildPacksDir, projectRoot, scope, buildRunner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (build): %v", err)
	}

	// ── EVIDENCE (the founder-mandated log) ────────────────────────────────────
	t.Logf("scope.Mode=%s scope.Files=%#v", scope.Mode, scope.Files)
	for _, v := range append(append([]gate.Violation(nil), lintViolations...), buildViolations...) {
		t.Logf("dispatched violation: File=%q ProjectWide=%t scope.Contains=%t Rule=%q",
			v.File, v.ProjectWide, scope.Contains(v.File), v.Rule)
	}

	// ── CLM-007: the raw dispatch output RETAINS the out-of-scope non-exempt
	// violation (dispatch does NOT filter — the leak is the absent step filter). ──
	var inScopeLint, outScopeLint *gate.Violation
	for i := range lintViolations {
		switch lintViolations[i].File {
		case inScope:
			inScopeLint = &lintViolations[i]
		case outScope:
			outScopeLint = &lintViolations[i]
		}
	}
	if outScopeLint == nil {
		t.Fatalf("dispatch output must RETAIN the out-of-scope non-exempt violation (proving the leak is the absent STEP filter, not dispatch); got %#v", lintViolations)
	}
	if inScopeLint == nil {
		t.Fatalf("dispatch output must include the in-scope violation too (both come from the one project-wide run); got %#v", lintViolations)
	}

	// ── CLM-002 evidence: the out-of-scope non-exempt violation's stamped File is
	// already repo-relative (equal to the SARIF uri, not an absolute engine path),
	// and scope.Contains resolves membership correctly on BOTH sides — FALSE for the
	// out-of-scope file, TRUE for the in-scope one. The normalization hypothesis is a
	// red herring: once the filter is CALLED, membership matches. ──────────────────
	if outScopeLint.File != outScope {
		t.Errorf("stamped File must already be the repo-relative form %q (normalization is not the culprit), got %q", outScope, outScopeLint.File)
	}
	if scope.Contains(outScopeLint.File) {
		t.Errorf("scope.Contains must return FALSE for the out-of-scope violation %q; if the STEP called the filter it would be dropped", outScopeLint.File)
	}
	if !scope.Contains(inScopeLint.File) {
		t.Errorf("scope.Contains must return TRUE for the in-scope violation %q so a filter would KEEP it (no vacuous green)", inScopeLint.File)
	}

	// ── Rules out hypothesis (b): the non-exempt engine's violations are stamped
	// ProjectWide==false; the exempt engine's are stamped ProjectWide==true. ────────
	for _, v := range lintViolations {
		if v.ProjectWide {
			t.Errorf("a NON-exempt (golangci-shaped) engine's violation must carry ProjectWide=false, got true for %q", v.File)
		}
	}
	if len(buildViolations) == 0 {
		t.Fatal("expected the exempt engine to emit its out-of-scope build violation")
	}
	for _, v := range buildViolations {
		if !v.ProjectWide {
			t.Errorf("an EXEMPT (go-build-shaped) engine's violation must carry ProjectWide=true, got false for %q", v.File)
		}
		if scope.Contains(v.File) {
			t.Errorf("the exempt violation's file %q was expected out-of-scope (its ProjectWide=true is why it still reds, not scope membership)", v.File)
		}
	}
}
