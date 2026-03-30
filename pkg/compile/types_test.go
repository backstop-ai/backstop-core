package compile_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/compile"
	"github.com/bmanson/backstop-core/pkg/schema"
)

func TestRule_DetectionStrategy(t *testing.T) {
	tests := []struct {
		name     string
		rule     compile.Rule
		expected string
	}{
		{
			name: "pattern strategy",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"strategy": "pattern",
					"pattern":  "foo",
				},
			},
			expected: "pattern",
		},
		{
			name: "metric strategy",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"strategy": "metric",
					"metric":   "cyclomatic_complexity",
				},
			},
			expected: "metric",
		},
		{
			name: "regex strategy",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"strategy": "regex",
					"regex":    "foo.*bar",
				},
			},
			expected: "regex",
		},
		{
			name: "delegated strategy",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"strategy":    "delegated",
					"enforced_by": "external-tool",
				},
			},
			expected: "delegated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rule.Strategy(); got != tt.expected {
				t.Fatalf("Strategy() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRule_IsAdvisory(t *testing.T) {
	tests := []struct {
		name     string
		rule     compile.Rule
		expected bool
	}{
		{
			name: "pattern rule with note but no semgrep is advisory",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"note":    "review manually",
					"pattern": "foo",
				},
			},
			expected: false,
		},
		{
			name: "pattern rule with semgrep is not advisory",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"note":    "review manually",
					"semgrep": map[string]interface{}{"pattern": "foo"},
				},
			},
			expected: false,
		},
		{
			name: "metric rule with note but no metric is advisory",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"note": "track over time",
				},
			},
			expected: true,
		},
		{
			name: "delegated rule with note but no enforced_by is advisory",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"note": "handled elsewhere",
				},
			},
			expected: true,
		},
		{
			name: "delegated rule with enforced_by is not advisory",
			rule: compile.Rule{
				Detection: map[string]interface{}{
					"note":        "handled elsewhere",
					"enforced_by": "custom-linter",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rule.IsAdvisory(); got != tt.expected {
				t.Fatalf("IsAdvisory() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestManifestRule_DefaultTier(t *testing.T) {
	rule := compile.ManifestRule{}

	if got := rule.EffectiveTier(); got != "baseline" {
		t.Fatalf("EffectiveTier() = %q, want %q", got, "baseline")
	}
}

func TestManifestRule_ExplicitTier(t *testing.T) {
	rule := compile.ManifestRule{ComplianceTier: "strict"}

	if got := rule.EffectiveTier(); got != "strict" {
		t.Fatalf("EffectiveTier() = %q, want %q", got, "strict")
	}
}

func TestSemgrepRule_SeverityUppercase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "error", expected: "ERROR"},
		{input: "warning", expected: "WARNING"},
		{input: "info", expected: "INFO"},
		{input: "custom", expected: "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := compile.MapSeverity(tt.input); got != tt.expected {
				t.Fatalf("MapSeverity(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCompileOptions_DefaultOutputDir(t *testing.T) {
	opts := compile.CompileOptions{}

	if got := opts.EffectiveOutputDir(); got != ".backstop/rules/" {
		t.Fatalf("EffectiveOutputDir() = %q, want %q", got, ".backstop/rules/")
	}
}

func TestCompileOptions_CustomOutputDir(t *testing.T) {
	opts := compile.CompileOptions{OutputDir: "generated/rules/"}

	if got := opts.EffectiveOutputDir(); got != "generated/rules/" {
		t.Fatalf("EffectiveOutputDir() = %q, want %q", got, "generated/rules/")
	}
}

type mockSchemaSource struct{}

func (m mockSchemaSource) LoadSchema(artifactType, version string) (*schema.Schema, error) {
	return &schema.Schema{}, nil
}

func TestRule_IsAdvisory_EmptyStringEnforcedBy(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":        "just a note",
			"enforced_by": "",
		},
	}
	if !rule.IsAdvisory() {
		t.Fatal("expected advisory when enforced_by is empty string")
	}
}

func TestRule_IsAdvisory_NilEnforcedBy(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":        "just a note",
			"enforced_by": nil,
		},
	}
	if !rule.IsAdvisory() {
		t.Fatal("expected advisory when enforced_by is nil")
	}
}

func TestRule_IsAdvisory_EmptyMapDetection(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":    "just a note",
			"semgrep": map[string]interface{}{},
		},
	}
	if !rule.IsAdvisory() {
		t.Fatal("expected advisory when semgrep is empty map")
	}
}

func TestRule_IsAdvisory_EmptySliceDetection(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":    "just a note",
			"pattern": []interface{}{},
		},
	}
	if !rule.IsAdvisory() {
		t.Fatal("expected advisory when pattern is empty slice")
	}
}

func TestRule_IsAdvisory_BoolFalseDetection(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":    "just a note",
			"semgrep": false,
		},
	}
	if !rule.IsAdvisory() {
		t.Fatal("expected advisory when semgrep is false")
	}
}

func TestRule_IsAdvisory_IntZeroDetection(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":   "just a note",
			"metric": 0,
		},
	}
	if !rule.IsAdvisory() {
		t.Fatal("expected advisory when metric is zero int")
	}
}

func TestRule_IsAdvisory_FloatZeroDetection(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":   "just a note",
			"metric": 0.0,
		},
	}
	if !rule.IsAdvisory() {
		t.Fatal("expected advisory when metric is zero float")
	}
}

func TestRule_IsAdvisory_NonZeroIntMeaningful(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":   "just a note",
			"metric": 42,
		},
	}
	if rule.IsAdvisory() {
		t.Fatal("expected not advisory when metric is non-zero int")
	}
}

func TestRule_IsAdvisory_UnknownTypeMeaningful(t *testing.T) {
	rule := compile.Rule{
		Detection: map[string]interface{}{
			"note":    "just a note",
			"semgrep": struct{}{},
		},
	}
	if rule.IsAdvisory() {
		t.Fatal("expected not advisory for unknown type (default case)")
	}
}

func TestRule_Strategy_NilDetection(t *testing.T) {
	rule := compile.Rule{}
	if got := rule.Strategy(); got != "" {
		t.Fatalf("Strategy() = %q, want empty", got)
	}
}

func TestRule_Strategy_MissingStrategyKey(t *testing.T) {
	rule := compile.Rule{Detection: map[string]interface{}{"other": "val"}}
	if got := rule.Strategy(); got != "" {
		t.Fatalf("Strategy() = %q, want empty", got)
	}
}

func TestRule_IsAdvisory_NilDetection(t *testing.T) {
	rule := compile.Rule{}
	if rule.IsAdvisory() {
		t.Fatal("expected not advisory when Detection is nil")
	}
}

func TestRule_IsAdvisory_NoNoteKey(t *testing.T) {
	rule := compile.Rule{Detection: map[string]interface{}{"semgrep": "pattern"}}
	if rule.IsAdvisory() {
		t.Fatal("expected not advisory when no note key")
	}
}

func TestSchemaSource_Interface(t *testing.T) {
	var source compile.SchemaSource = mockSchemaSource{}
	if source == nil {
		t.Fatal("expected schema source to satisfy interface")
	}
}
