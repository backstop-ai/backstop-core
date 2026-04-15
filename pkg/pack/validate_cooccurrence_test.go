package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

func TestValidateCoOccurrence_CodePackValid(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "code"
	m.Content.Scaffolds = []pack.Scaffold{
		{
			ID:        "svc",
			PairsWith: pack.PairsWith{Rules: []string{"demo-rule"}},
		},
	}
	m.Content.Ruleset.Rules[0].PairsWith = pack.PairsWith{Scaffolds: []string{"svc"}}

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-044")
}

func TestValidateCoOccurrence_ScaffoldNoPairedRule(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "code"
	m.Content.Scaffolds = []pack.Scaffold{
		{
			ID:        "svc",
			PairsWith: pack.PairsWith{Rules: []string{"missing-rule"}},
		},
	}
	m.Content.Ruleset.Rules[0].PairsWith = pack.PairsWith{Scaffolds: []string{"svc"}}

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-045")
}

func TestValidateCoOccurrence_RuleNoPairedContent(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "code"
	m.Content.Scaffolds = []pack.Scaffold{
		{ID: "svc"},
	}
	m.Content.Ruleset.Rules[0].PairsWith = pack.PairsWith{Scaffolds: []string{"unknown-scaffold"}}

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-046")
}

func TestValidateCoOccurrence_EnforcementWithScaffolds(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "enforcement"
	m.Content.Scaffolds = []pack.Scaffold{{ID: "svc"}}

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-047")
}

func TestValidateCoOccurrence_EnforcementWithSDK(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "enforcement"
	m.Content.SDK = &pack.SDK{Module: "example.com/sdk", Version: "1.0.0", Provides: []string{"x"}}

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-048")
}
