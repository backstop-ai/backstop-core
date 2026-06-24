package gate

import (
	"encoding/json"
	"testing"
)

// TestNewGateResultWithScope_WarningIsNonFailing_IncrementsStepsWarned
// (CLM-011): a "warning"-status step does NOT set Pass=false (the gate still
// exits 0) but DOES increment the new StepsWarned counter. A warning is a
// non-failing, counted status — neither treated like "fail" (which would block
// class 2, the outcome the bundle prohibits) nor silently dropped (which would
// vanish from the summary counts).
func TestNewGateResultWithScope_WarningIsNonFailing_IncrementsStepsWarned(t *testing.T) {
	steps := []StepResult{
		{StepName: StepTestSubstantiveness, Status: "pass", Violations: []Violation{}},
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{
			{Rule: "coverage", Message: "coverage capability absent; declare or waive", Severity: "warning"},
		}},
	}
	r := NewGateResultWithScope(steps, nil)

	if !r.Pass {
		t.Error("a warning-only (no fail) gate must still Pass (exit 0); warning must not flip Pass")
	}
	if r.StepsWarned != 1 {
		t.Errorf("StepsWarned = %d, want 1 (warning must be counted)", r.StepsWarned)
	}
	if r.StepsFailed != 0 {
		t.Errorf("StepsFailed = %d, want 0 (a warning is not a failure)", r.StepsFailed)
	}
	if r.StepsPassed != 1 {
		t.Errorf("StepsPassed = %d, want 1", r.StepsPassed)
	}
}

// TestNewGateResultWithScope_WarningWithFailure_StillFails confirms the
// exit-code path does not regress: a gate carrying a warning AND a fail still
// fails (Pass=false) — the warning is non-failing but does not mask a real
// failure (CLM-017 negative companion).
func TestNewGateResultWithScope_WarningWithFailure_StillFails(t *testing.T) {
	steps := []StepResult{
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{}},
		{StepName: StepCodeCheck, Status: "fail", Violations: []Violation{
			{Rule: "lint", Message: "boom", Severity: "error"},
		}},
	}
	r := NewGateResultWithScope(steps, nil)
	if r.Pass {
		t.Error("a gate with a real fail must not Pass even when warnings are present")
	}
	if r.StepsWarned != 1 {
		t.Errorf("StepsWarned = %d, want 1", r.StepsWarned)
	}
	if r.StepsFailed != 1 {
		t.Errorf("StepsFailed = %d, want 1", r.StepsFailed)
	}
}

// TestGateResult_StepsWarned_SerializedInJSON (CLM-016 counter half): the
// StepsWarned counter is serialized in JSON so a class-2 advisory is
// machine-readable via the summary counter, not only per-step.
func TestGateResult_StepsWarned_SerializedInJSON(t *testing.T) {
	steps := []StepResult{
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{}},
	}
	r := NewGateResultWithScope(steps, nil)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["steps_warned"]; !ok {
		t.Error("JSON output must contain steps_warned field")
	}
}
