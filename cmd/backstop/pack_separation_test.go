package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// SPEC-034 REQ-007 / Sharp Edge 9 — the mechanism/opinion pack boundary. The
// build/test/lint TOOLCHAIN MECHANISM lives in the reusable backstop/go-toolchain
// pack; the coding-standards OPINION lives in the separate backstop/go-standards
// pack. These tests assert the decomposition holds in BOTH directions (no bleed)
// and that the toolchain pack lands in lockstep with the bridge.

// goStandardsPackManifest loads the real installed backstop/go-standards opinion
// pack manifest from the repo's .backstop/packs.
func goStandardsPackManifest(t *testing.T) *pack.Manifest {
	t.Helper()
	p := filepath.Join(repoRoot(t), ".backstop", "packs", "backstop", "go-standards", "pack.yml")
	m, err := pack.ParseManifestFile(p)
	if err != nil {
		t.Fatalf("backstop/go-standards pack must parse: %v", err)
	}
	return m
}

// TestGoToolchainPack_SeparateFromGoStandardsPack (CLM-022) proves the build/test
// convert scripts (run via the sandbox mechanism) live in a reusable go-toolchain
// pack that is a DISTINCT artifact from the opinionated go-standards pack: the two
// manifests are different packs (different names + directories), and the convert
// scripts exist under the go-toolchain pack but NOT under the go-standards pack.
func TestGoToolchainPack_SeparateFromGoStandardsPack(t *testing.T) {
	toolchain := goToolchainManifest(t)
	standards := goStandardsPackManifest(t)

	if toolchain.Name == standards.Name {
		t.Fatalf("the toolchain and standards packs must be distinct artifacts, both named %q", toolchain.Name)
	}
	if toolchain.Name != "backstop/go-toolchain" {
		t.Errorf("toolchain pack name = %q, want backstop/go-toolchain", toolchain.Name)
	}
	if standards.Name != "backstop/go-standards" {
		t.Errorf("standards pack name = %q, want backstop/go-standards", standards.Name)
	}

	// The build/test convert scripts live under the go-toolchain pack.
	toolchainRoot := goToolchainPackRoot(t)
	for _, script := range []string{"build-to-sarif.sh", "test-to-sarif.sh"} {
		p := filepath.Join(toolchainRoot, "scripts", script)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("toolchain pack must own the convert script %s: %v", script, err)
		}
	}

	// The standards pack must NOT carry the toolchain convert scripts.
	standardsRoot := filepath.Dir(filepath.Join(repoRoot(t), ".backstop", "packs", "backstop", "go-standards", "pack.yml"))
	for _, script := range []string{"build-to-sarif.sh", "test-to-sarif.sh"} {
		p := filepath.Join(standardsRoot, "scripts", script)
		if _, err := os.Stat(p); err == nil {
			t.Errorf("standards pack must NOT carry the toolchain convert script %s (mechanism bleed)", script)
		}
	}
}

// TestGoToolchainPack_MechanismOnlyNoStandards (CLM-023) proves the go-toolchain
// pack contains ONLY toolchain mechanism (run + normalize via the go-build/go-test/
// golangci engines) and NO coding-standards rules: every rule binds a toolchain
// mechanism engine, none binds a standards engine (semgrep/ast-grep), and none
// carries a `standard:` declaration (the opinion marker).
func TestGoToolchainPack_MechanismOnlyNoStandards(t *testing.T) {
	m := goToolchainManifest(t)
	rules := m.Content.Ruleset.Rules
	if len(rules) == 0 {
		t.Fatal("go-toolchain pack must declare its toolchain rules")
	}

	violations := packSeparationViolations(m)
	for _, v := range violations {
		t.Errorf("go-toolchain mechanism bleed: %s", v)
	}

	// Direct structural restatement of the contract: every rule is a toolchain
	// mechanism engine; none is a standards engine; none has a `standard:` field.
	// The classification resolves through the pack manifest (resolveEngineRegistry(m))
	// so a PACK-declared mechanism engine (e.g. the SPEC-042 go-coverage producer,
	// category: mechanism, declared in the engines: block) classifies correctly — the
	// baked-registry-only (nil) lookup would miss a pack-declared engine.
	for _, r := range rules {
		if !isToolchainMechanismEngine(m, r.Engine) {
			t.Errorf("rule %q binds engine %q, which is not a toolchain mechanism engine — standards opinion must not live in the toolchain pack", r.ID, r.Engine)
		}
		if isStandardsOpinionEngine(m, r.Engine) {
			t.Errorf("rule %q binds standards engine %q in the toolchain pack (opinion bleed)", r.ID, r.Engine)
		}
		if r.Standard != "" {
			t.Errorf("rule %q carries a `standard:` declaration in the mechanism pack (opinion bleed): %q", r.ID, r.Standard)
		}
	}
}

// TestGoStandardsPack_OpinionOnlyNoToolchain (CLM-024) proves backstop-go-pack
// (backstop/go-standards) contains ONLY coding-standards opinion and NO build/test
// toolchain mechanism: every rule binds a standards engine (semgrep), none binds a
// toolchain mechanism engine (go-build/go-test/golangci).
func TestGoStandardsPack_OpinionOnlyNoToolchain(t *testing.T) {
	m := goStandardsPackManifest(t)
	rules := m.Content.Ruleset.Rules
	if len(rules) == 0 {
		t.Fatal("go-standards pack must declare its coding-standards rules")
	}

	violations := packSeparationViolations(m)
	for _, v := range violations {
		t.Errorf("go-standards opinion-pack bleed: %s", v)
	}

	for _, r := range rules {
		if isToolchainMechanismEngine(nil, r.Engine) {
			t.Errorf("rule %q binds toolchain mechanism engine %q in the standards pack (mechanism bleed)", r.ID, r.Engine)
		}
		if !isStandardsOpinionEngine(nil, r.Engine) {
			t.Errorf("rule %q binds engine %q, which is not a standards opinion engine — the standards pack carries opinion only", r.ID, r.Engine)
		}
	}
}

// TestGoToolchainPack_LandsInLockstepWithBridge (CLM-025) proves the bridge and the
// go-toolchain pack are wired to land in lockstep so no commit leaves Go build/test
// unenforced: EVERY engine the go-toolchain pack's rules declare is registered in
// the bridge's engine registry (the same registry dispatchPackEngines resolves
// through). A pack rule referencing an engine the bridge cannot run would be the
// enforcement-lapse window; this asserts the pack and the bridge agree on the
// engine set, so the pack is dispatchable the moment it is installed.
func TestGoToolchainPack_LandsInLockstepWithBridge(t *testing.T) {
	m := goToolchainManifest(t)
	// Resolve through the pack manifest so a PACK-declared engine (the SPEC-042
	// go-coverage producer, declared in the engines: block) is in lockstep too — the
	// baked-registry-only (nil) lookup would miss it and falsely read as a drift.
	reg := resolveEngineRegistry(m)

	var sawBuild, sawTest, sawLint bool
	for _, r := range m.Content.Ruleset.Rules {
		bind, err := reg.Lookup(r.Engine)
		if err != nil {
			t.Fatalf("go-toolchain rule %q declares engine %q that the bridge registry cannot run — the pack and bridge are NOT in lockstep (enforcement lapse): %v", r.ID, r.Engine, err)
		}
		// The toolchain engines must carry a real run command (build/test/lint),
		// not an empty binding — proving the bridge actually runs them.
		switch r.Engine {
		case "go-build":
			sawBuild = true
			if bind.Command == "" {
				t.Errorf("go-build engine binding has no command; the bridge cannot run the build pass")
			}
		case "go-test":
			sawTest = true
			if bind.Command == "" {
				t.Errorf("go-test engine binding has no command; the bridge cannot run the test pass")
			}
		case "golangci":
			sawLint = true
			if bind.Command == "" {
				t.Errorf("golangci engine binding has no command; the bridge cannot run the lint pass")
			}
		}
	}
	if !sawBuild || !sawTest || !sawLint {
		t.Errorf("the go-toolchain pack must carry all three native passes wired to the bridge (build=%v test=%v lint=%v)", sawBuild, sawTest, sawLint)
	}

	// Lockstep restatement: the bridge's registry is the SAME engine.Registry the
	// pack binds against, so there is no separate engine set that could drift.
	if _, ok := interface{}(reg).(engine.Registry); !ok {
		t.Errorf("the bridge must resolve pack engines through engine.Registry, got %T", reg)
	}
}

// TestPackSeparation_RejectsBleed proves the boundary enforcer is NOT vacuous: a
// synthetic pack that mixes a toolchain mechanism rule with a coding-standards
// opinion rule (and an opinion rule carrying a `standard:` marker alongside a
// mechanism engine) is REJECTED with violations. Without this adversarial case the
// no-bleed assertions above could pass on any input.
func TestPackSeparation_RejectsBleed(t *testing.T) {
	// The synthetic packs below reference the built-in engine NAMES (go-build,
	// golangci, semgrep) WITHOUT declaring them in their own engines: block, so the
	// classification must resolve them from the built-in registry. After ISSUE-027
	// the built-ins are pack DATA (the generic engines from the embedded base pack +
	// the Go toolchain engines from the go-toolchain pack); install the full built-in
	// set on the seam so go-build/golangci classify as mechanism and semgrep as
	// opinion — exactly the union production's resolveEngineRegistry sees once those
	// packs are installed. (A real pack declares its own engines and resolves them via
	// the m.Engines merge — proven non-vacuous by TestGoToolchainPack_MechanismOnlyNoStandards
	// running the check over the REAL go-toolchain manifest.)
	orig := engineRegistry
	t.Cleanup(func() { engineRegistry = orig })
	engineRegistry = builtinTestRegistry(t)

	// A pack mixing mechanism (go-build) and opinion (semgrep) rules.
	mixed := &pack.Manifest{
		Name: "backstop/bad-mixed",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "build", Engine: "go-build"},
			{ID: "no-panic", Engine: "semgrep", Standard: "No panic in library code"},
		}}},
	}
	v := packSeparationViolations(mixed)
	if len(v) == 0 {
		t.Fatal("a pack mixing toolchain mechanism and coding-standards opinion must be rejected, got no violations")
	}

	// A mechanism-only pack whose rule wrongly carries a `standard:` opinion marker.
	mechWithStandard := &pack.Manifest{
		Name: "backstop/bad-mech",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "build", Engine: "go-build", Standard: "leaked opinion"},
		}}},
	}
	if v := packSeparationViolations(mechWithStandard); len(v) == 0 {
		t.Error("a mechanism rule carrying a `standard:` declaration must be rejected as opinion bleed")
	}

	// An opinion-only pack whose rule wrongly binds a toolchain mechanism engine.
	opinionWithMech := &pack.Manifest{
		Name: "backstop/bad-opinion",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "no-panic", Engine: "semgrep", Standard: "No panic"},
			{ID: "sneaky-build", Engine: "golangci"},
		}}},
	}
	if v := packSeparationViolations(opinionWithMech); len(v) == 0 {
		t.Error("an opinion pack binding a toolchain mechanism engine must be rejected as mechanism bleed")
	}

	// A clean mechanism-only pack and a clean opinion-only pack produce no violations.
	cleanMech := &pack.Manifest{
		Name:    "backstop/clean-mech",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{{ID: "build", Engine: "go-build"}, {ID: "lint", Engine: "golangci"}}}},
	}
	if v := packSeparationViolations(cleanMech); len(v) != 0 {
		t.Errorf("a clean mechanism-only pack must produce no violations, got %v", v)
	}
	cleanOpinion := &pack.Manifest{
		Name:    "backstop/clean-opinion",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{{ID: "no-panic", Engine: "semgrep", Standard: "x"}}}},
	}
	if v := packSeparationViolations(cleanOpinion); len(v) != 0 {
		t.Errorf("a clean opinion-only pack must produce no violations, got %v", v)
	}
}

// TestPackSeparation_CategoryDrivesClassification proves the mechanism/opinion
// boundary derives ENTIRELY from EngineBinding.Category in the registry (ISSUE-015),
// not a hardcoded engine set in pack_separation.go. A brand-new engine registered
// purely as data — one tagged EngineCategoryMechanism, one tagged
// EngineCategoryOpinion, one left EngineCategoryUnset — is classified correctly by
// the SAME isToolchainMechanismEngine/isStandardsOpinionEngine helpers WITHOUT any
// edit to pack_separation.go, and a pack mixing the two new engines is rejected as
// bleed. This is the single-source-of-truth assertion: adding an engine is one
// Category declaration, and the separation enforcer picks it up automatically.
func TestPackSeparation_CategoryDrivesClassification(t *testing.T) {
	orig := engineRegistry
	t.Cleanup(func() { engineRegistry = orig })
	engineRegistry = builtinTestRegistry(t)

	// Three brand-new engines added as DATA ONLY (no edit to pack_separation.go):
	// classified by their declared Category.
	engineRegistry["newmech"] = engine.EngineBinding{
		Command:  "newmech run",
		Category: engine.EngineCategoryMechanism,
	}
	engineRegistry["newopinion"] = engine.EngineBinding{
		Command:  "newopinion scan",
		Category: engine.EngineCategoryOpinion,
	}
	engineRegistry["newneutral"] = engine.EngineBinding{
		Command: "newneutral run",
		// EngineCategoryUnset (zero value) — neither mechanism nor opinion.
	}

	// The classifiers read Category off the registry, so the new engines are
	// classified with no change to the helpers themselves.
	if !isToolchainMechanismEngine(nil, "newmech") {
		t.Error("an engine declared EngineCategoryMechanism must classify as toolchain mechanism")
	}
	if isStandardsOpinionEngine(nil, "newmech") {
		t.Error("a mechanism engine must NOT classify as opinion")
	}
	if !isStandardsOpinionEngine(nil, "newopinion") {
		t.Error("an engine declared EngineCategoryOpinion must classify as standards opinion")
	}
	if isToolchainMechanismEngine(nil, "newopinion") {
		t.Error("an opinion engine must NOT classify as mechanism")
	}
	// EngineCategoryUnset => neither (same as the pre-ISSUE-015 false/false), and an
	// engine absent from the registry resolves the same neutral way.
	if isToolchainMechanismEngine(nil, "newneutral") || isStandardsOpinionEngine(nil, "newneutral") {
		t.Error("an EngineCategoryUnset engine must classify as neither mechanism nor opinion")
	}
	if isToolchainMechanismEngine(nil, "does-not-exist") || isStandardsOpinionEngine(nil, "does-not-exist") {
		t.Error("an unregistered engine must classify as neither mechanism nor opinion")
	}

	// End-to-end: a pack mixing the new mechanism engine with the new opinion engine
	// is rejected as bleed — driven solely by the declared Categories.
	mixed := &pack.Manifest{
		Name: "backstop/new-mixed",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "m", Engine: "newmech"},
			{ID: "o", Engine: "newopinion", Standard: "some opinion"},
		}}},
	}
	if v := packSeparationViolations(mixed); len(v) == 0 {
		t.Error("a pack mixing a Category=Mechanism and a Category=Opinion engine must be rejected as bleed")
	}

	// A clean pack of only the neutral engine produces no violations (it is invisible
	// to the boundary, exactly like sandbox/config-file).
	neutral := &pack.Manifest{
		Name:    "backstop/new-neutral",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{{ID: "n", Engine: "newneutral"}}}},
	}
	if v := packSeparationViolations(neutral); len(v) != 0 {
		t.Errorf("a pack of only EngineCategoryUnset engines must produce no violations, got %v", v)
	}
}

// TestPackSeparation_NilAndEmptySafe pins the defensive guards: a nil manifest and
// an empty (no-rules) manifest are classified as neither mechanism nor opinion and
// produce no violations, so the boundary check never panics on a degenerate pack.
func TestPackSeparation_NilAndEmptySafe(t *testing.T) {
	if c := classifyPack(nil); c.HasMechanism || c.HasOpinion {
		t.Errorf("nil manifest must classify as neither mechanism nor opinion, got %+v", c)
	}
	if v := packSeparationViolations(nil); len(v) != 0 {
		t.Errorf("nil manifest must produce no violations, got %v", v)
	}

	empty := &pack.Manifest{Name: "backstop/empty"}
	c := classifyPack(empty)
	if c.HasMechanism || c.HasOpinion {
		t.Errorf("empty pack must classify as neither mechanism nor opinion, got %+v", c)
	}
	if v := packSeparationViolations(empty); len(v) != 0 {
		t.Errorf("empty pack must produce no violations, got %v", v)
	}
}

// TestPackSeparation_ReaderRunsNoCommandNoAllowlistNeeded proves the
// pack-separation reader (the THIRD resolveEngineRegistry caller) looks up only a
// binding's Category and runs NO command, so it is explicitly EXEMPT from the
// trusted-tool allowlist gate (SPEC-035 REQ-003/CLM-031). With the allowlist
// stubbed EMPTY — under which ANY tool would fail an allowlist check — the
// separation classification of a pack binding an un-allowlisted engine still
// succeeds and runs no command: an un-allowlisted engine here is not an execution
// path, so gating it would be wrong.
func TestPackSeparation_ReaderRunsNoCommandNoAllowlistNeeded(t *testing.T) {
	// Empty allowlist: if the separation reader consulted the allowlist, every
	// tool would be rejected. It must not consult it at all.
	orig := trustedToolAllowlist
	trustedToolAllowlist = func() map[string]string { return map[string]string{} }
	t.Cleanup(func() { trustedToolAllowlist = orig })

	// A pack declaring a provisioned engine whose tool is absent from the (empty)
	// allowlist. The separation reader classifies it by Category only.
	m := &pack.Manifest{
		Name:           "acme/separation-reader",
		NormalizedName: "acme/separation-reader",
		Engines: map[string]pack.EngineSpec{
			"acme-opinion": {Binding: engine.EngineBinding{
				Command:   "acme-never-allowlisted scan",
				InputMode: engine.InputModeRuleFlags,
				InputFlag: "--config",
				ScopeKind: engine.ScopeKindFileArgs,
				Provision: &engine.Provision{Tool: "acme-never-allowlisted", Version: "1.0.0"},
				Category:  engine.EngineCategoryOpinion,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "acme-opinion", Standard: "x"},
		}}},
	}

	// The reader resolves the engine's Category — NO command run, NO allowlist
	// consulted — and classifies the pack as opinion. It must not panic or error
	// on the un-allowlisted tool, because it never tries to run it.
	if !isStandardsOpinionEngine(m, "acme-opinion") {
		t.Error("the separation reader must classify the engine by its declared Category (opinion), reading no command and no allowlist")
	}
	if isToolchainMechanismEngine(m, "acme-opinion") {
		t.Error("an opinion-category engine must not classify as mechanism")
	}
	cls := classifyPack(m)
	if !cls.HasOpinion {
		t.Errorf("the pack must classify as opinion via the Category-only reader, got %+v", cls)
	}
	if v := packSeparationViolations(m); len(v) != 0 {
		t.Errorf("a single-opinion pack produces no separation violations regardless of the (empty) allowlist, got %v", v)
	}
}
