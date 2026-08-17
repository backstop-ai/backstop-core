package hollowtestgo

import "testing"

// FIXTURE POLARITY CONVENTION (SPEC-014, BUNDLE-005 REQ-011): the POSITIVE fixture
// is the CLEAN case that must NOT trigger; the NEGATIVE fixture is the VIOLATING
// case that MUST trigger.
//
// VIOLATING: a hollow test (calls a subject, asserts nothing) MUST trigger
// hollow-test-go. `helper` is declared in positive.go, same package — these files
// are under testdata/ and are ast-grep targets, never compiled.
func TestHollowExample(t *testing.T) {
	helper()
}
