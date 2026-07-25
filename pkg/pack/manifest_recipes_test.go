package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	validRecipesPackYML      = "testdata/recipes/valid/pack.yml"
	missingDirRecipesPackYML = "testdata/recipes/missing-dir/pack.yml"
)

// scaffoldsOnlyPackYML declares content.scaffolds and NO recipes: index — the
// zero-value guard for CLM-033 (the index is optional).
const scaffoldsOnlyPackYML = `name: acme/scaffolds-only
version: 1.0.0
language: any
archetype: code
description: Pack declaring content.scaffolds with no recipes index at all
content:
  scaffolds:
    - id: service-base
      version: 1.0.0
      tier: skeleton
      path: scaffolds/service-base
      test_command: make test
      description: Service base scaffold, declared independently of the recipes index.
      use_when: [services]
      assumes: [project-manifest]
      pairs_with:
        scaffolds: [service-base-extras]
`

// TestPackManifest_RecipesIndexMapsIdToDir covers CLM-032: the pack.yml recipes
// index maps a stable recipe-id to its directory. Driven through
// ParseManifestFile, the production entry point that knows the pack root.
func TestPackManifest_RecipesIndexMapsIdToDir(t *testing.T) {
	m, err := ParseManifestFile(validRecipesPackYML)
	if err != nil {
		t.Fatalf("ParseManifestFile returned error: %v", err)
	}
	if len(m.Recipes) != 2 {
		t.Fatalf("expected 2 recipes index entries, got %d (%v)", len(m.Recipes), m.Recipes)
	}
	want := map[string]string{
		"alpha": "recipes/alpha",
		"beta":  "recipes/beta",
	}
	for id, dir := range want {
		got, ok := m.Recipes[id]
		if !ok {
			t.Fatalf("recipes index missing id %q (got %v)", id, m.Recipes)
		}
		if got != dir {
			t.Fatalf("recipe %q: expected dir %q, got %q", id, dir, got)
		}
	}
}

// TestPackManifest_RecipesIndexDistinctFromScaffolds covers CLM-033: recipes is
// a distinct top-level key from content.scaffolds. Both survive parsing, neither
// is populated from the other, and a manifest declaring only content.scaffolds
// leaves Recipes nil.
func TestPackManifest_RecipesIndexDistinctFromScaffolds(t *testing.T) {
	m, err := ParseManifestFile(validRecipesPackYML)
	if err != nil {
		t.Fatalf("ParseManifestFile returned error: %v", err)
	}

	if len(m.Content.Scaffolds) != 1 {
		t.Fatalf("expected 1 scaffold, got %d", len(m.Content.Scaffolds))
	}
	if m.Content.Scaffolds[0].ID != "service-base" {
		t.Fatalf("unexpected scaffold id: %q", m.Content.Scaffolds[0].ID)
	}
	if m.Content.Scaffolds[0].Path != "scaffolds/service-base" {
		t.Fatalf("unexpected scaffold path: %q", m.Content.Scaffolds[0].Path)
	}

	if len(m.Recipes) != 2 {
		t.Fatalf("expected 2 recipes index entries, got %d (%v)", len(m.Recipes), m.Recipes)
	}
	// The scaffold must not leak into the recipes index, in either direction.
	if _, ok := m.Recipes["service-base"]; ok {
		t.Fatalf("recipes index populated from content.scaffolds: %v", m.Recipes)
	}
	for id, dir := range m.Recipes {
		if dir == "scaffolds/service-base" {
			t.Fatalf("recipe %q points at the scaffold path %q", id, dir)
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "pack.yml")
	if err := os.WriteFile(path, []byte(scaffoldsOnlyPackYML), 0o600); err != nil {
		t.Fatalf("write scaffolds-only pack.yml: %v", err)
	}
	only, err := ParseManifestFile(path)
	if err != nil {
		t.Fatalf("ParseManifestFile on scaffolds-only manifest returned error: %v", err)
	}
	if only.Recipes != nil {
		t.Fatalf("expected nil Recipes when the index is absent, got %v", only.Recipes)
	}
	if len(only.Content.Scaffolds) != 1 {
		t.Fatalf("expected 1 scaffold on the scaffolds-only manifest, got %d", len(only.Content.Scaffolds))
	}
}

// TestPackManifest_MultipleRecipesMultipleDirs covers CLM-034: two ids map to
// two DIFFERENT directories in the pack namespace.
func TestPackManifest_MultipleRecipesMultipleDirs(t *testing.T) {
	m, err := ParseManifestFile(validRecipesPackYML)
	if err != nil {
		t.Fatalf("ParseManifestFile returned error: %v", err)
	}
	if len(m.Recipes) < 2 {
		t.Fatalf("expected at least 2 recipes, got %d (%v)", len(m.Recipes), m.Recipes)
	}
	seen := make(map[string]string, len(m.Recipes))
	for id, dir := range m.Recipes {
		if dir == "" {
			t.Fatalf("recipe %q has an empty directory", id)
		}
		if other, dup := seen[dir]; dup {
			t.Fatalf("recipes %q and %q share directory %q", other, id, dir)
		}
		seen[dir] = id
		if _, err := os.Stat(filepath.Join(filepath.Dir(validRecipesPackYML), dir, "recipe.yml")); err != nil {
			t.Fatalf("recipe %q: expected a recipe.yml under %q: %v", id, dir, err)
		}
	}
}

// TestPackManifest_RecipesIndexMissingDirErrors covers CLM-035: a recipes entry
// pointing at a missing directory — or at a directory with no recipe.yml — is a
// pack-manifest validation error, while the valid index parses clean.
func TestPackManifest_RecipesIndexMissingDirErrors(t *testing.T) {
	// Case 1: the declared directory does not exist at all.
	_, err := ParseManifestFile(missingDirRecipesPackYML)
	if err == nil {
		t.Fatal("expected an error for a recipes entry pointing at an absent directory, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ghost") {
		t.Fatalf("error does not name the recipe id: %q", msg)
	}
	if !strings.Contains(msg, "recipes/ghost") {
		t.Fatalf("error does not name the missing directory: %q", msg)
	}

	// Case 2: the directory exists but carries no recipe.yml. Same manifest,
	// same declaration — the only difference is that the directory is present.
	data, readErr := os.ReadFile(missingDirRecipesPackYML)
	if readErr != nil {
		t.Fatalf("read missing-dir fixture: %v", readErr)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "recipes", "ghost"), 0o750); err != nil {
		t.Fatalf("create empty recipe directory: %v", err)
	}
	packPath := filepath.Join(root, "pack.yml")
	if err := os.WriteFile(packPath, data, 0o600); err != nil {
		t.Fatalf("write pack.yml: %v", err)
	}
	_, err = ParseManifestFile(packPath)
	if err == nil {
		t.Fatal("expected an error for a recipe directory with no recipe.yml, got nil")
	}
	msg = err.Error()
	if !strings.Contains(msg, "ghost") {
		t.Fatalf("error does not name the recipe id: %q", msg)
	}
	if !strings.Contains(msg, "recipe.yml") {
		t.Fatalf("error does not name the missing recipe.yml: %q", msg)
	}

	// Falsifying twin: a validator that rejected every recipes: index would pass
	// the two cases above. The valid fixture must parse clean.
	if _, err := ParseManifestFile(validRecipesPackYML); err != nil {
		t.Fatalf("valid recipes index rejected: %v", err)
	}
}
