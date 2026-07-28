package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// TestGateTypeHasDedicatedStep pins which gate_types own a dedicated gate step and
// must therefore be EXCLUDED from the generic pack_engines/findings dispatch.
func TestGateTypeHasDedicatedStep(t *testing.T) {
	dedicated := []engine.GateType{engine.GateTypeSubstantiveness, engine.GateTypeContracts, engine.GateTypeCoverage}
	for _, gt := range dedicated {
		if !gateTypeHasDedicatedStep(gt) {
			t.Errorf("gate_type %v must be dedicated-step (its own gate step dispatches it)", gt)
		}
	}
	generic := []engine.GateType{engine.GateTypeLint, engine.GateTypeBuild, engine.GateTypeTest, engine.GateTypeFindings}
	for _, gt := range generic {
		if gateTypeHasDedicatedStep(gt) {
			t.Errorf("gate_type %v is a generic stage and must run through pack_engines", gt)
		}
	}
}

// TestExcludeDedicatedStepRules_PartitionsByGateType proves the generic pack_engines
// dispatch keeps ONLY generic-stage rules and drops traceability rules (which their
// dedicated steps dispatch per-dimension). Routing is by DECLARED gate_type — this is
// the fix for the real-CLI defect where an installed contracts/substantiveness pack
// double-ran context-free through pack_engines and emitted garbage findings.
func TestExcludeDedicatedStepRules_PartitionsByGateType(t *testing.T) {
	m := &pack.Manifest{
		NormalizedName: "org/mixed",
		Engines: map[string]pack.EngineSpec{
			"sub-eng":  {Binding: engine.EngineBinding{GateType: engine.GateTypeSubstantiveness}},
			"con-eng":  {Binding: engine.EngineBinding{GateType: engine.GateTypeContracts}},
			"find-eng": {Binding: engine.EngineBinding{GateType: engine.GateTypeFindings}},
			"lint-eng": {Binding: engine.EngineBinding{GateType: engine.GateTypeLint}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "hollow", Engine: "sub-eng"},
			{ID: "contract", Engine: "con-eng"},
			{ID: "find", Engine: "find-eng"},
			{ID: "lint", Engine: "lint-eng"},
		}}},
	}

	out := excludeDedicatedStepRules([]*pack.Manifest{m})
	if len(out) != 1 {
		t.Fatalf("expected 1 manifest back, got %d", len(out))
	}
	got := map[string]bool{}
	for _, r := range out[0].Content.Ruleset.Rules {
		got[r.ID] = true
	}
	if got["hollow"] || got["contract"] {
		t.Errorf("pack_engines must NOT run dedicated-step (substantiveness/contracts) rules, got %v", got)
	}
	if !got["find"] || !got["lint"] {
		t.Errorf("pack_engines MUST run generic-stage (findings/lint) rules, got %v", got)
	}

	// The input manifest must not be mutated — the filter clones.
	if len(m.Content.Ruleset.Rules) != 4 {
		t.Errorf("excludeDedicatedStepRules must not mutate the input manifest, got %d rules", len(m.Content.Ruleset.Rules))
	}
}

// TestExcludeDedicatedStepRules_PassThroughWhenNoneExcluded leaves a manifest with
// only generic-stage rules untouched (same pointer, no needless clone).
func TestExcludeDedicatedStepRules_PassThroughWhenNoneExcluded(t *testing.T) {
	m := &pack.Manifest{
		NormalizedName: "org/lint",
		Engines: map[string]pack.EngineSpec{
			"lint-eng": {Binding: engine.EngineBinding{GateType: engine.GateTypeLint}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{{ID: "lint", Engine: "lint-eng"}}}},
	}
	out := excludeDedicatedStepRules([]*pack.Manifest{m})
	if len(out) != 1 || out[0] != m {
		t.Fatalf("a manifest with no dedicated-step rules must pass through unchanged (same pointer)")
	}
}
