package packval

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// RunArchetype enforces the per-archetype content contract for the three declared
// archetypes.
//
// `code` and `enforcement` keep their pack-level rule semantics byte-for-byte.
// `recipes` (ISSUE-085) is the archetype for a pack whose product IS recipes and their
// paired scaffolds. It imposes NO ruleset requirement and NO pairs_with obligation,
// because a recipes pack has no ruleset for a scaffold to pair with — those are `code`
// enforcement semantics, and demanding them of a recipes pack is exactly the padding
// this archetype exists to escape.
//
// The teeth move rather than disappear: the enforcement unit for a recipes pack is the
// RECIPE, not the pack. A scaffolding- or implementing-kind recipe REGENERATES its
// output on re-apply, so the applier is its drift enforcement and it must declare its
// own enforcement.rules. A templating recipe is one-shot and consumer-owned, so it has
// no drift story and is check-free by design — the obligation is one-directional, and
// a templating recipe that declares enforcement.rules anyway is NOT an error. Do not
// "harmonize" this branch back to a pack-level ruleset.
//
// packDir is required because the recipes branch reads each declared recipe.yml from
// disk. The parameter is symmetric with RunCoherence / RunFixtures / RunLayer, which
// already take it.
func RunArchetype(pack *PackManifest, packDir string) *PhaseResult {
	res := &PhaseResult{
		Phase:  "phase4-archetype",
		Status: "pass",
		Checks: 3, // archetype-rules, co-occurrence, recipe-enforcement
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}
	if pack.Archetype == "code" && len(pack.Content.Ruleset.Rules) == 0 {
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "code-rules", Message: "code pack requires rules"})
	}
	if pack.Archetype == "code" {
		ruleMap := map[string]bool{}
		for _, r := range pack.Content.Ruleset.Rules {
			ruleMap[r.ID] = true
			if len(r.PairsWith.Scaffolds) == 0 && r.PairsWith.SDK == "" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "pairs_with", Rule: r.ID, Message: "rule missing pairs_with"})
			}
		}
		for _, s := range pack.Content.Scaffolds {
			if len(s.PairsWith.Rules) == 0 {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "pairs_with", Rule: s.ID, Message: "scaffold missing pairs_with rules"})
				continue
			}
			for _, rid := range s.PairsWith.Rules {
				if !ruleMap[rid] {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "pairs_with", Rule: s.ID, Message: "scaffold pairs_with unresolved rule"})
				}
			}
		}
	}
	if pack.Archetype == "enforcement" {
		if len(pack.Content.Scaffolds) > 0 {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "enforcement-content", Message: "enforcement pack must not include scaffolds"})
		}
		if pack.Content.SDK != nil {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "enforcement-content", Message: "enforcement pack must not include sdk"})
		}
	}
	if pack.Archetype == "recipes" {
		checkRecipeEnforcement(pack, packDir, res)
	}
	if len(res.Errors) > 0 {
		res.Status = "fail"
	}
	return res
}

// checkRecipeEnforcement is the recipes archetype's content check.
//
// Rules are PERMITTED under `recipes` but never required. The guard against the
// archetype becoming a rule-less escape hatch is the recipes-required check below, not
// a prohibition nobody ratified: a pack with neither recipes nor a ruleset is an empty
// pack.
//
// Every error names the recipe in the MESSAGE as well as in Rule. That is not
// redundancy: the text formatter prints Phase/Check/Message and DISCARDS Rule, and text
// is the human default for `pack check`, so an id living only in Rule is invisible to
// the pack author standing at the terminal — the one person the diagnostic exists for.
// Rule stays populated for JSON consumers and for symmetry with phase1's scaffold ids.
//
// SCOPE GUARD: the enforcement check is PRESENCE-ONLY. The strings inside
// enforcement.rules are never resolved or cross-referenced against anything — there is
// no pack ruleset to resolve them against, and inventing one would re-import the very
// assumption this archetype exists to escape.
func checkRecipeEnforcement(pack *PackManifest, packDir string, res *PhaseResult) {
	if len(pack.Recipes) == 0 {
		res.Errors = append(res.Errors, ValidationError{
			Phase:        res.Phase,
			Check:        "recipes-required",
			Message:      "recipes pack declares no recipes; a pack with neither recipes nor a ruleset is an empty pack",
			ManifestPath: "recipes",
		})
		return
	}

	// Sorted so error order is deterministic across runs, mirroring how the runtime
	// model's own recipes-index validation iterates.
	ids := make([]string, 0, len(pack.Recipes))
	for id := range pack.Recipes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		relPath := filepath.Join(pack.Recipes[id], "recipe.yml")
		data, readErr := os.ReadFile(filepath.Join(packDir, relPath))
		if readErr != nil {
			res.Errors = append(res.Errors, recipeManifestError(res.Phase, id, fmt.Sprintf("reading %s: %v", relPath, readErr)))
			continue
		}
		manifest, parseErr := recipe.ParseRecipeManifest(data)
		if parseErr != nil {
			res.Errors = append(res.Errors, recipeManifestError(res.Phase, id, fmt.Sprintf("parsing %s: %v", relPath, parseErr)))
			continue
		}

		// Both failure paths above CONTINUE rather than falling through with a
		// zero-value manifest. Falling through would read the missing kind as "not
		// scaffolding, not implementing, therefore exempt" — a silent hole straight
		// through the teeth that would exempt exactly the recipes that failed.
		switch manifest.Kind {
		case recipe.KindScaffolding, recipe.KindImplementing:
			// `enforcement:` present with an empty rules list counts as ABSENT: an
			// empty list is not a declaration.
			if manifest.Enforcement == nil || len(manifest.Enforcement.Rules) == 0 {
				res.Errors = append(res.Errors, ValidationError{
					Phase:        res.Phase,
					Check:        "recipe-enforcement",
					Rule:         id,
					Message:      fmt.Sprintf("recipe %q: %s-kind recipe must declare enforcement.rules", id, manifest.Kind),
					ManifestPath: "recipes." + id,
				})
			}
		}
		// recipe.KindTemplating carries no obligation: one-shot, consumer-owned output
		// has no drift to police.
	}
}

// recipeManifestError builds the recipe-manifest diagnostic for a declared recipe whose
// recipe.yml could not be read or parsed, naming the recipe in the message text.
func recipeManifestError(phase, id, detail string) ValidationError {
	return ValidationError{
		Phase:        phase,
		Check:        "recipe-manifest",
		Rule:         id,
		Message:      fmt.Sprintf("recipe %q: %s", id, detail),
		ManifestPath: "recipes." + id,
	}
}
