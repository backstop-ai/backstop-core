package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

func TestValidateLayer3_SingleFileWithValidator(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "structural"

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-019")
}

func TestValidateLayer3_MultiFileWithValidator(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].InputScope = "multi_file"

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-020")
}

func TestValidateLayer3_MissingInputScope(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].InputScope = ""

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-021")
}

func TestValidateLayer3_MissingValidator(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].Validator = ""

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-022")
}

func TestValidateLayer3_InvalidInputScope(t *testing.T) {
	m := makeLayer3Manifest()
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].InputScope = "repo"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-023")
}

func TestValidateLayer1_WithInputScope(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].InputScope = "single_file"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-024")
}

func TestValidateLayer2_WithInputScope(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Layer = 2
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.rego"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"
	m.Content.Ruleset.Rules[0].InputScope = "single_file"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-025")
}

func TestValidateLayer1_WithValidator(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-026")
}

func TestValidateLayer2_WithValidator(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Layer = 2
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.rego"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-027")
}
