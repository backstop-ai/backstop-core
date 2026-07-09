package packval

import (
	"strings"
	"testing"
)

// layerErrorsByRule returns the set of rule IDs that produced a phase5 "invalid
// layer" error.
func layerErrorsByRule(res *PhaseResult) map[string]bool {
	out := map[string]bool{}
	for _, e := range res.Errors {
		if e.Check == "layer" && strings.Contains(e.Message, "invalid layer") {
			out[e.Rule] = true
		}
	}
	return out
}

// TestRunLayer_EngineRuleExemptFromLayer (CLM-004, B1): a rule with a non-empty
// RESOLVABLE engine and Layer==0 is EXEMPT from the layer 1..3 requirement (engine-
// model rules like go-standards carry no layer), while a plain rule with Layer==0 and
// no engine STILL errors "invalid layer".
func TestRunLayer_EngineRuleExemptFromLayer(t *testing.T) {
	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		Content: Content{Ruleset: Ruleset{Rules: []Rule{
			{ID: "engine-rule", Engine: "semgrep", RiskClass: "correctness"}, // Layer 0, resolvable engine
			{ID: "plain-rule", RiskClass: "correctness"},                     // Layer 0, NO engine
		}}},
	}
	res := RunLayer(pack, t.TempDir())
	layerErrs := layerErrorsByRule(res)
	if layerErrs["engine-rule"] {
		t.Error("engine-model rule (resolvable engine, no layer) must be EXEMPT from the invalid-layer error")
	}
	if !layerErrs["plain-rule"] {
		t.Error("plain rule with no engine and no layer must STILL error invalid layer")
	}
}

// TestRunLayer_UnresolvableEngineFailsLoud (CLM-013, B-GUARD): a claimless/layerless
// rule whose engine does NOT resolve must FAIL LOUD naming the engine — it is NOT
// silently exempted from the layer requirement. An EMPTY engine is never exempted
// either (it falls through to the normal invalid-layer error). This runs in the phase
// CHECK mode invokes (phase5), not phase3.
func TestRunLayer_UnresolvableEngineFailsLoud(t *testing.T) {
	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		Content: Content{Ruleset: Ruleset{Rules: []Rule{
			{ID: "bogus-rule", Engine: "totally-bogus", RiskClass: "correctness"},
			{ID: "empty-rule", RiskClass: "correctness"},
		}}},
	}
	res := RunLayer(pack, t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected phase5 to FAIL on an unresolvable engine, got status %q", res.Status)
	}
	namedBogus := false
	for _, e := range res.Errors {
		if e.Rule == "bogus-rule" && strings.Contains(e.Message, "totally-bogus") {
			namedBogus = true
		}
	}
	if !namedBogus {
		t.Errorf("expected a loud error naming the unresolvable engine %q; got %+v", "totally-bogus", res.Errors)
	}
	// The empty-engine rule must NOT slip through silently: it still errors.
	if !layerErrorsByRule(res)["empty-rule"] {
		t.Error("empty-engine rule with no layer must STILL error invalid layer (never exempted)")
	}
}
