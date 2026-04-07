package gate

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// mockValidator implements ArtifactValidator for testing.
type mockValidator struct {
	violations []Violation
	err        error
}

func (m *mockValidator) ValidateAll(_ context.Context) ([]Violation, error) {
	return m.violations, m.err
}

// mockChecker implements CodeChecker for testing.
type mockChecker struct {
	violations []Violation
	err        error
}

func (m *mockChecker) CheckAll(_ context.Context) ([]Violation, error) {
	return m.violations, m.err
}

// --- Artifact validation tests ---

// TestGate_ArtifactValidation_ReportsViolations verifies that artifact
// validation step reports violations under "artifact_validation" section.
func TestGate_ArtifactValidation_ReportsViolations(t *testing.T) {
	v := &mockValidator{
		violations: []Violation{
			{Rule: "schema", File: "specs/FOO.spec.md", Message: "missing field", Severity: "error"},
		},
	}
	step := StepArtifactValidationFunc(v)
	result := step(context.Background())

	if result.StepName != StepArtifactValidation {
		t.Errorf("expected step_name %q, got %q", StepArtifactValidation, result.StepName)
	}
	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Rule != "schema" {
		t.Errorf("expected violation rule %q, got %q", "schema", result.Violations[0].Rule)
	}
}

// TestGate_ArtifactValidation_PassWhenValid verifies artifact validation step
// reports pass when all artifacts are valid.
func TestGate_ArtifactValidation_PassWhenValid(t *testing.T) {
	v := &mockValidator{violations: nil}
	step := StepArtifactValidationFunc(v)
	result := step(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q", "pass", result.Status)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result.Violations))
	}
}

// --- Code check tests ---

// TestGate_CodeCheck_ReportsViolations verifies that code check step reports
// violations under "code_check" section.
func TestGate_CodeCheck_ReportsViolations(t *testing.T) {
	c := &mockChecker{
		violations: []Violation{
			{Rule: "lint", File: "main.go", Message: "unused import", Severity: "warning"},
			{Rule: "build", File: "main.go", Message: "compile error", Severity: "error"},
		},
	}
	step := StepCodeCheckFunc(c)
	result := step(context.Background())

	if result.StepName != StepCodeCheck {
		t.Errorf("expected step_name %q, got %q", StepCodeCheck, result.StepName)
	}
	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(result.Violations))
	}
}

// TestGate_CodeCheck_PassWhenClean verifies code check step reports pass when
// no violations are found.
func TestGate_CodeCheck_PassWhenClean(t *testing.T) {
	c := &mockChecker{violations: nil}
	step := StepCodeCheckFunc(c)
	result := step(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q", "pass", result.Status)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result.Violations))
	}
}

// --- ConfigError Unwrap tests ---

// TestGate_ConfigError_Unwrap verifies that ConfigError.Unwrap returns the
// wrapped error, enabling errors.Is and errors.As chains.
func TestGate_ConfigError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	ce := &ConfigError{Err: inner}

	unwrapped := ce.Unwrap()
	if unwrapped != inner {
		t.Errorf("expected Unwrap to return inner error, got %v", unwrapped)
	}

	// Verify errors.Is works through the wrapper
	sentinel := fmt.Errorf("sentinel")
	wrapped := &ConfigError{Err: fmt.Errorf("wrapping: %w", sentinel)}
	if !errors.Is(wrapped, sentinel) {
		t.Error("expected errors.Is to find sentinel through ConfigError.Unwrap")
	}
}

// --- Config error propagation tests ---

// TestGate_StepArtifactValidation_ConfigErrorSignal verifies that a
// config error from artifact validate propagates as a config error signal
// at the step level.
func TestGate_StepArtifactValidation_ConfigErrorSignal(t *testing.T) {
	v := &mockValidator{
		err: &ConfigError{Err: fmt.Errorf("schema loading failure")},
	}
	step := StepArtifactValidationFunc(v)
	result := step(context.Background())

	if !result.ConfigErr {
		t.Error("expected ConfigErr flag to be true on config error")
	}
	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
}

// TestGate_StepCodeCheck_ConfigErrorSignal verifies that a config error
// from code check propagates as a config error signal at the step level.
func TestGate_StepCodeCheck_ConfigErrorSignal(t *testing.T) {
	c := &mockChecker{
		err: &ConfigError{Err: fmt.Errorf("backstop dir missing")},
	}
	step := StepCodeCheckFunc(c)
	result := step(context.Background())

	if !result.ConfigErr {
		t.Error("expected ConfigErr flag to be true on config error")
	}
	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
}
