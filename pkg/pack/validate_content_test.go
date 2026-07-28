package pack_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

func TestValidateContent_EnforcementRulesetOnly(t *testing.T) {
	m := makeMinimalManifest()

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-001")
}

func TestValidateContent_EnforcementWithScaffolds(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Scaffolds = []pack.Scaffold{{ID: "scaffold-a"}}

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-002")
}

func TestValidateContent_EnforcementWithSDK(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.SDK = &pack.SDK{Module: "example.com/sdk", Version: "1.0.0", Provides: []string{"x"}}

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-003")
}

func TestValidateContent_CodeWithRulesetAndScaffolds(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "code"
	m.Content.Scaffolds = []pack.Scaffold{
		{
			ID:          "svc",
			Version:     "1.0.0",
			Tier:        "complete",
			Path:        "scaffolds/svc",
			TestCommand: "go test ./...",
			UseWhen:     []string{"service"},
			Assumes:     []string{"go"},
			PairsWith:   pack.PairsWith{Rules: []string{"demo-rule"}},
		},
	}
	m.Content.Ruleset.Rules[0].PairsWith = pack.PairsWith{Scaffolds: []string{"svc"}}

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-004")
}

func TestValidateContent_UnknownContentType(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules = nil
	m.Content.Scaffolds = nil
	m.Content.SDK = nil

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-005")
}
