package packval_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/packval"
)

// The recipe.yml bodies the phase4 cases declare. They are authored by hand
// rather than captured, which is correct for a DECLARATION: the captured-from-
// real-output rule governs tool output (SARIF, coverage), not manifests. What is
// never fabricated here is the expected report — every assertion reads the real
// pipeline result.
const (
	recipeScaffoldingWithEnforcement = `kind: scaffolding
version: 1.0.0
enforcement:
  rules:
    - recipe.service.layout
ops:
  - id: create-service-manifest
    kind: create
    target: service/service.json
    payload: payload/service.json
`

	recipeScaffoldingNoEnforcement = `kind: scaffolding
version: 1.0.0
ops:
  - id: create-service-manifest
    kind: create
    target: service/service.json
    payload: payload/service.json
`

	recipeScaffoldingEmptyEnforcementRules = `kind: scaffolding
version: 1.0.0
enforcement:
  rules: []
ops:
  - id: create-service-manifest
    kind: create
    target: service/service.json
    payload: payload/service.json
`

	recipeImplementingWithEnforcement = `kind: implementing
version: 1.0.0
enforcement:
  rules:
    - recipe.lint.config-present
ops:
  - id: create-lint-config
    kind: create
    target: lint.config.json
    payload: payload/lint.config.json
`

	recipeImplementingNoEnforcement = `kind: implementing
version: 1.0.0
ops:
  - id: create-lint-config
    kind: create
    target: lint.config.json
    payload: payload/lint.config.json
`

	recipeTemplatingNoEnforcement = `kind: templating
version: 1.0.0
ops:
  - id: insert-badge
    kind: insert
    target: README.md
    anchor: "<!-- badges -->"
    snippet: "![gated](docs/badge.svg)"
    manual: Add the badge line by hand under the badges comment.
`

	recipeTemplatingWithEnforcement = `kind: templating
version: 1.0.0
enforcement:
  rules:
    - recipe.readme.badge-present
ops:
  - id: insert-badge
    kind: insert
    target: README.md
    anchor: "<!-- badges -->"
    snippet: "![gated](docs/badge.svg)"
    manual: Add the badge line by hand under the badges comment.
`

	// recipeMalformed parses as YAML but is rejected by
	// recipe.ParseRecipeManifest: `bogus` is not one of the three declared kinds.
	recipeMalformed = `kind: bogus
version: 1.0.0
ops:
  - id: create-service-manifest
    kind: create
    target: service/service.json
`
)

// writeRecipeManifests materializes a {pack-relative recipe dir -> recipe.yml
// body} map under packDir. An EMPTY body creates the directory with NO
// recipe.yml, which is the missing-manifest case.
//
// It is deliberately local to this file: the existing packval test files carry
// their own helpers and this one exists only for the recipes-archetype cases.
func writeRecipeManifests(t *testing.T, packDir string, recipes map[string]string) {
	t.Helper()
	for rel, body := range recipes {
		full := filepath.Join(packDir, rel)
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		if body == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(full, "recipe.yml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// recipesPackManifest builds the field-guide shape in memory: archetype recipes,
// a top-level recipes index, ONE bare scaffold and NO ruleset.
func recipesPackManifest(index map[string]string) *packval.PackManifest {
	return &packval.PackManifest{
		Name:      "demo-org/recipes-field-guide",
		Version:   "1.0.0",
		Language:  "any",
		Archetype: "recipes",
		Recipes:   index,
		Content: packval.Content{
			Scaffolds: []packval.Scaffold{{ID: "service-base", Path: "scaffolds/service-base"}},
		},
	}
}

// runRecipesArchetype writes the given recipe bodies into a fresh temp pack dir
// and runs phase4 over a field-guide manifest indexing them.
func runRecipesArchetype(t *testing.T, recipes map[string]string) *packval.PhaseResult {
	t.Helper()
	dir := t.TempDir()
	writeRecipeManifests(t, dir, recipes)
	index := make(map[string]string, len(recipes))
	for rel := range recipes {
		index[filepath.Base(rel)] = rel
	}
	return packval.RunArchetype(recipesPackManifest(index), dir)
}

// firstErrorWithCheck returns the first error on the phase result whose Check
// matches, or nil.
func firstErrorWithCheck(res *packval.PhaseResult, check string) *packval.ValidationError {
	for i := range res.Errors {
		if res.Errors[i].Check == check {
			return &res.Errors[i]
		}
	}
	return nil
}

// assertNamesRecipe pins the diagnostic identity: the recipe id must reach BOTH
// the Rule slot (for JSON consumers, symmetric with phase1's scaffold ids) and
// the MESSAGE TEXT. The text renderer prints Phase/Check/Message only and
// discards Rule, so an id living only in Rule is invisible to the pack author at
// the terminal — the one person the diagnostic exists for.
func assertNamesRecipe(t *testing.T, e *packval.ValidationError, id string) {
	t.Helper()
	if e.Rule != id {
		t.Errorf("error must carry the recipe id in Rule, want %q got %q", id, e.Rule)
	}
	if !strings.Contains(e.Message, id) {
		t.Errorf("error message must NAME the recipe %q (the text renderer discards Rule), got: %q", id, e.Message)
	}
}

// TestPackVal_P1_RecipesArchetypeAccepted (CLM-001) pins that phase1's archetype
// enum admits `recipes`. The falsifier lives in the same test: an unknown
// archetype must still be rejected, so the enum grew by exactly one member
// rather than being opened up.
func TestPackVal_P1_RecipesArchetypeAccepted(t *testing.T) {
	m := &packval.PackManifest{
		Name:      "demo-org/recipes-field-guide",
		Version:   "1.0.0",
		Language:  "any",
		Archetype: "recipes",
		Content: packval.Content{
			Scaffolds: []packval.Scaffold{{ID: "service-base"}},
		},
	}
	if e := firstErrorWithCheck(packval.RunStructural(m, t.TempDir()), "archetype"); e != nil {
		t.Fatalf("archetype recipes must be accepted by phase1, got: %+v", *e)
	}

	m.Archetype = "hybrid"
	if firstErrorWithCheck(packval.RunStructural(m, t.TempDir()), "archetype") == nil {
		t.Fatal("phase1 must still reject an unknown archetype; the enum grew by exactly one member")
	}
}

// TestPackVal_Manifest_ParsesTopLevelRecipesIndex (CLM-002) pins that packval
// parses the TOP-LEVEL `recipes:` key — mirroring the runtime model's own
// top-level index — and that it and content.scaffolds never populate each other.
// `recipes:` is a top-level pack.yml key, not a content: key, so this is the
// honest parse rather than a back door into the content model.
func TestPackVal_Manifest_ParsesTopLevelRecipesIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pack.yml", strings.TrimSpace(`
name: demo-org/recipes-field-guide
version: 1.0.0
language: any
archetype: recipes
recipes:
  scaffold-service: recipes/scaffold-service
  adopt-lint: recipes/adopt-lint
content:
  scaffolds:
    - id: service-base
      path: scaffolds/service-base
`))

	m, err := packval.ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	want := map[string]string{
		"scaffold-service": "recipes/scaffold-service",
		"adopt-lint":       "recipes/adopt-lint",
	}
	if len(m.Recipes) != len(want) {
		t.Fatalf("want %d recipes parsed, got %d (%v)", len(want), len(m.Recipes), m.Recipes)
	}
	for id, dirPath := range want {
		if got := m.Recipes[id]; got != dirPath {
			t.Errorf("recipe %q: want dir %q, got %q", id, dirPath, got)
		}
	}

	// The scaffolds block populates INDEPENDENTLY of the recipes index.
	if len(m.Content.Scaffolds) != 1 || m.Content.Scaffolds[0].ID != "service-base" {
		t.Fatalf("content.scaffolds must parse independently, got %+v", m.Content.Scaffolds)
	}

	// Neither key leaks into the other: no recipe id appears as a scaffold, and no
	// scaffold id appears in the recipes index.
	for _, s := range m.Content.Scaffolds {
		if _, collides := m.Recipes[s.ID]; collides {
			t.Errorf("scaffold %q leaked into the recipes index", s.ID)
		}
	}
	for id := range m.Recipes {
		for _, s := range m.Content.Scaffolds {
			if s.ID == id {
				t.Errorf("recipe %q leaked into content.scaffolds", id)
			}
		}
	}
}

// TestPackVal_P1_RecipesOnlyPackStillFailsContentRequired (CLM-014) DOCUMENTS A
// DELIBERATE GAP, not a desired behavior.
//
// A pack that declares recipes and nothing else still fails phase1's
// "content is required" check, because this change adds the top-level `recipes:`
// parse WITHOUT widening the content model. SPEC-054's sharp edge
// "A RECIPES-ONLY pack does not validate yet — `recipes:` alone is not
// 'content'." names the three content-is-required sites as a tracked follow-up
// (cited by heading, never by line number — the offsets move).
//
// Whoever lands that widening owns this assertion and should INVERT it. Until
// then it is the live falsifiable marker that the gap is still open, rather than
// a silent hole.
func TestPackVal_P1_RecipesOnlyPackStillFailsContentRequired(t *testing.T) {
	m := &packval.PackManifest{
		Name:      "demo-org/recipes-only",
		Version:   "1.0.0",
		Language:  "any",
		Archetype: "recipes",
		Recipes:   map[string]string{"scaffold-service": "recipes/scaffold-service"},
	}

	res := packval.RunStructural(m, t.TempDir())
	if firstErrorWithCheck(res, "content") == nil {
		t.Fatalf("a recipes-ONLY pack must still fail content-is-required until SPEC-054's three-site widening lands; got %+v", res.Errors)
	}
}

// TestPackVal_P4_RecipesPack_ScaffoldsAndRecipesNoRulesetPasses (CLM-003) is the
// field-guide shape: scaffolds plus recipes, no ruleset, no decoy rule, no
// pairs_with wiring. The falsifier is baked into the fixture — the scaffold
// carries no pairs_with and the ruleset is empty, so a recipes branch that
// leaked `code` semantics fails right here.
func TestPackVal_P4_RecipesPack_ScaffoldsAndRecipesNoRulesetPasses(t *testing.T) {
	res := runRecipesArchetype(t, map[string]string{
		"recipes/scaffold-service": recipeScaffoldingWithEnforcement,
		"recipes/adopt-lint":       recipeImplementingWithEnforcement,
		"recipes/readme-badge":     recipeTemplatingNoEnforcement,
	})
	if res.Status != "pass" {
		t.Fatalf("the field-guide shape must pass phase4 with no ruleset and no pairs_with; got %s: %+v", res.Status, res.Errors)
	}
}

// TestPackVal_P4_RecipesPack_ScaffoldingRecipeWithoutEnforcementFails (CLM-005):
// a scaffolding-kind recipe regenerates its output, so it owes a drift story.
// Asserting the specific check AND the named id matters — a bare status check
// would pass against a validator failing for an unrelated reason.
func TestPackVal_P4_RecipesPack_ScaffoldingRecipeWithoutEnforcementFails(t *testing.T) {
	res := runRecipesArchetype(t, map[string]string{
		"recipes/scaffold-service": recipeScaffoldingNoEnforcement,
	})
	if res.Status != "fail" {
		t.Fatalf("a scaffolding recipe with no enforcement.rules must fail phase4, got %s", res.Status)
	}
	e := firstErrorWithCheck(res, "recipe-enforcement")
	if e == nil {
		t.Fatalf("want a recipe-enforcement error, got %+v", res.Errors)
	}
	assertNamesRecipe(t, e, "scaffold-service")
}

// TestPackVal_P4_RecipesPack_ImplementingRecipeWithoutEnforcementFails (CLM-006)
// is the same obligation on the other regenerating kind.
func TestPackVal_P4_RecipesPack_ImplementingRecipeWithoutEnforcementFails(t *testing.T) {
	res := runRecipesArchetype(t, map[string]string{
		"recipes/adopt-lint": recipeImplementingNoEnforcement,
	})
	if res.Status != "fail" {
		t.Fatalf("an implementing recipe with no enforcement.rules must fail phase4, got %s", res.Status)
	}
	e := firstErrorWithCheck(res, "recipe-enforcement")
	if e == nil {
		t.Fatalf("want a recipe-enforcement error, got %+v", res.Errors)
	}
	assertNamesRecipe(t, e, "adopt-lint")
}

// TestPackVal_P4_RecipesPack_TemplatingRecipeExempt (CLM-007): templating is
// one-shot and consumer-owned, so there is no drift story to enforce. The second
// case pins that the obligation is ONE-DIRECTIONAL — a templating recipe that
// declares enforcement.rules anyway is not an error — so a future "templating
// must not declare enforcement" cannot creep in unratified.
func TestPackVal_P4_RecipesPack_TemplatingRecipeExempt(t *testing.T) {
	cases := map[string]string{
		"no enforcement block":        recipeTemplatingNoEnforcement,
		"declares enforcement anyway": recipeTemplatingWithEnforcement,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			res := runRecipesArchetype(t, map[string]string{"recipes/readme-badge": body})
			if res.Status != "pass" {
				t.Fatalf("a templating recipe (%s) is exempt and must pass; got %s: %+v", name, res.Status, res.Errors)
			}
		})
	}
}

// TestPackVal_P4_RecipesPack_EmptyEnforcementRulesCountsAsAbsent (CLM-008): an
// empty list is not a declaration.
func TestPackVal_P4_RecipesPack_EmptyEnforcementRulesCountsAsAbsent(t *testing.T) {
	res := runRecipesArchetype(t, map[string]string{
		"recipes/scaffold-service": recipeScaffoldingEmptyEnforcementRules,
	})
	if res.Status != "fail" {
		t.Fatalf("`enforcement:` with `rules: []` must count as absent and fail, got %s", res.Status)
	}
	e := firstErrorWithCheck(res, "recipe-enforcement")
	if e == nil {
		t.Fatalf("want a recipe-enforcement error, got %+v", res.Errors)
	}
	assertNamesRecipe(t, e, "scaffold-service")
}

// TestPackVal_P4_RecipesPack_NoRecipesDeclaredFails (CLM-009) is the
// anti-escape-hatch assertion. A pack with no recipes and no ruleset is not a
// recipes pack; it is an empty pack. Without this the new archetype would admit
// exactly the rule-less packs the rejected direction 2 would have allowed.
func TestPackVal_P4_RecipesPack_NoRecipesDeclaredFails(t *testing.T) {
	m := recipesPackManifest(nil)
	res := packval.RunArchetype(m, t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("a recipes pack declaring no recipes must fail phase4, got %s", res.Status)
	}
	if firstErrorWithCheck(res, "recipes-required") == nil {
		t.Fatalf("want a recipes-required error, got %+v", res.Errors)
	}
}

// TestPackVal_P4_RecipesPack_UnreadableRecipeManifestFailsLoud (CLM-010) closes
// the silent hole straight through the teeth: an implementation that treats an
// unreadable manifest as "kind unknown, therefore not non-templating, therefore
// exempt" would make every failing recipe exempt from CLM-005/CLM-006. Both a
// MISSING recipe.yml and an UNPARSEABLE one must fail loud, naming the id.
func TestPackVal_P4_RecipesPack_UnreadableRecipeManifestFailsLoud(t *testing.T) {
	cases := map[string]string{
		"no recipe.yml on disk":  "",
		"unparseable recipe.yml": recipeMalformed,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			res := runRecipesArchetype(t, map[string]string{"recipes/scaffold-service": body})
			if res.Status != "fail" {
				t.Fatalf("an unreadable recipe manifest (%s) must fail loud, got %s: %+v", name, res.Status, res.Errors)
			}
			e := firstErrorWithCheck(res, "recipe-manifest")
			if e == nil {
				t.Fatalf("want a recipe-manifest error, got %+v", res.Errors)
			}
			assertNamesRecipe(t, e, "scaffold-service")
			if firstErrorWithCheck(res, "recipe-enforcement") != nil {
				t.Error("an unreadable manifest must not ALSO be adjudicated against the enforcement rule with a zero-value manifest")
			}
		})
	}
}

// TestPackVal_P4_CodeAndEnforcementArchetypeVerdictsUnchanged (CLM-013)
// re-asserts every pre-existing archetype verdict through the NEW phase4
// signature, so the recipes branch is provably additive. The enum grew by
// exactly one member: an unknown archetype is still rejected at phase1.
func TestPackVal_P4_CodeAndEnforcementArchetypeVerdictsUnchanged(t *testing.T) {
	codePack := func() *packval.PackManifest {
		return &packval.PackManifest{
			Name: "acme/example", Version: "1.0.0", Language: "go", Archetype: "code",
			Content: packval.Content{Ruleset: packval.Ruleset{Rules: []packval.Rule{{ID: "R1"}}}},
		}
	}

	noRules := codePack()
	noRules.Content.Ruleset.Rules = nil

	pairedCode := codePack()
	pairedCode.Content.Ruleset.Rules[0].PairsWith.Scaffolds = []string{"S1"}
	pairedCode.Content.Scaffolds = []packval.Scaffold{{ID: "S1", PairsWith: packval.PairsWith{Rules: []string{"R1"}}}}

	enforcementWithScaffolds := codePack()
	enforcementWithScaffolds.Archetype = "enforcement"
	enforcementWithScaffolds.Content.Scaffolds = []packval.Scaffold{{ID: "S1"}}

	enforcementWithSDK := codePack()
	enforcementWithSDK.Archetype = "enforcement"
	enforcementWithSDK.Content.SDK = &packval.SDK{Provides: []string{"x"}}

	enforcementRulesOnly := codePack()
	enforcementRulesOnly.Archetype = "enforcement"

	cases := []struct {
		name string
		pack *packval.PackManifest
		want string
	}{
		{"code pack with no rules fails", noRules, "fail"},
		{"code pack with rules and matched pairs_with passes", pairedCode, "pass"},
		{"enforcement pack with scaffolds fails", enforcementWithScaffolds, "fail"},
		{"enforcement pack with sdk fails", enforcementWithSDK, "fail"},
		{"rules-only enforcement pack passes", enforcementRulesOnly, "pass"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := packval.RunArchetype(tc.pack, t.TempDir()).Status; got != tc.want {
				t.Fatalf("archetype verdict moved: want %s, got %s", tc.want, got)
			}
		})
	}

	hybrid := codePack()
	hybrid.Archetype = "hybrid"
	if firstErrorWithCheck(packval.RunStructural(hybrid, t.TempDir()), "archetype") == nil {
		t.Fatal("phase1 must still reject archetype \"hybrid\"")
	}
}
