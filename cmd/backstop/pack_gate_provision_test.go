package main

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// withBinaryResolver installs a fake binary resolver for the duration of a test,
// so absence of `go` / `golangci-lint` is simulated without depending on the host
// PATH (REQ-008). present is the set of tool names the fake resolver "finds".
func withBinaryResolver(t *testing.T, present ...string) {
	t.Helper()
	set := map[string]bool{}
	for _, p := range present {
		set[p] = true
	}
	orig := binaryResolver
	binaryResolver = func(name string) (string, error) {
		if set[name] {
			return "/fake/bin/" + name, nil
		}
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() { binaryResolver = orig })
}

// TestProvision_GoAssumedPresentFailsLoud proves a missing `go` binary fails loud
// with a ConfigError (exit 2) naming the tool and is never auto-installed
// (CLM-026). The go-build/go-test engines carry nil Provision (assume-present);
// absence must surface as a hard config stop, NOT a silent skip or install.
func TestProvision_GoAssumedPresentFailsLoud(t *testing.T) {
	// golangci-lint present, go absent — isolate the `go` fail-loud.
	withBinaryResolver(t, "golangci-lint")
	m := onlyRules(goToolchainManifest(t), "go-build", "go-test")

	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("expected a fail-loud ConfigError for a missing `go` binary, got nil — that is a silent skip")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("missing `go` must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), "go") {
		t.Errorf("ConfigError must name the absent tool `go`, got: %v", cfgErr)
	}
	if strings.Contains(strings.ToLower(cfgErr.Error()), "install") {
		t.Errorf("assume-present native tool must NOT be auto-installed; error must not promise an install: %v", cfgErr)
	}
}

// TestProvision_GolangciAssumedPresentFailsLoud proves a missing `golangci-lint`
// binary fails loud with a ConfigError (exit 2) naming the tool and is never
// auto-installed (CLM-027).
func TestProvision_GolangciAssumedPresentFailsLoud(t *testing.T) {
	// go present, golangci-lint absent — isolate the golangci-lint fail-loud.
	withBinaryResolver(t, "go")
	m := onlyRules(goToolchainManifest(t), "golangci")

	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("expected a fail-loud ConfigError for a missing `golangci-lint` binary, got nil")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("missing `golangci-lint` must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), "golangci-lint") {
		t.Errorf("ConfigError must name the absent tool `golangci-lint`, got: %v", cfgErr)
	}
}

// TestProvision_SemgrepStillPinnedAndProvisioned proves the backstop-introduced
// semgrep engine remains pinned and auto-provisioned, distinct from the
// assume-present native toolchain (CLM-028). semgrep carries a Provision record,
// so its ABSENCE on PATH is NOT a fail-loud here — it is provisioned through the
// declared model, never an assume-present ConfigError.
func TestProvision_SemgrepStillPinnedAndProvisioned(t *testing.T) {
	// Nothing present on PATH: a semgrep-only pack must still pass provisioning,
	// because semgrep is pinned + auto-provisioned, not assume-present.
	withBinaryResolver(t /* nothing present */)
	m := semgrepOnlyManifest(t)

	if err := provisionEngines([]*pack.Manifest{m}); err != nil {
		t.Fatalf("semgrep is pinned + auto-provisioned, not assume-present; absence must NOT fail provisioning, got: %v", err)
	}

	// And the semgrep binding genuinely carries a pinned Provision (distinct from
	// the nil-Provision native toolchain engines) — the split is real.
	bind := resolveEngineRegistry(nil)["semgrep"]
	if bind.Provision == nil {
		t.Fatal("semgrep must carry a pinned Provision record (backstop-introduced engine), not be assume-present")
	}
	if bind.Provision.Version == "" {
		t.Error("semgrep Provision must be pinned to a version")
	}
}

// semgrepOnlyManifest builds an in-memory manifest with a single semgrep rule so
// the provisioning split can be exercised for a backstop-introduced engine
// without the host having semgrep installed.
func semgrepOnlyManifest(t *testing.T) *pack.Manifest {
	t.Helper()
	m := &pack.Manifest{NormalizedName: "test/semgrep-only"}
	m.Content.Ruleset.Rules = []pack.Rule{{ID: "s1", Engine: "semgrep", RulePath: "rules/x.yml"}}
	return m
}

// TestProvisionEngines_UnallowlistedToolFailsLoudBeforeProvision proves the
// allowlist trust gate fires at the EARLIEST tool-resolution walk
// (provisionEngines, the second resolveEngineRegistry caller) so an un-allowlisted
// provisioned tool is rejected with a *check.ConfigError BEFORE provisioning —
// not only at validate and dispatch (SPEC-035 REQ-003/CLM-030). The pack declares
// a provisioned engine whose tool is genuinely ABSENT from the fixture allowlist;
// provisionEngines must fail loud naming the tool, never reach a provision/skip.
func TestProvisionEngines_UnallowlistedToolFailsLoudBeforeProvision(t *testing.T) {
	f := withTestAllowlist(t)
	absent := f.absentTool(t)
	// A provisioned engine (non-nil Provision) whose tool is un-allowlisted. Its
	// Provision-carrying shape means provisionEngines would otherwise `continue`
	// past it as "pinned + auto-provisioned" — the gate must reject it first.
	m := &pack.Manifest{
		NormalizedName: "acme/unallowlisted",
		Engines: map[string]pack.EngineSpec{
			"acme-findings": {Binding: engine.EngineBinding{
				Command:   absent + " --sarif",
				InputMode: engine.InputModeRuleFlags,
				InputFlag: "--config",
				ScopeKind: engine.ScopeKindFileArgs,
				Provision: &engine.Provision{Tool: absent, Version: "1.0.0"},
				Category:  engine.EngineCategoryOpinion,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "acme-findings", RulePath: "rules/r.yml", Standard: "x"},
		}}},
	}

	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("an un-allowlisted provisioned tool must fail loud at provisionEngines (the earliest chokepoint), got nil")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("un-allowlisted tool at provision time must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), absent) {
		t.Errorf("ConfigError must name the un-allowlisted tool %q, got: %v", absent, cfgErr)
	}
	if !strings.Contains(cfgErr.Error(), "acme/unallowlisted") {
		t.Errorf("ConfigError must name the pack, got: %v", cfgErr)
	}
}
