package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

func makeLayer3Manifest() *pack.Manifest {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "sandbox"
	m.Content.Ruleset.Rules[0].InputScope = "single-file"
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"
	return m
}

func TestValidateLayer3Category_Presence(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "presence"

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-011")
}

func TestValidateLayer3Category_Structural(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "structural"

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-012")
}

func TestValidateLayer3Category_OtherWithJustification(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "other"
	m.Content.Ruleset.Rules[0].Justification = "Legacy ecosystem gap."

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-013")
}

func TestValidateLayer3Category_OtherMissingJustification(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "other"
	m.Content.Ruleset.Rules[0].Justification = "  "

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-014")
}

func TestValidateLayer3Category_Missing(t *testing.T) {
	m := makeLayer3Manifest()

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-015")
}

func TestValidateLayer3Category_Invalid(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "business"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-016")
}

func TestValidateLayer1_WithCategory(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "config-file"
	m.Content.Ruleset.Rules[0].Category = "structural"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-017")
}

func TestValidateLayer2_WithCategory(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.rego"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"
	m.Content.Ruleset.Rules[0].Category = "structural"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-018")
}
