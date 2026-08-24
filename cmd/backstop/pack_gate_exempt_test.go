package main

import (
	"context"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// TestExemption_EnginePathStampsProjectWideFromExemptProperty proves the engine
// dispatch stamps each produced gate.Violation.ProjectWide from ITS producing
// binding's ExemptFromScopeFilter value — the NEW bridge the engine path never
// had. A go-build (exempt=true) violation arrives with ProjectWide true
// (SPEC-041 CLM-012).
func TestExemption_EnginePathStampsProjectWideFromExemptProperty(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	sandboxRunner := directConvertSandboxRunner(nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner, sandboxRunner)
	violations := result.Violations
	if err != nil {
		t.Fatalf("dispatchPackEngines (build): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected build violations from the convert")
	}
	for _, v := range violations {
		if !v.ProjectWide {
			t.Fatalf("go-build (exempt=true) violation %q (%s) must arrive with ProjectWide=true via the declared bridge (CLM-012)", v.Message, v.File)
		}
	}
	// The stamp is only a bridge if the REAL gate scope decision READS it. Drive the
	// dispatched violations through gate.StepCodeCheckScopedFunc under a diff scope
	// holding NONE of their files: EVERY one must survive, since the ProjectWide
	// stamp is the only thing keeping it — the per-violation half of the claim, which
	// a "some survived" check would not catch.
	step := gate.StepCodeCheckScopedFunc(&fixedChecker{violations: violations}, diffScope("/repo", "cmd/unrelated.go"))
	if kept := step(context.Background()).Violations; len(kept) != len(violations) {
		t.Fatalf("every stamped go-build violation must survive diff-scope filtering on its stamp alone (CLM-012); kept %d of %d", len(kept), len(violations))
	}
}

// TestExemption_BuildBreakUnchangedFileStillRedsDiffScoped proves a go-build break
// in an UNCHANGED file still REDs a diff-scoped gate END-TO-END through the REAL
// filterViolations (pkg/gate/scope.go) because ProjectWide is set — the under-broad
// regression guard (SPEC-041 CLM-013). The breaking files are NOT in the diff scope.
func TestExemption_BuildBreakUnchangedFileStillRedsDiffScoped(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	sandboxRunner := directConvertSandboxRunner(nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner, sandboxRunner)
	violations := result.Violations
	if err != nil {
		t.Fatalf("dispatchPackEngines (build): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected build violations")
	}
	// Diff scope contains ONLY an unrelated changed file — none of the breaking
	// files (pkg/widget, pkg/gadget) are in scope.
	scope := diffScope("/repo", "cmd/main.go")
	step := gate.StepCodeCheckScopedFunc(&fixedChecker{violations: violations}, scope)
	if survived := step(context.Background()).Violations; len(survived) == 0 {
		t.Fatal("an unchanged-file go-build break must SURVIVE diff-scope filtering end-to-end via ProjectWide (CLM-013)")
	}
}

// TestExemption_LintViolationUnchangedFileIsFiltered proves a golangci violation in
// an UNCHANGED file IS scope-filtered out through the real filterViolations —
// exempt_from_scope_filter is false for golangci, so ProjectWide is false and the
// unchanged-file violation is dropped (SPEC-041 CLM-014).
func TestExemption_LintViolationUnchangedFileIsFiltered(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "golangci")
	sandboxRunner := directConvertSandboxRunner(nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"golangci-lint": readFixture(t, "golangci-v2.sarif")}}

	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner, sandboxRunner)
	violations := result.Violations
	if err != nil {
		t.Fatalf("dispatchPackEngines (lint): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected lint violations")
	}
	for _, v := range violations {
		if v.ProjectWide {
			t.Fatalf("golangci violation %q must NOT be ProjectWide (exempt=false) (CLM-014)", v.Message)
		}
	}
	// A diff scope over an unrelated changed file: none of the violation files are in it.
	step := gate.StepCodeCheckScopedFunc(&fixedChecker{violations: violations}, diffScope("/repo", "cmd/unrelated.go"))
	if survived := step(context.Background()).Violations; len(survived) != 0 {
		t.Fatalf("non-exempt unchanged-file lint violations must be filtered out end-to-end, %d survived (CLM-014): %#v", len(survived), survived)
	}
}

// TestExemption_TestFailureUnchangedFileStillRedsDiffScoped proves a go-test failure
// in an UNCHANGED file still REDs a diff-scoped gate END-TO-END through the REAL
// filterViolations (pkg/gate/scope.go) because ProjectWide is set —
// exempt_from_scope_filter is true for go-test, exactly as it is for go-build, so a
// whole-module test failure is never silently discarded because the failing test's
// file sits outside the diff (SPEC-041 CLM-015; ISSUE-129). Structural twin of
// TestExemption_BuildBreakUnchangedFileStillRedsDiffScoped.
func TestExemption_TestFailureUnchangedFileStillRedsDiffScoped(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	sandboxRunner := directConvertSandboxRunner(nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner, sandboxRunner)
	violations := result.Violations
	if err != nil {
		t.Fatalf("dispatchPackEngines (test): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected test violations")
	}
	for _, v := range violations {
		if !v.ProjectWide {
			t.Fatalf("go-test (exempt=true) violation %q (%s) must arrive with ProjectWide=true via the declared bridge (CLM-015)", v.Message, v.File)
		}
	}
	// Diff scope contains ONLY an unrelated changed file — none of the failing test
	// files are in scope.
	scope := diffScope("/repo", "cmd/main.go")
	step := gate.StepCodeCheckScopedFunc(&fixedChecker{violations: violations}, scope)
	if survived := step(context.Background()).Violations; len(survived) == 0 {
		t.Fatal("an unchanged-file go-test failure must SURVIVE diff-scope filtering end-to-end via ProjectWide (CLM-015)")
	}
}

// TestExemption_FindingsViolationUnchangedFileIsFiltered proves a findings
// (semgrep/ast-grep) violation in an UNCHANGED file IS scope-filtered out —
// exempt_from_scope_filter is false/unset for findings engines, so ProjectWide is
// false and the unchanged-file violation is dropped (SPEC-041 CLM-016).
func TestExemption_FindingsViolationUnchangedFileIsFiltered(t *testing.T) {
	// A findings-class engine (exempt false/unset) producing a SARIF finding pinned
	// to an unchanged file. exemptBinding leaves Convert empty so the runner's SARIF
	// is parsed directly.
	installExemptRegistry(t, map[string]engine.EngineBinding{
		"matrix-findings": exemptBinding(engine.GateTypeFindings, false),
	})
	manifests, packsDir := exemptManifest(t, "matrix-findings")
	runner := &matrixRunner{sarif: sarifForFile("pkg/unchanged/dead.go", "no-foo", "foo on unchanged file")}

	violations, err := dispatchPackEngines(manifests, packsDir, "/repo", nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (findings): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected findings violations")
	}
	for _, v := range violations {
		if v.ProjectWide {
			t.Fatalf("findings violation %q must NOT be ProjectWide (exempt false/unset) (CLM-016)", v.Message)
		}
	}
	// A diff scope over an unrelated changed file: none of the violation files are in it.
	step := gate.StepCodeCheckScopedFunc(&fixedChecker{violations: violations}, diffScope("/repo", "cmd/unrelated.go"))
	if survived := step(context.Background()).Violations; len(survived) != 0 {
		t.Fatalf("non-exempt unchanged-file findings violations must be filtered out end-to-end, %d survived (CLM-016): %#v", len(survived), survived)
	}
}
