package gate

// ISSUE-114 TASK-003 (CLM-001/004/005/006): a plan's machine-readable MandatedTests are
// DERIVED from its tasks' `claims[]` resolved against the SOURCE artifact's claims[].tests,
// unioned with any explicit test_names. These drive the REAL exported ResolveArtifactStatus
// over the plan_claim_derivation fixture root. Package `gate` (internal) to reuse the
// findRecord helper in artifact_status_obsoleted_test.go.

import (
	"testing"
)

// planDerivationRoot is the claim-derivation fixture project root: four source artifacts
// (three implemented specs + one obsoleted spec) and five plans, one per derivation outcome.
func planDerivationRoot() string { return "testdata/plan_claim_derivation/root" }

// mandatedNames flattens a record's MandatedTests to their function names.
func mandatedNames(mts []MandatedTest) []string {
	out := make([]string, 0, len(mts))
	for _, mt := range mts {
		out = append(out, mt.FuncName)
	}
	return out
}

// countMandated returns how many entries carry the given function name — a COUNT, not a
// contains check, because CLM-005's dedupe is only falsifiable by counting.
func countMandated(mts []MandatedTest, name string) int {
	n := 0
	for _, mt := range mts {
		if mt.FuncName == name {
			n++
		}
	}
	return n
}

// resolvePlanDerivationRoot resolves the fixture root or fails the test.
func resolvePlanDerivationRoot(t *testing.T) *ArtifactStatusResolution {
	t.Helper()
	res, err := ResolveArtifactStatus(planDerivationRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus(%s): %v", planDerivationRoot(), err)
	}
	return res
}

// TestResolveArtifactStatus_PlanDerivesMandatedTestsFromTaskClaims (CLM-001): PLAN-SPEC-850
// declares NO test_names anywhere, yet its record carries TestPlanDerive_Present — derived
// purely from its task's claims: [CLM-001] resolved against SPEC-850's claims[].tests.
// Attribution (SE2) points at the PLAN, not the source spec.
func TestResolveArtifactStatus_PlanDerivesMandatedTestsFromTaskClaims(t *testing.T) {
	res := resolvePlanDerivationRoot(t)
	rec := findRecord(t, res.Records, "PLAN-SPEC-850")

	if len(rec.MandatedTests) != 1 {
		t.Fatalf("PLAN-SPEC-850 MandatedTests = %v (len %d), want exactly [TestPlanDerive_Present]",
			mandatedNames(rec.MandatedTests), len(rec.MandatedTests))
	}
	mt := rec.MandatedTests[0]
	if mt.FuncName != "TestPlanDerive_Present" {
		t.Errorf("derived FuncName = %q, want TestPlanDerive_Present", mt.FuncName)
	}
	// SE2: the drift violation must attribute to the PLAN whose status is wrong, never to
	// the source spec (which is not drifting).
	if mt.SpecFile != rec.Path {
		t.Errorf("derived SpecFile = %q, want the PLAN's path %q", mt.SpecFile, rec.Path)
	}
	if mt.SpecID != "PLAN-SPEC-850" {
		t.Errorf("derived SpecID = %q, want PLAN-SPEC-850 (never SPEC-850)", mt.SpecID)
	}
	// SE2: ClaimID carries the SOURCE claim id, because driftBrokenPromiseMessage renders
	// it as ", claim %s" and the source claim is what explains why the test is owed.
	if mt.ClaimID != "CLM-001" {
		t.Errorf("derived ClaimID = %q, want the source claim id CLM-001", mt.ClaimID)
	}
}

// TestResolveArtifactStatus_PlanDerivationSkipsRetiredSource (CLM-004): SPEC-856 is
// `obsoleted` (retired-terminal), so its claims are RETRACTED and PLAN-SPEC-856 inherits
// none of them. The same test asserts SPEC-856's OWN record DOES carry the withdrawn test,
// proving the plan's emptiness is the guard's doing and not an empty-input artifact.
func TestResolveArtifactStatus_PlanDerivationSkipsRetiredSource(t *testing.T) {
	res := resolvePlanDerivationRoot(t)

	source := findRecord(t, res.Records, "SPEC-856")
	if countMandated(source.MandatedTests, "TestPlanDerive_RetiredOnly") != 1 {
		t.Fatalf("fixture precondition broken: SPEC-856 MandatedTests = %v, want it to CARRY "+
			"TestPlanDerive_RetiredOnly (isTerminalSpecStatus is false for obsoleted, so "+
			"ExtractMandatedTests must not drop it) — without this the guard test is vacuous",
			mandatedNames(source.MandatedTests))
	}

	plan := findRecord(t, res.Records, "PLAN-SPEC-856")
	if len(plan.MandatedTests) != 0 {
		t.Errorf("PLAN-SPEC-856 MandatedTests = %v, want ZERO — a retired source's claims are "+
			"withdrawn and the plan inherits none", mandatedNames(plan.MandatedTests))
	}
}

// TestResolveArtifactStatus_PlanDerivationReadsRecordClassNotIsTerminalSpecStatus (CLM-004):
// pin the disagreement the guard depends on. isTerminalSpecStatus does NOT name "obsoleted"
// while ClassifyArtifactStatus DOES classify it retired-terminal — so a future refactor that
// reaches for isTerminalSpecStatus here reopens the defect, and this test catches it.
func TestResolveArtifactStatus_PlanDerivationReadsRecordClassNotIsTerminalSpecStatus(t *testing.T) {
	if isTerminalSpecStatus("obsoleted") {
		t.Error("isTerminalSpecStatus(\"obsoleted\") = true; the guard's whole premise is that " +
			"these two authorities DISAGREE on obsoleted")
	}
	if got := ClassifyArtifactStatus(KindSpec, "obsoleted"); got != ClassRetiredTerminal {
		t.Errorf("ClassifyArtifactStatus(spec, obsoleted) = %v, want retired-terminal", got)
	}

	res := resolvePlanDerivationRoot(t)
	plan := findRecord(t, res.Records, "PLAN-SPEC-856")
	if len(plan.MandatedTests) != 0 {
		t.Errorf("PLAN-SPEC-856 MandatedTests = %v, want ZERO — the guard must key on the "+
			"record's Class, never on isTerminalSpecStatus", mandatedNames(plan.MandatedTests))
	}
}

// TestResolveArtifactStatus_PlanDerivationUnionsExplicitTestNamesWithoutDuplicates (CLM-005):
// PLAN-SPEC-854 declares test_names: [TestPlanDerive_Present] AND derives
// {TestPlanDerive_Present, TestPlanDerive_Absent} from CLM-001. The union is deduped by
// function name, and (SE4) the surviving Present entry is the EXPLICIT one.
func TestResolveArtifactStatus_PlanDerivationUnionsExplicitTestNamesWithoutDuplicates(t *testing.T) {
	res := resolvePlanDerivationRoot(t)
	rec := findRecord(t, res.Records, "PLAN-SPEC-854")

	if n := countMandated(rec.MandatedTests, "TestPlanDerive_Present"); n != 1 {
		t.Errorf("TestPlanDerive_Present appears %d times in %v, want EXACTLY once — a "+
			"duplicate yields two byte-identical drift violations for one real finding",
			n, mandatedNames(rec.MandatedTests))
	}
	if n := countMandated(rec.MandatedTests, "TestPlanDerive_Absent"); n != 1 {
		t.Errorf("TestPlanDerive_Absent appears %d times in %v, want exactly once (derived)",
			n, mandatedNames(rec.MandatedTests))
	}
	if len(rec.MandatedTests) != 2 {
		t.Errorf("PLAN-SPEC-854 MandatedTests = %v (len %d), want exactly 2 entries",
			mandatedNames(rec.MandatedTests), len(rec.MandatedTests))
	}

	// SE4: EXPLICIT wins the dedupe. planTaskMandatedTests puts the TASK id in ClaimID, so
	// the surviving entry's ClaimID identifies which channel produced it.
	for _, mt := range rec.MandatedTests {
		if mt.FuncName != "TestPlanDerive_Present" {
			continue
		}
		if mt.ClaimID != "TASK-001" {
			t.Errorf("surviving TestPlanDerive_Present entry has ClaimID %q, want the TASK id "+
				"TASK-001 — the EXPLICIT declaration must win the dedupe, not the derived one",
				mt.ClaimID)
		}
	}
}

// TestResolveArtifactStatus_PlanUnresolvedClaimRefContributesNothing (CLM-006): PLAN-ISSUE-853
// names a source (ISSUE-853) that does not exist, so its task claim resolves to nothing. That
// contributes ZERO mandated tests and emits no error of any kind.
//
// THIS TEST LEGITIMATELY PASSES PRE-FIX. It is the non-regression guard that derivation does
// not start inventing entries for claims it cannot resolve; making it fail first would require
// breaking the very behavior it exists to protect (see the plan's FALSIFICATION BAR).
func TestResolveArtifactStatus_PlanUnresolvedClaimRefContributesNothing(t *testing.T) {
	res, err := ResolveArtifactStatus(planDerivationRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus must not error on an unresolvable plan source: %v", err)
	}
	rec := findRecord(t, res.Records, "PLAN-ISSUE-853")
	if len(rec.MandatedTests) != 0 {
		t.Errorf("PLAN-ISSUE-853 MandatedTests = %v, want ZERO — an unresolvable claim "+
			"reference contributes nothing and is silent", mandatedNames(rec.MandatedTests))
	}
}
