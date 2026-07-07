package gate

import "testing"

// TestApplyPolicy_NewCode_GrandfathersPreexisting proves the new-code applies-to mode
// grandfathers pre-existing findings (CLM-002, CLM-003): a dimension with
// AppliesTo:new-code + level:block and a baseline containing the step's existing
// findings yields ZERO net-new, so the step PASSES — only a net-new finding would fail.
func TestApplyPolicy_NewCode_GrandfathersPreexisting(t *testing.T) {
	known := vio("R", "f.go", "known")
	baseline := &BaselineArtifact{Violations: []Violation{known}}
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{known}}

	policy := map[string]DimensionPolicy{
		"pack_engines": {Level: PolicyBlock, AppliesTo: AppliesToNewCode},
	}
	got := ApplyPolicy([]StepResult{step}, baseline, policy, nil)[0]
	if got.Status != "pass" {
		t.Errorf("applies-to:new-code must grandfather the baselined finding (net-new == 0) -> pass, got %s: %+v", got.Status, got.NewViolations)
	}

	// A net-new finding under the SAME policy still fails — grandfathering is only for
	// the pre-existing set, not a blanket excuse.
	withNew := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{known, vio("R", "f.go", "new")}}
	got2 := ApplyPolicy([]StepResult{withNew}, baseline, policy, nil)[0]
	if got2.Status != "fail" || len(got2.NewViolations) != 1 || got2.NewViolations[0].RegionHash != "new" {
		t.Errorf("applies-to:new-code must still fail on the net-new finding only, got status=%s new=%+v", got2.Status, got2.NewViolations)
	}
}

// TestApplyPolicy_AllCode_BlocksOnTotal proves the all-code applies-to mode counts
// EVERY violation regardless of the baseline (CLM-002, CLM-003), and that an ABSENT
// applies-to key behaves IDENTICALLY to explicit all-code — the strict default
// invariant. A baselined finding that new-code would grandfather still BLOCKS under
// all-code (and under the bare/absent key).
func TestApplyPolicy_AllCode_BlocksOnTotal(t *testing.T) {
	known := vio("R", "f.go", "known")
	baseline := &BaselineArtifact{Violations: []Violation{known}}
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{known}}

	// Explicit all-code: the baselined finding is NOT grandfathered — it counts and blocks.
	explicit := map[string]DimensionPolicy{
		"pack_engines": {Level: PolicyBlock, AppliesTo: AppliesToAllCode},
	}
	gotExplicit := ApplyPolicy([]StepResult{step}, baseline, explicit, nil)[0]
	if gotExplicit.Status != "fail" {
		t.Errorf("applies-to:all-code must block on the total (baselined finding not grandfathered), got %s", gotExplicit.Status)
	}

	// Absent applies-to key (empty string): MUST behave identically to explicit all-code
	// — the strict key-absent default (block on total), not silently new-code.
	absent := map[string]DimensionPolicy{
		"pack_engines": {Level: PolicyBlock},
	}
	gotAbsent := ApplyPolicy([]StepResult{step}, baseline, absent, nil)[0]
	if gotAbsent.Status != "fail" {
		t.Errorf("an absent applies-to key must default to all-code (block on total), got %s — the strict default must not silently grandfather", gotAbsent.Status)
	}
	if gotAbsent.Status != gotExplicit.Status {
		t.Errorf("absent applies-to must be byte-identical to explicit all-code: absent=%s explicit=%s", gotAbsent.Status, gotExplicit.Status)
	}
}

// TestApplyScopedPolicy_SourceOverride_AllCodeZeroTolerance proves the SPEC-047 REQ-007
// per-source path under the rename (CLM-003): a pack_engines dimension
// AppliesTo:new-code with a nested sources[backstop/self] AppliesTo:all-code blocks
// EVERY backstop/self finding (baselined or not) while another pack's pre-existing
// finding still grandfathers — byte-identical to the old baseline:false override.
func TestApplyScopedPolicy_SourceOverride_AllCodeZeroTolerance(t *testing.T) {
	self := vioSrc(selfNeutralRule, "pkg/gate/foo.go", "selfregion", "backstop/self")
	gostd := vioSrc(goStdRule, "pkg/x/x.go", "gostdregion", "backstop/go-standards")
	// BOTH findings are in the baseline (grandfathered under new-code).
	baseline := &BaselineArtifact{Violations: []Violation{self, gostd}}
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{self, gostd}}

	policy := map[string]DimensionPolicy{
		"pack_engines": {
			Level:     PolicyBlock,
			AppliesTo: AppliesToNewCode,
			Sources: map[string]DimensionPolicy{
				"backstop/self": {Level: PolicyBlock, AppliesTo: AppliesToAllCode},
			},
		},
	}
	got := ApplyPolicy([]StepResult{step}, baseline, policy, nil)[0]
	if got.Status != "fail" {
		t.Fatalf("the backstop/self all-code source override must block its baselined finding (zero tolerance), got %s", got.Status)
	}
	// backstop/self blocks despite being baselined; go-standards stays grandfathered.
	sawSelf, sawGostd := false, false
	for _, v := range got.NewViolations {
		switch v.SourcePack {
		case "backstop/self":
			sawSelf = true
		case "backstop/go-standards":
			sawGostd = true
		}
	}
	if !sawSelf {
		t.Error("the backstop/self all-code override must count its baselined finding among the blocking set (zero grandfathering)")
	}
	if sawGostd {
		t.Error("the go-standards finding stays new-code-grandfathered — it must NOT be in the blocking set")
	}
}

// TestApplyScopedPolicy_NewCode_NilBaseline_FailsLoud proves the anti-vacuous-green
// invariant through the rename (CLM-003): an AppliesTo:new-code source with a NIL
// baseline cannot grandfather, so every finding counts and BLOCKS — a missing baseline
// must never silently flip a whole dimension to green.
func TestApplyScopedPolicy_NewCode_NilBaseline_FailsLoud(t *testing.T) {
	gostd := vioSrc(goStdRule, "pkg/x/x.go", "gostdregion", "backstop/go-standards")
	self := vioSrc(selfNeutralRule, "pkg/gate/foo.go", "selfregion", "backstop/self")
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{gostd, self}}

	// A policy whose DEFAULT source is applies-to:new-code (would grandfather WITH a
	// baseline), plus a scoped all-code self source — driven with a NIL baseline.
	policy := map[string]DimensionPolicy{
		"pack_engines": {
			Level:     PolicyBlock,
			AppliesTo: AppliesToNewCode,
			Sources: map[string]DimensionPolicy{
				"backstop/self": {Level: PolicyBlock, AppliesTo: AppliesToAllCode},
			},
		},
	}
	got := ApplyPolicy([]StepResult{step}, nil, policy, nil)[0]
	if got.Status != "fail" {
		t.Fatalf("with a nil baseline, a new-code dimension must FAIL-LOUD (block all findings), not silently grandfather to green; got %s", got.Status)
	}
	sawGostd, sawSelf := false, false
	for _, v := range got.NewViolations {
		switch v.SourcePack {
		case "backstop/go-standards":
			sawGostd = true
		case "backstop/self":
			sawSelf = true
		}
	}
	if !sawGostd {
		t.Error("a new-code source's finding must BLOCK when no baseline is present (cannot grandfather without a baseline)")
	}
	if !sawSelf {
		t.Error("the all-code scoped source's finding must block")
	}
}

// TestApplyPolicy_LevelOrthogonalToAppliesTo proves level and applies-to are
// INDEPENDENT knobs (CLM-004): a dimension AppliesTo:new-code + level:warn surfaces
// net-new findings as status "warning" (never fail), so applies-to decides WHICH
// violations count while level decides WHAT happens once they do — the two are never
// merged.
func TestApplyPolicy_LevelOrthogonalToAppliesTo(t *testing.T) {
	known := vio("R", "f.go", "known")
	netnew := vio("R", "f.go", "new")
	baseline := &BaselineArtifact{Violations: []Violation{known}}
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{known, netnew}}

	policy := map[string]DimensionPolicy{
		"pack_engines": {Level: PolicyWarn, AppliesTo: AppliesToNewCode},
	}
	got := ApplyPolicy([]StepResult{step}, baseline, policy, nil)[0]
	if got.Status != "warning" {
		t.Errorf("applies-to:new-code + level:warn must surface the net-new finding as warning (never fail), got %s", got.Status)
	}

	// Contrast: the SAME applies-to:new-code with level:block DOES fail on the net-new
	// finding — proving level is the independent WHAT-happens knob.
	blockPolicy := map[string]DimensionPolicy{
		"pack_engines": {Level: PolicyBlock, AppliesTo: AppliesToNewCode},
	}
	gotBlock := ApplyPolicy([]StepResult{step}, baseline, blockPolicy, nil)[0]
	if gotBlock.Status != "fail" {
		t.Errorf("applies-to:new-code + level:block must fail on the net-new finding, got %s", gotBlock.Status)
	}
}
