package gate

import (
	"strings"
	"testing"
	"time"
)

// TestGateWaiver_Report_DistinctPassWithWaiversState proves a run passing BECAUSE
// of active waivers renders the distinct `PASS · N waivers` terminal state
// (CLM-050).
func TestGateWaiver_Report_DistinctPassWithWaiversState(t *testing.T) {
	accumulated := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"}),
	}
	read := memLineReader("app.go", map[int]string{
		5: "risky() // @waiver:pkg/rule-a:accepted-risk:2999-01-01",
	})
	g := &Gate{}
	res := g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if res.Status != "pass" {
		t.Fatalf("a run passing on active waivers must be a passing waiver step, got %q", res.Status)
	}
	if !strings.Contains(res.Reason, "PASS · 1 waivers") {
		t.Fatalf("waiver step must render the distinct `PASS · N waivers` state, got Reason=%q", res.Reason)
	}
}

// TestGateWaiver_Report_CleanRunNotWaiverState proves a clean run with zero
// waivers does NOT render as the waiver state (CLM-051).
func TestGateWaiver_Report_CleanRunNotWaiverState(t *testing.T) {
	accumulated := []StepResult{packEnginesStep()}
	read := memLineReader("app.go", map[int]string{})
	g := &Gate{}
	res := g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if strings.Contains(res.Reason, "PASS · ") {
		t.Fatalf("a zero-waiver run must NOT render the distinct waiver state, got Reason=%q", res.Reason)
	}
}

// TestGateWaiver_Report_SummaryAlwaysShown proves the active-waiver summary is
// always shown on a run with active waivers (CLM-052).
func TestGateWaiver_Report_SummaryAlwaysShown(t *testing.T) {
	accumulated := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"}),
	}
	read := memLineReader("app.go", map[int]string{
		5: "risky() // @waiver:pkg/rule-a:accepted-risk:2999-01-01",
	})
	g := &Gate{}
	res := g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if !strings.Contains(res.Reason, "pkg/rule-a") {
		t.Fatalf("the active-waiver summary must name the waived rule, got Reason=%q", res.Reason)
	}
}

// TestGateWaiver_Report_ActionableSubsetInline proves the actionable subset
// (expiring-soon and unused) is surfaced inline on every run (CLM-053).
func TestGateWaiver_Report_ActionableSubsetInline(t *testing.T) {
	expiringDate := waiverTestNow.Add(9 * 24 * time.Hour).Format("2006-01-02")
	accumulated := []StepResult{
		packEnginesStep(
			Violation{Rule: "pkg/expiring", File: "app.go", Line: 8, Severity: "error"},
			Violation{Rule: "pkg/real", File: "app.go", Line: 12, Severity: "error"},
		),
	}
	read := memLineReader("app.go", map[int]string{
		8:  "risky() // @waiver:pkg/expiring:accepted-risk:" + expiringDate,
		11: "// @waiver:pkg/ghost:deferred:2999-01-01",
		12: "risky()",
	})
	g := &Gate{}
	res := g.computeWaiverResult(accumulated, read, nil, waiverTestNow)
	if !strings.Contains(res.Reason, "expiring") {
		t.Errorf("the actionable subset must surface expiring-soon waivers inline, got Reason=%q", res.Reason)
	}
	if !strings.Contains(res.Reason, "unused") {
		t.Errorf("the actionable subset must surface unused/dangling waivers inline, got Reason=%q", res.Reason)
	}
}
