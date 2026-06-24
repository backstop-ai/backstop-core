package fixtures_test

import "testing"

// TestNoTargetCallExample is substantive but never references the target package
// `gate` (it only calls the helper `other`) — Q2 noTarget (RED).
func TestNoTargetCallExample(t *testing.T) {
	got := other.Compute()
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}
