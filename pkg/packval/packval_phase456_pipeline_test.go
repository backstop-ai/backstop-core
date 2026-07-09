package packval_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/engine"
	"github.com/bmanson/backstop-core/pkg/packval"
)

func TestPackVal_P4_CodePackNoRules(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules = nil
	if packval.RunArchetype(m).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P4_ScaffoldNoEnforcementRule(t *testing.T) {
	m := baseManifest()
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1"}}
	if packval.RunArchetype(m).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P4_CodePackWithRulesPass(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].PairsWith.Scaffolds = []string{"S1"}
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", PairsWith: packval.PairsWith{Rules: []string{"R1"}}}}
	if packval.RunArchetype(m).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P4_RuleNoPairsWithInCodePack(t *testing.T) {
	if packval.RunArchetype(baseManifest()).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P4_AllRulesHavePairsWith(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].PairsWith.Scaffolds = []string{"S1"}
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1", PairsWith: packval.PairsWith{Rules: []string{"R1"}}}}
	if packval.RunArchetype(m).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P4_EnforcementPackWithSDK(t *testing.T) {
	m := baseManifest()
	m.Archetype = "enforcement"
	m.Content.SDK = &packval.SDK{Provides: []string{"x"}}
	if packval.RunArchetype(m).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P4_EnforcementPackWithScaffolds(t *testing.T) {
	m := baseManifest()
	m.Archetype = "enforcement"
	m.Content.Scaffolds = []packval.Scaffold{{ID: "S1"}}
	if packval.RunArchetype(m).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P4_EnforcementPackRulesOnlyPass(t *testing.T) {
	m := baseManifest()
	m.Archetype = "enforcement"
	if packval.RunArchetype(m).Status != "pass" {
		t.Fatal("pass")
	}
}

func TestPackVal_P5_MissingLayer(t *testing.T) {
	m := baseManifest()
	// Clear the engine so the layer 1..3 requirement is ENFORCED: an engine-model
	// rule with a resolvable engine is now exempt from it (ISSUE-032 CLM-004), so this
	// test exercises the non-exempt (no-engine) path where a missing layer still fails.
	m.Content.Ruleset.Rules[0].Engine = ""
	m.Content.Ruleset.Rules[0].Layer = 0
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_InvalidLayer(t *testing.T) {
	m := baseManifest()
	// No engine: the layer requirement is enforced, so an out-of-range layer fails
	// (an engine-model rule would be exempt — ISSUE-032 CLM-004).
	m.Content.Ruleset.Rules[0].Engine = ""
	m.Content.Ruleset.Rules[0].Layer = 9
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_ValidLayerAndRiskClass(t *testing.T) {
	if packval.RunLayer(baseManifest(), makePackDir(t)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P5_Layer1WithCategoryFails(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Layer = 1
	m.Content.Ruleset.Rules[0].Category = "presence"
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_Layer2WithCategoryFails(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Layer = 2
	m.Content.Ruleset.Rules[0].Category = "presence"
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_Layer3WithCategoryPass(t *testing.T) {
	if packval.RunLayer(baseManifest(), makePackDir(t)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P5_Layer3MissingCategory(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Category = ""
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_CategoryPresenceAutoAccepted(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Category = "presence"
	if packval.RunLayer(m, makePackDir(t)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P5_CategoryStructuralAutoAccepted(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Category = "structural"
	if packval.RunLayer(m, makePackDir(t)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P5_CategoryOtherEmptyJustificationFails(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Category = "other"
	m.Content.Ruleset.Rules[0].Justification = ""
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_CategoryOtherMissingJustificationFails(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Category = "other"
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_CategoryOtherWithJustificationPass(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Category = "other"
	m.Content.Ruleset.Rules[0].Justification = "reason"
	if packval.RunLayer(m, makePackDir(t)).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P5_Layer3MissingInputScope(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].InputScope = ""
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_Layer3InvalidInputScope(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].InputScope = "single_file"
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_Layer3MissingValidator(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Validator = ""
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_Layer3ValidatorFileNotFound(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Validator = "validators/missing.sh"
	if packval.RunLayer(m, makePackDir(t)).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P5_Layer3ValidInputScopeAndValidator(t *testing.T) {
	if packval.RunLayer(baseManifest(), makePackDir(t)).Status != "pass" {
		t.Fatal("pass")
	}
}

func TestPackVal_P6_SecurityClaimNoBypassAttempt(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive[0].BypassAttempt = false
	if packval.RunRiskClass(m).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P6_SecurityClaimWithBypassAttempt(t *testing.T) {
	if packval.RunRiskClass(baseManifest()).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P6_NormalizeMixedFixtureFormats(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pack.yml", `
name: acme/example
version: 1.0.0
language: go
archetype: code
content:
  ruleset:
    rules:
      - id: R1
        risk_class: security
        claims:
          - id: C1
            fixtures:
              positive: [fixtures/p.go]
              negative:
                - path: fixtures/n.go
                  bypass_attempt: true
`)
	m, err := packval.ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil || m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive[0].Path == "" {
		t.Fatalf("parse err: %v", err)
	}
}
func TestPackVal_P6_NonSecurityNoBypassRequired(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "style"
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive[0].BypassAttempt = false
	if packval.RunRiskClass(m).Status != "pass" {
		t.Fatal("pass")
	}
}
func TestPackVal_P6_SecurityClaimsSharedFixtureFails(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims = append(m.Content.Ruleset.Rules[0].Claims, packval.Claim{ID: "C2", Fixtures: packval.Fixtures{Positive: []packval.FixtureRef{{Path: "fixtures/p.go", BypassAttempt: true}}, Negative: []packval.FixtureRef{{Path: "fixtures/n2.go"}}}})
	if packval.RunRiskClass(m).Status != "fail" {
		t.Fatal("fail")
	}
}
func TestPackVal_P6_SecurityClaimsIndependentFixturesPass(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims = append(m.Content.Ruleset.Rules[0].Claims, packval.Claim{ID: "C2", Fixtures: packval.Fixtures{Positive: []packval.FixtureRef{{Path: "fixtures/p2.go", BypassAttempt: true}}, Negative: []packval.FixtureRef{{Path: "fixtures/n2.go"}}}})
	if packval.RunRiskClass(m).Status != "pass" {
		t.Fatal("pass")
	}
}

func TestPackVal_CheckRunsManifestOnlyPhases(t *testing.T) {
	dir := makePackDir(t)
	p := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"})
	r := p.Run()
	for _, ph := range r.Phases {
		if ph.Phase == "phase3-fixtures" {
			t.Fatal("phase3 should not run in check")
		}
	}
}
func TestPackVal_CheckSkipsPhase3(t *testing.T) {
	dir := makePackDir(t)
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"}).Run()
	sawManifestPhase := false
	for _, ph := range r.Phases {
		if ph.Phase == "phase3-fixtures" && ph.Status != "skipped" {
			t.Fatal("phase3 fixtures must not execute in check mode")
		}
		if ph.Phase == "phase1-structural" {
			sawManifestPhase = true
		}
	}
	if !sawManifestPhase {
		t.Fatal("check mode must still run the manifest-only phases")
	}
}
func TestPackVal_TestRunsAllSixPhases(t *testing.T) {
	dir := makePackDir(t)
	p := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "test", Executor: &packval.MockExecutor{}})
	r := p.Run()
	if len(r.Phases) != 6 {
		t.Fatalf("want 6 phases got %d", len(r.Phases))
	}
}
func TestPackVal_EarlyTermination_P1FailSkipsP2(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pack.yml", "bad:[")
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "test"}).Run()
	if len(r.Phases) < 2 || r.Phases[1].Status != "skipped" {
		t.Fatal("expected skipped")
	}
}
func TestPackVal_EarlyTermination_P2FailSkipsP3(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims = nil
	dir := makePackDir(t)
	_ = os.WriteFile(filepath.Join(dir, "pack.yml"), []byte(strings.TrimSpace(`
name: acme/example
version: 1.0.0
language: go
archetype: code
content:
  ruleset:
    rules:
      - id: R1
        file: rules/r1.yml
        risk_class: security
        layer: 3
        category: presence
        input_scope: single-file
        validator: validators/v.sh
        claims: []
`)), 0o644)
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "test", Executor: &packval.MockExecutor{}}).Run()
	if len(r.Phases) < 3 || r.Phases[2].Status != "skipped" {
		t.Fatal("expected p3 skipped")
	}
	_ = m
}
func TestPackVal_EarlyTermination_P3FailSkipsP4P5P6(t *testing.T) {
	dir := makePackDir(t)
	mock := &packval.MockExecutor{EngineFn: func(_ string, _ engine.EngineBinding, _ []string) (packval.ExecutionResult, error) {
		return packval.ExecutionResult{Passed: false}, nil
	}}
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "test", Executor: mock}).Run()
	if r.Phases[3].Status != "skipped" || r.Phases[4].Status != "skipped" || r.Phases[5].Status != "skipped" {
		t.Fatal("expected downstream skips")
	}
}
func TestPackVal_EarlyTermination_P4FailSkipsP5(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "pack.yml", strings.TrimSpace(`
name: acme/example
version: 1.0.0
language: go
archetype: code
content: {ruleset: {rules: []}}
`))
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"}).Run()
	foundSkipped := false
	for _, ph := range r.Phases {
		if ph.Status == "skipped" {
			foundSkipped = true
		}
	}
	if !foundSkipped {
		t.Fatal("expected downstream skip")
	}
}
func TestPackVal_EarlyTermination_P5FailSkipsP6(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "pack.yml", strings.TrimSpace(`
name: acme/example
version: 1.0.0
language: go
archetype: code
content:
  ruleset:
    rules:
      - id: R1
        file: rules/r1.yml
        risk_class: security
        layer: 3
        category: presence
        input_scope: bad
        validator: validators/v.sh
        claims:
          - id: C1
            fixtures:
              positive:
                - path: fixtures/p.go
                  bypass_attempt: true
              negative:
                - fixtures/n.go
`))
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"}).Run()
	if r.Phases[len(r.Phases)-1].Status != "skipped" {
		t.Fatal("expected skip")
	}
}
func TestPackVal_AllPhasesRunOnSuccess(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "pack.yml", strings.TrimSpace(`
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
content:
  ruleset:
    rules:
      - id: R1
        engine: semgrep
        file: rules/r1.yml
        risk_class: security
        layer: 1
        claims:
          - id: C1
            fixtures:
              positive:
                - path: fixtures/p.go
                  bypass_attempt: true
              negative:
                - fixtures/n.go
`))
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "test", Executor: newFixtureMock(true, true)}).Run()
	last := r.Phases[len(r.Phases)-1]
	if last.Status == "skipped" {
		t.Fatal("expected full run")
	}
}
func TestPackVal_Idempotent(t *testing.T) {
	dir := makePackDir(t)
	p := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"})
	r1, r2 := p.Run(), p.Run()
	if r1.Status != r2.Status || len(r1.Errors) != len(r2.Errors) {
		t.Fatal("not idempotent")
	}
}
func TestPackVal_NoSideEffects(t *testing.T) {
	dir := makePackDir(t)
	before, err := os.ReadFile(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	_ = packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"}).Run()
	after, err := os.ReadFile(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("unexpected side effects")
	}
}
func TestPackVal_ErrorOrdering(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "pack.yml", strings.TrimSpace(`
name: acme/example
version: x
language: bad
archetype: bad
content: {}
`))
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"}).Run()
	if len(r.Errors) == 0 || r.Errors[0].Phase != "phase1-structural" {
		t.Fatal("unexpected ordering")
	}
}
