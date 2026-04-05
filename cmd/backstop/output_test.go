package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// TestCLI_Output_JSONFlag_ProducesJSON formats a result with JSONFormatter,
// verifies output is valid JSON. (CLM-013)
func TestCLI_Output_JSONFlag_ProducesJSON(t *testing.T) {
	f := &JSONFormatter{}
	result := validate.ValidationResult{}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	if !json.Valid([]byte(out)) {
		t.Errorf("output is not valid JSON: %s", out)
	}
}

// TestCLI_Output_JSON_HasSchemaVersion verifies JSON output includes
// schema_version field. (CLM-016)
func TestCLI_Output_JSON_HasSchemaVersion(t *testing.T) {
	f := &JSONFormatter{}
	result := validate.ValidationResult{}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	sv, ok := parsed["schema_version"]
	if !ok {
		t.Fatal("JSON output missing schema_version field")
	}
	if sv != "cli/v1" {
		t.Errorf("schema_version = %q, want %q", sv, "cli/v1")
	}
}

// TestCLI_Output_JSONAndHuman_IdenticalData formats same result with both
// formatters, verifies underlying violation data matches. (CLM-015)
func TestCLI_Output_JSONAndHuman_IdenticalData(t *testing.T) {
	result := validate.ValidationResult{
		Violations: []validate.Violation{
			{Rule: "R001", Message: "test violation"},
		},
	}

	jf := &JSONFormatter{}
	jsonOut, err := jf.FormatResult(result)
	if err != nil {
		t.Fatalf("JSONFormatter error: %v", err)
	}

	hf := &HumanFormatter{}
	humanOut, err := hf.FormatResult(result)
	if err != nil {
		t.Fatalf("HumanFormatter error: %v", err)
	}

	// JSON output should contain violation data
	if !strings.Contains(jsonOut, "R001") {
		t.Error("JSON output missing violation rule R001")
	}
	if !strings.Contains(jsonOut, "test violation") {
		t.Error("JSON output missing violation message")
	}

	// Human output should contain the same violation data
	if !strings.Contains(humanOut, "R001") {
		t.Error("Human output missing violation rule R001")
	}
	if !strings.Contains(humanOut, "test violation") {
		t.Error("Human output missing violation message")
	}
}

// TestCLI_Output_Default_ProducesHumanText formats a result with
// HumanFormatter, verifies output is human-readable text (not JSON). (CLM-014)
func TestCLI_Output_Default_ProducesHumanText(t *testing.T) {
	f := &HumanFormatter{}
	result := validate.ValidationResult{
		Violations: []validate.Violation{
			{Rule: "R001", Message: "something wrong"},
		},
	}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	// Should not be valid JSON (it's human-readable)
	if json.Valid([]byte(out)) {
		t.Error("human output should not be valid JSON")
	}
	// Should contain meaningful text
	if !strings.Contains(out, "R001") {
		t.Error("human output missing violation info")
	}
}

// TestCLI_NoColor_OmitsANSI sets NO_COLOR, formats with HumanFormatter,
// verifies no ANSI escape sequences in output. (CLM-030)
func TestCLI_NoColor_OmitsANSI(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	f := &HumanFormatter{}
	result := validate.ValidationResult{
		Violations: []validate.Violation{
			{Rule: "R001", Message: "test", Severity: "error"},
		},
	}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	if strings.Contains(out, "\033[") {
		t.Error("output contains ANSI escape codes when NO_COLOR is set")
	}
}

// --- ArtifactNewFormatter tests ---

func TestJSONArtifactNewFormatter_FormatNewResult(t *testing.T) {
	f := &JSONArtifactNewFormatter{}
	result := ArtifactNewResult{
		ArtifactType:  "spec",
		ID:            "001",
		FilePath:      "/tmp/specs/SPEC-001-my-spec.spec.md",
		SchemaVersion: "cli/v1",
	}
	out, err := f.FormatNewResult(result)
	if err != nil {
		t.Fatalf("FormatNewResult error: %v", err)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Errorf("output is not valid JSON: %s", out)
	}
	if !strings.Contains(out, "spec") {
		t.Error("JSON output missing artifact type")
	}
	if !strings.Contains(out, "001") {
		t.Error("JSON output missing ID")
	}
	if !strings.Contains(out, "schema_version") {
		t.Error("JSON output missing schema_version")
	}
}

func TestHumanArtifactNewFormatter_FormatNewResult(t *testing.T) {
	f := &HumanArtifactNewFormatter{}
	result := ArtifactNewResult{
		ArtifactType:  "spec",
		ID:            "001",
		FilePath:      "/tmp/specs/SPEC-001-my-spec.spec.md",
		SchemaVersion: "cli/v1",
	}
	out, err := f.FormatNewResult(result)
	if err != nil {
		t.Fatalf("FormatNewResult error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Error("human output missing 'Created'")
	}
	if !strings.Contains(out, "001") {
		t.Error("human output missing ID")
	}
	if !strings.Contains(out, "SPEC-001") {
		t.Error("human output missing file path")
	}
}

// TestCLI_NoColor_AllowsANSIWhenUnset unsets NO_COLOR, formats with
// HumanFormatter, verifies ANSI codes may be present. (CLM-031)
func TestCLI_NoColor_AllowsANSIWhenUnset(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	f := &HumanFormatter{}
	result := validate.ValidationResult{
		Violations: []validate.Violation{
			{Rule: "R001", Message: "test", Severity: "error"},
		},
	}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult error: %v", err)
	}
	// When NO_COLOR is not set, ANSI codes may be present
	// We verify the formatter runs without error; color is optional
	if len(out) == 0 {
		t.Error("output is empty")
	}
}
