package gate

import (
	"encoding/json"
	"strings"
	"testing"
)

// testGateResult builds a GateResult for output formatting tests.
func testGateResult() GateResult {
	steps := []StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", File: "main.go", Message: "unused var", Severity: "error"},
		}},
		{StepName: StepTestVerification, Status: "pass", Violations: []Violation{}},
		{StepName: StepTestSubstantiveness, Status: "pass", Violations: []Violation{}},
		{StepName: StepCoverageThreshold, Status: "pass", Violations: []Violation{}},
		{StepName: StepContractSignature, Status: "pass", Violations: []Violation{}},
		{StepName: StepBaselineComparison, Status: "skipped", Violations: []Violation{}, Reason: "baseline not implemented"},
		{StepName: StepWaiverResolution, Status: "skipped", Violations: []Violation{}, Reason: "waivers not implemented"},
		{StepName: StepLedgerIntegrity, Status: "skipped", Violations: []Violation{}, Reason: "ledger not implemented"},
	}
	return NewGateResult(steps)
}

// TestGate_JSONOutput_StructureComplete_EndToEnd verifies end-to-end JSON
// formatting produces valid JSON with all required fields.
func TestGate_JSONOutput_StructureComplete_EndToEnd(t *testing.T) {
	result := testGateResult()
	data, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	for _, field := range []string{"schema_version", "pass", "steps", "total_violations"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing field %q in JSON output", field)
		}
	}
}

// TestGate_HumanOutput_ProducedByDefault verifies human output is text, not JSON.
func TestGate_HumanOutput_ProducedByDefault(t *testing.T) {
	result := testGateResult()
	output := FormatHuman(result, true) // noColor=true for predictable output

	// Should not be valid JSON
	var raw json.RawMessage
	if json.Unmarshal([]byte(output), &raw) == nil {
		t.Error("human output should not be valid JSON")
	}

	// Should contain step names
	if !strings.Contains(output, StepArtifactValidation) {
		t.Error("human output should contain step names")
	}
}

// TestGate_HumanOutput_SummaryTable verifies human output displays a summary
// table with step name, status, and violation count.
func TestGate_HumanOutput_SummaryTable(t *testing.T) {
	result := testGateResult()
	output := FormatHuman(result, true)

	// Check that each step name appears
	for _, name := range AllStepNames {
		if !strings.Contains(output, name) {
			t.Errorf("summary table missing step %q", name)
		}
	}

	// Check that statuses appear
	if !strings.Contains(output, "pass") {
		t.Error("summary table missing 'pass' status")
	}
	if !strings.Contains(output, "fail") {
		t.Error("summary table missing 'fail' status")
	}
	if !strings.Contains(output, "skipped") {
		t.Error("summary table missing 'skipped' status")
	}
}

// TestGate_HumanOutput_SkippedStepReason verifies skipped steps show their
// reason in human output.
func TestGate_HumanOutput_SkippedStepReason(t *testing.T) {
	result := testGateResult()
	output := FormatHuman(result, true)

	if !strings.Contains(output, "baseline not implemented") {
		t.Error("human output should show reason for baseline_comparison skip")
	}
	if !strings.Contains(output, "waivers not implemented") {
		t.Error("human output should show reason for waiver_resolution skip")
	}
	if !strings.Contains(output, "ledger not implemented") {
		t.Error("human output should show reason for ledger_integrity skip")
	}
}

// TestGate_HumanOutput_OverallVerdict verifies human output ends with
// overall PASS or FAIL verdict.
func TestGate_HumanOutput_OverallVerdict(t *testing.T) {
	// Test FAIL case
	failResult := testGateResult()
	failOutput := FormatHuman(failResult, true)
	if !strings.Contains(failOutput, "FAIL") {
		t.Error("human output should contain FAIL verdict when steps fail")
	}

	// Test PASS case
	passResult := NewGateResult([]StepResult{
		{StepName: StepArtifactValidation, Status: "pass", Violations: []Violation{}},
	})
	passOutput := FormatHuman(passResult, true)
	if !strings.Contains(passOutput, "PASS") {
		t.Error("human output should contain PASS verdict when all steps pass")
	}
}

// TestGate_HumanOutput_NoColorEnvVar verifies human output respects NO_COLOR
// by not including ANSI escape sequences.
func TestGate_HumanOutput_NoColorEnvVar(t *testing.T) {
	result := testGateResult()
	output := FormatHuman(result, true) // noColor=true

	if strings.Contains(output, "\033[") {
		t.Error("human output with NO_COLOR should not contain ANSI escape sequences")
	}

	// With color enabled, should contain ANSI codes
	colorOutput := FormatHuman(result, false)
	if !strings.Contains(colorOutput, "\033[") {
		t.Error("human output without NO_COLOR should contain ANSI escape sequences")
	}
}
