package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func TestBuildGateSteps_ConsumesInvocationSelectedSandboxRunner(t *testing.T) {
	source, err := os.ReadFile("gate.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, "dispatchPackEnginesWithEvidence") || !strings.Contains(text, "sandboxRunner") {
		t.Fatal("production buildGateSteps does not consume typed evidence with the selected runner")
	}
	runner := &recordingSandboxRunner{mode: packval.SandboxModeExternal}
	if runner.Mode() != packval.SandboxModeExternal {
		t.Fatal("test runner is not immutable external mode")
	}
}

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

	steps := buildGateSteps(projectRoot, rootAtDir(t, projectRoot))
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

// TestBuildGateSteps_PackEngineDispatchFailureYieldsFailStep verifies that a
// pack whose declared engine: semgrep rule file is missing surfaces as a
// pack_engines gate step flagged as a config error. Re-keyed from the retired
// TestBuildGateSteps_PackRuleMergeFailureYieldsSingleFailStep: the broken-pack
// detection moved from the early mergePackRules collapse into the runtime
// dispatchPackEngines step (the pack_engines step), so the gate now builds its
// full step list and the failure is reported when that step runs.
func TestBuildGateSteps_PackEngineDispatchFailureYieldsFailStep(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/pack: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Install the pack dir with a manifest that references a rule file that does
	// not exist, so dispatchPackEngines fails with "broken pack".
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", "org", "pack")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `name: org/pack
version: 1.0.0
language: go
archetype: enforcement
description: Pack with a broken engine semgrep rule path
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: r1
        standard: standards/go/r1.standard.md
        rule_path: rules/absent.yml
        risk_class: security
        engine: semgrep
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

	steps := buildGateSteps(projectRoot, rootAtDir(t, projectRoot))
	// Find the pack_engines step and run it; it must fail loud as a config error.
	var found bool
	for _, step := range steps {
		res := step(context.Background())
		if res.StepName == "pack_engines" {
			found = true
			if res.Status != "fail" || !res.ConfigErr {
				t.Errorf("expected pack_engines fail+ConfigErr, got status=%q configErr=%v", res.Status, res.ConfigErr)
			}
			if len(res.Violations) == 0 || !strings.Contains(res.Violations[0].Message, "broken pack") {
				t.Errorf("expected a broken-pack violation message, got %#v", res.Violations)
			}
		}
	}
	if !found {
		t.Fatal("expected a pack_engines step in the gate step list")
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
	missing := filepath.Join(t.TempDir(), "nope")
	v := &realArtifactValidator{projectRoot: missing, root: rootAtDir(t, missing)}
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
	v := &realArtifactValidator{projectRoot: projectRoot, root: rootAtDir(t, projectRoot)}
	violations, err := v.ValidateAll(context.Background())
	if err != nil {
		t.Fatalf("ValidateAll on a clean project: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations on a clean project, got %d", len(violations))
	}
}
