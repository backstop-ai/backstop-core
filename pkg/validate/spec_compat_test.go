package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

func TestSpec_ExistingRules_FilenamePatternsStillWork(t *testing.T) {
	art := validSpecArtifact()
	art.Filename = "bad-spec.md"
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/filename-pattern")
}

func TestSpec_ExistingRules_RequirementsArrayStillValidated(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "requirements")
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/requirements-required")
}

func TestSpec_ExistingRules_ClaimsArrayStillValidated(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "claims")
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/claims-required")
}

func TestSpec_ExistingRules_VerificationBlockStillValidated(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "verification")
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/verification-required")
}
