package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// binaryPath holds the path to the built backstop binary.
var binaryPath string

// TestMain builds the backstop binary once for all integration tests.
func TestMain(m *testing.M) {
	// FIRST STATEMENT, before the temp dir and therefore before the `go build`
	// below. Under `go test` the Linux sandbox trampoline re-execs THIS TEST
	// BINARY with its working directory set to the pack directory, so without this
	// gate the unconditional build runs in a directory holding no .go files and
	// dies with "failed to build binary: ... no Go files in <dir>" before the
	// helper is ever recognised — the pack's real command never runs. That is
	// ISSUE-163.
	//
	// When this process is not a helper the call returns nil immediately, having
	// done nothing (on darwin it is sandbox_nonlinux.go's unconditional stub). When
	// it IS one, a successful call NEVER RETURNS — the helper execs the pack's
	// command. A non-nil error means this binary IS a helper whose sandbox failed
	// to install, so running the suite at that point would hand the parent the
	// suite's output as the sandboxed command's output.
	//
	// The other two members of this wiring family: pkg/packval/main_test.go's
	// TestMain (packval's own test binary) and cmd/backstop/main.go's runWith (the
	// shipped binary).
	if err := packval.MaybeRunSandboxHelper(); err != nil {
		var completion interface{ ExitCode() int }
		if errors.As(err, &completion) {
			os.Exit(completion.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "backstop sandbox helper: %v\n", err)
		os.Exit(sandboxHelperExitCode)
	}

	// Build the binary
	tmpDir, err := os.MkdirTemp("", "backstop-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	binaryPath = filepath.Join(tmpDir, "backstop")
	// execCommand is the package's parametric dispatch (root_test.go): the tool
	// name reaches exec.Command as a variable, which is what this harness needs
	// — it compiles the binary under test, so naming the toolchain here is the
	// subject of the test rather than routing baked into the shipped binary.
	cmd := execCommand("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	// os.Exit skips deferred calls, so the temp dir is removed explicitly after
	// the run rather than by a defer that would never fire.
	code := m.Run()
	if err := os.RemoveAll(tmpDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove temp dir %s: %v\n", tmpDir, err)
	}
	os.Exit(code)
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

// TestIntegration_CodeCheck_RealGoFile was removed by ISSUE-018: it ran the
// deleted `backstop code check` command. The gate integration test
// (TestIntegration_Gate_EndToEnd) covers the surviving enforcement surface.

// pack compile removed — standards live inside packs now (DD-18/DD-45).

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

func TestBaselineCIContract_GenerationUsesAllScope(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflowPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(workflow)
	if !strings.Contains(text, "branches: [main]") {
		t.Fatalf("ci workflow must target main branch baseline generation")
	}
	if !strings.Contains(text, "backstop gate --all") {
		t.Fatalf("ci workflow must run full-scope gate generation equivalent to --all")
	}
}

func TestBaselineCIContract_GenerationAndPublicationAreSeparable(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflowPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(workflow)
	generateIndex := strings.Index(text, "Generate baseline")
	publishIndex := strings.Index(text, "Publish baseline")
	if generateIndex < 0 {
		t.Fatalf("workflow must define explicit baseline generation step")
	}
	if publishIndex < 0 {
		t.Fatalf("workflow must define explicit baseline publication step")
	}
	if publishIndex <= generateIndex {
		t.Fatalf("baseline publication must be ordered after generation")
	}
	if !strings.Contains(text, "actions/upload-artifact") {
		t.Fatalf("workflow must publish baseline through GitHub Actions artifact upload")
	}
	if !strings.Contains(text, ".backstop/baseline.json") {
		t.Fatalf("workflow publication must target .backstop/baseline.json")
	}
}

func TestBaselineCIContract_ArtifactNamingAndLatestMainSelectionSemantics(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflowPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(workflow)
	if !strings.Contains(text, "baseline") {
		t.Fatalf("workflow must include baseline artifact naming semantics")
	}
	if !strings.Contains(strings.ToLower(text), "latest successful") {
		t.Fatalf("workflow contract docs must mention latest successful main run semantics")
	}
}

func TestBaselineCIContract_RuleSetChangeExceptionOnlySeedsOnFullBaseline(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflowPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(workflow)
	if !strings.Contains(text, "Generate baseline") {
		t.Fatalf("workflow must define Generate baseline step for rule-set-change seeding")
	}
	if !strings.Contains(text, "./backstop gate --all --json") {
		t.Fatalf("baseline generation must use full-scope --all gate run")
	}
	if strings.Contains(text, "Generate baseline\n        run: ./backstop gate --json") {
		t.Fatalf("baseline generation cannot rely on scoped default mode for seeding semantics")
	}
}

func TestBaselineCIContract_PullRequestGateKeepsChangedCodeEnforcement(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflowPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(workflow)
	if !strings.Contains(text, "pull_request:") {
		t.Fatalf("workflow must include pull_request trigger for changed-code enforcement")
	}
	if !strings.Contains(text, "branches: [main]") {
		t.Fatalf("pull request gate must target main for immediate regression enforcement")
	}
	// The verification checks are now reached THROUGH the gate rather than run
	// beside it (PLAN-ISSUE-020 TASK-016). This assertion previously required the
	// literal `go test -race -coverprofile=cover.out -covermode=atomic ./...`,
	// which the flip deleted: ISSUE-020's premise is that the Linux sandbox defect
	// stayed invisible because this repo's CI never called its own product, so
	// running the underlying tools directly is the gap rather than a weaker version
	// of the same check.
	//
	// The test's INTENT is unchanged — the pull-request gate must execute
	// verification that fails changed-code regressions — and `backstop gate --base`
	// is what executes it. The `--base` half is not decoration: a CI checkout is
	// pristine, so bare diff mode resolves merge-base HEAD origin/main to HEAD on a
	// push and finds nothing to check, which would enforce nothing at all.
	if !strings.Contains(text, "backstop gate --base") {
		t.Fatalf("pull request gate must execute verification checks that fail changed-code regressions, " +
			"via `backstop gate --base` on the blocking job")
	}
}
