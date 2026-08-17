package gate

// ISSUE-114 TASK-004 (CLM-002/003/004/007/008): the DIR-032 verification bar for plans —
// a NON-TERMINAL plan carrying NO test_names still reaches the delivered-but-open advisory,
// via claim-derived mandated tests. Every test drives the REAL ResolveArtifactStatus +
// ClassifyStatusDrift + SplitDriftResult over the plan_claim_derivation fixture root.

import (
	"strings"
	"testing"
)

// planDerivationPresent is the calibrated present-set every test below is judged against.
// TestPlanDerive_Absent is deliberately NOT in it; TestPlanDerive_RetiredOnly deliberately
// IS, so a missing retired guard fires a spurious advisory rather than passing silently.
func planDerivationPresent() map[string]bool {
	return presentSet("TestPlanDerive_Present", "TestPlanDerive_RetiredOnly")
}

// classifyPlanDerivationDrift resolves the fixture root and classifies drift over it.
func classifyPlanDerivationDrift(t *testing.T) (*ArtifactStatusResolution, StepResult) {
	t.Helper()
	res, err := ResolveArtifactStatus(planDerivationRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus(%s): %v", planDerivationRoot(), err)
	}
	return res, ClassifyStatusDrift(res.Records, planDerivationPresent())
}

// violationsForFile returns every violation, of any severity, attributed to a file path.
func violationsForFile(vs []Violation, path string) []Violation {
	var out []Violation
	for _, v := range vs {
		if v.File == path {
			out = append(out, v)
		}
	}
	return out
}

// TestStatusDrift_NonTerminalPlanAllDerivedTestsPresent_FiresAdvisory (CLM-002): THE DIR-032
// BAR. PLAN-SPEC-850 is `draft` and declares zero test_names, yet its derived mandated test
// is present — so the delivered-but-open advisory fires, warning-severity, attributed to the
// plan.
func TestStatusDrift_NonTerminalPlanAllDerivedTestsPresent_FiresAdvisory(t *testing.T) {
	res, combined := classifyPlanDerivationDrift(t)
	plan := findRecord(t, res.Records, "PLAN-SPEC-850")

	hits := violationsForFile(combined.Violations, plan.Path)
	if len(hits) != 1 {
		t.Fatalf("PLAN-SPEC-850 produced %d violations, want exactly 1 advisory (this is the "+
			"DIR-032 bar: a non-terminal plan with NO test_names must still be visible)", len(hits))
	}
	v := hits[0]
	if v.Rule != StepArtifactStatusDriftAdvisory {
		t.Errorf("advisory Rule = %q, want %q", v.Rule, StepArtifactStatusDriftAdvisory)
	}
	if v.Severity != "warning" {
		t.Errorf("advisory Severity = %q, want warning", v.Severity)
	}
	if !strings.Contains(v.Message, "PLAN-SPEC-850") {
		t.Errorf("advisory message must NAME the plan to be actionable, got: %s", v.Message)
	}
	if !strings.Contains(v.Message, "draft") {
		t.Errorf("advisory message must name the plan's status, got: %s", v.Message)
	}
}

// TestStatusDrift_NonTerminalPlanDerivedTestAbsent_NoAdvisory (CLM-003): the negative control
// that keeps CLM-002 honest. PLAN-SPEC-852 derives Present AND Absent; Absent is missing, so
// the plan does not look delivered and produces NO violation. Without this, CLM-002 would be
// satisfiable by code that warns on every non-terminal plan unconditionally.
func TestStatusDrift_NonTerminalPlanDerivedTestAbsent_NoAdvisory(t *testing.T) {
	res, combined := classifyPlanDerivationDrift(t)
	plan := findRecord(t, res.Records, "PLAN-SPEC-852")

	if hits := violationsForFile(combined.Violations, plan.Path); len(hits) != 0 {
		t.Errorf("PLAN-SPEC-852 produced %d violations, want ZERO — one of its derived mandated "+
			"tests is ABSENT, so it does not look delivered: %v", len(hits), hits)
	}
}

// TestStatusDrift_CompletedPlanDerivedAbsentTest_Blocks (CLM-007): the success-terminal
// direction gains the same reach. PLAN-SPEC-854 is `completed` and reaches
// TestPlanDerive_Absent ONLY by derivation (its test_names lists Present alone), so the
// absent test blocks as a broken promise — derived tests are not a warn-only second class.
func TestStatusDrift_CompletedPlanDerivedAbsentTest_Blocks(t *testing.T) {
	res, combined := classifyPlanDerivationDrift(t)
	plan := findRecord(t, res.Records, "PLAN-SPEC-854")

	blocks := errorViolationsForFile(combined.Violations, plan.Path)
	if len(blocks) != 1 {
		t.Fatalf("PLAN-SPEC-854 produced %d error-severity violations, want exactly 1 naming "+
			"the DERIVED absent test: %v", len(blocks), blocks)
	}
	v := blocks[0]
	if v.Rule != StepArtifactStatusDrift {
		t.Errorf("block Rule = %q, want %q", v.Rule, StepArtifactStatusDrift)
	}
	if !strings.Contains(v.Message, "TestPlanDerive_Absent") {
		t.Errorf("block message must name the derived absent test, got: %s", v.Message)
	}
	if strings.Contains(v.Message, "TestPlanDerive_Present") {
		t.Errorf("TestPlanDerive_Present is PRESENT and must not be reported absent, got: %s", v.Message)
	}
}

// TestStatusDrift_RetiredSourcePlanProducesNoAdvisory (CLM-004): the spurious-advisory guard.
// PLAN-SPEC-856 is `draft` and its source SPEC-856 is obsoleted, so it derives nothing and
// produces NO violation — even though TestPlanDerive_RetiredOnly IS in the present-set.
// Without the retired skip in the resolver this plan derives one test, finds it present,
// and warns.
func TestStatusDrift_RetiredSourcePlanProducesNoAdvisory(t *testing.T) {
	res, combined := classifyPlanDerivationDrift(t)

	if !planDerivationPresent()["TestPlanDerive_RetiredOnly"] {
		t.Fatal("fixture precondition broken: TestPlanDerive_RetiredOnly must be PRESENT, " +
			"otherwise this guard passes for the wrong reason")
	}
	plan := findRecord(t, res.Records, "PLAN-SPEC-856")
	if hits := violationsForFile(combined.Violations, plan.Path); len(hits) != 0 {
		t.Errorf("PLAN-SPEC-856 produced %d violations, want ZERO — its source is retired, so "+
			"its claims are withdrawn and it can never look delivered: %v", len(hits), hits)
	}
}

// TestStatusDrift_DerivedPlanAdvisoryIsWarningOnlyAndNeverBlocks (CLM-008): the STRUCTURAL
// property, asserted over the partitioned surfaces rather than a sample — every advisory
// violation is warning-severity under StepArtifactStatusDriftAdvisory, the block surface
// carries no non-terminal plan, and the advisory StepResult never fails.
func TestStatusDrift_DerivedPlanAdvisoryIsWarningOnlyAndNeverBlocks(t *testing.T) {
	res, combined := classifyPlanDerivationDrift(t)
	block, advisory := SplitDriftResult(combined)

	for _, v := range advisory.Violations {
		if v.Severity != "warning" {
			t.Errorf("advisory surface carries a %q-severity violation on %s — a mis-severity "+
				"here is what SplitDriftResult would route to the POLICIED side", v.Severity, v.File)
		}
		if v.Rule != StepArtifactStatusDriftAdvisory {
			t.Errorf("advisory surface carries Rule %q on %s, want %q",
				v.Rule, v.File, StepArtifactStatusDriftAdvisory)
		}
	}

	// The block surface must contain NO violation attributed to a non-terminal plan.
	for _, rec := range res.Records {
		if rec.Kind != KindPlan || rec.Class != ClassNonTerminal {
			continue
		}
		if hits := violationsForFile(block.Violations, rec.Path); len(hits) != 0 {
			t.Errorf("block surface carries %d violations for non-terminal plan %s — the "+
				"delivered-heuristic-is-not-proof asymmetry must be preserved exactly: %v",
				len(hits), rec.ID, hits)
		}
	}

	if advisory.Status == "fail" {
		t.Errorf("advisory StepResult Status = %q, must never be fail", advisory.Status)
	}
}
