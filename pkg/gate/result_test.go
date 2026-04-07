package gate

import (
	"encoding/json"
	"testing"
)

// TestGate_JSONOutput_StructureComplete verifies that GateResult marshals to JSON
// with schema_version, pass boolean, steps array, and summary fields.
func TestGate_JSONOutput_StructureComplete(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", File: "foo.go", Message: "unused var", Severity: "error"},
		}},
		{StepName: StepTestVerification, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"},
	}
	result := NewGateResult(steps)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal GateResult: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	for _, field := range []string{"schema_version", "pass", "steps", "total_violations", "steps_passed", "steps_failed", "steps_skipped"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing field %q in JSON output", field)
		}
	}
}

// TestGate_JSONOutput_StepFieldsPresent verifies each StepResult in JSON
// has step_name, status, and violations fields.
func TestGate_JSONOutput_StepFieldsPresent(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
	}
	result := NewGateResult(steps)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var envelope struct {
		Steps []map[string]json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(envelope.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(envelope.Steps))
	}

	for _, field := range []string{"step_name", "status", "violations"} {
		if _, ok := envelope.Steps[0][field]; !ok {
			t.Errorf("missing field %q in step JSON", field)
		}
	}
}

// TestGate_JSONOutput_SkippedStepHasReason verifies a skipped StepResult
// includes the reason field in JSON output.
func TestGate_JSONOutput_SkippedStepHasReason(t *testing.T) {
	steps := []StepResult{
		{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, Reason: "baseline not implemented"},
	}
	result := NewGateResult(steps)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var envelope struct {
		Steps []struct {
			Reason string `json:"reason"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(envelope.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(envelope.Steps))
	}
	if envelope.Steps[0].Reason != "baseline not implemented" {
		t.Errorf("expected reason %q, got %q", "baseline not implemented", envelope.Steps[0].Reason)
	}
}

// TestGate_JSONOutput_SchemaVersionGateV1 verifies GateResult.SchemaVersion is "gate/v1".
func TestGate_JSONOutput_SchemaVersionGateV1(t *testing.T) {
	result := NewGateResult(nil)
	if result.SchemaVersion != "gate/v1" {
		t.Errorf("expected schema_version %q, got %q", "gate/v1", result.SchemaVersion)
	}
}

// TestGate_JSONOutput_SummaryCounts verifies TotalViolations, StepsPassed,
// StepsFailed, and StepsSkipped are computed correctly.
func TestGate_JSONOutput_SummaryCounts(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "v1"},
			{Rule: "lint", Message: "v2"},
		}},
		{StepName: StepTestVerification, Status: "pass", Violations: []Violation{}},
		{StepName: StepTestSubstantiveness, Status: "fail", Violations: []Violation{
			{Rule: "hollow", Message: "v3"},
		}},
		{StepName: StepCoverageThreshold, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"},
		{StepName: StepContractSignature, Status: "pass", Violations: []Violation{}},
		{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, Reason: "baseline not implemented"},
		{StepName: StepWaiverResolution, Status: "skipped", Violations: []Violation{}, Reason: "waivers not implemented"},
		{StepName: StepLedgerIntegrity, Status: "skipped", Violations: []Violation{}, Reason: "ledger not implemented"},
	}
	result := NewGateResult(steps)

	if result.TotalViolations != 3 {
		t.Errorf("expected TotalViolations=3, got %d", result.TotalViolations)
	}
	if result.StepsPassed != 3 {
		t.Errorf("expected StepsPassed=3, got %d", result.StepsPassed)
	}
	if result.StepsFailed != 2 {
		t.Errorf("expected StepsFailed=2, got %d", result.StepsFailed)
	}
	if result.StepsSkipped != 4 {
		t.Errorf("expected StepsSkipped=4, got %d", result.StepsSkipped)
	}
}

// TestGate_JSONOutput_PassTrueWhenAllGreen verifies Pass is true when all steps pass.
func TestGate_JSONOutput_PassTrueWhenAllGreen(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "pass", Violations: []Violation{}},
	}
	result := NewGateResult(steps)
	if !result.Pass {
		t.Error("expected Pass=true when all steps pass")
	}
}

// TestGate_JSONOutput_PassFalseWhenAnyFail verifies Pass is false when any step fails.
func TestGate_JSONOutput_PassFalseWhenAnyFail(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "err"},
		}},
	}
	result := NewGateResult(steps)
	if result.Pass {
		t.Error("expected Pass=false when any step fails")
	}
}

// TestGate_JSONOutput_PassTrueWithSkippedSteps verifies Pass is true when
// steps pass and remaining steps are skipped.
func TestGate_JSONOutput_PassTrueWithSkippedSteps(t *testing.T) {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"},
	}
	result := NewGateResult(steps)
	if !result.Pass {
		t.Error("expected Pass=true when steps pass and remaining are skipped")
	}
}

// TestGate_JSONOutput_PassTrueWhenAllSkipped verifies Pass is true when all
// nine steps are skipped (no executed step failed).
func TestGate_JSONOutput_PassTrueWhenAllSkipped(t *testing.T) {
	var steps []StepResult
	for _, name := range AllStepNames {
		steps = append(steps, StepResult{StepName: name, Status: "skipped", Violations: []Violation{}, Reason: "not implemented"})
	}
	result := NewGateResult(steps)
	if !result.Pass {
		t.Error("expected Pass=true when all steps are skipped")
	}
}

// TestGate_CanonicalStepNames_AllPresent verifies the constant list of nine
// canonical step names.
func TestGate_CanonicalStepNames_AllPresent(t *testing.T) {
	if len(AllStepNames) != 9 {
		t.Fatalf("expected 9 canonical step names, got %d", len(AllStepNames))
	}
}

// TestGate_CanonicalStepNames_ExactMatch verifies step names match exact
// strings from REQ-011.
func TestGate_CanonicalStepNames_ExactMatch(t *testing.T) {
	expected := [9]string{
		"artifact_validation",
		"code_check",
		"test_verification",
		"test_substantiveness",
		"coverage_threshold",
		"contract_signature",
		"baseline_comparison",
		"waiver_resolution",
		"ledger_integrity",
	}
	if AllStepNames != expected {
		t.Errorf("canonical step names mismatch:\n  got:  %v\n  want: %v", AllStepNames, expected)
	}
}
