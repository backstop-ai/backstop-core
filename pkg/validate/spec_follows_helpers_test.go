package validate_test

import "github.com/bmanson/backstop-core/pkg/artifact"

func validSpecWithFollows(follows interface{}) *artifact.ParsedArtifact {
	art := validSpecArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	firstReq := reqs[0].(map[string]interface{})
	firstReq["follows"] = follows
	return art
}

func validSpecWithReviewQuestions(present bool, empty bool) *artifact.ParsedArtifact {
	art := validSpecArtifact()

	if present {
		art.Sections = append(art.Sections, "Review Questions")
		if empty {
			art.Frontmatter["review_questions"] = ""
		} else {
			art.Frontmatter["review_questions"] = "- What could fail under concurrent load?"
		}
	}

	return art
}
