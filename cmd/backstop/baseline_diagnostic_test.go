package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// The baseline job's FIRST EVER run (main, 30398137055) failed with exactly this and
// nothing more:
//
//	baseline generate: gate reported a configuration error (exit 2); refusing to
//	write a baseline from a gate that produced no steps
//
// The cause was recoverable — the internal gate had run a step whose ConfigErr was
// set and whose violation named the missing Layer-0 tool — and the wrapper discarded
// it, justified by a comment asserting "result.Steps is empty" that was true for one
// config-error class and assumed for all. Third instance in this lane of the same
// defect family: the information existed and the surfacing did not.

// TestConfigErrorDiagnostic_SurfacesTheStepsOwnReason is the regression lock.
func TestConfigErrorDiagnostic_SurfacesTheStepsOwnReason(t *testing.T) {
	steps := []gate.StepResult{
		{StepName: "pack_engines", Status: "fail", ConfigErr: true,
			Violations: []gate.Violation{{Rule: "pack_engines", Severity: "error",
				Message: "engine tool \"golangci-lint\" not found on PATH"}}},
	}

	got := configErrorDiagnostic(steps)

	if got == "" {
		t.Fatal("a ConfigErr-carrying step produced NO diagnostic; the operator sees only 'exit 2' and " +
			"cannot tell a missing tool from a broken config — the failure this test exists to prevent")
	}
	for _, want := range []string{"pack_engines", "golangci-lint"} {
		if !strings.Contains(got, want) {
			t.Errorf("the diagnostic omits %q; got: %s", want, got)
		}
	}
}

// TestConfigErrorDiagnostic_EmptyWhenGenuinelyStepless keeps the original wording
// honest for the case the old comment actually described.
//
// A gate that returned exit 2 BEFORE building any step has nothing to add, and
// inventing a diagnostic there would be worse than saying less.
func TestConfigErrorDiagnostic_EmptyWhenGenuinelyStepless(t *testing.T) {
	if got := configErrorDiagnostic(nil); got != "" {
		t.Errorf("a step-less gate must add nothing, got: %s", got)
	}
	ok := []gate.StepResult{{StepName: "artifact_validation", Status: "pass"}}
	if got := configErrorDiagnostic(ok); got != "" {
		t.Errorf("no step carried ConfigErr, so there is nothing to surface; got: %s", got)
	}
}
