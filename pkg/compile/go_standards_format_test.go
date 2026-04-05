package compile

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type semgrepRunResult struct {
	Results []map[string]interface{} `json:"results"`
}

func allSemgrepRules(t *testing.T) []map[string]interface{} {
	t.Helper()
	files := allSemgrepRuleFiles(t)
	rules := make([]map[string]interface{}, 0)
	for _, file := range files {
		rules = append(rules, loadSemgrepRulesFile(t, file)...)
	}
	return rules
}

func fixtureNames(t *testing.T, kind string) []string {
	t.Helper()
	root := goStandardsRepoRoot(t)
	dir := filepath.Join(root, "standards", "go", "testdata", kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir %q: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		names = append(names, e.Name())
	}
	return names
}

func hasFixtureForRule(ruleID string, names []string) bool {
	prefix := strings.ToLower(strings.ReplaceAll(ruleID, "GO-", "go-")) + "-"
	for _, n := range names {
		if strings.HasPrefix(n, prefix) {
			return true
		}
	}
	return false
}

func semgrepOutputForFile(t *testing.T, filePath string) semgrepRunResult {
	t.Helper()
	root := goStandardsRepoRoot(t)
	if _, err := exec.LookPath("semgrep"); err != nil {
		t.Skip("semgrep not installed in environment")
	}

	cmd := exec.Command(
		"semgrep",
		"--config", filepath.Join(root, "standards", "go", "rules"),
		"--json",
		"--quiet",
		filePath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("semgrep failed for %s: %v\n%s", filePath, err, string(out))
	}

	var parsed semgrepRunResult
	jsonStart := strings.IndexByte(string(out), '{')
	if jsonStart < 0 {
		t.Fatalf("semgrep output for %s did not contain json: %s", filePath, string(out))
	}
	if err := json.Unmarshal(out[jsonStart:], &parsed); err != nil {
		t.Fatalf("parse semgrep json for %s: %v\n%s", filePath, err, string(out))
	}
	return parsed
}

func TestGoStandard_AllRulesValidYAML(t *testing.T) {
	for _, path := range allSemgrepRuleFiles(t) {
		rules := loadSemgrepRulesFile(t, path)
		if len(rules) == 0 {
			t.Fatalf("rule file %q has empty rules array", path)
		}
	}
}

func TestGoStandard_AllRulesHaveRequiredFields(t *testing.T) {
	for _, rule := range allSemgrepRules(t) {
		if mapString(rule, "id") == "" {
			t.Fatal("rule missing id")
		}
		_, hasPattern := rule["pattern"]
		_, hasPatterns := rule["patterns"]
		_, hasRegex := rule["pattern-regex"]
		_, hasEither := rule["pattern-either"]
		if !hasPattern && !hasPatterns && !hasRegex && !hasEither {
			t.Fatalf("rule %q missing pattern/patterns/pattern-regex", mapString(rule, "id"))
		}
		if mapString(rule, "message") == "" {
			t.Fatalf("rule %q missing message", mapString(rule, "id"))
		}
		if mapString(rule, "severity") == "" {
			t.Fatalf("rule %q missing severity", mapString(rule, "id"))
		}
		if _, ok := rule["languages"].([]interface{}); !ok {
			t.Fatalf("rule %q missing languages array", mapString(rule, "id"))
		}
		if mapStringMap(rule, "metadata") == nil {
			t.Fatalf("rule %q missing metadata", mapString(rule, "id"))
		}
	}
}

func TestGoStandard_AllRuleIDsFollowFormat(t *testing.T) {
	re := regexp.MustCompile(`^go\.[a-z-]+\.[a-z0-9-]+$`)
	for _, rule := range allSemgrepRules(t) {
		id := mapString(rule, "id")
		if !re.MatchString(id) {
			t.Fatalf("rule id %q does not match go.<category>.<kebab-name>", id)
		}
	}
}

func TestGoStandard_AllRulesTargetGo(t *testing.T) {
	for _, rule := range allSemgrepRules(t) {
		langs, ok := rule["languages"].([]interface{})
		if !ok || len(langs) != 1 || langs[0] != "go" {
			t.Fatalf("rule %q must set languages to [go], got %v", mapString(rule, "id"), rule["languages"])
		}
	}
}

func TestGoStandard_AllRulesHaveBackstopRuleID(t *testing.T) {
	std := stdRulesArray(t)
	for _, rule := range allSemgrepRules(t) {
		backstop := mapStringMap(mapStringMap(rule, "metadata"), "backstop")
		if backstop == nil {
			t.Fatalf("rule %q missing metadata.backstop", mapString(rule, "id"))
		}
		ruleID := mapString(backstop, "rule_id")
		if ruleID == "" {
			t.Fatalf("rule %q missing metadata.backstop.rule_id", mapString(rule, "id"))
		}
		if findSTDRuleByID(std, ruleID) == nil {
			t.Fatalf("rule %q references unknown STD rule id %q", mapString(rule, "id"), ruleID)
		}
	}
}

func TestGoStandard_InvalidFixtureExistsPerRule(t *testing.T) {
	invalid := fixtureNames(t, "invalid")
	for _, rule := range allSemgrepRules(t) {
		backstop := mapStringMap(mapStringMap(rule, "metadata"), "backstop")
		ruleID := mapString(backstop, "rule_id")
		if !hasFixtureForRule(ruleID, invalid) {
			t.Fatalf("no invalid fixture found for %s", ruleID)
		}
	}
}

func TestGoStandard_InvalidFixtureNamingConvention(t *testing.T) {
	re := regexp.MustCompile(`^go-\d{3}-[a-z0-9-]+(_test)?\.go$`)
	for _, name := range fixtureNames(t, "invalid") {
		if !re.MatchString(name) {
			t.Fatalf("invalid fixture %q does not match naming convention", name)
		}
	}
}

func TestGoStandard_InvalidFixtureTriggersRule(t *testing.T) {
	root := goStandardsRepoRoot(t)
	for _, name := range fixtureNames(t, "invalid") {
		path := filepath.Join(root, "standards", "go", "testdata", "invalid", name)
		result := semgrepOutputForFile(t, path)
		if len(result.Results) == 0 {
			t.Fatalf("invalid fixture %q produced no semgrep findings", name)
		}
	}
}

func TestGoStandard_ValidFixtureExistsPerRule(t *testing.T) {
	valid := fixtureNames(t, "valid")
	for _, rule := range allSemgrepRules(t) {
		backstop := mapStringMap(mapStringMap(rule, "metadata"), "backstop")
		ruleID := mapString(backstop, "rule_id")
		if !hasFixtureForRule(ruleID, valid) {
			t.Fatalf("no valid fixture found for %s", ruleID)
		}
	}
}

func TestGoStandard_ValidFixtureNamingConvention(t *testing.T) {
	re := regexp.MustCompile(`^go-\d{3}-[a-z0-9-]+\.go$`)
	for _, name := range fixtureNames(t, "valid") {
		if !re.MatchString(name) {
			t.Fatalf("valid fixture %q does not match naming convention", name)
		}
	}
}

func TestGoStandard_ValidFixturePassesAllRules(t *testing.T) {
	root := goStandardsRepoRoot(t)
	for _, name := range fixtureNames(t, "valid") {
		path := filepath.Join(root, "standards", "go", "testdata", "valid", name)
		result := semgrepOutputForFile(t, path)
		if len(result.Results) != 0 {
			t.Fatalf("valid fixture %q produced findings: %d", name, len(result.Results))
		}
	}
}

func TestGoStandard_DeferredCategoriesEmpty(t *testing.T) {
	root := goStandardsRepoRoot(t)
	dirs := []string{
		"performance", "observability", "integration", "contracts", "concurrency", "resilience", "accessibility",
	}
	for _, d := range dirs {
		dir := filepath.Join(root, "standards", "go", "rules", d)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("expected deferred directory %q: %v", dir, err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".yml") {
				t.Fatalf("deferred directory %q must not contain rule file %q", d, e.Name())
			}
		}
	}
}

func TestGoStandard_ConcurrencyRulesCompile(t *testing.T) {
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
	if res == nil || res.Manifest == nil {
		t.Fatal("expected compile result with manifest")
	}

	found050 := false
	found051 := false
	for _, r := range res.Manifest.Rules {
		if r.ID == "GO-050" {
			found050 = true
		}
		if r.ID == "GO-051" {
			found051 = true
		}
	}
	if !found050 || !found051 {
		t.Fatalf("expected GO-050 and GO-051 in manifest, got GO-050=%v GO-051=%v", found050, found051)
	}
}

func TestGoStandard_AllRulesRunWithSemgrepOSS(t *testing.T) {
	root := goStandardsRepoRoot(t)
	if _, err := exec.LookPath("semgrep"); err != nil {
		t.Skip("semgrep not installed in environment")
	}
	cmd := exec.Command(
		"semgrep",
		"--config", filepath.Join(root, "standards", "go", "rules"),
		"--json",
		filepath.Join(root, "standards", "go", "testdata"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("semgrep OSS run failed: %v\n%s", err, string(out))
	}
}

func TestGoStandard_NoProFeaturesUsed(t *testing.T) {
	disallowed := []string{
		"mode: taint",
		"join:",
		"pattern-sources:",
		"pattern-sinks:",
		"pattern-propagators:",
	}
	for _, path := range allSemgrepRuleFiles(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %q: %v", path, err)
		}
		content := strings.ToLower(string(data))
		for _, token := range disallowed {
			if strings.Contains(content, strings.ToLower(token)) {
				t.Fatalf("rule file %q uses disallowed pro feature token %q", path, token)
			}
		}
	}
}
