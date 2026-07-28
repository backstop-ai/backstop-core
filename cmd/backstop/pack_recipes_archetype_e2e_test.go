package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// The two committed acceptance fixtures for ISSUE-085's `recipes` archetype.
const (
	recipesFieldGuideFixture  = "recipes-archetype-pack"
	recipesToothlessFixture   = "recipes-archetype-pack-no-enforcement"
	recipesFieldGuidePackName = "demo-org/recipes-field-guide"
)

// TestPackCheck_RecipesArchetypeFieldGuidePackPasses (CLM-011) runs the REAL
// `pack check` over the committed field-guide fixture: a pack that hands out
// recipes and a paired scaffold with no rule set, no engine binding and none of
// the rule-pairing padding ISSUE-085's reproduction documents.
//
// The second half is what makes this an acceptance rather than a smoke test: the
// committed manifest must not contain the padding keys. Without it the fixture
// could be quietly padded back into the shape the issue is about while this test
// stayed green. The assertion is coherent only because the load seam's scaffold
// requirements became archetype-conditional — read it together with
// TestParseManifestFile_RecipesArchetypePackLoads, not separately.
func TestPackCheck_RecipesArchetypeFieldGuidePackPasses(t *testing.T) {
	fixture := filepath.Join("testdata", recipesFieldGuideFixture)

	out, err := executeCommand(NewRootCommand(), "pack", "check", "--format", "json", fixture)
	if err != nil {
		t.Fatalf("the field-guide pack must pass `pack check`, got error: %v (out: %s)", err, out)
	}

	var result packval.Result
	if unmarshalErr := json.Unmarshal([]byte(out), &result); unmarshalErr != nil {
		t.Fatalf("parsing pack check json: %v (out: %s)", unmarshalErr, out)
	}
	if result.Status != "pass" {
		t.Fatalf("want overall pass, got %s: %+v", result.Status, result.Errors)
	}
	sawArchetypePhase := false
	for _, phase := range result.Phases {
		if phase.Phase != "phase4-archetype" {
			continue
		}
		sawArchetypePhase = true
		if phase.Status != "pass" {
			t.Errorf("phase4-archetype must pass, got %s: %+v", phase.Status, phase.Errors)
		}
	}
	if !sawArchetypePhase {
		t.Error("pack check must actually run phase4-archetype over the fixture")
	}

	manifest, readErr := os.ReadFile(filepath.Join(fixture, "pack.yml"))
	if readErr != nil {
		t.Fatalf("reading the committed fixture manifest: %v", readErr)
	}
	for _, padding := range []string{"ruleset", "engines:", "pairs_with", "test_command", "use_when", "assumes"} {
		if strings.Contains(string(manifest), padding) {
			t.Errorf("the field-guide fixture must stay unpadded, but its pack.yml declares %q — padding it makes this acceptance vacuous", padding)
		}
	}
}

// TestPackCheck_RecipesArchetypeMissingEnforcementFails (CLM-012) runs the same
// real command over the committed toothless fixture, in the DEFAULT TEXT format
// rather than json.
//
// Text specifically, because text is the human default and the renderer prints
// Phase/Check/Message while DISCARDING the ValidationError's Rule field. A
// json-only assertion would pass while the operator standing at the terminal
// still could not tell WHICH recipe failed — so a recipe id appearing in this
// output can only have come from the message text.
func TestPackCheck_RecipesArchetypeMissingEnforcementFails(t *testing.T) {
	out, err := executeCommand(NewRootCommand(), "pack", "check", filepath.Join("testdata", recipesToothlessFixture))
	if err == nil {
		t.Fatalf("a non-templating recipe with no enforcement.rules must fail `pack check` non-zero; got exit 0 (out: %s)", out)
	}
	if !strings.Contains(out, "recipe-enforcement") {
		t.Errorf("the report must name the recipe-enforcement check, got: %q", out)
	}
	if !strings.Contains(out, "scaffold-service") {
		t.Errorf("the printed report must NAME the offending recipe id (the text renderer discards Rule), got: %q", out)
	}
}

// TestPackAdd_RecipesArchetypePackInstalls (CLM-011, CLM-004) closes the
// uninstallable headline through the real `pack add`, which runs the packval
// pipeline in BOTH check and test modes unconditionally.
//
// It then drives the INSTALLED manifest through pack.ParseManifestFile — the
// gate-time load seam nothing in pkg/pack/distribution ever calls. Without that
// closing assertion a pack could install cleanly and still abort every later
// `backstop gate` at load, which is strictly worse than an honest install-time
// rejection. Driving both seams in one test is what makes "installable" mean
// "usable".
func TestPackAdd_RecipesArchetypePackInstalls(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Copy the fixture out of the repo tree so the install never writes into it.
	// The copy happens BEFORE the chdir, while the relative testdata path resolves.
	srcDir := filepath.Join(parent, "recipes-field-guide-src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyTree(t, filepath.Join("testdata", recipesFieldGuideFixture), srcDir)

	// backstop.yml must exist BEFORE the command runs: the config update reads it
	// and returns the error, so `pack add` hard-fails in a directory without one.
	writeFileForTest(t, projectDir, "backstop.yml", "packs:\n  "+recipesFieldGuidePackName+": local")

	restore := chdirForTest(t, projectDir)
	defer restore()

	out, err := executeCommand(NewRootCommand(), "pack", "add", "../recipes-field-guide-src")
	if err != nil {
		t.Fatalf("the field-guide pack must install through the real `pack add`, got error: %v (out: %s)", err, out)
	}

	dest := filepath.Join(projectDir, ".backstop", "packs", recipesFieldGuidePackName)
	if _, statErr := os.Stat(filepath.Join(dest, "pack.yml")); statErr != nil {
		t.Fatalf("pack.yml not materialized on disk: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "recipes", "scaffold-service", "recipe.yml")); statErr != nil {
		t.Errorf("the recipes tree must materialize with the pack: %v", statErr)
	}

	lockfile, lockErr := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if lockErr != nil {
		t.Fatalf("reading lock: %v", lockErr)
	}
	if _, ok := lockfile.Packs[recipesFieldGuidePackName]; !ok {
		t.Errorf("lock entry not written for %q", recipesFieldGuidePackName)
	}

	// The pack that INSTALLS is the pack that must LOAD.
	installed, parseErr := pack.ParseManifestFile(filepath.Join(dest, "pack.yml"))
	if parseErr != nil {
		t.Fatalf("the installed pack must parse at the gate-time load seam, got: %v", parseErr)
	}
	if installed.Archetype != "recipes" {
		t.Errorf("want archetype recipes at the load seam, got %q", installed.Archetype)
	}
}
