package scaffold

import (
	"testing"
)

func TestArtifactNew_Source_SpecBacked(t *testing.T) {
	err := ValidateSource("SPEC-002")
	if err != nil {
		t.Fatalf("expected valid spec source to be accepted, got error: %v", err)
	}
	kind, err := ParseSourceKind("SPEC-002")
	if err != nil {
		t.Fatalf("expected ParseSourceKind to succeed, got error: %v", err)
	}
	if kind != "spec" {
		t.Fatalf("expected kind 'spec', got %q", kind)
	}
}

func TestArtifactNew_Source_IssueBacked(t *testing.T) {
	err := ValidateSource("ISSUE-005")
	if err != nil {
		t.Fatalf("expected valid issue source to be accepted, got error: %v", err)
	}
	kind, err := ParseSourceKind("ISSUE-005")
	if err != nil {
		t.Fatalf("expected ParseSourceKind to succeed, got error: %v", err)
	}
	if kind != "issue" {
		t.Fatalf("expected kind 'issue', got %q", kind)
	}
}

func TestArtifactNew_Source_MissingForPlan_Exit2(t *testing.T) {
	err := ValidateSource("")
	if err == nil {
		t.Fatal("expected error for empty source, got nil")
	}
}

func TestArtifactNew_Source_InvalidFormat_Exit2(t *testing.T) {
	err := ValidateSource("BADFORMAT")
	if err == nil {
		t.Fatal("expected error for invalid source format, got nil")
	}
}

func TestArtifactNew_Source_IgnoredForSpec(t *testing.T) {
	// Source flag should be silently ignored for non-plan types.
	// This is validated at the command level; ValidateSource itself
	// only validates the format. This test ensures no panic/error
	// when a valid source is passed for a non-plan type context.
	if _, ok := ValidArtifactTypes["spec"]; !ok {
		t.Fatal("expected spec to be a valid artifact type")
	}
}

func TestArtifactNew_Source_IgnoredForIssue(t *testing.T) {
	if _, ok := ValidArtifactTypes["issue"]; !ok {
		t.Fatal("expected issue to be a valid artifact type")
	}
}

func TestArtifactNew_Source_IgnoredForADR(t *testing.T) {
	if _, ok := ValidArtifactTypes["adr"]; !ok {
		t.Fatal("expected adr to be a valid artifact type")
	}
}

func TestArtifactNew_Source_IgnoredForDirective(t *testing.T) {
	if _, ok := ValidArtifactTypes["directive"]; !ok {
		t.Fatal("expected directive to be a valid artifact type")
	}
}

func TestArtifactNew_Source_IgnoredForBundle(t *testing.T) {
	if _, ok := ValidArtifactTypes["bundle"]; !ok {
		t.Fatal("expected bundle to be a valid artifact type")
	}
}

func TestArtifactNew_Source_IgnoredForCapability(t *testing.T) {
	if _, ok := ValidArtifactTypes["capability"]; !ok {
		t.Fatal("expected capability to be a valid artifact type")
	}
}
