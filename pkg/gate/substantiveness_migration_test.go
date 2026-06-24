package gate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// substantiveness_migration_test.go proves the Phase-6 deletion removed the baked
// go/parser ANALYZER, not the enforcement: no analyzer symbol survives in pkg/gate, the
// package compiles without them, the "test must be substantive" INVARIANT still fails a
// hollow test via the pack path, and the changed-file SCOPE behavior is preserved.

// stripLineComments removes // line-comment text so the grep-style absence assertions
// match actual CODE (declarations / usages), not the symbol names that appear in the
// deletion-marker comments documenting WHAT was removed. Without this, a comment naming
// the deleted symbol would false-fire the absence check.
func stripLineComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// readGateSource concatenates the comment-stripped non-test .go sources of pkg/gate (the
// gate "source") for the grep-style absence assertions.
func readGateSource(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/gate dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		b.WriteString(stripLineComments(string(data)))
		b.WriteString("\n")
	}
	return b.String()
}

// TestSubstantiveness_BakedGoAnalyzerDeleted (CLM-001) — no go/parser/go/ast
// substantiveness analyzer remains: checkSubstantiveness, hasAssertions,
// assertionSelectors, callsTargetPackage, samePackage, and the StepTestSubstantiveness*
// constructors are ABSENT from the gate source.
func TestSubstantiveness_BakedGoAnalyzerDeleted(t *testing.T) {
	src := readGateSource(t)
	for _, sym := range []string{
		"func checkSubstantiveness",
		"func hasAssertions",
		"assertionSelectors",
		"func callsTargetPackage",
		"func targetPackageName", // the lowercase analyzer helper (relocated as TargetPackageName)
		"func StepTestSubstantivenessFunc",
		"func StepTestSubstantivenessScopedFunc",
	} {
		if strings.Contains(src, sym) {
			t.Errorf("deleted analyzer symbol %q still present in pkg/gate source", sym)
		}
	}
}

// TestSubstantiveness_AnalyzerCoupledTestsDeleted_PackageCompiles (CLM-027) — no test in
// pkg/gate references any deleted analyzer symbol; the analyzer-subject tests are deleted
// and the package compiles (this test compiling+running IS the compile proof).
func TestSubstantiveness_AnalyzerCoupledTestsDeleted_PackageCompiles(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/gate dir: %v", err)
	}
	// The relocated TargetPackageName is allowed; the deleted lowercase analyzer symbols
	// (with a word boundary) are not. We match calls/usages, not the exported relocation.
	banned := []*regexp.Regexp{
		regexp.MustCompile(`\bcheckSubstantiveness\b`),
		regexp.MustCompile(`\bhasAssertions\b`),
		regexp.MustCompile(`\bassertionSelectors\b`),
		regexp.MustCompile(`\bcallsTargetPackage\b`),
		regexp.MustCompile(`\btargetPackageName\b`),
		regexp.MustCompile(`\bStepTestSubstantivenessFunc\b`),
		regexp.MustCompile(`\bStepTestSubstantivenessScopedFunc\b`),
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		// This migration test file documents the banned names in its own assertions;
		// skip self so the regexes don't match the literals here.
		if e.Name() == "substantiveness_migration_test.go" {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		// Strip comments so deletion-marker comments naming the removed symbols don't
		// false-fire — only actual CODE references count.
		code := stripLineComments(string(data))
		for _, re := range banned {
			if re.MatchString(code) {
				t.Errorf("test file %s still references a deleted analyzer symbol matching %s", e.Name(), re)
			}
		}
	}
}

// TestQ1_AssertionVocabularyOnlyInPack_NotBaked (CLM-006) — the assertion vocabulary
// lives ONLY in the pack rule YAML: no hardcoded assertion-selector list (the deleted
// assertionSelectors map, or an inline require/assert/expect selector slice) exists in
// pkg/gate after the analyzer deletion.
func TestQ1_AssertionVocabularyOnlyInPack_NotBaked(t *testing.T) {
	src := readGateSource(t)
	if strings.Contains(src, "assertionSelectors") {
		t.Errorf("the baked assertionSelectors vocabulary map must NOT exist in pkg/gate after deletion")
	}
	// The pack rule YAML carries the vocabulary — confirm it is there (the vocabulary
	// moved, not vanished).
	ruleYAML, err := os.ReadFile(filepath.Join("testdata", "substantiveness-pack", "ast-grep", "hollow-test-go.yml"))
	if err != nil {
		t.Fatalf("reading pack hollow rule: %v", err)
	}
	if !strings.Contains(string(ruleYAML), "require") || !strings.Contains(string(ruleYAML), "assert") {
		t.Errorf("the assertion vocabulary must live in the pack rule YAML; got:\n%s", ruleYAML)
	}
}

// TestSubstantiveness_InvariantSurvivesDeletion_HollowStillFails (CLM-002) — the "test
// must be substantive" INVARIANT survives: a hollow mandated test still fails the
// substantiveness check via the pack path (real ast-grep Q1 finding → IsTestHollow →
// HollowFindingsToViolations raises a test_substantiveness violation).
func TestSubstantiveness_InvariantSurvivesDeletion_HollowStillFails(t *testing.T) {
	requireAstGrep(t)
	packDir := substPackDir(t)
	fixture := filepath.Join(packDir, "fixtures", "go", "hollow_test.go")

	findings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-go.yml", "hollow-test-go", substPackName, fixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (hollow): %v", err)
	}
	hollow, _ := RouteSubstantivenessFindings(findings, substHollowRuleID, substExtractionRuleID)
	violations := HollowFindingsToViolations(hollow)
	if len(violations) == 0 {
		t.Fatalf("the substantiveness invariant must survive deletion: a hollow test must still fail via the pack path; got no violations")
	}
	if violations[0].Rule != StepTestSubstantiveness {
		t.Errorf("the surviving violation must be a test_substantiveness violation; got %q", violations[0].Rule)
	}
}

// TestSubstantiveness_ScopeAwareThroughPackPath_Preserved (CLM-029) — the changed-file
// SCOPE behavior formerly in TestGateSteps_FilterToChangedFiles_TestSubstantiveness is
// PRESERVED through the pack path: an in-scope mandated test's hollow violation is
// raised, while an out-of-scope file's violation is suppressed. The scope filter is
// applied gate-side over the routed hollow findings (the same suppression the re-wired
// step applies), proving scope-aware coverage is not silently dropped.
func TestSubstantiveness_ScopeAwareThroughPackPath_Preserved(t *testing.T) {
	root := "/repo"
	inScope := "/repo/in_scope_test.go"
	outScope := "/repo/out_scope_test.go"

	// Two routed hollow findings, one per file.
	hollow := []Violation{
		{Rule: substHollowRuleID, File: inScope, Message: "test function TestIn has no assertions (hollow) func=TestIn"},
		{Rule: substHollowRuleID, File: outScope, Message: "test function TestOut has no assertions (hollow) func=TestOut"},
	}
	// Build the scope via the constructor so its fileSet (and path normalization
	// against ProjectRoot) is populated exactly as the production gate does.
	scope := newGateScope(root, GateScopeModeDiff, []string{inScope}, nil)

	var kept []Violation
	for _, v := range HollowFindingsToViolations(hollow) {
		if scope.Mode != GateScopeModeAll && v.File != "" && !scope.Contains(v.File) {
			continue
		}
		kept = append(kept, v)
	}

	if len(kept) != 1 {
		t.Fatalf("scope filtering must keep ONLY the in-scope finding; got %d: %#v", len(kept), kept)
	}
	if kept[0].File != inScope {
		t.Errorf("the kept violation must be the in-scope file %q; got %q", inScope, kept[0].File)
	}
}
