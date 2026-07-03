package gate

import "testing"

// vioSrc builds a Violation attributed to a pack/rule-source (the SPEC-047 REQ-007
// scoping key). RegionHash is set explicitly so baseline identity matching is
// deterministic (Rule|File|RegionHash), independent of SourcePack.
func vioSrc(rule, file, region, sourcePack string) Violation {
	return Violation{Rule: rule, File: file, Message: "m", Severity: "error", RegionHash: region, SourcePack: sourcePack}
}

// selfScopedPolicy is the shared-dimension policy the REQ-006 flip is expressed
// through: pack_engines defaults to block + baseline true (grandfathering
// go-standards/go-toolchain style debt), with a per-SOURCE override flipping
// backstop/self to block + ZERO baseline.
func selfScopedPolicy() map[string]DimensionPolicy {
	return map[string]DimensionPolicy{
		"pack_engines": {
			Level:    PolicyBlock,
			Baseline: true,
			Sources: map[string]DimensionPolicy{
				"backstop/self": {Level: PolicyBlock, Baseline: false},
			},
		},
	}
}

const (
	selfNeutralRule = "backstop/self/backstop.packs.backstop.self.rules.no-language-literal-on-neutral-spine"
	goStdRule       = "backstop/go-standards/backstop.packs.backstop.go-standards.rules.core.go.core.error-wrapping-required"
	goToolchainRule = "backstop/go-toolchain/errcheck"
)

// TestPolicy_SelfPackFlipBlocksFreshNeutralSpineFinding proves that with the
// per-pack key flipping backstop/self to block + ZERO baseline, a FRESH
// neutral-spine (baked-language) finding sourced from backstop/self on the SHARED
// pack_engines dimension BLOCKS the gate (CLM-037).
func TestPolicy_SelfPackFlipBlocksFreshNeutralSpineFinding(t *testing.T) {
	self := vioSrc(selfNeutralRule, "pkg/gate/foo.go", "freshneutralspine", "backstop/self")
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{self}}

	// Fresh: not in the baseline.
	got := ApplyPolicy([]StepResult{step}, &BaselineArtifact{Violations: []Violation{}}, selfScopedPolicy(), nil)[0]
	if got.Status != "fail" {
		t.Errorf("a fresh backstop/self neutral-spine finding must BLOCK under the scoped flip, got %s", got.Status)
	}

	// The ZERO-baseline aspect: even a BASELINED backstop/self neutral-spine finding
	// blocks, because the scoped baseline:false disables grandfathering for it (the
	// wall). This is what makes it a real zero-baseline flip, not just net-new.
	baselined := &BaselineArtifact{Violations: []Violation{self}}
	got2 := ApplyPolicy([]StepResult{step}, baselined, selfScopedPolicy(), nil)[0]
	if got2.Status != "fail" {
		t.Errorf("the backstop/self scoped baseline:false must block EVEN a baselined neutral-spine finding (zero grandfathering), got %s", got2.Status)
	}
}

// TestPolicy_SelfPackFlipLeavesGoStandardsBaselinedFindingGrandfathered proves that
// with backstop/self flipped to block + zero baseline on the SAME shared
// pack_engines dimension, a baselined go-standards style finding does NOT block —
// it stays grandfathered, unaffected by the backstop/self-scoped flip (CLM-038).
func TestPolicy_SelfPackFlipLeavesGoStandardsBaselinedFindingGrandfathered(t *testing.T) {
	gostd := vioSrc(goStdRule, "pkg/x/x.go", "gostdregion", "backstop/go-standards")
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{gostd}}
	baseline := &BaselineArtifact{Violations: []Violation{gostd}} // grandfathered

	got := ApplyPolicy([]StepResult{step}, baseline, selfScopedPolicy(), nil)[0]
	if got.Status != "pass" {
		t.Errorf("a baselined go-standards finding must stay grandfathered (pass) under the backstop/self-scoped flip, got %s: %+v", got.Status, got.NewViolations)
	}
}

// TestPolicy_SelfPackFlipLeavesGoToolchainBaselinedFindingGrandfathered proves that
// with backstop/self flipped to block + zero baseline on the SAME shared
// pack_engines dimension, a baselined go-toolchain style finding does NOT block — it
// stays grandfathered (CLM-039).
func TestPolicy_SelfPackFlipLeavesGoToolchainBaselinedFindingGrandfathered(t *testing.T) {
	gotc := vioSrc(goToolchainRule, "pkg/y/y.go", "gotcregion", "backstop/go-toolchain")
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{gotc}}
	baseline := &BaselineArtifact{Violations: []Violation{gotc}} // grandfathered

	got := ApplyPolicy([]StepResult{step}, baseline, selfScopedPolicy(), nil)[0]
	if got.Status != "pass" {
		t.Errorf("a baselined go-toolchain finding must stay grandfathered (pass) under the backstop/self-scoped flip, got %s: %+v", got.Status, got.NewViolations)
	}
}

// TestPolicy_ScopedNilBaselineBlocksFailLoudNotSilentGreen proves the
// anti-vacuous-green invariant for the per-source path: with NO baseline (nil — a
// fresh checkout before the CI-pulled baseline is present), a baseline:TRUE scoped
// source cannot grandfather, so its findings BLOCK (fail-loud), exactly mirroring the
// unscoped path. A nil baseline must NEVER silently flip a whole dimension to green —
// which would be the vacuous-green the bundle fights, in the very code that flips
// backstop/self to block.
func TestPolicy_ScopedNilBaselineBlocksFailLoudNotSilentGreen(t *testing.T) {
	// A go-standards finding on a baseline:TRUE source (the dimension default), plus a
	// backstop/self finding on the zero-baseline scoped source — both under a policy
	// that HAS per-source scoping (so the scoped path runs).
	gostd := vioSrc(goStdRule, "pkg/x/x.go", "gostdregion", "backstop/go-standards")
	self := vioSrc(selfNeutralRule, "pkg/gate/foo.go", "freshneutralspine", "backstop/self")
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{gostd, self}}

	// NIL baseline: grandfathering is impossible. Both the baseline:true go-standards
	// finding AND the zero-baseline self finding must COUNT and BLOCK — never silently
	// grandfathered to green.
	got := ApplyPolicy([]StepResult{step}, nil, selfScopedPolicy(), nil)[0]
	if got.Status != "fail" {
		t.Fatalf("with a nil baseline, a scoped block dimension must FAIL-LOUD (block all findings), not silently grandfather to green; got %s", got.Status)
	}
	// Both findings must be in the blocking set — neither is grandfathered without a
	// baseline (the degraded case blocks; it does not pass).
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
		t.Error("a baseline:true source's finding must BLOCK when no baseline is present (cannot grandfather without a baseline) — mirroring the unscoped fail-loud path")
	}
	if !sawSelf {
		t.Error("the zero-baseline scoped source's finding must block")
	}
}

// TestPolicy_FlipUsesPerPackKeyNotWholeDimensionZeroBaseline is the DENYLIST: the
// flip is delivered via the per-pack/source key (filtering on
// gate.Violation.SourcePack), NOT a whole-dimension pack_engines:{block,
// baseline:false} entry (which would wrongly block go-standards/go-toolchain
// baselined debt). Proven ABSENT by contrast (CLM-040).
func TestPolicy_FlipUsesPerPackKeyNotWholeDimensionZeroBaseline(t *testing.T) {
	self := vioSrc(selfNeutralRule, "pkg/gate/foo.go", "freshneutralspine", "backstop/self")
	gostd := vioSrc(goStdRule, "pkg/x/x.go", "gostdregion", "backstop/go-standards")
	gotc := vioSrc(goToolchainRule, "pkg/y/y.go", "gotcregion", "backstop/go-toolchain")
	// go-standards + go-toolchain are baselined; self is fresh.
	baseline := &BaselineArtifact{Violations: []Violation{gostd, gotc}}
	step := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{self, gostd, gotc}}

	// PER-PACK KEY (correct): the step fails on the self finding ALONE; go-standards
	// and go-toolchain stay grandfathered (they must NOT appear among the blocking
	// net-new violations).
	got := ApplyPolicy([]StepResult{step}, baseline, selfScopedPolicy(), nil)[0]
	if got.Status != "fail" {
		t.Fatalf("the scoped flip must fail on the fresh backstop/self finding, got %s", got.Status)
	}
	for _, v := range got.NewViolations {
		if v.SourcePack == "backstop/go-standards" || v.SourcePack == "backstop/go-toolchain" {
			t.Errorf("the per-pack key must leave %s grandfathered — it must NOT be among the blocking net-new violations, got %+v", v.SourcePack, v)
		}
	}
	sawSelf := false
	for _, v := range got.NewViolations {
		if v.SourcePack == "backstop/self" {
			sawSelf = true
		}
	}
	if !sawSelf {
		t.Error("the blocking net-new set must include the backstop/self neutral-spine finding")
	}

	// WHOLE-DIMENSION zero-baseline (the PROHIBITED alternative): a
	// pack_engines:{block, baseline:false} entry with NO per-source scoping WOULD
	// wrongly block the baselined go-standards debt — proving why the per-pack key is
	// required and why the whole-dimension form must be ABSENT.
	wholeDim := map[string]DimensionPolicy{"pack_engines": {Level: PolicyBlock, Baseline: false}}
	gostdOnly := StepResult{StepName: "pack_engines", Status: "fail", Violations: []Violation{gostd}}
	wrong := ApplyPolicy([]StepResult{gostdOnly}, &BaselineArtifact{Violations: []Violation{gostd}}, wholeDim, nil)[0]
	if wrong.Status != "fail" {
		t.Error("a whole-dimension pack_engines:{block, baseline:false} MUST (wrongly) block baselined go-standards debt — that is exactly why it is prohibited and the per-pack key is used instead")
	}
	// And the scoped policy must NOT do that to go-standards (contrast, already
	// asserted above via the combined-step grandfathering).
	scopedGostd := ApplyPolicy([]StepResult{gostdOnly}, &BaselineArtifact{Violations: []Violation{gostd}}, selfScopedPolicy(), nil)[0]
	if scopedGostd.Status != "pass" {
		t.Errorf("the per-pack-scoped flip must leave go-standards baselined debt grandfathered (pass), got %s", scopedGostd.Status)
	}
}
