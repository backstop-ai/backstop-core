package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/recipe"
	"gopkg.in/yaml.v3"
)

// SPEC-054 CLM-063 — THE anti-stub-green guard for the transform seam.
//
// Every op-level test in phases 5-9 supplies its OWN TransformDispatch, so a CLI
// that wires a dispatch which does nothing still passes all of them. Only a run of
// the SHIPPED `backstop recipe apply` command, driving a REAL allowlisted engine
// over CAPTURED bytes, can falsify that — which is what this file does.
//
// Nothing language- or tool-shaped is typed here: the pack name and recipe id below
// are the committed fixture's own coordinates, and every extension, target, rule
// path and tool name is read from the fixture's pack.yml / recipe.yml.
const (
	recipeE2EProject  = "recipe-apply-e2e"
	recipeE2EPassPack = "demo-org/recipe-pack"
	recipeE2EPassID   = "rewrite"
)

// The two CAPTURED fixture stages. They name a directory and two files inside the
// fixture pack — backstop's own capture convention — and carry no extension: the
// extension comes from the recipe's DECLARED target.
const (
	recipeFixturesDir = "fixtures"
	recipeStageBefore = "before"
	recipeStageAfter  = "after"
)

// stageRecipeE2EProject copies the committed fixture consumer project — its
// installed packs under .backstop/packs included — into a fresh temp root, so a
// run mutates the copy and never the tracked fixture.
func stageRecipeE2EProject(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("testdata", recipeE2EProject))
	if err != nil {
		t.Fatalf("resolve fixture project %q: %v", recipeE2EProject, err)
	}
	dst := t.TempDir()
	copyTree(t, src, dst)
	return dst
}

// stagedRecipe parses the INSTALLED pack manifest and its colocated recipe manifest
// out of the staged project and returns the pinned ref to drive the CLI with, the
// parsed recipe manifest, and the recipe's directory. The version pin, the declared
// target and the rule path all come from this parsed DATA rather than being retyped
// in a test literal.
func stagedRecipe(t *testing.T, projectRoot string, packName string, recipeID string) (string, *recipe.RecipeManifest, string) {
	t.Helper()

	packRoot := filepath.Join(projectRoot, ".backstop", "packs", filepath.FromSlash(packName))
	manifest, err := pack.ParseManifestFile(filepath.Join(packRoot, "pack.yml"))
	if err != nil {
		t.Fatalf("parse installed pack %q: %v", packName, err)
	}

	indexed, declared := manifest.Recipes[recipeID]
	if !declared {
		t.Fatalf("pack %q indexes no recipe %q (indexed: %v)", packName, recipeID, manifest.Recipes)
	}

	recipeDir := filepath.Join(packRoot, filepath.FromSlash(indexed))
	data, err := os.ReadFile(filepath.Join(recipeDir, "recipe.yml"))
	if err != nil {
		t.Fatalf("read recipe manifest for %q: %v", recipeID, err)
	}
	parsed, err := recipe.ParseRecipeManifest(data)
	if err != nil {
		t.Fatalf("parse recipe manifest for %q: %v", recipeID, err)
	}
	if len(parsed.Ops) == 0 {
		t.Fatalf("recipe %q declares no ops; the fixture cannot drive a dispatch", recipeID)
	}

	ref := recipe.RecipeRef{Pack: packName, Recipe: recipeID, Version: parsed.Version}
	return ref.String(), parsed, recipeDir
}

// capturedFixture reads one CAPTURED stage of the fixture pair. The file's
// extension is taken from the recipe's DECLARED target so the parser-selecting
// suffix is fixture data, never a literal in this file.
func capturedFixture(t *testing.T, recipeDir string, declaredTarget string, stage string) []byte {
	t.Helper()
	path := filepath.Join(recipeDir, recipeFixturesDir, stage+filepath.Ext(declaredTarget))
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read captured %s fixture %q: %v", stage, path, err)
	}
	return body
}

// stageDeclaredTarget writes body at the recipe's DECLARED target inside the staged
// project and returns the absolute path, so an assertion reads the same file the
// applier resolved.
func stageDeclaredTarget(t *testing.T, projectRoot string, declaredTarget string, body []byte) string {
	t.Helper()
	path := filepath.Join(projectRoot, filepath.FromSlash(declaredTarget))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create declared target directory for %q: %v", declaredTarget, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("stage declared target %q: %v", declaredTarget, err)
	}
	return path
}

// runRecipeApplyCLI drives the ASSEMBLED root command — the shipped path, not an
// internal helper — from inside projectRoot, and returns the command's error plus
// whatever it printed.
func runRecipeApplyCLI(t *testing.T, projectRoot string, ref string) (string, error) {
	t.Helper()
	t.Chdir(projectRoot)

	out := new(bytes.Buffer)
	root := NewRootCommand()
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"recipe", "apply", ref})
	err := root.Execute()
	return out.String(), err
}

// TestRecipeApply_E2E_RealEngineTransform_CapturedFixture proves CLM-063: driven
// through the actual CLI, a REAL allowlisted engine runs the recipe's DECLARED
// rewrite rule against the CAPTURED before-fixture and the target's BYTES become
// the captured after-state. A wired-but-no-op TransformDispatch leaves the bytes at
// before and fails here — the whole point of the test.
//
// If the engine is unavailable this test FAILS; it must never skip. The engine is a
// TOOL the trusted-tool allowlist provisions at a pinned version, not an optional
// pack, so its absence is an environment defect and a skip would recreate exactly
// the vacuous green this claim exists to close.
func TestRecipeApply_E2E_RealEngineTransform_CapturedFixture(t *testing.T) {
	projectRoot := stageRecipeE2EProject(t)
	ref, manifest, recipeDir := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2EPassID)

	declaredTarget := manifest.Ops[0].Target
	before := capturedFixture(t, recipeDir, declaredTarget, recipeStageBefore)
	after := capturedFixture(t, recipeDir, declaredTarget, recipeStageAfter)
	if bytes.Equal(before, after) {
		t.Fatalf("the captured fixture pair is not falsifying: before == after, so a no-op dispatch would pass this test")
	}

	targetPath := stageDeclaredTarget(t, projectRoot, declaredTarget, before)

	output, err := runRecipeApplyCLI(t, projectRoot, ref)
	if err != nil {
		t.Fatalf("backstop recipe apply %s failed: %v\noutput:\n%s", ref, err, output)
	}

	got, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read transformed target %q: %v", targetPath, readErr)
	}
	if bytes.Equal(got, before) {
		t.Fatalf("the target is byte-identical to the captured before-state: the transform dispatch ran as a no-op\noutput:\n%s", output)
	}
	if !bytes.Equal(got, after) {
		t.Errorf("transformed target bytes =\n%s\nwant the captured after-state =\n%s", got, after)
	}

	adoptions, adoptErr := recipe.ReadAdoptions(filepath.Join(projectRoot, recipe.AdoptionRecordName))
	if adoptErr != nil {
		t.Fatalf("read adoption record: %v", adoptErr)
	}
	entry, adopted := adoptions.Recipes[recipeE2EPassPack+":"+recipeE2EPassID]
	if !adopted {
		t.Fatalf("adoption record carries no entry for %s:%s (entries: %v)", recipeE2EPassPack, recipeE2EPassID, adoptions.Recipes)
	}
	if entry.Version != manifest.Version {
		t.Errorf("adoption entry version = %q, want the applied pin %q", entry.Version, manifest.Version)
	}
	if entry.Adopted == "" {
		t.Errorf("adoption entry carries no adopted timestamp: %+v", entry)
	}
}

// The wiring's FAILURE modes, driven through the same shipped command as the happy
// path. They matter for the same reason CLM-063 does: a CLI that silently swallows an
// unusable reference, an uninstalled pack, or an engine that refused to run would look
// identical to one that worked, and every op-level test in pkg/recipe would still pass.

// mutateFixtureYAML rewrites one staged YAML document in place. The staged project is a
// throwaway copy, so a variant the committed fixture does not ship (an engines: block
// with none or several provisioned bindings, an op of a different kind) is DERIVED from
// the fixture's own data instead of being retyped here — which is what keeps the tool
// names, versions and paths in these tests fixture-sourced.
func mutateFixtureYAML(t *testing.T, path string, mutate func(doc map[string]any)) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q for mutation: %v", path, err)
	}
	doc := map[string]any{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode %q for mutation: %v", path, err)
	}

	mutate(doc)

	rewritten, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("encode mutated %q: %v", path, err)
	}
	if err := os.WriteFile(path, rewritten, 0o644); err != nil {
		t.Fatalf("write mutated %q: %v", path, err)
	}
}

// stagedPackManifestPath locates one staged pack's manifest.
func stagedPackManifestPath(projectRoot string, packName string) string {
	return filepath.Join(projectRoot, ".backstop", "packs", filepath.FromSlash(packName), "pack.yml")
}

// requireConfigError asserts the CLI refused BEFORE applying anything: a
// *check.ConfigError (exit 2) whose message names each of want.
func requireConfigError(t *testing.T, err error, output string, want ...string) {
	t.Helper()

	if err == nil {
		t.Fatalf("the command succeeded; it must refuse\noutput:\n%s", output)
	}
	var configErr *check.ConfigError
	if !errors.As(err, &configErr) {
		t.Fatalf("error is %T (%v), want a *check.ConfigError — nothing was applied, so this is exit-2 shaped", err, err)
	}
	for _, fragment := range want {
		if !strings.Contains(configErr.Message, fragment) {
			t.Errorf("refusal message %q does not name %q", configErr.Message, fragment)
		}
	}
}

// TestRecipeApply_CLI_RefusesBeforeApplying covers the four ways a run can fail before
// a single op executes. Each is exit-2 shaped and names what the operator must change,
// because in every one of them NOTHING was applied — reporting them as violations
// (exit 1) would tell an agent that the project is dirty when the invocation is.
func TestRecipeApply_CLI_RefusesBeforeApplying(t *testing.T) {
	t.Run("no project config is discoverable", func(t *testing.T) {
		// A bare directory with no backstop.yml above it: the applier writes beneath the
		// project the config declares, so with no config there is no root to write into.
		bare := t.TempDir()
		staged := stageRecipeE2EProject(t)
		ref, _, _ := stagedRecipe(t, staged, recipeE2EPassPack, recipeE2EPassID)

		output, err := runRecipeApplyCLI(t, bare, ref)
		requireConfigError(t, err, output, "config")
	})

	t.Run("the reference is unpinned", func(t *testing.T) {
		projectRoot := stageRecipeE2EProject(t)
		unpinned := recipeE2EPassPack + ":" + recipeE2EPassID

		output, err := runRecipeApplyCLI(t, projectRoot, unpinned)
		requireConfigError(t, err, output, unpinned, "unpinned")
	})

	t.Run("a declared pack is not installed", func(t *testing.T) {
		projectRoot := stageRecipeE2EProject(t)
		ref, _, _ := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2EPassID)

		// backstop.yml still DECLARES the pack; only the installed tree is gone. That is
		// the everyday state of a fresh clone (.backstop/packs is gitignored), and it must
		// fail loud naming the pack rather than resolving against a partial corpus.
		uninstalled := filepath.Join(projectRoot, ".backstop", "packs", filepath.FromSlash(recipeE2EMismatchPack))
		if err := os.RemoveAll(uninstalled); err != nil {
			t.Fatalf("remove installed pack: %v", err)
		}

		output, err := runRecipeApplyCLI(t, projectRoot, ref)
		requireConfigError(t, err, output, recipeE2EMismatchPack)
	})

	t.Run("the pack indexes no such recipe", func(t *testing.T) {
		projectRoot := stageRecipeE2EProject(t)
		_, manifest, _ := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2EPassID)

		absent := recipe.RecipeRef{
			Pack:    recipeE2EPassPack,
			Recipe:  recipeE2EPassID + "-absent",
			Version: manifest.Version,
		}.String()

		output, err := runRecipeApplyCLI(t, projectRoot, absent)
		requireConfigError(t, err, output, recipeE2EPassPack, "declares no recipe")
	})
}

// TestRecipeApply_CLI_UnreachableTargetRelaysDeclaredManualVerbatim proves the
// fail-loud half of REQ-011 survives the CLI: when the declared site cannot be reached
// — here the target does not exist — the run fails as a VIOLATION (exit 1, not exit 2:
// the invocation was fine, the apply is what failed) and the op's declared `manual:`
// text is relayed WORD FOR WORD, never paraphrased. The applier cannot compose that
// instruction, so a CLI that dropped it would leave the operator with nothing.
func TestRecipeApply_CLI_UnreachableTargetRelaysDeclaredManualVerbatim(t *testing.T) {
	projectRoot := stageRecipeE2EProject(t)
	ref, manifest, _ := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2EPassID)

	// The target is deliberately NOT staged — nothing is written at the declared path.
	declaredManual := strings.TrimSpace(manifest.Ops[0].Manual)
	if declaredManual == "" {
		t.Fatalf("fixture defect: the op declares no manual text, so this test cannot prove it is relayed")
	}

	output, err := runRecipeApplyCLI(t, projectRoot, ref)
	if err == nil {
		t.Fatalf("the command succeeded with no target to rewrite\noutput:\n%s", output)
	}

	var configErr *check.ConfigError
	if errors.As(err, &configErr) {
		t.Fatalf("error is a *check.ConfigError (%v); an apply that got as far as running ops is a violation, not a config refusal", configErr)
	}
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error is %T (%v), want an *ExitCodeError", err, err)
	}
	if exitErr.Code != ExitViolations {
		t.Errorf("exit code = %d, want %d (violations)", exitErr.Code, ExitViolations)
	}
	if !strings.Contains(exitErr.Message, declaredManual) {
		t.Errorf("failure message does not relay the declared manual text VERBATIM.\ngot:\n%s\nwant it to contain:\n%s", exitErr.Message, declaredManual)
	}
}

// TestRecipeApply_CLI_SurfacesEngineOutputOnEngineFailure proves the dispatch reports
// what the ENGINE said. The rule file is corrupted so the real engine refuses it; the
// failure must carry the engine's own diagnostic, which is only possible because the
// transform captures COMBINED output — a stdout-only capture reduces this to a bare
// exit status and leaves the operator guessing. The target must also survive untouched.
func TestRecipeApply_CLI_SurfacesEngineOutputOnEngineFailure(t *testing.T) {
	projectRoot := stageRecipeE2EProject(t)
	ref, manifest, recipeDir := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2EPassID)

	declaredTarget := manifest.Ops[0].Target
	before := capturedFixture(t, recipeDir, declaredTarget, recipeStageBefore)
	targetPath := stageDeclaredTarget(t, projectRoot, declaredTarget, before)

	// The rule is declared PACK-relative, so it resolves under the pack root — the
	// same base the applier resolves it against.
	packRoot := filepath.Dir(stagedPackManifestPath(projectRoot, recipeE2EPassPack))
	rulePath := filepath.Join(packRoot, filepath.FromSlash(manifest.Ops[0].Rule))
	if _, err := os.Stat(rulePath); err != nil {
		t.Fatalf("the declared rule is not where the pack declares it (%q): %v", rulePath, err)
	}
	if err := os.WriteFile(rulePath, []byte("{{ not a rule document"), 0o644); err != nil {
		t.Fatalf("corrupt the declared rule: %v", err)
	}

	output, err := runRecipeApplyCLI(t, projectRoot, ref)
	if err == nil {
		t.Fatalf("the command succeeded with an unreadable rule\noutput:\n%s", output)
	}
	if !strings.Contains(err.Error(), filepath.Base(rulePath)) {
		t.Errorf("the failure does not carry the engine's own diagnostic naming the rule it rejected; got:\n%v", err)
	}

	after, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read declared target: %v", readErr)
	}
	if !bytes.Equal(after, before) {
		t.Errorf("the target was modified by a run whose engine failed:\n%s", after)
	}
}

// TestRecipeApply_CLI_RunThatWritesNothingAdoptsNothing proves the adoption record is
// not written for a run that materialized nothing. The staged recipe's single op is
// re-declared as `step` — the family that is recognized and sequenced but never
// executed — so the run SUCCEEDS while writing no file. Recording an adoption anyway
// would tell the NEXT apply that a file the consumer owns is recipe-owned and safe to
// regenerate over.
func TestRecipeApply_CLI_RunThatWritesNothingAdoptsNothing(t *testing.T) {
	projectRoot := stageRecipeE2EProject(t)
	ref, manifest, recipeDir := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2EPassID)

	declaredTarget := manifest.Ops[0].Target
	before := capturedFixture(t, recipeDir, declaredTarget, recipeStageBefore)
	targetPath := stageDeclaredTarget(t, projectRoot, declaredTarget, before)

	mutateFixtureYAML(t, filepath.Join(recipeDir, "recipe.yml"), func(doc map[string]any) {
		ops, ok := doc["ops"].([]any)
		if !ok || len(ops) == 0 {
			t.Fatalf("staged recipe declares no ops to re-kind: %#v", doc["ops"])
		}
		op, ok := ops[0].(map[string]any)
		if !ok {
			t.Fatalf("staged op is not a mapping: %#v", ops[0])
		}
		op["kind"] = string(recipe.OpStep)
	})

	output, err := runRecipeApplyCLI(t, projectRoot, ref)
	if err != nil {
		t.Fatalf("a recipe whose only op is reserved-but-unexecuted must succeed: %v\noutput:\n%s", err, output)
	}

	if _, statErr := os.Stat(filepath.Join(projectRoot, recipe.AdoptionRecordName)); statErr == nil {
		t.Errorf("a run that wrote nothing recorded an adoption")
	}
	got, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read declared target: %v", readErr)
	}
	if !bytes.Equal(got, before) {
		t.Errorf("a reserved step op rewrote the target:\n%s", got)
	}
}
