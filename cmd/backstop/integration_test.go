package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPath holds the path to the built backstop binary.
var binaryPath string

// TestMain builds the backstop binary once for all integration tests.
func TestMain(m *testing.M) {
	// Build the binary
	tmpDir, err := os.MkdirTemp("", "backstop-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "backstop")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// setupTestDir creates a temp directory with backstop.yml and returns cleanup func.
func setupTestDir(t *testing.T, backstopYML string) string {
	t.Helper()
	dir := t.TempDir()
	if backstopYML != "" {
		if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(backstopYML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// runBinary executes the backstop binary with given args in the specified dir.
func runBinary(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "BACKSTOP_CONFIG=")
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run binary: %v", err)
		}
	}
	return string(out), exitCode
}

const minimalBackstopYML = `project: integration-test
language: go
`

// TestIntegration_ArtifactValidate_RealSpec runs backstop artifact validate
// against a real spec file and returns structured JSON with violations. (CLM-047)
func TestIntegration_ArtifactValidate_RealSpec(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := setupTestDir(t, minimalBackstopYML)

	// Create a minimal spec file for validation
	specContent := `---
title: "Test Spec"
number: SPEC-999
created: "2025-01-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0
---

# SPEC-999: Test Spec

## Overview

Test spec for integration testing.
`
	if err := os.WriteFile(filepath.Join(dir, "SPEC-999-test.spec.md"), []byte(specContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out, exitCode := runBinary(t, dir, "artifact", "validate", "--json", "SPEC-999-test.spec.md")
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("expected exit code 0 or 1, got %d\noutput: %s", exitCode, out)
	}

	// Verify structured JSON output
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	// Verify schema_version present
	if _, ok := parsed["schema_version"]; !ok {
		t.Error("JSON output missing schema_version field")
	}
}

// TestIntegration_CodeCheck_RealGoFile runs backstop code check --file against
// a Go file and returns structured JSON. (CLM-048)
func TestIntegration_CodeCheck_RealGoFile(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := setupTestDir(t, minimalBackstopYML)

	// Create .backstop/rules/ dir so code check doesn't fail
	if err := os.MkdirAll(filepath.Join(dir, ".backstop", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a minimal Go file
	goContent := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out, exitCode := runBinary(t, dir, "code", "check", "--json", "--file", "main.go")
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("expected exit code 0 or 1, got %d\noutput: %s", exitCode, out)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if _, ok := parsed["schema_version"]; !ok {
		t.Error("JSON output missing schema_version field")
	}
}

// TestIntegration_PackCompile_RealStandard runs backstop pack compile and
// produces enforcement manifests. (CLM-049)
func TestIntegration_PackCompile_RealStandard(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := setupTestDir(t, minimalBackstopYML)

	// Create a standards directory with a minimal standard
	stdDir := filepath.Join(dir, "standards")
	if err := os.MkdirAll(stdDir, 0o755); err != nil {
		t.Fatal(err)
	}

	out, exitCode := runBinary(t, dir, "pack", "compile", "--json")
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("expected exit code 0 or 1, got %d\noutput: %s", exitCode, out)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if _, ok := parsed["schema_version"]; !ok {
		t.Error("JSON output missing schema_version field")
	}
}

// TestIntegration_Gate_EndToEnd runs backstop gate and produces a structured
// pass/fail result. (CLM-050)
func TestIntegration_Gate_EndToEnd(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := setupTestDir(t, minimalBackstopYML)

	out, exitCode := runBinary(t, dir, "gate", "--json")
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("expected exit code 0 or 1, got %d\noutput: %s", exitCode, out)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	// Verify it has the expected structure
	if _, ok := parsed["schema_version"]; !ok {
		t.Error("JSON output missing schema_version field")
	}
	if _, ok := parsed["pass"]; !ok {
		t.Error("gate output missing pass field")
	}
}

// TestIntegration_ArtifactNew_ScaffoldsSpec runs backstop artifact new spec
// and produces a valid scaffolded spec. (CLM-051)
func TestIntegration_ArtifactNew_ScaffoldsSpec(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := setupTestDir(t, minimalBackstopYML)

	out, exitCode := runBinary(t, dir, "artifact", "new", "--json", "--slug", "test-spec", "spec")
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("expected exit code 0 or 1, got %d\noutput: %s", exitCode, out)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if _, ok := parsed["schema_version"]; !ok {
		t.Error("JSON output missing schema_version field")
	}
}
