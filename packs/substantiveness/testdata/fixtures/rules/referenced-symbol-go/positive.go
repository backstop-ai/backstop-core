package referencedsymbolgo

import (
	"strings"
	"testing"
)

// Q2 positive: a package-qualified call inside a test → a referenced-symbol finding.
func TestReferencesSymbol(t *testing.T) {
	got := strings.ToUpper("x")
	if got != "X" {
		t.Fatalf("expected X, got %s", got)
	}
}
