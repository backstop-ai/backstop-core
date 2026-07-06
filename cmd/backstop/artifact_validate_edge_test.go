package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// TestArtifactValidate_ZeroArtifacts_Exit0_EmptyViolations verifies that
// zero discovered artifacts returns exit code 0 with empty violations. (CLM-049)
func TestArtifactValidate_ZeroArtifacts_Exit0_EmptyViolations(t *testing.T) {
	// Directory with backstop.yml but no artifact files
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"README.md":   "# Project",
		"src/main.go": "package main",
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		SchemaFS:    SchemaFS,
	}
	result, err := ValidateArtifacts(cfg)
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	if !result.Pass {
		t.Error("expected pass=true for zero artifacts")
	}
	if result.ViolationsCount != 0 {
		t.Errorf("expected 0 violations, got %d", result.ViolationsCount)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected empty violations list, got %d items", len(result.Violations))
	}

	// Exit code should be 0
	exitCode := ExitWithResult(validate.ValidationResult{Violations: result.Violations}, nil)
	if exitCode != ExitPass {
		t.Errorf("expected exit code %d, got %d", ExitPass, exitCode)
	}
}

// TestArtifactValidate_ZeroArtifacts_WarningMessage verifies that zero
// discovered artifacts emits a warning message to stderr. (CLM-050)
func TestArtifactValidate_ZeroArtifacts_WarningMessage(t *testing.T) {
	// Directory with backstop.yml but no artifact files
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, nil)

	_, stderr, exitCode := runValidateCommand(t, dir)

	if exitCode != ExitPass {
		t.Errorf("expected exit code %d for zero artifacts, got %d", ExitPass, exitCode)
	}

	if !strings.Contains(stderr, "warning") || !strings.Contains(stderr, "no artifacts") {
		t.Errorf("expected warning about no artifacts in stderr, got: %q", stderr)
	}
}

// TestArtifactValidate_Discover_ParseFailure_ConfigError verifies that a file
// matching an artifact pattern but failing to parse produces a config error
// (exit 2), not a validation error. (CLM-039)
func TestArtifactValidate_Discover_ParseFailure_ConfigError(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/broken.spec.md": "---\nthis is: [invalid yaml\n  because: {it has\n---\n\n# Broken\n",
	})

	cfg := ValidateConfig{
		ProjectRoot: dir,
		SchemaFS:    SchemaFS,
	}
	_, err := ValidateArtifacts(cfg)
	if err == nil {
		t.Fatal("expected config error for unparseable artifact, got nil")
	}

	// Error should mention the parse failure
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected error to mention 'parse', got: %v", err)
	}

	// This should be a config error (exit 2), not validation (exit 1)
	exitCode := ExitWithResult(validate.ValidationResult{}, err)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitCode)
	}
}

// TestArtifactValidate_Command_JSONConfigError exercises the JSON config error
// path through the Cobra command. Ensures exit code 2 with JSON output.
func TestArtifactValidate_Command_JSONConfigError(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/broken.spec.md": "---\nthis is: [invalid yaml\n---\n",
	})

	stdout, _, exitCode := runValidateCommand(t, dir, "--json")
	if exitCode != ExitConfigError {
		t.Errorf("expected exit %d, got %d", ExitConfigError, exitCode)
	}
	// JSON output should still be valid JSON with the error
	if stdout != "" {
		// If there's JSON output, it should be parseable
		var discard map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &discard); err != nil {
			t.Errorf("JSON config error output should be valid JSON: %v", err)
		}
	}
}

// TestArtifactValidate_Command_WithSpecIDFlag exercises the --spec ID path
// through the Cobra command.
func TestArtifactValidate_Command_WithSpecIDFlag(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md": validSpecContent("SPEC-001"),
	})

	_, _, exitCode := runValidateCommand(t, dir, "--spec", "SPEC-001")
	// Should succeed (or have violations, but not config error)
	if exitCode == ExitConfigError {
		t.Errorf("unexpected config error for valid --spec ID")
	}
}

// TestArtifactValidate_Command_WithAllFlag exercises the --all path.
func TestArtifactValidate_Command_WithAllFlag(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	_, _, exitCode := runValidateCommand(t, dir, "--all")
	if exitCode == ExitConfigError {
		t.Errorf("unexpected config error with --all flag")
	}
}

// TestArtifactValidate_Command_HumanViolations exercises human output with
// violations through the Cobra command.
func TestArtifactValidate_Command_HumanViolations(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
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

	stdout, stderr, exitCode := runValidateCommand(t, dir)
	if exitCode != ExitViolations {
		t.Errorf("expected exit %d, got %d", ExitViolations, exitCode)
	}
	_ = stdout
	_ = stderr
}

// TestArtifactValidate_Command_HumanConfigError exercises human-mode config
// error through the Cobra command.
func TestArtifactValidate_Command_HumanConfigError(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/broken.spec.md": "---\nthis is: [invalid yaml\n---\n",
	})

	_, stderr, exitCode := runValidateCommand(t, dir)
	if exitCode != ExitConfigError {
		t.Errorf("expected exit %d, got %d", ExitConfigError, exitCode)
	}
	if !strings.Contains(stderr, "Error:") {
		t.Errorf("expected error message in stderr, got: %q", stderr)
	}
}

// TestArtifactValidate_ExitCodeError_StringMethod covers ExitCodeError.Error().
func TestArtifactValidate_ExitCodeError_StringMethod(t *testing.T) {
	e := &ExitCodeError{Code: ExitViolations, Message: "test error"}
	if got := e.Error(); got != "test error" {
		t.Errorf("Error() = %q, want %q", got, "test error")
	}
}

// TestArtifactValidate_Command_MultipleTypeFlags exercises multiple type flags.
func TestArtifactValidate_Command_MultipleTypeFlags(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-a.spec.md":       validSpecContent("SPEC-001"),
		"plans/PLAN-SPEC-001-a.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	// Pass empty string as flag values to exercise the Changed path
	_, _, exitCode := runValidateCommand(t, dir, "--spec", "", "--plan", "")
	if exitCode == ExitConfigError {
		t.Errorf("unexpected config error for multiple type flags")
	}
}
