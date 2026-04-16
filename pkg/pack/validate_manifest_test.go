package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
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
						Layer:     1,
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

func TestValidateManifest_AccumulatesErrors(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].Layer = 2
	m.Content.Ruleset.Rules[0].RulePath = ""
	m.Content.Ruleset.Rules[0].Standard = ""

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-007")
	requireError(t, errs, "CLM-008")
}
