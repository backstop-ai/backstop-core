package hollowtestgo

import "testing"

// Q1 positive: a hollow test (calls a subject, asserts nothing) → RED finding.
// Under testdata/ so the Go toolchain + the gate coverage step ignore it; it is a
// pack fixture (an ast-grep target), not a compiled unit of backstop.
func helper() int { return 1 }

func TestHollowExample(t *testing.T) {
	helper()
}
