package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

func TestValidateSecurityFixtures_WithBypass(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "security"
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative[0].BypassAttempt = true

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-028")
}

func TestValidateSecurityFixtures_NoBypass(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "security"
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative[0].BypassAttempt = false

	errs := pack.ValidateManifest(m)

	requireError(t, errs, "CLM-029")
}

func TestValidateSecurityFixtures_NonSecurityNoBypass(t *testing.T) {
	m := makeMinimalManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "style"
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative[0].BypassAttempt = false

	errs := pack.ValidateManifest(m)

	requireNoError(t, errs, "CLM-030")
}
