package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// mkDirAll creates dir (and parents) or fails the test.
func mkDirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

// writeFileStr writes content to path (creating parents) or fails the test.
func writeFileStr(t *testing.T, path, content string) {
	t.Helper()
	mkDirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// readFileStr reads path or fails the test.
func readFileStr(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// emptySarifRunner is a CommandRunner whose RunStdout returns an empty SARIF
// log (zero findings). It lets the findings-engine path run without a live
// tool when a test only cares about the sandbox branch or input gathering.
type emptySarifRunner struct{}

func (emptySarifRunner) Run(context.Context, string, ...string) ([]byte, error) { return nil, nil }
func (emptySarifRunner) RunStdout(context.Context, string, ...string) ([]byte, error) {
	return []byte(`{"version":"2.1.0","runs":[]}`), nil
}

// capturingRunner records the args of every RunStdout call (lastArgs is the
// most recent; allConfigs accumulates every --config value across calls) and
// returns canned stdout, so tests can assert how dispatch shapes the engine
// invocation from a rule's input_mode.
type capturingRunner struct {
	out        []byte
	err        error
	lastName   string
	lastArgs   []string
	allConfigs []string
	calls      int
}

func (r *capturingRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return r.out, r.err
}

func (r *capturingRunner) RunStdout(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls++
	r.lastName = name
	r.lastArgs = append([]string(nil), args...)
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--config" || args[i] == "--rule" {
			r.allConfigs = append(r.allConfigs, args[i+1])
		}
	}
	return r.out, r.err
}

// TestGateDispatch_GroupsRulesByEngine proves dispatch groups a pack's rules by
// declared engine and runs each engine once via the EngineBinding table
// (CLM-002): a pack with two semgrep rules and one sandbox rule runs the semgrep
// command exactly once (one invocation carrying both rule files) and the sandbox
// validator separately.
func TestGateDispatch_GroupsRulesByEngine(t *testing.T) {
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "semgrep"))
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "a.yml"), "rules: []\n")
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "b.yml"), "rules: []\n")
	writeFileStr(t, filepath.Join(packRoot, "v.sh"), "#!/bin/sh\nexit 0\n")

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}
	sandboxCalls := 0
	orig := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) { sandboxCalls++; return nil, nil }
	t.Cleanup(func() { sandboxedRun = orig })

	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "a", Engine: "semgrep", RulePath: "semgrep/a.yml", Standard: "x"},
			{ID: "b", Engine: "semgrep", RulePath: "semgrep/b.yml", Standard: "x"},
			{ID: "v", Engine: "sandbox", Validator: "v.sh", InputScope: "multi-file", Category: "presence"},
		}}},
	}}

	if _, err := dispatchPackEngines(manifests, packsDir, t.TempDir(), nil, rec); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if rec.calls != 1 {
		t.Errorf("expected semgrep run exactly once (grouped), got %d invocations", rec.calls)
	}
	if len(rec.allConfigs) != 2 {
		t.Errorf("expected both semgrep rule files in the single invocation, got %v", rec.allConfigs)
	}
	if sandboxCalls != 1 {
		t.Errorf("expected the sandbox validator to run once, got %d", sandboxCalls)
	}
}

// TestGateDispatch_NewEngineNeedsNoExecutorEdit proves a newly registered
// EngineBinding dispatches without editing any executor switch (CLM-003): a
// custom engine injected into the package-level registry is looked up and run
// purely from its declaration.
func TestGateDispatch_NewEngineNeedsNoExecutorEdit(t *testing.T) {
	orig := engineRegistry
	t.Cleanup(func() { engineRegistry = orig })
	// Clone and add a brand-new findings engine that is SARIF-native (no convert).
	engineRegistry = builtinTestRegistry(t)
	engineRegistry["customlint"] = engine.EngineBinding{
		Command:   "customlint",
		InputMode: engine.InputModeRuleFlags,
		InputFlag: "--rule",
	}

	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "customlint"))
	writeFileStr(t, filepath.Join(packRoot, "customlint", "r.yml"), "rules: []\n")

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[{"results":[{"ruleId":"x","message":{"text":"m"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"f.go"},"region":{"startLine":3}}}]}]}]}`)}

	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "customlint", RulePath: "customlint/r.yml"},
		}}},
	}}
	violations, err := dispatchPackEngines(manifests, packsDir, t.TempDir(), nil, rec)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if rec.lastName != "customlint" {
		t.Errorf("expected the custom engine command to run, got %q", rec.lastName)
	}
	if len(violations) != 1 || !strings.HasPrefix(violations[0].Rule, "org/pack/") {
		t.Fatalf("expected one namespaced violation from the new engine, got %#v", violations)
	}
}

// TestGateDispatch_UnknownEngineFailsLoud proves a declared engine name unknown
// to the binding table is a fail-loud config error, not a silent skip (CLM-020).
func TestGateDispatch_UnknownEngineFailsLoud(t *testing.T) {
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "nonexistent-engine", RulePath: "r.yml"},
		}}},
	}}
	_, err := dispatchPackEngines(manifests, t.TempDir(), t.TempDir(), nil, emptySarifRunner{})
	if err == nil {
		t.Fatal("expected a fail-loud error for an unknown engine")
	}
	if !strings.Contains(err.Error(), "unknown engine") {
		t.Errorf("error = %q, want it to name the unknown engine", err.Error())
	}
}

// TestGateDispatch_ParserIsSarifViaLookup proves the dispatch resolves the SARIF
// parser via lookupParser and owns no engine enumeration (CLM-019/CLM-036): a
// findings engine's SARIF output maps to a violation, demonstrating the
// check.ParsePackFindings (lookupParser-"sarif") path is the only parser used.
func TestGateDispatch_ParserIsSarifViaLookup(t *testing.T) {
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "semgrep"))
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "r.yml"), "rules: []\n")

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[{"results":[{"ruleId":"no-eval","message":{"text":"eval forbidden"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"danger.go"},"region":{"startLine":9}}}]}]}]}`)}
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "no-eval", Engine: "semgrep", RulePath: "semgrep/r.yml", Standard: "x"},
		}}},
	}}
	violations, err := dispatchPackEngines(manifests, packsDir, t.TempDir(), nil, rec)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 SARIF-parsed violation, got %d", len(violations))
	}
	if violations[0].File != "danger.go" || violations[0].Message != "eval forbidden" {
		t.Errorf("SARIF fields not mapped: %#v", violations[0])
	}
	if violations[0].Rule != "org/pack/no-eval" {
		t.Errorf("expected namespaced rule id, got %q", violations[0].Rule)
	}
}

// TestEngineDispatch_PackFindingsParseSarifOnly is a source self-check proving
// the dispatch path resolves only the SARIF parser and never references
// golangci-json or eslint-json (CLM-036 / Review Question 4).
func TestEngineDispatch_PackFindingsParseSarifOnly(t *testing.T) {
	src := readFileStr(t, "pack_gate.go")
	for _, banned := range []string{"golangci-json", "eslint-json"} {
		if strings.Contains(src, banned) {
			t.Errorf("pack_gate.go references %q; the pack dispatch path must resolve only parseSarif", banned)
		}
	}
	if !strings.Contains(src, "ParsePackFindings") {
		t.Error("pack_gate.go should resolve findings via check.ParsePackFindings (the SARIF lookupParser path)")
	}
}

// emptySarifRunnerIsCommandRunner is a compile-time assertion that
// emptySarifRunner satisfies check.CommandRunner (and keeps the check import
// referenced). Written as a function rather than a package-level `var _ = ...`
// so it carries no package-level mutable state.
func emptySarifRunnerIsCommandRunner() check.CommandRunner { return emptySarifRunner{} }
