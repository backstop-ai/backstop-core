package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/recipe"
)

// ISSUE-079's LIVE REPRO, driven through the shipped command.
//
// Every op-level test in pkg/recipe supplies its own ApplyOptions. Only a run of
// the SHIPPED `backstop recipe apply` over an INSTALLED pack reproduces what was
// observed on 2026-07-25: a create op whose declared target was templated against
// a DEFAULTED param wrote `./{{ config_dir }}/service.json` on disk and the
// command exited 0. A silent wrong-output defect looks identical to a correct run
// from the exit code alone, which is why this test walks the filesystem.
const recipeE2ETemplatedID = "templated"

// declaredParamDefaults reads the recipe's DECLARED param defaults off the parsed
// manifest — the scope direct mode substitutes with when the caller supplies
// nothing. Building the expectation from this rather than from a hardcoded path
// is what keeps the assertion from agreeing with a broken applier by coincidence.
func declaredParamDefaults(manifest *recipe.RecipeManifest) map[string]string {
	defaults := make(map[string]string, len(manifest.Params))
	for _, spec := range manifest.Params {
		if !spec.Required || spec.Default != "" {
			defaults[spec.Name] = spec.Default
		}
	}

	return defaults
}

// TestRecipeApply_CLI_TemplatedTargetWritesAtTheSubstitutedPath proves the shipped
// command resolves a templated declared target before writing (CLM-012), and that
// nothing carrying a literal placeholder is left anywhere beneath the project
// root (CLM-008).
//
// No `--param` flag is used or needed: the repro ran on the recipe's own declared
// DEFAULT, and the CLI param-input surface is ISSUE-081's.
func TestRecipeApply_CLI_TemplatedTargetWritesAtTheSubstitutedPath(t *testing.T) {
	projectRoot := stageRecipeE2EProject(t)
	ref, manifest, recipeDir := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2ETemplatedID)

	createOp := manifest.Ops[0]
	params := declaredParamDefaults(manifest)
	declaredTarget := createOp.Target
	if !strings.Contains(declaredTarget, "{{") {
		t.Fatalf("fixture defect: the declared target %q carries no placeholder, so the live repro is not reproduced here", declaredTarget)
	}
	substitutedTarget, err := recipe.Substitute(declaredTarget, params)
	if err != nil {
		t.Fatalf("render the declared target %q against the recipe's declared defaults: %v", declaredTarget, err)
	}

	rawPayload, err := os.ReadFile(filepath.Join(recipeDir, filepath.FromSlash(createOp.Payload)))
	if err != nil {
		t.Fatalf("read the declared payload %q: %v", createOp.Payload, err)
	}
	wantContent, err := recipe.Substitute(string(rawPayload), params)
	if err != nil {
		t.Fatalf("render the declared payload against the recipe's declared defaults: %v", err)
	}

	output, err := runRecipeApplyCLI(t, projectRoot, ref)
	if err != nil {
		t.Fatalf("backstop recipe apply %s failed: %v\noutput:\n%s", ref, err, output)
	}

	got, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(substitutedTarget)))
	if err != nil {
		t.Fatalf("the file did not land at the SUBSTITUTED target %q: %v\noutput:\n%s", substitutedTarget, err, output)
	}
	if string(got) != wantContent {
		t.Errorf("materialized content =\n%q\nwant the substituted payload\n%q", string(got), wantContent)
	}

	// The literal live-repro symptom: `./{{ config_dir }}/service.json` on disk
	// beside an exit-0 run. Walk the whole staged root rather than probing the one
	// path the fix is expected to produce.
	walkErr := filepath.WalkDir(projectRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			return relErr
		}
		for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
			if strings.Contains(segment, "{{") || strings.Contains(segment, "}}") {
				t.Errorf("the staged project holds %q, a path segment carrying a literal placeholder", rel)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk the staged project root %q: %v", projectRoot, walkErr)
	}

	adoptions, err := recipe.ReadAdoptions(filepath.Join(projectRoot, recipe.AdoptionRecordName))
	if err != nil {
		t.Fatalf("read adoption record: %v", err)
	}
	entry, adopted := adoptions.Recipes[recipeE2EPassPack+":"+recipeE2ETemplatedID]
	if !adopted {
		t.Fatalf("adoption record carries no entry for %s:%s (entries: %v)", recipeE2EPassPack, recipeE2ETemplatedID, adoptions.Recipes)
	}
	if entry.Version != manifest.Version {
		t.Errorf("adoption entry version = %q, want the applied pin %q", entry.Version, manifest.Version)
	}
}
