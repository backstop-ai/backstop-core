package gate

import "testing"

// substantiveness_join_q1_test.go drives the Q1 consumption helper (IsTestHollow):
// given the hollow partition returned by RouteSubstantivenessFindings, the helper
// maps each routed hollow finding back to its mandated test (by File + the func read
// from the finding's structured Properties, ISSUE-062) and reports whether a given
// test is hollow — language-agnostically (Go and TS findings flow identically).
// Driven by SYNTHETIC routed findings here; the REAL ast-grep end-to-end assertions
// (the mandated CLM-003/004/012/013/014 names) live in the Phase-3 real-engine file.
// These helper-unit names are DISTINCT and must NOT reuse the mandated names.

// TestQ1Helper_RoutedHollow_PerTestVerdict — a routed hollow finding for test X
// makes X hollow; a test with no hollow finding is not hollow.
func TestQ1Helper_RoutedHollow_PerTestVerdict(t *testing.T) {
	hollow := []Violation{
		{Rule: testHollowRuleID, File: "x_test.go", Properties: map[string]string{"func": "TestX"}},
	}
	testX := MandatedTest{FuncName: "TestX", FilePath: "x_test.go"}
	testY := MandatedTest{FuncName: "TestY", FilePath: "y_test.go"}

	if !IsTestHollow(hollow, testX) {
		t.Errorf("TestX has a routed hollow finding — IsTestHollow should be true")
	}
	if IsTestHollow(hollow, testY) {
		t.Errorf("TestY has NO hollow finding — IsTestHollow should be false")
	}
}

// TestQ1Helper_HollowKeyedByFuncNotJustFile — two tests in the SAME file: only the
// one whose Properties[func] matches the hollow finding is hollow, proving the verdict
// keys on (File, func), not file alone.
func TestQ1Helper_HollowKeyedByFuncNotJustFile(t *testing.T) {
	hollow := []Violation{
		{Rule: testHollowRuleID, File: "shared_test.go", Properties: map[string]string{"func": "TestHollow"}},
	}
	hollowTest := MandatedTest{FuncName: "TestHollow", FilePath: "shared_test.go"}
	substantiveTest := MandatedTest{FuncName: "TestSubstantive", FilePath: "shared_test.go"}

	if !IsTestHollow(hollow, hollowTest) {
		t.Errorf("TestHollow should be hollow (its Properties[func] matches)")
	}
	if IsTestHollow(hollow, substantiveTest) {
		t.Errorf("TestSubstantive in the same file must NOT inherit the hollow verdict (func differs)")
	}
}

// TestQ1Helper_TsHollowFlowsIdentically — a TS hollow finding (structured func
// property) keys the same way as a Go finding, proving the helper is language-agnostic
// (the gate consumption the TS proof rides — CLM-014's gate side).
func TestQ1Helper_TsHollowFlowsIdentically(t *testing.T) {
	hollow := []Violation{
		{Rule: "backstop/ts-proof/hollow-test-ts", File: "hollow.test.ts", Properties: map[string]string{"func": "ts-hollow"}},
	}
	tsTest := MandatedTest{FuncName: "ts-hollow", FilePath: "hollow.test.ts"}
	if !IsTestHollow(hollow, tsTest) {
		t.Errorf("a TS hollow finding with a structured func property must flow through IsTestHollow identically")
	}
}
