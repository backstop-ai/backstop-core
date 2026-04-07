package gate

import (
	"context"
	"os"
	"path/filepath"
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
