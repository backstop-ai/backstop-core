package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// SPEC-035 TASK-021 — dispatch trust-gate matrix + sandbox-exempt tests.
//
// These tests drive a REAL allowlist (loaded from testdata/allowlist-fixtures.yml
// via the Phase-1 harness) with a genuine ABSENT-tool cell and a genuine
// version-divergent cell — never a stub-open allowlist (Sharp Edge 3 / the spec's
// Verification section forbids stubbing the allowlist open on the dispatch path
// under test). The fixture allowlist is injected through the trustedToolAllowlist
// seam so the dispatch gate consults it instead of the production
// engine.TrustedToolAllowlist(), while still exercising the real CheckToolAllowed.

// withTestAllowlist installs the fixture allowlist (testdata/allowlist-fixtures.yml)
// as the dispatch/provision trust floor for the duration of a test, and returns
// the loaded fixtures so a test can read the lock pins. It deliberately does NOT
// stub the allowlist open: the fixture has a genuine absent-tool cell (acme-absent)
// and a genuine version-divergent cell (acme-drift).
func withTestAllowlist(t *testing.T) allowlistFixtures {
	t.Helper()
	f := loadAllowlistFixtures(t)
	orig := trustedToolAllowlist
	trustedToolAllowlist = func() map[string]string { return f.Allowlist }
	t.Cleanup(func() { trustedToolAllowlist = orig })
	return f
}

// allowlistDispatchManifest builds a manifest with one pack-declared findings
// engine whose tool/version are supplied by the test, so each allowlist matrix
// cell (allowlisted+pinned, absent, divergent) can be driven by choosing the
// tool/version against the fixture allowlist + lock. The Provision.Version is the
// lock-pinned version the dispatch gate feeds into CheckToolAllowed (CLM-029).
func allowlistDispatchManifest(t *testing.T, packDir, command, tool, lockedVersion string) *pack.Manifest {
	t.Helper()
	packRoot := filepath.Join(packDir, "acme", "pack")
	if err := os.MkdirAll(filepath.Join(packRoot, "rules"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "rules", "r.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write rule: %v", err)
	}
	return &pack.Manifest{
		NormalizedName: "acme/pack",
		Engines: map[string]pack.EngineSpec{
			"acme-findings": {Binding: engine.EngineBinding{
				Command:   command,
				InputMode: engine.InputModeRuleFlags,
				InputFlag: "--config",
				ScopeKind: engine.ScopeKindFileArgs,
				Provision: &engine.Provision{Tool: tool, Version: lockedVersion},
				Category:  engine.EngineCategoryOpinion,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "acme-findings", RulePath: "rules/r.yml", Standard: "x"},
		}}},
	}
}

// TestAllowlist_AllowlistedPinnedToolRuns proves an allowlisted + lock-pinned
// tool's command reaches RunStdout — the trust gate PASSES (SPEC-035 CLM-005).
// acme-scan is on the fixture allowlist at 2.3.1 and the lock pins 2.3.1.
func TestAllowlist_AllowlistedPinnedToolRuns(t *testing.T) {
	f := withTestAllowlist(t)
	locked, pinned := f.lockedVersion("acme-scan")
	if !pinned {
		t.Fatal("fixture invariant: acme-scan must be lock-pinned for the gate-passes cell")
	}
	packDir := t.TempDir()
	m := allowlistDispatchManifest(t, packDir, "acme-scan --sarif", "acme-scan", locked)

	rec := &recordingStdoutRunner{stdout: []byte(`{"version":"2.1.0","runs":[]}`)}
	if _, err := dispatchPackEngines([]*pack.Manifest{m}, packDir, t.TempDir(), nil, rec); err != nil {
		t.Fatalf("an allowlisted+pinned tool must pass the trust gate and run, got: %v", err)
	}
	if !rec.runStdoutWasCalled() {
		t.Fatal("an allowlisted+pinned tool's command must reach RunStdout (the gate passed)")
	}
}

// TestAllowlist_UnallowlistedToolFailsLoud proves an un-allowlisted tool yields a
// *check.ConfigError (exit 2) naming the tool and pack, and the command is never
// run (SPEC-035 CLM-006). acme-absent is genuinely absent from the fixture
// allowlist.
func TestAllowlist_UnallowlistedToolFailsLoud(t *testing.T) {
	f := withTestAllowlist(t)
	absent := f.absentTool(t)
	packDir := t.TempDir()
	// Even if the lock "pins" it, the tool is absent from the allowlist.
	m := allowlistDispatchManifest(t, packDir, absent+" --sarif", absent, "1.0.0")

	rec := &recordingStdoutRunner{}
	_, err := dispatchPackEngines([]*pack.Manifest{m}, packDir, t.TempDir(), nil, rec)
	if err == nil {
		t.Fatal("an un-allowlisted tool must fail loud, got nil — that is the arbitrary-exec hole")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("un-allowlisted tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !contains(cfgErr.Error(), absent) || !contains(cfgErr.Error(), "acme/pack") {
		t.Errorf("ConfigError must name the tool %q and the pack acme/pack, got: %v", absent, cfgErr)
	}
	if rec.runStdoutWasCalled() {
		t.Fatal("an un-allowlisted tool's command must NEVER be handed to RunStdout")
	}
}

// TestAllowlist_AllowlistedButUnpinnedToolFailsLoud proves an allowlisted tool
// whose lock pin does NOT match its allowlisted version yields a ConfigError and
// never runs (SPEC-035 CLM-007). acme-unlocked is on the allowlist but absent from
// the lock; acme-drift is on the allowlist at 1.0.0 but the lock pins 9.9.9.
func TestAllowlist_AllowlistedButUnpinnedToolFailsLoud(t *testing.T) {
	f := withTestAllowlist(t)
	// acme-unlocked: allowlisted, NOT in the lock at all.
	if _, ok := f.Allowlist["acme-unlocked"]; !ok {
		t.Fatal("fixture invariant: acme-unlocked must be on the allowlist")
	}
	if _, pinned := f.lockedVersion("acme-unlocked"); pinned {
		t.Fatal("fixture invariant: acme-unlocked must be ABSENT from the lock (the unpinned cell)")
	}
	packDir := t.TempDir()
	// The lock has no pin for acme-unlocked; feed an empty/wrong locked version.
	m := allowlistDispatchManifest(t, packDir, "acme-unlocked --sarif", "acme-unlocked", "")

	rec := &recordingStdoutRunner{}
	_, err := dispatchPackEngines([]*pack.Manifest{m}, packDir, t.TempDir(), nil, rec)
	if err == nil {
		t.Fatal("an allowlisted-but-unpinned tool must fail loud, got nil")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("unpinned tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if rec.runStdoutWasCalled() {
		t.Fatal("an allowlisted-but-unpinned tool's command must NEVER reach RunStdout")
	}
}

// TestAllowlist_GateBlocksBeforeRunStdout proves the gate sits BEFORE
// splitCommand/RunStdout: using the recording fake runner, an un-allowlisted
// tool's command is NEVER handed to RunStdout (SPEC-035 CLM-008). This is the
// strongest no-partial-execution assertion — the command bytes never reach the
// runner at all.
func TestAllowlist_GateBlocksBeforeRunStdout(t *testing.T) {
	f := withTestAllowlist(t)
	absent := f.absentTool(t)
	packDir := t.TempDir()
	m := allowlistDispatchManifest(t, packDir, absent+" --sarif", absent, "1.0.0")

	rec := &recordingStdoutRunner{}
	if _, err := dispatchPackEngines([]*pack.Manifest{m}, packDir, t.TempDir(), nil, rec); err == nil {
		t.Fatal("expected the trust gate to block an un-allowlisted tool")
	}
	if rec.runStdoutWasCalled() {
		t.Fatalf("CLM-008: the gate must sit BEFORE RunStdout — an un-allowlisted command was handed to the runner: %v", rec.runStdoutCalls)
	}
}

// TestAllowlist_SandboxEngineNotSubjectToToolAllowlist proves the sandbox engine
// (no command, input_mode none) is not subject to the command-allowlist and runs
// as before (SPEC-035 CLM-009). The fixture allowlist does not list the sandbox's
// pack-shipped validator, yet the sandbox rule still dispatches.
func TestAllowlist_SandboxEngineNotSubjectToToolAllowlist(t *testing.T) {
	withTestAllowlist(t)
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	packDir := filepath.Join(projectRoot, ".backstop", "packs")
	packRoot := filepath.Join(packDir, "acme", "sb")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "v.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write validator: %v", err)
	}
	m := &pack.Manifest{
		NormalizedName: "acme/sb",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "sb", Engine: "sandbox", Validator: "v.sh", InputScope: "multi-file", Category: "presence"},
		}}},
	}

	called := false
	sandboxRunner := &recordingSandboxRunner{mode: packval.SandboxModeNative, runFn: func(string, []string, string) (packval.SandboxRunResult, error) {
		called = true
		return packval.SandboxRunResult{}, nil
	}}

	if _, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, packDir, projectRoot, nil, emptySarifRunner{}, sandboxRunner); err != nil {
		t.Fatalf("the sandbox engine carries no tool and must not be allowlist-gated, got: %v", err)
	}
	if !called {
		t.Fatal("the sandbox validator must still run — it is exempt from the command-allowlist (CLM-009)")
	}
}

// TestAllowlist_VersionPinReadsFromLockNotSecondSource proves the dispatch gate's
// version comparison reads the lock-pinned version (the binding's Provision.Version,
// which rides the lock), not a literal in TrustedToolAllowlist: a tool allowlisted
// at vX whose lock pins vY fails loud (SPEC-035 CLM-029). acme-drift is allowlisted
// at 1.0.0 but the fixture lock pins 9.9.9.
func TestAllowlist_VersionPinReadsFromLockNotSecondSource(t *testing.T) {
	f := withTestAllowlist(t)
	allowlisted := f.Allowlist["acme-drift"]
	locked, pinned := f.lockedVersion("acme-drift")
	if !pinned || allowlisted == locked {
		t.Fatalf("fixture invariant: acme-drift must be allowlisted at vX (%q) and lock-pinned at a DIFFERENT vY (%q)", allowlisted, locked)
	}
	packDir := t.TempDir()
	// Feed the LOCK version (9.9.9) as the Provision.Version — the pin rides the
	// lock. It diverges from the allowlist pin (1.0.0), so the gate must fail loud.
	m := allowlistDispatchManifest(t, packDir, "acme-drift --sarif", "acme-drift", locked)

	rec := &recordingStdoutRunner{}
	_, err := dispatchPackEngines([]*pack.Manifest{m}, packDir, t.TempDir(), nil, rec)
	if err == nil {
		t.Fatal("allowlisted-at-vX / locked-at-vY must fail loud, proving the comparison reads the lock not a literal")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("version-divergence must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if rec.runStdoutWasCalled() {
		t.Fatal("a version-divergent tool's command must never reach RunStdout")
	}
}
