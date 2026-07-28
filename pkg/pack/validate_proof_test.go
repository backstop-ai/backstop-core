package pack_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

func TestValidateFixtureProof_EnforcementWithFixtures(t *testing.T) {
	m := makeMinimalManifest()

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-049")
}

func TestValidateFixtureProof_CodeWithFixtures(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "code"
	m.Content.Scaffolds = []pack.Scaffold{
		{ID: "svc", PairsWith: pack.PairsWith{Rules: []string{"demo-rule"}}},
	}
	m.Content.Ruleset.Rules[0].PairsWith = pack.PairsWith{Scaffolds: []string{"svc"}}

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-050")
}

func TestValidateFixtureProof_RuleNoClaims(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Claims = nil

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-051")
}

func TestValidateConstraint_Layer3MissingIsolationFields(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "sandbox"
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].InputScope = ""
	m.Content.Ruleset.Rules[0].Validator = ""

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-040")
}
