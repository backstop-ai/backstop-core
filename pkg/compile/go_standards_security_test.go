package compile

import (
	"testing"
)

func loadSecurityRules(t *testing.T) []map[string]interface{} {
	t.Helper()
	root := goStandardsRepoRoot(t)
	return loadSemgrepRulesFile(t, root+"/standards/go/rules/security/go-security.yml")
}

func TestGoStandard_SecurityRuleExists_GO060(t *testing.T) {
	rule := findRuleByID(loadSecurityRules(t), "go.security.no-hardcoded-credentials")
	if rule == nil {
		t.Fatal("expected GO-060 semgrep rule")
	}
}

func TestGoStandard_SecurityRuleExists_GO061(t *testing.T) {
	rule := findRuleByID(loadSecurityRules(t), "go.security.no-weak-password-hashing")
	if rule == nil {
		t.Fatal("expected GO-061 semgrep rule")
	}
}

func TestGoStandard_SecurityRuleExists_GO062(t *testing.T) {
	rule := findRuleByID(loadSecurityRules(t), "go.security.no-sql-concatenation")
	if rule == nil {
		t.Fatal("expected GO-062 semgrep rule")
	}
}

func TestGoStandard_SecurityRuleExists_GO063(t *testing.T) {
	rule := findRuleByID(loadSecurityRules(t), "go.security.no-sensitive-data-in-logs")
	if rule == nil {
		t.Fatal("expected GO-063 semgrep rule")
	}
}

func TestGoStandard_SecurityRuleMetadata_ComplianceRefs(t *testing.T) {
	for _, rule := range loadSecurityRules(t) {
		backstop := mapStringMap(mapStringMap(rule, "metadata"), "backstop")
		if backstop == nil {
			t.Fatalf("rule %q missing metadata.backstop", mapString(rule, "id"))
		}
		refsRaw, ok := backstop["references"].([]interface{})
		if !ok || len(refsRaw) == 0 {
			t.Fatalf("rule %q missing CWE/OWASP references", mapString(rule, "id"))
		}
	}
}
