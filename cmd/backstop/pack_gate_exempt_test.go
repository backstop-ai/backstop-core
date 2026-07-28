package main

import (
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
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
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
}

// TestExemption_BuildBreakUnchangedFileStillRedsDiffScoped proves a go-build break
// in an UNCHANGED file still REDs a diff-scoped gate END-TO-END through the REAL
// filterViolations (pkg/gate/scope.go) because ProjectWide is set — the under-broad
// regression guard (SPEC-041 CLM-013). The breaking files are NOT in the diff scope.
func TestExemption_BuildBreakUnchangedFileStillRedsDiffScoped(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (build): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected build violations")
	}
	// Diff scope contains ONLY an unrelated changed file — none of the breaking
	// files (pkg/widget, pkg/gadget) are in scope.
	scope := diffScope("/repo", "cmd/main.go")
	survived := filterThroughGate(t, scope, violations)
	if len(survived) == 0 {
		t.Fatal("an unchanged-file go-build break must SURVIVE diff-scope filtering end-to-end via ProjectWide (CLM-013)")
	}
}

// TestExemption_LintViolationUnchangedFileIsFiltered proves a golangci violation in
// an UNCHANGED file IS scope-filtered out through the real filterViolations —
// exempt_from_scope_filter is false for golangci, so ProjectWide is false and the
// unchanged-file violation is dropped (SPEC-041 CLM-014).
func TestExemption_LintViolationUnchangedFileIsFiltered(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "golangci")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"golangci-lint": readFixture(t, "golangci-v2.sarif")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
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
	assertAllFilteredWhenUnchanged(t, violations, "CLM-014")
}

// TestExemption_TestViolationUnchangedFileIsFiltered proves a go-test violation in
// an UNCHANGED file IS scope-filtered out — exempt_from_scope_filter is false for
// go-test, so ProjectWide is false and the unchanged-file violation is dropped
// (SPEC-041 CLM-015).
func TestExemption_TestViolationUnchangedFileIsFiltered(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (test): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected test violations")
	}
	for _, v := range violations {
		if v.ProjectWide {
			t.Fatalf("go-test violation %q must NOT be ProjectWide (exempt=false) (CLM-015)", v.Message)
		}
	}
	assertAllFilteredWhenUnchanged(t, violations, "CLM-015")
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
	assertAllFilteredWhenUnchanged(t, violations, "CLM-016")
}

// assertAllFilteredWhenUnchanged drives the violations through the REAL
// filterViolations with a diff scope that contains NONE of their files, asserting
// every non-exempt violation is dropped (the scope-filter half of the matrix).
func assertAllFilteredWhenUnchanged(t *testing.T, violations []gate.Violation, clm string) {
	t.Helper()
	// A diff scope over an unrelated changed file: none of the violation files are in it.
	scope := diffScope("/repo", "cmd/unrelated.go")
	survived := filterThroughGate(t, scope, violations)
	if len(survived) != 0 {
		t.Fatalf("non-exempt unchanged-file violations must be filtered out end-to-end, %d survived (%s): %#v", len(survived), clm, survived)
	}
}
