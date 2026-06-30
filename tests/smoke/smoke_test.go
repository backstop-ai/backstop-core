package smoke_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPath holds the path to the built backstop binary for all smoke tests.
var binaryPath string

// TestMain builds the backstop binary once and runs all smoke tests.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "backstop-smoke-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "backstop")

	// Build from the cmd/backstop package relative to the project root.
	projectRoot := findProjectRoot()
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/backstop/")
	cmd.Dir = projectRoot
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build backstop binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// findProjectRoot walks up from the current directory to find go.mod.
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintf(os.Stderr, "go.mod not found\n")
			os.Exit(1)
		}
		dir = parent
	}
}

// --- Helper functions ---

// runBackstop executes the backstop binary with the given args in dir.
// Returns stdout+stderr combined output and the exit code.
func runBackstop(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	// Clear BACKSTOP_CONFIG so discovery uses walk-up from dir.
	cmd.Env = append(os.Environ(), "BACKSTOP_CONFIG=")
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run backstop binary: %v", err)
		}
	}
	return string(out), exitCode
}

// createBackstopYml writes a backstop.yml into dir.
func createBackstopYml(t *testing.T, dir string) {
	t.Helper()
	content := "project: smoke-test\nlanguage: go\n"
	writeFile(t, filepath.Join(dir, "backstop.yml"), content)
}

// createSpec writes a spec file into dir/specs/ with the given options.
// Returns the path to the spec file.
func createSpec(t *testing.T, dir string, opts specOpts) string {
	t.Helper()
	specsDir := filepath.Join(dir, "specs")
	mustMkdir(t, specsDir)

	var claimsYAML string
	for _, c := range opts.claims {
		testsYAML := ""
		for _, tn := range c.tests {
			testsYAML += fmt.Sprintf("\n      - %s", tn)
		}
		claimsYAML += fmt.Sprintf(`
  - id: %s
    requirement: REQ-001
    text: "%s"
    tests:%s
`, c.id, c.text, testsYAML)
	}

	var contractsYAML string
	for _, c := range opts.contracts {
		contractsYAML += fmt.Sprintf(`
  - file: %s
    provides:
      - name: %s
        kind: %s
        signature: "%s"
`, c.file, c.name, c.kind, c.signature)
	}

	var verificationYAML string
	if opts.coverageThreshold > 0 {
		verificationYAML = fmt.Sprintf(`verification:
  level: unit
  test_command: go test %s -race
  coverage_threshold: %d
`, opts.testPkg, opts.coverageThreshold)
	} else {
		// Use level: static which does not require a coverage_threshold.
		verificationYAML = `verification:
  level: static
  test_command: go test ./... -race
`
	}

	content := fmt.Sprintf(`---
title: "SPEC-999: Smoke Test Spec"
number: SPEC-999
created: "2026-01-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Smoke test spec for integration testing.
  package: %s

%s
requirements:
  - id: REQ-001
    text: "The system must do the thing."

claims:
%s
contracts:
%s
---

# SPEC-999: Smoke Test Spec

## Overview

Smoke test spec.

## Requirements

- REQ-001: The system must do the thing.

## Implementation

Implementation details.

## Verification

Tests verify behavior.
`, opts.implPackage, verificationYAML, claimsYAML, contractsYAML)

	specPath := filepath.Join(specsDir, "SPEC-999-smoke-test.spec.md")
	writeFile(t, specPath, content)
	return specPath
}

type specOpts struct {
	implPackage      string
	claims           []claimOpts
	contracts        []contractOpts
	coverageThreshold int
	testPkg          string
}

type claimOpts struct {
	id    string
	text  string
	tests []string
}

type contractOpts struct {
	file      string
	name      string
	kind      string
	signature string
}

// createGoSource writes a Go source file into dir/pkg/smoke/.
func createGoSource(t *testing.T, dir, pkgName, code string) string {
	t.Helper()
	srcDir := filepath.Join(dir, "pkg", pkgName)
	mustMkdir(t, srcDir)
	path := filepath.Join(srcDir, pkgName+".go")
	writeFile(t, path, code)
	return path
}

// createGoTest writes a Go test file into dir/pkg/smoke/.
func createGoTest(t *testing.T, dir, pkgName, code string) string {
	t.Helper()
	srcDir := filepath.Join(dir, "pkg", pkgName)
	mustMkdir(t, srcDir)
	path := filepath.Join(srcDir, pkgName+"_test.go")
	writeFile(t, path, code)
	return path
}

// createGoMod writes a go.mod file into dir so that `go test` works on the fixtures.
func createGoMod(t *testing.T, dir, module string) {
	t.Helper()
	content := fmt.Sprintf("module %s\n\ngo 1.21\n", module)
	writeFile(t, filepath.Join(dir, "go.mod"), content)
}

// createBackstopRulesDir creates the .backstop/rules/ directory.
func createBackstopRulesDir(t *testing.T, dir string) {
	t.Helper()
	mustMkdir(t, filepath.Join(dir, ".backstop", "rules"))
}

// installSmokeGoToolchainPack declares + installs a LOCAL backstop/go-toolchain pack
// carrying the pack-declared classification.test globs + test_name_patterns DATA (no
// execution engines) so the gate's de-Go'd test-discovery capability is PRESENT
// (SPEC-045): with the baked `_test.go` walk + `funcPattern` deleted, a project that
// declares NO toolchain pack has test-discovery capability ABSENT (a non-blocking
// warning), so a compliant project — and the missing-test/hollow scenarios that depend
// on discovery — must declare one. It (re)writes backstop.yml with the pack declared,
// the pack manifest under .backstop/packs/, and a local backstop.lock (local entries
// skip the hash check; the lock is required because packs are now declared).
func installSmokeGoToolchainPack(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "backstop.yml"),
		"project: smoke-test\nlanguage: go\npacks:\n  backstop/go-toolchain: local\n")
	// Declare ONLY the test globs + test_name_patterns (the de-Go'd test-discovery
	// capability). Deliberately NO classification.source: declaring source globs would
	// make coverage capability PRESENT, but this fixture has no coverage engine, so
	// every changed `.go` would RED as coverage_unmeasured. Omitting source keeps
	// coverage capability-absent (a non-blocking warning, as before) while
	// test-discovery (test globs + patterns) is present.
	packYml := "name: backstop/go-toolchain\n" +
		"version: 1.0.0\n" +
		"language: go\n" +
		"archetype: code\n" +
		"description: Smoke go-toolchain fixture — classification.test + test_name_patterns DATA only (no engines).\n" +
		"classification:\n" +
		"  test:\n" +
		"    - \"**/*_test.go\"\n" +
		"    - \"**/testdata/**\"\n" +
		"test_name_patterns:\n" +
		"  - \"^\\\\s*func\\\\s+(Test\\\\w+)\\\\s*\\\\(\"\n" +
		"content:\n" +
		"  sdk:\n" +
		"    module: example/go-toolchain-fixture\n" +
		"    version: 1.0.0\n" +
		"    provides:\n" +
		"      - classification\n"
	writeFile(t, filepath.Join(dir, ".backstop", "packs", "backstop", "go-toolchain", "pack.yml"), packYml)
	lock := "packs:\n" +
		"    backstop/go-toolchain:\n" +
		"        content_hash: \"\"\n" +
		"        git_ref: null\n" +
		"        install_date: 2026-01-01T00:00:00Z\n" +
		"        name: backstop/go-toolchain\n" +
		"        source_type: local\n" +
		"        version: null\n"
	writeFile(t, filepath.Join(dir, "backstop.lock"), lock)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// parseGateJSON parses gate JSON output into a GateResult-like struct.
func parseGateJSON(t *testing.T, raw string) gateResult {
	t.Helper()
	var result gateResult
	// Gate output may have non-JSON lines; find the JSON object.
	trimmed := strings.TrimSpace(raw)
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		t.Fatalf("failed to parse gate JSON output: %v\nraw output:\n%s", err, raw)
	}
	return result
}

type gateResult struct {
	SchemaVersion   string       `json:"schema_version"`
	Pass            bool         `json:"pass"`
	TotalViolations int          `json:"total_violations"`
	StepsPassed     int          `json:"steps_passed"`
	StepsFailed     int          `json:"steps_failed"`
	StepsSkipped    int          `json:"steps_skipped"`
	Steps           []stepResult `json:"steps"`
}

type stepResult struct {
	StepName   string      `json:"step_name"`
	Status     string      `json:"status"`
	Violations []violation `json:"violations"`
	Reason     string      `json:"reason,omitempty"`
}

type violation struct {
	Rule     string `json:"rule"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
}

// findStep returns the step result with the given name, or fails the test.
func findStep(t *testing.T, result gateResult, stepName string) stepResult {
	t.Helper()
	for _, s := range result.Steps {
		if s.StepName == stepName {
			return s
		}
	}
	t.Fatalf("step %q not found in gate output; steps: %v", stepName, stepNames(result))
	return stepResult{}
}

func stepNames(r gateResult) []string {
	names := make([]string, len(r.Steps))
	for i, s := range r.Steps {
		names[i] = s.StepName
	}
	return names
}

// --- Scenario 1: Happy path — gate passes on compliant project ---

func TestSmoke_GatePassesOnCompliantProject(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	dir := t.TempDir()
	createBackstopYml(t, dir)
	installSmokeGoToolchainPack(t, dir)
	createBackstopRulesDir(t, dir)
	createGoMod(t, dir, "smoketest")

	// Create source with a function matching the contract.
	// No branches so a single test call achieves 100% coverage.
	createGoSource(t, dir, "smoke", `package smoke

func Compute(x int) error {
	return nil
}
`)

	// Create test with the mandated test name and real assertions.
	// Use external test package (smoke_test) so the substantiveness checker
	// sees the selector call smoke.Compute.
	createGoTest(t, dir, "smoke", `package smoke_test

import (
	"testing"

	"smoketest/pkg/smoke"
)

func TestCompute_ReturnsNil(t *testing.T) {
	err := smoke.Compute(42)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
`)

	// Create spec with claim pointing to the test, and contract matching the function.
	// Include coverage_threshold matching the required level for unit tests.
	createSpec(t, dir, specOpts{
		implPackage:       "pkg/smoke",
		coverageThreshold: 90,
		testPkg:           "./pkg/smoke/...",
		claims: []claimOpts{
			{id: "CLM-001", text: "Compute returns nil for positive input", tests: []string{"TestCompute_ReturnsNil"}},
		},
		contracts: []contractOpts{
			{file: "pkg/smoke/smoke.go", name: "Compute", kind: "function", signature: "func Compute(x int) error"},
		},
	})

	out, exitCode := runBackstop(t, dir, "gate", "--json")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput:\n%s", exitCode, out)
	}

	result := parseGateJSON(t, out)

	if !result.Pass {
		t.Errorf("expected pass=true, got false\nsteps: %+v", result.Steps)
	}

	// Steps 3 (test_verification) and 4 (test_substantiveness) should pass.
	testVerify := findStep(t, result, "test_verification")
	if testVerify.Status != "pass" {
		t.Errorf("test_verification: expected pass, got %s; violations: %+v", testVerify.Status, testVerify.Violations)
	}

	testSub := findStep(t, result, "test_substantiveness")
	if testSub.Status != "pass" {
		t.Errorf("test_substantiveness: expected pass, got %s; violations: %+v", testSub.Status, testSub.Violations)
	}

	// Step 6 (contract_signature) should pass.
	contract := findStep(t, result, "contract_signature")
	if contract.Status != "pass" {
		t.Errorf("contract_signature: expected pass, got %s; violations: %+v", contract.Status, contract.Violations)
	}

	// Deferred steps should be skipped.
	baseline := findStep(t, result, "baseline_comparison")
	if baseline.Status != "skipped" {
		t.Errorf("baseline_comparison: expected skipped, got %s", baseline.Status)
	}
}

// --- Scenario 2: Missing mandated test — gate step 3 fails ---

func TestSmoke_GateFailsMissingMandatedTest(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	dir := t.TempDir()
	createBackstopYml(t, dir)
	installSmokeGoToolchainPack(t, dir)
	createBackstopRulesDir(t, dir)
	createGoMod(t, dir, "smoketest")

	createGoSource(t, dir, "smoke", `package smoke

func Compute(x int) error {
	return nil
}
`)

	// Test file exists but does NOT contain the mandated test name.
	createGoTest(t, dir, "smoke", `package smoke

import "testing"

func TestSomethingElse(t *testing.T) {
	err := Compute(1)
	if err != nil {
		t.Fatal(err)
	}
}
`)

	// Spec mandates TestCompute_ReturnsNil but it does not exist.
	createSpec(t, dir, specOpts{
		implPackage: "pkg/smoke",
		claims: []claimOpts{
			{id: "CLM-001", text: "Compute works", tests: []string{"TestCompute_ReturnsNil"}},
		},
	})

	out, exitCode := runBackstop(t, dir, "gate", "--json")

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput:\n%s", exitCode, out)
	}

	result := parseGateJSON(t, out)
	testVerify := findStep(t, result, "test_verification")

	if testVerify.Status != "fail" {
		t.Errorf("test_verification: expected fail, got %s", testVerify.Status)
	}

	// Check that the violation mentions the missing test name.
	found := false
	for _, v := range testVerify.Violations {
		if strings.Contains(v.Message, "TestCompute_ReturnsNil") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected violation mentioning TestCompute_ReturnsNil, got: %+v", testVerify.Violations)
	}
}

// --- Scenario 3: Hollow test — gate step 4 fails ---

func TestSmoke_GateFailsHollowTest(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	dir := t.TempDir()
	createBackstopYml(t, dir)
	createBackstopRulesDir(t, dir)
	createGoMod(t, dir, "smoketest")

	createGoSource(t, dir, "smoke", `package smoke

func Compute(x int) error {
	return nil
}
`)

	// Mandated test exists but is hollow (empty body, no assertions).
	createGoTest(t, dir, "smoke", `package smoke

import "testing"

func TestCompute_ReturnsNil(t *testing.T) {}
`)

	createSpec(t, dir, specOpts{
		implPackage: "pkg/smoke",
		claims: []claimOpts{
			{id: "CLM-001", text: "Compute works", tests: []string{"TestCompute_ReturnsNil"}},
		},
	})

	out, exitCode := runBackstop(t, dir, "gate", "--json")

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput:\n%s", exitCode, out)
	}

	result := parseGateJSON(t, out)
	testSub := findStep(t, result, "test_substantiveness")

	if testSub.Status != "fail" {
		t.Errorf("test_substantiveness: expected fail, got %s", testSub.Status)
	}

	if len(testSub.Violations) == 0 {
		t.Error("expected at least one violation for hollow test")
	}
}

// --- Scenario 4: Coverage below threshold — gate step 5 fails ---

func TestSmoke_GateFailsCoverageBelowThreshold(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	dir := t.TempDir()
	createBackstopYml(t, dir)
	createBackstopRulesDir(t, dir)
	createGoMod(t, dir, "smoketest")

	// Source with multiple branches so coverage will be low.
	createGoSource(t, dir, "smoke", `package smoke

func Compute(x int) error {
	if x < 0 {
		return nil
	}
	if x > 100 {
		return nil
	}
	if x == 42 {
		return nil
	}
	if x == 7 {
		return nil
	}
	return nil
}
`)

	// Test that only covers one branch.
	createGoTest(t, dir, "smoke", `package smoke

import "testing"

func TestCompute_ReturnsNil(t *testing.T) {
	err := Compute(1)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
`)

	// Spec declares coverage_threshold: 95 which cannot be met.
	createSpec(t, dir, specOpts{
		implPackage:       "pkg/smoke",
		coverageThreshold: 95,
		testPkg:           "./pkg/smoke/...",
		claims: []claimOpts{
			{id: "CLM-001", text: "Compute works", tests: []string{"TestCompute_ReturnsNil"}},
		},
	})

	out, exitCode := runBackstop(t, dir, "gate", "--json")

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput:\n%s", exitCode, out)
	}

	result := parseGateJSON(t, out)
	coverage := findStep(t, result, "coverage_threshold")

	if coverage.Status != "fail" {
		t.Errorf("coverage_threshold: expected fail, got %s", coverage.Status)
	}

	// Verify violation mentions the threshold.
	found := false
	for _, v := range coverage.Violations {
		if strings.Contains(v.Message, "below threshold") || strings.Contains(v.Message, "95") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected coverage violation mentioning threshold, got: %+v", coverage.Violations)
	}
}

// --- Scenario 5: Contract signature mismatch — gate step 6 fails ---

func TestSmoke_GateFailsContractSignatureMismatch(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	dir := t.TempDir()
	createBackstopYml(t, dir)
	createBackstopRulesDir(t, dir)
	createGoMod(t, dir, "smoketest")

	// Code has `func Compute(x string) error` — parameter is string.
	createGoSource(t, dir, "smoke", `package smoke

func Compute(x string) error {
	return nil
}
`)

	createGoTest(t, dir, "smoke", `package smoke

import "testing"

func TestCompute_ReturnsNil(t *testing.T) {
	err := Compute("hello")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
`)

	// Spec contract declares `func Compute(x int) error` — parameter is int.
	createSpec(t, dir, specOpts{
		implPackage: "pkg/smoke",
		claims: []claimOpts{
			{id: "CLM-001", text: "Compute works", tests: []string{"TestCompute_ReturnsNil"}},
		},
		contracts: []contractOpts{
			{file: "pkg/smoke/smoke.go", name: "Compute", kind: "function", signature: "func Compute(x int) error"},
		},
	})

	out, exitCode := runBackstop(t, dir, "gate", "--json")

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput:\n%s", exitCode, out)
	}

	result := parseGateJSON(t, out)
	contract := findStep(t, result, "contract_signature")

	if contract.Status != "fail" {
		t.Errorf("contract_signature: expected fail, got %s", contract.Status)
	}

	// Verify violation mentions signature mismatch.
	found := false
	for _, v := range contract.Violations {
		if strings.Contains(v.Message, "mismatch") || strings.Contains(v.Message, "Compute") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected contract violation mentioning mismatch, got: %+v", contract.Violations)
	}
}

// --- Scenario 6: Code standards violation — code check catches it ---

func TestSmoke_CodeCheckCatchesViolation(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	// The code check pass executors (lint, build, test, semgrep) are currently
	// stubs that return empty results. Until the executors are wired to real
	// tools, code check cannot detect violations. Skip with explanation.
	t.Skip("code check pass executors are stubs — cannot detect violations until executors are implemented")

	// When executors are implemented, this test should:
	// 1. Create a project fixture with .backstop/rules/ containing semgrep config
	// 2. Add a Go file with `var globalState = map[string]string{}` (GO-003 violation)
	// 3. Run `backstop code check --json --all`
	// 4. Assert exit code 1 and violations in JSON output
}

// --- Scenario 7: Invalid artifact — artifact validate catches it ---

func TestSmoke_ArtifactValidateCatchesMissingClaims(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	dir := t.TempDir()
	createBackstopYml(t, dir)

	// Create a spec file with valid frontmatter but missing claims section in body.
	specsDir := filepath.Join(dir, "specs")
	mustMkdir(t, specsDir)

	specContent := `---
title: "SPEC-999: Incomplete Spec"
number: SPEC-999
created: "2026-01-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Missing claims.
  package: pkg/foo

verification:
  level: unit
  test_command: go test ./...
---

# SPEC-999: Incomplete Spec

## Overview

This spec is missing the required claims section.

## Requirements

- REQ-001: Something.

## Implementation

Details.

## Verification

Tests.
`
	writeFile(t, filepath.Join(specsDir, "SPEC-999-incomplete.spec.md"), specContent)

	out, exitCode := runBackstop(t, dir, "artifact", "validate", "--json")

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\noutput:\n%s", exitCode, out)
	}

	// Verify output mentions claims.
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "claim") {
		t.Errorf("expected output to mention 'claim', got:\n%s", out)
	}
}

// --- Scenario 8: Config error — gate halts immediately ---

func TestSmoke_GateExitsOnMissingConfig(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}

	// Temp dir with NO backstop.yml.
	dir := t.TempDir()

	out, exitCode := runBackstop(t, dir, "gate", "--json")

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d\noutput:\n%s", exitCode, out)
	}

	// Gate returns exit code 2 on config error. The CLI silences errors
	// (SilenceErrors=true) so output may be empty — the exit code is the
	// primary signal. If output is present, it should mention config.
	trimmed := strings.TrimSpace(out)
	if trimmed != "" {
		lower := strings.ToLower(trimmed)
		if !strings.Contains(lower, "config") && !strings.Contains(lower, "backstop.yml") {
			t.Errorf("expected output to mention config or backstop.yml, got:\n%s", out)
		}
	}
	// Exit code 2 alone is sufficient to confirm config error behavior.
}
