package initialize

import (
	"fmt"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// stepCIName is the report name for step 9.
const stepCIName = "ci"

// stepCI wires CI when — and only when — the consumer passed `--ci` (SPEC-069
// REQ-016, REQ-017, REQ-035).
//
// IT IS NOT A CAPABILITY. There is no `--no-ci` because OMISSION IS THE OPT-OUT, and
// adding `ci` to the capability set would give one outcome (no CI wired) two report
// paths and two justifications. No `ci` verb or subcommand exists either.
//
// The step does exactly three things:
//
//  1. hands the `--ci` value to recipe.ParseRecipeRef BYTE-IDENTICALLY;
//  2. on ANY failure, surfaces that error VERBATIM attributed to this step, lets every
//     other step complete, and marks the step a broken promise;
//  3. on success, reads the result's Written and Preserved slices and runs EVERY
//     preserve through the ONE shared classifier.
//
// INIT IMPLEMENTS NO CI DETECTION. It never enumerates installed packs looking for a
// CI pack, never probes for a platform config file, and adds NO guidance text of its
// own. The shipped errors already name what was missing and what IS available, which
// is precisely why init adds nothing: any further guidance belongs to `recipe apply`.
//
// NO RECIPE PARAM IS SUPPLIED. A recipe declaring a param required with no default
// cannot be applied by init at all; the shipped apply's own error surfaces verbatim
// and the consumer finishes with `backstop recipe apply --param`. Deriving one — most
// temptingly the project basename already computed for `project:` — would be core
// constructing recipe INPUT, the same defect one layer in as core constructing the
// pack half of a ref.
func stepCI(projectRoot string, ref string, applier RecipeApplier) (StepReport, []ClassifiedPreserve) {
	// THE SKIP IS KEYED ON THE FLAG BEING ABSENT, NOT ON THE VALUE LOOKING BLANK.
	//
	// An earlier version trimmed first, which made `--ci "   "` report as a deliberate
	// no-op exiting 0 — a ref the consumer DID supply, silently reclassified as one they
	// did not. That is the exact pair the exit-code matrix exists to keep apart:
	// omitted is a reported no-op, supplied-and-unresolvable is a broken promise. A
	// whitespace value now falls through to ParseRecipeRef, whose own error surfaces
	// verbatim like every other malformed ref.
	if ref == "" {
		return StepReport{
			Step:    stepCIName,
			Outcome: OutcomeSkipped,
			Detail: "no CI was wired, because --ci was not supplied. To wire it later, run " +
				"`backstop recipe apply " + pinnedRecipeRefShape + "` with a CI recipe an installed pack declares",
		}, nil
	}

	// The parse runs HERE, on the consumer's string, before anything else touches it.
	// Its failure is fail-loud and verbatim: the shipped parser accepts only a fully
	// pinned ref — no "latest", no default version, no tolerance branch — so init
	// performs no pin defaulting anywhere.
	if _, err := recipe.ParseRecipeRef(ref); err != nil {
		return failedRecipeStep(stepCIName, err), nil
	}

	// The WHOLE ORIGINAL STRING goes to the applier, never a ref reassembled from the
	// parse above. Reassembly would be core constructing a ref out of parts it had
	// taken apart, which is the same defect as constructing one from nothing.
	outcome, err := applier.Apply(projectRoot, ref)
	if err != nil {
		return failedRecipeStep(stepCIName, err), nil
	}

	classified := classifyApplyPreserves(outcome)
	return recipeStepReport(stepCIName, outcome, classified, ciPreserveSentence), classified
}

// pinnedRecipeRefShape is the ref SHAPE init names when it tells a consumer how to run
// a recipe later.
//
// It is the SHAPE and never an example, because an example would require core to hold
// a pack name, a recipe id and a version — the three things it must not hold.
const pinnedRecipeRefShape = "<pack>:<recipe>@<version>"

// failedRecipeStep renders a resolve or apply failure as a broken promise carrying the
// shipped error VERBATIM.
//
// It adds an attribution prefix and NOTHING ELSE: no re-classification, no guidance,
// no re-wording. Every failure mode — a malformed ref, an uninstalled pack, an
// undeclared recipe, a pin mismatch, an unparseable pack-side recipe manifest — is
// surfaced identically, because the shipped error already says what went wrong and
// init classifying them differently would be init claiming knowledge it does not have.
func failedRecipeStep(step string, err error) StepReport {
	return StepReport{
		Step:    step,
		Outcome: OutcomeBrokenPromise,
		Detail:  fmt.Sprintf("the %s recipe could not be applied: %s", step, err.Error()),
	}
}

// ciPreserveSentence is the CI step's USER-OWNED wording.
//
// ★ IT IS THIS STEP'S ALONE (Sharp Edge 19). "No backstop gate was wired into this
// file" is an assertion about CI WIRING. CLM-144 requires ONE preserve classifier
// shared by both recipe steps, which makes sharing this STRING look like the same kind
// of de-duplication. It is not: the scaffold step knows nothing about CI, so lending
// it this sentence would have init tell a consumer something it cannot know.
func ciPreserveSentence(preserve ClassifiedPreserve) string {
	switch preserve.Class {
	case PreserveWaiverCovered:
		return fmt.Sprintf("%s was left as you customized it, and that divergence is accounted for by rule %s under the covering waiver %s — the gate IS wired into it",
			preserve.Path, preserve.Rule, preserve.CoveringWaiver)
	case PreserveUserOwned:
		return fmt.Sprintf("%s is your own file and was left in place, so %s. Wire one in by hand, or run `backstop recipe apply %s` once you have moved your own configuration aside",
			preserve.Path, noGateWiredWording, pinnedRecipeRefShape)
	default:
		return indeterminatePreserveSentence(preserve)
	}
}

// noGateWiredWording is the CI step's assertion, held as a constant so the scaffold
// step's negative assertion and this positive one cannot drift apart.
const noGateWiredWording = "no backstop gate was wired into this file"

// indeterminatePreserveSentence is the wording BOTH recipe steps use for the class
// neither of them can resolve.
//
// It is shared because the honest statement is identical at both steps: init cannot
// tell whether a never-adopted templating recipe left the consumer's own file alone or
// whether an adopted one's output is already in there. Naming the file and admitting
// that is the cost of an ambiguity core cannot close, and it deliberately uses NO "no
// gate was wired" language.
func indeterminatePreserveSentence(preserve ClassifiedPreserve) string {
	return fmt.Sprintf("%s was left in place by a one-shot recipe, and init CANNOT DETERMINE whether that recipe's output is present in it or whether the file is entirely your own. Open it and check; if the recipe's content is missing, run `backstop recipe apply %s` after moving your own version aside",
		preserve.Path, pinnedRecipeRefShape)
}

// recipeStepReport renders one recipe step's outcome, using the caller's per-class
// sentence renderer.
//
// WHAT IS PER-STEP IS THE SENTENCE ALONE. Class, gap-ness and exit consequence are
// identical across the two steps, which is why they share this renderer and differ
// only in the function they hand it.
//
// A USER-OWNED or INDETERMINATE preserve is a GAP even alongside successfully written
// files, and BOTH facts are reported in full — the writes are named and so is the gap.
// Only an apply whose every preserve is WAIVER-COVERED, or which preserves nothing at
// all, is reported as success.
func recipeStepReport(step string, outcome ApplyOutcome, classified []ClassifiedPreserve, sentence func(ClassifiedPreserve) string) StepReport {
	lines := []string{}
	if len(outcome.Written) > 0 {
		lines = append(lines, "wrote "+strings.Join(outcome.Written, ", "))
	}

	gap := false
	for _, preserve := range classified {
		if preserve.Class.IsGap() {
			gap = true
		}
		lines = append(lines, sentence(preserve))
	}

	if len(lines) == 0 {
		lines = append(lines, "the recipe applied and neither wrote nor preserved anything")
	}

	report := StepReport{Step: step, Outcome: OutcomeDelivered, Detail: strings.Join(lines, "; ")}
	if gap {
		report.Outcome = OutcomeBrokenPromise
	}
	return report
}
