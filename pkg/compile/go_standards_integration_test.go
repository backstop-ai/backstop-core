package compile

import (
	"os"
	"path/filepath"
	"testing"
)

func compileSTDGO001(t *testing.T) *CompileResult {
	t.Helper()

	root := goStandardsRepoRoot(t)
	standardPath := filepath.Join(root, "standards", "go", "STD-GO-001-go-code-standards.standard.md")

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	res, err := Compile(standardPath, CompileOptions{OutputDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compile standard: %v", err)
	}
	return res
}

func manifestRuleByID(manifest *EnforcementManifest, id string) *ManifestRule {
	for i := range manifest.Rules {
		if manifest.Rules[i].ID == id {
			return &manifest.Rules[i]
		}
	}
	return nil
}

func semgrepRuleByID(rules []SemgrepRule, id string) *SemgrepRule {
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i]
		}
	}
	return nil
}

func nativeRuleByID(rules []NativeCheck, id string) *NativeCheck {
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i]
		}
	}
	return nil
}

func TestGoStandard_PackCompile_ProducesManifest(t *testing.T) {
	res := compileSTDGO001(t)
	if res.Manifest == nil {
		t.Fatal("expected compile manifest")
	}
	if res.Manifest.Standard != "STD-GO-001" {
		t.Fatalf("manifest standard = %q, want STD-GO-001", res.Manifest.Standard)
	}
	if len(res.Manifest.Rules) == 0 {
		t.Fatal("expected non-empty manifest rules")
	}
}

func TestGoStandard_PackCompile_SemgrepIncludesPatternRules(t *testing.T) {
	res := compileSTDGO001(t)
	for _, id := range []string{
		"GO-003", "GO-004", "GO-005", "GO-006", "GO-010", "GO-011", "GO-012", "GO-013",
		"GO-021", "GO-032", "GO-060", "GO-061", "GO-062", "GO-063",
	} {
		if semgrepRuleByID(res.SemgrepRules, id) == nil {
			t.Fatalf("expected semgrep rule %s", id)
		}
	}
}

func TestGoStandard_PackCompile_NativeIncludesMetricRules(t *testing.T) {
	res := compileSTDGO001(t)
	for _, id := range []string{"GO-001", "GO-002", "GO-030"} {
		if nativeRuleByID(res.NativeChecks, id) == nil {
			t.Fatalf("expected native metric rule %s", id)
		}
	}
}

func TestGoStandard_PackCompile_DelegatedNotInSemgrep(t *testing.T) {
	res := compileSTDGO001(t)
	for _, id := range []string{"GO-020", "GO-040", "GO-051"} {
		mr := manifestRuleByID(res.Manifest, id)
		if mr == nil {
			t.Fatalf("expected delegated rule %s in manifest", id)
		}
		if mr.Enforcement != "delegated" {
			t.Fatalf("manifest rule %s enforcement = %q, want delegated", id, mr.Enforcement)
		}
		if semgrepRuleByID(res.SemgrepRules, id) != nil {
			t.Fatalf("delegated rule %s must not be in semgrep output", id)
		}
	}
}

func TestGoStandard_StrategyMapping_PatternToSemgrep(t *testing.T) {
	res := compileSTDGO001(t)
	rule := semgrepRuleByID(res.SemgrepRules, "GO-003")
	if rule == nil {
		t.Fatal("expected GO-003 semgrep rule")
	}
	if rule.Pattern == "" {
		t.Fatal("GO-003 should map to semgrep pattern field")
	}
}

func TestGoStandard_StrategyMapping_MetricToNative(t *testing.T) {
	res := compileSTDGO001(t)
	rule := nativeRuleByID(res.NativeChecks, "GO-001")
	if rule == nil {
		t.Fatal("expected GO-001 native check")
	}
	if rule.Metric == "" || rule.Threshold == nil {
		t.Fatalf("GO-001 native check missing metric/threshold: %+v", *rule)
	}
}

func TestGoStandard_StrategyMapping_RegexToSemgrep(t *testing.T) {
	res := compileSTDGO001(t)
	rule := semgrepRuleByID(res.SemgrepRules, "GO-021")
	if rule == nil {
		t.Fatal("expected GO-021 semgrep rule")
	}
	if rule.PatternRegex == "" {
		t.Fatal("GO-021 should map to pattern-regex")
	}
}

func TestGoStandard_StrategyMapping_DelegatedToManifest(t *testing.T) {
	res := compileSTDGO001(t)
	rule := manifestRuleByID(res.Manifest, "GO-020")
	if rule == nil {
		t.Fatal("expected GO-020 manifest rule")
	}
	if rule.Enforcement != "delegated" {
		t.Fatalf("GO-020 enforcement = %q, want delegated", rule.Enforcement)
	}
	if rule.DelegatedTo == nil || rule.DelegatedTo.Tool == "" || rule.DelegatedTo.Rule == "" {
		t.Fatalf("GO-020 delegated target incomplete: %+v", rule.DelegatedTo)
	}
}

func TestGoStandard_StrategyMapping_AdvisoryToManifest(t *testing.T) {
	res := compileSTDGO001(t)

	// GO-031 (table-driven-tests) is advisory: has strategy+note but no semgrep pattern.
	mr := manifestRuleByID(res.Manifest, "GO-031")
	if mr == nil {
		t.Fatal("GO-031 not found in manifest")
	}
	if mr.Enforcement != "advisory" {
		t.Errorf("GO-031 enforcement = %q, want advisory", mr.Enforcement)
	}

	// Advisory rules must NOT appear in semgrep output.
	sr := semgrepRuleByID(res.SemgrepRules, "go.testing.table-driven-tests")
	if sr != nil {
		t.Error("advisory rule GO-031 should not appear in semgrep rules")
	}
}
