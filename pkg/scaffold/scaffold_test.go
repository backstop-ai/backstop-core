package scaffold

import (
	"strings"
	"testing"
)

// --- Frontmatter tests ---

func TestArtifactNew_Frontmatter_Spec(t *testing.T) {
	content, err := Scaffold("spec", "001", "my-spec", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, field := range []string{"title:", "number:", "created:", "status:", "schema_version:", "spec_version:"} {
		if !strings.Contains(s, field) {
			t.Errorf("spec frontmatter missing %q", field)
		}
	}
	if !strings.Contains(s, "SPEC-001") {
		t.Error("spec frontmatter missing SPEC-001 number")
	}
}

func TestArtifactNew_Frontmatter_PlanSpec(t *testing.T) {
	content, err := Scaffold("plan", "002", "my-plan", "2026-04-04", "SPEC-002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "plan_id: PLAN-SPEC-002") {
		t.Error("plan frontmatter missing plan_id PLAN-SPEC-002")
	}
	if !strings.Contains(s, "spec_id: SPEC-002") {
		t.Error("plan frontmatter missing spec_id SPEC-002")
	}
	if !strings.Contains(s, "created:") {
		t.Error("plan frontmatter missing created field")
	}
	if !strings.Contains(s, "status:") {
		t.Error("plan frontmatter missing status field")
	}
}

func TestArtifactNew_Frontmatter_PlanIssue(t *testing.T) {
	content, err := Scaffold("plan", "005", "my-plan", "2026-04-04", "ISSUE-005")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "plan_id: PLAN-ISSUE-005") {
		t.Error("plan frontmatter missing plan_id PLAN-ISSUE-005")
	}
	// Issue-backed plans use spec_id (the validator's backing-artifact field accepts
	// SPEC-NNN OR ISSUE-NNN); the scaffold must NOT emit the validator-rejected issue_id
	// (ISSUE-009).
	if !strings.Contains(s, "spec_id: ISSUE-005") {
		t.Error("issue-backed plan frontmatter missing spec_id ISSUE-005")
	}
	if strings.Contains(s, "issue_id:") {
		t.Error("issue-backed plan must not contain the validator-rejected issue_id (ISSUE-009)")
	}
}

func TestArtifactNew_Frontmatter_Issue(t *testing.T) {
	content, err := Scaffold("issue", "003", "my-issue", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, field := range []string{"title:", "schema_version:", "issue:"} {
		if !strings.Contains(s, field) {
			t.Errorf("issue frontmatter missing %q", field)
		}
	}
	// Check nested issue block fields
	for _, field := range []string{"id:", "type:", "status:", "created:"} {
		if !strings.Contains(s, field) {
			t.Errorf("issue nested block missing %q", field)
		}
	}
}

func TestArtifactNew_Frontmatter_ADR(t *testing.T) {
	content, err := Scaffold("adr", "0001", "my-adr", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, field := range []string{"title:", "number:", "created:", "status:", "schema_version:", "deciders:", "decisions:"} {
		if !strings.Contains(s, field) {
			t.Errorf("adr frontmatter missing %q", field)
		}
	}
}

func TestArtifactNew_Frontmatter_Directive(t *testing.T) {
	content, err := Scaffold("directive", "001", "my-dir", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, field := range []string{"title:", "number:", "created:", "status:", "schema_version:"} {
		if !strings.Contains(s, field) {
			t.Errorf("directive frontmatter missing %q", field)
		}
	}
}

func TestArtifactNew_Frontmatter_Bundle(t *testing.T) {
	content, err := Scaffold("bundle", "001", "my-bundle", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, field := range []string{"title:", "number:", "created:", "schema_version:", "bundle:"} {
		if !strings.Contains(s, field) {
			t.Errorf("bundle frontmatter missing %q", field)
		}
	}
	// Check nested bundle block
	for _, field := range []string{"name:", "version:", "category:"} {
		if !strings.Contains(s, field) {
			t.Errorf("bundle nested block missing %q", field)
		}
	}
}

func TestArtifactNew_Frontmatter_Capability(t *testing.T) {
	content, err := Scaffold("capability", "001", "my-cap", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, field := range []string{"capability:", "id:", "title:", "status:", "strictness:"} {
		if !strings.Contains(s, field) {
			t.Errorf("capability frontmatter missing %q", field)
		}
	}
}

func TestArtifactNew_Frontmatter_DefaultDate(t *testing.T) {
	content, err := Scaffold("spec", "001", "my-spec", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "2026-04-04") {
		t.Error("date field should default to provided date 2026-04-04")
	}
}

func TestArtifactNew_Frontmatter_DefaultStatusDraft(t *testing.T) {
	for _, artType := range []string{"spec", "adr"} {
		content, err := Scaffold(artType, "001", "test", "2026-04-04", "")
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", artType, err)
		}
		if !strings.Contains(string(content), "status: draft") {
			t.Errorf("%s should default to status: draft", artType)
		}
	}
	// Plan (spec-backed)
	content, err := Scaffold("plan", "001", "test", "2026-04-04", "SPEC-001")
	if err != nil {
		t.Fatalf("unexpected error for plan: %v", err)
	}
	if !strings.Contains(string(content), "status: draft") {
		t.Error("plan should default to status: draft")
	}
}

func TestArtifactNew_Frontmatter_DefaultStatusOpen(t *testing.T) {
	content, err := Scaffold("issue", "001", "test", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(content), "status: open") {
		t.Error("issue should default to status: open")
	}
}

// TestArtifactNew_Bundle_ExploringDefaults proves the fresh bundle stamps a v2-valid
// body: a v2-enum category (the retired v1 `idea` category is gone) and an exploring
// maturity, so it validates against bundle/v2 (ISSUE-032 Defect E / CLM-009).
func TestArtifactNew_Bundle_ExploringDefaults(t *testing.T) {
	content, err := Scaffold("bundle", "001", "test", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "category: feature") {
		t.Error("bundle should default to a v2-enum category (feature)")
	}
	if strings.Contains(s, "category: idea") {
		t.Error("bundle must NOT use the retired v1 category 'idea'")
	}
	if !strings.Contains(s, "maturity: exploring") {
		t.Error("bundle should start at maturity exploring")
	}
}

// --- Body section tests ---

func TestArtifactNew_Sections_Spec(t *testing.T) {
	content, err := Scaffold("spec", "001", "my-spec", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, heading := range []string{"## Overview", "## Requirements", "## Implementation", "## Verification"} {
		if !strings.Contains(s, heading) {
			t.Errorf("spec body missing section %q", heading)
		}
	}
}

func TestArtifactNew_Sections_Plan(t *testing.T) {
	content, err := Scaffold("plan", "001", "my-plan", "2026-04-04", "SPEC-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "phases: []") {
		t.Error("plan body should contain 'phases: []'")
	}
}

func TestArtifactNew_Sections_Issue(t *testing.T) {
	content, err := Scaffold("issue", "001", "my-issue", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "## Problem") {
		t.Error("issue body missing ## Problem section")
	}
}

func TestArtifactNew_Sections_ADR(t *testing.T) {
	content, err := Scaffold("adr", "0001", "my-adr", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, heading := range []string{"## Context", "## Decision", "## Consequences"} {
		if !strings.Contains(s, heading) {
			t.Errorf("adr body missing section %q", heading)
		}
	}
}

func TestArtifactNew_Sections_Directive(t *testing.T) {
	content, err := Scaffold("directive", "001", "my-dir", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, heading := range []string{"## Description"} {
		if !strings.Contains(s, heading) {
			t.Errorf("directive body missing section %q", heading)
		}
	}
}

func TestArtifactNew_Sections_Bundle(t *testing.T) {
	content, err := Scaffold("bundle", "001", "my-bundle", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, heading := range []string{"## Overview", "## Components"} {
		if !strings.Contains(s, heading) {
			t.Errorf("bundle body missing section %q", heading)
		}
	}
}

func TestArtifactNew_Sections_Capability(t *testing.T) {
	content, err := Scaffold("capability", "001", "my-cap", "2026-04-04", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(content)
	for _, field := range []string{"infrastructure_specs:", "quality_gates:"} {
		if !strings.Contains(s, field) {
			t.Errorf("capability body missing field %q", field)
		}
	}
}
