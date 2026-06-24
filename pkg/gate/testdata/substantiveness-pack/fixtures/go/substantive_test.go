package fixtures_test

import "testing"

// TestSubstantiveExample has an assertion-shaped call (t.Fatalf) — substantive (Q1 GREEN).
func TestSubstantiveExample(t *testing.T) {
	got := gate.Build()
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}
