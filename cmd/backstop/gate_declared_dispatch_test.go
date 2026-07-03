package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// spec046StepNamesContain reports whether the built gate step list for projectRoot
// includes a step with the given name.
func spec046StepNamesContain(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// TestGate_DeclaredToolchainPackDispatchedUniformly (CLM-004): a project declaring a
// toolchain pack ONLY in `packs:` (no `language:`) has that pack dispatched through
// the ordinary pack_engines / dispatchPackEngines step — toolchain packs flow
// through the declared-pack path like every other pack, with no language-derived
// bridge.
func TestGate_DeclaredToolchainPackDispatchedUniformly(t *testing.T) {
	root := spec046InstallGoToolchainProject(t, "spec046-declared-go-toolchain.yml")

	// The declared-pack path resolves the toolchain pack from `packs:`.
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("declared-pack path must resolve the toolchain pack: %v", err)
	}
	if countToolchainPacks(packs) != 1 {
		t.Fatalf("the declared toolchain pack must be resolved via packs: alone, got %d toolchain packs", countToolchainPacks(packs))
	}

	// The live gate dispatches it through the uniform pack_engines step.
	names := gateStepNames(t, root, emptyDiffScope())
	if !spec046StepNamesContain(names, "pack_engines") {
		t.Fatalf("a project declaring backstop/go-toolchain in packs: must dispatch it through the pack_engines step (the uniform declared-pack path); steps=%v", names)
	}
}

// TestGate_NoDeclaredToolchainPackYieldsWarnNotSynthesized (CLM-005): a project
// declaring NO toolchain pack and NO `language:` gets ZERO toolchain packs and the
// WARN-only "enforcement not configured" state — the deleted bridge no longer
// synthesizes a pack from a derived name (the prohibited-path negative).
func TestGate_NoDeclaredToolchainPackYieldsWarnNotSynthesized(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte("project: spec046-none\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	if countToolchainPacks(packs) != 0 {
		t.Fatalf("a project declaring NO toolchain pack (and NO language) must have ZERO toolchain packs — the bridge must not synthesize one, got %d", countToolchainPacks(packs))
	}
	warn, emitted := toolchainEnforcementStatus(packs)
	if !emitted {
		t.Fatal("zero declared toolchain packs must emit the WARN-only enforcement-not-configured state, not silence")
	}
	if !strings.Contains(warn.Reason, noToolchainPackMessage()) {
		t.Fatalf("the warn state must carry the loud no-toolchain-pack message, got %q", warn.Reason)
	}

	// Live gate: the warn step is present in the built step list.
	names := gateStepNames(t, root, emptyDiffScope())
	if !spec046StepNamesContain(names, toolchainEnforcementStepName()) {
		t.Fatalf("the no-toolchain-pack WARN step must appear in the built gate step list; steps=%v", names)
	}
}

// TestGate_DispatchesGoToolchainViaDeclaredPackPathWithoutLanguageField (CLM-007,
// MANDATED): the live gate STILL dispatches backstop/go-toolchain for the
// backstop-core-shape config via the DECLARED-pack path with NO `language:` field
// present — the dogfood no-regression guarantee after the bridge deletion.
func TestGate_DispatchesGoToolchainViaDeclaredPackPathWithoutLanguageField(t *testing.T) {
	// The fixture is the dogfood shape: it declares backstop/go-toolchain in packs:
	// and carries NO `language:` field at all.
	raw, err := os.ReadFile(spec046FixturePath(t, "spec046-declared-go-toolchain.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "language:") {
		t.Fatal("the no-regression fixture must carry NO `language:` field — CLM-007 proves dispatch works without it")
	}

	root := spec046InstallGoToolchainProject(t, "spec046-declared-go-toolchain.yml")
	names := gateStepNames(t, root, emptyDiffScope())
	if !spec046StepNamesContain(names, "pack_engines") {
		t.Fatalf("the gate must STILL dispatch backstop/go-toolchain via the declared-pack path with no language: field — the dogfood must not regress; steps=%v", names)
	}
}

// TestGate_PolyglotDeclaredToolchainPacksBothDispatched (CLM-008): a repo declaring
// BOTH backstop/go-toolchain and backstop/bun-toolchain dispatches BOTH through the
// uniform declared-pack path — more than one toolchain pack per repo. Proven over
// declared-pack manifest STUBS because the bun-toolchain pack does not exist until
// SPEC-047; both stubs survive into the dispatch set the pack_engines step consumes.
func TestGate_PolyglotDeclaredToolchainPacksBothDispatched(t *testing.T) {
	declared := []*pack.Manifest{
		spec046ToolchainManifest("backstop/go-toolchain"),
		spec046ToolchainManifest("backstop/bun-toolchain"),
	}
	if countToolchainPacks(declared) != 2 {
		t.Fatalf("a polyglot repo declaring two toolchain packs must count both, got %d", countToolchainPacks(declared))
	}
	// The dispatch set the pack_engines step consumes is excludeDedicatedStepRules(packs);
	// both toolchain packs must survive into it (dispatched uniformly, not special-cased).
	dispatchSet := excludeDedicatedStepRules(declared)
	got := map[string]bool{}
	for _, m := range dispatchSet {
		got[m.NormalizedName] = true
	}
	for _, want := range []string{"backstop/go-toolchain", "backstop/bun-toolchain"} {
		if !got[want] {
			t.Errorf("polyglot toolchain pack %q must flow into the uniform dispatch set, missing", want)
		}
	}
}

// TestGate_CountToolchainPacksCountsDeclaredOnly (CLM-009): countToolchainPacks
// takes NO `bridged` argument and returns the count of `packs:` entries whose
// normalized name ends in `-toolchain` — declared packs only.
func TestGate_CountToolchainPacksCountsDeclaredOnly(t *testing.T) {
	declared := []*pack.Manifest{
		spec046ToolchainManifest("backstop/go-toolchain"),
		spec046ToolchainManifest("backstop/bun-toolchain"),
		{Name: "backstop/go-standards", NormalizedName: "backstop/go-standards"}, // NOT a toolchain pack
		{Name: "backstop/contracts", NormalizedName: "backstop/contracts"},       // NOT a toolchain pack
	}
	if got := countToolchainPacks(declared); got != 2 {
		t.Fatalf("countToolchainPacks must count ONLY the declared `-toolchain` packs, got %d want 2", got)
	}
	if got := countToolchainPacks(nil); got != 0 {
		t.Fatalf("countToolchainPacks(nil) must be 0, got %d", got)
	}
}

// TestGate_ToolchainEnforcementKeysOnDeclaredOnly (CLM-010): toolchainEnforcementStatus
// emits the WARN-only state iff ZERO toolchain packs are DECLARED (no `bridged`
// input) and is suppressed when at least one is declared.
func TestGate_ToolchainEnforcementKeysOnDeclaredOnly(t *testing.T) {
	// Zero declared toolchain packs -> warn emitted.
	warn, emitted := toolchainEnforcementStatus([]*pack.Manifest{
		{Name: "backstop/go-standards", NormalizedName: "backstop/go-standards"},
	})
	if !emitted {
		t.Fatal("zero DECLARED toolchain packs must emit the warn state")
	}
	if warn.Status != "warning" {
		t.Fatalf("the no-toolchain-pack state must be a non-failing warning, got %q", warn.Status)
	}

	// At least one declared toolchain pack -> suppressed.
	if _, emitted := toolchainEnforcementStatus([]*pack.Manifest{spec046ToolchainManifest("backstop/go-toolchain")}); emitted {
		t.Fatal("a declared toolchain pack must SUPPRESS the no-toolchain-pack warn state")
	}
}
