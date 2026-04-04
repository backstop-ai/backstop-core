package compile

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"gopkg.in/yaml.v3"
)

func goStandardsRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve caller path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func loadSemgrepRulesFile(t *testing.T, path string) []map[string]interface{} {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read semgrep rules file %q: %v", path, err)
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse semgrep rules file %q: %v", path, err)
	}

	rawRules, ok := doc["rules"]
	if !ok {
		t.Fatalf("semgrep rules file %q missing top-level rules array", path)
	}

	items, ok := rawRules.([]interface{})
	if !ok {
		t.Fatalf("semgrep rules file %q has non-array rules field", path)
	}

	rules := make([]map[string]interface{}, 0, len(items))
	for i, item := range items {
		rule, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("semgrep rules file %q has non-object rules[%d]", path, i)
		}
		rules = append(rules, rule)
	}

	return rules
}

func loadStandardArtifact(t *testing.T) *artifact.ParsedArtifact {
	t.Helper()

	root := goStandardsRepoRoot(t)
	standardPath := filepath.Join(root, "standards", "go", "STD-GO-001-go-code-standards.standard.md")

	art, err := artifact.ParseFile(standardPath)
	if err != nil {
		t.Fatalf("parse standard artifact %q: %v", standardPath, err)
	}

	return art
}

func findRuleByID(rules []map[string]interface{}, id string) map[string]interface{} {
	for _, rule := range rules {
		if strings.TrimSpace(mapString(rule, "id")) == id {
			return rule
		}
	}
	return nil
}

func findSTDRuleByID(rules []interface{}, id string) map[string]interface{} {
	for _, raw := range rules {
		rule, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.TrimSpace(mapString(rule, "id")) == id {
			return rule
		}
	}
	return nil
}

func allSemgrepRuleFiles(t *testing.T) []string {
	t.Helper()

	root := goStandardsRepoRoot(t)
	rulesRoot := filepath.Join(root, "standards", "go", "rules")

	var files []string
	err := filepath.WalkDir(rulesRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".yml") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk semgrep rule files under %q: %v", rulesRoot, err)
	}

	sort.Strings(files)
	return files
}
