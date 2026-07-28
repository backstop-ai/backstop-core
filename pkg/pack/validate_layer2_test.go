package pack_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

func TestValidateLayer2_BothFields(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-006")
}

func TestValidateLayer2_MissingRuleField(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-020-engine-field-contract")
}

func TestValidateLayer2_MissingStandard(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-020-engine-field-contract")
}

func TestValidateLayer1_WithRuleField(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "config-file"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-020-engine-field-contract")
}

func TestValidateLayer3_WithRuleField(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "sandbox"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].InputScope = "single-file"
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-020-engine-field-contract")
}
