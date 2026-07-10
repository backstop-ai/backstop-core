package gate

import (
	"context"
	"testing"
)

// waiverWiringSteps returns a synthetic pipeline: a pack_engines step emitting one
// finding at app.go:5, followed by the registered placeholder waiver step.
func waiverWiringSteps() []StepFunc {
	pe := func(context.Context) StepResult {
		return StepResult{
			StepName:   StepPackEngines,
			Status:     "fail",
			Violations: []Violation{{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"}},
		}
	}
	return []StepFunc{pe, StepWaiverResolutionScopedFunc(nil)}
}

// TestGateWaiver_Wiring_WithWaiverEnablesReconciliation proves the WithWaiver
// Option sets g.waiverEnabled and the Run-loop swaps StepWaiverResolution's
// placeholder for computeWaiverResult (a co-located @waiver over a synthetic
// pack_engines StepResult is suppressed); constructing WITHOUT WithWaiver leaves
// the swap un-fired and nothing is suppressed (CLM-068).
func TestGateWaiver_Wiring_WithWaiverEnablesReconciliation(t *testing.T) {
	read := memLineReader("app.go", map[int]string{
		5: "risky() // @waiver:pkg/rule-a:accepted-risk:2999-01-01",
	})

	// WITH WithWaiver: the swap fires — the finding is suppressed.
	enabled := New(WithSteps(waiverWiringSteps()), WithWaiver(read, nil, waiverTestNow))
	res, _ := enabled.Run(context.Background())
	if stepHasViolation(res.Steps, StepPackEngines, "app.go", 5, "pkg/rule-a") {
		t.Fatal("with WithWaiver the reconciliation swap must fire and suppress the co-located finding")
	}
	if len(res.ActiveWaivers) != 1 {
		t.Fatalf("with WithWaiver the active waiver must be persisted onto GateResult.ActiveWaivers, got %d", len(res.ActiveWaivers))
	}

	// WITHOUT WithWaiver: the swap does NOT fire — the finding stands.
	disabled := New(WithSteps(waiverWiringSteps()))
	res2, _ := disabled.Run(context.Background())
	if !stepHasViolation(res2.Steps, StepPackEngines, "app.go", 5, "pkg/rule-a") {
		t.Fatal("without WithWaiver nothing must be suppressed (the swap is guarded on g.waiverEnabled)")
	}
	if len(res2.ActiveWaivers) != 0 {
		t.Fatalf("without WithWaiver no active waivers must be recorded, got %d", len(res2.ActiveWaivers))
	}
}
