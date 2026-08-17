package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/baseengines"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

func TestGateIntegration_LoadsPacksFromConfig(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")

	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	if len(packs) != 1 {
		t.Fatalf("expected 1 pack, got %d", len(packs))
	}
	if packs[0].NormalizedName != "test-org/test-pack" {
		t.Fatalf("unexpected pack name: %q", packs[0].NormalizedName)
	}
}

func TestGateIntegration_NoPacks(t *testing.T) {
	projectRoot := t.TempDir()
	copyFixtureFile(t,
		filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "packgate", "backstop-no-packs.yml"),
		filepath.Join(projectRoot, "backstop.yml"))

	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	if len(packs) != 0 {
		t.Fatalf("expected no packs, got %d", len(packs))
	}
}

func TestGateIntegration_MissingPackDir(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(`project: test
language: go
packs:
  test-org/missing-pack: "1.0.0"
`), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}

	_, err := loadInstalledPacks(projectRoot)
	if err == nil {
		t.Fatal("expected error for missing pack dir")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing pack diagnostic, got: %v", err)
	}
}

func TestGateIntegration_LockRunsFirst(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate-hash-mismatch")
	steps := buildGateSteps(projectRoot, rootAtDir(t, projectRoot))
	if len(steps) == 0 {
		t.Fatal("expected steps")
	}
	first := steps[0](context.Background())
	if first.StepName != "pack_lock_verification" {
		t.Fatalf("expected first step to be pack lock verification, got %q", first.StepName)
	}
	if first.Status != "fail" {
		t.Fatalf("expected lock failure, got %q", first.Status)
	}
}

func TestGateIntegration_MissingLockfile(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	if err := os.Remove(filepath.Join(projectRoot, "backstop.lock")); err != nil {
		t.Fatalf("remove lockfile: %v", err)
	}

	err := verifyPackLock(projectRoot, []string{"test-org/test-pack"})
	if err == nil {
		t.Fatal("expected missing lockfile error")
	}
	if !strings.Contains(err.Error(), "missing_lockfile") {
		t.Fatalf("expected missing_lockfile diagnostic, got: %v", err)
	}
}

func TestGateIntegration_ExtraUnlockedPack(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.lock"), []byte("packs: {}\n"), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
	err := verifyPackLock(projectRoot, []string{"test-org/test-pack"})
	if err == nil {
		t.Fatal("expected extra unlocked pack error")
	}
	if !strings.Contains(err.Error(), "extra_unlocked") {
		t.Fatalf("expected extra_unlocked diagnostic, got: %v", err)
	}
}

func TestGateIntegration_LockHashMismatch(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate-hash-mismatch")
	err := verifyPackLock(projectRoot, []string{"test-org/test-pack"})
	if err == nil {
		t.Fatal("expected hash mismatch error")
	}
	if !strings.Contains(err.Error(), "hash_mismatch") {
		t.Fatalf("expected hash_mismatch diagnostic, got: %v", err)
	}
}

// TestGateIntegration_SemgrepRulesMerged asserts dispatchPackEngines gathers the
// engine: semgrep rule-flags input as one absolute rule path resolved relative
// to the per-engine pack dir. Re-keyed from the retired mergePackRules direct
// caller onto a capturingRunner that records the semgrep --config args.
func TestGateIntegration_SemgrepRulesMerged(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}
	orig := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) { return nil, nil }
	defer func() { sandboxedRun = orig }()

	if _, err := dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, rec); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	var rulePaths []string
	for i := 0; i+1 < len(rec.lastArgs); i++ {
		if rec.lastArgs[i] == "--config" {
			rulePaths = append(rulePaths, rec.lastArgs[i+1])
		}
	}
	if len(rulePaths) != 1 {
		t.Fatalf("expected 1 semgrep rule path, got %d: %v", len(rulePaths), rec.lastArgs)
	}
	if !filepath.IsAbs(rulePaths[0]) {
		t.Fatalf("expected absolute rule path, got %q", rulePaths[0])
	}
	if !strings.HasSuffix(rulePaths[0], "rules/no-eval.yml") {
		t.Fatalf("unexpected rule path: %q", rulePaths[0])
	}
}

func TestGateIntegration_NamespacedRuleIDs(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	if len(packs) != 1 || len(packs[0].Content.Ruleset.Rules) == 0 {
		t.Fatalf("expected rules in fixture pack")
	}
	got := packs[0].Content.Ruleset.Rules[0].NamespacedID
	if !strings.HasPrefix(got, "test-org/test-pack/") {
		t.Fatalf("expected namespaced rule id, got %q", got)
	}
}

// TestGateIntegration_SandboxValidatorExecuted asserts the engine==sandbox
// exit-code branch of dispatchPackEngines invokes the validator on the full
// project root. Re-keyed from TestGateIntegration_Layer3ValidatorExecuted (the
// fixture rule is now engine: sandbox).
func TestGateIntegration_SandboxValidatorExecuted(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}

	var called bool
	orig := sandboxedRun
	sandboxedRun = func(cmd string, args []string, packDir string) ([]byte, error) {
		called = true
		if !strings.Contains(cmd, "check-middleware.sh") {
			t.Fatalf("unexpected validator command %q", cmd)
		}
		if len(args) != 1 || args[0] != projectRoot {
			t.Fatalf("expected full-project arg %q, got %#v", projectRoot, args)
		}
		return []byte("middleware.go missing"), errors.New("exit status 1")
	}
	defer func() { sandboxedRun = orig }()

	violations, err := dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, emptySarifRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if !called {
		t.Fatal("expected validator execution")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

// TestGateIntegration_SandboxNamespacedIDs asserts the namespaced violation id
// survives the re-keyed engine==sandbox exit-code branch. Re-keyed from
// TestGateIntegration_Layer3NamespacedIDs.
func TestGateIntegration_SandboxNamespacedIDs(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}

	orig := sandboxedRun
	sandboxedRun = func(cmd string, args []string, packDir string) ([]byte, error) {
		return []byte("middleware.go missing"), errors.New("exit status 1")
	}
	defer func() { sandboxedRun = orig }()

	violations, err := dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, emptySarifRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected validator violation")
	}
	if !strings.HasPrefix(violations[0].Rule, "test-org/test-pack/") {
		t.Fatalf("expected namespaced rule id, got %q", violations[0].Rule)
	}
}

func TestGateIntegration_BrokenPackRuleFile(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate-broken")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}

	_, err = dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, emptySarifRunner{})
	if err == nil {
		t.Fatal("expected error for missing pack rule file")
	}
	if !strings.Contains(err.Error(), "does-not-exist.yml") {
		t.Fatalf("expected missing file diagnostic, got: %v", err)
	}
	if !strings.Contains(err.Error(), "broken pack") {
		t.Fatalf("expected broken-pack diagnostic naming the pack, got: %v", err)
	}
}

func TestGateIntegration_ToolConfigApplied(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	if len(packs[0].ToolConfig) == 0 {
		t.Fatal("fixture pack should include tool_config")
	}
	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}
	orig := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) { return nil, nil }
	defer func() { sandboxedRun = orig }()
	if _, err := dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, rec); err != nil {
		t.Fatalf("dispatchPackEngines should not require runtime tool_config merge: %v", err)
	}
	configs := 0
	for i := 0; i+1 < len(rec.lastArgs); i++ {
		if rec.lastArgs[i] == "--config" {
			configs++
		}
	}
	if configs != 1 {
		t.Fatalf("expected one semgrep rule input while ignoring tool_config runtime merge, got %d", configs)
	}
}

func TestGateIntegration_MultiplePacksEnforced(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate-multi")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	if len(packs) != 2 {
		t.Fatalf("expected 2 packs, got %d", len(packs))
	}

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}

	orig := sandboxedRun
	sandboxedRun = func(cmd string, args []string, packDir string) ([]byte, error) {
		if strings.Contains(cmd, "check-printf.sh") {
			return []byte("fmt.Printf usage detected"), errors.New("exit status 1")
		}
		return []byte("ok"), nil
	}
	defer func() { sandboxedRun = orig }()

	violations, err := dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, rec)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	// First pack supplies one engine: semgrep rule input (gathered as --config).
	if len(rec.allConfigs) < 1 {
		t.Fatalf("expected the first pack's semgrep rule input gathered, got %v", rec.allConfigs)
	}
	if len(violations) != 1 {
		t.Fatalf("expected one engine: sandbox violation from second pack, got %d", len(violations))
	}
}

func TestGateIntegration_MultiPackAttribution(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate-multi")
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}

	// Add a synthetic engine: sandbox rule to the second pack (test-pack) so both
	// packs emit through the re-keyed exit-code branch.
	packs[1].Content.Ruleset.Rules = append(packs[1].Content.Ruleset.Rules, pack.Rule{
		ID:           "test-pack-validator",
		NamespacedID: pack.NamespacedRuleID(packs[1].NormalizedName, "test-pack-validator"),
		Engine:       "sandbox",
		Validator:    filepath.Join("..", "other-pack", "validators", "check-printf.sh"),
	})

	orig := sandboxedRun
	sandboxedRun = func(cmd string, args []string, packDir string) ([]byte, error) {
		return []byte("violation"), errors.New("exit status 1")
	}
	defer func() { sandboxedRun = orig }()

	violations, err := dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, emptySarifRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) < 2 {
		t.Fatalf("expected at least 2 violations for attribution, got %d", len(violations))
	}

	foundFirst := false
	foundSecond := false
	for _, violation := range violations {
		if strings.HasPrefix(violation.Rule, "test-org/test-pack/") && violation.SourcePack == "test-org/test-pack" {
			foundFirst = true
		}
		if strings.HasPrefix(violation.Rule, "test-org/other-pack/") && violation.SourcePack == "test-org/other-pack" {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Fatalf("expected per-pack attribution, got violations: %#v", violations)
	}
}

// TestGateIntegration_SandboxSingleFileScope verifies that single-file
// input_scope validators are invoked per-file, not once with the project root,
// through the re-keyed engine==sandbox branch. Re-keyed from
// TestGateIntegration_Layer3SingleFileScope.
func TestGateIntegration_SandboxSingleFileScope(t *testing.T) {
	projectRoot := t.TempDir()
	// Create source files
	if err := os.WriteFile(filepath.Join(projectRoot, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("writing main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "util.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("writing util.go: %v", err)
	}

	// Create pack with single-file validator
	packDir := filepath.Join(projectRoot, ".backstop", "packs", "test-org", "sf-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("creating pack dir: %v", err)
	}
	validatorScript := filepath.Join(packDir, "check.sh")
	if err := os.WriteFile(validatorScript, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("writing validator script: %v", err)
	}

	manifests := []*pack.Manifest{{
		Name:           "test-org/sf-pack",
		NormalizedName: "test-org/sf-pack",
		Content: pack.Content{
			Ruleset: pack.Ruleset{
				Rules: []pack.Rule{{
					ID:         "sf-check",
					Engine:     "sandbox",
					Validator:  "check.sh",
					InputScope: "single-file",
				}},
			},
		},
	}}

	var calls []string
	orig := sandboxedRun
	sandboxedRun = func(cmd string, args []string, dir string) ([]byte, error) {
		if len(args) > 0 {
			calls = append(calls, args[0])
		}
		return []byte("fail"), errors.New("exit status 1")
	}
	defer func() { sandboxedRun = orig }()

	violations, err := dispatchPackEngines(manifests, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, emptySarifRunner{})
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	// Should have been called per-file, not once with project root
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 single-file calls, got %d: %v", len(calls), calls)
	}
	// Each violation should be per-file
	if len(violations) < 2 {
		t.Fatalf("expected at least 2 violations (one per file), got %d", len(violations))
	}
	for _, v := range violations {
		if v.SourcePack != "test-org/sf-pack" {
			t.Errorf("expected SourcePack 'test-org/sf-pack', got %q", v.SourcePack)
		}
	}
}

// packDeclaredFindingsManifest builds an in-memory manifest carrying a top-level
// engines: block that declares a NEW findings engine ("acme-findings") plus one
// rule bound to it. The engine is SARIF-native (no Convert) and provisioned via a
// pinned record whose tool is on the test allowlist, so the dispatch trust gate
// passes when the lock matches the pin. This is the SPEC-035 pack-declared-binding
// fixture: the engine exists ONLY in the manifest's engines block, never in
// DefaultRegistry, so a successful dispatch proves the merge made it reachable.
func packDeclaredFindingsManifest(t *testing.T, packDir string, command, tool, version string) *pack.Manifest {
	t.Helper()
	packRoot := filepath.Join(packDir, "acme", "pack")
	if err := os.MkdirAll(filepath.Join(packRoot, "rules"), 0o755); err != nil {
		t.Fatalf("mkdir pack rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "rules", "r.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}
	return &pack.Manifest{
		NormalizedName: "acme/pack",
		Engines: map[string]pack.EngineSpec{
			"acme-findings": {
				Binding: engineBindingForTest(command, tool, version),
			},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "acme-findings", RulePath: "rules/r.yml", Standard: "x"},
		}}},
	}
}

// engineBindingForTest constructs an engine.EngineBinding for a pack-declared
// rule-flags findings engine, mirroring what parseEngineSpec would yield, with a
// pinned Provision so the dispatch trust gate has a tool+version to check.
func engineBindingForTest(command, tool, version string) engine.EngineBinding {
	return engine.EngineBinding{
		Command:   command,
		InputMode: engine.InputModeRuleFlags,
		InputFlag: "--config",
		ScopeKind: engine.ScopeKindFileArgs,
		Provision: &engine.Provision{Tool: tool, Version: version},
		Category:  engine.EngineCategoryOpinion,
	}
}

// TestResolveRegistry_PackDeclaredBindingMerged proves a pack-declared engine
// binding is merged into the registry resolveEngineRegistry returns, so a rule on
// that engine resolves to the pack-declared binding (SPEC-035 CLM-002). The engine
// "acme-findings" exists ONLY in the manifest's engines: block — never in
// DefaultRegistry — so its presence in the resolved registry proves the merge.
func TestResolveRegistry_PackDeclaredBindingMerged(t *testing.T) {
	m := packDeclaredFindingsManifest(t, t.TempDir(), "acme-scan --sarif", "acme-scan", "2.3.1")

	// Precondition: the engine is genuinely absent from the fallback registry, so a
	// hit can only come from the merge.
	if _, err := baseengines.Registry().Lookup("acme-findings"); err == nil {
		t.Fatal("test precondition: acme-findings must NOT be a built-in engine")
	}

	reg := resolveEngineRegistry(m)
	binding, err := reg.Lookup("acme-findings")
	if err != nil {
		t.Fatalf("pack-declared engine must be merged into the resolved registry: %v", err)
	}
	if binding.Command != "acme-scan --sarif" {
		t.Errorf("resolved binding must be the pack-declared one, got Command=%q", binding.Command)
	}
}

// TestResolveRegistry_PackBindingResolutionIsDeterministic proves a pack engines:
// block redeclaring a built-in name resolves to a WELL-DEFINED binding (the decided
// merge rule: pack-declared OVERRIDES the same-named built-in), not map-iteration
// order (SPEC-035 CLM-004 / Sharp Edge 9). The redeclared command differs from the
// DefaultRegistry "semgrep --sarif --quiet" so the winner is observable.
func TestResolveRegistry_PackBindingResolutionIsDeterministic(t *testing.T) {
	const redeclared = "acme-vendored-semgrep --sarif --quiet --extra"
	m := &pack.Manifest{
		NormalizedName: "acme/redeclares",
		Engines: map[string]pack.EngineSpec{
			"semgrep": {Binding: engine.EngineBinding{
				Command:   redeclared,
				InputMode: engine.InputModeRuleFlags,
				InputFlag: "--config",
				Provision: &engine.Provision{Tool: "semgrep", Version: "1.156.0"},
				Category:  engine.EngineCategoryOpinion,
			}},
		},
	}

	builtin := baseengines.Registry()["semgrep"].Command

	// Resolve repeatedly: the result must be stable AND the pack-declared command,
	// never the built-in (proving a decided override, not iteration order).
	for i := 0; i < 8; i++ {
		got, err := resolveEngineRegistry(m).Lookup("semgrep")
		if err != nil {
			t.Fatalf("redeclared engine must resolve: %v", err)
		}
		if got.Command == builtin {
			t.Fatalf("pack-declared binding must OVERRIDE the same-named built-in (got the built-in %q)", builtin)
		}
		if got.Command != redeclared {
			t.Fatalf("merge resolution is not deterministic: got %q, want %q", got.Command, redeclared)
		}
	}
}

// TestDispatch_PackDeclaredEngineRunsViaGenericDispatch proves a pack-declared
// engine dispatches through the EXISTING runFindingsEngine/gatherEngineInputs with
// no dispatcher change — the binding fields drive it (SPEC-035 CLM-003). The
// engine's tool is allowlisted+lock-pinned so the trust gate passes and the command
// reaches the recording runner.
func TestDispatch_PackDeclaredEngineRunsViaGenericDispatch(t *testing.T) {
	withTestAllowlist(t)
	packDir := t.TempDir()
	m := packDeclaredFindingsManifest(t, packDir, "acme-scan --sarif", "acme-scan", "2.3.1")

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}
	if _, err := dispatchPackEngines([]*pack.Manifest{m}, packDir, t.TempDir(), nil, rec); err != nil {
		t.Fatalf("dispatchPackEngines for a pack-declared engine: %v", err)
	}
	if rec.lastName != "acme-scan" {
		t.Errorf("the pack-declared engine's command must run via the generic dispatch, got runner name %q", rec.lastName)
	}
	// The rule's --config rule file must have been gathered by the unchanged
	// gatherEngineInputs (rule-flags mode).
	sawConfig := false
	for i := 0; i+1 < len(rec.lastArgs); i++ {
		if rec.lastArgs[i] == "--config" && strings.HasSuffix(rec.lastArgs[i+1], "rules/r.yml") {
			sawConfig = true
		}
	}
	if !sawConfig {
		t.Errorf("generic gatherEngineInputs must feed the pack-declared engine its rule file, got args %v", rec.lastArgs)
	}
}

func fixtureProjectRoot(t *testing.T, name string) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", name)
	copyDir(t, src, dst)
	return dst
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(filepath.Dir(wd))
}

func copyFixtureFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", dst, err)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		t.Fatalf("copy fixture %s -> %s: %v", src, dst, err)
	}
}
