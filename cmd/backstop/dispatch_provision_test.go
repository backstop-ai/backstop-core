package main

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// recordingBinaryResolver installs a binary resolver that records every tool name
// it is asked to resolve and reports each `present` tool as found, all others as
// absent. It returns the slice of requested names so a test can assert WHICH
// tools the provisioning path probed on PATH (the behavioral fingerprint of the
// split-provisioning model). Distinct from withBinaryResolver, which does not
// expose the probe set.
func recordingBinaryResolver(t *testing.T, present ...string) *[]string {
	t.Helper()
	set := map[string]bool{}
	for _, p := range present {
		set[p] = true
	}
	requested := &[]string{}
	orig := binaryResolver
	binaryResolver = func(name string) (string, error) {
		*requested = append(*requested, name)
		if set[name] {
			return "/fake/bin/" + name, nil
		}
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() { binaryResolver = orig })
	return requested
}

// TestProvision_NativeAssumedPresentFailsLoud proves a Layer-0 native engine (nil
// Provision) whose binary is absent fails loud with a *check.ConfigError naming
// the tool and is NEVER auto-installed (CLM-041 / REQ-019): provisioning probes
// the binary on PATH, finds it absent, and stops with a config error — no install
// attempt. Substantive: asserts the ConfigError type, that it names the absent
// tool, and that the error never promises an install.
func TestProvision_NativeAssumedPresentFailsLoud(t *testing.T) {
	// `go` absent (golangci-lint present, to isolate the `go` fail-loud). go-build
	// carries a nil Provision => assume-present Layer-0.
	requested := recordingBinaryResolver(t, "golangci-lint")
	m := onlyRules(goToolchainManifest(t), "go-build")

	// Precondition: the engine under test really is assume-present (nil Provision).
	if b := resolveEngineRegistry(nil)["go-build"]; b.Provision != nil {
		t.Fatalf("test precondition: go-build must be assume-present (nil Provision), got %+v", b.Provision)
	}

	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("an absent assume-present native tool must fail loud, got nil — that is a silent skip")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("absent native tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), "go") {
		t.Errorf("ConfigError must name the absent tool, got: %v", cfgErr)
	}
	if strings.Contains(strings.ToLower(cfgErr.Error()), "install") &&
		!strings.Contains(strings.ToLower(cfgErr.Error()), "never auto-provision") {
		t.Errorf("assume-present native tool must NOT be auto-installed; error must not promise an install: %v", cfgErr)
	}
	// And the path genuinely probed `go` on PATH (the fail-loud is from a real
	// absence, not an unrelated error).
	if !sliceContains(*requested, "go") {
		t.Errorf("provisioning must probe the assume-present tool on PATH, requested %v", *requested)
	}
}

// TestProvision_IntroducedEngineAutoProvisioned proves a backstop-introduced
// engine (semgrep/ast-grep) with a pinned Provision is provisioned+verified via
// the declared (lock) model and is NOT failed-loud for PATH absence (CLM-042 /
// REQ-019): with nothing on PATH, a semgrep+ast-grep pack still passes
// provisioning because those engines carry pinned Provision records. Substantive:
// asserts provisioning succeeds with an empty PATH AND that the bindings really
// carry pinned versions (the split is real, not a skip).
func TestProvision_IntroducedEngineAutoProvisioned(t *testing.T) {
	// Nothing present on PATH.
	recordingBinaryResolver(t /* nothing present */)

	m := &pack.Manifest{NormalizedName: "test/introduced"}
	m.Content.Ruleset.Rules = []pack.Rule{
		{ID: "s1", Engine: "semgrep", RulePath: "semgrep/x.yml", Standard: "x"},
		{ID: "a1", Engine: "ast-grep", RulePath: "ast-grep/y.yml", Standard: "x"},
	}

	if err := provisionEngines([]*pack.Manifest{m}); err != nil {
		t.Fatalf("backstop-introduced engines are pinned + auto-provisioned; absence on PATH must NOT fail provisioning, got: %v", err)
	}

	for _, name := range []string{"semgrep", "ast-grep"} {
		b := resolveEngineRegistry(nil)[name]
		if b.Provision == nil {
			t.Errorf("%s must carry a pinned Provision record (backstop-introduced), got nil", name)
			continue
		}
		if b.Provision.Tool != name || b.Provision.Version == "" {
			t.Errorf("%s Provision must be pinned to a tool+version, got %+v", name, b.Provision)
		}
	}
}

// TestProvision_EnsureSemgrepRetired proves semgrep provisioning is driven by the
// declared Provision/lock mechanism and NOT by the bespoke EnsureSemgrep install
// ladder (CLM-043 / REQ-019). DISTINCT ANGLE from the existing source-grep test
// TestProvision_EnsureSemgrepBespokeInstallRetired: this is BEHAVIORAL — it
// asserts the provisioning PATH never probes/installs semgrep at all (the pinned-
// Provision branch skips it), so EnsureSemgrep cannot be on the provision path,
// while a nil-Provision sibling engine in the SAME pack IS probed. That contrast
// proves the split is what governs provisioning, not EnsureSemgrep.
func TestProvision_EnsureSemgrepRetired(t *testing.T) {
	// golangci-lint present so the assume-present sibling passes; semgrep absent on
	// PATH. If EnsureSemgrep (the bespoke ladder) were on the provision path, it
	// would probe/install semgrep; the declared model must NOT.
	requested := recordingBinaryResolver(t, "golangci-lint")

	m := &pack.Manifest{NormalizedName: "test/semgrep-plus-native"}
	m.Content.Ruleset.Rules = []pack.Rule{
		{ID: "s1", Engine: "semgrep", RulePath: "semgrep/x.yml", Standard: "x"},
		{ID: "g1", Engine: "golangci", RulePath: "golangci/.golangci.yml", Standard: "x"},
	}

	if err := provisionEngines([]*pack.Manifest{m}); err != nil {
		t.Fatalf("semgrep is pinned+declared; provisioning must pass without EnsureSemgrep, got: %v", err)
	}

	// BEHAVIORAL: semgrep must NEVER be probed on PATH by provisioning — that is the
	// fingerprint that the bespoke EnsureSemgrep ladder is off the provision path.
	if sliceContains(*requested, "semgrep") {
		t.Errorf("provisioning probed `semgrep` on PATH (%v); the declared Provision model — not EnsureSemgrep — must drive semgrep, so it must NOT be probed", *requested)
	}
	// CONTRAST: the nil-Provision native sibling (golangci-lint) WAS probed, proving
	// the path is live and semgrep's absence-from-probe is a deliberate skip, not a
	// no-op run.
	if !sliceContains(*requested, "golangci-lint") {
		t.Errorf("the assume-present native sibling must be probed (proving provisioning ran), requested %v", *requested)
	}
	// And the pinned Provision record is what governs semgrep (the declared driver).
	if b := resolveEngineRegistry(nil)["semgrep"]; b.Provision == nil || b.Provision.Tool != "semgrep" || b.Provision.Version == "" {
		t.Fatalf("semgrep must be governed by a pinned Provision record (the declared driver), got %+v", resolveEngineRegistry(nil)["semgrep"].Provision)
	}
}

// sliceContains reports whether s is in xs.
func sliceContains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
