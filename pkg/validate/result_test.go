package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

func TestViolation_StructFields(t *testing.T) {
	v := validate.Violation{
		Rule:     "base/title-required",
		File:     "ADR-0001-test.adr.md",
		Message:  "artifact title is missing",
		Severity: "error",
	}

	if v.Rule != "base/title-required" {
		t.Errorf("Rule = %q, want %q", v.Rule, "base/title-required")
	}
	if v.File != "ADR-0001-test.adr.md" {
		t.Errorf("File = %q, want %q", v.File, "ADR-0001-test.adr.md")
	}
	if v.Message != "artifact title is missing" {
		t.Errorf("Message = %q, want %q", v.Message, "artifact title is missing")
	}
	if v.Severity != "error" {
		t.Errorf("Severity = %q, want %q", v.Severity, "error")
	}
}

func TestValidationResult_PassWhenNoViolations(t *testing.T) {
	r := validate.ValidationResult{
		Violations: []validate.Violation{},
	}

	if !r.Pass() {
		t.Error("Pass() = false, want true when Violations is empty")
	}
}

func TestValidationResult_FailWhenViolations(t *testing.T) {
	r := validate.ValidationResult{
		Violations: []validate.Violation{
			{Rule: "base/title-required", File: "test.md", Message: "missing", Severity: "error"},
		},
	}

	if r.Pass() {
		t.Error("Pass() = true, want false when Violations is non-empty")
	}
}
