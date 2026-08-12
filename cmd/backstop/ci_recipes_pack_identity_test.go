package main

// SPEC-067 REQ-001 / REQ-002 / REQ-009 — pack identity, fleet declaration,
// enforcement-rule resolution, and the STRUCTURAL pack-test verdict.
//
// Every test here reads the REAL installed pack. Its absence fails LOUDLY.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// TestCIRecipes_PackInstalledAndParseableAtExpectedPath proves CLM-001: the pack
// is present and parseable at `.backstop/packs/backstop-ai/ci-workflows/pack.yml`,
// declares that manifest name and `archetype: recipes` — and its ABSENCE fails
// the suite rather than skipping it.
func TestCIRecipes_PackInstalledAndParseableAtExpectedPath(t *testing.T) {
	root := ciPackRoot(t)
	if filepath.ToSlash(root) == "" {
		t.Fatalf("the resolved pack root is empty")
	}

	manifest := ciParsedPack(t)
	if manifest.Name != ciPackName {
		t.Errorf("installed pack declares name %q, want %q — the manifest name IS the install identity", manifest.Name, ciPackName)
	}
	if manifest.Archetype != "recipes" {
		t.Errorf("installed pack declares archetype %q, want %q", manifest.Archetype, "recipes")
	}
	if manifest.Version == "" {
		t.Errorf("installed pack declares no version")
	}
}

// TestCIRecipes_RecipeIndexIsExactlyTheFourPlatforms proves CLM-002: the
// `recipes:` key set is EXACTLY the four declared ids, each mapped to a directory
// holding a readable recipe.yml. Compared as SORTED SETS, so a fifth id fails.
func TestCIRecipes_RecipeIndexIsExactlyTheFourPlatforms(t *testing.T) {
	manifest := ciParsedPack(t)

	indexed := []string{}
	for id := range manifest.Recipes {
		indexed = append(indexed, id)
	}
	sort.Strings(indexed)

	want := ciAllRecipeIDs()
	sort.Strings(want)

	if strings.Join(indexed, ",") != strings.Join(want, ",") {
		t.Fatalf("the recipes index is %v, want exactly %v", indexed, want)
	}

	for _, id := range want {
		path := filepath.Join(ciRecipeDir(t, id), "recipe.yml")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("recipe %q's indexed directory holds no readable recipe.yml at %s: %v", id, path, err)
		}
	}
}

// TestCIRecipes_PackShipsNoOIDCOrProviderRecipe proves CLM-003 (absence): no
// OIDC / ingest / emitter recipe and no supabase / vercel / nextjs provider
// recipe. Those surfaces belong to later specs and their absence here is
// deliberate, not an oversight.
func TestCIRecipes_PackShipsNoOIDCOrProviderRecipe(t *testing.T) {
	denied := []string{"oidc", "ingest", "emitter", "supabase", "vercel", "nextjs"}

	manifest := ciParsedPack(t)
	for id, dir := range manifest.Recipes {
		for _, banned := range denied {
			if strings.Contains(strings.ToLower(id), banned) {
				t.Errorf("the pack indexes recipe %q, whose id names the out-of-scope surface %q", id, banned)
			}
			if strings.Contains(strings.ToLower(dir), banned) {
				t.Errorf("recipe %q maps to directory %q, which names the out-of-scope surface %q", id, dir, banned)
			}
		}
	}

	entries, err := os.ReadDir(filepath.Join(ciPackRoot(t), "recipes"))
	if err != nil {
		t.Fatalf("read the pack's recipes directory: %v", err)
	}
	for _, entry := range entries {
		for _, banned := range denied {
			if strings.Contains(strings.ToLower(entry.Name()), banned) {
				t.Errorf("the recipes directory holds %q, which names the out-of-scope surface %q", entry.Name(), banned)
			}
		}
	}
}

// TestCIRecipes_SingleProvisionedEngineBindingUnderDistinctName proves CLM-004:
// exactly ONE binding carries `provision:`, its key is `semgrep-ci`, there is no
// binding keyed `semgrep` (which would shadow the base-engines binding every
// OTHER installed pack resolves through), and its pin EQUALS the allowlist pin
// read from the package rather than retyped here.
func TestCIRecipes_SingleProvisionedEngineBindingUnderDistinctName(t *testing.T) {
	manifest := ciParsedPack(t)

	provisioned := []string{}
	for name, spec := range manifest.Engines {
		if spec.Binding.Provision != nil {
			provisioned = append(provisioned, name)
		}
	}
	sort.Strings(provisioned)

	if len(provisioned) != 1 {
		t.Fatalf("the pack declares %d provisioned engine bindings (%v), want exactly 1 — `recipe apply` calls provisionedEngineBinding unconditionally and refuses any other count", len(provisioned), provisioned)
	}
	const expectedBindingName = "semgrep-ci"
	if provisioned[0] != expectedBindingName {
		t.Errorf("the provisioned binding is keyed %q, want %q; a binding keyed `semgrep` would override the base-engines default for every other pack in a consumer project", provisioned[0], expectedBindingName)
	}
	if _, shadowing := manifest.Engines["semgrep"]; shadowing {
		t.Errorf("the pack declares a binding keyed `semgrep`, which shadows the embedded base-engines binding")
	}

	wantPin, allowlisted := engine.TrustedToolAllowlist()["semgrep"]
	if !allowlisted {
		t.Fatalf("the trusted-tool allowlist declares no semgrep pin; this claim has nothing to compare against")
	}
	binding := manifest.Engines[provisioned[0]].Binding
	if binding.Provision.Version != wantPin {
		t.Errorf("the provisioned binding pins semgrep %q, want the allowlist pin %q — the trust gate runs at BOTH parse and dispatch, so any other pin is an exit-2 config error", binding.Provision.Version, wantPin)
	}
	if binding.Provision.Tool != "semgrep" {
		t.Errorf("the provisioned binding provisions tool %q, want %q", binding.Provision.Tool, "semgrep")
	}
}

// TestCIRecipes_FleetDeclaresPackAtOneVersionInBothFiles proves CLM-005:
// backstop.yml and backstop.lock both declare the pack, at the SAME version
// string, so `backstop pack install` materializes the pack these tests resolve.
//
// It deliberately does NOT assert a PARTICULAR version — the pack may be revved
// without editing core, which is the property this whole spec is about.
func TestCIRecipes_FleetDeclaresPackAtOneVersionInBothFiles(t *testing.T) {
	repoRoot := ciRepoRoot(t)

	cfg, err := config.LoadConfigFromPath(filepath.Join(repoRoot, "backstop.yml"))
	if err != nil {
		t.Fatalf("load backstop.yml: %v", err)
	}
	declared, inConfig := cfg.Packs[ciPackName]
	if !inConfig {
		t.Fatalf("backstop.yml declares no %s entry (declares: %v)", ciPackName, cfg.Packs)
	}

	lockfile, lockErr := distribution.ReadLockfile(filepath.Join(repoRoot, "backstop.lock"))
	if lockErr != nil {
		t.Fatalf("read backstop.lock: %v", lockErr)
	}
	entry, inLock := lockfile.Packs[ciPackName]
	if !inLock {
		t.Fatalf("backstop.lock carries no %s entry", ciPackName)
	}

	if entry.Version != declared {
		t.Errorf("backstop.yml declares %s at %q while backstop.lock records %q; `pack install` would materialize a version the tests were not written against", ciPackName, declared, entry.Version)
	}
	if entry.Version != ciParsedPack(t).Version {
		t.Errorf("the fleet declares %s at %q but the INSTALLED pack.yml declares %q", ciPackName, entry.Version, ciParsedPack(t).Version)
	}
}

// TestCIRecipes_PackRuleIncludePathsMatchNoTrackedCoreFile proves CLM-006: every
// rule file declares a non-empty `paths: include:` set, and the UNION of those
// patterns matches ZERO files tracked in backstop-core.
//
// That is REQ-002's verdict-neutrality guarantee proven STRUCTURALLY rather than
// by re-running the gate and hoping. The match uses ciBasenameMatch, which models
// semgrep's slashless-include semantics (Sharp Edge 10); the only authority is
// semgrep, and the pack's scripts/falsify.sh is what runs it.
func TestCIRecipes_PackRuleIncludePathsMatchNoTrackedCoreFile(t *testing.T) {
	tracked := ciTrackedFiles(t)
	patterns := ciAllIncludePatterns(t)

	offenders := []string{}
	for pattern, ruleID := range patterns {
		for _, file := range tracked {
			if ciBasenameMatch(t, pattern, file) {
				offenders = append(offenders, pattern+" (rule "+ruleID+") matches tracked file "+file)
			}
		}
	}
	sort.Strings(offenders)

	if len(offenders) != 0 {
		t.Fatalf("the pack's include patterns match %d tracked backstop-core file(s); adopting the pack would move core's gate verdict:\n%s",
			len(offenders), strings.Join(offenders, "\n"))
	}

	// A cross-check on the check: core's one workflow must have been examined.
	sawCoreWorkflow := false
	for _, file := range tracked {
		if file == ".github/workflows/ci.yml" {
			sawCoreWorkflow = true
		}
	}
	if !sawCoreWorkflow {
		t.Errorf("`git ls-files` did not list .github/workflows/ci.yml, so the neutrality measurement did not cover core's only workflow")
	}
}

// TestCIRecipes_CoreAdoptsThePackWithoutApplyingAnyRecipe proves CLM-007
// (absence): core's own `.github/workflows/ci.yml` is byte-unchanged, it is not
// any recipe's declared target, and the repository root carries no
// `.gitlab-ci.yml`, `bitbucket-pipelines.yml` or `Jenkinsfile`. Core adopts the
// PACK, never a recipe.
func TestCIRecipes_CoreAdoptsThePackWithoutApplyingAnyRecipe(t *testing.T) {
	repoRoot := ciRepoRoot(t)
	const coreWorkflow = ".github/workflows/ci.yml"

	if ciGitDiffIsDirty(t, repoRoot, coreWorkflow) {
		t.Errorf("%s differs from its committed content; this spec edits no core workflow", coreWorkflow)
	}

	for _, id := range ciAllRecipeIDs() {
		manifest := ciRecipeManifest(t, id)
		for _, op := range manifest.Ops {
			if filepath.ToSlash(op.Target) == coreWorkflow {
				t.Errorf("recipe %q declares %s as its target; core's workflow is bespoke and is not a recipe target", id, coreWorkflow)
			}
		}
	}

	for _, unwanted := range []string{".gitlab-ci.yml", "bitbucket-pipelines.yml", "Jenkinsfile"} {
		if _, err := os.Stat(filepath.Join(repoRoot, unwanted)); err == nil {
			t.Errorf("the repository root carries %s; core applied a recipe to itself", unwanted)
		}
	}
}

// ciGitDiffIsDirty reports whether one tracked path differs from HEAD. It reads
// git rather than comparing against a copy, so nothing here can drift.
func ciGitDiffIsDirty(t *testing.T, repoRoot string, path string) bool {
	t.Helper()

	committed := ciGitShow(t, repoRoot, "HEAD:"+path)
	onDisk, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(committed) != string(onDisk)
}

// TestCIRecipes_EveryDeclaredEnforcementRuleResolvesInThePackRuleset proves
// CLM-057, in BOTH directions: every id named in any recipe's
// `enforcement.rules` exists as a declared rule, and the ruleset declares exactly
// the rules those four lists name.
//
// packval's archetype check is PRESENCE-ONLY — it never resolves those strings
// against anything — which is why this cross-reference is the spec's obligation
// and not the pipeline's.
func TestCIRecipes_EveryDeclaredEnforcementRuleResolvesInThePackRuleset(t *testing.T) {
	declared := map[string]bool{}
	for _, rule := range ciParsedPack(t).Content.Ruleset.Rules {
		if declared[rule.ID] {
			t.Errorf("the pack ruleset declares rule %q twice", rule.ID)
		}
		declared[rule.ID] = true
	}

	enforced := map[string]bool{}
	for _, id := range ciAllRecipeIDs() {
		enforcement := ciRecipeManifest(t, id).Enforcement
		if enforcement == nil || len(enforcement.Rules) == 0 {
			t.Fatalf("recipe %q declares no enforcement.rules", id)
		}
		for _, ruleID := range enforcement.Rules {
			enforced[ruleID] = true
			if !declared[ruleID] {
				t.Errorf("recipe %q enforces rule %q, which the pack ruleset does not declare", id, ruleID)
			}
		}
	}

	for ruleID := range declared {
		if !enforced[ruleID] {
			t.Errorf("the pack ruleset declares rule %q that no recipe's enforcement.rules names", ruleID)
		}
	}
	if len(declared) != len(enforced) {
		t.Errorf("the ruleset declares %d rules while the four recipes name %d; the two sets must be equal", len(declared), len(enforced))
	}
}

// TestCIRecipes_InstalledPackClearsRealPackTestStructurally proves CLM-058 by
// running the REAL `backstop pack test` pipeline over the installed pack.
//
// ★ THIS IS A STRUCTURAL VERDICT ONLY AND ASSERTS NOTHING ABOUT RULE FIRING. ★
// The manifest parses, it validates against the pack schema, and every declared
// recipe, rule and engine binding is well-formed. That is all.
//
// packval phase 3's fixture-EXECUTION step is guarded on a manifest field real
// packs do not declare, so it is a SILENT NO-OP for this pack and for every pack
// in the current fleet — backstop-core ISSUE-092, open and risk-classed critical.
// A green result here is NOT evidence that any of the twelve rules fires on a
// defective file. Anyone reading this test as firing evidence should be stopped
// by this comment: the firing proof lives in the pack's own scripts/falsify.sh,
// which drives semgrep directly.
func TestCIRecipes_InstalledPackClearsRealPackTestStructurally(t *testing.T) {
	packRoot := ciPackRoot(t)

	out, err := executeCommand(NewRootCommand(), "pack", "test", "--format", "json", packRoot)
	if err != nil {
		t.Fatalf("`pack test` over the installed pack failed: %v\noutput:\n%s", err, out)
	}

	result := packval.Result{}
	if unmarshalErr := json.Unmarshal([]byte(ciJSONBody(out)), &result); unmarshalErr != nil {
		t.Fatalf("decode the pack test result: %v\noutput:\n%s", unmarshalErr, out)
	}
	if result.Status != "pass" {
		t.Errorf("`pack test` status = %q, want %q\noutput:\n%s", result.Status, "pass", out)
	}
	for _, phase := range result.Phases {
		if len(phase.Errors) != 0 {
			t.Errorf("phase %s reported %d error(s): %+v", phase.Phase, len(phase.Errors), phase.Errors)
		}
	}
	if len(result.Phases) == 0 {
		t.Errorf("`pack test` reported no phases at all, so a green status proves nothing\noutput:\n%s", out)
	}
}

// ciJSONBody trims anything the CLI printed before the JSON document.
func ciJSONBody(out string) string {
	if index := strings.Index(out, "{"); index > 0 {
		return out[index:]
	}
	return out
}
