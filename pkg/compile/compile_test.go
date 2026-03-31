package compile_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/compile"
	"github.com/bmanson/backstop-core/pkg/schema"
)

type realSchemaSource struct {
	schemaPath    string
	artifactsRoot string
}

func (s realSchemaSource) LoadSchema(_, _ string) (*schema.Schema, error) {
	return schema.LoadArtifactSchema(s.schemaPath, s.artifactsRoot)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func testSchemaSource(t *testing.T) compile.SchemaSource {
	t.Helper()
	root := repoRoot(t)
	return realSchemaSource{
		schemaPath:    filepath.Join(root, "artifacts", "standard", "v1", "schema.json"),
		artifactsRoot: filepath.Join(root, "artifacts"),
	}
}

func writeTestStandard(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func testStandard(number, language, scope, status, rules string) string {
	languageLine := ""
	if language != "" {
		languageLine = "language: " + language + "\n"
	}

	return fmt.Sprintf(`---
title: Test Standard
number: %s
created: "2026-01-01"
status: %s
schema_version: standard/v1
pack: test
scope: %s
%srules:
%s
---

# Test Standard

## Overview

Test overview.

## Rules

Rules content.

## Examples

Examples content.
`, number, status, scope, languageLine, rules)
}

func rulePattern(id, name, severity, pattern, tier string, languages []string) string {
	tierLine := ""
	if tier != "" {
		tierLine = fmt.Sprintf("    compliance_tier: %s\n", tier)
	}
	languagesRule := ""
	languagesDetection := ""
	if len(languages) > 0 {
		languagesRule = "    languages:\n"
		languagesDetection = "      languages:\n"
		for _, l := range languages {
			languagesRule += fmt.Sprintf("      - %s\n", l)
			languagesDetection += fmt.Sprintf("        - %s\n", l)
		}
	}

	return fmt.Sprintf(`  - id: %s
    name: %s
    category: structure
    severity: %s
    description: %s description
%s%s    detection:
      strategy: pattern
      semgrep: "%s"
%s    fix: %s fix
`, id, name, severity, name, tierLine, languagesRule, pattern, languagesDetection, name)
}

func ruleRegex(id, name, severity, regex, tier string, languages []string) string {
	tierLine := ""
	if tier != "" {
		tierLine = fmt.Sprintf("    compliance_tier: %s\n", tier)
	}
	languagesRule := ""
	languagesDetection := ""
	if len(languages) > 0 {
		languagesRule = "    languages:\n"
		languagesDetection = "      languages:\n"
		for _, l := range languages {
			languagesRule += fmt.Sprintf("      - %s\n", l)
			languagesDetection += fmt.Sprintf("        - %s\n", l)
		}
	}

	return fmt.Sprintf(`  - id: %s
    name: %s
    category: naming
    severity: %s
    description: %s description
%s%s    detection:
      strategy: regex
      pattern: "%s"
%s    fix: %s fix
`, id, name, severity, name, tierLine, languagesRule, regex, languagesDetection, name)
}

func ruleMetric(id, name, severity, metric, op string, threshold any, tier string) string {
	tierLine := ""
	if tier != "" {
		tierLine = fmt.Sprintf("    compliance_tier: %s\n", tier)
	}
	return fmt.Sprintf(`  - id: %s
    name: %s
    category: performance
    severity: %s
    description: %s description
%s    detection:
      strategy: metric
      metric: %s
      operator: "%s"
      threshold: %v
      exclude:
        - "_test.go"
    fix: %s fix
`, id, name, severity, name, tierLine, metric, op, threshold, name)
}

func ruleDelegated(id, name, severity, tool, delegatedRule, tier string) string {
	tierLine := ""
	if tier != "" {
		tierLine = fmt.Sprintf("    compliance_tier: %s\n", tier)
	}
	return fmt.Sprintf(`  - id: %s
    name: %s
    category: structure
    severity: %s
    description: %s description
%s    detection:
      strategy: delegated
      enforced_by: "%s"
      rule: "%s"
`, id, name, severity, name, tierLine, tool, delegatedRule)
}

func ruleAdvisory(id, name string) string {
	return fmt.Sprintf(`  - id: %s
    name: %s
    category: documentation
    severity: info
    description: %s description
    detection:
      strategy: pattern
      note: "manual review"
`, id, name, name)
}

func compileStandard(t *testing.T, standardPath string, opts compile.CompileOptions) *compile.CompileResult {
	t.Helper()
	res, err := compile.Compile(standardPath, opts)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return res
}

func outputBytes(t *testing.T, paths []string) map[string][]byte {
	t.Helper()
	m := make(map[string][]byte, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", p, err)
		}
		m[filepath.Base(p)] = b
	}
	return m
}

func TestCompile_ParsesRulesFromStandard(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-001", "max-file-length", "error", "var $X = ...", "baseline", nil) +
		ruleMetric("GO-002", "max-func-length", "warning", "function_lines", ">", 60, "standard")
	path := writeTestStandard(t, dir, "STD-GO-001-parse.standard.md", testStandard("STD-GO-001", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if res.Manifest == nil || len(res.Manifest.Rules) != 2 {
		t.Fatalf("manifest rules = %d, want 2", len(res.Manifest.Rules))
	}
}

func TestCompile_PatternRuleEmitsSemgrep(t *testing.T) {
	dir := t.TempDir()
	pattern := "var $NAME = ..."
	rules := rulePattern("GO-003", "no-global-state", "error", pattern, "baseline", nil)
	path := writeTestStandard(t, dir, "STD-GO-003-pattern.standard.md", testStandard("STD-GO-003", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 1 || res.SemgrepRules[0].Pattern != pattern {
		t.Fatalf("semgrep pattern = %q, want %q", res.SemgrepRules[0].Pattern, pattern)
	}
}

func TestCompile_RegexRuleEmitsSemgrepRegex(t *testing.T) {
	dir := t.TempDir()
	regexYAML := "fmt\\\\.Println"
	wantRegex := "fmt\\.Println"
	rules := ruleRegex("GO-004", "no-println", "warning", regexYAML, "baseline", nil)
	path := writeTestStandard(t, dir, "STD-GO-004-regex.standard.md", testStandard("STD-GO-004", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 1 || res.SemgrepRules[0].PatternRegex != wantRegex {
		t.Fatalf("semgrep regex = %q, want %q", res.SemgrepRules[0].PatternRegex, wantRegex)
	}
}

func TestCompile_MetricRuleEmitsNativeCheck(t *testing.T) {
	dir := t.TempDir()
	rules := ruleMetric("GO-005", "max-file-lines", "error", "file_lines", ">", 500, "baseline")
	path := writeTestStandard(t, dir, "STD-GO-005-metric.standard.md", testStandard("STD-GO-005", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.NativeChecks) != 1 {
		t.Fatalf("native checks len = %d, want 1", len(res.NativeChecks))
	}
	got := res.NativeChecks[0]
	if got.Metric != "file_lines" || got.Operator != ">" || fmt.Sprintf("%v", got.Threshold) != "500" {
		t.Fatalf("native check mismatch: %+v", got)
	}
	if got.Language != "go" {
		t.Fatalf("native check Language = %q, want %q (should inherit from standard)", got.Language, "go")
	}
}

func TestCompile_InvalidStandardReturnsViolations(t *testing.T) {
	dir := t.TempDir()
	content := `---
title: Invalid Standard
created: "2026-01-01"
status: active
schema_version: standard/v1
pack: test
scope: language
language: go
rules:
  - id: GO-001
    name: bad
    category: structure
    severity: error
    description: bad
    detection:
      strategy: pattern
      semgrep: "x"
---
# Invalid Standard
## Overview
x
## Rules
x
## Examples
x
`
	path := writeTestStandard(t, dir, "STD-GO-006-invalid.standard.md", content)
	outDir := filepath.Join(dir, "out")
	res, err := compile.Compile(path, compile.CompileOptions{OutputDir: outDir, SchemaSource: testSchemaSource(t)})
	if err == nil {
		t.Fatal("expected error")
	}
	if res != nil {
		t.Fatalf("expected nil result on validation error")
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "STD-GO-006.manifest.json")); !os.IsNotExist(statErr) {
		t.Fatalf("manifest should not exist, stat err = %v", statErr)
	}
}

func TestCompile_OutputDirectoryConfigurable(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "custom-out")
	rules := rulePattern("GO-007", "configurable-output", "error", "x", "baseline", nil)
	path := writeTestStandard(t, dir, "STD-GO-007-output.standard.md", testStandard("STD-GO-007", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: outDir, SchemaSource: testSchemaSource(t)})
	if len(res.OutputPaths) == 0 {
		t.Fatal("expected output paths")
	}
	for _, p := range res.OutputPaths {
		if !strings.HasPrefix(p, outDir+string(os.PathSeparator)) && p != outDir {
			t.Fatalf("path %q not in output dir %q", p, outDir)
		}
	}
}

func TestCompile_OutputDirectoryDefault(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	rules := rulePattern("GO-008", "default-output", "error", "x", "baseline", nil)
	path := writeTestStandard(t, dir, "STD-GO-008-default.standard.md", testStandard("STD-GO-008", "go", "language", "active", rules))
	res := compileStandard(t, path, compile.CompileOptions{SchemaSource: testSchemaSource(t)})

	wantManifest := filepath.Join(".backstop", "rules", "STD-GO-008.manifest.json")
	found := false
	for _, p := range res.OutputPaths {
		if filepath.Clean(p) == filepath.Clean(wantManifest) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected manifest path %q in output paths: %v", wantManifest, res.OutputPaths)
	}
}

func TestCompile_ManifestContainsAllRules(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-009", "pattern-rule", "error", "x", "baseline", nil) +
		ruleRegex("GO-010", "regex-rule", "warning", "x+", "baseline", nil) +
		ruleMetric("GO-011", "metric-rule", "error", "file_lines", ">", 500, "baseline") +
		ruleDelegated("GO-012", "delegated-rule", "warning", "golangci-lint", "revive/exported", "baseline")
	path := writeTestStandard(t, dir, "STD-GO-009-manifest-all.standard.md", testStandard("STD-GO-009", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Manifest.Rules) != 4 {
		t.Fatalf("manifest rules len = %d, want 4", len(res.Manifest.Rules))
	}
}

func TestCompile_ManifestEnforcementMethods(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-013", "pattern-rule", "error", "x", "baseline", nil) +
		ruleRegex("GO-014", "regex-rule", "warning", "x+", "baseline", nil) +
		ruleMetric("GO-015", "metric-rule", "error", "file_lines", ">", 500, "baseline") +
		ruleDelegated("GO-016", "delegated-rule", "warning", "golangci-lint", "revive/exported", "baseline")
	path := writeTestStandard(t, dir, "STD-GO-010-manifest-enforcement.standard.md", testStandard("STD-GO-010", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	got := map[string]string{}
	for _, mr := range res.Manifest.Rules {
		got[mr.ID] = mr.Enforcement
	}
	if got["GO-013"] != "semgrep" || got["GO-014"] != "semgrep" || got["GO-015"] != "native" || got["GO-016"] != "delegated" {
		t.Fatalf("unexpected enforcement mapping: %#v", got)
	}
}

func TestCompile_OutputFilenameFromStandardNumber(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-017", "pattern-rule", "error", "x", "baseline", nil) +
		ruleMetric("GO-018", "metric-rule", "error", "file_lines", ">", 500, "baseline")
	path := writeTestStandard(t, dir, "STD-GO-001-filenames.standard.md", testStandard("STD-GO-001", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	bases := make([]string, 0, len(res.OutputPaths))
	for _, p := range res.OutputPaths {
		bases = append(bases, filepath.Base(p))
	}
	sort.Strings(bases)
	want := []string{"STD-GO-001.manifest.json", "STD-GO-001.native.json", "STD-GO-001.semgrep.yml"}
	if strings.Join(bases, ",") != strings.Join(want, ",") {
		t.Fatalf("output files = %v, want %v", bases, want)
	}
}

func TestCompile_Idempotent(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	rules := rulePattern("GO-019", "idempotent-pattern", "error", "x", "baseline", nil) +
		ruleMetric("GO-020", "idempotent-metric", "warning", "file_lines", ">", 300, "baseline")
	path := writeTestStandard(t, dir, "STD-GO-011-idempotent.standard.md", testStandard("STD-GO-011", "go", "language", "active", rules))

	first := compileStandard(t, path, compile.CompileOptions{OutputDir: outDir, SchemaSource: testSchemaSource(t)})
	firstBytes := outputBytes(t, first.OutputPaths)
	second := compileStandard(t, path, compile.CompileOptions{OutputDir: outDir, SchemaSource: testSchemaSource(t)})
	secondBytes := outputBytes(t, second.OutputPaths)

	if len(firstBytes) != len(secondBytes) {
		t.Fatalf("output count mismatch")
	}
	for name, b1 := range firstBytes {
		b2 := secondBytes[name]
		if string(b1) != string(b2) {
			t.Fatalf("file %s differs between runs", name)
		}
	}
}

func TestCompile_DelegatedRulesInManifest(t *testing.T) {
	dir := t.TempDir()
	rules := ruleDelegated("GO-021", "delegated-rule", "warning", "golangci-lint", "revive/exported", "baseline")
	path := writeTestStandard(t, dir, "STD-GO-012-delegated.standard.md", testStandard("STD-GO-012", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Manifest.Rules) != 1 || res.Manifest.Rules[0].DelegatedTo == nil {
		t.Fatalf("expected delegated target in manifest")
	}
	d := res.Manifest.Rules[0].DelegatedTo
	if d.Tool != "golangci-lint" || d.Rule != "revive/exported" {
		t.Fatalf("delegated target = %+v", d)
	}
}

func TestCompile_DelegatedRulesNotInSemgrep(t *testing.T) {
	dir := t.TempDir()
	rules := ruleDelegated("GO-022", "delegated-rule", "warning", "golangci-lint", "revive/exported", "baseline")
	path := writeTestStandard(t, dir, "STD-GO-013-delegated-no-semgrep.standard.md", testStandard("STD-GO-013", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 0 {
		t.Fatalf("expected no semgrep rules, got %d", len(res.SemgrepRules))
	}
}

func TestCompile_AdvisoryRulesExcluded(t *testing.T) {
	dir := t.TempDir()
	rules := ruleAdvisory("GO-023", "advisory-only")
	path := writeTestStandard(t, dir, "STD-GO-014-advisory.standard.md", testStandard("STD-GO-014", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 0 {
		t.Fatalf("expected no semgrep rules")
	}
}

func TestCompile_AdvisoryRulesExcludedFromManifest(t *testing.T) {
	dir := t.TempDir()
	rules := ruleAdvisory("GO-024", "advisory-only")
	path := writeTestStandard(t, dir, "STD-GO-015-advisory-manifest.standard.md", testStandard("STD-GO-015", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Manifest.Rules) != 0 {
		t.Fatalf("expected advisory rule excluded from manifest")
	}
}

func TestCompile_AdvisoryDelegatedWithoutEnforcedBy(t *testing.T) {
	dir := t.TempDir()
	rules := `  - id: GO-025
    name: delegated-advisory
    category: structure
    severity: warning
    description: delegated advisory
    detection:
      strategy: delegated
      note: "handled manually"
      enforced_by: ""
      rule: ""
`
	path := writeTestStandard(t, dir, "STD-GO-016-advisory-delegated.standard.md", testStandard("STD-GO-016", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Manifest.Rules) != 0 || len(res.SemgrepRules) != 0 || len(res.NativeChecks) != 0 {
		t.Fatalf("expected delegated advisory excluded from all outputs")
	}
}

func TestCompile_ManifestIncludesComplianceTier(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-026", "tiered-rule", "error", "x", "strict", nil)
	path := writeTestStandard(t, dir, "STD-GO-017-tier.standard.md", testStandard("STD-GO-017", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Manifest.Rules) != 1 || res.Manifest.Rules[0].ComplianceTier != "strict" {
		t.Fatalf("compliance tier = %q, want strict", res.Manifest.Rules[0].ComplianceTier)
	}
}

func TestCompile_ManifestTierDefaultsToBaseline(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-027", "default-tier", "error", "x", "", nil)
	path := writeTestStandard(t, dir, "STD-GO-018-tier-default.standard.md", testStandard("STD-GO-018", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Manifest.Rules) != 1 || res.Manifest.Rules[0].EffectiveTier() != "baseline" {
		t.Fatalf("effective tier = %q, want baseline", res.Manifest.Rules[0].EffectiveTier())
	}
}

func TestCompile_DeprecatedStandardEmitsWarning(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-028", "deprecated-rule", "warning", "x", "baseline", nil)
	content := testStandard("STD-GO-019", "go", "language", "deprecated", rules)
	content = strings.Replace(content, "rules:\n", "superseded_by: STD-GO-999\nrules:\n", 1)
	path := writeTestStandard(t, dir, "STD-GO-019-deprecated.standard.md", content)

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Warnings) == 0 {
		t.Fatal("expected warning for deprecated standard")
	}
}

func TestCompile_DeprecatedWarningContainsStandardNumber(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-029", "deprecated-rule", "warning", "x", "baseline", nil)
	path := writeTestStandard(t, dir, "STD-GO-020-deprecated.standard.md", testStandard("STD-GO-020", "go", "language", "deprecated", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "STD-GO-020") {
		t.Fatalf("warning does not contain standard number: %v", res.Warnings)
	}
}

func TestCompile_DeprecatedStandardStillProducesOutput(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-030", "deprecated-rule", "warning", "x", "baseline", nil)
	path := writeTestStandard(t, dir, "STD-GO-021-deprecated-output.standard.md", testStandard("STD-GO-021", "go", "language", "deprecated", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if res.Manifest == nil {
		t.Fatal("expected manifest for deprecated standard")
	}
}

func TestCompile_UniversalPatternRuleUsesPerRuleLanguages(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("CORE-001", "universal-pattern", "error", "x", "baseline", []string{"go"})
	path := writeTestStandard(t, dir, "STD-CORE-001-universal-pattern.standard.md", testStandard("STD-CORE-001", "", "universal", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 1 || strings.Join(res.SemgrepRules[0].Languages, ",") != "go" {
		t.Fatalf("languages = %v, want [go]", res.SemgrepRules[0].Languages)
	}
}

func TestCompile_UniversalRegexRuleUsesPerRuleLanguages(t *testing.T) {
	dir := t.TempDir()
	rules := ruleRegex("CORE-002", "universal-regex", "warning", "x+", "baseline", []string{"python"})
	path := writeTestStandard(t, dir, "STD-CORE-002-universal-regex.standard.md", testStandard("STD-CORE-002", "", "universal", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 1 || strings.Join(res.SemgrepRules[0].Languages, ",") != "python" {
		t.Fatalf("languages = %v, want [python]", res.SemgrepRules[0].Languages)
	}
}

func TestCompile_UniversalMetricRuleNoLanguageRequired(t *testing.T) {
	dir := t.TempDir()
	rules := ruleMetric("CORE-003", "universal-metric", "error", "file_lines", ">", 500, "baseline")
	path := writeTestStandard(t, dir, "STD-CORE-003-universal-metric.standard.md", testStandard("STD-CORE-003", "", "universal", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.NativeChecks) != 1 {
		t.Fatalf("expected native check, got %d", len(res.NativeChecks))
	}
}

func TestCompile_UniversalPatternWithoutLanguagesFails(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("CORE-004", "universal-pattern-no-langs", "error", "x", "baseline", nil)
	path := writeTestStandard(t, dir, "STD-CORE-004-universal-no-langs.standard.md", testStandard("STD-CORE-004", "", "universal", "active", rules))

	res, err := compile.Compile(path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if err == nil {
		t.Fatal("expected error for missing languages")
	}
	if res != nil {
		t.Fatal("expected nil result on failure")
	}
}

func TestCompile_DefaultSchemaResolution(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	rules := rulePattern("GO-031", "default-schema", "error", "x", "baseline", nil)
	path := writeTestStandard(t, dir, "STD-GO-031-default-schema.standard.md", testStandard("STD-GO-031", "go", "language", "active", rules))

	res, err := compile.Compile(path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if res.Manifest == nil {
		t.Fatal("expected manifest")
	}
}

func TestCompile_DuplicateRuleIDs(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-032", "dup-a", "error", "x", "baseline", nil) +
		ruleMetric("GO-032", "dup-b", "warning", "file_lines", ">", 100, "baseline")
	path := writeTestStandard(t, dir, "STD-GO-032-duplicate.standard.md", testStandard("STD-GO-032", "go", "language", "active", rules))

	res, err := compile.Compile(path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if err == nil {
		t.Fatal("expected duplicate ID error")
	}
	if res != nil {
		t.Fatal("expected nil result")
	}
}

func TestCompile_AllDelegatedStandard(t *testing.T) {
	dir := t.TempDir()
	rules := ruleDelegated("GO-033", "delegated-a", "warning", "golangci-lint", "rule/a", "baseline") +
		ruleDelegated("GO-034", "delegated-b", "error", "golangci-lint", "rule/b", "strict")
	outDir := filepath.Join(dir, "out")
	path := writeTestStandard(t, dir, "STD-GO-033-all-delegated.standard.md", testStandard("STD-GO-033", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: outDir, SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 0 || len(res.NativeChecks) != 0 {
		t.Fatalf("expected no semgrep/native outputs")
	}
	if _, err := os.Stat(filepath.Join(outDir, "STD-GO-033.semgrep.yml")); !os.IsNotExist(err) {
		t.Fatalf("semgrep file should not exist")
	}
	if _, err := os.Stat(filepath.Join(outDir, "STD-GO-033.native.json")); !os.IsNotExist(err) {
		t.Fatalf("native file should not exist")
	}
}

func TestCompile_MixedStrategiesStandard(t *testing.T) {
	dir := t.TempDir()
	rules := rulePattern("GO-035", "mix-pattern", "error", "x", "baseline", nil) +
		ruleRegex("GO-036", "mix-regex", "warning", "x+", "standard", nil) +
		ruleMetric("GO-037", "mix-metric", "error", "file_lines", ">", 400, "strict") +
		ruleDelegated("GO-038", "mix-delegated", "warning", "golangci-lint", "revive/exported", "baseline") +
		ruleAdvisory("GO-039", "mix-advisory")
	path := writeTestStandard(t, dir, "STD-GO-034-mixed.standard.md", testStandard("STD-GO-034", "go", "language", "active", rules))

	res := compileStandard(t, path, compile.CompileOptions{OutputDir: filepath.Join(dir, "out"), SchemaSource: testSchemaSource(t)})
	if len(res.SemgrepRules) != 2 {
		t.Fatalf("semgrep rules len = %d, want 2", len(res.SemgrepRules))
	}
	if len(res.NativeChecks) != 1 {
		t.Fatalf("native checks len = %d, want 1", len(res.NativeChecks))
	}
	if len(res.Manifest.Rules) != 4 {
		t.Fatalf("manifest rules len = %d, want 4", len(res.Manifest.Rules))
	}

	manifestPath := filepath.Join(dir, "out", "STD-GO-034.manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("manifest json invalid: %v", err)
	}
}

func TestCompile_BadSchemaVersionWithSource(t *testing.T) {
	dir := t.TempDir()
	content := testStandard("STD-TEST-001", "go", "language", "active", `  - id: T-001
    name: test
    category: testing
    severity: error
    description: test
    detection:
      strategy: pattern
      semgrep: "foo"
    fix: bar`)
	// Replace schema_version with invalid value
	content = strings.Replace(content, "schema_version: standard/v1", "schema_version: invalid", 1)
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error for bad schema_version")
	}
	if !strings.Contains(err.Error(), "invalid schema_version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompile_ParseError(t *testing.T) {
	_, err := compile.Compile("/nonexistent/file.standard.md", compile.CompileOptions{
		OutputDir:    t.TempDir(),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCompile_RulesNotArray(t *testing.T) {
	dir := t.TempDir()
	content := `---
title: Bad Rules
number: STD-TEST-001
created: "2026-01-01"
status: active
schema_version: standard/v1
language: go
scope: language
rules: "not an array"
---

# Bad Rules

## Rationale

Test.

## Primary Sources

Test.
`
	path := writeTestStandard(t, dir, "STD-TEST-001-bad.standard.md", content)
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error for non-array rules")
	}
}

func TestCompile_MissingRules(t *testing.T) {
	dir := t.TempDir()
	content := `---
title: No Rules
number: STD-TEST-001
created: "2026-01-01"
status: active
schema_version: standard/v1
language: go
scope: language
---

# No Rules

## Rationale

Test.

## Primary Sources

Test.
`
	path := writeTestStandard(t, dir, "STD-TEST-001-norules.standard.md", content)
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error for missing rules")
	}
}

type failingSchemaSource struct{}

func (f failingSchemaSource) LoadSchema(_, _ string) (*schema.Schema, error) {
	return nil, fmt.Errorf("schema not found")
}

func TestCompile_FailingSchemaSource(t *testing.T) {
	dir := t.TempDir()
	content := testStandard("STD-TEST-001", "go", "language", "active",
		rulePattern("T-001", "test", "error", "foo", "baseline", nil))
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: failingSchemaSource{},
	})
	if err == nil {
		t.Fatal("expected error from failing schema source")
	}
	if !strings.Contains(err.Error(), "schema not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompile_UnsupportedStrategy(t *testing.T) {
	// The validator rejects unknown strategies before the compiler routes them.
	// This test verifies the validator catches it (the compile.go default branch
	// is a defensive guard that can't be reached through normal flow).
	dir := t.TempDir()
	rules := `  - id: T-001
    name: test
    category: testing
    severity: error
    description: test desc
    detection:
      strategy: unknown_strategy
    fix: fix it`
	content := testStandard("STD-TEST-001", "go", "language", "active", rules)
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error for unsupported strategy")
	}
}

func TestCompile_UnwritableOutputDir(t *testing.T) {
	dir := t.TempDir()
	content := testStandard("STD-TEST-001", "go", "language", "active",
		rulePattern("T-001", "test", "error", "foo", "baseline", nil))
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)

	unwritable := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(unwritable, 0o555); err != nil {
		t.Fatal(err)
	}
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(unwritable, "nested", "deep"),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error for unwritable output directory")
	}
}

func TestCompile_DeprecatedWithSupersededBy(t *testing.T) {
	dir := t.TempDir()
	rules := ruleMetric("T-001", "test", "error", "file_lines", ">", 500, "baseline")
	content := fmt.Sprintf(`---
title: Test Standard
number: STD-TEST-001
created: "2026-01-01"
status: deprecated
schema_version: standard/v1
language: go
pack: test
scope: language
superseded_by: STD-TEST-002
rules:
%s
---

# Test Standard

## Overview

Test overview.

## Rules

Rules content.

## Examples

Examples content.
`, rules)
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)
	result, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: testSchemaSource(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warnings for deprecated standard")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "STD-TEST-002") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning to contain superseded_by, got: %v", result.Warnings)
	}
}

func TestCompile_RuleItemNotObject(t *testing.T) {
	dir := t.TempDir()
	content := `---
title: Bad Rule Item
number: STD-TEST-001
created: "2026-01-01"
status: active
schema_version: standard/v1
language: go
pack: test
scope: language
rules:
  - "just a string, not an object"
---

# Bad Rule Item

## Overview

Test overview.

## Rules

Rules content.

## Examples

Examples content.
`
	path := writeTestStandard(t, dir, "STD-TEST-001-badrule.standard.md", content)
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error for non-object rule item")
	}
}

func TestCompile_NonUniversalEmptyLanguage(t *testing.T) {
	dir := t.TempDir()
	content := testStandard("STD-TEST-001", "", "language", "active",
		rulePattern("T-001", "test", "error", "foo", "baseline", nil))
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error when language scope has no language")
	}
}

func TestCompile_UnwritableManifestFile(t *testing.T) {
	dir := t.TempDir()
	content := testStandard("STD-TEST-001", "go", "language", "active",
		rulePattern("T-001", "test", "error", "foo", "baseline", nil))
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-create the manifest as a directory so WriteFile fails
	if err := os.MkdirAll(filepath.Join(outDir, "STD-TEST-001.manifest.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    outDir,
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error writing manifest to directory path")
	}
}

func TestCompile_UnwritableSemgrepFile(t *testing.T) {
	dir := t.TempDir()
	content := testStandard("STD-TEST-001", "go", "language", "active",
		rulePattern("T-001", "test", "error", "foo", "baseline", nil))
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-create semgrep file as a directory so WriteSemgrepFile fails
	if err := os.MkdirAll(filepath.Join(outDir, "STD-TEST-001.semgrep.yml"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    outDir,
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error writing semgrep to directory path")
	}
}

func TestCompile_UnwritableNativeFile(t *testing.T) {
	dir := t.TempDir()
	content := testStandard("STD-TEST-001", "go", "language", "active",
		ruleMetric("T-001", "test", "error", "file_lines", ">", 500, "baseline"))
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-create native file as a directory so WriteNativeChecksFile fails
	if err := os.MkdirAll(filepath.Join(outDir, "STD-TEST-001.native.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    outDir,
		SchemaSource: testSchemaSource(t),
	})
	if err == nil {
		t.Fatal("expected error writing native checks to directory path")
	}
}

func TestCompile_DefaultSchemaResolutionBadPath(t *testing.T) {
	dir := t.TempDir()
	// Standard with schema_version that won't resolve on filesystem
	content := testStandard("STD-TEST-001", "go", "language", "active",
		rulePattern("T-001", "test", "error", "foo", "baseline", nil))
	path := writeTestStandard(t, dir, "STD-TEST-001-test.standard.md", content)

	// Use nil SchemaSource so it falls back to filesystem resolution,
	// but we're in a temp dir with no artifacts/ directory
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := compile.Compile(path, compile.CompileOptions{
		OutputDir: filepath.Join(dir, "out"),
	})
	if err == nil {
		t.Fatal("expected error from filesystem schema resolution in wrong directory")
	}
}
