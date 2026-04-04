package compile

import (
	"strings"
	"testing"
)

func loadTestRules(t *testing.T) []map[string]interface{} {
	t.Helper()
	root := goStandardsRepoRoot(t)
	return loadSemgrepRulesFile(t, root+"/standards/go/rules/test/go-test.yml")
}

func TestGoStandard_TestRuleExists_GO032(t *testing.T) {
	rule := findRuleByID(loadTestRules(t), "go.test.no-time-sleep-in-tests")
	if rule == nil {
		t.Fatal("expected GO-032 semgrep rule")
	}
}

func TestGoStandard_TestRule_GO032_ScopedToTestFiles(t *testing.T) {
	rule := findRuleByID(loadTestRules(t), "go.test.no-time-sleep-in-tests")
	if rule == nil {
		t.Fatal("expected GO-032 semgrep rule")
	}
	paths := mapStringMap(rule, "paths")
	if paths == nil {
		t.Fatal("GO-032 missing paths block")
	}
	includeRaw, ok := paths["include"].([]interface{})
	if !ok || len(includeRaw) == 0 {
		t.Fatal("GO-032 paths.include must be a non-empty array")
	}
	foundTestGlob := false
	for _, item := range includeRaw {
		if s, ok := item.(string); ok && strings.Contains(s, "_test.go") {
			foundTestGlob = true
			break
		}
	}
	if !foundTestGlob {
		t.Fatalf("GO-032 paths.include must scope to test files, got %v", includeRaw)
	}
}

func TestGoStandard_TestRule_GO030_HasNote(t *testing.T) {
	rule := requireSTDRule(t, "GO-030")
	detection := mapStringMap(rule, "detection")
	if detection == nil {
		t.Fatal("GO-030 missing detection block")
	}
	if got := mapString(detection, "strategy"); got != "metric" {
		t.Fatalf("GO-030 strategy = %q, want metric", got)
	}
	note := strings.ToLower(mapString(detection, "note"))
	if note == "" || !strings.Contains(note, "metric") {
		t.Fatalf("GO-030 note must explain metric limitation, got %q", note)
	}
}

func TestGoStandard_TestRule_GO031_HasNote(t *testing.T) {
	rule := requireSTDRule(t, "GO-031")
	detection := mapStringMap(rule, "detection")
	if detection == nil {
		t.Fatal("GO-031 missing detection block")
	}
	if got := mapString(detection, "strategy"); got != "pattern" {
		t.Fatalf("GO-031 strategy = %q, want pattern", got)
	}
	note := strings.ToLower(mapString(detection, "note"))
	if note == "" || !strings.Contains(note, "custom analysis") {
		t.Fatalf("GO-031 note must explain custom analysis requirement, got %q", note)
	}
}
