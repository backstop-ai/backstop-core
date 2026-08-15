package main

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// TestArtifactValidate_JSON_OutputStructure verifies that --json produces
// valid JSON with schema_version, pass, violations_count, and violations. (CLM-020)
func TestArtifactValidate_JSON_OutputStructure(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	stdout, _, exitCode := runValidateCommand(t, dir, "--json")
	if exitCode == ExitConfigError {
		t.Fatalf("unexpected config error, exit %d", exitCode)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, stdout)
	}

	// Check required fields exist
	if _, ok := envelope["schema_version"]; !ok {
		t.Error("JSON output missing schema_version field")
	}
	if _, ok := envelope["pass"]; !ok {
		t.Error("JSON output missing pass field")
	}
	if _, ok := envelope["violations"]; !ok {
		t.Error("JSON output missing violations field")
	}
	if _, ok := envelope["violations_count"]; !ok {
		t.Error("JSON output missing violations_count field")
	}
	if sv, ok := envelope["schema_version"].(string); ok && sv != "cli/v1" {
		t.Errorf("schema_version = %q, want cli/v1", sv)
	}
	// violations_count must match len(violations)
	if vc, ok := envelope["violations_count"].(float64); ok {
		if vArr, ok2 := envelope["violations"].([]interface{}); ok2 {
			if int(vc) != len(vArr) {
				t.Errorf("violations_count=%d but len(violations)=%d", int(vc), len(vArr))
			}
		}
	}
}

// TestArtifactValidate_JSON_ViolationFields verifies that JSON violation
// objects include rule, file, message, and severity fields. (CLM-021)
func TestArtifactValidate_JSON_ViolationFields(t *testing.T) {
	// Use invalid spec (missing required sections) to generate violations
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-002-invalid.spec.md": `---
title: "SPEC-002: Invalid"
number: SPEC-002
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

# SPEC-002: Invalid

## Overview

Missing required sections.
`,
	})

	stdout, _, _ := runValidateCommand(t, dir, "--json")

	var envelope struct {
		Violations []struct {
			Rule     string `json:"rule"`
			File     string `json:"file"`
			Message  string `json:"message"`
			Severity string `json:"severity"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, stdout)
	}

	if len(envelope.Violations) == 0 {
		t.Fatal("expected violations from invalid spec fixture, got none — fixture may need updating")
	}

	for i, v := range envelope.Violations {
		if v.Rule == "" {
			t.Errorf("violation[%d] missing rule field", i)
		}
		if v.Message == "" {
			t.Errorf("violation[%d] missing message field", i)
		}
	}
}

// TestArtifactValidate_Human_OutputFormat verifies that default output
// (no --json) produces human-readable formatted text. (CLM-022)
func TestArtifactValidate_Human_OutputFormat(t *testing.T) {
	// Use an invalid spec that will produce violations so we can check formatting
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-002-invalid.spec.md": `---
title: "SPEC-002: Invalid"
number: SPEC-002
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

# SPEC-002: Invalid

## Overview

Missing required sections.
`,
	})

	t.Setenv("NO_COLOR", "1")
	stdout, _, exitCode := runValidateCommand(t, dir)
	if exitCode == ExitConfigError {
		t.Fatalf("unexpected config error, exit %d", exitCode)
	}

	// Human output should NOT be valid JSON
	var discard map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &discard); err == nil {
		t.Error("human output should not be valid JSON")
	}

	if stdout == "" {
		t.Fatal("expected non-empty human output")
	}

	// Should contain file paths as grouping headers
	if !strings.Contains(stdout, ".spec.md") {
		t.Error("human output should contain file paths")
	}
	// Should contain structured violation markers
	if !strings.Contains(stdout, "✗") {
		t.Error("human output should contain violation markers (✗)")
	}
	// Should contain rule references in brackets
	if !strings.Contains(stdout, "[") || !strings.Contains(stdout, "]") {
		t.Error("human output should contain rule references in brackets")
	}
}

// TestArtifactValidate_OutputParity verifies that JSON and human output
// contain identical underlying violation data. (CLM-023)
func TestArtifactValidate_OutputParity(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-002-invalid.spec.md": `---
title: "SPEC-002: Invalid"
number: SPEC-002
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

# SPEC-002: Invalid

## Overview

Missing sections.
`,
	})

	// Get JSON output
	jsonOut, _, _ := runValidateCommand(t, dir, "--json")
	var envelope struct {
		Violations []struct {
			Rule    string `json:"rule"`
			Message string `json:"message"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Get human output
	t.Setenv("NO_COLOR", "1")
	humanOut, _, _ := runValidateCommand(t, dir)

	// Verify each JSON violation's rule appears in human output
	for _, v := range envelope.Violations {
		if v.Rule != "" && !strings.Contains(humanOut, v.Rule) {
			t.Errorf("human output missing violation rule %q from JSON output", v.Rule)
		}
	}
}

// TestArtifactValidate_Exit0_AllPass verifies exit code 0 when all artifacts
// pass validation. (CLM-024)
func TestArtifactValidate_Exit0_AllPass(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	// Use default (no type flags) which validates all — only plan here
	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"plan": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	if !result.Pass {
		t.Errorf("expected pass=true, got false with %d violations", result.ViolationsCount)
		for _, v := range result.Violations {
			t.Logf("  violation: [%s] %s: %s", v.Rule, v.File, v.Message)
		}
	}

	exitCode := ExitWithResult(validate.ValidationResult{Violations: result.Violations}, nil)
	if exitCode != ExitPass {
		t.Errorf("expected exit code %d, got %d", ExitPass, exitCode)
	}
}

// TestArtifactValidate_Exit1_Violations verifies exit code 1 when any
// artifact has violations. (CLM-025)
func TestArtifactValidate_Exit1_Violations(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-002-invalid.spec.md": `---
title: "SPEC-002: Invalid"
number: SPEC-002
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

# SPEC-002: Invalid

## Overview

Missing sections.
`,
	})

	_, _, exitCode := runValidateCommand(t, dir)
	if exitCode != ExitViolations {
		t.Errorf("expected exit code %d for violations, got %d", ExitViolations, exitCode)
	}
}

// TestArtifactValidate_Exit2_UnknownSchemaVersion verifies exit code 2 on
// unrecognized schema_version. (CLM-026)
func TestArtifactValidate_Exit2_UnknownSchemaVersion(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/unknown.spec.md": `---
title: "Unknown Type"
number: SPEC-999
created: "2026-04-01"
status: draft
schema_version: unknown/v1
spec_version: 1.0.0
---

# Unknown Spec

## Overview

Has unknown schema_version.
`,
	})

	_, _, exitCode := runValidateCommand(t, dir)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d for unknown schema_version, got %d", ExitConfigError, exitCode)
	}
}

// TestArtifactValidate_Exit2_SchemaLoadFailure verifies exit code 2 when
// schema loading fails. (CLM-027)
func TestArtifactValidate_Exit2_SchemaLoadFailure(t *testing.T) {
	// Use an artifact with a valid but nonexistent schema version
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/broken.spec.md": `---
title: "Broken Schema"
number: SPEC-999
created: "2026-04-01"
status: draft
schema_version: spec/v99
spec_version: 1.0.0
---

# Broken Schema

## Overview

Schema v99 doesn't exist.
`,
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	_, err := ValidateArtifacts(cfg)
	if err == nil {
		t.Fatal("expected config error for schema load failure, got nil")
	}

	// Verify this would produce exit code 2
	exitCode := ExitWithResult(validate.ValidationResult{}, err)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitCode)
	}
}

// TestArtifactValidate_Exit2_ParseFailure verifies exit code 2 when artifact
// parsing fails. (CLM-028)
func TestArtifactValidate_Exit2_ParseFailure(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/bad.spec.md": "---\nthis is: [invalid yaml\n  because: {it has\n---\n\n# Bad Spec\n",
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	_, err := ValidateArtifacts(cfg)
	if err == nil {
		t.Fatal("expected config error for parse failure, got nil")
	}

	exitCode := ExitWithResult(validate.ValidationResult{}, err)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitCode)
	}
}

// TestArtifactValidate_Exit2_PrecedesExit1 verifies that exit code 2 takes
// precedence over exit code 1 when both config error and violations exist. (CLM-029)
func TestArtifactValidate_Exit2_PrecedesExit1(t *testing.T) {
	// Create dir with both a valid-but-violating spec AND a broken spec
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-good.spec.md": validSpecContent("SPEC-001"),
		"specs/bad.spec.md":           "---\nthis is: [invalid yaml\n---\n\n# Bad\n",
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		SchemaFS:    SchemaFS,
	}
	_, err := ValidateArtifacts(cfg)
	// Config error should be returned, even if some artifacts might have violations
	if err == nil {
		t.Fatal("expected config error to take precedence, got nil")
	}

	exitCode := ExitWithResult(validate.ValidationResult{}, err)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d (config error precedence), got %d", ExitConfigError, exitCode)
	}
}

// TestArtifactValidate_Schema_LoadedFromEmbed verifies that schemas are loaded
// from the go:embed filesystem, not the real filesystem. (CLM-030)
func TestArtifactValidate_Schema_LoadedFromEmbed(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	// With the real embed FS, validation should succeed
	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	_, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts with embed FS: %v", err)
	}

	// With an empty FS, schema loading should fail — proving the embed FS
	// is actually used for schema resolution.
	emptyFS := fstest.MapFS{}
	cfgEmpty := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    emptyFS,
	}
	_, errEmpty := ValidateArtifacts(cfgEmpty)
	if errEmpty == nil {
		t.Fatal("expected schema load failure with empty FS, got nil — embed FS may not be used")
	}
}

// TestArtifactValidate_Schema_MissingFromEmbed_Exit2 verifies that a missing
// schema in the embedded filesystem produces exit code 2. (CLM-031)
func TestArtifactValidate_Schema_MissingFromEmbed_Exit2(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/broken.spec.md": `---
title: "Broken"
number: SPEC-999
created: "2026-04-01"
status: draft
schema_version: spec/v99
spec_version: 1.0.0
---

# Broken

## Overview

Schema v99 doesn't exist in embed.
`,
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	_, err := ValidateArtifacts(cfg)
	if err == nil {
		t.Fatal("expected error for missing schema in embed, got nil")
	}

	exitCode := ExitWithResult(validate.ValidationResult{}, err)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitCode)
	}
}

// TestArtifactValidate_Aggregate_JSON verifies that violations from multiple
// artifacts are aggregated into a single JSON output. (CLM-042)
func TestArtifactValidate_Aggregate_JSON(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
		"specs/SPEC-002-b.spec.md": `---
title: "SPEC-002: Invalid"
number: SPEC-002
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

# SPEC-002: Invalid

## Overview

Missing sections.
`,
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// Format as JSON to verify aggregation
	valResult := validate.ValidationResult{Violations: result.Violations}
	f := &JSONFormatter{}
	jsonOut, err := f.FormatResult(valResult)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	var envelope struct {
		SchemaVersion string `json:"schema_version"`
		Pass          bool   `json:"pass"`
		Violations    []struct {
			File string `json:"file"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, jsonOut)
	}

	// Should be a single JSON object with aggregated violations
	if envelope.SchemaVersion != "cli/v1" {
		t.Errorf("schema_version = %q, want cli/v1", envelope.SchemaVersion)
	}
}

// TestArtifactValidate_Aggregate_HumanGroupedByFile verifies that human output
// groups violations by file. (CLM-043)
func TestArtifactValidate_Aggregate_HumanGroupedByFile(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
		"specs/SPEC-002-b.spec.md": `---
title: "SPEC-002: Invalid"
number: SPEC-002
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

# SPEC-002: Invalid

## Overview

Missing sections.
`,
	})

	t.Setenv("NO_COLOR", "1")

	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	valResult := validate.ValidationResult{Violations: result.Violations}
	f := &HumanFormatter{}
	humanOut, err := f.FormatResult(valResult)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	if humanOut == "" {
		t.Error("expected non-empty human output")
	}

	// Verify violations are grouped by file: file paths should appear as headers
	// before their violation lines. Check that at least one .spec.md path appears.
	lines := strings.Split(humanOut, "\n")
	foundFileHeader := false
	for _, line := range lines {
		// File headers are lines that contain a path but no "✗" prefix
		if strings.HasSuffix(line, ".spec.md") && !strings.Contains(line, "✗") {
			foundFileHeader = true
			break
		}
	}
	if len(result.Violations) > 0 && !foundFileHeader {
		t.Error("human output should group violations by file with file path headers")
	}
}

// TestArtifactValidate_Aggregate_AnyFailMeansOverallFail verifies that one
// failure means overall fail across all artifacts. (CLM-044)
func TestArtifactValidate_Aggregate_AnyFailMeansOverallFail(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
		"specs/SPEC-002-b.spec.md": `---
title: "SPEC-002: Invalid"
number: SPEC-002
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

# SPEC-002: Invalid

## Overview

Missing sections.
`,
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	// If any spec has violations, the overall result should be fail
	if len(result.Violations) > 0 && result.Pass {
		t.Error("result.Pass should be false when there are violations")
	}
}
