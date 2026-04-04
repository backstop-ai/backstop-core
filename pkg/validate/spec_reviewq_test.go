package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

func TestSpec_ReviewQuestions_PresentPasses(t *testing.T) {
	art := validSpecWithReviewQuestions(true, false)
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/review-questions-empty")
}

func TestSpec_ReviewQuestions_EmptyFails(t *testing.T) {
	art := validSpecWithReviewQuestions(true, true)
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/review-questions-empty")
}

func TestSpec_ReviewQuestions_OmittedPasses(t *testing.T) {
	art := validSpecWithReviewQuestions(false, false)
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/review-questions-empty")
}
