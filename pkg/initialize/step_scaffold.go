package initialize

import (
	"fmt"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// stepScaffoldName is the report name for step 6.
const stepScaffoldName = "scaffold"

// stepScaffold delivers the project's first SOURCE file — through a pack recipe, never
// from core (SPEC-069 REQ-009, REQ-035).
//
// DD-7 requires init to produce a first source file, because a compiler run over an
// empty repository can RED on "no inputs". It does produce one, WITHOUT core holding a
// single byte of it: the file arrives as a recipe PAYLOAD the consumer named. What
// REQ-009 forbids is core AUTHORING the file; what it REQUIRES is init DELIVERING one
// through a pack.
//
// IT SITS AT STEP 6 AND THE POSITION IS LOAD-BEARING. Not at 1, because a recipe
// cannot resolve out of a pack that is not installed yet. Not beside the CI step at 9,
// because the toolchain step is EXACTLY the run that would hit DD-7's empty-project
// failure — grouping "the two recipe steps" reads as a tidy-up and manufactures the
// failure this step exists to prevent.
//
// ONE SEAM, TWO CALLERS. This step calls the SAME RecipeApplier the CI step calls, with
// the ref ITS OWN flag supplied, and runs the returned preserves through the SAME
// classifier. Adding it required no new method, no per-step variant, and no place for
// core to construct a ref part — which is what the opaque-string interface bought.
//
// CORE CONSTRUCTS NO PART OF THE REF and NO PART OF THE FILE: no template, no payload,
// no filename, no extension, no path. And no param, to this apply or the CI one.
func stepScaffold(projectRoot string, ref string, applier RecipeApplier) (StepReport, []ClassifiedPreserve) {
	// Keyed on the flag being ABSENT, not on the value looking blank — see the CI
	// step's note. A whitespace value is a ref the consumer SUPPLIED, so it falls
	// through to ParseRecipeRef and fails loudly rather than being reclassified as an
	// omission that exits 0.
	if ref == "" {
		return StepReport{
			Step:    stepScaffoldName,
			Outcome: OutcomeSkipped,
			Detail: "no source file was scaffolded, because --scaffold was not supplied. To scaffold one later, run " +
				"`backstop recipe apply " + pinnedRecipeRefShape + "` with a scaffold recipe an installed pack declares. " +
				"Not every pack ecosystem ships one, so a skipped scaffold is not an error",
		}, nil
	}

	if _, err := recipe.ParseRecipeRef(ref); err != nil {
		return failedRecipeStep(stepScaffoldName, err), nil
	}

	outcome, err := applier.Apply(projectRoot, ref)
	if err != nil {
		return failedRecipeStep(stepScaffoldName, err), nil
	}

	classified := classifyApplyPreserves(outcome)
	return recipeStepReport(stepScaffoldName, outcome, classified, scaffoldPreserveSentence), classified
}

// scaffoldPreserveSentence is the SCAFFOLD step's wording.
//
// ★ ITS USER-OWNED SENTENCE IS ITS OWN, AND IT IS ONE COPY-PASTE FROM BEING FALSE
// (Sharp Edge 19). The CI step says "no backstop gate was wired into this file"; this
// step must NEVER borrow it. This step knows nothing about CI at all, so that sentence
// here would have init assert something about a file it was never asked to wire a gate
// into — the same species of false report REQ-035 exists to prevent, in a third place.
//
// What this step says instead is what it actually knows: the consumer's own file was
// left in place, so the recipe's declared source file was NOT written.
func scaffoldPreserveSentence(preserve ClassifiedPreserve) string {
	switch preserve.Class {
	case PreserveWaiverCovered:
		return fmt.Sprintf("%s was left as you customized it, and that divergence is accounted for by rule %s under the covering waiver %s",
			preserve.Path, preserve.Rule, preserve.CoveringWaiver)
	case PreserveUserOwned:
		return fmt.Sprintf("%s is your own file and was left in place, so the recipe's declared source file was NOT written. Move your version aside and run `backstop recipe apply %s` if you want the recipe's, or keep yours and ignore this",
			preserve.Path, pinnedRecipeRefShape)
	default:
		return indeterminatePreserveSentence(preserve)
	}
}
