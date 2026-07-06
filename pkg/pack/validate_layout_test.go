package pack_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

func containsPath(layout []string, want string) bool {
	for _, p := range layout {
		if p == want {
			return true
		}
	}
	return false
}

func TestExpectedLayout_PackYmlAlways(t *testing.T) {
	layout := pack.ExpectedLayout(makeMinimalManifest(), baseTestRegistry())
	if !containsPath(layout, "pack.yml") {
		t.Fatalf("expected pack.yml in %#v", layout)
	}
}

func TestExpectedLayout_GoModAlways(t *testing.T) {
	layout := pack.ExpectedLayout(makeMinimalManifest(), baseTestRegistry())
	if !containsPath(layout, "go.mod") {
		t.Fatalf("expected go.mod in %#v", layout)
	}
}

func TestExpectedLayout_EnforcementWithLayer2(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "semgrep"
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.rego"
	m.Content.Ruleset.Rules[0].Standard = "STD-GO-001"

	layout := pack.ExpectedLayout(m, baseTestRegistry())
	if !containsPath(layout, "rules/") {
		t.Fatalf("expected rules/ in %#v", layout)
	}
}

func TestExpectedLayout_WithLayer3(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Engine = "sandbox"
	m.Content.Ruleset.Rules[0].Category = "structural"
	m.Content.Ruleset.Rules[0].InputScope = "single-file"
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"

	layout := pack.ExpectedLayout(m, baseTestRegistry())
	if !containsPath(layout, "validators/") {
		t.Fatalf("expected validators/ in %#v", layout)
	}
}

func TestExpectedLayout_CodePack(t *testing.T) {
	m := makeMinimalManifest()
	m.Archetype = "code"

	layout := pack.ExpectedLayout(m, baseTestRegistry())
	if !containsPath(layout, "scaffolds/") {
		t.Fatalf("expected scaffolds/ in %#v", layout)
	}
}

func TestExpectedLayout_FixturesAlways(t *testing.T) {
	layout := pack.ExpectedLayout(makeMinimalManifest(), baseTestRegistry())
	if !containsPath(layout, "fixtures/rules/") {
		t.Fatalf("expected fixtures/rules/ in %#v", layout)
	}
}

func TestValidateFixtureDir_MatchesRuleID(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive[0].Path = "fixtures/rules/demo-rule/case-pass.txt"
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative[0].Path = "fixtures/rules/demo-rule/case-fail.txt"

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-034")
}

func TestValidateFixtureDir_Mismatch(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive[0].Path = "fixtures/rules/other-rule/case-pass.txt"
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative[0].Path = "fixtures/rules/other-rule/case-fail.txt"

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-035")
	for _, err := range errs {
		if strings.Contains(err.Field, "fixtures") {
			return
		}
	}
	t.Fatalf("expected fixture field error, got %#v", errs)
}
