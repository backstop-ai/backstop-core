package referencedsymbolgo

import "testing"

// FIXTURE POLARITY CONVENTION (SPEC-014, BUNDLE-005 REQ-011): the POSITIVE fixture
// is the CLEAN case that must NOT trigger; the NEGATIVE fixture is the VIOLATING
// case that MUST trigger. Both rules share one sgconfig (ISSUE-028), so a clean
// fixture must be clean against BOTH.

func localHelper() int { return 1 }

// mustBeOne is NOT Test-named, so referenced-symbol-go's `inside: matches: is-test`
// never reaches into it — a helper may hold t.Fatalf freely.
func mustBeOne(t *testing.T, got int) {
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

// CLEAN: no package-qualified call inside a Test-named function, so
// referenced-symbol-go does not fire; and an assertion-vocabulary call, so
// hollow-test-go does not either.
func TestNoQualifiedCall(t *testing.T) {
	mustBeOne(t, localHelper())
}
