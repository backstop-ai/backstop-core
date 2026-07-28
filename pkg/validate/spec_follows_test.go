package validate_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/validate"
)

func TestSpec_Follows_SingleStandardRule(t *testing.T) {
	art := validSpecWithFollows("STD-GO-001:GO-010")
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/requirement-follows-format")
	assertNoViolationRule(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_SingleRecipeName(t *testing.T) {
	art := validSpecWithFollows("error-handling-recipe")
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/requirement-follows-format")
	assertNoViolationRule(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_ArrayValid(t *testing.T) {
	art := validSpecWithFollows([]interface{}{"STD-GO-001:GO-010", "error-handling-recipe"})
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/requirement-follows-format")
	assertNoViolationRule(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_OmittedPasses(t *testing.T) {
	art := validSpecArtifact()
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/requirement-follows-format")
	assertNoViolationRule(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_StandardRuleFormatPasses(t *testing.T) {
	art := validSpecWithFollows("STD-JAVA-001:J-042")
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/requirement-follows-format")
	assertNoViolationRule(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_RecipeNameFormatPasses(t *testing.T) {
	art := validSpecWithFollows("error-handling-recipe")
	result := validate.Spec(art, specSchema())
	assertNoViolationRule(t, result, "spec/requirement-follows-format")
	assertNoViolationRule(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_EmptyStringFails(t *testing.T) {
	art := validSpecWithFollows("")
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_EmptyArrayFails(t *testing.T) {
	art := validSpecWithFollows([]interface{}{})
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/requirement-follows-empty")
}

func TestSpec_Follows_InvalidFormatFails(t *testing.T) {
	art := validSpecWithFollows("INVALID")
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/requirement-follows-format")
}

func TestSpec_Follows_NeitherFormatFails(t *testing.T) {
	cases := []string{"123-bad", "STD-go-001:go-010"}
	for _, follows := range cases {
		t.Run(follows, func(t *testing.T) {
			art := validSpecWithFollows(follows)
			result := validate.Spec(art, specSchema())
			assertHasViolation(t, result, "spec/requirement-follows-format")
		})
	}
}

func TestSpec_Follows_MixedValidInvalidFails(t *testing.T) {
	art := validSpecWithFollows([]interface{}{"STD-GO-001:GO-010", "INVALID"})
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/requirement-follows-format")
}

func TestSpec_Follows_LowercaseStdPrefixFails(t *testing.T) {
	art := validSpecWithFollows("std-go-001:GO-010")
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/requirement-follows-format")
}

func TestSpec_Follows_UppercaseRecipeNameFails(t *testing.T) {
	art := validSpecWithFollows("Error-Handling")
	result := validate.Spec(art, specSchema())
	assertHasViolation(t, result, "spec/requirement-follows-format")
}
