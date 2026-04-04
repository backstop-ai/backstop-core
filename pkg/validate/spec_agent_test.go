package validate_test

import (
	"os"
	"strings"
	"testing"
)

func TestSpecAuthorAgent_ContainsFollowsInstructions(t *testing.T) {
	data, err := os.ReadFile("../../.claude/agents/spec-author.md")
	if err != nil {
		t.Fatalf("cannot read spec-author agent file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "follows") || !strings.Contains(content, "STD-LANG-NNN:RULE-ID") {
		t.Fatalf("spec-author agent missing follows field instructions")
	}
}

func TestSpecAuthorAgent_ContainsReviewQuestionsInstructions(t *testing.T) {
	data, err := os.ReadFile("../../.claude/agents/spec-author.md")
	if err != nil {
		t.Fatalf("cannot read spec-author agent file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Review Questions") || !strings.Contains(content, "adversarial") {
		t.Fatalf("spec-author agent missing review questions instructions")
	}
}

func TestSpecAuthorAgent_ContainsEscalationInstructions(t *testing.T) {
	data, err := os.ReadFile("../../.claude/agents/spec-author.md")
	if err != nil {
		t.Fatalf("cannot read spec-author agent file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "DD-13") || !strings.Contains(content, "escalate") {
		t.Fatalf("spec-author agent missing DD-13 escalation instructions")
	}
}

func TestImplReviewerAgent_ContainsReviewQuestionsInstructions(t *testing.T) {
	data, err := os.ReadFile("../../.claude/agents/impl-reviewer.md")
	if err != nil {
		t.Fatalf("cannot read impl-reviewer agent file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Review Questions") || !strings.Contains(content, "code review") {
		t.Fatalf("impl-reviewer agent missing review questions instructions")
	}
}

func TestImplReviewerAgent_ContainsFollowsCheckInstructions(t *testing.T) {
	data, err := os.ReadFile("../../.claude/agents/impl-reviewer.md")
	if err != nil {
		t.Fatalf("cannot read impl-reviewer agent file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "follows") || !strings.Contains(content, "standard rules") {
		t.Fatalf("impl-reviewer agent missing follows standard-rules checks")
	}
}
