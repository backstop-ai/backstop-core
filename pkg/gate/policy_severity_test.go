package gate

import "testing"

// A WARNING-SEVERITY VIOLATION MUST NEVER BLOCK. That is founder law recorded as
// "loud != blocking": block defects and broken promises, warn-with-guidance for
// capability signals, and the enemy is vacuous green rather than passing.
//
// The defect these lock was found in CI run 30395875188, where coverage_threshold
// reported fail with ONE violation and that violation was the severity=warning
// coverage exclusion NOTICE. step_coverage.go had correctly returned pass; the policy
// layer overwrote it, because both of its verdict paths decide by COUNTING violations
// and `grep -c Severity pkg/gate/policy.go` returned 0. A suppression notice was
// failing the gate it was built to keep informative.

func warningViolation(rule string) Violation {
	return Violation{Rule: rule, Message: "a non-blocking notice", Severity: "warning", File: "a.go"}
}

func errorViolation(rule string) Violation {
	return Violation{Rule: rule, Message: "a real defect", Severity: "error", File: "b.go"}
}

// TestApplyPolicy_WarningOnlyStepSurvivesAsPass is the regression lock.
//
// "SurvivesAsPass" in the name means SURVIVES AS NON-BLOCKING. The status is "warning"
// — surfaced, exit 0, gate.Pass true — which is strictly more informative than "pass"
// and is the vocabulary policy.go already uses for that state.
func TestApplyPolicy_WarningOnlyStepSurvivesAsPass(t *testing.T) {
	step := StepResult{
		StepName:   StepCoverageThreshold,
		Status:     "pass",
		Violations: []Violation{warningViolation("coverage_exclusion")},
	}
	policy := map[string]DimensionPolicy{StepCoverageThreshold: {Level: PolicyBlock}}

	got := ApplyPolicy([]StepResult{step}, nil, policy, &GateScope{Mode: GateScopeModeAll})

	// "warning" rather than "pass": both are NON-BLOCKING (gate.Pass stays true and the
	// exit code stays 0), and "warning" is the vocabulary this file already uses for a
	// surfaced-but-not-failing dimension. Reporting "pass" would hide that something was
	// surfaced at all, which is the opposite of loud-not-blocking.
	if got[0].Status == "fail" {
		t.Errorf("a step whose ONLY violation is severity=warning FAILED; a notice that blocks is a " +
			"contradiction in terms, and this is the exact shape that failed CI run 30395875188")
	}
	if got[0].Status != "warning" {
		t.Errorf("expected the non-blocking-but-surfaced status %q, got %q", "warning", got[0].Status)
	}
	if len(got[0].Violations) != 1 {
		t.Errorf("the warning was dropped from the violations list (%d left); non-blocking must not mean "+
			"invisible — the suppression still has to be reported", len(got[0].Violations))
	}
}

// TestApplyPolicy_ErrorOnlyStepStillFails guards the other direction, so the fix
// cannot silently disarm blocking.
func TestApplyPolicy_ErrorOnlyStepStillFails(t *testing.T) {
	step := StepResult{
		StepName:   StepCoverageThreshold,
		Status:     "fail",
		Violations: []Violation{errorViolation("coverage_threshold")},
	}
	policy := map[string]DimensionPolicy{StepCoverageThreshold: {Level: PolicyBlock}}

	got := ApplyPolicy([]StepResult{step}, nil, policy, &GateScope{Mode: GateScopeModeAll})

	if got[0].Status != "fail" {
		t.Fatalf("an error-severity violation must still FAIL a blocking dimension; got %q. If this passes, "+
			"the severity fix disarmed the gate", got[0].Status)
	}
}

// TestApplyPolicy_MixedSeveritiesFailAndKeepTheWarning is the case that proves the
// two behaviours coexist: the error blocks, and the warning is still reported.
//
// An implementation that filtered warnings out of the LIST rather than out of the
// BLOCKING COUNT would satisfy both tests above and quietly delete the suppression
// notice from the report.
func TestApplyPolicy_MixedSeveritiesFailAndKeepTheWarning(t *testing.T) {
	step := StepResult{
		StepName:   StepCoverageThreshold,
		Status:     "fail",
		Violations: []Violation{warningViolation("coverage_exclusion"), errorViolation("coverage_threshold")},
	}
	policy := map[string]DimensionPolicy{StepCoverageThreshold: {Level: PolicyBlock}}

	got := ApplyPolicy([]StepResult{step}, nil, policy, &GateScope{Mode: GateScopeModeAll})

	if got[0].Status != "fail" {
		t.Errorf("a step carrying an error must fail even when a warning sits beside it; got %q", got[0].Status)
	}
	if len(got[0].Violations) != 2 {
		t.Errorf("the report lost a violation (%d of 2 left); the fix must change what BLOCKS, never what "+
			"is REPORTED", len(got[0].Violations))
	}
}

// TestApplyScopedPolicy_WarningOnlyStepSurvivesAsPass covers the SECOND verdict path.
//
// applyScopedPolicy is taken whenever a dimension carries per-pack Sources overrides —
// backstop.yml has one today (pack_engines) — so a fix applied to only the
// dimension-default path would leave this one still failing on notices.
func TestApplyScopedPolicy_WarningOnlyStepSurvivesAsPass(t *testing.T) {
	step := StepResult{
		StepName:   StepPackEngines,
		Status:     "pass",
		Violations: []Violation{{Rule: "notice", Severity: "warning", File: "a.go", SourcePack: "acme/pack"}},
	}
	policy := map[string]DimensionPolicy{
		StepPackEngines: {Level: PolicyBlock, Sources: map[string]DimensionPolicy{"acme/pack": {Level: PolicyBlock}}},
	}

	got := ApplyPolicy([]StepResult{step}, nil, policy, &GateScope{Mode: GateScopeModeAll})

	if got[0].Status == "fail" {
		t.Errorf("the SCOPED policy path FAILED a warning-only step. Both paths counted violations; " +
			"fixing only the dimension-default one would leave this hole open")
	}
	if got[0].Status != "warning" {
		t.Errorf("expected %q from the scoped path, got %q", "warning", got[0].Status)
	}
}
