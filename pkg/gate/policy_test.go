package gate

import "testing"

func vio(rule, file, region string) Violation {
	return Violation{Rule: rule, File: file, Message: "m", Severity: "error", RegionHash: region}
}

// TestApplyPolicy_LevelsAndBaseline exercises the full level x baseline matrix plus the
// pass-throughs (no policy entry, config error, neutralized aggregate baseline step).
func TestApplyPolicy_LevelsAndBaseline(t *testing.T) {
	// Baseline grandfathers exactly one finding (region "known").
	baseline := &BaselineArtifact{Violations: []Violation{vio("R", "f.go", "known")}}

	steps := []StepResult{
		// block + baseline: one known (grandfathered) + one new -> must FAIL on the new one only.
		{StepName: StepContractSignature, Status: "fail", Violations: []Violation{vio("R", "f.go", "known"), vio("R", "f.go", "new")}},
		// block + baseline, all known -> PASS (net-new == 0).
		{StepName: "pack_engines", Status: "fail", Violations: []Violation{vio("R", "f.go", "known")}},
		// warn: has findings but must NOT fail (status warning).
		{StepName: StepCoverageThreshold, Status: "fail", Violations: []Violation{vio("R", "g.go", "x")}},
		// off: disabled -> skipped, violations cleared.
		{StepName: StepTestVerification, Status: "fail", Violations: []Violation{vio("R", "h.go", "y")}},
		// no policy entry -> unchanged.
		{StepName: StepArtifactValidation, Status: "fail", Violations: []Violation{vio("R", "i.go", "z")}},
		// config error -> preserved untouched even though a policy entry exists.
		{StepName: StepTestSubstantiveness, Status: "fail", ConfigErr: true, Violations: []Violation{vio("R", "j.go", "q")}},
		// aggregate baseline step -> neutralized when any policy is configured.
		{StepName: StepBaselineComparison, Status: "fail", Violations: []Violation{vio("R", "k.go", "w")}},
	}

	policy := map[string]DimensionPolicy{
		StepContractSignature:   {Level: PolicyBlock, Baseline: true},
		"pack_engines":          {Level: PolicyBlock, Baseline: true},
		StepCoverageThreshold:   {Level: PolicyWarn},
		StepTestVerification:    {Level: PolicyOff},
		StepTestSubstantiveness: {Level: PolicyBlock},
	}

	got := map[string]StepResult{}
	for _, s := range ApplyPolicy(steps, baseline, policy, nil) {
		got[s.StepName] = s
	}

	if s := got[StepContractSignature]; s.Status != "fail" || len(s.NewViolations) != 1 || s.NewViolations[0].RegionHash != "new" {
		t.Errorf("block+baseline w/ net-new: want fail on the 'new' finding, got status=%s new=%+v", s.Status, s.NewViolations)
	}
	if s := got["pack_engines"]; s.Status != "pass" {
		t.Errorf("block+baseline all-grandfathered: want pass, got %s", s.Status)
	}
	if s := got[StepCoverageThreshold]; s.Status != "warning" {
		t.Errorf("warn level: want warning (never fail), got %s", s.Status)
	}
	if s := got[StepTestVerification]; s.Status != "skipped" || len(s.Violations) != 0 {
		t.Errorf("off level: want skipped w/ cleared violations, got status=%s n=%d", s.Status, len(s.Violations))
	}
	if s := got[StepArtifactValidation]; s.Status != "fail" {
		t.Errorf("no policy entry: want unchanged (fail), got %s", s.Status)
	}
	if s := got[StepTestSubstantiveness]; s.Status != "fail" || !s.ConfigErr {
		t.Errorf("config error must be preserved, got status=%s configErr=%v", s.Status, s.ConfigErr)
	}
	if s := got[StepBaselineComparison]; s.Status != "skipped" {
		t.Errorf("aggregate baseline step must be neutralized under policy, got %s", s.Status)
	}

	// Empty policy is a no-op (backward compatible): every step is returned unchanged.
	noop := ApplyPolicy(steps, baseline, nil, nil)
	if len(noop) != len(steps) || noop[0].Status != "fail" {
		t.Errorf("empty policy must be a no-op, got %+v", noop[0])
	}
}
