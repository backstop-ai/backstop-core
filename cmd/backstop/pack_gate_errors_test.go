package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// nilRunner is a CommandRunner returning empty output; used where dispatch
// fails before (or independently of) running the engine command.
type nilRunner struct{}

func (nilRunner) Run(context.Context, string, ...string) ([]byte, error)       { return nil, nil }
func (nilRunner) RunStdout(context.Context, string, ...string) ([]byte, error) { return nil, nil }

// TestLoadInstalledPacks_MissingConfig verifies that a missing backstop.yml
// surfaces a wrapped "loading backstop.yml" error.
func TestLoadInstalledPacks_MissingConfig(t *testing.T) {
	_, err := loadInstalledPacks(t.TempDir()) // no backstop.yml
	if err == nil {
		t.Fatal("expected error loading packs without backstop.yml")
	}
	if !strings.Contains(err.Error(), "loading backstop.yml") {
		t.Errorf("error = %q, want it to mention loading backstop.yml", err.Error())
	}
}

// TestLoadInstalledPacks_DeclaredPackMissingFromDisk verifies that a pack
// declared in backstop.yml but absent from .backstop/packs yields a descriptive
// "missing from" error.
func TestLoadInstalledPacks_DeclaredPackMissingFromDisk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/ghost: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadInstalledPacks(dir)
	if err == nil {
		t.Fatal("expected error for declared-but-absent pack")
	}
	if !strings.Contains(err.Error(), "missing from") {
		t.Errorf("error = %q, want it to mention missing from", err.Error())
	}
}

// TestDispatchPackEngines_BrokenSemgrepRuleFile verifies that an engine:semgrep
// rule whose file is absent on disk yields a "broken pack" error. Re-keyed from
// TestMergePackRules_BrokenRuleFile onto dispatchPackEngines.
func TestDispatchPackEngines_BrokenSemgrepRuleFile(t *testing.T) {
	packDir := t.TempDir()
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r1", Engine: "semgrep", RulePath: "rules/absent.yml"},
		}}},
	}}
	_, err := dispatchPackEngines(manifests, packDir, t.TempDir(), nilRunner{})
	if err == nil {
		t.Fatal("expected error for missing semgrep rule file")
	}
	if !strings.Contains(err.Error(), "broken pack") {
		t.Errorf("error = %q, want it to mention broken pack", err.Error())
	}
}

// TestDispatchPackEngines_BrokenValidatorMissingFile verifies that an
// engine:sandbox rule whose validator script is absent yields a "broken pack"
// error naming the missing validator. Re-keyed from
// TestRunPackValidators_BrokenValidatorMissingFile onto the engine==sandbox
// exit-code branch.
func TestDispatchPackEngines_BrokenValidatorMissingFile(t *testing.T) {
	packDir := t.TempDir()
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "v1", Engine: "sandbox", Validator: "missing.sh"},
		}}},
	}}
	_, err := dispatchPackEngines(manifests, packDir, t.TempDir(), nilRunner{})
	if err == nil {
		t.Fatal("expected error for missing sandbox validator")
	}
	if !strings.Contains(err.Error(), "broken pack") {
		t.Errorf("error = %q, want it to mention broken pack", err.Error())
	}
}

// TestDispatchSandbox_EmptyOutputFallsBackToErrString verifies that when a
// failing sandbox validator produces no stdout/stderr, the violation message
// falls back to the runner error string rather than being empty. Re-keyed from
// TestRunPackValidators_EmptyOutputFallsBackToErrString.
func TestDispatchSandbox_EmptyOutputFallsBackToErrString(t *testing.T) {
	projectRoot := t.TempDir()
	packRootRel := filepath.Join("org", "pack")
	packsDir := filepath.Join(projectRoot, ".backstop", "packs")
	packRoot := filepath.Join(packsDir, packRootRel)
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	validator := filepath.Join(packRoot, "v.sh")
	if err := os.WriteFile(validator, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) {
		return []byte("   "), errors.New("exit status 7") // whitespace-only output
	}
	t.Cleanup(func() { sandboxedRun = orig })

	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "v1", Engine: "sandbox", Validator: "v.sh"},
		}}},
	}}
	violations, err := dispatchPackEngines(manifests, packsDir, projectRoot, nilRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Message != "exit status 7" {
		t.Errorf("message = %q, want the error string fallback %q", violations[0].Message, "exit status 7")
	}
	if violations[0].Severity != "error" {
		t.Errorf("severity = %q, want error", violations[0].Severity)
	}
}

// TestDispatchSandbox_PassingValidatorNoViolation verifies that an
// engine:sandbox validator that exits clean produces no violation (the continue
// branch). Re-keyed from TestRunPackValidators_PassingValidatorNoViolation.
func TestDispatchSandbox_PassingValidatorNoViolation(t *testing.T) {
	projectRoot := t.TempDir()
	packsDir := filepath.Join(projectRoot, ".backstop", "packs")
	packRoot := filepath.Join(packsDir, "org", "pack")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "v.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { sandboxedRun = orig })

	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "v1", Engine: "sandbox", Validator: "v.sh"},
		}}},
	}}
	violations, err := dispatchPackEngines(manifests, packsDir, projectRoot, nilRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations from a passing validator, got %d", len(violations))
	}
}
