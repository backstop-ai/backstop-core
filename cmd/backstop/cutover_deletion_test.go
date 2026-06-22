package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SPEC-034 phase-3 CUTOVER deletion assertions (REQ-002/REQ-003/REQ-005/REQ-008).
// Each asserts a specific bespoke symbol / path is ACTUALLY GONE — not merely
// bypassed. They are red while the bespoke code still exists and go green only
// after the phase-3 refactors land. Kept disjoint from strangler_guard_test.go
// (D-081 file-disjointness).

func readCheckSource(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "pkg", "check", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading pkg/check/%s: %v", name, err)
	}
	return string(b)
}

// TestCutover_GoShortCircuitRemoved (CLM-004) proves the `if language == "go"`
// short-circuit and goBuiltinExecutors are deleted from registry.go/check.go —
// the short-circuit BRANCH is gone, not merely bypassed.
func TestCutover_GoShortCircuitRemoved(t *testing.T) {
	registry := readCheckSource(t, "registry.go")
	if strings.Contains(registry, `language == "go"`) {
		t.Error("registry.go still contains the `language == \"go\"` short-circuit branch; it must be deleted")
	}
	if containsIdent(registry, "goBuiltinExecutors") {
		t.Error("registry.go still references goBuiltinExecutors; the Go special-case construction must be deleted")
	}
	check := readCheckSource(t, "check.go")
	if containsIdent(check, "goBuiltinExecutors") {
		t.Error("check.go still defines goBuiltinExecutors; it must be deleted")
	}
}

// TestCutover_BespokeExecutorTypesDeleted (CLM-006) proves the bespoke
// lintExecutor, buildExecutor, and testExecutor types are deleted from pkg/check.
func TestCutover_BespokeExecutorTypesDeleted(t *testing.T) {
	for _, name := range []string{"check.go", "registry.go"} {
		src := readCheckSource(t, name)
		for _, typ := range []string{"lintExecutor", "buildExecutor", "testExecutor"} {
			// Catch both the type decl (`type lintExecutor struct`) and any
			// construction (`&lintExecutor{`).
			if strings.Contains(src, "type "+typ+" ") || strings.Contains(src, "&"+typ+"{") || strings.Contains(src, typ+"{") {
				t.Errorf("%s still declares/constructs bespoke executor %q; it must be deleted", name, typ)
			}
		}
	}
}

// TestCutover_GoBuildTestFormatsRemoved (CLM-007) proves parseGoBuildErrors and
// parseGoTestFailures are deleted AND their go-build / go-test named formats are
// removed from formatParsers — the NAMED-FORMAT entries are gone, not just the
// parser funcs.
func TestCutover_GoBuildTestFormatsRemoved(t *testing.T) {
	check := readCheckSource(t, "check.go")
	for _, fn := range []string{"parseGoBuildErrors", "parseGoTestFailures"} {
		if containsIdent(check, fn) {
			t.Errorf("check.go still defines %q; it relocated to the pack convert script (DD-2) and must be deleted", fn)
		}
	}
	parsers := readCheckSource(t, "parsers.go")
	for _, fmtName := range []string{`"go-build"`, `"go-test"`} {
		if strings.Contains(parsers, fmtName) {
			t.Errorf("parsers.go still registers the %s named format in formatParsers; it must be removed", fmtName)
		}
	}
}

// TestCutover_BespokeLintPathRemoved (CLM-018) proves lintExecutor,
// parseGolangciJSON, the golangci-json formatParsers entry, AND the
// version-adaptive flag logic (golangciOutputArgs, golangciMajorVersion,
// golangciVersionRe) are all deleted from pkg/check — the version-flag symbols and
// the named-format entry are gone, not just the executor type.
func TestCutover_BespokeLintPathRemoved(t *testing.T) {
	check := readCheckSource(t, "check.go")
	for _, sym := range []string{
		"lintExecutor", "parseGolangciJSON",
		"golangciOutputArgs", "golangciMajorVersion", "golangciVersionRe",
	} {
		if containsIdent(check, sym) {
			t.Errorf("check.go still references bespoke lint symbol %q; the lint pass runs as the golangci config-file engine now — delete it", sym)
		}
	}
	parsers := readCheckSource(t, "parsers.go")
	if strings.Contains(parsers, `"golangci-json"`) {
		t.Error("parsers.go still registers the golangci-json named format; it must be removed (lint is native v2 SARIF now)")
	}
}

// TestProvision_EnsureSemgrepBespokeInstallRetired (CLM-029) asserts BOTH halves
// so the Sharp-Edge-10 silent gap cannot open: (a) EnsureSemgrep's bespoke install
// ladder (PATH-probe / .backstop/tools / pip-install) is removed from the native
// Run path, AND (b) semgrep REMAINS provisioned through the declared model. NOT a
// bare symbol-absent grep — the provisioning-still-present assertion is mandatory.
func TestProvision_EnsureSemgrepBespokeInstallRetired(t *testing.T) {
	// (a) is subsumed by the outright deletion of pkg/check/semgrep.go (ISSUE-018):
	// the bespoke install ladder cannot survive in a file that no longer exists.
	// That absence is pinned by pkg/check's TestInProcessSemgrepExecutor_Removed,
	// so this test no longer source-greps the deleted file.

	// (b) semgrep REMAINS provisioned through the declared model: its engine
	// binding keeps a pinned Provision record (auto-provisioned), so retiring the
	// ladder did not silently drop semgrep.
	bind := resolveEngineRegistry()["semgrep"]
	if bind.Provision == nil || bind.Provision.Tool != "semgrep" || bind.Provision.Version == "" {
		t.Fatalf("semgrep must remain pinned + auto-provisioned through declared provisioning, got %+v", bind.Provision)
	}
}
