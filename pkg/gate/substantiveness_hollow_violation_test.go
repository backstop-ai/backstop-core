package gate

import (
	"strings"
	"testing"
)

// substantiveness_hollow_violation_test.go drives the hollow-finding →
// test_substantiveness violation conversion: one violation per routed hollow
// finding, preserving the deleted analyzer's "test X has no assertions (hollow)"
// report-surface message. The Violation carries NO gate_type field (Sharp Edge 5),
// so the assertions key on Rule + Message only.

// TestQ1_GateConsumesHollowFindings_RaisesViolation (CLM-005) — the gate raises
// exactly one test_substantiveness violation per routed hollow finding, carrying the
// preserved "test X has no assertions (hollow)" semantics with the pinned routing
// token stripped from the user-facing message.
func TestQ1_GateConsumesHollowFindings_RaisesViolation(t *testing.T) {
	hollow := []Violation{
		{Rule: testHollowRuleID, File: "a_test.go", Message: "test function TestA has no assertions (hollow) func=TestA"},
		{Rule: testHollowRuleID, File: "b_test.go", Message: "test function TestB has no assertions (hollow) func=TestB"},
	}

	got := HollowFindingsToViolations(hollow)

	if len(got) != 2 {
		t.Fatalf("expected exactly one violation per hollow finding (2); got %d: %+v", len(got), got)
	}
	for _, v := range got {
		if v.Rule != StepTestSubstantiveness {
			t.Errorf("violation Rule = %q, want %q", v.Rule, StepTestSubstantiveness)
		}
		if !strings.Contains(v.Message, "has no assertions (hollow)") {
			t.Errorf("violation must preserve the hollow report semantics; got %q", v.Message)
		}
		// The pinned routing token must NOT leak into the user-facing message.
		if strings.Contains(v.Message, "func=") {
			t.Errorf("user-facing message must strip the pinned func= token; got %q", v.Message)
		}
		if v.File == "" {
			t.Errorf("violation must carry the finding's File for scope filtering")
		}
	}

	// The first violation's message is exactly the preserved analyzer phrasing.
	if want := "test function TestA has no assertions (hollow)"; got[0].Message != want {
		t.Errorf("violation[0].Message = %q, want %q", got[0].Message, want)
	}
}

// TestQ1_HollowFindingsToViolations_ForwardsSourceLine (CLM-001, CLM-005) — the
// conversion forwards each source hollow finding's OWN Line onto the violation it
// constructs, so the SPEC-049 waiver reconciliation can byte-scan the finding's own
// line for a @waiver token (ISSUE-116). Two DISTINCT non-zero lines make the
// assertion unsatisfiable by any constant, and the Line-0 input pins that a
// locationless finding is passed through honestly rather than defaulted. The
// non-Line fields are re-asserted because this fix must NOT become ISSUE-106's
// severity change.
func TestQ1_HollowFindingsToViolations_ForwardsSourceLine(t *testing.T) {
	hollow := []Violation{
		{Rule: testHollowRuleID, File: "a_test.go", Line: 12, Message: "test function TestA has no assertions (hollow) func=TestA"},
		{Rule: testHollowRuleID, File: "./b_test.go", Line: 7, Message: "test function TestB has no assertions (hollow) func=TestB"},
		{Rule: testHollowRuleID, File: "c_test.go", Message: "test function TestC has no assertions (hollow) func=TestC"},
	}

	got := HollowFindingsToViolations(hollow)

	if len(got) != len(hollow) {
		t.Fatalf("expected one violation per hollow finding (%d); got %d: %+v", len(hollow), len(got), got)
	}

	// Each violation's Line is ITS OWN source finding's Line — positionally, so a
	// constant (or a swap) cannot satisfy this.
	for i, want := range []int{12, 7, 0} {
		if got[i].Line != want {
			t.Errorf("violation[%d].Line = %d, want %d (the source finding's own line)", i, got[i].Line, want)
		}
	}

	// CLM-005 non-regression: Rule, File, Message and Severity are unchanged.
	wantFiles := []string{"a_test.go", "b_test.go", "c_test.go"}
	wantMessages := []string{
		"test function TestA has no assertions (hollow)",
		"test function TestB has no assertions (hollow)",
		"test function TestC has no assertions (hollow)",
	}
	for i, v := range got {
		if v.Rule != StepTestSubstantiveness {
			t.Errorf("violation[%d].Rule = %q, want %q", i, v.Rule, StepTestSubstantiveness)
		}
		if v.File != wantFiles[i] {
			t.Errorf("violation[%d].File = %q, want the normalized %q", i, v.File, wantFiles[i])
		}
		if v.Message != wantMessages[i] {
			t.Errorf("violation[%d].Message = %q, want %q", i, v.Message, wantMessages[i])
		}
		if v.Severity != "error" {
			t.Errorf("violation[%d].Severity = %q, want %q", i, v.Severity, "error")
		}
	}
}
