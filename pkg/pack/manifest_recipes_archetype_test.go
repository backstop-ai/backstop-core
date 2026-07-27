package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file pins the GATE-TIME LOAD seam for the `recipes` archetype (ISSUE-085).
//
// Widening packval alone would ship a pack that installs and then breaks
// `backstop gate`: the gate parses EVERY declared pack through ParseManifestFile
// on every run and hard-fails the run on a parse error, while nothing in
// pkg/pack/distribution calls ParseManifestFile at all — so `pack add` never
// reaches these checks. Install and load would disagree, and a field-guide pack
// would install cleanly and then make the project ungateable. Both halves of the
// seam are pinned here: the archetype enum, and the scaffold requirements.

// TestValidateArchetype_Recipes (CLM-004) pins that the load-path enum admits
// `recipes`, and — in the same test — that it grew by exactly one member.
func TestValidateArchetype_Recipes(t *testing.T) {
	if err := validateArchetype("recipes"); err != nil {
		t.Fatalf("archetype recipes must be accepted at the load seam, got: %v", err)
	}
	if err := validateArchetype("library"); err == nil {
		t.Fatal("an unknown archetype must still be rejected; the enum grew by exactly one member")
	}
}

// TestValidateScaffold_RecipesArchetypeRelaxesRulePairingRequirements (CLM-015)
// is the blocker's own unit. validateScaffold runs unconditionally over every
// scaffold at load and today demands a valid tier, a test_command, use_when,
// assumes AND pairs_with — five cascading failures for the field-guide pack's
// bare {id, path} scaffold, four of them the enforcement-pack padding ISSUE-085
// exists to remove.
//
// Under `recipes` those five REQUIREMENTS do not apply. The relaxation drops
// requirements, never VALIDATION: a DECLARED tier must still be complete or
// skeleton, and sample_config values must still be strings. Every other
// archetype keeps all five byte-for-byte — without those cases a relaxation that
// leaked past the recipes branch would silently gut the five existing negative
// assertions in TestValidateScaffold_{MissingTestCommand, MissingUseWhen,
// MissingAssumes, MissingPairsWith, EmptyUseWhen} while their names survived.
func TestValidateScaffold_RecipesArchetypeRelaxesRulePairingRequirements(t *testing.T) {
	bare := func() Scaffold {
		return Scaffold{ID: "service-base", Path: "scaffolds/service-base"}
	}
	bogusTier := bare()
	bogusTier.Tier = "bogus"

	declaredTier := bare()
	declaredTier.Tier = "skeleton"

	nonStringSampleConfig := bare()
	nonStringSampleConfig.SampleConfig = map[string]any{"retries": 3}

	cases := []struct {
		name      string
		scaffold  Scaffold
		archetype string
		wantErr   bool
	}{
		{"bare scaffold under recipes is accepted", bare(), "recipes", false},
		{"bare scaffold under code still fails", bare(), "code", true},
		{"bare scaffold under enforcement still fails", bare(), "enforcement", true},
		{"recipes: a DECLARED tier is still validated", bogusTier, "recipes", true},
		{"recipes: a valid declared tier is accepted", declaredTier, "recipes", false},
		{"recipes: sample_config values must still be strings", nonStringSampleConfig, "recipes", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateScaffold(tc.scaffold, tc.archetype)
			if tc.wantErr && err == nil {
				t.Fatalf("archetype %q: expected an error, got nil", tc.archetype)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("archetype %q: expected no error, got: %v", tc.archetype, err)
			}
		})
	}
}

// TestParseManifestFile_RecipesArchetypePackLoads (CLM-004, CLM-015) integrates
// the two units above at the seam the gate actually uses.
//
// The scaffold below MUST stay unpadded. If this test goes red, the fix is the
// archetype-conditional branch in validateScaffold — never adding tier /
// test_command / use_when / assumes / pairs_with to this manifest. Padding it
// would make CLM-004 mean "loads only if padded", which is precisely the outcome
// ISSUE-085 exists to prevent.
func TestParseManifestFile_RecipesArchetypePackLoads(t *testing.T) {
	dir := t.TempDir()

	// validateRecipesIndex fail-louds on an index entry whose directory carries no
	// recipe.yml, so the declared dirs must be real.
	for _, rel := range []string{"recipes/scaffold-service", "recipes/readme-badge"} {
		if err := os.MkdirAll(filepath.Join(dir, rel), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel, "recipe.yml"), []byte("kind: scaffolding\nversion: 1.0.0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	manifestPath := filepath.Join(dir, "pack.yml")
	if err := os.WriteFile(manifestPath, []byte(strings.TrimSpace(`
name: demo-org/recipes-field-guide
version: 1.0.0
language: any
archetype: recipes
description: Field-guide pack shipping recipes and a paired scaffold, with no rule set of its own.
recipes:
  scaffold-service: recipes/scaffold-service
  readme-badge: recipes/readme-badge
content:
  scaffolds:
    - id: service-base
      path: scaffolds/service-base
`)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ParseManifestFile(manifestPath)
	if err != nil {
		t.Fatalf("the UNPADDED field-guide pack must load at the gate-time seam, got: %v", err)
	}
	if m.Archetype != "recipes" {
		t.Errorf("want archetype recipes, got %q", m.Archetype)
	}
	want := map[string]string{
		"scaffold-service": "recipes/scaffold-service",
		"readme-badge":     "recipes/readme-badge",
	}
	if len(m.Recipes) != len(want) {
		t.Fatalf("want %d recipes parsed, got %d (%v)", len(want), len(m.Recipes), m.Recipes)
	}
	for id, rel := range want {
		if got := m.Recipes[id]; got != rel {
			t.Errorf("recipe %q: want dir %q, got %q", id, rel, got)
		}
	}
}
