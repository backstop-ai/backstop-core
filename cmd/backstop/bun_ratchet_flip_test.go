package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
)

const neutralSpineRuleFragment = "no-language-literal-on-neutral-spine"

// committedBaselineNeutralSpineFiles returns the set of file paths that carry a
// backstop/self neutral-spine finding in the COMMITTED .backstop/baseline.json.
func committedBaselineNeutralSpineFiles(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".backstop", "baseline.json"))
	if err != nil {
		t.Fatalf("read committed baseline: %v", err)
	}
	var artifact gate.BaselineArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatalf("parse baseline: %v", err)
	}
	var files []string
	for _, v := range artifact.Violations {
		if strings.Contains(v.Rule, neutralSpineRuleFragment) {
			files = append(files, v.File+"|"+v.Identity)
		}
	}
	return files
}

// baselineHasNeutralSpineForSite reports whether the committed baseline carries a
// neutral-spine finding whose file/identity references siteSubstr.
func baselineHasNeutralSpineForSite(t *testing.T, siteSubstr string) bool {
	t.Helper()
	for _, entry := range committedBaselineNeutralSpineFiles(t) {
		if strings.Contains(entry, siteSubstr) {
			return true
		}
	}
	return false
}

// dogfoodPolicy loads the committed dogfood backstop.yml and maps its enforcement
// policy into the gate's per-dimension (per-source) policy — exactly the mapping the
// live gate applies.
func dogfoodPolicy(t *testing.T) map[string]gate.DimensionPolicy {
	t.Helper()
	cfg, err := config.LoadConfigFromPath(filepath.Join(repoRoot(t), "backstop.yml"))
	if err != nil {
		t.Fatalf("load dogfood backstop.yml: %v", err)
	}
	return gatePolicyFromConfig(cfg)
}

// TestRatchet_CoverageMeasurablePathSiteUnGrandfatheredAfterDeGo proves SITE 1: after
// the coverage measurable-path is de-Go'd (SPEC-043), the committed baseline carries
// ZERO backstop/self neutral-spine findings for pkg/gate/step_coverage.go's
// measurable-path site (CLM-030).
func TestRatchet_CoverageMeasurablePathSiteUnGrandfatheredAfterDeGo(t *testing.T) {
	if baselineHasNeutralSpineForSite(t, "step_coverage.go") {
		t.Errorf("the committed baseline must carry ZERO neutral-spine findings for the de-Go'd pkg/gate/step_coverage.go measurable-path site (ratchet un-grandfathering)")
	}
}

// TestRatchet_TestVerifyDiscoverySiteUnGrandfatheredAfterDeGo proves SITE 2: after
// test-verify discovery is de-Go'd (SPEC-045), the committed baseline carries ZERO
// neutral-spine findings for pkg/gate/step_testverify.go's discovery site (CLM-031).
func TestRatchet_TestVerifyDiscoverySiteUnGrandfatheredAfterDeGo(t *testing.T) {
	if baselineHasNeutralSpineForSite(t, "step_testverify.go") {
		t.Errorf("the committed baseline must carry ZERO neutral-spine findings for the de-Go'd pkg/gate/step_testverify.go discovery site")
	}
}

// TestRatchet_GoPackageMatchersSiteUnGrandfatheredAfterDeGo proves SITE 3: after the
// go-package/`./...` matchers are de-Go'd (SPEC-043), the committed baseline carries
// ZERO neutral-spine findings for cmd/backstop/gate.go's goFilePackageMatchesTarget
// and step_coverage.go's coverageSpecRelevantToFile sites (CLM-032).
func TestRatchet_GoPackageMatchersSiteUnGrandfatheredAfterDeGo(t *testing.T) {
	// cmd/backstop/gate.go must carry no neutral-spine finding (the go-package matcher
	// site is gone); step_coverage.go is covered by SITE 1.
	if baselineHasNeutralSpineForSite(t, "cmd/backstop/gate.go") {
		t.Errorf("the committed baseline must carry ZERO neutral-spine findings for the de-Go'd cmd/backstop/gate.go go-package matcher site")
	}
	// And the source itself must be clean of the de-Go'd matcher.
	src := readFileStr(t, "gate.go")
	if strings.Contains(src, "goFilePackageMatchesTarget") {
		t.Error("gate.go must no longer carry goFilePackageMatchesTarget (the go-package matcher was de-Go'd by SPEC-043)")
	}
}

// TestRatchet_SelfPackEnforcementFlipsToBlockZeroBaselineWhenAllSitesClean proves the
// TERMINAL FLIP: with all three sites clean, the dogfood backstop.yml sets
// backstop/self's neutral-spine enforcement to level block with a ZERO baseline VIA
// the REQ-007 per-pack key (CLM-033).
func TestRatchet_SelfPackEnforcementFlipsToBlockZeroBaselineWhenAllSitesClean(t *testing.T) {
	policy := dogfoodPolicy(t)
	pe, ok := policy["pack_engines"]
	if !ok {
		t.Fatal("dogfood backstop.yml must declare a pack_engines policy")
	}
	self, ok := pe.Sources["backstop/self"]
	if !ok {
		t.Fatalf("the flip must be delivered via the per-pack key: pack_engines.sources must scope backstop/self, got sources=%v", pe.Sources)
	}
	if self.Level != gate.PolicyBlock {
		t.Errorf("backstop/self must be flipped to level block, got %q", self.Level)
	}
	if self.AppliesTo != gate.AppliesToAllCode {
		t.Errorf("backstop/self must be flipped to a ZERO baseline (applies-to:all-code, no grandfathering), got applies-to=%q", self.AppliesTo)
	}
}

// TestRatchet_ReintroducedBakedLanguageLiteralRedsOutright proves THE WALL: after the
// flip, a deliberately reintroduced baked language literal on a neutral-spine site
// REDs the gate outright as net-new against the zero baseline (CLM-034). Driven
// through the REAL dogfood policy applied to a fresh backstop/self neutral-spine
// finding.
func TestRatchet_ReintroducedBakedLanguageLiteralRedsOutright(t *testing.T) {
	// A deliberately reintroduced baked `.go` literal on the neutral spine surfaces as
	// a backstop/self neutral-spine finding.
	reintroduced := gate.Violation{
		Rule:       "backstop/self/backstop.packs.backstop.self.rules." + neutralSpineRuleFragment,
		File:       "pkg/gate/step_coverage.go",
		Message:    `language literal on the neutral spine: if !strings.HasSuffix(path, ".go")`,
		Severity:   "error",
		RegionHash: "reintroduced-baked-go-literal",
		SourcePack: "backstop/self",
	}
	step := gate.StepResult{StepName: "pack_engines", Status: "fail", Violations: []gate.Violation{reintroduced}}

	// A baseline that ALREADY carries this exact finding — the discriminating setup:
	// under the OLD grandfathering policy the finding would be excused; only the flip's
	// ZERO baseline walls it off. This proves the WALL is the flip, not mere net-newness.
	baseline := &gate.BaselineArtifact{Violations: []gate.Violation{reintroduced}}

	// PRE-FLIP (the old whole-dimension grandfathering): the finding is baselined ⇒
	// grandfathered ⇒ passes. This is the wall the flip removes.
	preFlip := map[string]gate.DimensionPolicy{"pack_engines": {Level: gate.PolicyBlock, AppliesTo: gate.AppliesToNewCode}}
	if got := gate.ApplyPolicy([]gate.StepResult{step}, baseline, preFlip, nil)[0]; got.Status == "fail" {
		t.Fatalf("sanity: under the OLD grandfathering policy a baselined finding must be excused (pass), got %s — the wall test would not be discriminating", got.Status)
	}

	// POST-FLIP (the committed dogfood policy): the backstop/self zero-baseline blocks
	// the reintroduced literal OUTRIGHT, even though it is in the baseline (CLM-034).
	got := gate.ApplyPolicy([]gate.StepResult{step}, baseline, dogfoodPolicy(t), nil)[0]
	if got.Status != "fail" {
		t.Errorf("THE WALL: after the flip, a reintroduced baked language literal on the neutral spine must RED outright against the zero baseline, got %s", got.Status)
	}
}

// TestRatchet_FlipSequencedAfterPillarASitesClean proves the ORDERING GUARD: the
// terminal flip is PROHIBITED while any of the three Pillar-A sites still flags — the
// flip is applied ONLY BECAUSE all three sites are de-Go'd and clean (CLM-035). If a
// site still carried a neutral-spine literal, the flip would prematurely RED the
// dogfood gate, so the flip's presence is gated on the sites being clean.
func TestRatchet_FlipSequencedAfterPillarASitesClean(t *testing.T) {
	sitesClean := pillarASitesClean(t)
	flipApplied := flipPresentInDogfoodConfig(t)

	if !sitesClean {
		// The ordering guard: with a still-flagging site, the flip MUST NOT be applied.
		if flipApplied {
			t.Error("ORDERING GUARD VIOLATED: the terminal backstop/self block+zero-baseline flip must NOT be applied while a Pillar-A site still flags a neutral-spine literal")
		}
		return
	}
	// All three sites are clean ⇒ the flip is legitimately sequenced last and applied.
	if !flipApplied {
		t.Error("all three Pillar-A sites are de-Go'd and clean, so the terminal flip must be applied (backstop/self → block + zero baseline via the per-pack key)")
	}
}

// pillarASitesClean reports whether the three Pillar-A consumer sites carry NO baked
// neutral-spine language literal (`.go`/`_test.go`/`./...`) — the de-Go'd precondition
// for the terminal flip. This is a coarse source-text PROXY for the ordering guard;
// the REAL wall is the LIVE gate under the flip (a reintroduced literal REDs — proven
// by TestRatchet_ReintroducedBakedLanguageLiteralRedsOutright / CLM-034). The proxy is
// STRICT: it counts ANY such literal as not-clean, INCLUDING a nosemgrep-suppressed
// one — a suppressed baked literal is still a baked assumption on the neutral spine,
// and the de-Go'd sites carry none, so the strict proxy never false-negatives here.
func pillarASitesClean(t *testing.T) bool {
	t.Helper()
	for _, rel := range []string{
		filepath.Join("pkg", "gate", "step_coverage.go"),
		filepath.Join("pkg", "gate", "step_testverify.go"),
		filepath.Join("cmd", "backstop", "gate.go"),
	} {
		src, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		for _, banned := range []string{`"` + `.go"`, `"_test` + `.go"`, `"./` + `..."`} {
			if strings.Contains(string(src), banned) {
				return false
			}
		}
	}
	return true
}

// flipPresentInDogfoodConfig reports whether the dogfood backstop.yml applies the
// backstop/self → block + zero-baseline flip via the per-pack key.
func flipPresentInDogfoodConfig(t *testing.T) bool {
	t.Helper()
	pe, ok := dogfoodPolicy(t)["pack_engines"]
	if !ok {
		return false
	}
	self, ok := pe.Sources["backstop/self"]
	return ok && self.Level == gate.PolicyBlock && self.AppliesTo == gate.AppliesToAllCode
}
