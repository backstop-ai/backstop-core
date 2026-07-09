package engine

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestParseScopeKind covers the scope_kind spelling parser (ISSUE-032 B0): the empty
// and file-args spellings default to ScopeKindFileArgs, project-wide resolves, and an
// unknown spelling fails loud.
func TestParseScopeKind(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want ScopeKind
	}{
		{"", ScopeKindFileArgs},
		{"file-args", ScopeKindFileArgs},
		{"project-wide", ScopeKindProjectWide},
	} {
		got, err := ParseScopeKind(tc.in)
		if err != nil {
			t.Errorf("ParseScopeKind(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("ParseScopeKind(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
	if _, err := ParseScopeKind("bogus"); err == nil {
		t.Error("ParseScopeKind(bogus) should fail loud")
	}
}

// TestParseEngineCategory covers the category spelling parser.
func TestParseEngineCategory(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want EngineCategory
	}{
		{"", EngineCategoryUnset},
		{"mechanism", EngineCategoryMechanism},
		{"opinion", EngineCategoryOpinion},
	} {
		got, err := ParseEngineCategory(tc.in)
		if err != nil {
			t.Errorf("ParseEngineCategory(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("ParseEngineCategory(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
	if _, err := ParseEngineCategory("bogus"); err == nil {
		t.Error("ParseEngineCategory(bogus) should fail loud")
	}
}

// TestEnumUnmarshalYAML_Valid proves a raw EngineBinding yaml decode resolves the
// STRING enum spellings (ISSUE-032 B0/CLM-012) — the decoder packval relies on.
func TestEnumUnmarshalYAML_Valid(t *testing.T) {
	var b EngineBinding
	src := "scope_kind: project-wide\ncategory: mechanism\ngate_type: build\n"
	if err := yaml.Unmarshal([]byte(src), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if b.ScopeKind != ScopeKindProjectWide {
		t.Errorf("ScopeKind = %v, want ScopeKindProjectWide", b.ScopeKind)
	}
	if b.Category != EngineCategoryMechanism {
		t.Errorf("Category = %v, want EngineCategoryMechanism", b.Category)
	}
	if b.GateType != GateTypeBuild {
		t.Errorf("GateType = %v, want GateTypeBuild", b.GateType)
	}
}

// TestEnumUnmarshalYAML_InvalidString proves each enum decoder fails loud on an
// unrecognized string spelling.
func TestEnumUnmarshalYAML_InvalidString(t *testing.T) {
	for _, src := range []string{
		"scope_kind: bogus\n",
		"category: bogus\n",
		"gate_type: bogus\n",
	} {
		var b EngineBinding
		if err := yaml.Unmarshal([]byte(src), &b); err == nil {
			t.Errorf("expected fail-loud unmarshal error for %q", src)
		}
	}
}

// TestEnumUnmarshalYAML_NonStringNode proves each enum decoder errors when the yaml
// node is not a scalar string (e.g. a sequence), rather than silently zero-valuing.
func TestEnumUnmarshalYAML_NonStringNode(t *testing.T) {
	for _, src := range []string{
		"scope_kind: [a, b]\n",
		"category: {k: v}\n",
		"gate_type: [1, 2]\n",
	} {
		var b EngineBinding
		if err := yaml.Unmarshal([]byte(src), &b); err == nil {
			t.Errorf("expected error decoding non-string enum node for %q", src)
		}
	}
}
