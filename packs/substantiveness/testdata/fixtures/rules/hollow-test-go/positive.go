package hollowtestgo

import "testing"

// FIXTURE POLARITY CONVENTION (SPEC-014, BUNDLE-005 REQ-011): the POSITIVE fixture
// is the CLEAN case that must NOT trigger; the NEGATIVE fixture is the VIOLATING
// case that MUST trigger. Misreading that convention is what produced ISSUE-148.
//
// Both of this pack's rules ship in ONE ast-grep/sgconfig.yml (ISSUE-028), so
// packval's per-fixture verdict is "did ANY rule fire". A clean fixture must
// therefore be clean against BOTH rules. Files under testdata/ are ast-grep
// targets, never compiled by the Go toolchain.

func helper() int { return 1 }

// mustEqual is NOT Test-named, so referenced-symbol-go's `inside: matches: is-test`
// never reaches into it — a helper may hold t.Fatalf freely.
func mustEqual(t *testing.T, got, want int) {
	if got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
}

// CLEAN: triggers neither rule. The assertion is the UNQUALIFIED call `mustEqual` —
// its identifier matches the hollow rule's assertion vocabulary (`must`), so this
// test is not hollow, while staying a plain identifier call rather than a
// selector_expression, so the Q2 extraction rule does not fire. A `t.Fatalf` here
// would fire Q2 and make this fixture a false positive.
func TestSubstantiveExample(t *testing.T) {
	mustEqual(t, helper(), 1)
}

// TestMain is Go's harness entry point and is BY DESIGN never assertion-bearing;
// the hollow rule EXEMPTS it by name (ISSUE-035 CLM-001). The body is EMPTY on
// purpose: `os.Exit(m.Run())` would fire the Q2 rule. Empty, it is clean ONLY
// because the exemption holds — which is what makes it a real pin on the exemption.
func TestMain(m *testing.M) {}
