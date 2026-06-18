package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

func TestValidateLayer2_BothFields(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-006")
}

func TestValidateLayer2_MissingRuleField(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-007")
}

func TestValidateLayer2_MissingStandard(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-008")
}

func TestValidateLayer1_WithRuleField(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "config-file"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-009")
}

func TestValidateLayer3_WithRuleField(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "sandbox"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo-rule.rego"
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].InputScope = "single-file"
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-010")
}
