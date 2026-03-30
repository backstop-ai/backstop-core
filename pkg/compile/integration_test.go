package compile_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/compile"
	"github.com/bmanson/backstop-core/pkg/schema"
	"gopkg.in/yaml.v3"
)

type integrationSchemaSource struct {
	statusEnum []string
}

func (s integrationSchemaSource) LoadSchema(artifactType, version string) (*schema.Schema, error) {
	statuses := s.statusEnum
	if len(statuses) == 0 {
		statuses = []string{"draft", "active", "deprecated", "retired"}
	}

	return &schema.Schema{
		ArtifactType:     artifactType,
		FilenamePattern:  `STD-[A-Z]+-\d{3}-.+\.standard\.md`,
		RequiredMetadata: []string{"title", "number", "created", "status", "schema_version"},
		StatusEnum:       statuses,
		RequiredSections: []string{"Rationale", "Primary Sources"},
	}, nil
}

func integrationTestStandard(number, language, scope, status, extra, rules string) string {
	languageLine := ""
	if language != "" {
		languageLine = fmt.Sprintf("language: %s\n", language)
	}

	return fmt.Sprintf(`---
title: Test Standard
number: %s
created: "2026-01-01"
status: %s
schema_version: standard/v1
pack: test
%s
scope: %s
%srules:
%s
---

# Test Standard

## Rationale

Test rationale.

## Primary Sources

Test sources.
`, number, status, languageLine, scope, extra, rules)
}

func TestIntegration_CompileRealGoStandard(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "standards", "go", "STD-GO-001-go-code-standards.standard.md")
	if _, err := os.Stat(path); err != nil {
		t.Skip("real standard not found")
	}

	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	tmpDir := t.TempDir()
	res, err := compile.Compile(path, compile.CompileOptions{OutputDir: tmpDir})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if res == nil || res.Manifest == nil {
		t.Fatal("expected compile result and manifest")
	}

	if res.Manifest.Standard != "STD-GO-001" {
		t.Fatalf("manifest standard = %q, want STD-GO-001", res.Manifest.Standard)
	}
	if res.Manifest.Language != "go" {
		t.Fatalf("manifest language = %q, want go", res.Manifest.Language)
	}
	if len(res.Manifest.Rules) == 0 {
		t.Fatal("expected manifest to include rules")
	}
	if res.Manifest.SemgrepConfig == "" {
		t.Fatal("expected semgrep config filename")
	}
	if res.Manifest.NativeChecksFile == "" {
		t.Fatal("expected native checks filename")
	}

	hasDelegated := false
	for _, rule := range res.Manifest.Rules {
		if strings.TrimSpace(rule.ComplianceTier) == "" {
			t.Fatalf("rule %s has empty compliance tier", rule.ID)
		}
		if rule.Enforcement == "delegated" && rule.DelegatedTo != nil && strings.TrimSpace(rule.DelegatedTo.Tool) != "" {
			hasDelegated = true
		}
	}
	if !hasDelegated {
		t.Fatal("expected at least one delegated rule with delegated tool")
	}

	manifestPath := filepath.Join(tmpDir, "STD-GO-001.manifest.json")
	semgrepPath := filepath.Join(tmpDir, res.Manifest.SemgrepConfig)
	nativePath := filepath.Join(tmpDir, res.Manifest.NativeChecksFile)

	for _, p := range []string{manifestPath, semgrepPath, nativePath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected output file %q to exist: %v", p, err)
		}
	}

	semgrepData, err := os.ReadFile(semgrepPath)
	if err != nil {
		t.Fatalf("read semgrep file: %v", err)
	}
	var semgrepDoc map[string]any
	if err := yaml.Unmarshal(semgrepData, &semgrepDoc); err != nil {
		t.Fatalf("semgrep yaml invalid: %v", err)
	}

	nativeData, err := os.ReadFile(nativePath)
	if err != nil {
		t.Fatalf("read native checks file: %v", err)
	}
	var nativeDoc map[string]any
	if err := json.Unmarshal(nativeData, &nativeDoc); err != nil {
		t.Fatalf("native json invalid: %v", err)
	}
}

func TestIntegration_Idempotency(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "standards", "go", "STD-GO-001-go-code-standards.standard.md")
	if _, err := os.Stat(path); err != nil {
		t.Skip("real standard not found")
	}

	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	outA := t.TempDir()
	outB := t.TempDir()

	first, err := compile.Compile(path, compile.CompileOptions{OutputDir: outA})
	if err != nil {
		t.Fatalf("first compile error: %v", err)
	}
	second, err := compile.Compile(path, compile.CompileOptions{OutputDir: outB})
	if err != nil {
		t.Fatalf("second compile error: %v", err)
	}

	if first.Manifest == nil || second.Manifest == nil {
		t.Fatal("expected manifests for both compile runs")
	}

	filesA := []string{
		filepath.Join(outA, fmt.Sprintf("%s.manifest.json", first.Manifest.Standard)),
		filepath.Join(outA, first.Manifest.SemgrepConfig),
		filepath.Join(outA, first.Manifest.NativeChecksFile),
	}
	filesB := []string{
		filepath.Join(outB, fmt.Sprintf("%s.manifest.json", second.Manifest.Standard)),
		filepath.Join(outB, second.Manifest.SemgrepConfig),
		filepath.Join(outB, second.Manifest.NativeChecksFile),
	}

	for i := range filesA {
		a, err := os.ReadFile(filesA[i])
		if err != nil {
			t.Fatalf("read first output %q: %v", filesA[i], err)
		}
		b, err := os.ReadFile(filesB[i])
		if err != nil {
			t.Fatalf("read second output %q: %v", filesB[i], err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("output files differ: %s vs %s", filepath.Base(filesA[i]), filepath.Base(filesB[i]))
		}
	}
}

func TestIntegration_UniversalStandard(t *testing.T) {
	dir := t.TempDir()
	rules := `  - id: UNI-001
    name: universal-pattern
    category: testing
    severity: warning
    description: A universal pattern rule
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: "dangerous_function(...)"
      languages:
        - go
    languages:
      - go
    fix: Remove dangerous function
`

	content := integrationTestStandard("STD-UNI-001", "", "universal", "active", "", rules)
	path := writeTestStandard(t, dir, "STD-UNI-001-universal.standard.md", content)

	res, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: integrationSchemaSource{},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(res.SemgrepRules) != 1 {
		t.Fatalf("semgrep rules len = %d, want 1", len(res.SemgrepRules))
	}
	if len(res.SemgrepRules[0].Languages) != 1 || res.SemgrepRules[0].Languages[0] != "go" {
		t.Fatalf("semgrep languages = %v, want [go]", res.SemgrepRules[0].Languages)
	}
}

func TestIntegration_DeprecatedStandard(t *testing.T) {
	dir := t.TempDir()
	rules := `  - id: GO-901
    name: deprecated-metric
    category: performance
    severity: warning
    description: Metric rule for deprecated standard
    compliance_tier: baseline
    detection:
      strategy: metric
      metric: file_lines
      operator: ">"
      threshold: 1000
    fix: Split large files
`
	content := integrationTestStandard("STD-GO-901", "go", "language", "deprecated", "superseded_by: STD-GO-002\n", rules)
	path := writeTestStandard(t, dir, "STD-GO-901-deprecated.standard.md", content)

	res, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    filepath.Join(dir, "out"),
		SchemaSource: integrationSchemaSource{statusEnum: []string{"draft", "active", "deprecated", "retired"}},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected deprecation warning")
	}
	warning := strings.Join(res.Warnings, " ")
	if !strings.Contains(warning, "STD-GO-901") || !strings.Contains(warning, "STD-GO-002") {
		t.Fatalf("warning %q must contain standard and superseded_by", warning)
	}
}

func TestIntegration_AllDelegatedStandard(t *testing.T) {
	dir := t.TempDir()
	rules := `  - id: GO-951
    name: delegated-one
    category: structure
    severity: warning
    description: Delegated rule one
    compliance_tier: baseline
    detection:
      strategy: delegated
      enforced_by: golangci-lint
      rule: revive/exported
  - id: GO-952
    name: delegated-two
    category: testing
    severity: error
    description: Delegated rule two
    compliance_tier: strict
    detection:
      strategy: delegated
      enforced_by: custom-checker
      rule: checker/rule-two
`
	content := integrationTestStandard("STD-GO-950", "go", "language", "active", "", rules)
	outDir := filepath.Join(dir, "out")
	path := writeTestStandard(t, dir, "STD-GO-950-all-delegated.standard.md", content)

	res, err := compile.Compile(path, compile.CompileOptions{
		OutputDir:    outDir,
		SchemaSource: integrationSchemaSource{},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(res.SemgrepRules) != 0 {
		t.Fatalf("semgrep rules len = %d, want 0", len(res.SemgrepRules))
	}
	if len(res.NativeChecks) != 0 {
		t.Fatalf("native checks len = %d, want 0", len(res.NativeChecks))
	}
	if res.Manifest.SemgrepConfig != "" {
		t.Fatalf("manifest semgrep config = %q, want empty", res.Manifest.SemgrepConfig)
	}
	if res.Manifest.NativeChecksFile != "" {
		t.Fatalf("manifest native checks file = %q, want empty", res.Manifest.NativeChecksFile)
	}
	if len(res.Manifest.Rules) == 0 {
		t.Fatal("expected delegated manifest rules")
	}

	if _, err := os.Stat(filepath.Join(outDir, "STD-GO-950.semgrep.yml")); !os.IsNotExist(err) {
		t.Fatalf("expected no semgrep file, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "STD-GO-950.native.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no native file, stat err = %v", err)
	}
}
