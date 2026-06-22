package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

func makeMinimalManifest() *pack.Manifest {
	return &pack.Manifest{
		Name:        "acme/demo",
		Version:     "1.0.0",
		Language:    "go",
		Archetype:   "enforcement",
		Description: "minimal manifest",
		Content: pack.Content{
			Ruleset: pack.Ruleset{
				Version: "1.0.0",
				Rules: []pack.Rule{
					{
						ID:        "demo-rule",
						RiskClass: "security",
						Engine:    "config-file",
						Claims: []pack.Claim{
							{
								ID:   "claim-1",
								Text: "must enforce behavior",
								Fixtures: pack.Fixtures{
									Positive: []pack.FixtureEntry{{Path: "fixtures/rules/demo-rule/positive.txt"}},
									Negative: []pack.FixtureEntry{{Path: "fixtures/rules/demo-rule/negative.txt", BypassAttempt: true}},
								},
							},
						},
					},
				},
			},
		},
	}
}

func hasError(errs []pack.ValidationError, rule string) bool {
	for _, err := range errs {
		if err.Rule == rule {
			return true
		}
	}
	return false
}

func requireError(t *testing.T, errs []pack.ValidationError, rule string) {
	t.Helper()
	if !hasError(errs, rule) {
		t.Fatalf("expected error %q, got %#v", rule, errs)
	}
}

func requireNoError(t *testing.T, errs []pack.ValidationError, rule string) {
	t.Helper()
	if hasError(errs, rule) {
		t.Fatalf("did not expect error %q, got %#v", rule, errs)
	}
}

// TestExpectedLayout_DerivedFromInputModeNotEngineName asserts ExpectedLayout
// derives the rules/ vs validators/ layout from each rule's RESOLVED binding
// InputMode/ScopeKind — NOT from the engine names "semgrep"/"ast-grep"/"sandbox".
// Two pack-declared engines with NON-standard names drive it: a rule-fed mode
// (rule-flags) yields rules/, and input_mode none yields validators/. A
// name-switch implementation would see neither name and emit neither directory,
// so this fixture can only pass off the InputMode derivation. CLM-025.
func TestExpectedLayout_DerivedFromInputModeNotEngineName(t *testing.T) {
	m := makeMinimalManifest()
	m.Engines = map[string]pack.EngineSpec{
		"acme-rulefed": {Binding: engine.EngineBinding{
			Command:   "acme-scan",
			InputMode: engine.InputModeRuleFlags,
			InputFlag: "--config",
			ScopeKind: engine.ScopeKindFileArgs,
		}},
		"acme-validator": {Binding: engine.EngineBinding{
			Command:   "",
			InputMode: engine.InputModeNone,
			ScopeKind: engine.ScopeKindFileArgs,
		}},
	}
	m.Content.Ruleset.Rules = []pack.Rule{
		{ID: "rule-fed", Engine: "acme-rulefed", RulePath: "rules/a.yml", Standard: "STD"},
		{ID: "validator-rule", Engine: "acme-validator", Validator: "validators/v.sh", InputScope: "single-file", Category: "structural"},
	}

	layout := pack.ExpectedLayout(m)
	if !containsPath(layout, "rules/") {
		t.Errorf("rule-fed (rule-flags) engine must yield rules/, got %#v", layout)
	}
	if !containsPath(layout, "validators/") {
		t.Errorf("input_mode none engine must yield validators/, got %#v", layout)
	}
}

// TestFieldContract_DeclaredOnBindingNotNameKeyed asserts the validator verifies
// a rule's populated fields against its declared engine's DECLARED FieldContract
// (on the binding), NOT a map keyed by engine name. The engine "acme-strict" is
// absent from any name-keyed default-contract map, so the only way its
// requires:[rule_path] contract can fire is by reading the DECLARED contract off
// the binding. A rule missing rule_path under it must fail; a complete rule must
// pass. CLM-036 (pkg/pack level).
func TestFieldContract_DeclaredOnBindingNotNameKeyed(t *testing.T) {
	declared := map[string]pack.EngineSpec{
		"acme-strict": {Binding: engine.EngineBinding{
			Command:   "acme-scan",
			InputMode: engine.InputModeRuleFlags,
			InputFlag: "--config",
			ScopeKind: engine.ScopeKindFileArgs,
			FieldContract: engine.FieldContract{
				Requires: []string{engine.FieldRulePath},
				Forbids:  []string{engine.FieldValidator},
			},
		}},
	}

	// Missing the declared-required rule_path: must produce a field-contract error
	// naming the field and engine.
	missing := makeMinimalManifest()
	missing.Engines = declared
	missing.Content.Ruleset.Rules = []pack.Rule{
		{ID: "demo-rule", Engine: "acme-strict", RiskClass: "security",
			Claims: missing.Content.Ruleset.Rules[0].Claims},
	}
	errs := pack.ValidateManifest(missing)
	if !hasFieldEngineError(errs, "rule_path", "acme-strict") {
		t.Fatalf("declared requires:[rule_path] must fire for a missing rule_path naming field+engine, got %#v", errs)
	}

	// Complete rule (rule_path present, no forbidden validator): the declared
	// contract is satisfied, so no field-contract error for this engine.
	ok := makeMinimalManifest()
	ok.Engines = declared
	ok.Content.Ruleset.Rules = []pack.Rule{
		{ID: "demo-rule", Engine: "acme-strict", RiskClass: "security", RulePath: "rules/a.yml",
			Claims: ok.Content.Ruleset.Rules[0].Claims},
	}
	errs = pack.ValidateManifest(ok)
	if hasFieldEngineError(errs, "rule_path", "acme-strict") {
		t.Fatalf("a rule satisfying the declared contract must not error on rule_path, got %#v", errs)
	}
}

func TestValidateManifest_AccumulatesErrors(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].RulePath = ""
	m.Content.Ruleset.Rules[0].Standard = ""

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-007")
	requireError(t, errs, "CLM-008")
}
