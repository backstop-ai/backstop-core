package gate

import (
	"strings"
	"testing"
)

// TestGateWaiver_Output_PrefilledTokenOnBlockedWaivable proves REQ-014's PRODUCTION
// wiring, not just the PrefilledWaiverToken unit: a finding that remains blocked on
// the waivable surface must carry a pre-filled @waiver token that the human gate
// OUTPUT actually emits. This is the regression guard for the "computed-but-never-
// surfaced" dark-wiring gap — the same class the reconciliation pass and construction
// site are guarded against.
func TestGateWaiver_Output_PrefilledTokenOnBlockedWaivable(t *testing.T) {
	v := Violation{Rule: "pkg/rule", File: "app.go", Line: 5, Severity: "error", Message: "boom"}
	accumulated := []StepResult{stepWith(StepPackEngines, v)}
	// No @waiver token on the finding's line → it stays blocked, so it must get a hint.
	read := memLineReader("app.go", map[int]string{5: "code() // nothing to see here"})
	g := &Gate{}
	g.computeWaiverResult(accumulated, read, nil, waiverTestNow)

	out := FormatHuman(GateResult{Steps: accumulated}, true)
	want := "@waiver:pkg/rule:accepted-risk:"
	if !strings.Contains(out, want) {
		t.Fatalf("REQ-014: gate output must hand the author a pre-filled token for a blocked waivable finding; want %q in:\n%s", want, out)
	}
}

// TestGateWaiver_Output_NoTokenForNonWaivableStructural proves the hint is NOT
// emitted for a structural (non-waivable) finding — the affordance only appears
// where a waiver would actually work.
func TestGateWaiver_Output_NoTokenForNonWaivableStructural(t *testing.T) {
	v := Violation{Rule: "artifact_status_drift", File: "specs/x.spec.md", Severity: "error", Message: "drift"}
	accumulated := []StepResult{stepWith(StepArtifactStatusDrift, v)}
	read := memLineReader("specs/x.spec.md", map[int]string{})
	g := &Gate{}
	g.computeWaiverResult(accumulated, read, nil, waiverTestNow)

	out := FormatHuman(GateResult{Steps: accumulated}, true)
	if strings.Contains(out, "@waiver:") {
		t.Fatalf("a structural non-waivable finding must NOT get a pre-filled token:\n%s", out)
	}
}
