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

// ISSUE-112 — provisioning presence-checks EVERY declared engine tool, whether it
// carries a pinned Provision record or not. The pin is a trusted-tool allowlist
// entry (a TRUST key), never an installer: nothing in backstop fetches a pinned
// tool, so exempting pinned bindings from the presence probe let an engine whose
// binary was absent report a clean, finding-free scan. These tests pin the
// corrected contract on both branches.

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

// TestProvision_SemgrepStillPinnedAndProvisioned proves semgrep's pinned
// Provision record survives while its PATH absence now fails loud.
//
// ISSUE-112 SUPERSEDES this test's former assertion (CLM-028: "absence on PATH
// must NOT fail provisioning") — a pinned Provision is a TRUST record, not an
// installer, so a pinned tool that is absent from PATH scanned nothing and must
// refuse rather than pass green. The name is kept for continuity with the
// SPEC-031/SPEC-034 history a reader following this test will want.
//
// The still-true half is preserved: the semgrep binding genuinely carries a
// pinned Provision with a non-empty version — the record is real, only its
// meaning was mis-stated.
func TestProvision_SemgrepStillPinnedAndProvisioned(t *testing.T) {
	// Nothing present on PATH: a semgrep-only pack must now REFUSE, because a
	// pinned Provision never installed semgrep and never asserted its presence.
	withBinaryResolver(t /* nothing present */)
	m := semgrepOnlyManifest(t)

	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("a pinned-Provision tool absent from PATH must fail loud, got nil — that is the silent vacuous pass ISSUE-112 reports")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("an absent pinned tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), "semgrep") {
		t.Errorf("ConfigError must name the absent tool `semgrep`, got: %v", cfgErr)
	}

	// STILL TRUE (unchanged by ISSUE-112): the semgrep binding genuinely carries a
	// pinned Provision record, distinct from the nil-Provision native toolchain.
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

// pinnedProvisionManifest builds an in-memory manifest with ONE pack-declared
// engine carrying a non-nil (pinned) Provision, so the ISSUE-112 presence check
// can be driven for a provision-pinned tool. command supplies the argv[0] that
// gets PROBED; tool/version supply the Provision record that governs TRUST — the
// two are deliberately separable, because a pack may pin `ast-grep` and invoke
// `sg` (the divergence the refusal message must report on both sides).
func pinnedProvisionManifest(packName, command, tool, version string) *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: packName,
		Engines: map[string]pack.EngineSpec{
			"acme-findings": {Binding: engine.EngineBinding{
				Command:   command,
				InputMode: engine.InputModeRuleFlags,
				InputFlag: "--config",
				ScopeKind: engine.ScopeKindFileArgs,
				Provision: &engine.Provision{Tool: tool, Version: version},
				Category:  engine.EngineCategoryOpinion,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "acme-findings", RulePath: "rules/r.yml", Standard: "x"},
		}}},
	}
}

// TestProvision_ProvisionPinnedToolAbsentFailsLoud proves the ISSUE-112 fix in its
// general form (CLM-001): a pack-declared engine carrying a PINNED Provision whose
// binary is absent from PATH fails loud with a *check.ConfigError (exit 2) naming
// the tool, instead of being waved through as "pinned + auto-provisioned". The
// pack's tool is ALLOWLISTED (acme-scan at its fixture-pinned version) so the trust
// gate passes cleanly and the PRESENCE probe is unambiguously what fires.
func TestProvision_ProvisionPinnedToolAbsentFailsLoud(t *testing.T) {
	f := withTestAllowlist(t)
	const tool = "acme-scan"
	pinned, ok := f.lockedVersion(tool)
	if !ok {
		t.Fatalf("fixture invariant: %q must be lock-pinned so the trust gate passes and the presence probe is what fires", tool)
	}
	requested := recordingBinaryResolver(t /* nothing present */)

	m := pinnedProvisionManifest("acme/pinned", tool+" --sarif", tool, pinned)
	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("a provision-pinned tool absent from PATH must fail loud, got nil — the pin is a trust record, not an installer")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("an absent provision-pinned tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), tool) {
		t.Errorf("ConfigError must name the absent tool %q, got: %v", tool, cfgErr)
	}
	// The refusal came from a REAL probe, not an unrelated early error.
	if !sliceContains(*requested, tool) {
		t.Errorf("provisioning must PROBE the pinned tool on PATH, requested %v", *requested)
	}
}

// TestProvision_ProvisionPinnedToolPresentPasses is the anti-vacuous twin of
// TestProvision_ProvisionPinnedToolAbsentFailsLoud (CLM-001): the SAME pack with
// the SAME pinned engine provisions cleanly once its binary resolves on PATH.
// Without this, the presence check would be satisfiable by refusing everything.
func TestProvision_ProvisionPinnedToolPresentPasses(t *testing.T) {
	f := withTestAllowlist(t)
	const tool = "acme-scan"
	pinned, ok := f.lockedVersion(tool)
	if !ok {
		t.Fatalf("fixture invariant: %q must be lock-pinned", tool)
	}
	requested := recordingBinaryResolver(t, tool)

	m := pinnedProvisionManifest("acme/pinned", tool+" --sarif", tool, pinned)
	if err := provisionEngines([]*pack.Manifest{m}); err != nil {
		t.Fatalf("a provision-pinned tool PRESENT on PATH must provision cleanly, got: %v", err)
	}
	// And it really was probed — a pass that never looked would prove nothing.
	if !sliceContains(*requested, tool) {
		t.Errorf("the passing run must still have PROBED the tool, requested %v", *requested)
	}
}

// TestProvision_AbsentToolErrorNamesPackAndPin proves the refusal message carries
// everything a human needs to act (CLM-001): the PROBED argv[0] — the binary that
// must actually go on PATH — plus the declaring pack and the Provision tool+version
// that the pin rode in on, and NO promise that backstop will install it.
//
// The fixture is built ON the argv[0]-vs-Provision.Tool divergence deliberately
// (a pack may pin `ast-grep` and invoke `sg`), with two names that are not
// substrings of each other, so "names both" is a falsifiable assertion rather than
// one string accidentally satisfying two checks.
func TestProvision_AbsentToolErrorNamesPackAndPin(t *testing.T) {
	f := withTestAllowlist(t)
	const (
		pinnedTool = "acme-scan"      // the TRUST key (Provision.Tool)
		probedArgv = "acme-probe-cli" // what exec would actually resolve (argv[0])
		packName   = "acme/divergent"
	)
	pinnedVersion, ok := f.lockedVersion(pinnedTool)
	if !ok {
		t.Fatalf("fixture invariant: %q must be lock-pinned", pinnedTool)
	}
	if strings.Contains(probedArgv, pinnedTool) || strings.Contains(pinnedTool, probedArgv) {
		t.Fatalf("test invariant: %q and %q must not be substrings of each other, or 'names both' is unfalsifiable", probedArgv, pinnedTool)
	}
	withBinaryResolver(t /* nothing present */)

	m := pinnedProvisionManifest(packName, probedArgv+" --sarif", pinnedTool, pinnedVersion)
	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("an absent provision-pinned tool must fail loud, got nil")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	msg := cfgErr.Error()
	if !strings.Contains(msg, probedArgv) {
		t.Errorf("message must name the PROBED argv[0] %q — the binary the user must put on PATH, got: %s", probedArgv, msg)
	}
	if !strings.Contains(msg, pinnedTool) {
		t.Errorf("message must name the pinned Provision tool %q (trust attribution), got: %s", pinnedTool, msg)
	}
	if !strings.Contains(msg, pinnedVersion) {
		t.Errorf("message must name the pinned version %q, got: %s", pinnedVersion, msg)
	}
	if !strings.Contains(msg, packName) {
		t.Errorf("message must name the declaring pack %q, got: %s", packName, msg)
	}
	// The pin is an allowlist entry, not an installer: the message must SAY so
	// rather than leave a reader expecting backstop to fetch the tool.
	if !strings.Contains(msg, "does not install") {
		t.Errorf("message must state that backstop does not install pack-declared tools (the pin is a trust entry, not an installer), got: %s", msg)
	}
}

// TestProvision_AllowlistRefusalPrecedesPresenceProbe is the ordering guard
// (CLM-003): an UN-allowlisted provisioned tool is still refused by the trust gate
// BEFORE any presence probe runs. Without this, folding provision-pinned bindings
// into the presence check could quietly reorder the two gates, leaving
// TestProvisionEngines_UnallowlistedToolFailsLoudBeforeProvision passing for the
// NEW reason (absent from PATH) rather than the one it exists to pin.
func TestProvision_AllowlistRefusalPrecedesPresenceProbe(t *testing.T) {
	f := withTestAllowlist(t)
	absent := f.absentTool(t)
	requested := recordingBinaryResolver(t /* nothing present */)

	m := pinnedProvisionManifest("acme/unallowlisted-order", absent+" --sarif", absent, "1.0.0")
	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("an un-allowlisted provisioned tool must be refused by the trust gate, got nil")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("trust-gate refusal must be a *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), "allowlist") {
		t.Errorf("the refusal must be the ALLOWLIST refusal, not a presence refusal, got: %v", cfgErr)
	}
	// THE ORDERING ASSERTION: the trust gate short-circuits before the PATH probe,
	// so no binary was ever looked up.
	if len(*requested) != 0 {
		t.Errorf("the allowlist refusal must precede any presence probe; provisioning probed %v", *requested)
	}
}
