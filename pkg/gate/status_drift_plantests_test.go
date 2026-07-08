package gate

// ISSUE-048 facet-3 tests (CLM-009): the resolver parses completed-plan
// phases[].tasks[].test_names into the plan record's MandatedTests, so a completed plan is
// subject to the SAME success-terminal absent-test BLOCK already applied to issues/specs.
// Package `gate` (internal) to reuse presentSet + the findRecord/errorViolationsForFile
// helpers.

import (
	"strings"
	"testing"
)

// TestResolveArtifactStatus_ParsesPlanTaskTestNames (CLM-009): a completed plan's task
// test_names are parsed into its record's MandatedTests.
func TestResolveArtifactStatus_ParsesPlanTaskTestNames(t *testing.T) {
	res, err := ResolveArtifactStatus(vocabDriftRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	plan := findRecord(t, res.Records, "PLAN-SPEC-840")
	if plan.Class != ClassSuccessTerminal {
		t.Fatalf("PLAN-SPEC-840 (completed) class = %v, want success-terminal", plan.Class)
	}
	names := map[string]bool{}
	for _, mt := range plan.MandatedTests {
		names[mt.FuncName] = true
	}
	for _, want := range []string{"TestDriftVocab_Present", "TestDriftVocab_Absent"} {
		if !names[want] {
			t.Errorf("PLAN-SPEC-840 MandatedTests missing %q; got %v", want, names)
		}
	}
}

// TestStatusDrift_CompletedPlanAbsentTaskTest_Blocks (CLM-009): with a present-set that
// lacks TestDriftVocab_Absent, the completed PLAN-SPEC-840 blocks on the absent task test;
// PLAN-SPEC-841 (all task test_names present) emits NO violation.
func TestStatusDrift_CompletedPlanAbsentTaskTest_Blocks(t *testing.T) {
	res, err := ResolveArtifactStatus(vocabDriftRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	present := presentSet("TestDriftVocab_Present")
	drift := ClassifyStatusDrift(res.Records, present)

	// PLAN-SPEC-840: an error-severity broken-promise for the ABSENT test.
	blocking := findRecord(t, res.Records, "PLAN-SPEC-840")
	got := errorViolationsForFile(drift.Violations, blocking.Path)
	if len(got) == 0 {
		t.Fatalf("PLAN-SPEC-840 must block on the absent TestDriftVocab_Absent, got no error violation")
	}
	foundAbsent := false
	for _, v := range got {
		if v.Rule == StepArtifactStatusDrift && strings.Contains(v.Message, "TestDriftVocab_Absent") {
			foundAbsent = true
		}
	}
	if !foundAbsent {
		t.Errorf("PLAN-SPEC-840 block must name the absent TestDriftVocab_Absent; got %+v", got)
	}

	// PLAN-SPEC-841: all task test_names present -> no violation.
	allPresent := findRecord(t, res.Records, "PLAN-SPEC-841")
	if got := errorViolationsForFile(drift.Violations, allPresent.Path); len(got) != 0 {
		t.Errorf("PLAN-SPEC-841 (all test_names present) must emit no violation, got %+v", got)
	}
}
