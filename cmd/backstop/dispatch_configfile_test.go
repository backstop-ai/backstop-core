package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// sarifOneFinding is a one-result SARIF log used as a config-file engine's native
// stdout (golangci-lint v2 emits SARIF; no convert). Returned from a func (not a
// package const/var) to keep no global mutable state.
func sarifOneFinding() string {
	return `{"version":"2.1.0","runs":[{"results":[{"ruleId":"errcheck","message":{"text":"unchecked error"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"x.go"},"region":{"startLine":4}}}]}]}]}`
}

// TestGateDispatch_ConfigFileEngineRunsOwnRules proves a config-file engine
// (golangci) runs the tool's OWN built-in rules tuned by an optional pack config,
// passing ONLY the config path — never rule files (CLM-039 / REQ-021): the
// engine invocation carries `--config <pack-config>` and the project target, and
// the resulting finding is namespaced to the pack. Substantive: asserts the exact
// shape of the invocation (config-only, no per-rule inputs) and the namespaced
// violation.
func TestGateDispatch_ConfigFileEngineRunsOwnRules(t *testing.T) {
	// The manifest binds the built-in golangci engine without declaring it; after
	// ISSUE-027 golangci is pack DATA (the go-toolchain pack), so install the full
	// built-in set on the seam to resolve it — the union production sees with the
	// go-toolchain pack installed.
	origReg := engineRegistry
	t.Cleanup(func() { engineRegistry = origReg })
	engineRegistry = builtinTestRegistry(t)

	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "lint-pack")
	mkDirAll(t, filepath.Join(packRoot, "golangci"))
	cfgRel := "golangci/.golangci.yml"
	writeFileStr(t, filepath.Join(packRoot, cfgRel), "linters:\n  enable: [errcheck]\n")

	rec := &capturingRunner{out: []byte(sarifOneFinding())}
	manifest := &pack.Manifest{
		NormalizedName: "org/lint-pack",
		// A config-file engine takes ONE optional pack config (the first rule that
		// declares a rule_path supplies it); the tool runs its own rules.
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "lint", Engine: "golangci", RulePath: cfgRel, Standard: "x"},
		}}},
	}

	violations, err := dispatchPackEngines([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, rec)
	if err != nil {
		t.Fatalf("dispatchPackEngines (config-file engine): %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 namespaced violation from the config-file engine, got %d: %#v", len(violations), violations)
	}
	if violations[0].Rule != "org/lint-pack/errcheck" {
		t.Errorf("config-file engine violation must be namespaced, got %q", violations[0].Rule)
	}

	// The invocation must carry exactly one --config pointing at the pack config and
	// NO per-rule input files (the tool runs its own rules).
	if rec.lastName != "golangci-lint" {
		t.Fatalf("expected the golangci-lint command to run, got %q", rec.lastName)
	}
	cfgCount := 0
	for i, a := range rec.lastArgs {
		if a == "--config" {
			cfgCount++
			if i+1 >= len(rec.lastArgs) || !strings.HasSuffix(rec.lastArgs[i+1], ".golangci.yml") {
				t.Errorf("--config must point at the pack config, got args %v", rec.lastArgs)
			}
		}
	}
	if cfgCount != 1 {
		t.Errorf("config-file engine must pass exactly one --config (its own rules, one optional config), got %d in %v", cfgCount, rec.lastArgs)
	}
	// It must NOT be fed rule files as inputs (that is the rule-flags
	// shape, not config-file).
	if strings.Contains(strings.Join(rec.lastArgs, " "), ".golangci.yml .golangci.yml") {
		t.Errorf("config-file engine must inject a single config, not repeated rule files, got %v", rec.lastArgs)
	}
}

// TestGateDispatch_ConfigFileEngineNeedsNoGo proves adding a config-driven linter
// is a pure EngineBinding declaration that rides the SAME dispatch path with NO
// edit to the dispatch switch (CLM-040 / REQ-018): a brand-new config-file
// binding registered into a test registry dispatches through the identical
// gatherEngineInputs/runFindingsEngine path and produces a namespaced violation.
// Substantive: the new engine is data-only (a Registry entry), yet it fires.
func TestGateDispatch_ConfigFileEngineNeedsNoGo(t *testing.T) {
	orig := engineRegistry
	t.Cleanup(func() { engineRegistry = orig })
	// A second, brand-new config-file engine — added purely as data (no Go in the
	// dispatch switch). It is SARIF-native (no convert) like golangci v2.
	engineRegistry = builtinTestRegistry(t)
	engineRegistry["customlint"] = engine.EngineBinding{
		Command:       "customlint check",
		InputMode:     engine.InputModeConfigFile,
		InputFlag:     "--config",
		ScopeKind:     engine.ScopeKindProjectWide,
		ProjectTarget: "./...",
	}

	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "custom-pack")
	mkDirAll(t, filepath.Join(packRoot, "customlint"))
	cfgRel := "customlint/config.toml"
	writeFileStr(t, filepath.Join(packRoot, cfgRel), "[lint]\nenabled = true\n")

	rec := &capturingRunner{out: []byte(sarifOneFinding())}
	manifest := &pack.Manifest{
		NormalizedName: "org/custom-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "custom", Engine: "customlint", RulePath: cfgRel, Standard: "x"},
		}}},
	}

	violations, err := dispatchPackEngines([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, rec)
	if err != nil {
		t.Fatalf("a new config-file engine must dispatch with no executor edit: %v", err)
	}
	if rec.lastName != "customlint" {
		t.Errorf("the new config-file engine command must run from its declaration, got %q", rec.lastName)
	}
	if len(violations) != 1 || violations[0].Rule != "org/custom-pack/errcheck" {
		t.Fatalf("the new config-file engine must produce a namespaced violation via the same path, got %#v", violations)
	}
	// It rode the config-file input shape: one --config, project target appended.
	args := strings.Join(rec.lastArgs, " ")
	if !strings.Contains(args, "--config") || !strings.Contains(args, "config.toml") {
		t.Errorf("the new engine must inject its pack config via --config, got %v", rec.lastArgs)
	}
	if !strings.Contains(args, "./...") {
		t.Errorf("a ScopeKindProjectWide config-file engine must append its ProjectTarget, got %v", rec.lastArgs)
	}
}
