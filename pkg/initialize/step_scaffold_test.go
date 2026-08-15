package initialize

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// runScaffoldWithOutcome runs the scaffold step against a scripted apply outcome.
func runScaffoldWithOutcome(t *testing.T, outcome ApplyOutcome) (StepReport, []ClassifiedPreserve) {
	t.Helper()
	_, applier, _, _, _ := defaultFakes()
	ref := "fixture/pack:the-scaffold@1.0.0"
	applier.outcomes[ref] = outcome
	return stepScaffold("/project", ref, applier)
}

// TestInit_ScaffoldRefIsPassedThroughVerbatim (SPEC-069 CLM-126).
//
// Byte-identical, exactly as CLM-072 requires of `--ci`. `--scaffold` follows `--ci`'s
// governance shape precisely so there is one posture to reason about, not two.
func TestInit_ScaffoldRefIsPassedThroughVerbatim(t *testing.T) {
	_, applier, _, _, _ := defaultFakes()
	ref := "some-org/some-scaffold-pack:first-source@2.0.1"

	stepScaffold("/project", ref, applier)

	if len(applier.calls) != 1 {
		t.Fatalf("the applier was called %d times, want exactly once", len(applier.calls))
	}
	if applier.calls[0].Ref != ref {
		t.Fatalf("the applier received %q, want the byte-identical %q", applier.calls[0].Ref, ref)
	}
}

// TestInit_ScaffoldOmittedAttemptsNoRecipeResolution (SPEC-069 CLM-127).
func TestInit_ScaffoldOmittedAttemptsNoRecipeResolution(t *testing.T) {
	_, applier, _, _, _ := defaultFakes()

	stepScaffold("/project", "", applier)

	if len(applier.calls) != 0 {
		t.Fatalf("the applier was called %d times with --scaffold omitted; no resolution or apply may be attempted", len(applier.calls))
	}
}

// TestInit_ScaffoldOmittedReportsTheSkipAndHowToDoItLater (SPEC-069 CLM-128).
//
// Not every pack ecosystem ships a scaffold recipe, so a skipped optional step is not
// an error. DD-7's failure mode does not go unreported when this step is skipped — it
// arrives at the toolchain step as REQ-011's own case (b)/(c) report, which prints what
// the entrypoint did and diagnoses nothing.
func TestInit_ScaffoldOmittedReportsTheSkipAndHowToDoItLater(t *testing.T) {
	_, applier, _, _, _ := defaultFakes()

	report, preserves := stepScaffold("/project", "", applier)

	if report.Outcome != OutcomeSkipped {
		t.Fatalf("the omitted-scaffold step reported %v, want OutcomeSkipped", report.Outcome)
	}
	if len(preserves) != 0 {
		t.Fatalf("the omitted-scaffold step produced %d preserves", len(preserves))
	}
	if !strings.Contains(strings.ToLower(report.Detail), "no source file was scaffolded") {
		t.Fatalf("the skip report does not state that no source file was scaffolded.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, "backstop recipe apply") {
		t.Fatalf("the skip report does not name `backstop recipe apply`.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, pinnedRefShape) {
		t.Fatalf("the skip report does not name the pinned ref shape %s.\ngot: %s", pinnedRefShape, report.Detail)
	}
}

// TestInit_UnresolvableScaffoldRefSurfacesTheResolveErrorVerbatim (SPEC-069 CLM-134).
//
// The shipped error VERBATIM, attributed to the SCAFFOLD step. Init adds no guidance
// and classifies no resolve failure differently — REQ-017's posture, applied to this
// step.
func TestInit_UnresolvableScaffoldRefSurfacesTheResolveErrorVerbatim(t *testing.T) {
	_, shipped := shippedResolveError(t, "uninstalled-pack")
	if shipped == nil {
		t.Fatal("the uninstalled-pack fixture produced no error")
	}

	_, applier, _, _, _ := defaultFakes()
	ref := "fixture/pack:the-recipe@1.0.0"
	applier.failures[ref] = shipped

	report, preserves := stepScaffold("/project", ref, applier)

	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("an unresolvable --scaffold reported %v, want OutcomeBrokenPromise: the consumer asked for a source file and init did not deliver one", report.Outcome)
	}
	if report.Step != stepScaffoldName {
		t.Fatalf("the failure was attributed to step %q, want %q", report.Step, stepScaffoldName)
	}
	if !strings.Contains(report.Detail, shipped.Error()) {
		t.Fatalf("the shipped error was not surfaced verbatim.\nwant to contain: %s\ngot:             %s", shipped.Error(), report.Detail)
	}
	if len(preserves) != 0 {
		t.Fatalf("a failed apply produced %d preserves", len(preserves))
	}
}

// TestInit_ArbitraryPackNameInTheScaffoldRefIsDispatchedNormally (SPEC-069 CLM-137).
//
// Core holds no default scaffold recipe and no scaffold-pack roster, so a consumer
// naming an entirely different pack is equally valid — precisely because core never
// inspects the ref.
func TestInit_ArbitraryPackNameInTheScaffoldRefIsDispatchedNormally(t *testing.T) {
	plausible := "backstop-ai/some-toolchain:first-source@1.0.0"
	arbitrary := "nobody-has-ever-published-this/pack:first-source@1.0.0"

	_, plausibleApplier, _, _, _ := defaultFakes()
	plausibleReport, _ := stepScaffold("/project", plausible, plausibleApplier)

	_, arbitraryApplier, _, _, _ := defaultFakes()
	arbitraryReport, _ := stepScaffold("/project", arbitrary, arbitraryApplier)

	if len(arbitraryApplier.calls) != len(plausibleApplier.calls) {
		t.Fatalf("an arbitrary pack name was dispatched %d times and a plausible one %d times; core inspects no pack name",
			len(arbitraryApplier.calls), len(plausibleApplier.calls))
	}
	if arbitraryReport.Outcome != plausibleReport.Outcome {
		t.Fatalf("an arbitrary pack name produced outcome %v and a plausible one %v", arbitraryReport.Outcome, plausibleReport.Outcome)
	}
}

// TestInit_ScaffoldUserOwnedPreserveNamesTheFileAndSaysNoSourceFileWasWritten
// (SPEC-069 CLM-140).
//
// ★ THE SCAFFOLD STEP'S USER-OWNED SENTENCE IS ITS OWN (Sharp Edge 19). The assertion
// is made in BOTH directions on purpose: positively, that the report states the
// consumer's own file was left in place and the recipe's declared source file was
// therefore NOT written; and negatively, that the CI step's "no backstop gate was
// wired" sentence does NOT appear. This step knows nothing about CI wiring, and
// CLM-144's one-classifier requirement makes sharing the STRING look like the same
// kind of de-duplication. It is not.
func TestInit_ScaffoldUserOwnedPreserveNamesTheFileAndSaysNoSourceFileWasWritten(t *testing.T) {
	report, classified := runScaffoldWithOutcome(t, preserveOutcome(recipe.KindScaffolding, nil,
		recipe.PreservedDivergence{Path: "src/main.entry"},
		recipe.PreservedDivergence{Path: "src/second.entry"},
	))

	if len(classified) != 2 {
		t.Fatalf("the step classified %d preserves, want 2", len(classified))
	}
	for _, preserve := range classified {
		if preserve.Class != PreserveUserOwned {
			t.Fatalf("%s classified %v, want PreserveUserOwned", preserve.Path, preserve.Class)
		}
		if !strings.Contains(report.Detail, preserve.Path) {
			t.Fatalf("the report does not NAME the preserved file %s.\ngot: %s", preserve.Path, report.Detail)
		}
	}

	detail := strings.ToLower(report.Detail)
	if !strings.Contains(detail, "left in place") {
		t.Fatalf("the scaffold report does not state that the consumer's own file was left in place.\ngot: %s", report.Detail)
	}
	if !strings.Contains(detail, "was not written") {
		t.Fatalf("the scaffold report does not state that the recipe's declared source file was NOT written.\ngot: %s", report.Detail)
	}
	if strings.Contains(detail, noGateWiredSentence) {
		t.Fatalf("the scaffold report borrowed the CI step's %q sentence. This step knows nothing about CI wiring, so that sentence here asserts something init cannot know about a file it was never asked to wire a gate into.\ngot: %s",
			noGateWiredSentence, report.Detail)
	}
	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("a user-owned scaffold preserve reported %v, want OutcomeBrokenPromise", report.Outcome)
	}
}

// TestInit_ScaffoldStepUsesTheSameClassifierAsTheCIStep asserts the two steps agree on
// the CLASS while differing on the SENTENCE. Additive, and it is the behavioral twin
// of CLM-144's structural count: one classifier means identical classes, and a
// per-step classifier would show up here as a disagreement.
func TestInit_ScaffoldStepUsesTheSameClassifierAsTheCIStep(t *testing.T) {
	cases := []struct {
		kind       string
		divergence recipe.PreservedDivergence
	}{
		{recipe.KindScaffolding, recipe.PreservedDivergence{Path: "p"}},
		{recipe.KindImplementing, recipe.PreservedDivergence{Path: "p"}},
		{recipe.KindTemplating, recipe.PreservedDivergence{Path: "p"}},
		{recipe.KindTemplating, recipe.PreservedDivergence{Path: "p", Rule: "r", CoveringWaiver: "w"}},
	}

	for _, tc := range cases {
		outcome := preserveOutcome(tc.kind, nil, tc.divergence)

		ciReport, ciClassified := runCIWithOutcome(t, outcome)
		scaffoldReport, scaffoldClassified := runScaffoldWithOutcome(t, outcome)

		if ciClassified[0].Class != scaffoldClassified[0].Class {
			t.Fatalf("kind=%s pair=%v: the CI step classified %v and the scaffold step %v; both must run through the ONE shared classifier",
				tc.kind, tc.divergence, ciClassified[0].Class, scaffoldClassified[0].Class)
		}
		if ciReport.Outcome != scaffoldReport.Outcome {
			t.Fatalf("kind=%s: the two steps disagreed on the outcome (%v vs %v); gap-ness is identical across them, only the SENTENCE differs",
				tc.kind, ciReport.Outcome, scaffoldReport.Outcome)
		}
	}
}
