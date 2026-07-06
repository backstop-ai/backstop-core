package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// TestGateViolationsToCheck / TestPackNamesFromManifests were removed with the
// `backstop code check` command (ISSUE-018): gateViolationsToCheck and
// packNamesFromManifests were code_check.go-only helpers deleted along with it.

func TestDeclaredPackNames_NilConfig(t *testing.T) {
	names := declaredPackNames(nil)
	if len(names) != 0 {
		t.Errorf("expected 0 names for nil config, got %d", len(names))
	}
}

func TestDeclaredPackNames_EmptyConfig(t *testing.T) {
	cfg := &config.Config{}
	names := declaredPackNames(cfg)
	if len(names) != 0 {
		t.Errorf("expected 0 names for empty config, got %d", len(names))
	}
}

func TestLoadInstalledPacks_NoPacks(t *testing.T) {
	dir := t.TempDir()
	// No backstop.yml — returns nil packs
	packs, err := loadInstalledPacks(dir)
	if err != nil {
		// Expected — no backstop.yml
		return
	}
	if len(packs) != 0 {
		t.Errorf("expected 0 packs, got %d", len(packs))
	}
}

func TestVerifyPackLock_NoPacks(t *testing.T) {
	dir := t.TempDir()
	// No packs — verify should pass or return nil error
	err := verifyPackLock(dir, nil)
	if err != nil {
		t.Errorf("expected no error for nil packs, got: %v", err)
	}
}

// TestDispatchPackEngines_NoPacks: dispatch over zero packs yields zero
// violations and no error. Re-keyed from TestMergePackRules_NoPacks (the
// findings-feeder no-packs case).
func TestDispatchPackEngines_NoPacks(t *testing.T) {
	violations, err := dispatchPackEngines(nil, t.TempDir(), t.TempDir(), nil, nilRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

// TestDispatchPackEngines_NoPacksValidators is the symmetric twin of
// TestDispatchPackEngines_NoPacks, re-keyed from the retired
// TestRunPackValidators_NoPacks (which carried no .Layer literal): dispatch over
// zero packs yields zero violations through the consolidated path that now folds
// in the sandbox validator branch.
func TestDispatchPackEngines_NoPacksValidators(t *testing.T) {
	violations, err := dispatchPackEngines(nil, t.TempDir(), t.TempDir(), nil, nilRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

// TestDispatchSandbox_SkipsNonSandboxEngines verifies the engine==sandbox
// exit-code branch does not emit validator violations for non-sandbox
// (config-file / semgrep) rules. Re-keyed from TestRunPackValidators_SkipsNonLayer3:
// the layer:1/layer:2 rules become engine: config-file / engine: semgrep, and
// the assertion is that the sandbox branch skips them (no validator violation).
func TestDispatchSandbox_SkipsNonSandboxEngines(t *testing.T) {
	packDir := t.TempDir()
	packRoot := filepath.Join(packDir, "test/pack")
	os.MkdirAll(filepath.Join(packRoot, "rules"), 0o755)
	os.WriteFile(filepath.Join(packRoot, "rules", "r2.yml"), []byte("rules: []"), 0o644)

	// Stub the semgrep engine runner so the engine: semgrep rule produces empty
	// SARIF (no findings) and the config-file rule injects nothing; neither
	// should reach the sandbox exit-code branch.
	manifests := []*pack.Manifest{{
		NormalizedName: "test/pack",
		Content: pack.Content{
			Ruleset: pack.Ruleset{
				Rules: []pack.Rule{
					{ID: "r1", Engine: "config-file"},
					{ID: "r2", Engine: "semgrep", RulePath: "rules/r2.yml", Standard: "x"},
				},
			},
		},
	}}
	violations, err := dispatchPackEngines(manifests, packDir, t.TempDir(), nil, emptySarifRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-sandbox engines, got %d: %v", len(violations), violations)
	}
}

// TestDispatchSemgrep_GathersRuleFlagsInputs verifies the group-by-engine input
// gathering for engine: semgrep rules (the ex-layer-2 rule-flags path), with the
// engine: sandbox rule excluded from the findings group. Re-keyed from
// TestMergePackRules_CollectsLayer2Paths: it asserts the semgrep command is
// invoked with exactly the two rule files as repeated --config inputs while the
// sandbox rule is not fed to semgrep.
func TestDispatchSemgrep_GathersRuleFlagsInputs(t *testing.T) {
	packDir := t.TempDir()
	packRoot := filepath.Join(packDir, "test/pack")
	os.MkdirAll(filepath.Join(packRoot, "rules"), 0o755)
	os.WriteFile(filepath.Join(packRoot, "rules", "r1.yml"), []byte("rules: []"), 0o644)
	os.WriteFile(filepath.Join(packRoot, "rules", "r2.yml"), []byte("rules: []"), 0o644)
	os.WriteFile(filepath.Join(packRoot, "v.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755)

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}
	manifests := []*pack.Manifest{{
		NormalizedName: "test/pack",
		Content: pack.Content{
			Ruleset: pack.Ruleset{
				Rules: []pack.Rule{
					{ID: "r1", Engine: "semgrep", RulePath: "rules/r1.yml", Standard: "x"},
					{ID: "r2", Engine: "semgrep", RulePath: "rules/r2.yml", Standard: "x"},
					{ID: "r3", Engine: "sandbox", Validator: "v.sh", InputScope: "multi-file", Category: "presence"},
				},
			},
		},
	}}
	orig := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { sandboxedRun = orig })

	if _, err := dispatchPackEngines(manifests, packDir, t.TempDir(), nil, rec); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	configs := 0
	for i := 0; i+1 < len(rec.lastArgs); i++ {
		if rec.lastArgs[i] == "--config" {
			configs++
		}
	}
	if configs != 2 {
		t.Errorf("expected 2 --config rule-flags inputs for semgrep, got %d: %v", configs, rec.lastArgs)
	}
}

func TestVerifyPackLock_EmptyPacks(t *testing.T) {
	err := verifyPackLock(t.TempDir(), []string{})
	if err != nil {
		t.Errorf("expected no error for empty packs, got: %v", err)
	}
}
