package compile

import (
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func stdRulesArray(t *testing.T) []interface{} {
	t.Helper()

	art := loadStandardArtifact(t)
	rawRules, ok := art.Frontmatter["rules"]
	if !ok {
		t.Fatal("STD-GO-001 missing rules array")
	}
	rules, ok := rawRules.([]interface{})
	if !ok {
		t.Fatal("STD-GO-001 rules is not an array")
	}
	return rules
}

func requireSTDRule(t *testing.T, ruleID string) map[string]interface{} {
	t.Helper()

	rule := findSTDRuleByID(stdRulesArray(t), ruleID)
	if rule == nil {
		t.Fatalf("expected STD-GO-001 to contain rule %s", ruleID)
	}
	return rule
}

func assertRuleFieldEquals(t *testing.T, rule map[string]interface{}, field, want string) {
	t.Helper()
	if got := mapString(rule, field); got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func assertDetectionStrategy(t *testing.T, rule map[string]interface{}, want string) map[string]interface{} {
	t.Helper()

	detection := mapStringMap(rule, "detection")
	if detection == nil {
		t.Fatal("rule missing detection map")
	}
	if got := mapString(detection, "strategy"); got != want {
		t.Fatalf("detection.strategy = %q, want %q", got, want)
	}
	return detection
}

func TestGoStandard_STD_GO001_HasRule_GO004(t *testing.T) {
	rule := requireSTDRule(t, "GO-004")
	assertRuleFieldEquals(t, rule, "name", "no-init-functions")
	assertRuleFieldEquals(t, rule, "category", "structure")
	assertRuleFieldEquals(t, rule, "severity", "error")
	assertRuleFieldEquals(t, rule, "compliance_tier", "baseline")
	assertDetectionStrategy(t, rule, "pattern")
}

func TestGoStandard_STD_GO001_HasRule_GO005(t *testing.T) {
	rule := requireSTDRule(t, "GO-005")
	assertRuleFieldEquals(t, rule, "name", "constructor-injection-required")
	assertRuleFieldEquals(t, rule, "category", "structure")
	assertRuleFieldEquals(t, rule, "severity", "warning")
	assertRuleFieldEquals(t, rule, "compliance_tier", "standard")
	assertDetectionStrategy(t, rule, "pattern")
}

func TestGoStandard_STD_GO001_HasRule_GO006(t *testing.T) {
	rule := requireSTDRule(t, "GO-006")
	assertRuleFieldEquals(t, rule, "name", "structured-logging-required")
	assertRuleFieldEquals(t, rule, "category", "structure")
	assertRuleFieldEquals(t, rule, "severity", "warning")
	assertRuleFieldEquals(t, rule, "compliance_tier", "standard")
	assertDetectionStrategy(t, rule, "pattern")
}

func TestGoStandard_STD_GO001_HasSecurityRules(t *testing.T) {
	cases := []struct {
		id       string
		name     string
		severity string
		tier     string
	}{
		{id: "GO-060", name: "no-hardcoded-credentials", severity: "error", tier: "baseline"},
		{id: "GO-061", name: "no-weak-password-hashing", severity: "error", tier: "baseline"},
		{id: "GO-062", name: "no-sql-concatenation", severity: "error", tier: "baseline"},
		{id: "GO-063", name: "no-sensitive-data-in-logs", severity: "warning", tier: "standard"},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			rule := requireSTDRule(t, tc.id)
			assertRuleFieldEquals(t, rule, "name", tc.name)
			assertRuleFieldEquals(t, rule, "category", "security")
			assertRuleFieldEquals(t, rule, "severity", tc.severity)
			assertRuleFieldEquals(t, rule, "compliance_tier", tc.tier)
			assertDetectionStrategy(t, rule, "pattern")
		})
	}
}

func TestGoStandard_STD_GO001_HasRule_GO010(t *testing.T) {
	rule := requireSTDRule(t, "GO-010")
	assertDetectionStrategy(t, rule, "pattern")
}

func TestGoStandard_STD_GO001_HasRule_GO012(t *testing.T) {
	rule := requireSTDRule(t, "GO-012")
	detection := assertDetectionStrategy(t, rule, "pattern")
	if mapString(detection, "constraint") == "" {
		t.Fatal("GO-012 must include function-length constraint")
	}
}

func TestGoStandard_STD_GO001_HasRule_GO020(t *testing.T) {
	rule := requireSTDRule(t, "GO-020")
	assertDetectionStrategy(t, rule, "delegated")
}

func TestGoStandard_STD_GO001_HasRule_GO021(t *testing.T) {
	rule := requireSTDRule(t, "GO-021")
	assertDetectionStrategy(t, rule, "regex")
}

func TestGoStandard_STD_GO001_HasRule_GO030(t *testing.T) {
	rule := requireSTDRule(t, "GO-030")
	assertDetectionStrategy(t, rule, "metric")
}

func TestGoStandard_STD_GO001_HasRule_GO031(t *testing.T) {
	rule := requireSTDRule(t, "GO-031")
	detection := assertDetectionStrategy(t, rule, "pattern")
	if mapString(detection, "note") == "" {
		t.Fatal("GO-031 must include a detection note")
	}
}

func TestGoStandard_STD_GO001_HasRule_GO040(t *testing.T) {
	rule := requireSTDRule(t, "GO-040")
	assertDetectionStrategy(t, rule, "delegated")
}

func TestGoStandard_STD_GO001_ValidatesAgainstSchema(t *testing.T) {
	art := loadStandardArtifact(t)
	root := goStandardsRepoRoot(t)
	sch, err := schema.LoadArtifactSchema(
		filepath.Join(root, "artifacts", "standard", "v1", "schema.json"),
		filepath.Join(root, "artifacts"),
	)
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}

	result := validate.Standard(art, sch)
	if !result.Pass() {
		t.Fatalf("STD-GO-001 failed schema validation: %+v", result.Violations)
	}
}
