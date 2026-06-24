package referencedsymbolgo

import "testing"

// Q2 negative: no package-qualified call → no referenced-symbol finding.
func localHelper() int { return 1 }

func TestNoQualifiedCall(t *testing.T) {
	got := localHelper()
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}
