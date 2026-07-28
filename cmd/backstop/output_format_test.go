package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// TestCLI_JSONFormatter_FormatResult_Passing verifies that the JSON formatter
// emits a schema-versioned envelope with pass=true and an empty (non-null)
// violations array when there are no violations.
func TestCLI_JSONFormatter_FormatResult_Passing(t *testing.T) {
	f := &JSONFormatter{}
	out, err := f.FormatResult(validate.ValidationResult{})
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	var env struct {
		SchemaVersion   string `json:"schema_version"`
		Pass            bool   `json:"pass"`
		ViolationsCount int    `json:"violations_count"`
		Violations      []struct {
			Rule    string `json:"rule"`
			Message string `json:"message"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if env.SchemaVersion != "cli/v1" {
		t.Errorf("schema_version = %q, want %q", env.SchemaVersion, "cli/v1")
	}
	if !env.Pass {
		t.Error("expected pass=true for empty result")
	}
	if env.ViolationsCount != 0 {
		t.Errorf("violations_count = %d, want 0", env.ViolationsCount)
	}
	// violations must be an empty array, never null, for stable wire contract.
	if !strings.Contains(out, `"violations": []`) {
		t.Errorf("expected empty violations array in output, got:\n%s", out)
	}
}

// TestCLI_JSONFormatter_FormatResult_Failing verifies that violations are
// faithfully serialized (rule, file, message, severity) and pass=false when the
// result carries an error violation.
func TestCLI_JSONFormatter_FormatResult_Failing(t *testing.T) {
	f := &JSONFormatter{}
	result := validate.ValidationResult{Violations: []validate.Violation{
		{Rule: "REQ-001", File: "a.go", Message: "missing thing", Severity: "error"},
		{Rule: "REQ-002", File: "b.go", Message: "another", Severity: "warning"},
	}}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	var env struct {
		Pass            bool `json:"pass"`
		ViolationsCount int  `json:"violations_count"`
		Violations      []struct {
			Rule     string `json:"rule"`
			File     string `json:"file"`
			Message  string `json:"message"`
			Severity string `json:"severity"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if env.Pass {
		t.Error("expected pass=false when violations present")
	}
	if env.ViolationsCount != 2 || len(env.Violations) != 2 {
		t.Fatalf("expected 2 violations, got count=%d len=%d", env.ViolationsCount, len(env.Violations))
	}
	if env.Violations[0].Rule != "REQ-001" || env.Violations[0].File != "a.go" || env.Violations[0].Severity != "error" {
		t.Errorf("first violation not serialized faithfully: %+v", env.Violations[0])
	}
	if env.Violations[1].Message != "another" {
		t.Errorf("second violation message = %q, want %q", env.Violations[1].Message, "another")
	}
}

// TestCLI_HumanFormatter_FormatResult_NoColor verifies the NO_COLOR path: no
// ANSI escape codes are emitted, violations are grouped under their file
// headers, and the failure status line appears.
func TestCLI_HumanFormatter_FormatResult_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	f := &HumanFormatter{}
	result := validate.ValidationResult{Violations: []validate.Violation{
		{Rule: "REQ-001", File: "a.go", Message: "first", Severity: "error"},
		{Rule: "REQ-002", File: "a.go", Message: "second", Severity: "warning"},
		{Rule: "REQ-003", File: "b.go", Message: "third", Severity: "error"},
	}}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI codes under NO_COLOR, got:\n%q", out)
	}
	// Both violations on a.go must appear under a single a.go header.
	if strings.Count(out, "a.go\n") != 1 {
		t.Errorf("expected exactly one a.go header, got:\n%s", out)
	}
	for _, want := range []string{"a.go", "b.go", "[REQ-001] first", "[REQ-003] third", "Checks failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

// TestCLI_HumanFormatter_FormatResult_Color verifies the colored path emits
// ANSI escape sequences for the file header and error-severity violations.
func TestCLI_HumanFormatter_FormatResult_Color(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	f := &HumanFormatter{}
	result := validate.ValidationResult{Violations: []validate.Violation{
		{Rule: "REQ-001", File: "a.go", Message: "boom", Severity: "error"},
	}}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if !strings.Contains(out, "\033[1m") {
		t.Errorf("expected bold ANSI for file header, got:\n%q", out)
	}
	if !strings.Contains(out, "\033[31m") {
		t.Errorf("expected red ANSI for error violation, got:\n%q", out)
	}
}

// TestCLI_HumanFormatter_FormatResult_NoFileViolation verifies that a violation
// with an empty File is grouped under the "(no file)" header.
func TestCLI_HumanFormatter_FormatResult_NoFileViolation(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	f := &HumanFormatter{}
	result := validate.ValidationResult{Violations: []validate.Violation{
		{Rule: "REQ-001", File: "", Message: "global problem", Severity: "error"},
	}}
	out, err := f.FormatResult(result)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if !strings.Contains(out, "(no file)") {
		t.Errorf("expected (no file) header for fileless violation, got:\n%s", out)
	}
}

// TestCLI_HumanFormatter_FormatResult_PassWithColor verifies the passing,
// colored branch emits the green success status line.
func TestCLI_HumanFormatter_FormatResult_PassWithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	f := &HumanFormatter{}
	out, err := f.FormatResult(validate.ValidationResult{})
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if !strings.Contains(out, "\033[32m") || !strings.Contains(out, "All checks passed") {
		t.Errorf("expected green success line, got:\n%q", out)
	}
}

// TestCLI_HumanFormatter_FormatResult_PassNoColor verifies the passing,
// no-color branch emits a plain success status line.
func TestCLI_HumanFormatter_FormatResult_PassNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	f := &HumanFormatter{}
	out, err := f.FormatResult(validate.ValidationResult{})
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI codes, got:\n%q", out)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("expected plain success line, got:\n%s", out)
	}
}

// TestCLI_HumanArtifactNewFormatter_FormatNewResult verifies the human
// artifact-new formatter renders the created file path and ID.
func TestCLI_HumanArtifactNewFormatter_FormatNewResult(t *testing.T) {
	f := &HumanArtifactNewFormatter{}
	out, err := f.FormatNewResult(ArtifactNewResult{ArtifactType: "spec", ID: "042", FilePath: "specs/SPEC-042.spec.md"})
	if err != nil {
		t.Fatalf("FormatNewResult: %v", err)
	}
	if !strings.Contains(out, "specs/SPEC-042.spec.md") || !strings.Contains(out, "042") {
		t.Errorf("expected path and ID in output, got %q", out)
	}
}

// TestCLI_JSONArtifactNewFormatter_FormatNewResult verifies the JSON
// artifact-new formatter serializes the result fields.
func TestCLI_JSONArtifactNewFormatter_FormatNewResult(t *testing.T) {
	f := &JSONArtifactNewFormatter{}
	out, err := f.FormatNewResult(ArtifactNewResult{ArtifactType: "issue", ID: "007", FilePath: "issues/ISSUE-007.issue.md"})
	if err != nil {
		t.Fatalf("FormatNewResult: %v", err)
	}
	var decoded ArtifactNewResult
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if decoded.ArtifactType != "issue" || decoded.ID != "007" || decoded.FilePath != "issues/ISSUE-007.issue.md" {
		t.Errorf("round-trip mismatch: %+v", decoded)
	}
}
