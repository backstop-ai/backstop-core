package compile_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bmanson/backstop-core/pkg/compile"
	"gopkg.in/yaml.v3"
)

func TestEmitSemgrepRule_Pattern(t *testing.T) {
	rule := compile.Rule{
		ID:          "RULE-001",
		Name:        "Test rule",
		Severity:    "error",
		Description: "desc",
		Detection: map[string]interface{}{
			"strategy": "pattern",
			"semgrep":  "some_pattern($X)",
		},
	}

	got := compile.EmitSemgrepRule(rule, []string{"go"})

	if got.ID != "RULE-001" {
		t.Fatalf("ID = %q, want %q", got.ID, "RULE-001")
	}
	if got.Message != "desc" {
		t.Fatalf("Message = %q, want %q", got.Message, "desc")
	}
	if got.Severity != "ERROR" {
		t.Fatalf("Severity = %q, want %q", got.Severity, "ERROR")
	}
	if !reflect.DeepEqual(got.Languages, []string{"go"}) {
		t.Fatalf("Languages = %v, want %v", got.Languages, []string{"go"})
	}
	if got.Pattern != "some_pattern($X)" {
		t.Fatalf("Pattern = %q, want %q", got.Pattern, "some_pattern($X)")
	}
	if got.PatternRegex != "" {
		t.Fatalf("PatternRegex = %q, want empty", got.PatternRegex)
	}
}

func TestEmitSemgrepRule_Regex(t *testing.T) {
	rule := compile.Rule{
		Severity: "warning",
		Detection: map[string]interface{}{
			"strategy": "regex",
			"pattern":  "fmt\\.Println",
		},
	}

	got := compile.EmitSemgrepRule(rule, []string{"go"})

	if got.PatternRegex != "fmt\\.Println" {
		t.Fatalf("PatternRegex = %q, want %q", got.PatternRegex, "fmt\\.Println")
	}
	if got.Pattern != "" {
		t.Fatalf("Pattern = %q, want empty", got.Pattern)
	}
}

func TestEmitSemgrepRule_SeverityMapping(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{name: "error", severity: "error", want: "ERROR"},
		{name: "warning", severity: "warning", want: "WARNING"},
		{name: "info", severity: "info", want: "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := compile.Rule{Severity: tt.severity}

			got := compile.EmitSemgrepRule(rule, []string{"go"})

			if got.Severity != tt.want {
				t.Fatalf("Severity = %q, want %q", got.Severity, tt.want)
			}
		})
	}
}

func TestEmitSemgrepRule_MultipleLanguages(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"strategy": "pattern",
			"semgrep":  "foo($X)",
		},
	}

	got := compile.EmitSemgrepRule(rule, []string{"go", "typescript"})

	want := []string{"go", "typescript"}
	if !reflect.DeepEqual(got.Languages, want) {
		t.Fatalf("Languages = %v, want %v", got.Languages, want)
	}
}

func TestWriteSemgrepFile_ValidYAML(t *testing.T) {
	rules := []compile.SemgrepRule{
		{ID: "RULE-001", Message: "m1", Severity: "ERROR", Languages: []string{"go"}, Pattern: "foo($X)"},
		{ID: "RULE-002", Message: "m2", Severity: "WARNING", Languages: []string{"go"}, PatternRegex: "fmt\\.Println"},
	}

	path := filepath.Join(t.TempDir(), "semgrep.yml")
	if err := compile.WriteSemgrepFile(rules, path); err != nil {
		t.Fatalf("WriteSemgrepFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var doc map[string][]map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	items, ok := doc["rules"]
	if !ok {
		t.Fatalf("expected top-level rules key")
	}
	if len(items) != 2 {
		t.Fatalf("rules length = %d, want %d", len(items), 2)
	}
	if items[0]["id"] != "RULE-001" {
		t.Fatalf("first rule id = %v, want %q", items[0]["id"], "RULE-001")
	}
	if items[1]["id"] != "RULE-002" {
		t.Fatalf("second rule id = %v, want %q", items[1]["id"], "RULE-002")
	}
}

func TestWriteSemgrepFile_EmptyRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "semgrep.yml")

	if err := compile.WriteSemgrepFile([]compile.SemgrepRule{}, path); err != nil {
		t.Fatalf("WriteSemgrepFile() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, got err = %v", err)
	}
}

func TestWriteSemgrepFile_YAMLSpecialChars(t *testing.T) {
	rules := []compile.SemgrepRule{
		{ID: "RULE-001", Message: "m1", Severity: "ERROR", Languages: []string{"go"}, Pattern: "foo: {bar}"},
	}

	path := filepath.Join(t.TempDir(), "semgrep.yml")
	if err := compile.WriteSemgrepFile(rules, path); err != nil {
		t.Fatalf("WriteSemgrepFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var doc map[string][]map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if len(doc["rules"]) != 1 {
		t.Fatalf("rules length = %d, want %d", len(doc["rules"]), 1)
	}
}

func TestEmitSemgrepRule_Exceptions(t *testing.T) {
	rule := compile.Rule{
		ID:          "RULE-003",
		Severity:    "error",
		Description: "no globals",
		Detection: map[string]interface{}{
			"strategy": "pattern",
			"semgrep":  "var $NAME = ...",
			"exceptions": []interface{}{
				"sync.Once",
				"regexp.MustCompile",
			},
		},
	}
	sr := compile.EmitSemgrepRule(rule, []string{"go"})
	if len(sr.PatternNotRegex) != 2 {
		t.Fatalf("PatternNotRegex len = %d, want 2", len(sr.PatternNotRegex))
	}
	if sr.PatternNotRegex[0] != "sync.Once" {
		t.Fatalf("PatternNotRegex[0] = %q, want %q", sr.PatternNotRegex[0], "sync.Once")
	}
}

func TestWriteSemgrepFile_PatternWithExceptions(t *testing.T) {
	rules := []compile.SemgrepRule{
		{
			ID:              "RULE-003",
			Message:         "no globals",
			Severity:        "ERROR",
			Languages:       []string{"go"},
			Pattern:         "var $NAME = ...",
			PatternNotRegex: []string{"sync.Once", "regexp.MustCompile"},
		},
	}
	path := filepath.Join(t.TempDir(), "test.yml")
	if err := compile.WriteSemgrepFile(rules, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string][]map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	rule := doc["rules"][0]
	// Should have "patterns" key, not "pattern"
	if _, ok := rule["pattern"]; ok {
		t.Fatal("expected 'patterns' composite, got bare 'pattern'")
	}
	patterns, ok := rule["patterns"].([]interface{})
	if !ok {
		t.Fatalf("expected patterns to be a list, got %T", rule["patterns"])
	}
	// Should have 3 entries: 1 pattern + 2 pattern-not-regex
	if len(patterns) != 3 {
		t.Fatalf("patterns len = %d, want 3", len(patterns))
	}
}
