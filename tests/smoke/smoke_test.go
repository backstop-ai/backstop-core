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
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove temp dir %s: %v\n", tmpDir, err)
		}
	}()

	binaryPath = filepath.Join(tmpDir, "backstop")

	// Build from the cmd/backstop package relative to the project root.
	projectRoot := findProjectRoot()
	// @waiver:backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules.no-baked-tool-exec:deferred:2026-10-24 self-pack scoping over backstop-core's OWN test harness is ESCALATED and pending a founder posture decision (this harness must build and probe the module under test because backstop-core IS that module); remove this waiver when that decision lands
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
		// @waiver:backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules.no-baked-language-token:deferred:2026-10-24 self-pack scoping over backstop-core's OWN test harness is ESCALATED and pending a founder posture decision (this harness must build and probe the module under test because backstop-core IS that module); remove this waiver when that decision lands
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

	// Default to draft: this is the status createSpec hardcoded before ISSUE-075,
	// and TestSmoke_GatePassesOnCompliantProject / ...ArtifactValidateCatchesMissingClaims
	// depend on it (at draft, the substantiveness/coverage/contracts dimensions stay
	// capability-absent warnings). TestSmoke_CreateSpecDefaultsToDraftStatus guards it.
	specStatus := opts.status
	if specStatus == "" {
		specStatus = "draft"
	}

	content := fmt.Sprintf(`---
title: "SPEC-999: Smoke Test Spec"
number: SPEC-999
created: "2026-01-01"
status: %s
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
`, specStatus, opts.implPackage, verificationYAML, claimsYAML, contractsYAML)

	specPath := filepath.Join(specsDir, "SPEC-999-smoke-test.spec.md")
	writeFile(t, specPath, content)
	return specPath
}

type specOpts struct {
	// status is the spec frontmatter status. Empty means "draft" — the historical
	// hardcoded value every existing caller relies on.
	status            string
	implPackage       string
	claims            []claimOpts
	contracts         []contractOpts
	coverageThreshold int
	testPkg           string
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
	// @waiver:backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules.no-baked-language-token:deferred:2026-10-24 self-pack scoping over backstop-core's OWN test harness is ESCALATED and pending a founder posture decision (this harness must build and probe the module under test because backstop-core IS that module); remove this waiver when that decision lands
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

// createBackstopYmlDeclaring writes a backstop.yml that DECLARES a single
// traceability dimension via enforcement.toolchain (gate_type) WITHOUT installing
// the pack that provides its capability. Post-cutover (SPEC-037/038/041) the baked
// Go analyzers for substantiveness/contracts/coverage are deleted, so a declared-
// but-unprovided dimension is a broken promise: the SPEC-036 polarity classifier
// lands it in ClassDeclaredIntentUnmet and the gate BLOCKS (exit 2) with a
// `<dim>_declared_intent_unmet` violation — an engine-free RED with teeth. (True
// code-content defect detection — hollow test / low coverage / signature mismatch —
// now requires an installed pack engine and is covered by the pending installed-pack
// acceptance smoke; see project_gate_dogfood_mostly_dark.)
func createBackstopYmlDeclaring(t *testing.T, dir, gateType string) {
	t.Helper()
	content := "project: smoke-test\nlanguage: go\n" +
		"enforcement:\n" +
		"  toolchain:\n" +
		"    declared-pass:\n" +
		"      command: \"true\"\n" +
		"      format: \"sarif\"\n" +
		"      gate_type: \"" + gateType + "\"\n"
	writeFile(t, filepath.Join(dir, "backstop.yml"), content)
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
	// Gate output may have non-JSON lines (e.g. a trailing "Error: gate: exit code 2"
	// on a blocking exit); slice to the outermost JSON object before decoding.
	trimmed := strings.TrimSpace(raw)
	if start := strings.Index(trimmed, "{"); start >= 0 {
		if end := strings.LastIndex(trimmed, "}"); end > start {
			trimmed = trimmed[start : end+1]
		}
	}
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

// specFrontmatter reads the spec at path and returns ONLY its YAML frontmatter —
// the region between the opening and closing `---` delimiters. Status assertions
// must be scoped to this region so a mention anywhere in the markdown body can
// never satisfy them.
func specFrontmatter(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spec %s: %v", path, err)
	}
	content := string(raw)
	const opener = "---\n"
	if !strings.HasPrefix(content, opener) {
		t.Fatalf("spec %s does not open with a frontmatter delimiter; got:\n%s", path, content)
	}
	rest := content[len(opener):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatalf("spec %s has no closing frontmatter delimiter; got:\n%s", path, content)
	}
	return rest[:end]
}

// TestSmoke_CreateSpecHonorsStatusOverride pins half of the fixture contract the
// ISSUE-075 fix rests on: a caller can force the generated SPEC-999 to a status
// other than draft. Without the override, test_verification's implemented-only
// mandated-test scoping (ISSUE-054) filters every mandated test out before the
// discovery walk and the missing-mandated-test scenario becomes a vacuous pass.
func TestSmoke_CreateSpecHonorsStatusOverride(t *testing.T) {
	specPath := createSpec(t, t.TempDir(), specOpts{
		status:      "implemented",
		implPackage: "pkg/smoke",
		claims: []claimOpts{
			{id: "CLM-001", text: "Compute works", tests: []string{"TestCompute_ReturnsNil"}},
		},
	})

	fm := specFrontmatter(t, specPath)
	if !strings.Contains(fm, "status: implemented") {
		t.Errorf("expected frontmatter to declare `status: implemented`, got:\n%s", fm)
	}
	if strings.Contains(fm, "status: draft") {
		t.Errorf("expected the hardcoded draft status to be replaced, but frontmatter still declares `status: draft`:\n%s", fm)
	}
}

// TestSmoke_CreateSpecDefaultsToDraftStatus guards the other half: a specOpts that
// supplies no status must still emit `status: draft`, preserving today's behavior
// byte-for-byte for every existing caller. TestSmoke_GatePassesOnCompliantProject
// and TestSmoke_ArtifactValidateCatchesMissingClaims both depend on draft keeping
// substantiveness/coverage/contracts at capability-absent warnings, so flipping the
// default reds HERE rather than silently reshaping those scenarios.
func TestSmoke_CreateSpecDefaultsToDraftStatus(t *testing.T) {
	specPath := createSpec(t, t.TempDir(), specOpts{
		implPackage: "pkg/smoke",
		claims: []claimOpts{
			{id: "CLM-001", text: "Compute works", tests: []string{"TestCompute_ReturnsNil"}},
		},
	})

	fm := specFrontmatter(t, specPath)
	if !strings.Contains(fm, "status: draft") {
		t.Errorf("expected frontmatter to default to `status: draft`, got:\n%s", fm)
	}
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

	// test_verification is the one traceability dimension the fixture wires a
	// capability for (the go-toolchain pack declares classification.test +
	// test_name_patterns), so it runs for real and PASSES on the compliant fixture.
	testVerify := findStep(t, result, "test_verification")
	if testVerify.Status != "pass" {
		t.Errorf("test_verification: expected pass, got %s; violations: %+v", testVerify.Status, testVerify.Violations)
	}

	// Post-cutover (SPEC-037/038/041) substantiveness, coverage, and contracts are
	// installed-pack-keyed capabilities with NO baked Go analyzer. The compliant
	// fixture declares NONE of those packs (and does not declare them via
	// enforcement.toolchain), so each lands ClassCapabilityAbsent: a conspicuous
	// non-blocking `warning` (exit 0), never a vacuous pass. This is the honest
	// packs-only shape — the gate says "I can't check this, adopt the pack" rather
	// than silently greening. (The declared-but-unprovided BLOCK path is proven by
	// the TestSmoke_GateBlocksDeclared* scenarios; real content-defect detection is
	// covered by the pending installed-pack acceptance smoke.)
	for _, dim := range []string{"test_substantiveness", "coverage_threshold", "contract_signature"} {
		step := findStep(t, result, dim)
		if step.Status != "warning" {
			t.Errorf("%s: expected warning (capability absent), got %s; violations: %+v", dim, step.Status, step.Violations)
		}
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
	//
	// SPEC-999 MUST be `implemented` here, and that is load-bearing rather than
	// incidental. test_verification scopes mandated tests to IMPLEMENTED specs only
	// (ISSUE-054, commit 2164994): StepTestVerificationScopedFunc calls
	// filterDueMandatedTests → contractsAreDue in pkg/gate/step_testverify.go, which
	// drops the mandated tests of any spec whose status is not "implemented" — a draft
	// spec has not yet promised anything. At the fixture's previous hardcoded `draft`,
	// TestCompute_ReturnsNil was filtered out BEFORE the capability guard and the
	// discovery walk, the step hit its `len(mandated) == 0` early clean-pass return,
	// and this scenario passed while observing nothing at all. That was ISSUE-075: for
	// twelve days the one smoke test meant to prove test_verification blocks on a
	// broken test promise was structurally unable to fail.
	//
	// If the implemented-only scoping rule ever changes, THIS fixture must be revisited
	// in the same change — otherwise the scenario silently re-vacuates exactly as before.
	createSpec(t, dir, specOpts{
		status:      "implemented",
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

// --- Scenarios 3–5: declared-but-unprovided dimension → gate BLOCKS (exit 2) ---
//
// Post-cutover (SPEC-037/038/041) the baked Go analyzers for substantiveness,
// coverage, and contracts are DELETED — each is now an installed-pack-keyed
// capability. So a project that DECLARES a dimension via enforcement.toolchain
// (gate_type) but does NOT install the pack that provides it has made a broken
// promise: the SPEC-036 polarity classifier lands it in ClassDeclaredIntentUnmet
// and the gate BLOCKS with a `<dim>_declared_intent_unmet` violation (ConfigErr →
// exit 2). This is the engine-free RED-with-teeth these smoke tests assert.
//
// (These scenarios previously asserted in-binary detection of a hollow test, low
// coverage, and a signature mismatch from code content alone. That is no longer
// possible: those defects are only detectable by an installed pack ENGINE, and are
// covered by the pending installed-pack acceptance smoke — see
// project_gate_dogfood_mostly_dark. Retitled to reflect what they now prove.)

// blocksDeclaredWithoutPack is the shared body for the three broken-promise
// scenarios: a project declaring dim via enforcement.toolchain with no providing
// pack must BLOCK (exit 2) with the dimension's declared_intent_unmet violation.
func blocksDeclaredWithoutPack(t *testing.T, dim, stepName string) stepResult {
	t.Helper()
	if binaryPath == "" {
		t.Skip("binary not built")
		return stepResult{}
	}

	dir := t.TempDir()
	createBackstopYmlDeclaring(t, dir, dim)
	createBackstopRulesDir(t, dir)
	createGoMod(t, dir, "smoketest")
	// An empty specs dir so the (undeclared) test_verification step has nothing to
	// verify rather than erroring on a missing specs/ dir — the ONLY step that must
	// fail is the declared-but-unprovided dimension's.
	mustMkdir(t, filepath.Join(dir, "specs"))

	out, exitCode := runBackstop(t, dir, "gate", "--json")

	if exitCode != 2 {
		t.Fatalf("expected exit code 2 (broken-promise block), got %d\noutput:\n%s", exitCode, out)
	}

	result := parseGateJSON(t, out)
	step := findStep(t, result, stepName)

	if step.Status != "fail" {
		t.Errorf("%s: expected fail (declared-intent-unmet), got %s; violations: %+v", stepName, step.Status, step.Violations)
	}

	// The violation must be the broken-promise rule, and its message must name the
	// missing pack/capability (fail-loud-and-useful), not a code-content finding.
	found := false
	for _, v := range step.Violations {
		if v.Rule == dim+"_declared_intent_unmet" && strings.Contains(v.Message, "broken promise") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a %s_declared_intent_unmet broken-promise violation, got: %+v", dim, step.Violations)
	}

	// Returned so each scenario can pin ITS OWN dimension against the literal rule id
	// rather than the one this helper derives from its `dim` argument — a caller that
	// gets threaded the wrong dimension reds in the caller, where it is readable.
	return step
}

// hasViolationRule reports whether step carries a violation with exactly rule.
func hasViolationRule(step stepResult, rule string) bool {
	for _, v := range step.Violations {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

// --- Scenario 3: substantiveness declared, no pack → block ---

func TestSmoke_GateBlocksDeclaredSubstantivenessWithoutPack(t *testing.T) {
	step := blocksDeclaredWithoutPack(t, "substantiveness", "test_substantiveness")

	if step.StepName != "test_substantiveness" {
		t.Fatalf("expected the substantiveness dimension to resolve to the test_substantiveness step, got %q", step.StepName)
	}
	if !hasViolationRule(step, "substantiveness_declared_intent_unmet") {
		t.Errorf("expected a substantiveness_declared_intent_unmet violation on test_substantiveness, got: %+v", step.Violations)
	}
}

// --- Scenario 4: coverage declared, no pack → block ---

func TestSmoke_GateBlocksDeclaredCoverageWithoutPack(t *testing.T) {
	step := blocksDeclaredWithoutPack(t, "coverage", "coverage_threshold")

	if step.StepName != "coverage_threshold" {
		t.Fatalf("expected the coverage dimension to resolve to the coverage_threshold step, got %q", step.StepName)
	}
	if !hasViolationRule(step, "coverage_declared_intent_unmet") {
		t.Errorf("expected a coverage_declared_intent_unmet violation on coverage_threshold, got: %+v", step.Violations)
	}
}

// --- Scenario 5: contracts declared, no pack → block ---

func TestSmoke_GateBlocksDeclaredContractsWithoutPack(t *testing.T) {
	step := blocksDeclaredWithoutPack(t, "contracts", "contract_signature")

	if step.StepName != "contract_signature" {
		t.Fatalf("expected the contracts dimension to resolve to the contract_signature step, got %q", step.StepName)
	}
	if !hasViolationRule(step, "contracts_declared_intent_unmet") {
		t.Errorf("expected a contracts_declared_intent_unmet violation on contract_signature, got: %+v", step.Violations)
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
