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

// --- SPEC-045 pack-declared discovery helpers ---
// These construct the pack-declared SourceClassifier + TestNameMatcher the de-Go'd
// discovery walk consumes, mirroring the go-toolchain / bun-toolchain pack DATA.
// The binary holds NO baked test convention; every input below is DATA.

// goTestPattern is the go-toolchain pack's declared test-name regex AS DATA.
const goTestPattern = `^\s*func\s+(Test\w+)\s*\(`

// bunTestPattern is the bun-toolchain pack's declared test-name regex AS DATA
// (capture group 1 = the test(...)/describe(...)/it(...) string arg).
const bunTestPattern = "(?:\\bit|\\btest|\\bdescribe)\\s*\\(\\s*['\"`]([^'\"`]+)"

func goTestClassifier() SourceClassifier {
	return NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
}

func bunTestClassifier() SourceClassifier {
	return NewSourceClassifier([]string{"**/*.ts", "**/*.tsx"}, []string{"**/*.test.ts", "**/*.spec.ts"})
}

func unionTestClassifier() SourceClassifier {
	return NewSourceClassifier(
		[]string{"**/*.go", "**/*.ts"},
		[]string{"**/*_test.go", "**/testdata/**", "**/*.test.ts", "**/*.spec.ts"},
	)
}

func goTestMatcher(t *testing.T) TestNameMatcher {
	t.Helper()
	m, err := NewTestNameMatcher([]string{goTestPattern})
	if err != nil {
		t.Fatalf("NewTestNameMatcher(go): %v", err)
	}
	return m
}

func bunTestMatcher(t *testing.T) TestNameMatcher {
	t.Helper()
	m, err := NewTestNameMatcher([]string{bunTestPattern})
	if err != nil {
		t.Fatalf("NewTestNameMatcher(bun): %v", err)
	}
	return m
}

func unionTestMatcher(t *testing.T) TestNameMatcher {
	t.Helper()
	m, err := NewTestNameMatcher([]string{goTestPattern, bunTestPattern})
	if err != nil {
		t.Fatalf("NewTestNameMatcher(union): %v", err)
	}
	return m
}

func emptyTestMatcher(t *testing.T) TestNameMatcher {
	t.Helper()
	m, err := NewTestNameMatcher(nil)
	if err != nil {
		t.Fatalf("NewTestNameMatcher(nil): %v", err)
	}
	return m
}

// writeRawFile writes arbitrary content (a non-Go test file, a source file) at a
// path RELATIVE to dir, creating parent directories — used to stage `.test.ts`,
// `.go`, and `README.md` fixtures the discovery matrix walks.
func writeRawFile(t *testing.T, dir, relPath, content string) string {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", relPath, err)
	}
	return full
}

// SPEC-037: TestGate_TargetPackageName and TestGate_TestBodyHelpers were DELETED with
// the baked analyzer they subject (targetPackageName / hasAssertions / callsTargetPackage).
// TargetPackageName coverage is MIGRATED to the relocated pkg/gate.TargetPackageName as
// TestTargetPackageName_MigratedBehaviorPreserved (substantiveness_join_test.go, CLM-028).

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

	step := StepTestVerificationFunc(specDir, codeDir, goTestClassifier(), goTestMatcher(t))
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

	step := StepTestVerificationFunc(specDir, codeDir, goTestClassifier(), goTestMatcher(t))
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

	step := StepTestVerificationFunc(specDir, codeDir, goTestClassifier(), goTestMatcher(t))
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

	result := StepTestVerificationScopedFunc(specDir, codeDir, newGateScope(codeDir, GateScopeModeDiff, []string{changed}, nil), goTestClassifier(), goTestMatcher(t))(context.Background())
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

	result := StepTestVerificationScopedFunc(specDir, codeDir, newGateScope(root, GateScopeModeDiff, []string{"specs/test.spec.md"}, nil), goTestClassifier(), goTestMatcher(t))(context.Background())
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
	result := ResolveMandatedTestPaths(mandated, codeDir, goTestClassifier(), goTestMatcher(t))

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
	result := ResolveMandatedTestPaths(mandated, codeDir, goTestClassifier(), goTestMatcher(t))

	if result[0].FilePath != "" {
		t.Errorf("expected empty FilePath for missing test, got %q", result[0].FilePath)
	}
}

// SPEC-037: the analyzer-coupled substantiveness tests
// (TestGateSteps_FilterToChangedFiles_TestSubstantiveness,
// TestGate_HasAssertions_HelperPatterns,
// TestGate_TestSubstantiveness_SubstantiveTestPasses/HollowTestFails/NoTargetCallFails)
// were DELETED with the baked analyzer they call (StepTestSubstantiveness*Func /
// checkSubstantiveness). Their enforcement is re-proven through the pack path:
//   - the hollow/substantive/noTarget verdicts in substantiveness_q1_findings_test.go
//     and substantiveness_strangler_test.go (real ast-grep), and
//   - the changed-file SCOPE behavior in
//     TestSubstantiveness_ScopeAwareThroughPackPath_Preserved
//     (substantiveness_migration_test.go, CLM-029).

// ════════════════════════════════════════════════════════════════════════════
// SPEC-045 REQ-001 — test-FILE discovery via pack-declared TEST globs (IsTestFile)
// collectTestFuncNamesScoped(codeDir, scope, classifier, matcher) keys discovery on
// classifier.IsTestFile (NO baked `_test.go` walk) + matcher.FindName (NO funcPattern).
// ════════════════════════════════════════════════════════════════════════════

const tsTestBody = "test('renders the widget', () => {\n  expect(1).toBe(1)\n})\n"
const goTestBody = "package x\n\nimport \"testing\"\n\nfunc TestFoo(t *testing.T) {\n\tif false {\n\t\tt.Fatal(\"x\")\n\t}\n}\n"

// TestTestVerify_TSTestFileDiscoveredViaDeclaredGlobs (CLM-001, MANDATED): a TS
// `.test.ts` file IS discovered as a test file via the declared bun test glob
// `**/*.test.ts`, and its test name is extracted from the declared bun pattern.
func TestTestVerify_TSTestFileDiscoveredViaDeclaredGlobs(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "app/foo.test.ts", tsTestBody)

	found := collectTestFuncNamesScoped(codeDir, nil, bunTestClassifier(), bunTestMatcher(t))
	if _, ok := found["renders the widget"]; !ok {
		t.Fatalf("a `.test.ts` file must be discovered via the declared bun test glob and its name extracted, got %#v", found)
	}
}

// TestTestVerify_TSSpecFileDiscoveredViaDeclaredGlobs (CLM-002): a TS `.spec.ts`
// test file IS discovered via the declared bun test glob `**/*.spec.ts`.
func TestTestVerify_TSSpecFileDiscoveredViaDeclaredGlobs(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "app/foo.spec.ts", tsTestBody)

	found := collectTestFuncNamesScoped(codeDir, nil, bunTestClassifier(), bunTestMatcher(t))
	if _, ok := found["renders the widget"]; !ok {
		t.Fatalf("a `.spec.ts` file must be discovered via the declared bun test glob `**/*.spec.ts`, got %#v", found)
	}
}

// TestTestVerify_GoTestFileStillDiscoveredViaGoGlobs (CLM-003, MANDATED): a Go
// `_test.go` file IS STILL discovered via the declared go-toolchain test glob
// `**/*_test.go` — the symmetric proof that de-Go'ing did not regress Go.
func TestTestVerify_GoTestFileStillDiscoveredViaGoGlobs(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "pkg/x/foo_test.go", goTestBody)

	found := collectTestFuncNamesScoped(codeDir, nil, goTestClassifier(), goTestMatcher(t))
	if _, ok := found["TestFoo"]; !ok {
		t.Fatalf("a `_test.go` file must STILL be discovered via the declared go test glob, got %#v", found)
	}
}

// TestTestVerify_NonTestTSSourceNotDiscovered (CLM-004): a non-test TS source file
// (`app/foo.ts`) matching no declared test glob is NOT discovered, even though it
// contains a `test(...)` line — IsTestFile gates the walk.
func TestTestVerify_NonTestTSSourceNotDiscovered(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "app/foo.ts", tsTestBody)

	found := collectTestFuncNamesScoped(codeDir, nil, bunTestClassifier(), bunTestMatcher(t))
	if len(found) != 0 {
		t.Fatalf("a non-test `.ts` source file (no matching test glob) must NOT be discovered, got %#v", found)
	}
}

// TestTestVerify_NonTestGoSourceNotDiscovered (CLM-005): a non-test Go source file
// (`pkg/x/foo.go`) matching no declared test glob is NOT discovered, even with a
// `func TestFoo(` line.
func TestTestVerify_NonTestGoSourceNotDiscovered(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "pkg/x/foo.go", goTestBody)

	found := collectTestFuncNamesScoped(codeDir, nil, goTestClassifier(), goTestMatcher(t))
	if len(found) != 0 {
		t.Fatalf("a non-test `.go` source file (no matching test glob) must NOT be discovered, got %#v", found)
	}
}

// TestTestVerify_UnmatchedFileNotDiscovered (CLM-006): a file matching no declared
// glob at all (`README.md`) is NOT discovered.
func TestTestVerify_UnmatchedFileNotDiscovered(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "README.md", "# readme\ntest('x', () => {})\n")

	found := collectTestFuncNamesScoped(codeDir, nil, unionTestClassifier(), unionTestMatcher(t))
	if len(found) != 0 {
		t.Fatalf("a file matching no declared glob must NOT be discovered, got %#v", found)
	}
}

// TestTestVerify_NoBakedGoWalk_GoTestNotDiscoveredWithoutGoGlobs (CLM-007, de-Go
// proof): with ONLY bun test globs declared, a `_test.go` file is NOT discovered —
// proving the baked `_test.go` walk is gone and discovery keys only on the globs.
func TestTestVerify_NoBakedGoWalk_GoTestNotDiscoveredWithoutGoGlobs(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "pkg/x/foo_test.go", goTestBody)

	found := collectTestFuncNamesScoped(codeDir, nil, bunTestClassifier(), bunTestMatcher(t))
	if _, ok := found["TestFoo"]; ok {
		t.Fatalf("with ONLY bun test globs declared, a `_test.go` file must NOT be discovered (baked Go walk gone), got %#v", found)
	}
	if len(found) != 0 {
		t.Fatalf("expected zero discoveries when no declared glob matches the `_test.go` file, got %#v", found)
	}
}

// TestTestVerify_OutOfScopeTestFileSkipped (CLM-008): scope filtering is retained —
// an out-of-scope test file is skipped by the scoped discovery walk.
func TestTestVerify_OutOfScopeTestFileSkipped(t *testing.T) {
	codeDir := t.TempDir()
	inScope := writeRawFile(t, codeDir, "pkg/a/a_test.go", strings.ReplaceAll(goTestBody, "TestFoo", "TestInScope"))
	writeRawFile(t, codeDir, "pkg/b/b_test.go", strings.ReplaceAll(goTestBody, "TestFoo", "TestOutOfScope"))

	scope := newGateScope(codeDir, GateScopeModeDiff, []string{inScope}, nil)
	found := collectTestFuncNamesScoped(codeDir, scope, goTestClassifier(), goTestMatcher(t))

	if _, ok := found["TestInScope"]; !ok {
		t.Errorf("the in-scope test file must be discovered, got %#v", found)
	}
	if _, ok := found["TestOutOfScope"]; ok {
		t.Errorf("the out-of-scope test file must be skipped by the scoped walk, got %#v", found)
	}
}

// TestTestVerify_DiscoversAcrossUnionedTestGlobs (CLM-009): with both go and bun
// test globs declared on ONE merged classifier, BOTH a `_test.go` and a `.test.ts`
// file are discovered (polyglot union).
func TestTestVerify_DiscoversAcrossUnionedTestGlobs(t *testing.T) {
	codeDir := t.TempDir()
	writeRawFile(t, codeDir, "pkg/x/foo_test.go", goTestBody)
	writeRawFile(t, codeDir, "app/foo.test.ts", tsTestBody)

	found := collectTestFuncNamesScoped(codeDir, nil, unionTestClassifier(), unionTestMatcher(t))
	if _, ok := found["TestFoo"]; !ok {
		t.Errorf("the go `_test.go` member must be discovered from the merged classifier, got %#v", found)
	}
	if _, ok := found["renders the widget"]; !ok {
		t.Errorf("the bun `.test.ts` member must be discovered from the merged classifier, got %#v", found)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// SPEC-045 REQ-002 — test-NAME extraction from pack-declared regex DATA
// (TestNameMatcher.FindName returns capture group 1 of the first matching pattern)
// ════════════════════════════════════════════════════════════════════════════

// TestTestNameMatcher_ExtractsGoFuncNameFromDeclaredPattern (CLM-010).
func TestTestNameMatcher_ExtractsGoFuncNameFromDeclaredPattern(t *testing.T) {
	name, ok := goTestMatcher(t).FindName("func TestFoo(t *testing.T) {")
	if !ok || name != "TestFoo" {
		t.Fatalf("expected (TestFoo, true) from the declared go pattern, got (%q, %v)", name, ok)
	}
}

// TestTestNameMatcher_ExtractsTSTestNameFromDeclaredPattern (CLM-011).
func TestTestNameMatcher_ExtractsTSTestNameFromDeclaredPattern(t *testing.T) {
	name, ok := bunTestMatcher(t).FindName("test('renders the widget', () => {")
	if !ok || name != "renders the widget" {
		t.Fatalf("expected (renders the widget, true) from the declared bun test pattern, got (%q, %v)", name, ok)
	}
}

// TestTestNameMatcher_ExtractsTSDescribeNameFromDeclaredPattern (CLM-012).
func TestTestNameMatcher_ExtractsTSDescribeNameFromDeclaredPattern(t *testing.T) {
	name, ok := bunTestMatcher(t).FindName("describe('widget suite', () => {")
	if !ok || name != "widget suite" {
		t.Fatalf("expected (widget suite, true) from the declared bun describe pattern, got (%q, %v)", name, ok)
	}
}

// TestTestNameMatcher_ExtractsTSItNameFromDeclaredPattern (CLM-013).
func TestTestNameMatcher_ExtractsTSItNameFromDeclaredPattern(t *testing.T) {
	name, ok := bunTestMatcher(t).FindName("it('does the thing', async () => {")
	if !ok || name != "does the thing" {
		t.Fatalf("expected (does the thing, true) from the declared bun it pattern, got (%q, %v)", name, ok)
	}
}

// TestTestNameMatcher_NonMatchingLineYieldsNoName (CLM-014).
func TestTestNameMatcher_NonMatchingLineYieldsNoName(t *testing.T) {
	name, ok := unionTestMatcher(t).FindName("const x = 1")
	if ok || name != "" {
		t.Fatalf("a line matching no declared pattern must yield no name, got (%q, %v)", name, ok)
	}
}

// TestTestNameMatcher_NoBakedFuncTest_GoLineNotExtractedWithoutGoPattern (CLM-015,
// de-Go proof): with ONLY bun patterns declared, a Go `func TestFoo(` line extracts
// NO name — proving the baked `func Test` literal is gone.
func TestTestNameMatcher_NoBakedFuncTest_GoLineNotExtractedWithoutGoPattern(t *testing.T) {
	name, ok := bunTestMatcher(t).FindName("func TestFoo(t *testing.T) {")
	if ok {
		t.Fatalf("with only bun patterns declared, a Go func-Test line must extract NO name, got (%q, true)", name)
	}
}

// TestTestNameMatcher_InvalidRegexIsLoudError (CLM-016, loud-not-silent): an invalid
// declared regex makes NewTestNameMatcher return a construction error.
func TestTestNameMatcher_InvalidRegexIsLoudError(t *testing.T) {
	m, err := NewTestNameMatcher([]string{"(["})
	if err == nil {
		t.Fatalf("an invalid declared regex must be a LOUD construction error, got nil err (matcher %#v)", m)
	}
}

// TestTestVerify_NoBakedFuncTestRegexLiteral (CLM-017, source guard): no
// `func\s+(Test\w+)` regex literal remains in any non-test pkg/gate source file —
// a reintroduced baked pattern fails this guard.
func TestTestVerify_NoBakedFuncTestRegexLiteral(t *testing.T) {
	// The deleted funcPattern's distinctive literal, assembled at runtime so this
	// guard does not itself contain the forbidden literal.
	forbidden := "func" + `\s+(Test\w+)`
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/gate dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		if strings.Contains(string(data), forbidden) {
			t.Errorf("non-test source %s still contains the baked Go test-name regex literal %q — extraction must key only on pack-declared TestNamePatterns", e.Name(), forbidden)
		}
	}
}

// TestTestNameMatcher_MergesPatternsAcrossPacks (CLM-018, union): with both the go
// and bun name patterns declared, the merged matcher extracts a Go func name from a
// Go line AND a TS test name from a TS line.
func TestTestNameMatcher_MergesPatternsAcrossPacks(t *testing.T) {
	m := unionTestMatcher(t)
	if name, ok := m.FindName("func TestBar(t *testing.T) {"); !ok || name != "TestBar" {
		t.Errorf("merged matcher must extract the Go name, got (%q, %v)", name, ok)
	}
	if name, ok := m.FindName("it('does the thing', () => {"); !ok || name != "does the thing" {
		t.Errorf("merged matcher must extract the TS name, got (%q, %v)", name, ok)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// SPEC-045 REQ-005 — discovery capability-absent is a DISTINCT visible warning
// (EITHER test globs OR name patterns missing => capability absent), never a silent
// pass nor a mass false "not found" fail; full capability never masks a real miss.
// ════════════════════════════════════════════════════════════════════════════

// TestTestVerify_DiscoveryCapabilityAbsentIsVisibleWarningNotSilentOrFalseFail
// (CLM-031, both-absent): NO test globs AND NO name patterns + mandated tests exist
// => a DISTINCT visible `warning` naming the absent capability — never an
// unqualified pass and never a mass false "not found" fail.
func TestTestVerify_DiscoveryCapabilityAbsentIsVisibleWarningNotSilentOrFalseFail(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_SomeMandatedTest"},
	})

	emptyClassifier := NewSourceClassifier(nil, nil)
	result := StepTestVerificationFunc(specDir, codeDir, emptyClassifier, emptyTestMatcher(t))(context.Background())

	if result.Status != "warning" {
		t.Fatalf("both-absent capability must be a DISTINCT visible warning (not pass, not mass-fail), got status=%q violations=%#v", result.Status, result.Violations)
	}
	combined := result.Reason
	for _, v := range result.Violations {
		combined += " " + v.Message
	}
	if !strings.Contains(strings.ToLower(combined), "capab") && !strings.Contains(strings.ToLower(combined), "discover") {
		t.Errorf("the warning must NAME the absent test-discovery capability, got %q", combined)
	}
	// Must NOT be a mass false not-found fail: no "not found" violation for the mandated test.
	for _, v := range result.Violations {
		if strings.Contains(v.Message, "not found") {
			t.Errorf("capability-absent must NOT surface a false 'not found' violation, got %#v", v)
		}
	}
}

// TestTestVerify_DiscoveryCapabilityAbsentDoesNotBlock (CLM-032): the
// capability-absent state is NON-blocking — status is not `fail`, ConfigErr false.
func TestTestVerify_DiscoveryCapabilityAbsentDoesNotBlock(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_SomeMandatedTest"},
	})

	emptyClassifier := NewSourceClassifier(nil, nil)
	result := StepTestVerificationFunc(specDir, codeDir, emptyClassifier, emptyTestMatcher(t))(context.Background())

	if result.Status == "fail" {
		t.Fatalf("capability-absent must be non-blocking (not fail), got %#v", result)
	}
	if result.ConfigErr {
		t.Errorf("capability-absent advisory must not be a config error (non-blocking), got ConfigErr=true")
	}
}

// TestTestVerify_CapabilityPresentGenuineMissStillFails (CLM-033, no masking): with
// BOTH test globs AND name patterns declared (capability present) and a mandated
// test genuinely absent, the step still RAISES a LOUD blocking failure.
func TestTestVerify_CapabilityPresentGenuineMissStillFails(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_GenuinelyMissing"},
	})
	// No test file — genuine miss under full capability.

	result := StepTestVerificationFunc(specDir, codeDir, goTestClassifier(), goTestMatcher(t))(context.Background())
	if result.Status != "fail" {
		t.Fatalf("a genuine miss under FULL capability must stay a LOUD blocking fail, got %#v", result)
	}
	if len(result.Violations) == 0 {
		t.Errorf("expected a not-found violation for the genuinely-missing mandated test")
	}
}

// TestTestVerify_TestGlobsDeclaredButNoNamePatterns_IsVisibleWarningNotMassFail
// (CLM-037, the PARTIAL trap): test globs declared BUT no name patterns (FindName
// returns false for every line) => the DISTINCT visible warning naming the missing
// name patterns — NOT a mass false "not found" blocking fail. This is exactly the
// case the EITHER-absent (`||`) guard exists to intercept.
func TestTestVerify_TestGlobsDeclaredButNoNamePatterns_IsVisibleWarningNotMassFail(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_SomeMandatedTest"},
		{"CLM-002", "TestGate_AnotherMandatedTest"},
	})
	// Globs declared (the walk would find files) but NO name patterns => FindName
	// returns false for every line; without the OR guard this becomes a mass fail.
	writeRawFile(t, codeDir, "pkg/x/foo_test.go", goTestBody)

	result := StepTestVerificationFunc(specDir, codeDir, goTestClassifier(), emptyTestMatcher(t))(context.Background())

	if result.Status != "warning" {
		t.Fatalf("globs-but-no-patterns must be the DISTINCT visible warning, NOT a mass not-found fail, got status=%q violations=%#v", result.Status, result.Violations)
	}
	combined := result.Reason
	for _, v := range result.Violations {
		combined += " " + v.Message
		if strings.Contains(v.Message, "not found") {
			t.Errorf("the partial case must NOT report mandated tests as falsely 'not found', got %#v", v)
		}
	}
	if !strings.Contains(strings.ToLower(combined), "pattern") {
		t.Errorf("the partial-case warning must NAME the missing name patterns, got %q", combined)
	}
}
