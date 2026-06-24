package fixtures_test

import "testing"

// TestCallsTargetExample is substantive (t.Fatalf) and references the target package
// `gate` via a package-qualified call — Q2 GREEN.
func TestCallsTargetExample(t *testing.T) {
	got := gate.Build()
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}
