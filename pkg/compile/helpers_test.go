package compile

import (
	"os"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
)

func Test_frontmatterString_NilMap(t *testing.T) {
	if got := frontmatterString(nil, "key"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func Test_frontmatterString_NonStringValue(t *testing.T) {
	m := map[string]interface{}{"key": 42}
	if got := frontmatterString(m, "key"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func Test_mapString_NilMap(t *testing.T) {
	if got := mapString(nil, "key"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func Test_mapString_NonStringValue(t *testing.T) {
	m := map[string]interface{}{"key": true}
	if got := mapString(m, "key"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func Test_mapStringMap_NilMap(t *testing.T) {
	if got := mapStringMap(nil, "key"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func Test_mapStringMap_NonMapValue(t *testing.T) {
	m := map[string]interface{}{"key": "not a map"}
	if got := mapStringMap(m, "key"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func Test_mapStrings_NilMap(t *testing.T) {
	if got := mapStrings(nil, "key"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func Test_mapStrings_NonArrayValue(t *testing.T) {
	m := map[string]interface{}{"key": "not an array"}
	if got := mapStrings(m, "key"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func Test_mapStrings_NonStringItems(t *testing.T) {
	m := map[string]interface{}{"key": []interface{}{42, true}}
	got := mapStrings(m, "key")
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func Test_detectionString_NilMap(t *testing.T) {
	if got := detectionString(nil, "key"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func Test_detectionString_NonStringValue(t *testing.T) {
	m := map[string]interface{}{"key": 99}
	if got := detectionString(m, "key"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func Test_resolveRuleLanguages_NonUniversalEmptyLanguage(t *testing.T) {
	rule := Rule{Languages: []string{"go"}}
	if got := resolveRuleLanguages("language", "", rule); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func Test_parseRules_MissingRules(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Frontmatter: map[string]interface{}{},
	}
	_, err := parseRules(art)
	if err == nil {
		t.Fatal("expected error for missing rules")
	}
}

func Test_parseRules_NotArray(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Frontmatter: map[string]interface{}{"rules": "not an array"},
	}
	_, err := parseRules(art)
	if err == nil {
		t.Fatal("expected error for non-array rules")
	}
}

func Test_parseRules_ItemNotObject(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Frontmatter: map[string]interface{}{
			"rules": []interface{}{"just a string"},
		},
	}
	_, err := parseRules(art)
	if err == nil {
		t.Fatal("expected error for non-object rule item")
	}
}

func Test_parseRules_DuplicateIDs(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Frontmatter: map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{"id": "DUP-001", "detection": map[string]interface{}{"strategy": "pattern"}},
				map[string]interface{}{"id": "DUP-001", "detection": map[string]interface{}{"strategy": "metric"}},
			},
		},
	}
	_, err := parseRules(art)
	if err == nil {
		t.Fatal("expected error for duplicate rule IDs")
	}
}

func Test_Compile_ResolveSchemaPathError(t *testing.T) {
	dir := t.TempDir()
	// Write a standard with missing schema_version so ResolveSchemaPath fails
	content := `---
title: No Schema Version
number: STD-TEST-001
created: "2026-01-01"
status: active
language: go
scope: language
rules:
  - id: T-001
    name: test
    detection:
      strategy: pattern
      semgrep: foo
---

# No Schema Version

## Rationale

Test.

## Primary Sources

Test.
`
	path := dir + "/STD-TEST-001-test.standard.md"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// nil SchemaSource triggers filesystem resolution
	_, err := Compile(path, CompileOptions{OutputDir: dir + "/out"})
	if err == nil {
		t.Fatal("expected error from ResolveSchemaPath with missing schema_version")
	}
}
