package hollowtestgo

import "testing"

// Q1 negative: a substantive test (has an assertion) → no finding (GREEN).
func TestSubstantiveExample(t *testing.T) {
	got := helper()
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}
