package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/recipe"
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
