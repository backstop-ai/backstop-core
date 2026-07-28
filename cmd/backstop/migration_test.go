package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/baseengines"
	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// SPEC-035 Phase 6 / TASK-032 — DefaultRegistry OQ-1 disposition, dispatch-level
// coverage (option-AGNOSTIC).
//
// OQ-1 resolved to OPTION (i): DefaultRegistry stays as the incremental fallback
// the stage-1 merge (resolveEngineRegistry) overrides/extends. These tests assert
// the INVARIANTS through the REAL dispatch path (dispatchPackEngines + the real
// checkEngineToolAllowed trust gate), complementing the leaf-level merge/contract
// proofs in pkg/pack/engine/migration_test.go (the mandated CLM-027/CLM-028/CLM-037
// test names). The invariants hold whether the built-ins come from the
// DefaultRegistry fallback or a default pack — these exercise the fallback (option i):
//
//   - CLM-027: a rule whose engine is a BUILT-IN (semgrep, resolved purely from the
//     DefaultRegistry fallback with no pack-declared engine) still dispatches and
//     reaches RunStdout — the built-in binding's command + pinned version are
//     available to dispatch.
//   - CLM-028: the dispatch-time trust gate fires on a BUILT-IN binding's tool
//     exactly as on a pack-declared one — under an allowlist that omits the
//     built-in's tool, the built-in rule fails loud and never reaches RunStdout.
//
// The fixture allowlist (testdata/allowlist-fixtures.yml) deliberately does NOT
// list "semgrep", so it is the genuine un-allowlisted cell for the built-in tool.

// builtinSemgrepManifest builds a manifest with ONE rule bound to the BUILT-IN
// semgrep engine and NO pack-declared engines: the engine resolves entirely from
// the DefaultRegistry fallback via resolveEngineRegistry. The rule file lives under
// the pack's semgrep/ layout so gatherEngineInputs resolves it.
func builtinSemgrepManifest(t *testing.T, packsDir string) *pack.Manifest {
	t.Helper()
	packRoot := filepath.Join(packsDir, "org", "pack")
	if err := os.MkdirAll(filepath.Join(packRoot, "semgrep"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "semgrep", "r.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write rule: %v", err)
	}
	return &pack.Manifest{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "semgrep", RulePath: "semgrep/r.yml", Standard: "x"},
		}}},
	}
}

// TestMigration_BuiltinBindingsDispatchThroughFallback proves CLM-027 at the
// dispatch level: a rule bound to the BUILT-IN semgrep engine (no pack-declared
// engines block) still dispatches via the DefaultRegistry fallback and reaches
// RunStdout. The built-in's command + pinned version are available to dispatch —
// the migration to the merge model does not strand the built-ins. The allowlist is
// stubbed to PIN semgrep at its built-in version so the trust gate passes and the
// command runs (proving availability, not the negative path).
func TestMigration_BuiltinBindingsDispatchThroughFallback(t *testing.T) {
	builtin := baseengines.Registry()["semgrep"]
	if builtin.Provision == nil {
		t.Fatal("fixture invariant: the built-in semgrep binding must carry a Provision")
	}
	// Pin semgrep at its built-in locked version so the trust gate passes.
	orig := trustedToolAllowlist
	trustedToolAllowlist = func() map[string]string {
		return map[string]string{builtin.Provision.Tool: builtin.Provision.Version}
	}
	t.Cleanup(func() { trustedToolAllowlist = orig })

	packsDir := t.TempDir()
	m := builtinSemgrepManifest(t, packsDir)

	rec := &recordingStdoutRunner{stdout: []byte(`{"version":"2.1.0","runs":[]}`)}
	if _, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, rec); err != nil {
		t.Fatalf("a built-in engine resolved from the DefaultRegistry fallback must still dispatch, got: %v", err)
	}
	if !rec.runStdoutWasCalled() {
		t.Fatal("CLM-027: the built-in semgrep binding's command must reach RunStdout — the fallback built-in is available to dispatch")
	}
	if rec.runStdoutCalls[0] != "semgrep" {
		t.Errorf("CLM-027: the built-in semgrep COMMAND must be the one dispatched, got %q", rec.runStdoutCalls[0])
	}
}

// TestMigration_AllowlistGatesBuiltinToolThroughDispatch proves CLM-028 at the
// dispatch level: the trust gate fires on a BUILT-IN binding's tool exactly as on a
// pack-declared one. Under the fixture allowlist (which omits "semgrep"), a rule
// bound to the BUILT-IN semgrep engine fails loud with a *check.ConfigError and its
// command NEVER reaches RunStdout — the allowlist is the trust floor regardless of
// the binding's source (built-in or pack-declared). A built-in source does NOT
// bypass the gate.
func TestMigration_AllowlistGatesBuiltinToolThroughDispatch(t *testing.T) {
	f := withTestAllowlist(t)
	// The fixture allowlist must NOT contain semgrep — it is the genuine
	// un-allowlisted cell for the built-in's tool (not a stub-open hole).
	if _, present := f.Allowlist["semgrep"]; present {
		t.Fatal("fixture invariant broken: the allowlist must OMIT semgrep to drive the built-in un-allowlisted cell (CLM-028)")
	}

	packsDir := t.TempDir()
	m := builtinSemgrepManifest(t, packsDir)

	rec := &recordingStdoutRunner{}
	_, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, rec)
	if err == nil {
		t.Fatal("CLM-028: a built-in binding whose tool is absent from the allowlist must fail loud — the built-in source must not bypass the trust floor")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("the built-in un-allowlisted tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !contains(cfgErr.Error(), "semgrep") || !contains(cfgErr.Error(), "org/pack") {
		t.Errorf("the ConfigError must name the tool semgrep and the pack org/pack, got: %v", cfgErr)
	}
	if rec.runStdoutWasCalled() {
		t.Fatal("CLM-028: a built-in binding's un-allowlisted command must NEVER reach RunStdout — the gate sits before dispatch for built-ins too")
	}
}
