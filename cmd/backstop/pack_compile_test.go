package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/compile"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testStandardPattern returns a valid .standard.md with a pattern rule
// that produces manifest + semgrep output.
func testStandardPattern(number, language string) string {
	lang := strings.ToUpper(language)
	return fmt.Sprintf(`---
title: "Test Standard %s"
number: STD-%s-%s
created: "2026-01-01"
status: active
schema_version: standard/v1
pack: test
scope: language
language: %s

rules:
  - id: %s-001
    name: test-pattern-rule
    category: structure
    severity: error
    description: test pattern rule
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: "var $X = ..."
    fix: fix it
---

# STD-%s-%s: Test Standard

## Overview

Test overview.

## Rules

Rules content.

## Examples

Examples content.
`, number, lang, number, language, lang, lang, number)
}

// testStandardMetric returns a valid .standard.md with a metric rule
// that produces manifest + native checks output.
func testStandardMetric(number, language string) string {
	lang := strings.ToUpper(language)
	return fmt.Sprintf(`---
title: "Test Metric Standard %s"
number: STD-%s-%s
created: "2026-01-01"
status: active
schema_version: standard/v1
pack: test
scope: language
language: %s

rules:
  - id: %s-001
    name: test-metric-rule
    category: performance
    severity: error
    description: test metric rule
    compliance_tier: baseline
    detection:
      strategy: metric
      metric: file_lines
      operator: ">"
      threshold: 500
      exclude:
        - "_test.go"
    fix: split the file
---

# STD-%s-%s: Test Metric Standard

## Overview

Test overview.

## Rules

Rules content.

## Examples

Examples content.
`, number, lang, number, language, lang, lang, number)
}

// testStandardDeprecated returns a deprecated .standard.md with superseded_by.
func testStandardDeprecated(number, supersededBy, language string) string {
	lang := strings.ToUpper(language)
	return fmt.Sprintf(`---
title: "Deprecated Standard %s"
number: STD-%s-%s
created: "2026-01-01"
status: deprecated
schema_version: standard/v1
pack: test
scope: language
language: %s
superseded_by: %s

rules:
  - id: %s-001
    name: test-deprecated-rule
    category: structure
    severity: error
    description: test deprecated rule
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: "var $X = ..."
    fix: fix it
---

# STD-%s-%s: Deprecated Standard

## Overview

Deprecated test overview.

## Rules

Rules content.

## Examples

Examples content.
`, number, lang, number, language, supersededBy, lang, lang, number)
}

// testStandardInvalid returns a .standard.md that will fail compilation
// (unsupported strategy).
func testStandardInvalid(number string) string {
	return fmt.Sprintf(`---
title: "Invalid Standard %s"
number: STD-GO-%s
created: "2026-01-01"
status: active
schema_version: standard/v1
pack: test
scope: language
language: go

rules:
  - id: GO-001
    name: test-invalid-rule
    category: structure
    severity: error
    description: test invalid rule
    compliance_tier: baseline
    detection:
      strategy: unknown_strategy
    fix: cannot fix
---

# STD-GO-%s: Invalid Standard

## Overview

Invalid test overview.

## Rules

Rules content.

## Examples

Examples content.
`, number, number, number)
}

// testSchemaSource returns an embeddedSchemaSource using the embedded SchemaFS.
func testSchemaSource(t *testing.T) *embeddedSchemaSource {
	t.Helper()
	src := &embeddedSchemaSource{fsys: SchemaFS}
	t.Cleanup(src.cleanup)
	return src
}

// setupProjectDir creates a minimal project directory with backstop.yml
// and optional standards, returning the project root path.
func setupProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, "backstop.yml"), "project: test\nlanguage: go\n")
	return dir
}

// setupProjectWithStandards creates a project dir with backstop.yml and
// writes standard files into the standards/ directory.
func setupProjectWithStandards(t *testing.T, standards map[string]string) string {
	t.Helper()
	dir := setupProjectDir(t)
	for name, content := range standards {
		writeFixture(t, filepath.Join(dir, "standards", name), content)
	}
	return dir
}

// runPackCompileInDir runs the pack compile command in the given directory
// and returns stdout output and error.
func runPackCompileInDir(t *testing.T, dir string, jsonMode bool) (string, error) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Clear BACKSTOP_CONFIG to use walk-up discovery
	t.Setenv("BACKSTOP_CONFIG", "")

	root := NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	args := []string{"pack", "compile"}
	if jsonMode {
		args = append(args, "--json")
	}
	root.SetArgs(args)
	execErr := root.Execute()
	return buf.String(), execErr
}

// ---------------------------------------------------------------------------
// Phase 3: Config loading and exit codes
// ---------------------------------------------------------------------------

// TestPackCompile_ExitCode2OnMissingConfig verifies exit code 2 when
// backstop.yml is missing. (CLM-027)
func TestPackCompile_ExitCode2OnMissingConfig(t *testing.T) {
	dir := t.TempDir() // empty — no backstop.yml
	_, err := runPackCompileInDir(t, dir, false)
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitErr.Code)
	}
}

// TestPackCompile_ExitCode2OnInvalidConfig verifies exit code 2 when
// backstop.yml is invalid YAML. (CLM-028)
func TestPackCompile_ExitCode2OnInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, "backstop.yml"), ": invalid {{[[\n")
	_, err := runPackCompileInDir(t, dir, false)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitErr.Code)
	}
}

// TestPackCompile_ConfigLoadedBeforeDiscovery verifies config error occurs
// before any standard discovery or compilation. (CLM-029)
func TestPackCompile_ConfigLoadedBeforeDiscovery(t *testing.T) {
	dir := t.TempDir() // no backstop.yml
	// Create a standards dir with a file — if discovery ran, it would find this
	writeFixture(t, filepath.Join(dir, "standards", "STD-GO-001-test-valid.standard.md"),
		testStandardPattern("001", "go"))

	_, err := runPackCompileInDir(t, dir, false)
	if err == nil {
		t.Fatal("expected config error before discovery")
	}

	// Verify the error is about config, not about standards
	if !strings.Contains(err.Error(), "config") && !strings.Contains(err.Error(), "backstop.yml") {
		t.Errorf("expected config-related error, got: %v", err)
	}

	// Verify no output files were created (compilation didn't run)
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if _, statErr := os.Stat(rulesDir); statErr == nil {
		t.Error("output directory should not exist — config should fail before compilation")
	}
}

// TestPackCompile_ExitCode0OnSuccess verifies exit code 0 when all
// standards compile successfully. (CLM-011)
func TestPackCompile_ExitCode0OnSuccess(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	_, err := runPackCompileInDir(t, dir, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPackCompile_ExitCode1OnCompilationError verifies exit code 1 when
// a standard fails compilation. (CLM-012)
func TestPackCompile_ExitCode1OnCompilationError(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-999-test-invalid.standard.md": testStandardInvalid("999"),
	})

	_, err := runPackCompileInDir(t, dir, false)
	if err == nil {
		t.Fatal("expected error for compilation failure")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitViolations {
		t.Errorf("expected exit code %d, got %d", ExitViolations, exitErr.Code)
	}
}

// TestPackCompile_ExitCode2OnConfigError verifies exit code 2 when a
// configuration error prevents compilation. (CLM-013)
func TestPackCompile_ExitCode2OnConfigError(t *testing.T) {
	// Same as ExitCode2OnMissingConfig — config error = exit 2
	dir := t.TempDir()
	_, err := runPackCompileInDir(t, dir, false)
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitErr.Code)
	}
}

// TestPackCompile_ExitCode2OnAllStandardsDirsMissing verifies exit code 2
// when no configured standards directories exist. (CLM-014)
func TestPackCompile_ExitCode2OnAllStandardsDirsMissing(t *testing.T) {
	dir := setupProjectDir(t)
	// Don't create the standards/ directory — it should not exist

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"nonexistent/", "also-missing/"},
		outputDir:     ".backstop/rules",
		schemaSource:  testSchemaSource(t),
	}

	_, err := runPackCompileWithOpts(opts)
	if err == nil {
		t.Fatal("expected error when all standards dirs missing")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitErr.Code)
	}
}

// TestPackCompile_PartialDirsMissingCompilesAndWarns verifies that when some
// configured dirs exist and others don't, compilation proceeds with warnings. (CLM-030)
func TestPackCompile_PartialDirsMissingCompilesAndWarns(t *testing.T) {
	dir := setupProjectDir(t)
	stdDir := filepath.Join(dir, "existing-standards")
	writeFixture(t, filepath.Join(stdDir, "STD-GO-001-test-valid.standard.md"),
		testStandardPattern("001", "go"))

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{stdDir, filepath.Join(dir, "nonexistent")},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should have compiled the standard from the existing dir
	if result.Summary.Compiled != 1 {
		t.Errorf("expected 1 compiled, got %d", result.Summary.Compiled)
	}

	// Should have warning about missing directory
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "nonexistent") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about nonexistent directory, warnings: %v", result.Warnings)
	}
}

// ---------------------------------------------------------------------------
// Phase 4: Compilation delegation + output directory
// ---------------------------------------------------------------------------

// TestPackCompile_CallsCompileForEachStandard verifies compile.Compile is
// called for each discovered standard. (CLM-004)
func TestPackCompile_CallsCompileForEachStandard(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-first.standard.md":  testStandardPattern("001", "go"),
		"STD-GO-002-test-second.standard.md": testStandardMetric("002", "go"),
	})

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Total != 2 {
		t.Errorf("expected 2 total, got %d", result.Summary.Total)
	}
	if result.Summary.Compiled != 2 {
		t.Errorf("expected 2 compiled, got %d", result.Summary.Compiled)
	}
	if len(result.Standards) != 2 {
		t.Fatalf("expected 2 standard results, got %d", len(result.Standards))
	}
}

// TestPackCompile_OutputDirIsBackstopRules verifies output files go to
// .backstop/rules/. (CLM-005)
func TestPackCompile_OutputDirIsBackstopRules(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, std := range result.Standards {
		for _, p := range std.OutputPaths {
			if !strings.Contains(p, ".backstop") || !strings.Contains(p, "rules") {
				t.Errorf("output path %q does not contain .backstop/rules", p)
			}
		}
	}
}

// TestPackCompile_DelegatesCompilationToPkgCompile verifies the command
// delegates to pkg/compile and produces the same output. (CLM-006)
func TestPackCompile_DelegatesCompilationToPkgCompile(t *testing.T) {
	dir := setupProjectDir(t)
	stdContent := testStandardPattern("001", "go")
	stdPath := filepath.Join(dir, "standards", "STD-GO-001-test-valid.standard.md")
	writeFixture(t, stdPath, stdContent)

	outDir := filepath.Join(dir, ".backstop", "rules")

	// Run via command adapter
	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     outDir,
		schemaSource:  testSchemaSource(t),
	}
	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}

	// Run directly via pkg/compile
	directOutDir := filepath.Join(dir, "direct-output")
	if err := os.MkdirAll(directOutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	directResult, directErr := compile.Compile(stdPath, compile.CompileOptions{SchemaSource: testSchemaSource(t),
		OutputDir: directOutDir,
	})
	if directErr != nil {
		t.Fatalf("direct compile error: %v", directErr)
	}

	// Verify same standard number
	if len(result.Standards) != 1 {
		t.Fatalf("expected 1 standard result, got %d", len(result.Standards))
	}
	if result.Standards[0].Standard != directResult.Manifest.Standard {
		t.Errorf("standard mismatch: command=%q, direct=%q",
			result.Standards[0].Standard, directResult.Manifest.Standard)
	}

	// Verify same number of output files
	if len(result.Standards[0].OutputPaths) != len(directResult.OutputPaths) {
		t.Errorf("output path count mismatch: command=%d, direct=%d",
			len(result.Standards[0].OutputPaths), len(directResult.OutputPaths))
	}

	// Verify output file contents are identical (same manifest data)
	for i, cmdPath := range result.Standards[0].OutputPaths {
		cmdData, err := os.ReadFile(cmdPath)
		if err != nil {
			t.Fatalf("reading command output: %v", err)
		}
		directData, err := os.ReadFile(directResult.OutputPaths[i])
		if err != nil {
			t.Fatalf("reading direct output: %v", err)
		}
		if !bytes.Equal(cmdData, directData) {
			t.Errorf("output file %d differs between command and direct compile", i)
		}
	}
}

// TestPackCompile_ProducesAllOutputTypes verifies manifest JSON, semgrep YAML,
// and native checks JSON are produced for a standard with both rule types. (CLM-007)
func TestPackCompile_ProducesAllOutputTypes(t *testing.T) {
	// Create a standard with both pattern and metric rules
	stdContent := `---
title: Multi-Rule Standard
number: STD-GO-010
created: "2026-01-01"
status: active
schema_version: standard/v1
pack: test
scope: language
language: go

rules:
  - id: GO-011
    name: test-pattern
    category: structure
    severity: error
    description: pattern rule
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: "var $X = ..."
    fix: fix it
  - id: GO-012
    name: test-metric
    category: performance
    severity: error
    description: metric rule
    compliance_tier: baseline
    detection:
      strategy: metric
      metric: file_lines
      operator: ">"
      threshold: 500
      exclude:
        - "_test.go"
    fix: split the file
---

# STD-GO-010: Multi-Rule Standard

## Overview

Standard with both pattern and metric rules.

## Rules

Rules content.

## Examples

Examples content.
`
	dir := setupProjectDir(t)
	writeFixture(t, filepath.Join(dir, "standards", "STD-GO-010-multi-rule.standard.md"), stdContent)

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Standards) != 1 {
		t.Fatalf("expected 1 standard, got %d", len(result.Standards))
	}

	outputPaths := result.Standards[0].OutputPaths
	if len(outputPaths) != 3 {
		t.Fatalf("expected 3 output files (manifest, semgrep, native), got %d: %v",
			len(outputPaths), outputPaths)
	}

	hasManifest, hasSemgrep, hasNative := false, false, false
	for _, p := range outputPaths {
		base := filepath.Base(p)
		if strings.HasSuffix(base, ".manifest.json") {
			hasManifest = true
		}
		if strings.HasSuffix(base, ".semgrep.yml") {
			hasSemgrep = true
		}
		if strings.HasSuffix(base, ".native.json") {
			hasNative = true
		}
	}
	if !hasManifest {
		t.Error("missing manifest JSON output")
	}
	if !hasSemgrep {
		t.Error("missing semgrep YAML output")
	}
	if !hasNative {
		t.Error("missing native checks JSON output")
	}
}

// TestPackCompile_CreatesOutputDir verifies the command creates .backstop/rules/
// if it does not exist. (CLM-019)
func TestPackCompile_CreatesOutputDir(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	outDir := filepath.Join(dir, ".backstop", "rules")
	// Verify it doesn't exist yet
	if _, err := os.Stat(outDir); err == nil {
		t.Fatal("output dir should not exist yet")
	}

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     outDir,
		schemaSource:  testSchemaSource(t),
	}

	_, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it was created
	info, err := os.Stat(outDir)
	if err != nil {
		t.Fatalf("output dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("output path should be a directory")
	}
}

// TestPackCompile_OverwritesExistingFiles verifies the command overwrites
// output files from previous compilations. (CLM-020)
func TestPackCompile_OverwritesExistingFiles(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	outDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write an old file that should be overwritten
	oldManifest := filepath.Join(outDir, "STD-GO-001.manifest.json")
	writeFixture(t, oldManifest, `{"old": "data"}`)

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     outDir,
		schemaSource:  testSchemaSource(t),
	}

	_, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was overwritten with new content
	data, err := os.ReadFile(oldManifest)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	if strings.Contains(string(data), `"old"`) {
		t.Error("old manifest content should have been overwritten")
	}
	if !strings.Contains(string(data), "STD-GO-001") {
		t.Error("new manifest should contain standard number")
	}
}

// TestPackCompile_NoStandardsExitCode0 verifies exit code 0 when no
// .standard.md files are found. (CLM-021)
func TestPackCompile_NoStandardsExitCode0(t *testing.T) {
	dir := setupProjectDir(t)
	// Create empty standards directory
	if err := os.MkdirAll(filepath.Join(dir, "standards"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := runPackCompileInDir(t, dir, false)
	if err != nil {
		t.Fatalf("expected no error for empty standards, got: %v", err)
	}
}

// TestPackCompile_NoStandardsReportsZero verifies the summary reports
// 0 standards compiled when none are found. (CLM-022)
func TestPackCompile_NoStandardsReportsZero(t *testing.T) {
	dir := setupProjectDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "standards"), 0o755); err != nil {
		t.Fatal(err)
	}

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Total != 0 {
		t.Errorf("expected 0 total, got %d", result.Summary.Total)
	}
	if result.Summary.Compiled != 0 {
		t.Errorf("expected 0 compiled, got %d", result.Summary.Compiled)
	}
}

// ---------------------------------------------------------------------------
// Phase 5: Output modes, deprecation, partial failure, idempotency
// ---------------------------------------------------------------------------

// TestPackCompile_JSONOutputStructure verifies JSON output has the expected
// structure with standards, warnings, errors, and summary. (CLM-008)
func TestPackCompile_JSONOutputStructure(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	out, err := runPackCompileInDir(t, dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse via envelope to verify schema_version is present in JSON output
	var envelope compileJSONEnvelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("JSON unmarshal error: %v\noutput: %s", err, out)
	}

	// REQ-004: schema_version is added by the output formatter envelope,
	// not carried in PackCompileResult itself.
	if envelope.SchemaVersion != "cli/v1" {
		t.Errorf("expected schema_version cli/v1 in JSON envelope, got %q", envelope.SchemaVersion)
	}

	// Also parse into plain PackCompileResult to confirm it works without schema_version
	var parsed PackCompileResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("JSON unmarshal error: %v\noutput: %s", err, out)
	}

	if len(parsed.Standards) != 1 {
		t.Errorf("expected 1 standard in JSON, got %d", len(parsed.Standards))
	}
	if parsed.Standards[0].Standard != "STD-GO-001" {
		t.Errorf("expected standard STD-GO-001, got %q", parsed.Standards[0].Standard)
	}
	if parsed.Summary.Compiled != 1 {
		t.Errorf("expected summary.compiled=1, got %d", parsed.Summary.Compiled)
	}
	// Warnings and errors should be arrays (even if empty)
	if parsed.Warnings == nil {
		t.Error("warnings should be non-nil array")
	}
	if parsed.Errors == nil {
		t.Error("errors should be non-nil array")
	}
}

// TestPackCompile_HumanOutputFormat verifies human output includes standard
// names, output files, and summary. (CLM-009)
func TestPackCompile_HumanOutputFormat(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	out, err := runPackCompileInDir(t, dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Compiled") {
		t.Error("human output should contain 'Compiled' header")
	}
	if !strings.Contains(out, "OK") {
		t.Error("human output should contain 'OK' for successful compilation")
	}
	if !strings.Contains(out, ".standard.md") {
		t.Error("human output should contain source file path")
	}
	if !strings.Contains(out, "Summary:") {
		t.Error("human output should contain summary line")
	}
}

// TestPackCompile_OutputModesIdenticalData verifies JSON and human modes
// produce identical underlying data. (CLM-010)
func TestPackCompile_OutputModesIdenticalData(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	// Get JSON output
	jsonOut, err := runPackCompileInDir(t, dir, true)
	if err != nil {
		t.Fatalf("JSON mode error: %v", err)
	}

	var jsonResult PackCompileResult
	if err := json.Unmarshal([]byte(jsonOut), &jsonResult); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// Get human output
	humanOut, err := runPackCompileInDir(t, dir, false)
	if err != nil {
		t.Fatalf("human mode error: %v", err)
	}

	// Verify human output reflects the same data
	if jsonResult.Summary.Compiled != 1 {
		t.Errorf("JSON compiled count should be 1, got %d", jsonResult.Summary.Compiled)
	}
	if !strings.Contains(humanOut, "1 compiled") {
		t.Error("human output should show 1 compiled in summary")
	}
	if !strings.Contains(humanOut, fmt.Sprintf("%d total", jsonResult.Summary.Total)) {
		t.Error("human output total should match JSON total")
	}
}

// TestPackCompile_DeprecatedWarningInOutput verifies deprecation warnings
// appear in command output. (CLM-015)
func TestPackCompile_DeprecatedWarningInOutput(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-003-test-deprecated.standard.md": testStandardDeprecated("003", "STD-GO-004", "go"),
	})

	out, err := runPackCompileInDir(t, dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result PackCompileResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Fatal("expected deprecation warning in output")
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "deprecated") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no deprecation warning found in warnings: %v", result.Warnings)
	}
}

// TestPackCompile_DeprecatedStandardStillCompiles verifies deprecated standards
// produce output files (not skipped). (CLM-016)
func TestPackCompile_DeprecatedStandardStillCompiles(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-003-test-deprecated.standard.md": testStandardDeprecated("003", "STD-GO-004", "go"),
	})

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Compiled != 1 {
		t.Errorf("expected 1 compiled (deprecated still compiles), got %d", result.Summary.Compiled)
	}
	if result.Summary.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Summary.Failed)
	}
	if len(result.Standards) != 1 || len(result.Standards[0].OutputPaths) == 0 {
		t.Error("deprecated standard should produce output files")
	}
}

// TestPackCompile_DeprecatedWarningContainsDetails verifies deprecation warning
// includes standard number and superseded_by reference. (CLM-017)
func TestPackCompile_DeprecatedWarningContainsDetails(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-003-test-deprecated.standard.md": testStandardDeprecated("003", "STD-GO-004", "go"),
	})

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Fatal("expected deprecation warning")
	}

	warning := result.Warnings[0]
	if !strings.Contains(warning, "STD-GO-003") {
		t.Errorf("warning should contain standard number STD-GO-003, got: %s", warning)
	}
	if !strings.Contains(warning, "STD-GO-004") {
		t.Errorf("warning should contain superseded_by STD-GO-004, got: %s", warning)
	}
}

// TestPackCompile_IdempotentOutput verifies running compile twice on same
// input produces byte-identical output files. (CLM-018)
func TestPackCompile_IdempotentOutput(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-001-test-valid.standard.md": testStandardPattern("001", "go"),
	})

	outDir := filepath.Join(dir, ".backstop", "rules")

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     outDir,
		schemaSource:  testSchemaSource(t),
	}

	// First run
	result1, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}

	// Read first run output files
	firstOutputs := make(map[string][]byte)
	for _, std := range result1.Standards {
		for _, p := range std.OutputPaths {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("reading first output: %v", err)
			}
			firstOutputs[filepath.Base(p)] = data
		}
	}

	// Second run
	result2, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}

	// Compare outputs byte-for-byte
	for _, std := range result2.Standards {
		for _, p := range std.OutputPaths {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("reading second output: %v", err)
			}
			base := filepath.Base(p)
			first, ok := firstOutputs[base]
			if !ok {
				t.Errorf("file %s from second run not found in first run", base)
				continue
			}
			if !bytes.Equal(first, data) {
				t.Errorf("file %s differs between runs", base)
			}
		}
	}
}

// TestPackCompile_PartialCompilationOnMixedResults verifies that valid
// standards compile even when others fail. (CLM-023)
func TestPackCompile_PartialCompilationOnMixedResults(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-005-test-valid.standard.md":   testStandardPattern("005", "go"),
		"STD-GO-999-test-invalid.standard.md": testStandardInvalid("999"),
	})

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	// No error returned from runPackCompileWithOpts for partial failure —
	// the exit code is determined by the caller based on result.Summary.Failed
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Compiled != 1 {
		t.Errorf("expected 1 compiled, got %d", result.Summary.Compiled)
	}
	if result.Summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Summary.Failed)
	}

	// Verify the valid standard's output files exist
	outDir := filepath.Join(dir, ".backstop", "rules")
	manifestPath := filepath.Join(outDir, "STD-GO-005.manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("valid standard output should exist: %v", err)
	}
}

// TestPackCompile_ReportsAllFailures verifies all failures are reported
// when multiple standards fail. (CLM-024)
func TestPackCompile_ReportsAllFailures(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-997-test-invalid-a.standard.md": testStandardInvalid("997"),
		"STD-GO-998-test-invalid-b.standard.md": testStandardInvalid("998"),
		"STD-GO-005-test-valid.standard.md":     testStandardPattern("005", "go"),
	})

	opts := packCompileOpts{
		projectRoot:   dir,
		standardsDirs: []string{"standards/"},
		outputDir:     filepath.Join(dir, ".backstop", "rules"),
		schemaSource:  testSchemaSource(t),
	}

	result, err := runPackCompileWithOpts(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", result.Summary.Failed)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 error entries, got %d: %v", len(result.Errors), result.Errors)
	}

	// Verify both failures are mentioned
	errStr := strings.Join(result.Errors, "\n")
	if !strings.Contains(errStr, "test-invalid-a") {
		t.Error("first failure not reported")
	}
	if !strings.Contains(errStr, "test-invalid-b") {
		t.Error("second failure not reported")
	}
}

// TestPackCompile_ExitCode1OnPartialFailure verifies exit code 1 when some
// standards succeed and others fail. (CLM-025)
func TestPackCompile_ExitCode1OnPartialFailure(t *testing.T) {
	dir := setupProjectWithStandards(t, map[string]string{
		"STD-GO-005-test-valid.standard.md":   testStandardPattern("005", "go"),
		"STD-GO-999-test-invalid.standard.md": testStandardInvalid("999"),
	})

	_, err := runPackCompileInDir(t, dir, false)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitViolations {
		t.Errorf("expected exit code %d, got %d", ExitViolations, exitErr.Code)
	}
}

// ---------------------------------------------------------------------------
// Phase 6: Formatting edge cases and error paths
// ---------------------------------------------------------------------------

// TestPackCompile_HumanOutputZeroStandards verifies human output formatting
// when no standards are compiled (zero total).
func TestPackCompile_HumanOutputZeroStandards(t *testing.T) {
	result := &PackCompileResult{
		Standards: []CompileStandardResult{},
		Warnings:  []string{},
		Errors:    []string{},
		Summary:   CompileSummary{Total: 0, Compiled: 0, Failed: 0},
	}

	out := formatPackCompileHuman(result)

	if !strings.Contains(out, "Compiled 0 standards") {
		t.Errorf("expected 'Compiled 0 standards' header, got: %s", out)
	}
	if !strings.Contains(out, "0 total, 0 compiled, 0 failed") {
		t.Errorf("expected zero summary, got: %s", out)
	}
	// Should not contain Warnings section when there are none
	if strings.Contains(out, "Warnings:") {
		t.Error("should not show Warnings section when there are none")
	}
}

// TestPackCompile_HumanOutputWithWarnings verifies human output includes the
// Warnings section when warnings are present.
func TestPackCompile_HumanOutputWithWarnings(t *testing.T) {
	result := &PackCompileResult{
		Standards: []CompileStandardResult{
			{
				Standard:    "STD-GO-001",
				SourceFile:  "/tmp/STD-GO-001.standard.md",
				OutputPaths: []string{"/tmp/rules/STD-GO-001.manifest.json"},
				Warnings:    []string{"standard is deprecated"},
			},
		},
		Warnings: []string{"standard is deprecated"},
		Errors:   []string{},
		Summary:  CompileSummary{Total: 1, Compiled: 1, Failed: 0},
	}

	out := formatPackCompileHuman(result)

	if !strings.Contains(out, "Warnings:") {
		t.Error("expected Warnings section in output")
	}
	if !strings.Contains(out, "standard is deprecated") {
		t.Errorf("expected deprecation warning text, got: %s", out)
	}
}

// TestPackCompile_HumanOutputWithFailures verifies human output shows FAIL
// lines for standards with errors and includes failure count in header.
func TestPackCompile_HumanOutputWithFailures(t *testing.T) {
	result := &PackCompileResult{
		Standards: []CompileStandardResult{
			{
				Standard:   "STD-GO-001",
				SourceFile: "/tmp/STD-GO-001.standard.md",
				Error:      "unsupported strategy",
			},
		},
		Warnings: []string{},
		Errors:   []string{"STD-GO-001: unsupported strategy"},
		Summary:  CompileSummary{Total: 1, Compiled: 0, Failed: 1},
	}

	out := formatPackCompileHuman(result)

	if !strings.Contains(out, "(1 failed)") {
		t.Errorf("expected '(1 failed)' in header, got: %s", out)
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("expected FAIL marker in output")
	}
	if !strings.Contains(out, "unsupported strategy") {
		t.Errorf("expected error message in output, got: %s", out)
	}
}

// TestPackCompile_JSONEnvelopeSchemaVersion verifies that JSON output wraps
// the result with schema_version via the envelope, while PackCompileResult
// itself does not carry schema_version (REQ-004).
func TestPackCompile_JSONEnvelopeSchemaVersion(t *testing.T) {
	result := &PackCompileResult{
		Standards: []CompileStandardResult{},
		Warnings:  []string{},
		Errors:    []string{},
		Summary:   CompileSummary{Total: 0, Compiled: 0, Failed: 0},
	}

	out := formatPackCompileResult(result, true)

	// Verify schema_version appears in JSON output
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	sv, ok := raw["schema_version"]
	if !ok {
		t.Fatal("expected schema_version in JSON output")
	}
	if sv != "cli/v1" {
		t.Errorf("expected schema_version cli/v1, got %v", sv)
	}

	// Verify PackCompileResult struct itself has no SchemaVersion field
	// by marshaling the result directly (not via envelope)
	directJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var directRaw map[string]interface{}
	if err := json.Unmarshal(directJSON, &directRaw); err != nil {
		t.Fatal(err)
	}
	if _, has := directRaw["schema_version"]; has {
		t.Error("PackCompileResult should not have schema_version field — it belongs in the envelope")
	}
}
