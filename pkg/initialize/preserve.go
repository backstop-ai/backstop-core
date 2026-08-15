package initialize

import "github.com/backstop-ai/backstop-core/pkg/recipe"

// classifyPreserve is THE ONE preserve classifier, shared by every recipe apply init
// performs (SPEC-069 REQ-035 / CLM-144).
//
// It reads exactly the two observables init holds — the divergence's
// Rule/CoveringWaiver pair, and the resolved recipe's DECLARED kind — and nothing
// else. A second, step-local copy is the "one authority, not a copy" hazard this spec
// refuses everywhere else: two classifiers drifting would let one recipe step report a
// class the other would not.
//
// THE PAIR OUTRANKS THE KIND, and the branch order below says so. A populated pair is
// producer (c) at EVERY declared kind, `templating` included; the kind is consulted
// ONLY to settle the empty-pair case.
//
// WHY THE KIND CANNOT DO MORE THAN THAT. preserveOrRegenerate (pkg/recipe/apply.go)
// tests `!own.adopted` FIRST and returns immediately, so its kind test is UNREACHABLE
// for any recipe no prior apply adopted. A never-adopted `kind: templating` recipe
// therefore returns a value BYTE-IDENTICAL to an adopted-and-materialized one, and the
// adoption bit that would separate them is not carried by anything the apply returns.
// The kind settles the empty-pair case only for the two kinds where the ambiguous
// branch is unreachable. That is not a gap to be closed by cleverness — reconstructing
// the applier's adoption identity here would be a second derivation of it, and
// surfacing the applier's own bit would require editing pkg/recipe, which REQ-009
// forbids. The ambiguity is RECORDED, not engineered around.
//
// WHAT DIFFERS BETWEEN THE TWO CALLING STEPS IS THE SENTENCE, NEVER THE CLASS. The CI
// step's user-owned wording is a statement about a gate; the scaffold step's is a
// statement about a source file. De-duplicating the STRING along with the classifier
// would make init tell a consumer something it cannot know.
func classifyPreserve(divergence recipe.PreservedDivergence, recipeKind string) PreserveClass {
	if divergence.Rule != "" && divergence.CoveringWaiver != "" {
		return PreserveWaiverCovered
	}

	switch recipeKind {
	case recipe.KindScaffolding, recipe.KindImplementing:
		return PreserveUserOwned
	default:
		// recipe.KindTemplating, and any kind this classifier does not recognize.
		// Defaulting an UNKNOWN kind to the conservative class is deliberate: a kind
		// init cannot reason about is exactly the state DD-15's "on 'I cannot tell',
		// REFUSE" posture governs, and defaulting it to user-owned would have init
		// assert "no backstop gate was wired into this file" on no evidence.
		return PreserveIndeterminate
	}
}

// classifyApplyPreserves is THE ONE HOP both recipe steps take to reach the
// classifier.
//
// It is a LOOP, not a second classifier: it reads no recipe kind of its own and makes
// no decision, which is why the structural claim counts kind-READING functions rather
// than functions that merely touch preserves. Its value is that there is exactly ONE
// named call site of classifyPreserve in the whole init source set, so "both recipe
// steps run their preserves through the shared classifier" is a chain a test can walk
// hop by hop instead of a property it has to take on trust.
func classifyApplyPreserves(outcome ApplyOutcome) []ClassifiedPreserve {
	classified := make([]ClassifiedPreserve, 0, len(outcome.Preserved))
	for _, divergence := range outcome.Preserved {
		classified = append(classified, ClassifiedPreserve{
			Path:           divergence.Path,
			Class:          classifyPreserve(divergence, outcome.RecipeKind),
			Rule:           divergence.Rule,
			CoveringWaiver: divergence.CoveringWaiver,
		})
	}
	return classified
}
