package gate

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// substantiveness_q1_findings_test.go runs REAL ast-grep over the Phase-1 fixtures
// through the pkg/gate dispatch glue (dispatchAstGrepRule) and resolves the verdict
// through the Phase-2 consumption helpers (RouteSubstantivenessFindings / IsTestHollow).
// These tests exercise the real pack/engine/gate seam — a stub emitting nothing fails
// the hollow claims. ast-grep absence is a t.Fatal (NOT t.Skip): a skipped real-engine
// test is a silent gap (the exact vacuous-green failure mode the spec exists to kill).

const (
	substPackName          = "backstop/substantiveness"
	substHollowRuleID      = "backstop/substantiveness/hollow-test-go"
	substExtractionRuleID  = "backstop/substantiveness/referenced-symbol-go"
	tsProofPackName        = "backstop/ts-proof"
	tsProofHollowRuleID    = "backstop/ts-proof/hollow-test-ts"
)

// requireAstGrep fails loud (NOT skip) if ast-grep is absent — a skip would re-hide a
// real-engine gap (vacuous green).
func requireAstGrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Fatalf("ast-grep binary not found on PATH: %v — install ast-grep 0.43.0 "+
			"(e.g. `brew install ast-grep`); this real-engine test MUST NOT be skipped "+
			"(a skip is silent vacuous green)", err)
	}
}

func substPackDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", "substantiveness-pack"))
	if err != nil {
		t.Fatalf("resolving substantiveness pack dir: %v", err)
	}
	return abs
}

func tsProofPackDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", "ts-proof-pack"))
	if err != nil {
		t.Fatalf("resolving ts-proof pack dir: %v", err)
	}
	return abs
}

// TestQ1_Go_HollowTest_ProducesFinding (CLM-003) — the Go hollow fixture produces an
// ast-grep hollow-test SARIF finding (RED) through the real dispatch path, and the
// Phase-2 helper maps it back to the mandated test as hollow.
func TestQ1_Go_HollowTest_ProducesFinding(t *testing.T) {
	requireAstGrep(t)
	packDir := substPackDir(t)
	fixture := filepath.Join(packDir, "fixtures", "go", "hollow_test.go")

	findings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-go.yml", "hollow-test-go", substPackName, fixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (Go hollow): %v", err)
	}
	hollow, _ := RouteSubstantivenessFindings(findings, substHollowRuleID, substExtractionRuleID)
	if len(hollow) == 0 {
		t.Fatalf("a hollow Go test must produce a hollow-test finding (RED); got none — a stub/empty pack would fail here")
	}
	mt := MandatedTest{FuncName: "TestHollowExample", FilePath: fixture}
	if !IsTestHollow(hollow, mt) {
		t.Errorf("the hollow finding must map back to TestHollowExample as hollow")
	}
}

// TestQ1_Go_SubstantiveTest_ProducesNoFinding (CLM-004) — the Go substantive fixture
// produces NO hollow-test finding (GREEN) through the real dispatch path.
func TestQ1_Go_SubstantiveTest_ProducesNoFinding(t *testing.T) {
	requireAstGrep(t)
	packDir := substPackDir(t)
	fixture := filepath.Join(packDir, "fixtures", "go", "substantive_test.go")

	findings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-go.yml", "hollow-test-go", substPackName, fixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (Go substantive): %v", err)
	}
	hollow, _ := RouteSubstantivenessFindings(findings, substHollowRuleID, substExtractionRuleID)
	if len(hollow) != 0 {
		t.Fatalf("a substantive Go test must produce NO hollow-test finding (GREEN); got %d: %+v", len(hollow), hollow)
	}
	mt := MandatedTest{FuncName: "TestSubstantiveExample", FilePath: fixture}
	if IsTestHollow(hollow, mt) {
		t.Errorf("TestSubstantiveExample must NOT be hollow")
	}
}

// TestQ1_Go_TestMain_ProducesNoHollowFinding (ISSUE-035 CLM-001) — TestMain(m *testing.M),
// Go's harness entry point, is BY DESIGN never assertion-bearing and must NOT be flagged
// hollow. Running the substantiveness Q1 rule over a fixture containing both TestMain and
// a genuine hollow stub, the routed hollow findings must NOT key back to TestMain.
func TestQ1_Go_TestMain_ProducesNoHollowFinding(t *testing.T) {
	requireAstGrep(t)
	packDir := substPackDir(t)
	fixture := filepath.Join(packDir, "fixtures", "go", "testmain_fixture_test.go")

	findings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-go.yml", "hollow-test-go", substPackName, fixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (TestMain fixture): %v", err)
	}
	hollow, _ := RouteSubstantivenessFindings(findings, substHollowRuleID, substExtractionRuleID)

	mtMain := MandatedTest{FuncName: "TestMain", FilePath: fixture}
	if IsTestHollow(hollow, mtMain) {
		t.Errorf("TestMain must be EXEMPT from the hollow-test rule (harness entry point, never asserts); got a hollow finding for it: %+v", hollow)
	}
}

// TestQ1_Go_TestMainExemption_StillFlagsGenuineHollow (ISSUE-035 CLM-002) — the
// over-correction guard: the TestMain exemption must NOT blanket-suppress the rule. In
// the SAME fixture pass that exempts TestMain, a genuinely hollow stub in the same file
// MUST still produce a hollow finding that keys back to it.
func TestQ1_Go_TestMainExemption_StillFlagsGenuineHollow(t *testing.T) {
	requireAstGrep(t)
	packDir := substPackDir(t)
	fixture := filepath.Join(packDir, "fixtures", "go", "testmain_fixture_test.go")

	findings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-go.yml", "hollow-test-go", substPackName, fixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (TestMain fixture): %v", err)
	}
	hollow, _ := RouteSubstantivenessFindings(findings, substHollowRuleID, substExtractionRuleID)

	if len(hollow) == 0 {
		t.Fatalf("the genuine hollow stub must still produce a hollow finding even with the TestMain exemption; got none")
	}
	mtStub := MandatedTest{FuncName: "TestGenuinelyHollowStub", FilePath: fixture}
	if !IsTestHollow(hollow, mtStub) {
		t.Errorf("TestGenuinelyHollowStub must STILL be flagged hollow (exemption is name-scoped to TestMain, not blanket)")
	}
}

// TestTS_HollowTestTs_ProducesFinding_RealAstGrep (CLM-012) — the hollow .test.ts
// fixture produces an ast-grep hollow-test SARIF finding via real ast-grep on the
// shared TS proof pack.
func TestTS_HollowTestTs_ProducesFinding_RealAstGrep(t *testing.T) {
	requireAstGrep(t)
	packDir := tsProofPackDir(t)
	fixture := filepath.Join(packDir, "fixtures", "ts", "hollow.test.ts")

	findings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-ts.yml", "hollow-test-ts", tsProofPackName, fixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (TS hollow): %v", err)
	}
	hollow, _ := RouteSubstantivenessFindings(findings, tsProofHollowRuleID, "")
	if len(hollow) == 0 {
		t.Fatalf("a hollow .test.ts must produce a hollow-test finding via real ast-grep; got none")
	}
}

// TestTS_SubstantiveTestTs_ProducesNoFinding_RealAstGrep (CLM-013) — the substantive
// .test.ts fixture produces NO hollow-test finding via real ast-grep.
func TestTS_SubstantiveTestTs_ProducesNoFinding_RealAstGrep(t *testing.T) {
	requireAstGrep(t)
	packDir := tsProofPackDir(t)
	fixture := filepath.Join(packDir, "fixtures", "ts", "substantive.test.ts")

	findings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-ts.yml", "hollow-test-ts", tsProofPackName, fixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (TS substantive): %v", err)
	}
	hollow, _ := RouteSubstantivenessFindings(findings, tsProofHollowRuleID, "")
	if len(hollow) != 0 {
		t.Fatalf("a substantive .test.ts must produce NO hollow-test finding; got %d: %+v", len(hollow), hollow)
	}
}

// TestTS_SubstantivenessRidesSharedDispatch_NotStub (CLM-014) — the TS rule rides the
// SAME ast-grep dispatch path (dispatchAstGrepRule) as the Go rule, and the claim is
// unsatisfiable by a stub: the hollow .test.ts yields a real finding AND the substantive
// one yields none, so a stub emitting nothing (or everything) fails one polarity.
func TestTS_SubstantivenessRidesSharedDispatch_NotStub(t *testing.T) {
	requireAstGrep(t)
	packDir := tsProofPackDir(t)

	hollowFixture := filepath.Join(packDir, "fixtures", "ts", "hollow.test.ts")
	substFixture := filepath.Join(packDir, "fixtures", "ts", "substantive.test.ts")

	hf, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-ts.yml", "hollow-test-ts", tsProofPackName, hollowFixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (TS hollow, shared path): %v", err)
	}
	sf, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-ts.yml", "hollow-test-ts", tsProofPackName, substFixture)
	if err != nil {
		t.Fatalf("dispatchAstGrepRule (TS substantive, shared path): %v", err)
	}
	hollowH, _ := RouteSubstantivenessFindings(hf, tsProofHollowRuleID, "")
	hollowS, _ := RouteSubstantivenessFindings(sf, tsProofHollowRuleID, "")

	// Both polarities must hold — a stub cannot satisfy RED-on-hollow AND GREEN-on-
	// substantive simultaneously through the same shared dispatch.
	if len(hollowH) == 0 {
		t.Errorf("hollow .test.ts must produce a finding through the shared dispatch (not a stub)")
	}
	if len(hollowS) != 0 {
		t.Errorf("substantive .test.ts must produce NO finding through the shared dispatch; got %+v", hollowS)
	}
	// The namespaced rule ID proves the finding came through the shared namespacing
	// (pack.NamespacedRuleID), not a parallel mock.
	if len(hollowH) > 0 && hollowH[0].Rule != tsProofHollowRuleID {
		t.Errorf("finding rule id = %q, want namespaced %q", hollowH[0].Rule, tsProofHollowRuleID)
	}
}
