package main

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
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
// engine (semgrep/ast-grep) keeps its pinned Provision record while its PATH
// absence now fails loud.
//
// ISSUE-112 SUPERSEDES this test's former assertion (CLM-042: "absence on PATH
// must NOT fail provisioning"). Nothing in backstop installs a pinned tool — the
// Provision record is a TRUST allowlist pin — so a pinned engine whose binary is
// absent scanned nothing, and reporting that as a clean provision is the vacuous
// green ISSUE-112 reports. The name is kept for continuity with the SPEC-031
// history a reader following this test will want.
//
// The still-true half is preserved verbatim: both bindings genuinely carry pinned
// tool+version Provision records — the split is real, only its consequence was
// mis-stated.
func TestProvision_IntroducedEngineAutoProvisioned(t *testing.T) {
	// Nothing present on PATH.
	requested := recordingBinaryResolver(t /* nothing present */)

	m := &pack.Manifest{NormalizedName: "test/introduced"}
	m.Content.Ruleset.Rules = []pack.Rule{
		{ID: "s1", Engine: "semgrep", RulePath: "semgrep/x.yml", Standard: "x"},
		{ID: "a1", Engine: "ast-grep", RulePath: "ast-grep/y.yml", Standard: "x"},
	}

	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("pinned engines absent from PATH must fail loud, got nil — a pin is a trust record, never an install")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("an absent pinned tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	// Probing is deterministic and sorted, so `ast-grep` is the tool reached first
	// and named — the ordering provisionEngines is required to preserve.
	if !strings.Contains(cfgErr.Error(), "ast-grep") {
		t.Errorf("ConfigError must name the first absent tool in sorted probe order (`ast-grep`), got: %v", cfgErr)
	}
	if !sliceContains(*requested, "ast-grep") {
		t.Errorf("provisioning must PROBE the pinned tool on PATH, requested %v", *requested)
	}

	// STILL TRUE (unchanged by ISSUE-112): both engines carry pinned Provision
	// records naming a tool and a version.
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

// TestProvision_EnsureSemgrepRetired proves the bespoke EnsureSemgrep install
// ladder is off the provisioning path, using the STRONGER behavioral fingerprint
// ISSUE-112 makes available.
//
// ISSUE-112 SUPERSEDES this test's former fingerprint (CLM-043: "provisioning must
// NOT probe `semgrep`"). That fingerprint was inverted by the fix: provisioning now
// PROBES semgrep on PATH and REFUSES when it is absent — and still installs
// nothing. Probe-then-refuse is a sharper proof that no install ladder is on this
// path than probe-not-at-all ever was, because an install ladder would have made
// the absence recoverable. The name is kept for continuity with the SPEC-031
// history a reader following this test will want.
//
// The source-level half of "EnsureSemgrep is gone" is proved elsewhere and is not
// weakened here: TestInProcessSemgrepExecutor_Removed
// (pkg/check/semgrep_removal_test.go) greps the production sources. This test owns
// only the behavioral angle.
//
// The nil-Provision sibling contrast is preserved: golangci-lint is probed too, so
// the two branches are shown to reach the SAME presence check rather than one of
// them being dead.
func TestProvision_EnsureSemgrepRetired(t *testing.T) {
	// The rules below bind the built-in semgrep (pinned Provision) and golangci
	// (nil-Provision assume-present native) engines WITHOUT declaring them. After
	// ISSUE-027 golangci is pack DATA (the go-toolchain pack), not a baked binding, so
	// install the full built-in set on the seam — the same union production's
	// resolveEngineRegistry sees with the base + go-toolchain packs installed — so
	// golangci resolves to its nil-Provision, golangci-lint-command binding.
	origReg := engineRegistry
	t.Cleanup(func() { engineRegistry = origReg })
	engineRegistry = builtinTestRegistry(t)

	// golangci-lint present so the assume-present sibling passes; semgrep absent on
	// PATH. Sorted probe order reaches golangci-lint first (present, clears), then
	// semgrep (absent) — so both branches are exercised in one run.
	requested := recordingBinaryResolver(t, "golangci-lint")

	m := &pack.Manifest{NormalizedName: "test/semgrep-plus-native"}
	m.Content.Ruleset.Rules = []pack.Rule{
		{ID: "s1", Engine: "semgrep", RulePath: "semgrep/x.yml", Standard: "x"},
		{ID: "g1", Engine: "golangci", RulePath: "golangci/.golangci.yml", Standard: "x"},
	}

	err := provisionEngines([]*pack.Manifest{m})
	if err == nil {
		t.Fatal("semgrep absent from PATH must fail loud — no install ladder rescues it, got nil")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("an absent pinned tool must surface as *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if !strings.Contains(cfgErr.Error(), "semgrep") {
		t.Errorf("the refusal must name `semgrep`, got: %v", cfgErr)
	}

	// BEHAVIORAL FINGERPRINT: provisioning PROBED semgrep on PATH and, finding it
	// absent, REFUSED — it did not fetch it. An EnsureSemgrep install ladder on this
	// path would have made the absence recoverable instead of terminal.
	if !sliceContains(*requested, "semgrep") {
		t.Errorf("provisioning must PROBE `semgrep` on PATH (%v) — probe-then-refuse is what proves nothing installs it", *requested)
	}
	// CONTRAST: the nil-Provision native sibling (golangci-lint) is probed by the
	// SAME check, proving one presence check governs both branches rather than the
	// pinned branch having its own (or none).
	if !sliceContains(*requested, "golangci-lint") {
		t.Errorf("the assume-present native sibling must be probed by the same check, requested %v", *requested)
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
