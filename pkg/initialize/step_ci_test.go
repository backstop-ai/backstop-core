package initialize

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// noGateWiredSentence is the CI step's USER-OWNED assertion, quoted once so both the
// positive assertion here and the NEGATIVE assertion in the scaffold step's test are
// about the same string (Sharp Edge 19).
const noGateWiredSentence = "no backstop gate was wired into this file"

// pinnedRefShape is the shape init names when it tells a consumer how to do the step
// later. Naming the SHAPE rather than a concrete ref is the point: core holds no pack
// name, no recipe id and no version.
const pinnedRefShape = "<pack>:<recipe>@<version>"

// TestInit_CIRefIsPassedThroughVerbatim (SPEC-069 CLM-072).
//
// Byte-identical to what the consumer typed. No trimming, completing, defaulting or
// normalization: the whole string is OPAQUE to core, which is what makes a consumer
// naming an entirely different pack equally valid.
func TestInit_CIRefIsPassedThroughVerbatim(t *testing.T) {
	_, applier, _, _, _ := defaultFakes()
	ref := "some-org/some-ci-pack:some-recipe@1.4.2"

	stepCI("/project", ref, applier)

	if len(applier.calls) != 1 {
		t.Fatalf("the applier was called %d times, want exactly once", len(applier.calls))
	}
	if applier.calls[0].Ref != ref {
		t.Fatalf("the applier received %q, want the byte-identical %q", applier.calls[0].Ref, ref)
	}
}

// TestInit_CIOmittedAttemptsNoRecipeResolution (SPEC-069 CLM-074).
func TestInit_CIOmittedAttemptsNoRecipeResolution(t *testing.T) {
	_, applier, _, _, _ := defaultFakes()

	stepCI("/project", "", applier)

	if len(applier.calls) != 0 {
		t.Fatalf("the applier was called %d times with --ci omitted; NO resolution or apply may be attempted at all", len(applier.calls))
	}
}

// TestInit_CIOmittedReportsTheSkipAndHowToWireItLater (SPEC-069 CLM-075).
//
// A skipped optional step is not an error, but it is not silent either: the report
// states that no CI was wired and names `backstop recipe apply` plus the pinned ref
// SHAPE as the way to do it later.
func TestInit_CIOmittedReportsTheSkipAndHowToWireItLater(t *testing.T) {
	_, applier, _, _, _ := defaultFakes()

	report, preserves := stepCI("/project", "", applier)

	if report.Outcome != OutcomeSkipped {
		t.Fatalf("the omitted-CI step reported %v, want OutcomeSkipped — omission IS the opt-out, not a failure", report.Outcome)
	}
	if len(preserves) != 0 {
		t.Fatalf("the omitted-CI step produced %d preserves; it applied nothing", len(preserves))
	}
	if !strings.Contains(report.Detail, "backstop recipe apply") {
		t.Fatalf("the skip report does not name `backstop recipe apply`.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, pinnedRefShape) {
		t.Fatalf("the skip report does not name the pinned ref shape %s.\ngot: %s", pinnedRefShape, report.Detail)
	}
	if !strings.Contains(strings.ToLower(report.Detail), "no ci") {
		t.Fatalf("the skip report does not state that no CI was wired.\ngot: %s", report.Detail)
	}
}

// TestInit_ArbitraryPackNameInTheCIRefIsDispatchedNormally (SPEC-069 CLM-078).
//
// A ref naming a pack that appears NOWHERE in core is dispatched normally, BECAUSE
// core holds no allowlist of pack names. This is the falsifier for a baked roster: an
// implementation that recognized "the CI pack" would have to treat an unknown one
// differently.
func TestInit_ArbitraryPackNameInTheCIRefIsDispatchedNormally(t *testing.T) {
	familiar := "backstop-ai/ci-workflows:github-actions-gate@1.0.0"
	arbitrary := "entirely-unheard-of-org/nobody-has-this-pack:whatever-recipe@9.9.9"

	_, familiarApplier, _, _, _ := defaultFakes()
	familiarReport, _ := stepCI("/project", familiar, familiarApplier)

	_, arbitraryApplier, _, _, _ := defaultFakes()
	arbitraryReport, _ := stepCI("/project", arbitrary, arbitraryApplier)

	if len(arbitraryApplier.calls) != len(familiarApplier.calls) {
		t.Fatalf("an arbitrary pack name was dispatched %d times and a familiar-looking one %d times; core inspects no pack name",
			len(arbitraryApplier.calls), len(familiarApplier.calls))
	}
	if arbitraryReport.Outcome != familiarReport.Outcome {
		t.Fatalf("an arbitrary pack name produced outcome %v and a familiar-looking one %v; the two must be indistinguishable to core",
			arbitraryReport.Outcome, familiarReport.Outcome)
	}
}

// TestInit_UnpinnedCIRefSurfacesTheParseErrorVerbatim (SPEC-069 CLM-079).
//
// ParseRecipeRef accepts ONLY `<pack>:<recipe>@<version>` with a MANDATORY strict-
// semver pin — no "latest", no default version, no tolerance branch — so init performs
// no pin defaulting anywhere. The shipped error surfaces VERBATIM.
func TestInit_UnpinnedCIRefSurfacesTheParseErrorVerbatim(t *testing.T) {
	unpinned := "some-org/pack:some-recipe"

	// The expected text is the SHIPPED parser's own, obtained by calling it — not a
	// string this test invented and then asserted against itself.
	_, shipped := recipe.ParseRecipeRef(unpinned)
	if shipped == nil {
		t.Fatal("ParseRecipeRef accepted an unpinned ref; the fixture no longer exercises the claim")
	}

	_, applier, _, _, _ := defaultFakes()
	report, _ := stepCI("/project", unpinned, applier)

	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("an unpinned --ci reported %v, want OutcomeBrokenPromise", report.Outcome)
	}
	if !strings.Contains(report.Detail, shipped.Error()) {
		t.Fatalf("the report does not carry the shipped parse error VERBATIM.\nwant to contain: %s\ngot:             %s", shipped.Error(), report.Detail)
	}
	if len(applier.calls) != 0 {
		t.Fatal("an unpinned ref reached the applier; the parse failure must stop the step before any apply")
	}
}

// shippedResolveError produces a REAL error from the shipped resolve path for the
// named failure mode, so the surfacing claims assert against the message the binary
// actually emits rather than one this test made up.
func shippedResolveError(t *testing.T, mode string) (string, error) {
	t.Helper()

	packsDir := t.TempDir()
	ref, err := recipe.ParseRecipeRef("fixture/pack:the-recipe@1.0.0")
	if err != nil {
		t.Fatalf("the fixture ref does not parse: %v", err)
	}

	indexed := &pack.Manifest{
		Name: "fixture/pack", NormalizedName: "fixture/pack",
		Recipes: map[string]string{"the-recipe": "recipes/the-recipe"},
	}

	switch mode {
	case "uninstalled-pack":
		_, resolveErr := recipe.ResolveRecipe(ref, map[string]*pack.Manifest{}, packsDir)
		return "", resolveErr
	case "undeclared-recipe":
		bare := &pack.Manifest{Name: "fixture/pack", NormalizedName: "fixture/pack", Recipes: map[string]string{"a-different-recipe": "recipes/other"}}
		_, resolveErr := recipe.ResolveRecipe(ref, map[string]*pack.Manifest{"fixture/pack": bare}, packsDir)
		return "", resolveErr
	case "pin-mismatch":
		writeRecipeManifest(t, packsDir, "kind: templating\nversion: 2.0.0\nops:\n  - id: noop\n    kind: create\n    target: a.txt\n    payload: p.txt\n")
		_, resolveErr := recipe.ResolveRecipe(ref, map[string]*pack.Manifest{"fixture/pack": indexed}, packsDir)
		return "", resolveErr
	case "unparseable-manifest":
		writeRecipeManifest(t, packsDir, "this: [is not: valid yaml\n")
		_, resolveErr := recipe.ResolveRecipe(ref, map[string]*pack.Manifest{"fixture/pack": indexed}, packsDir)
		return "", resolveErr
	default:
		t.Fatalf("unknown resolve failure mode %q", mode)
		return "", nil
	}
}

// writeRecipeManifest lays a recipe.yml at the path the fixture pack's index names.
func writeRecipeManifest(t *testing.T, packsDir, body string) {
	t.Helper()
	dir := filepath.Join(packsDir, "recipes", "the-recipe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating the fixture recipe directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "recipe.yml"), []byte(body), 0o644); err != nil {
		t.Fatalf("writing the fixture recipe manifest: %v", err)
	}
}

// assertResolveFailureSurfacedVerbatim is the body the four resolve-failure claims
// share: the shipped error reaches the report VERBATIM, attributed to the CI step,
// with NO guidance of init's own bolted on.
func assertResolveFailureSurfacedVerbatim(t *testing.T, mode string) {
	t.Helper()

	_, shipped := shippedResolveError(t, mode)
	if shipped == nil {
		t.Fatalf("the %s fixture produced no error; it no longer exercises the claim", mode)
	}

	_, applier, _, _, _ := defaultFakes()
	ref := "fixture/pack:the-recipe@1.0.0"
	applier.failures[ref] = shipped

	report, preserves := stepCI("/project", ref, applier)

	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("a %s failure reported %v, want OutcomeBrokenPromise: CI was ASKED for and not delivered", mode, report.Outcome)
	}
	if !strings.Contains(report.Detail, shipped.Error()) {
		t.Fatalf("the %s failure's shipped error was not surfaced verbatim.\nwant to contain: %s\ngot:             %s", mode, shipped.Error(), report.Detail)
	}
	if report.Step != stepCIName {
		t.Fatalf("the failure was attributed to step %q, want %q", report.Step, stepCIName)
	}
	if len(preserves) != 0 {
		t.Fatalf("a failed apply produced %d preserves; a failure yields no verdict to classify", len(preserves))
	}
}

// TestInit_UninstalledPackInCIRefSurfacesTheResolveErrorVerbatim (SPEC-069 CLM-080).
func TestInit_UninstalledPackInCIRefSurfacesTheResolveErrorVerbatim(t *testing.T) {
	assertResolveFailureSurfacedVerbatim(t, "uninstalled-pack")
}

// TestInit_UndeclaredRecipeInCIRefSurfacesTheResolveErrorVerbatim (SPEC-069 CLM-081).
func TestInit_UndeclaredRecipeInCIRefSurfacesTheResolveErrorVerbatim(t *testing.T) {
	assertResolveFailureSurfacedVerbatim(t, "undeclared-recipe")
}

// TestInit_PinMismatchInCIRefSurfacesTheResolveErrorVerbatim (SPEC-069 CLM-082).
func TestInit_PinMismatchInCIRefSurfacesTheResolveErrorVerbatim(t *testing.T) {
	assertResolveFailureSurfacedVerbatim(t, "pin-mismatch")
}

// TestInit_UnparseableRecipeManifestSurfacesTheResolveErrorVerbatim (SPEC-069
// CLM-102).
//
// This one is a PACK DEFECT rather than a bad consumer ref, and the claim is that init
// treats it IDENTICALLY: it classifies no failure mode differently, because any
// classification would be init adding a diagnosis the shipped error already carries.
func TestInit_UnparseableRecipeManifestSurfacesTheResolveErrorVerbatim(t *testing.T) {
	assertResolveFailureSurfacedVerbatim(t, "unparseable-manifest")
}

// TestInit_EveryOtherStepCompletesWhenTheCIStepFails (SPEC-069 CLM-083).
//
// The CI step fails LOUDLY and LOCALLY: it returns a report, never an abort, so the
// runner's remaining steps still execute and are still reported with their own
// outcomes.
func TestInit_EveryOtherStepCompletesWhenTheCIStepFails(t *testing.T) {
	installer, applier, gates, tools, seeds := defaultFakes()
	ref := "fixture/pack:the-recipe@1.0.0"
	applier.failures[ref] = errors.New("the shipped resolve refused this ref")

	runner := newTestRunner(t, installer, applier, gates, tools, seeds)
	root := t.TempDir()

	result, err := runner.Run(Options{
		ProjectRoot:  root,
		Capabilities: allCapabilities(t),
		CIRecipeRef:  ref,
	})
	if err != nil {
		t.Fatalf("a CI failure aborted the whole run: %v", err)
	}

	failing := requireStep(t, result.Steps, stepCIName)
	if failing.Outcome != OutcomeBrokenPromise {
		t.Fatalf("the CI step reported %v, want OutcomeBrokenPromise", failing.Outcome)
	}

	// Every OTHER step still ran and still carries its own outcome.
	for _, other := range []string{stepGitName, stepConfigName, stepLayoutName, stepPacksName, stepGitignoreName, stepBaselineName, stepObserveName} {
		step := requireStep(t, result.Steps, other)
		if step.Outcome == OutcomeBrokenPromise {
			t.Fatalf("step %q was dragged into failure by the CI step: %s", other, step.Detail)
		}
	}
	if gates.calls != 1 {
		t.Fatalf("the observe step ran %d times after a CI failure, want once — every other step still completes", gates.calls)
	}
}

// preserveOutcome scripts one apply outcome carrying the supplied preserves at the
// supplied declared kind.
func preserveOutcome(kind string, written []string, preserved ...recipe.PreservedDivergence) ApplyOutcome {
	return ApplyOutcome{Written: written, Preserved: preserved, RecipeKind: kind}
}

// runCIWithOutcome runs the CI step against a scripted apply outcome.
func runCIWithOutcome(t *testing.T, outcome ApplyOutcome) (StepReport, []ClassifiedPreserve) {
	t.Helper()
	_, applier, _, _, _ := defaultFakes()
	ref := "fixture/pack:the-recipe@1.0.0"
	applier.outcomes[ref] = outcome
	return stepCI("/project", ref, applier)
}

// TestInit_UserOwnedPreserveNamesEveryPreservedFile (SPEC-069 CLM-096).
func TestInit_UserOwnedPreserveNamesEveryPreservedFile(t *testing.T) {
	report, classified := runCIWithOutcome(t, preserveOutcome(recipe.KindScaffolding, nil,
		recipe.PreservedDivergence{Path: ".gitlab-ci.yml"},
		recipe.PreservedDivergence{Path: "Jenkinsfile"},
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
}

// TestInit_UserOwnedPreserveStatesNoGateWasWired (SPEC-069 CLM-097).
//
// ★ This sentence belongs to this class at THIS step alone. It is an assertion about
// CI wiring, and it is true here precisely because the never-clobber protection left
// the consumer's own file in place with no gate in it.
func TestInit_UserOwnedPreserveStatesNoGateWasWired(t *testing.T) {
	report, _ := runCIWithOutcome(t, preserveOutcome(recipe.KindScaffolding, nil,
		recipe.PreservedDivergence{Path: ".gitlab-ci.yml"},
	))

	if !strings.Contains(strings.ToLower(report.Detail), noGateWiredSentence) {
		t.Fatalf("the user-owned CI report does not state in words that no gate was wired.\nwant to contain: %q\ngot: %s", noGateWiredSentence, report.Detail)
	}
}

// TestInit_UserOwnedPreserveGivesTheConsumerANextAction (SPEC-069 CLM-098).
func TestInit_UserOwnedPreserveGivesTheConsumerANextAction(t *testing.T) {
	report, _ := runCIWithOutcome(t, preserveOutcome(recipe.KindImplementing, nil,
		recipe.PreservedDivergence{Path: ".gitlab-ci.yml"},
	))

	if !strings.Contains(report.Detail, "backstop recipe apply") {
		t.Fatalf("the user-owned CI report gives the consumer no next action.\ngot: %s", report.Detail)
	}
}

// TestInit_TemplatingKindEmptyPairPreserveIsReportedAsIndeterminateGap (SPEC-069
// CLM-113).
//
// ★ It names the file and states init CANNOT DETERMINE whether the recipe's output is
// present, with a next action — and uses NO "no gate was wired" language, because that
// is the half init cannot know. Asserting it either way would be a false report; this
// is the honest one.
func TestInit_TemplatingKindEmptyPairPreserveIsReportedAsIndeterminateGap(t *testing.T) {
	report, classified := runCIWithOutcome(t, preserveOutcome(recipe.KindTemplating, nil,
		recipe.PreservedDivergence{Path: ".github/workflows/gate.yml"},
	))

	if len(classified) != 1 || classified[0].Class != PreserveIndeterminate {
		t.Fatalf("the templating empty-pair preserve classified %v, want PreserveIndeterminate", classified)
	}
	if !strings.Contains(report.Detail, ".github/workflows/gate.yml") {
		t.Fatalf("the indeterminate report does not name the file.\ngot: %s", report.Detail)
	}
	detail := strings.ToLower(report.Detail)
	if !strings.Contains(detail, "cannot determine") {
		t.Fatalf("the indeterminate report does not state that init cannot determine whether the recipe's output is present.\ngot: %s", report.Detail)
	}
	if strings.Contains(detail, noGateWiredSentence) {
		t.Fatalf("the indeterminate report asserts that no gate was wired; that is exactly the half init cannot know, and asserting it is a false positive about a one-shot that already materialized.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, "backstop recipe apply") {
		t.Fatalf("the indeterminate report gives no next action.\ngot: %s", report.Detail)
	}
}

// TestInit_WaiverCoveredPreserveNamesTheRuleAndCoveringToken (SPEC-069 CLM-115).
//
// ★ Accountable customization: the gate IS wired and the consumer accounted for the
// divergence with a valid token, so init names both and reports NO gap.
func TestInit_WaiverCoveredPreserveNamesTheRuleAndCoveringToken(t *testing.T) {
	rule := "ci-workflows/gate-job-present"
	covering := coveringWaiverFixture(rule, "accepted-risk", "2027-03-01")

	report, classified := runCIWithOutcome(t, preserveOutcome(recipe.KindScaffolding, nil,
		recipe.PreservedDivergence{Path: ".github/workflows/gate.yml", Rule: rule, CoveringWaiver: covering},
	))

	if len(classified) != 1 || classified[0].Class != PreserveWaiverCovered {
		t.Fatalf("the populated-pair preserve classified %v, want PreserveWaiverCovered", classified)
	}
	if !strings.Contains(report.Detail, rule) {
		t.Fatalf("the waiver-covered report does not name the rule %q.\ngot: %s", rule, report.Detail)
	}
	if !strings.Contains(report.Detail, covering) {
		t.Fatalf("the waiver-covered report does not name the covering waiver %q.\ngot: %s", covering, report.Detail)
	}
	if strings.Contains(strings.ToLower(report.Detail), noGateWiredSentence) {
		t.Fatalf("the waiver-covered report claims no gate was wired; the gate demonstrably IS wired and the customization is accounted for.\ngot: %s", report.Detail)
	}
	if report.Outcome == OutcomeBrokenPromise {
		t.Fatalf("a waiver-covered preserve failed the step; the accountable class is the one preserve class that leaves an apply successful.\ngot: %s", report.Detail)
	}
}

// TestInit_PartialApplyWithAUserOwnedPreserveIsReportedAsAGap (SPEC-069 CLM-100).
//
// A USER-OWNED preserve occurring ALONGSIDE successfully written files is still a gap.
// Both facts are reported in full: the writes are named AND the gap is named.
func TestInit_PartialApplyWithAUserOwnedPreserveIsReportedAsAGap(t *testing.T) {
	report, _ := runCIWithOutcome(t, preserveOutcome(recipe.KindScaffolding,
		[]string{".github/workflows/gate.yml"},
		recipe.PreservedDivergence{Path: ".gitlab-ci.yml"},
	))

	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("a partial apply with a user-owned preserve reported %v, want OutcomeBrokenPromise", report.Outcome)
	}
	if !strings.Contains(report.Detail, ".github/workflows/gate.yml") {
		t.Fatalf("the report drops the file that WAS written; both facts are reported in full.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, ".gitlab-ci.yml") {
		t.Fatalf("the report drops the preserved file.\ngot: %s", report.Detail)
	}
}

// TestInit_ApplyWithNoPreservesIsReportedAsSuccess (SPEC-069 CLM-101).
func TestInit_ApplyWithNoPreservesIsReportedAsSuccess(t *testing.T) {
	report, classified := runCIWithOutcome(t, preserveOutcome(recipe.KindScaffolding,
		[]string{".github/workflows/gate.yml"},
	))

	if len(classified) != 0 {
		t.Fatalf("an apply that preserved nothing produced %d classified preserves", len(classified))
	}
	if report.Outcome != OutcomeDelivered {
		t.Fatalf("an apply with no preserves reported %v, want OutcomeDelivered (%s)", report.Outcome, report.Detail)
	}
	if !strings.Contains(report.Detail, ".github/workflows/gate.yml") {
		t.Fatalf("the success report does not name what was written.\ngot: %s", report.Detail)
	}
}

// coveringWaiverFixture builds a covering-waiver value from its parts.
//
// It is ASSEMBLED rather than written as a source literal, and that is not cosmetic:
// the gate's waiver-resolution pass BYTE-SCANS source lines for the waiver grammar, so
// a literal token here would be read as a REAL waiver this test file was claiming — and
// because the literal ends at a closing quote, its expiry does not parse, which
// surfaces as a MALFORMED-WAIVER gate failure about a string that was only ever test
// data. The applier hands this value through opaquely, so its exact spelling is
// irrelevant to what is under test; only that init NAMES it is.
func coveringWaiverFixture(rule, reason, expiry string) string {
	return "@" + "waiver:" + rule + ":" + reason + ":" + expiry
}

// TestInit_WhitespaceOnlyRefIsSuppliedNotOmitted closes the whitespace-shaped hole in
// the exit-code matrix, for BOTH flag-governed steps.
//
// `--ci "   "` is a ref the consumer SUPPLIED. Treating it as an omission would collapse
// the pair the whole matrix is organized around — omitted is a deliberate no-op exiting
// 0, supplied-and-unresolvable is a broken promise exiting non-zero — and it would do so
// silently, because the report would read exactly like an honest skip.
//
// The step must therefore hand it to ParseRecipeRef like any other malformed ref and
// surface that error verbatim.
func TestInit_WhitespaceOnlyRefIsSuppliedNotOmitted(t *testing.T) {
	// The shipped parser's own verdict on a blank ref, obtained by calling it rather
	// than assumed.
	_, shipped := recipe.ParseRecipeRef("   ")
	if shipped == nil {
		t.Fatal("ParseRecipeRef accepted a whitespace-only ref; this claim no longer has a failure mode to surface")
	}

	steps := map[string]func(string, string, RecipeApplier) (StepReport, []ClassifiedPreserve){
		stepCIName:       stepCI,
		stepScaffoldName: stepScaffold,
	}

	for name, step := range steps {
		t.Run(name, func(t *testing.T) {
			_, applier, _, _, _ := defaultFakes()

			report, _ := step("/project", "   ", applier)

			if report.Outcome == OutcomeSkipped {
				t.Fatalf("the %s step reported a whitespace-only ref as SKIPPED. The consumer supplied that flag; reclassifying it as an omission exits 0 on a promise init did not keep, and the report reads exactly like an honest skip.\ngot: %s",
					name, report.Detail)
			}
			if report.Outcome != OutcomeBrokenPromise {
				t.Fatalf("the %s step reported %v for a whitespace-only ref, want OutcomeBrokenPromise", name, report.Outcome)
			}
			if !strings.Contains(report.Detail, shipped.Error()) {
				t.Fatalf("the %s step did not surface the shipped parse error verbatim.\nwant to contain: %s\ngot:             %s",
					name, shipped.Error(), report.Detail)
			}
			if len(applier.calls) != 0 {
				t.Fatalf("the %s step reached the applier with an unparseable ref", name)
			}
		})
	}

	// And the genuinely-omitted case still skips, so the fix did not simply delete the
	// skip path.
	for name, step := range steps {
		_, applier, _, _, _ := defaultFakes()
		if report, _ := step("/project", "", applier); report.Outcome != OutcomeSkipped {
			t.Fatalf("the %s step no longer reports a genuinely omitted flag as a skip: %v", name, report.Outcome)
		}
	}
}
