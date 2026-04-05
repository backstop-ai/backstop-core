package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

// TestArtifactValidate_Scope_SpecAll verifies that --spec flag without ID
// validates all spec artifacts. (CLM-009)
func TestArtifactValidate_Scope_SpecAll(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md":          validSpecContent("SPEC-001"),
		"specs/SPEC-002-b.spec.md":          validSpecContent("SPEC-002"),
		"plans/PLAN-SPEC-001-a.plan.yml":    validPlanContent("PLAN-SPEC-001", "SPEC-001"),
		"bundles/cli.bundle.md": `---
title: Test Bundle
schema_version: bundle/v1

bundle:
  name: test
  version: "1.0.0"
  created: "2026-04-01"
  updated: "2026-04-01"
  category: tool
---

# Test Bundle

## Problem Statement

Test.
`,
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Should validate both specs but not the plan or bundle
	for _, v := range result.Violations {
		if !strings.HasSuffix(v.File, ".spec.md") {
			t.Errorf("unexpected violation from non-spec file: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_SpecByID verifies that --spec SPEC-002
// validates only the spec with that ID. (CLM-010)
func TestArtifactValidate_Scope_SpecByID(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
		"specs/SPEC-002-b.spec.md": validSpecContent("SPEC-002"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": "SPEC-002"},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// All violations should come from SPEC-002 only
	for _, v := range result.Violations {
		if !strings.Contains(v.File, "SPEC-002") {
			t.Errorf("expected violations only from SPEC-002, got: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_PlanAll verifies that --plan flag without ID
// validates all plan artifacts. (CLM-011)
func TestArtifactValidate_Scope_PlanAll(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md":       validSpecContent("SPEC-001"),
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"plan": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Violations (if any) should only be from plan files
	for _, v := range result.Violations {
		if !strings.HasSuffix(v.File, ".plan.yml") {
			t.Errorf("unexpected violation from non-plan file: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_ADRAll verifies that --adr flag without ID
// validates all ADR artifacts. (CLM-012)
func TestArtifactValidate_Scope_ADRAll(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"adrs/ADR-0001-test.adr.md": `---
number: ADR-0001
created: "2026-04-01"
status: Accepted
deciders: "@test"
decisions: "D-001"
schema_version: adr/v1
---

# ADR-0001: Test

## Thesis

Test.

## Alternatives

None.

## Decision

Accepted.

## Consequences

None.
`,
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"adr": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	for _, v := range result.Violations {
		if !strings.HasSuffix(v.File, ".adr.md") {
			t.Errorf("unexpected violation from non-adr file: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_BundleAll verifies that --bundle flag without ID
// validates all bundle artifacts. (CLM-013)
func TestArtifactValidate_Scope_BundleAll(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"bundles/cli.bundle.md": `---
title: Test Bundle
schema_version: bundle/v1

bundle:
  name: test
  version: "1.0.0"
  created: "2026-04-01"
  updated: "2026-04-01"
  category: tool
---

# Test Bundle

## Problem Statement

Test.
`,
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"bundle": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	for _, v := range result.Violations {
		if !strings.HasSuffix(v.File, ".bundle.md") {
			t.Errorf("unexpected violation from non-bundle file: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_IssueAll verifies that --issue flag without ID
// validates all issue artifacts. (CLM-014)
func TestArtifactValidate_Scope_IssueAll(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"issues/ISSUE-001-test.issue.md": `---
title: Test Issue
number: ISSUE-001
schema_version: issue/v1
---

# ISSUE-001: Test Issue

## Problem

Test.
`,
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"issue": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	for _, v := range result.Violations {
		if !strings.HasSuffix(v.File, ".issue.md") {
			t.Errorf("unexpected violation from non-issue file: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_StandardAll verifies that --standard flag without ID
// validates all standard artifacts. (CLM-015)
func TestArtifactValidate_Scope_StandardAll(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"standards/SEC-001-test.standard.md": `---
title: Test Standard
number: SEC-001
created: "2026-04-01"
status: active
schema_version: standard/v1
pack: security
scope: language
---

# SEC-001: Test Standard

## Overview

Test.

## Rules

None.
`,
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"standard": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	for _, v := range result.Violations {
		if !strings.HasSuffix(v.File, ".standard.md") {
			t.Errorf("unexpected violation from non-standard file: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_MultipleFlags verifies that combining multiple
// type flags validates artifacts of those types only. (CLM-016)
func TestArtifactValidate_Scope_MultipleFlags(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md":       validSpecContent("SPEC-001"),
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
		"bundles/cli.bundle.md": `---
title: Test Bundle
schema_version: bundle/v1

bundle:
  name: test
  version: "1.0.0"
  created: "2026-04-01"
  updated: "2026-04-01"
  category: tool
---

# Test Bundle

## Problem Statement

Test.
`,
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": "", "plan": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Violations should only come from spec or plan files, not bundles
	for _, v := range result.Violations {
		if !strings.HasSuffix(v.File, ".spec.md") && !strings.HasSuffix(v.File, ".plan.yml") {
			t.Errorf("unexpected violation from non-spec/plan file: %s", v.File)
		}
	}
}

// TestArtifactValidate_Scope_DefaultAll verifies that when no type flags are
// provided, all artifact types are validated. (CLM-017)
func TestArtifactValidate_Scope_DefaultAll(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md":       validSpecContent("SPEC-001"),
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: nil, // No filters = default all
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Both spec and plan should have been validated
	if result.ArtifactsFound != 2 {
		t.Errorf("expected ArtifactsFound=2 (spec + plan), got %d", result.ArtifactsFound)
	}
}

// TestArtifactValidate_AllFlag_ValidatesEverything verifies that --all flag
// validates every artifact across all six types. (CLM-018)
func TestArtifactValidate_AllFlag_ValidatesEverything(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md":       validSpecContent("SPEC-001"),
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		All:         true,
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Should have processed all artifacts (spec and plan)
	if result.ArtifactsFound != 2 {
		t.Errorf("expected ArtifactsFound=2 with --all, got %d", result.ArtifactsFound)
	}
}

// TestArtifactValidate_AllFlag_PrecedesTypeFlags verifies that --all takes
// precedence over type-scoping flags. (CLM-019)
func TestArtifactValidate_AllFlag_PrecedesTypeFlags(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md":       validSpecContent("SPEC-001"),
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	// --all with --spec should still validate everything
	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": ""},
		All:         true,
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// --all overrides --spec, so both spec and plan should be processed
	if result.ArtifactsFound != 2 {
		t.Errorf("expected ArtifactsFound=2 (--all overrides --spec), got %d", result.ArtifactsFound)
	}
}

// TestArtifactValidate_IDScope_SpecMatchesByNumber verifies that --spec SPEC-002
// matches by the number metadata field, not by filename. (CLM-040)
func TestArtifactValidate_IDScope_SpecMatchesByNumber(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
		"specs/SPEC-002-b.spec.md": validSpecContent("SPEC-002"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": "SPEC-002"},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Only SPEC-002 violations should be present
	for _, v := range result.Violations {
		if strings.Contains(v.File, "SPEC-001") {
			t.Errorf("should not validate SPEC-001 when scoped to SPEC-002: %s", v.File)
		}
	}
}

// TestArtifactValidate_IDScope_PlanMatchesByPlanID verifies that
// --plan PLAN-SPEC-001 matches by the plan_id metadata field, not number. (CLM-051)
func TestArtifactValidate_IDScope_PlanMatchesByPlanID(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
		"plans/PLAN-SPEC-002-b.plan.yml": validPlanContent("PLAN-SPEC-002", "SPEC-002"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"plan": "PLAN-SPEC-001"},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Only PLAN-SPEC-001 violations should be present
	for _, v := range result.Violations {
		if strings.Contains(v.File, "PLAN-SPEC-002") {
			t.Errorf("should not validate PLAN-SPEC-002 when scoped to PLAN-SPEC-001: %s", v.File)
		}
	}
}

// TestArtifactValidate_IDScope_NotFound_Exit2 verifies that scoping by a
// nonexistent artifact ID produces a config error (exit 2). (CLM-041)
func TestArtifactValidate_IDScope_NotFound_Exit2(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": "SPEC-999"},
		SchemaFS:    SchemaFS,
	}
	_, err := ValidateArtifacts(cfg)
	if err == nil {
		t.Fatal("expected config error for nonexistent artifact ID, got nil")
	}
	if !strings.Contains(err.Error(), "SPEC-999") {
		t.Errorf("error should mention the missing ID: %v", err)
	}
}

// TestArtifactValidate_ThinAdapter_CallsPkgValidate verifies that the command
// calls pkg/validate functions and does not reimplement validation logic. (CLM-045)
func TestArtifactValidate_ThinAdapter_CallsPkgValidate(t *testing.T) {
	// Use an invalid spec that pkg/validate will flag with violations.
	invalidSpec := `---
title: "SPEC-099: Missing Sections"
number: SPEC-099
created: "2026-04-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Missing sections.
  package: pkg/test

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 90
---

# SPEC-099: Missing Sections

## Overview

This spec is missing required sections.
`
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-099-missing.spec.md": invalidSpec,
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Also run pkg/validate directly on the same artifact to compare
	art, parseErr := artifact.ParseFile(filepath.Join(dir, "specs", "SPEC-099-missing.spec.md"))
	if parseErr != nil {
		t.Fatalf("ParseFile: %v", parseErr)
	}
	schemaPath, _ := schema.ResolveSchemaPath(art)
	sch, _ := loadSchemaFromFS(SchemaFS, schemaPath)
	directResult := validate.Spec(art, sch)

	// The command's violations should match pkg/validate's violations
	if len(result.Violations) != len(directResult.Violations) {
		t.Errorf("command produced %d violations, pkg/validate produced %d — command may not be using pkg/validate",
			len(result.Violations), len(directResult.Violations))
	}
	for i, v := range result.Violations {
		if i < len(directResult.Violations) {
			if v.Rule != directResult.Violations[i].Rule {
				t.Errorf("violation[%d] rule mismatch: command=%q, pkg/validate=%q", i, v.Rule, directResult.Violations[i].Rule)
			}
		}
	}
}

// TestArtifactValidate_BackstopYml_LoadedAsPrerequisite verifies that
// backstop.yml is loaded before artifact validation begins. (CLM-046)
func TestArtifactValidate_BackstopYml_LoadedAsPrerequisite(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	// Run the command (which triggers config loading) from a dir with backstop.yml
	_, _, exitCode := runValidateCommand(t, dir)
	if exitCode == ExitConfigError {
		t.Error("command should succeed when backstop.yml is present")
	}
}

// TestArtifactValidate_BackstopYml_Missing_Exit2 verifies that a missing
// backstop.yml produces exit code 2. (CLM-047)
func TestArtifactValidate_BackstopYml_Missing_Exit2(t *testing.T) {
	dir := t.TempDir() // No backstop.yml

	_, _, exitCode := runValidateCommand(t, dir)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d for missing backstop.yml, got %d", ExitConfigError, exitCode)
	}
}

// TestArtifactValidate_BackstopYml_Invalid_Exit2 verifies that an invalid
// backstop.yml produces exit code 2. (CLM-048)
func TestArtifactValidate_BackstopYml_Invalid_Exit2(t *testing.T) {
	dir := setupArtifactTestDir(t, "invalid: [yaml: {broken", nil)

	_, _, exitCode := runValidateCommand(t, dir)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d for invalid backstop.yml, got %d", ExitConfigError, exitCode)
	}
}

// testSchemaFS returns the embedded schema filesystem for use in tests.
// This ensures tests use the same schemas as the production command.
func testSchemaFS() fs.FS {
	return SchemaFS
}
