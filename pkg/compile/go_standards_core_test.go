package compile

import (
	"strings"
	"testing"
)

func loadCoreRules(t *testing.T) []map[string]interface{} {
	t.Helper()
	root := goStandardsRepoRoot(t)
	return loadSemgrepRulesFile(t, root+"/standards/go/rules/core/go-core.yml")
}

func TestGoStandard_CoreRuleExists_GO003(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.no-global-mutable-state")
	if rule == nil {
		t.Fatal("expected GO-003 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO004(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.no-init-functions")
	if rule == nil {
		t.Fatal("expected GO-004 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO005(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.constructor-injection")
	if rule == nil {
		t.Fatal("expected GO-005 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO006(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.structured-logging")
	if rule == nil {
		t.Fatal("expected GO-006 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO010(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.no-ignored-errors")
	if rule == nil {
		t.Fatal("expected GO-010 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO011(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.error-wrapping-required")
	if rule == nil {
		t.Fatal("expected GO-011 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO012(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.no-naked-returns")
	if rule == nil {
		t.Fatal("expected GO-012 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO013(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.no-panic-in-library-code")
	if rule == nil {
		t.Fatal("expected GO-013 semgrep rule")
	}
}

func TestGoStandard_CoreRuleExists_GO021(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.error-type-suffix")
	if rule == nil {
		t.Fatal("expected GO-021 semgrep rule")
	}
	if rule["pattern-regex"] == nil {
		t.Fatal("GO-021 must use pattern-regex")
	}
}

func TestGoStandard_CoreRuleMetadata(t *testing.T) {
	for _, rule := range loadCoreRules(t) {
		backstop := mapStringMap(mapStringMap(rule, "metadata"), "backstop")
		if backstop == nil {
			t.Fatalf("rule %q missing metadata.backstop", mapString(rule, "id"))
		}
		if mapString(backstop, "category") == "" {
			t.Fatalf("rule %q missing metadata.backstop.category", mapString(rule, "id"))
		}
		if mapString(backstop, "rule_id") == "" {
			t.Fatalf("rule %q missing metadata.backstop.rule_id", mapString(rule, "id"))
		}
		if mapString(backstop, "compliance_tier") == "" {
			t.Fatalf("rule %q missing metadata.backstop.compliance_tier", mapString(rule, "id"))
		}
	}
}

func TestGoStandard_CoreRule_GO012_HasNote(t *testing.T) {
	rule := findRuleByID(loadCoreRules(t), "go.core.no-naked-returns")
	if rule == nil {
		t.Fatal("expected GO-012 semgrep rule")
	}
	backstop := mapStringMap(mapStringMap(rule, "metadata"), "backstop")
	if backstop == nil {
		t.Fatal("GO-012 missing metadata.backstop")
	}
	note := mapString(backstop, "note")
	if note == "" || !strings.Contains(strings.ToLower(note), "function-length") {
		t.Fatalf("GO-012 note should mention function-length limitation, got %q", note)
	}
}

func TestGoStandard_STD_GO001_GO020_HasDelegationNote(t *testing.T) {
	rule := requireSTDRule(t, "GO-020")
	detection := mapStringMap(rule, "detection")
	if detection == nil {
		t.Fatal("GO-020 missing detection block")
	}
	if got := mapString(detection, "strategy"); got != "delegated" {
		t.Fatalf("GO-020 strategy = %q, want delegated", got)
	}
	note := mapString(detection, "note")
	if !strings.Contains(strings.ToLower(note), "golangci-lint") {
		t.Fatalf("GO-020 note should mention golangci-lint, got %q", note)
	}
}

func TestGoStandard_GO040_NotInSemgrepYAML(t *testing.T) {
	for _, path := range allSemgrepRuleFiles(t) {
		for _, rule := range loadSemgrepRulesFile(t, path) {
			backstop := mapStringMap(mapStringMap(rule, "metadata"), "backstop")
			if mapString(backstop, "rule_id") == "GO-040" {
				t.Fatalf("GO-040 should not appear in semgrep yaml, found in %s", path)
			}
		}
	}
}
