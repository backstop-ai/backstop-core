package gate

import "testing"

func TestPackSandbox_NativeApplicationEvidenceIsDeterministicWithoutGlobalState(t *testing.T) {
	falseSteps := []StepResult{
		{StepName: "external", Status: "pass", NativeSandboxApplied: false},
		{StepName: "no_sandbox_path", Status: "pass", NativeSandboxApplied: false},
		{StepName: "native_setup_failed", Status: "fail", NativeSandboxApplied: false},
	}
	if result := NewGateResult(falseSteps); result.NativeSandboxApplied {
		t.Fatal("false-only step evidence reduced to true")
	}

	trueStep := StepResult{StepName: "native_acknowledged", Status: "pass"}
	trueStep.NativeSandboxApplied = true
	for _, steps := range [][]StepResult{
		append(append([]StepResult{}, falseSteps...), trueStep),
		append([]StepResult{trueStep}, falseSteps...),
	} {
		if result := NewGateResult(steps); !result.NativeSandboxApplied {
			t.Fatal("acknowledged native evidence was not OR-reduced")
		}
	}

	if result := NewGateResult(falseSteps); result.NativeSandboxApplied {
		t.Fatal("evidence leaked from an independent prior reduction")
	}
}
