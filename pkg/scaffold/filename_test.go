package scaffold

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// rootAt builds an UNCONFIGURED artifact.Root at path, which reproduces exactly the
// behavior these tests asserted when TargetDir took a bare projectRoot string.
func rootAt(path string) artifact.Root {
	return artifact.Root{Path: path}
}

// --- Directory tests ---

func TestArtifactNew_Directory_Spec(t *testing.T) {
	got := TargetDir("spec", rootAt("/root"))
	if got != "/root/specs" {
		t.Fatalf("expected /root/specs, got %q", got)
	}
}

func TestArtifactNew_Directory_Plan(t *testing.T) {
	got := TargetDir("plan", rootAt("/root"))
	if got != "/root/plans" {
		t.Fatalf("expected /root/plans, got %q", got)
	}
}

func TestArtifactNew_Directory_Issue(t *testing.T) {
	got := TargetDir("issue", rootAt("/root"))
	if got != "/root/issues" {
		t.Fatalf("expected /root/issues, got %q", got)
	}
}

func TestArtifactNew_Directory_ADR(t *testing.T) {
	got := TargetDir("adr", rootAt("/root"))
	if got != "/root/adrs" {
		t.Fatalf("expected /root/adrs, got %q", got)
	}
}

func TestArtifactNew_Directory_Directive(t *testing.T) {
	got := TargetDir("directive", rootAt("/root"))
	if got != "/root/directives" {
		t.Fatalf("expected /root/directives, got %q", got)
	}
}

func TestArtifactNew_Directory_Bundle(t *testing.T) {
	got := TargetDir("bundle", rootAt("/root"))
	if got != "/root/bundles" {
		t.Fatalf("expected /root/bundles, got %q", got)
	}
}

func TestArtifactNew_Directory_Capability(t *testing.T) {
	got := TargetDir("capability", rootAt("/root"))
	if got != "/root/capabilities" {
		t.Fatalf("expected /root/capabilities, got %q", got)
	}
}

// --- Filename tests ---

func TestArtifactNew_Filename_Spec(t *testing.T) {
	got := Filename("spec", "001", "my-spec", "")
	expected := "SPEC-001-my-spec.spec.md"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArtifactNew_Filename_PlanSpec(t *testing.T) {
	got := Filename("plan", "002", "my-plan", "SPEC-002")
	expected := "PLAN-SPEC-002-my-plan.plan.yml"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArtifactNew_Filename_PlanIssue(t *testing.T) {
	got := Filename("plan", "005", "my-plan", "ISSUE-005")
	expected := "PLAN-ISSUE-005-my-plan.plan.yml"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArtifactNew_Filename_Issue(t *testing.T) {
	got := Filename("issue", "003", "my-issue", "")
	expected := "ISSUE-003-my-issue.issue.md"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArtifactNew_Filename_ADR(t *testing.T) {
	got := Filename("adr", "0001", "my-adr", "")
	expected := "ADR-0001-my-adr.adr.md"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArtifactNew_Filename_Directive(t *testing.T) {
	got := Filename("directive", "010", "my-dir", "")
	expected := "DIR-010-my-dir.directive.md"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArtifactNew_Filename_Bundle(t *testing.T) {
	got := Filename("bundle", "001", "my-bundle", "")
	expected := "BUNDLE-001-my-bundle.bundle.md"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestArtifactNew_Filename_Capability(t *testing.T) {
	got := Filename("capability", "001", "my-cap", "")
	expected := "CAP-001-my-cap.capability.yml"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
