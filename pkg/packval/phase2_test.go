package packval

import (
	"strings"
	"testing"
)

// claimsErrorsByRule returns the set of rule IDs that produced a phase2 "rule has no
// claims" error.
func claimsErrorsByRule(res *PhaseResult) map[string]bool {
	out := map[string]bool{}
	for _, e := range res.Errors {
		if e.Check == "claims" && strings.Contains(e.Message, "rule has no claims") {
			out[e.Rule] = true
		}
	}
	return out
}

// TestRunCoherence_EngineRuleExemptFromClaims (CLM-005, B2): a claimless rule with a
// non-empty RESOLVABLE engine is EXEMPT from the "rule has no claims" error (mechanism
// engine rules like go-toolchain's build/test/coverage carry an engine and no claims),
// while a claimless rule with NO engine STILL errors.
func TestRunCoherence_EngineRuleExemptFromClaims(t *testing.T) {
	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		Content: Content{Ruleset: Ruleset{Rules: []Rule{
			{ID: "engine-rule", Engine: "semgrep", RiskClass: "correctness"}, // claimless, resolvable engine
			{ID: "plain-rule", RiskClass: "correctness"},                     // claimless, NO engine
		}}},
	}
	res := RunCoherence(pack, t.TempDir())
	claimsErrs := claimsErrorsByRule(res)
	if claimsErrs["engine-rule"] {
		t.Error("claimless rule with a resolvable engine must be EXEMPT from the no-claims error")
	}
	if !claimsErrs["plain-rule"] {
		t.Error("claimless rule with no engine must STILL error rule has no claims")
	}
}

// TestRunCoherence_UnresolvableEngineFailsLoud (CLM-013, B-GUARD): a claimless rule
// whose engine does NOT resolve must FAIL LOUD naming the engine — it is NOT silently
// exempted from the claims requirement. An EMPTY engine is never exempted. Runs in the
// phase CHECK mode invokes (phase2), not phase3.
func TestRunCoherence_UnresolvableEngineFailsLoud(t *testing.T) {
	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		Content: Content{Ruleset: Ruleset{Rules: []Rule{
			{ID: "bogus-rule", Engine: "totally-bogus", RiskClass: "correctness"},
			{ID: "empty-rule", RiskClass: "correctness"},
		}}},
	}
	res := RunCoherence(pack, t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected phase2 to FAIL on an unresolvable engine, got status %q", res.Status)
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
	if !claimsErrorsByRule(res)["empty-rule"] {
		t.Error("empty-engine claimless rule must STILL error rule has no claims (never exempted)")
	}
}

// TestBadEnginePack_FailsCheckAndTest (CLM-013, B-GUARD): the vacuous-green guard
// fixture — a rule with `engine: totally-bogus` (no file, no claims, no layer) — must
// FAIL both `pack check` (mode check, which never runs phase3) AND `pack test` (mode
// test). The unknown engine must be named. This proves the exemption is not a free
// escape hatch dodgeable by declaring a bogus engine.
func TestBadEnginePack_FailsCheckAndTest(t *testing.T) {
	for _, mode := range []string{"check", "test"} {
		p := NewPipeline("testdata/bad-engine-pack", PipelineOptions{Mode: mode})
		res := p.Run()
		if res.Status != "fail" {
			t.Errorf("mode %s: expected bad-engine pack to FAIL, got status %q", mode, res.Status)
		}
		named := false
		for _, e := range res.Errors {
			if strings.Contains(e.Message, "totally-bogus") {
				named = true
			}
		}
		if !named {
			t.Errorf("mode %s: expected a loud error naming the unknown engine totally-bogus; got %+v", mode, res.Errors)
		}
	}
}
