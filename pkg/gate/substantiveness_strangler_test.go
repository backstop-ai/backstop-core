package gate

import (
	"path/filepath"
	"testing"
)

// substantiveness_strangler_test.go is the strangler-equivalence pass (REQ-006 / DD-9):
// for each Phase-1 Go verdict-matrix fixture it computes BOTH the pack-path verdict
// (real ast-grep Q1 hollow + Q2 extraction → route → key → set-join) AND the LIVE
// pre-deletion go/parser analyzer verdict (checkSubstantiveness), asserting they MATCH
// per cell. This parity proof LICENSES the Phase-6 analyzer deletion (Sharp Edge 3) and
// MUST pass BEFORE that deletion. Real ast-grep over real fixtures — a stub emitting
// nothing fails the hollow-fixture equivalence (CLM-023). ast-grep absence is a t.Fatal
// (NOT t.Skip): a skipped real-engine equivalence pass is a silent gap.

// stranglerFixture resolves a verdict-matrix fixture and runs both the pack-path and
// the live-analyzer verdict, asserting equivalence on (hollow, noTarget).
func assertStranglerEquivalence(t *testing.T, fixtureFile, funcName string) (packHollow, packNoTarget bool) {
	t.Helper()
	requireAstGrep(t)
	packDir := substPackDir(t)
	filePath := filepath.Join(packDir, "fixtures", "go", fixtureFile)
	const targetPkg = "gate"

	ph, pnt, err := packVerdict(packDir, filePath, funcName, targetPkg)
	if err != nil {
		t.Fatalf("packVerdict(%s): %v", fixtureFile, err)
	}
	ah, ant := analyzerVerdict(filePath, funcName, targetPkg)

	if ph != ah {
		t.Errorf("%s: pack hollow=%v != analyzer hollow=%v", fixtureFile, ph, ah)
	}
	if pnt != ant {
		t.Errorf("%s: pack noTarget=%v != analyzer noTarget=%v", fixtureFile, pnt, ant)
	}
	return ph, pnt
}

// TestStrangler_Go_Hollow_PackEqualsAnalyzer (CLM-018) — hollow fixture: pack Q1 RED
// equals analyzer hollow verdict.
func TestStrangler_Go_Hollow_PackEqualsAnalyzer(t *testing.T) {
	hollow, noTarget := assertStranglerEquivalence(t, "hollow_test.go", "TestHollowExample")
	if !hollow {
		t.Errorf("hollow fixture must be hollow (Q1 RED) on the pack path")
	}
	if noTarget {
		t.Errorf("hollow fixture references the target — noTarget must be false")
	}
}

// TestStrangler_Go_Substantive_PackEqualsAnalyzer (CLM-019) — substantive fixture:
// pack Q1 GREEN equals analyzer substantive verdict.
func TestStrangler_Go_Substantive_PackEqualsAnalyzer(t *testing.T) {
	hollow, _ := assertStranglerEquivalence(t, "substantive_test.go", "TestSubstantiveExample")
	if hollow {
		t.Errorf("substantive fixture must NOT be hollow (Q1 GREEN) on the pack path")
	}
}

// TestStrangler_Go_NoTarget_PackEqualsAnalyzer (CLM-020) — no-target fixture: pack Q2
// noTarget RED equals analyzer noTarget verdict.
func TestStrangler_Go_NoTarget_PackEqualsAnalyzer(t *testing.T) {
	_, noTarget := assertStranglerEquivalence(t, "notarget_test.go", "TestNoTargetCallExample")
	if !noTarget {
		t.Errorf("no-target fixture must raise noTarget (Q2 RED) on the pack path")
	}
}

// TestStrangler_Go_CallsTarget_PackEqualsAnalyzer (CLM-021) — calls-target fixture:
// pack Q2 GREEN equals analyzer calls-target verdict.
func TestStrangler_Go_CallsTarget_PackEqualsAnalyzer(t *testing.T) {
	_, noTarget := assertStranglerEquivalence(t, "callstarget_test.go", "TestCallsTargetExample")
	if noTarget {
		t.Errorf("calls-target fixture references the target — noTarget must be false (Q2 GREEN)")
	}
}

// TestStrangler_Go_SamePackage_PackEqualsAnalyzer (CLM-022) — same-package fixture:
// pack Q2 GREEN (same-package satisfaction) equals analyzer short-circuit verdict.
func TestStrangler_Go_SamePackage_PackEqualsAnalyzer(t *testing.T) {
	_, noTarget := assertStranglerEquivalence(t, "samepackage_test.go", "TestSamePackageExample")
	if noTarget {
		t.Errorf("same-package fixture must satisfy the join via short-circuit — noTarget must be false")
	}
}

// TestStrangler_NotSatisfiableByStub_RealFindingsRequired (CLM-023) — the equivalence
// pass is unsatisfiable by a stub: a pack-path verdict computed from EMPTY findings
// (the stub case) diverges from the live analyzer's hollow verdict on the hollow
// fixture, so a silent-gap pack FAILS rather than passing vacuously.
func TestStrangler_NotSatisfiableByStub_RealFindingsRequired(t *testing.T) {
	requireAstGrep(t)
	packDir := substPackDir(t)
	filePath := filepath.Join(packDir, "fixtures", "go", "hollow_test.go")
	const targetPkg = "gate"

	// The live analyzer says the hollow fixture IS hollow.
	analyzerHollow, _ := analyzerVerdict(filePath, "TestHollowExample", targetPkg)
	if !analyzerHollow {
		t.Fatalf("precondition: the live analyzer must report the hollow fixture as hollow")
	}

	// A STUB pack emits nothing → no hollow finding → IsTestHollow is false. That
	// DIVERGES from the analyzer's hollow verdict, so the equivalence claim cannot be
	// satisfied vacuously by an empty pack.
	stubHollow := IsTestHollow(nil, MandatedTest{FuncName: "TestHollowExample", FilePath: filePath})
	if stubHollow == analyzerHollow {
		t.Fatalf("a stub (empty findings) must NOT reproduce the analyzer's hollow verdict — " +
			"the equivalence pass would be vacuous")
	}

	// The REAL pack path, by contrast, DOES reproduce it (real findings required).
	realHollow, _, err := packVerdict(packDir, filePath, "TestHollowExample", targetPkg)
	if err != nil {
		t.Fatalf("packVerdict (real): %v", err)
	}
	if realHollow != analyzerHollow {
		t.Errorf("the REAL pack path must reproduce the analyzer's hollow verdict; got %v want %v", realHollow, analyzerHollow)
	}
}
