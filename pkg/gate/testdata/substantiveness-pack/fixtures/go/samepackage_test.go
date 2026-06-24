package gate

import "testing"

// TestSamePackageExample resides in the target package `gate` itself and calls the
// subject directly (no package qualifier) — Q2 GREEN via same-package short-circuit.
func TestSamePackageExample(t *testing.T) {
	got := Run()
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}
