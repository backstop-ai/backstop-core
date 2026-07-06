package fixtures_test

import (
	"os"
	"testing"
)

// TestMain is Go's harness entry point (TestMain(m *testing.M)). It is BY DESIGN
// never assertion-bearing — it only sets up/tears down and delegates to m.Run —
// so the Q1 hollow-test rule must NOT flag it as hollow (ISSUE-035 Category 1 /
// CLM-001). It shares this fixture file with a genuine hollow stub so a single
// ast-grep pass proves the exemption is NAME-scoped, not a blanket suppression.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// TestGenuinelyHollowStub calls a subject and asserts nothing — a genuine hollow
// test. The TestMain exemption must NOT excuse it: it MUST still produce a hollow
// finding in the same pass that exempts TestMain (over-correction guard / CLM-002).
func TestGenuinelyHollowStub(t *testing.T) {
	gate.Build()
}
