package compile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/compile"
)

func TestEmitNativeCheck_MetricRule(t *testing.T) {
	rule := compile.Rule{
		ID:          "RULE-M01",
		Name:        "Complexity limit",
		Severity:    "error",
		Description: "Keep complexity low",
		Detection: map[string]interface{}{
			"strategy":  "metric",
			"metric":    "cyclomatic_complexity",
			"operator":  "<=",
			"threshold": 10,
			"exclude":   []interface{}{"vendor/", "generated/"},
		},
		Languages: []string{"go"},
	}

	check := compile.EmitNativeCheck(rule)

	if check.ID != "RULE-M01" {
		t.Fatalf("ID = %q, want %q", check.ID, "RULE-M01")
	}
	if check.Message != "Keep complexity low" {
		t.Fatalf("Message = %q, want %q", check.Message, "Keep complexity low")
	}
	if check.Severity != "error" {
		t.Fatalf("Severity = %q, want %q", check.Severity, "error")
	}
	if check.Language != "go" {
		t.Fatalf("Language = %q, want %q", check.Language, "go")
	}
	if check.Metric != "cyclomatic_complexity" {
		t.Fatalf("Metric = %q, want %q", check.Metric, "cyclomatic_complexity")
	}
	if check.Operator != "<=" {
		t.Fatalf("Operator = %q, want %q", check.Operator, "<=")
	}
	if threshold, ok := check.Threshold.(int); !ok || threshold != 10 {
		t.Fatalf("Threshold = %#v, want int(10)", check.Threshold)
	}

	wantExclude := []string{"vendor/", "generated/"}
	if len(check.Exclude) != len(wantExclude) {
		t.Fatalf("Exclude length = %d, want %d", len(check.Exclude), len(wantExclude))
	}
	for i := range wantExclude {
		if check.Exclude[i] != wantExclude[i] {
			t.Fatalf("Exclude[%d] = %q, want %q", i, check.Exclude[i], wantExclude[i])
		}
	}
}

func TestEmitNativeCheck_NoExclude(t *testing.T) {
	rule := compile.Rule{
		ID:          "RULE-M02",
		Severity:    "warning",
		Description: "Keep function length low",
		Detection: map[string]interface{}{
			"strategy":  "metric",
			"metric":    "function_length",
			"operator":  "<=",
			"threshold": 50,
		},
		Languages: []string{"go"},
	}

	check := compile.EmitNativeCheck(rule)
	if len(check.Exclude) != 0 {
		t.Fatalf("Exclude = %#v, want nil or empty", check.Exclude)
	}
}

func TestWriteNativeChecksFile_ValidJSON(t *testing.T) {
	checks := []compile.NativeCheck{
		{ID: "RULE-M01", Message: "msg1", Severity: "error", Language: "go", Metric: "cyclomatic_complexity", Operator: "<=", Threshold: 10},
		{ID: "RULE-M02", Message: "msg2", Severity: "warning", Language: "go", Metric: "function_length", Operator: "<=", Threshold: 50},
	}

	path := filepath.Join(t.TempDir(), "native-checks.json")
	if err := compile.WriteNativeChecksFile(checks, path); err != nil {
		t.Fatalf("WriteNativeChecksFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var parsed struct {
		Checks []compile.NativeCheck `json:"checks"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(parsed.Checks) != 2 {
		t.Fatalf("checks length = %d, want 2", len(parsed.Checks))
	}
	if parsed.Checks[0].ID != "RULE-M01" {
		t.Fatalf("checks[0].ID = %q, want %q", parsed.Checks[0].ID, "RULE-M01")
	}
	if parsed.Checks[1].ID != "RULE-M02" {
		t.Fatalf("checks[1].ID = %q, want %q", parsed.Checks[1].ID, "RULE-M02")
	}

	if !containsIndentedJSON(string(data)) {
		t.Fatalf("expected pretty-printed JSON with indentation, got: %q", string(data))
	}
}

func TestWriteNativeChecksFile_EmptyChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native-checks.json")
	if err := compile.WriteNativeChecksFile([]compile.NativeCheck{}, path); err != nil {
		t.Fatalf("WriteNativeChecksFile() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected no file to be created, stat err = %v", err)
	}
}

func containsIndentedJSON(s string) bool {
	return len(s) > 0 &&
		contains(s, "\n") &&
		contains(s, "\n  \"checks\": [") &&
		contains(s, "\n    {")
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (index(s, sub) >= 0))
}

func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
