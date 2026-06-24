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
