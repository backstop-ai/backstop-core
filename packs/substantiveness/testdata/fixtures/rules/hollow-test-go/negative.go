package hollowtestgo

import (
	"os"
	"testing"
)

// Q1 negative: a substantive test (has an assertion) → no finding (GREEN).
func TestSubstantiveExample(t *testing.T) {
	got := helper()
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

// Q1 negative: TestMain(m *testing.M) is Go's harness entry point and is BY DESIGN
// never assertion-bearing. The hollow-test rule EXEMPTS it by name, so it produces
// no finding (GREEN) even though it holds no assertion (ISSUE-035 CLM-001).
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
