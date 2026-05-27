package gate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSpecFixture creates a minimal spec file in dir with the given claims
// and mandated test names.
func writeSpecFixture(t *testing.T, dir, filename string, claims []struct{ id, testName string }) {
	t.Helper()

	content := `---
title: "Test Spec"
number: TEST-001
created: "2026-01-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Test spec
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/... -race
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Test requirement
    supports: cli:REQ-001

claims:
`
	for _, c := range claims {
		content += "  - id: " + c.id + "\n"
		content += "    requirement: REQ-001\n"
		content += "    text: Test claim\n"
		content += "    tests:\n"
		content += "      - " + c.testName + "\n"
	}
	content += "---\n\n# Test Spec\n"

	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("writing spec fixture: %v", err)
	}
}

// writeTestFile creates a Go test file in dir with the given function names.
func writeTestFile(t *testing.T, dir, filename string, funcNames []string) {
	t.Helper()

	content := "package gate_test\n\nimport \"testing\"\n\n"
	for _, name := range funcNames {
		content += "func " + name + "(t *testing.T) {\n"
		content += "\tif true != true {\n"
		content += "\t\tt.Fatal(\"impossible\")\n"
		content += "\t}\n"
		content += "}\n\n"
	}

	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("writing test fixture: %v", err)
	}
}

// --- Test verification tests (step 3) ---

// TestGate_TestVerification_MandatedTestExists verifies that a mandated test
// function that exists in test files produces a pass.
func TestGate_TestVerification_MandatedTestExists(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()

	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_FixtureTestExists"},
	})
	writeTestFile(t, codeDir, "gate_test.go", []string{"TestGate_FixtureTestExists"})

	step := StepTestVerificationFunc(specDir, codeDir)
	result := step(context.Background())

	if result.StepName != StepTestVerification {
		t.Errorf("expected step_name %q, got %q", StepTestVerification, result.StepName)
	}
	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_TestVerification_MandatedTestMissing verifies that a missing
// mandated test function produces a failure.
func TestGate_TestVerification_MandatedTestMissing(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()

	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_DoesNotExist"},
	})
	// No test file — the mandated test doesn't exist.

	step := StepTestVerificationFunc(specDir, codeDir)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for missing test")
	}
}

// TestGate_TestVerification_CollectsAllSpecClaims verifies that test names
// from multiple specs and claims are all collected.
func TestGate_TestVerification_CollectsAllSpecClaims(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()

	writeSpecFixture(t, specDir, "spec-a.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_FromSpecA"},
		{"CLM-002", "TestGate_AlsoFromSpecA"},
	})
	writeSpecFixture(t, specDir, "spec-b.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_FromSpecB"},
	})

	// Provide test files that have all three functions
	writeTestFile(t, codeDir, "a_test.go", []string{"TestGate_FromSpecA", "TestGate_AlsoFromSpecA"})
	writeTestFile(t, codeDir, "b_test.go", []string{"TestGate_FromSpecB"})

	step := StepTestVerificationFunc(specDir, codeDir)
	result := step(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

func TestGateSteps_FilterToChangedFiles_TestVerification(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	changed := filepath.Join(codeDir, "changed_test.go")
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_ChangedTest"},
		{"CLM-002", "TestGate_UnchangedMissingTest"},
	})
	writeTestFile(t, codeDir, "changed_test.go", []string{"TestGate_ChangedTest"})

	result := StepTestVerificationScopedFunc(specDir, codeDir, newGateScope(codeDir, GateScopeModeDiff, []string{changed}, nil))(context.Background())
	if result.Status != "pass" || len(result.Violations) != 0 {
		t.Fatalf("expected scoped verification to ignore missing tests outside changed test files, got status=%s violations=%#v", result.Status, result.Violations)
	}
}

func TestGateSteps_FilterToChangedSpec_VerifiesMissingMandatedTest(t *testing.T) {
	root := t.TempDir()
	specDir := filepath.Join(root, "specs")
	codeDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_MissingFromChangedSpec"},
	})

	result := StepTestVerificationScopedFunc(specDir, codeDir, newGateScope(root, GateScopeModeDiff, []string{"specs/test.spec.md"}, nil))(context.Background())
	if result.Status != "fail" || len(result.Violations) != 1 {
		t.Fatalf("expected changed spec to verify missing mandated test, got status=%s violations=%#v", result.Status, result.Violations)
	}
	if !strings.Contains(result.Violations[0].Message, "TestGate_MissingFromChangedSpec") {
		t.Fatalf("expected violation to mention missing test, got %#v", result.Violations[0])
	}
}

// --- ExtractSpecVerifications tests ---

// TestGate_ExtractSpecVerifications_HappyPath verifies that verification
// blocks are extracted from spec files with both test_command and coverage_threshold.
func TestGate_ExtractSpecVerifications_HappyPath(t *testing.T) {
	specDir := t.TempDir()

	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestSomething"},
	})

	specs, err := ExtractSpecVerifications(specDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec verification, got %d", len(specs))
	}
	if specs[0].SpecID != "TEST-001" {
		t.Errorf("expected SpecID %q, got %q", "TEST-001", specs[0].SpecID)
	}
	if specs[0].TestCommand != "go test ./pkg/gate/... -race" {
		t.Errorf("expected TestCommand %q, got %q", "go test ./pkg/gate/... -race", specs[0].TestCommand)
	}
	if specs[0].CoverageThreshold != 80 {
		t.Errorf("expected CoverageThreshold 80, got %d", specs[0].CoverageThreshold)
	}
}

// TestGate_ExtractSpecVerifications_InvalidDir verifies that a non-existent
// directory returns an error.
func TestGate_ExtractSpecVerifications_InvalidDir(t *testing.T) {
	_, err := ExtractSpecVerifications("/nonexistent/path/to/specs")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

// TestGate_ExtractSpecVerifications_SkipsNonSpecFiles verifies that files
// without the .spec.md suffix are ignored.
func TestGate_ExtractSpecVerifications_SkipsNonSpecFiles(t *testing.T) {
	specDir := t.TempDir()

	// Write a non-spec file
	if err := os.WriteFile(filepath.Join(specDir, "notes.md"), []byte("# Notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	specs, err := ExtractSpecVerifications(specDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 0 {
		t.Errorf("expected 0 spec verifications, got %d", len(specs))
	}
}

// --- ExtractContractEntries tests ---

// TestGate_ExtractContractEntries_HappyPath verifies that contract entries
// are extracted from spec files and paths are resolved relative to projectRoot.
func TestGate_ExtractContractEntries_HappyPath(t *testing.T) {
	specDir := t.TempDir()

	content := `---
title: "Contract Spec"
number: SPEC-001
created: "2026-01-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Contract test
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/... -race
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Test req
    supports: cli:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Test claim
    tests:
      - TestSomething

contracts:
  - file: pkg/gate/gate.go
    provides:
      - name: New
        kind: function
        signature: "func New(opts ...Option) *Gate"
---

# Contract Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "contract.spec.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	contracts, err := ExtractContractEntries(specDir, "/project/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract entry, got %d", len(contracts))
	}
	if contracts[0].Name != "New" {
		t.Errorf("expected name %q, got %q", "New", contracts[0].Name)
	}
	if contracts[0].Kind != "function" {
		t.Errorf("expected kind %q, got %q", "function", contracts[0].Kind)
	}
	// Relative path should be joined with projectRoot
	expectedPath := filepath.Join("/project/root", "pkg/gate/gate.go")
	if contracts[0].File != expectedPath {
		t.Errorf("expected file %q, got %q", expectedPath, contracts[0].File)
	}
}

// TestGate_ExtractContractEntries_InvalidDir verifies that a non-existent
// directory returns an error.
func TestGate_ExtractContractEntries_InvalidDir(t *testing.T) {
	_, err := ExtractContractEntries("/nonexistent/path", "/root")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

// --- ResolveMandatedTestPaths tests ---

// TestGate_ResolveMandatedTestPaths_ResolvesExisting verifies that mandated
// tests get their FilePath set when the test function exists in codeDir.
func TestGate_ResolveMandatedTestPaths_ResolvesExisting(t *testing.T) {
	codeDir := t.TempDir()
	writeTestFile(t, codeDir, "foo_test.go", []string{"TestFoo_Works"})

	mandated := []MandatedTest{
		{FuncName: "TestFoo_Works", SpecID: "SPEC-001", ClaimID: "CLM-001"},
	}
	result := ResolveMandatedTestPaths(mandated, codeDir)

	if result[0].FilePath == "" {
		t.Error("expected FilePath to be set for existing test function")
	}
	if !filepath.IsAbs(result[0].FilePath) || !strings.HasSuffix(result[0].FilePath, "foo_test.go") {
		t.Errorf("expected FilePath ending in foo_test.go, got %q", result[0].FilePath)
	}
}

// TestGate_ResolveMandatedTestPaths_MissingTestUnresolved verifies that
// mandated tests that do not exist in codeDir keep an empty FilePath.
func TestGate_ResolveMandatedTestPaths_MissingTestUnresolved(t *testing.T) {
	codeDir := t.TempDir()
	// No test files in codeDir

	mandated := []MandatedTest{
		{FuncName: "TestDoesNotExist", SpecID: "SPEC-001", ClaimID: "CLM-001"},
	}
	result := ResolveMandatedTestPaths(mandated, codeDir)

	if result[0].FilePath != "" {
		t.Errorf("expected empty FilePath for missing test, got %q", result[0].FilePath)
	}
}

func TestGateSteps_FilterToChangedFiles_TestSubstantiveness(t *testing.T) {
	codeDir := t.TempDir()
	changed := filepath.Join(codeDir, "changed_test.go")
	unchanged := filepath.Join(codeDir, "unchanged_test.go")
	if err := os.WriteFile(changed, []byte("package gate_test\n\nimport \"testing\"\n\nfunc TestGate_Changed(t *testing.T) { t.Fatal(\"substantive\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unchanged, []byte("package gate_test\n\nimport \"testing\"\n\nfunc TestGate_Unchanged(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := StepTestSubstantivenessScopedFunc([]MandatedTest{
		{FuncName: "TestGate_Changed", FilePath: changed, TargetPkg: "gate"},
		{FuncName: "TestGate_Unchanged", FilePath: unchanged, TargetPkg: "gate"},
	}, newGateScope(codeDir, GateScopeModeDiff, []string{changed}, nil))(context.Background())
	if result.Status != "pass" || len(result.Violations) != 0 {
		t.Fatalf("expected substantiveness to ignore unchanged hollow test, got status=%s violations=%#v", result.Status, result.Violations)
	}
}

func TestGate_HasAssertions_HelperPatterns(t *testing.T) {
	codeDir := t.TempDir()
	testFile := filepath.Join(codeDir, "helpers_test.go")
	if err := os.WriteFile(testFile, []byte(`package gate_test

import "testing"

func requireThing() {}
func helper(t *testing.T) {}

func TestGate_HelperName(t *testing.T) { requireThing() }
func TestGate_HelperReceivesT(t *testing.T) { helper(t) }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TestGate_HelperName", "TestGate_HelperReceivesT"} {
		hollow, noTarget := checkSubstantiveness(testFile, name, "gate")
		if hollow || noTarget {
			t.Fatalf("expected %s to be substantive enough, hollow=%v noTarget=%v", name, hollow, noTarget)
		}
	}
}

// --- Test substantiveness tests (step 4) ---

// TestGate_TestSubstantiveness_SubstantiveTestPasses verifies that a test
// function with assertions and target package calls passes.
func TestGate_TestSubstantiveness_SubstantiveTestPasses(t *testing.T) {
	// Use the testdata fixture that has assertions and calls pkg/gate
	testFile, err := filepath.Abs("testdata/substantive-test.go")
	if err != nil {
		t.Fatal(err)
	}

	tests := []MandatedTest{
		{FuncName: "TestSubstantiveExample", FilePath: testFile, TargetPkg: "gate"},
	}
	step := StepTestSubstantivenessFunc(tests)
	result := step(context.Background())

	if result.StepName != StepTestSubstantiveness {
		t.Errorf("expected step_name %q, got %q", StepTestSubstantiveness, result.StepName)
	}
	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_TestSubstantiveness_HollowTestFails verifies that a test function
// with no assertions is detected as hollow.
func TestGate_TestSubstantiveness_HollowTestFails(t *testing.T) {
	testFile, err := filepath.Abs("testdata/hollow-test.go")
	if err != nil {
		t.Fatal(err)
	}

	tests := []MandatedTest{
		{FuncName: "TestHollowExample", FilePath: testFile, TargetPkg: "gate"},
	}
	step := StepTestSubstantivenessFunc(tests)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for hollow test")
	}
}

// TestGate_TestSubstantiveness_NoTargetCallFails verifies that a test function
// with assertions but no call to the target package is detected.
func TestGate_TestSubstantiveness_NoTargetCallFails(t *testing.T) {
	testFile, err := filepath.Abs("testdata/no-target-call-test.go")
	if err != nil {
		t.Fatal(err)
	}

	tests := []MandatedTest{
		{FuncName: "TestNoTargetCallExample", FilePath: testFile, TargetPkg: "gate"},
	}
	step := StepTestSubstantivenessFunc(tests)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for test with no target package call")
	}
}
