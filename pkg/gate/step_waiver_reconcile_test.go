package gate

import (
	"testing"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/waiver"
)

// waiverTestNow is a stable now well before the far-future token expiries used in
// the gate waiver tests.
var waiverTestNow = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// newTestPolicyNonWaivable builds a declared Policy marking the given rule-ids
// (plus critical severity) non-waivable.
func newTestPolicyNonWaivable(ruleIDs ...string) waiver.Policy {
	return waiver.NewDeclaredPolicy(ruleIDs, []string{"critical"})
}

// memLineReader builds an in-memory waiver.LineReader over a single file's lines.
func memLineReader(file string, lines map[int]string) waiver.LineReader {
	return func(f string, line int) (string, bool) {
		if f != file {
			return "", false
		}
		s, ok := lines[line]
		return s, ok
	}
}

// packEnginesStep builds a synthetic accumulated pack_engines StepResult.
func packEnginesStep(vs ...Violation) StepResult {
	return StepResult{StepName: StepPackEngines, Status: statusFor(vs), Violations: vs}
}

func statusFor(vs []Violation) string {
	if len(vs) > 0 {
		return "fail"
	}
	return "pass"
}

// stepHasViolation reports whether a step's Violations contain a file+line+rule.
func stepHasViolation(steps []StepResult, stepName, file string, line int, rule string) bool {
	for _, s := range steps {
		if s.StepName != stepName {
			continue
		}
		for _, v := range s.Violations {
			if v.File == file && v.Line == line && v.Rule == rule {
				return true
			}
		}
	}
	return false
}

// TestGateWaiver_Suppress_MutatesAccumulatedPackEngines proves the reconciliation
// pass REMOVES suppressed findings from the already-accumulated pack_engines
// violation set — the subtraction actually mutates the accumulated results
// (CLM-067). A pass that computes a result but never removes the finding
// suppresses nothing.
func TestGateWaiver_Suppress_MutatesAccumulatedPackEngines(t *testing.T) {
	accumulated := []StepResult{
		packEnginesStep(Violation{Rule: "pkg/rule-a", File: "app.go", Line: 5, Severity: "error"}),
	}
	read := memLineReader("app.go", map[int]string{
		5: "risky() // @waiver:pkg/rule-a:accepted-risk:2999-01-01",
	})
	g := &Gate{}
	res := g.computeWaiverResult(accumulated, read, nil, waiverTestNow)

	if stepHasViolation(accumulated, StepPackEngines, "app.go", 5, "pkg/rule-a") {
		t.Fatal("suppressed finding was NOT removed from the accumulated pack_engines violations (no real subtraction)")
	}
	if res.StepName != StepWaiverResolution {
		t.Errorf("waiver step name = %q, want %q", res.StepName, StepWaiverResolution)
	}
	if len(g.activeWaivers) != 1 {
		t.Fatalf("expected 1 active waiver recorded, got %d", len(g.activeWaivers))
	}
}

// TestGateWaiver_AuditFeed_ActiveSetExposed proves the active []waiver.Waiver is
// exposed to the step-9 audit surface via ActiveWaiverFeed reading the new
// GateResult.ActiveWaivers carrier (CLM-062).
func TestGateWaiver_AuditFeed_ActiveSetExposed(t *testing.T) {
	waivers := []waiver.Waiver{
		{RuleID: "pkg/rule-a", Reason: waiver.ReasonAcceptedRisk, Expiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	res := GateResult{ActiveWaivers: waivers}
	feed := ActiveWaiverFeed(res)
	if len(feed) != 1 {
		t.Fatalf("ActiveWaiverFeed returned %d waivers, want 1", len(feed))
	}
	if feed[0].RuleID != "pkg/rule-a" {
		t.Errorf("feed rule-id = %q, want pkg/rule-a", feed[0].RuleID)
	}
}

// TestGateWaiver_AuditFeed_EntryCarriesRuleReasonExpiry proves each audit-feed
// entry carries the rule-id, reason-code, and expiry of its waiver (CLM-063).
func TestGateWaiver_AuditFeed_EntryCarriesRuleReasonExpiry(t *testing.T) {
	expiry := time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)
	res := GateResult{ActiveWaivers: []waiver.Waiver{
		{RuleID: "pkg/rule-b", Reason: waiver.ReasonDeferred, Expiry: expiry},
	}}
	feed := ActiveWaiverFeed(res)
	if len(feed) != 1 {
		t.Fatalf("want 1 feed entry, got %d", len(feed))
	}
	e := feed[0]
	if e.RuleID != "pkg/rule-b" || e.Reason != waiver.ReasonDeferred || !e.Expiry.Equal(expiry) {
		t.Errorf("feed entry missing rule/reason/expiry: %+v", e)
	}
}
