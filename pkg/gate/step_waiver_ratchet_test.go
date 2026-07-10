package gate

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGateWaiver_Ratchet_ActiveSatisfies proves a valid ACTIVE waiver satisfies
// the file-level ratchet for its finding — the finding is subtracted from the
// accumulated set BEFORE baseline captures NewViolations, so it does not count
// (CLM-055).
func TestGateWaiver_Ratchet_ActiveSatisfies(t *testing.T) {
	accumulated := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"}),
	}
	read := memLineReader("app.go", map[int]string{
		5: "risky() // @waiver:pkg/rule-a:accepted-risk:2999-01-01",
	})
	g := &Gate{}
	g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if stepHasViolation(accumulated, StepPackEngines, "app.go", 5, "pkg/rule-a") {
		t.Fatal("an ACTIVE waiver must subtract its finding so the ratchet does not count it")
	}
	if len(g.activeWaivers) != 1 {
		t.Fatalf("the active waiver must be recorded, got %d", len(g.activeWaivers))
	}
}

// TestGateWaiver_Ratchet_ExpiredDoesNotSatisfy proves an EXPIRED waiver does NOT
// satisfy the ratchet — the live finding stands and demands action (CLM-056).
func TestGateWaiver_Ratchet_ExpiredDoesNotSatisfy(t *testing.T) {
	accumulated := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"}),
	}
	read := memLineReader("app.go", map[int]string{
		5: "risky() // @waiver:pkg/rule-a:accepted-risk:2026-05-01",
	})
	g := &Gate{}
	g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if !stepHasViolation(accumulated, StepPackEngines, "app.go", 5, "pkg/rule-a") {
		t.Fatal("an EXPIRED waiver must NOT subtract its finding; the live finding must stand")
	}
	if len(g.activeWaivers) != 0 {
		t.Fatalf("an expired waiver must not be recorded active, got %d", len(g.activeWaivers))
	}
}

// TestGateWaiver_Ratchet_UnusedSatisfiesNothing proves an UNUSED waiver satisfies
// nothing (CLM-057).
func TestGateWaiver_Ratchet_UnusedSatisfiesNothing(t *testing.T) {
	accumulated := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/real", File: "app.go", Line: 5, Severity: "error"}),
	}
	read := memLineReader("app.go", map[int]string{
		4: "// @waiver:pkg/ghost:deferred:2999-01-01",
		5: "risky()",
	})
	g := &Gate{}
	g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if !stepHasViolation(accumulated, StepPackEngines, "app.go", 5, "pkg/real") {
		t.Fatal("an UNUSED (dangling) waiver must subtract nothing; the live finding must stand")
	}
	if len(g.activeWaivers) != 0 {
		t.Fatalf("a dangling waiver must not be recorded active, got %d", len(g.activeWaivers))
	}
}

// TestGateWaiver_Ratchet_BaselineDoesNotAuthorWaivers proves baseline generation
// authors NO @waiver tokens — baseline generation and waiver authoring stay
// DISTINCT operations (CLM-058). A machine snapshot of findings never writes a
// waiver token.
func TestGateWaiver_Ratchet_BaselineDoesNotAuthorWaivers(t *testing.T) {
	steps := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error", Message: "some finding"}),
	}
	artifact := NewBaselineArtifactFromSteps(steps, "2026-06-01T00:00:00Z", "deadbeef", "test")
	data, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("marshaling baseline: %v", err)
	}
	if strings.Contains(string(data), "@waiver") {
		t.Fatal("baseline generation must NOT author any @waiver token")
	}
}
