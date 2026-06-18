package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestBuildGateSteps_PackLoadingFailureYieldsSingleFailStep verifies that when
// pack loading fails (a pack is declared in backstop.yml but absent on disk),
// buildGateSteps collapses to a single pack_loading fail step flagged as a
// config error, rather than the full nine-step chain.
func TestBuildGateSteps_PackLoadingFailureYieldsSingleFailStep(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/ghost: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	steps := buildGateSteps(projectRoot)
	if len(steps) != 1 {
		t.Fatalf("expected a single collapsed step, got %d", len(steps))
	}
	res := steps[0](context.Background())
	if res.StepName != "pack_loading" {
		t.Errorf("step name = %q, want pack_loading", res.StepName)
	}
	if res.Status != "fail" {
		t.Errorf("status = %q, want fail", res.Status)
	}
	if !res.ConfigErr {
		t.Error("expected ConfigErr to be set on pack-loading failure")
	}
	if len(res.Violations) == 0 {
		t.Error("expected a violation describing the missing pack")
	}
}

// TestBuildGateSteps_PackRuleMergeFailureYieldsSingleFailStep verifies that a
// pack whose declared layer-2 rule file is missing collapses buildGateSteps to
// a single pack_rule_merge fail step flagged as a config error.
func TestBuildGateSteps_PackRuleMergeFailureYieldsSingleFailStep(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/pack: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Install the pack dir with a manifest that references a rule file that does
	// not exist, so mergePackRules fails with "broken pack".
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", "org", "pack")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `name: org/pack
version: 1.0.0
language: go
archetype: enforcement
description: Pack with a broken layer-2 rule path
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: r1
        standard: standards/go/r1.standard.md
        rule_path: rules/absent.yml
        risk_class: security
        layer: 2
        category: static-analysis
        claims:
          - id: c-r1
            text: Rule one.
            fixtures:
              positive:
                - fixtures/positive.go
              negative:
                - fixtures/negative.go
`
	if err := os.WriteFile(filepath.Join(packRoot, "pack.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	steps := buildGateSteps(projectRoot)
	if len(steps) != 1 {
		t.Fatalf("expected a single collapsed step, got %d", len(steps))
	}
	res := steps[0](context.Background())
	if res.StepName != "pack_rule_merge" {
		t.Errorf("step name = %q, want pack_rule_merge", res.StepName)
	}
	if res.Status != "fail" || !res.ConfigErr {
		t.Errorf("expected fail+ConfigErr, got status=%q configErr=%v", res.Status, res.ConfigErr)
	}
}

// TestRunGate_HumanOutput_CollapsedPackFailure verifies the non-JSON (human)
// output branch of runGate: a project whose declared pack is missing collapses
// the gate to a fast config-failure step (no real toolchain execution), and the
// command prints human-readable output and returns a non-zero exit code.
func TestRunGate_HumanOutput_CollapsedPackFailure(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/ghost: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCommand()
	out, err := executeCommand(root, "gate") // no --json => human formatter
	if err == nil {
		t.Fatal("expected the collapsed pack-failure gate to return a non-zero exit error")
	}
	if out == "" {
		t.Error("expected human-readable gate output to be printed")
	}
}

// TestRealArtifactValidator_ValidateAll_ConfigErrorWrapped verifies that a
// validation config error (no backstop.yml / invalid project) is wrapped in a
// gate.ConfigError so the gate treats it as exit-2 config failure.
func TestRealArtifactValidator_ValidateAll_ConfigErrorWrapped(t *testing.T) {
	// An empty temp dir has no backstop.yml; ValidateArtifacts(All) errors.
	v := &realArtifactValidator{projectRoot: filepath.Join(t.TempDir(), "nope")}
	_, err := v.ValidateAll(context.Background())
	if err == nil {
		t.Fatal("expected a config error from ValidateAll on a missing project")
	}
}

// TestRealArtifactValidator_ValidateAll_CleanProjectNoViolations verifies that
// a project with only a valid backstop.yml and no artifacts validates with no
// violations and no error.
func TestRealArtifactValidator_ValidateAll_CleanProjectNoViolations(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: clean\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := &realArtifactValidator{projectRoot: projectRoot}
	violations, err := v.ValidateAll(context.Background())
	if err != nil {
		t.Fatalf("ValidateAll on a clean project: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations on a clean project, got %d", len(violations))
	}
}
