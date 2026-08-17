package referencedsymbolgo

import (
	"strings"
	"testing"
)

// FIXTURE POLARITY CONVENTION (SPEC-014, BUNDLE-005 REQ-011): the POSITIVE fixture
// is the CLEAN case that must NOT trigger; the NEGATIVE fixture is the VIOLATING
// case that MUST trigger.
//
// VIOLATING: a package-qualified call inside a test MUST produce a
// referenced-symbol extraction finding. The `strings.ToUpper` hit is the evidence;
// the `t.Fatalf` hit is boilerplate that any substantive Go test carries.
func TestReferencesSymbol(t *testing.T) {
	got := strings.ToUpper("x")
	if got != "X" {
		t.Fatalf("expected X, got %s", got)
	}
}
