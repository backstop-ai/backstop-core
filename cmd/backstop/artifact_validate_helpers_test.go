package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// setupArtifactTestDir creates a temp directory with a backstop.yml and optional
// artifact files. Returns the temp directory path. The directory is automatically
// cleaned up when the test completes.
func setupArtifactTestDir(t *testing.T, backstopYML string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	if backstopYML != "" {
		err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(backstopYML), 0o644)
		if err != nil {
			t.Fatalf("write backstop.yml: %v", err)
		}
	}

	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", relPath, err)
		}
	}

	return dir
}

// runValidateCommand executes the artifact validate command with the given args
// in the given directory. Returns stdout, stderr, and the exit code.
func runValidateCommand(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	root := NewRootCommand()
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)

	// Prepend "artifact validate" to args
	fullArgs := append([]string{"artifact", "validate"}, args...)
	root.SetArgs(fullArgs)

	// Change to the test directory so config discovery finds backstop.yml
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("chdir back: %v", err)
		}
	}()

	execErr := root.Execute()

	exitCode = ExitPass
	if execErr != nil {
		// Check for ExitCodeError with specific exit code
		if exitErr, ok := execErr.(*ExitCodeError); ok {
			exitCode = exitErr.Code
		} else {
			exitCode = ExitConfigError
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// copyFixtureDir copies all files from srcDir into destDir recursively.
func copyFixtureDir(t *testing.T, srcDir, destDir string) {
	t.Helper()
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture dir: %v", err)
	}
}

// artifactTestBackstopYML is the minimal backstop.yml content for artifact validation testing.
const artifactTestBackstopYML = "project: test-project\nlanguage: go\n"

// validSpecContent returns a valid spec fixture with the given number.
func validSpecContent(number string) string {
	return `---
title: "` + number + `: Test Spec"
number: ` + number + `
created: "2026-04-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: A test spec.
  package: pkg/test

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 90
---

# ` + number + `: Test Spec

## Overview

Test spec overview.

## Requirements

Requirements are defined in frontmatter.

## Implementation

Implementation details.

## Verification

Verification details.
`
}

// validPlanContent returns a valid plan fixture with the given plan_id.
func validPlanContent(planID, specID string) string {
	return `---
plan_id: ` + planID + `
spec_id: ` + specID + `
spec_version: 1.0.0
schema_version: plan/v1
created: "2026-04-01"
status: draft
target_repo: test-repo
target_module: github.com/test/repo
test_command: "go test ./..."
coverage_threshold: 90

notes: |
  Test plan.

phases:
  - id: phase-1
    name: "Phase 1: Implementation"
    tasks:
      - id: TASK-001
        type: test
        title: "Write tests"
        description: |
          Write tests for the feature.
        files:
          - pkg/test/test_test.go
        claims:
          - CLM-001
        depends_on: []

      - id: TASK-002
        type: implementation
        title: "Implement feature"
        description: |
          Implement the feature.
        files:
          - pkg/test/test.go
        claims:
          - CLM-001
        depends_on:
          - TASK-001

      - id: TASK-003
        type: verification
        title: "Verify implementation"
        description: |
          Verify that implementation is correct.
        files:
          - pkg/test/test.go
        claims:
          - CLM-001
        depends_on:
          - TASK-002
`
}
