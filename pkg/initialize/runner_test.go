package initialize

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// runBare runs a Runner over the inert fakes in a fresh temp project, returning the
// result and the fakes so a test can assert on what each seam saw.
func runBare(t *testing.T, capabilities map[Capability]bool) (Result, string, *fakePackInstaller, *fakeRecipeApplier, *fakeGateRunner, *fakeToolchainProber, *unavailableSeeder) {
	t.Helper()
	installer, applier, gates, tools, seeds := defaultFakes()
	runner := newTestRunner(t, installer, applier, gates, tools, seeds)
	root := t.TempDir()

	result, err := runner.Run(Options{ProjectRoot: root, Capabilities: capabilities})
	if err != nil {
		t.Fatalf("the run errored: %v", err)
	}
	return result, root, installer, applier, gates, tools, seeds
}

// capabilityStepNames maps each capability to the report name its step produces, so a
// subtraction claim can assert on the STEP rather than on the flag it removed. It is
// built fresh per call: a shared package-level map is state one test could mutate for
// every other.
func capabilityStepNames() map[Capability]string {
	return map[Capability]string{
		CapabilityGit:       stepGitName,
		CapabilitySdlc:      stepLayoutName,
		CapabilityGitignore: stepGitignoreName,
		CapabilityPacks:     stepPacksName,
		CapabilityToolchain: stepToolchainName,
		CapabilityBaseline:  stepBaselineName,
		CapabilityObserve:   stepObserveName,
	}
}

// TestInit_BareRunResolvesAllSevenDefaultCapabilities (SPEC-069 CLM-003).
//
// Bare init resolves EXACTLY the seven default capabilities and RUNS all seven — the
// resolution and the execution are both asserted, because a set that resolved seven
// and ran five would satisfy a resolution-only test.
func TestInit_BareRunResolvesAllSevenDefaultCapabilities(t *testing.T) {
	result, _, _, _, _, _, _ := runBare(t, allCapabilities(t))

	for capability, step := range capabilityStepNames() {
		if _, ran := findStep(result.Steps, step); !ran {
			t.Fatalf("the %s capability resolved but its %q step did not run.\nran: %v", capability, step, stepNames(result.Steps))
		}
	}
	if _, ran := findStep(result.Steps, stepConfigName); !ran {
		t.Fatal("the unconditional config step did not run")
	}
}

// assertSubtractsOnly is the body the seven one-capability subtraction claims share.
//
// They are SEVEN claims and seven tests rather than one table, deliberately: a matrix
// asserted in aggregate hides a missing member, and each flag is separately
// load-bearing.
func assertSubtractsOnly(t *testing.T, subtracted Capability) {
	t.Helper()
	result, _, _, _, _, _, _ := runBare(t, capabilitiesExcept(t, string(subtracted)))

	removed := capabilityStepNames()[subtracted]
	if _, ran := findStep(result.Steps, removed); ran {
		t.Fatalf("--no-%s left its %q step running.\nran: %v", subtracted, removed, stepNames(result.Steps))
	}

	for capability, step := range capabilityStepNames() {
		if capability == subtracted {
			continue
		}
		if _, ran := findStep(result.Steps, step); !ran {
			t.Fatalf("--no-%s ALSO removed the %s capability's %q step; subtraction removes exactly one.\nran: %v",
				subtracted, capability, step, stepNames(result.Steps))
		}
	}
}

// TestInit_NoGitSubtractsOnlyTheGitCapability (SPEC-069 CLM-004).
func TestInit_NoGitSubtractsOnlyTheGitCapability(t *testing.T) {
	assertSubtractsOnly(t, CapabilityGit)
}

// TestInit_NoSdlcSubtractsOnlyTheSdlcCapability (SPEC-069 CLM-005).
func TestInit_NoSdlcSubtractsOnlyTheSdlcCapability(t *testing.T) {
	assertSubtractsOnly(t, CapabilitySdlc)
}

// TestInit_NoGitignoreSubtractsOnlyTheGitignoreCapability (SPEC-069 CLM-006).
func TestInit_NoGitignoreSubtractsOnlyTheGitignoreCapability(t *testing.T) {
	assertSubtractsOnly(t, CapabilityGitignore)
}

// TestInit_NoPacksSubtractsOnlyThePacksCapability (SPEC-069 CLM-007).
func TestInit_NoPacksSubtractsOnlyThePacksCapability(t *testing.T) {
	assertSubtractsOnly(t, CapabilityPacks)
}

// TestInit_NoToolchainSubtractsOnlyTheToolchainCapability (SPEC-069 CLM-008).
func TestInit_NoToolchainSubtractsOnlyTheToolchainCapability(t *testing.T) {
	assertSubtractsOnly(t, CapabilityToolchain)
}

// TestInit_NoBaselineSubtractsOnlyTheBaselineCapability (SPEC-069 CLM-009).
func TestInit_NoBaselineSubtractsOnlyTheBaselineCapability(t *testing.T) {
	assertSubtractsOnly(t, CapabilityBaseline)
}

// TestInit_NoObserveSubtractsOnlyTheObserveCapability (SPEC-069 CLM-010).
func TestInit_NoObserveSubtractsOnlyTheObserveCapability(t *testing.T) {
	assertSubtractsOnly(t, CapabilityObserve)
}

// TestInit_ReportNamesEveryResolvedCapabilityWithAnOutcome (SPEC-069 CLM-002).
//
// Every capability in the RESOLVED set appears with an outcome, so no step is silently
// absent from the account of what init did. A step that ran and reported nothing is
// indistinguishable, to the consumer, from one that never ran.
func TestInit_ReportNamesEveryResolvedCapabilityWithAnOutcome(t *testing.T) {
	for _, capabilities := range []map[Capability]bool{
		allCapabilities(t),
		capabilitiesExcept(t, "sdlc"),
		capabilitiesOnly(t, "git", "packs"),
	} {
		result, _, _, _, _, _, _ := runBare(t, capabilities)

		for capability := range capabilities {
			step := capabilityStepNames()[capability]
			report, ran := findStep(result.Steps, step)
			if !ran {
				t.Fatalf("resolved capability %s produced no report.\nran: %v", capability, stepNames(result.Steps))
			}
			if strings.TrimSpace(report.Detail) == "" {
				t.Fatalf("the %q step reported an outcome with no detail; a consumer cannot act on a bare status", step)
			}
		}
	}
}

// TestInit_ExecutesTheTranscribedStepSequenceInOrder (SPEC-069 CLM-090).
//
// git, config, layout, packs, gitignore, scaffold, toolchain, baseline, ci, observe.
// Subtracting a capability removes its step and REORDERS NOTHING.
func TestInit_ExecutesTheTranscribedStepSequenceInOrder(t *testing.T) {
	installer, applier, gates, tools, seeds := defaultFakes()
	// A toolchain report has to come from the prober, since the prober IS the
	// toolchain step in production.
	tools.reports = []StepReport{{Step: stepToolchainName, Outcome: OutcomeDelivered, Detail: "ran one declared entrypoint"}}
	runner := newTestRunner(t, installer, applier, gates, tools, seeds)

	result, err := runner.Run(Options{
		ProjectRoot:       t.TempDir(),
		Capabilities:      allCapabilities(t),
		PackRefs:          []string{"acme/pack@1.0.0"},
		ScaffoldRecipeRef: "acme/pack:first-source@1.0.0",
		CIRecipeRef:       "acme/pack:ci@1.0.0",
	})
	if err != nil {
		t.Fatalf("the run errored: %v", err)
	}

	want := []string{
		stepGitName, stepConfigName, stepLayoutName, stepPacksName, stepGitignoreName,
		stepScaffoldName, stepToolchainName, stepBaselineName, stepCIName, stepObserveName,
	}
	got := stepNames(result.Steps)
	if len(got) != len(want) {
		t.Fatalf("the run produced %d step reports, want %d.\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("step %d was %q, want %q.\ngot:  %v\nwant: %v", i, got[i], want[i], got, want)
		}
	}

	// Subtraction removes and never reorders.
	narrowed, err := runner.Run(Options{
		ProjectRoot:  t.TempDir(),
		Capabilities: capabilitiesExcept(t, "git", "baseline"),
	})
	if err != nil {
		t.Fatalf("the narrowed run errored: %v", err)
	}
	narrowedNames := stepNames(narrowed.Steps)
	previous := -1
	for _, name := range narrowedNames {
		position := indexOfName(want, name)
		if position <= previous {
			t.Fatalf("subtraction reordered the sequence: %v is not a subsequence of %v", narrowedNames, want)
		}
		previous = position
	}
}

// indexOfName returns the position of name in names, or -1.
func indexOfName(names []string, name string) int {
	for i, candidate := range names {
		if candidate == name {
			return i
		}
	}
	return -1
}

// TestInit_GitignoreCarriesPackEntriesBecausePacksInstallFirst (SPEC-069 CLM-111).
//
// ★ DEVIATION 1, ASSERTED BY CONSEQUENCE. A `--pack` run on a fresh repo emits a
// `.gitignore` that ALREADY CARRIES the installed pack's declared stdout_artifact
// entries AT THE MOMENT the gitignore step completes. The same run with the two steps
// swapped would emit a file missing every pack-derived entry — the cross-repo ignore
// divergence this ordering exists to end.
//
// The assertion is on the FILE CONTENT at that boundary, not on a list of step names:
// a name list is satisfied by the very swap this claim exists to catch.
func TestInit_GitignoreCarriesPackEntriesBecausePacksInstallFirst(t *testing.T) {
	root := t.TempDir()
	installer, applier, gates, tools, seeds := defaultFakes()

	// The fake installer MATERIALIZES a pack the way the real one does — a manifest
	// under .backstop/packs/ plus the backstop.yml entry — so the gitignore step has
	// something real to read. Without that the claim would assert nothing about
	// ordering, only that the fake was called.
	installer.onInstall = func(projectRoot, ref string) {
		writeFile(t, projectRoot, "backstop.yml", "project: ordering-fixture\npacks:\n  fixture/ordered: 1.0.0\n")
		writeFile(t, projectRoot, ".backstop/packs/fixture/ordered/pack.yml", `
name: fixture/ordered
version: 1.0.0
language: neutral
archetype: enforcement
description: A pack whose engine declares a stdout_artifact, so the gitignore step has a pack-derived entry to carry.
engines:
  writes-an-artifact:
    command: grep -rn
    input_mode: none
    scope_kind: project-wide
    gate_type: test
    stdout_artifact: build/ordering-proof.json
content:
  ruleset:
    version: 1.0.0
    rules: []
`)
	}

	runner := newTestRunner(t, installer, applier, gates, tools, seeds)
	result, err := runner.Run(Options{
		ProjectRoot:  root,
		Capabilities: capabilitiesOnly(t, "packs", "gitignore"),
		PackRefs:     []string{"fixture/ordered@1.0.0"},
	})
	if err != nil {
		t.Fatalf("the run errored: %v", err)
	}

	gitignore := requireStep(t, result.Steps, stepGitignoreName)
	if gitignore.Outcome == OutcomeBrokenPromise {
		t.Fatalf("the gitignore step failed: %s", gitignore.Detail)
	}

	body := readFile(t, root, ".gitignore")
	if !strings.Contains(body, "build/ordering-proof.json") {
		t.Fatalf("the .gitignore does not carry the installed pack's declared stdout_artifact. The gitignore step ran BEFORE the pack was installed, so the entry set was computed over an empty pack corpus.\n---\n%s", body)
	}
}

// TestInit_ScaffoldedFileIsOnDiskBeforeTheToolchainStepRuns (SPEC-069 CLM-138).
//
// ★ DEVIATION 2, AND THE GUARD AGAINST THE OBVIOUS REFACTOR. With both `--pack` and
// `--scaffold` supplied, the scaffold recipe's declared file is ALREADY ON DISK when
// the toolchain step executes its first entrypoint.
//
// The fake prober STATS THE FILE AT PROBE TIME, which is the only way to observe the
// state at that boundary. Asserting the step-name list alone is satisfiable vacuously
// by exactly the refactor this claim exists to catch — moving `scaffold` down beside
// `ci` reads as a tidy-up and keeps every name-list assertion passing while
// manufacturing the empty-project failure the step exists to prevent.
func TestInit_ScaffoldedFileIsOnDiskBeforeTheToolchainStepRuns(t *testing.T) {
	root := t.TempDir()
	installer, applier, gates, tools, seeds := defaultFakes()

	scaffoldRef := "fixture/pack:first-source@1.0.0"
	scaffoldTarget := "src/first-source.txt"
	applier.outcomes[scaffoldRef] = ApplyOutcome{Written: []string{scaffoldTarget}, RecipeKind: recipe.KindScaffolding}
	applier.onApply = func(projectRoot, ref string) {
		if ref == scaffoldRef {
			writeFile(t, projectRoot, scaffoldTarget, "the recipe's declared source file\n")
		}
	}

	onDiskAtProbeTime := false
	tools.onProbe = func(projectRoot string) {
		_, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(scaffoldTarget)))
		onDiskAtProbeTime = err == nil
	}
	tools.reports = []StepReport{{Step: stepToolchainName, Outcome: OutcomeDelivered, Detail: "ran one declared entrypoint"}}

	runner := newTestRunner(t, installer, applier, gates, tools, seeds)
	if _, err := runner.Run(Options{
		ProjectRoot:       root,
		Capabilities:      allCapabilities(t),
		PackRefs:          []string{"fixture/pack@1.0.0"},
		ScaffoldRecipeRef: scaffoldRef,
	}); err != nil {
		t.Fatalf("the run errored: %v", err)
	}

	if tools.calls != 1 {
		t.Fatalf("the toolchain prober ran %d times, want once", tools.calls)
	}
	if !onDiskAtProbeTime {
		t.Fatalf("%s was NOT on disk when the toolchain step executed. The scaffold step ran after the toolchain step, which re-manufactures the empty-project entrypoint failure the scaffold step exists to prevent.", scaffoldTarget)
	}
}

// TestInit_OverwritesNoPreExistingConsumerFile (SPEC-069 CLM-040).
//
// Every file present before the run and not owned by backstop is byte-identical after
// it. Converge, never clobber.
func TestInit_OverwritesNoPreExistingConsumerFile(t *testing.T) {
	root := t.TempDir()

	// A spread of files a run might be tempted to touch, including the two backstop
	// DOES own — which must also be preserved, since they were already there.
	consumer := map[string]string{
		"backstop.yml":              "project: the-consumers-own-name\n",
		".gitignore":                "# the consumer's own\nsecret.env\n",
		"README.md":                 "# a project that already existed\n",
		"src/existing.entry":        "content that predates init\n",
		"config/settings.manifest":  "settings = true\n",
		".backstop/specs/SPEC-1.md": "an artifact that was already here\n",
	}
	for path, body := range consumer {
		writeFile(t, root, path, body)
	}

	installer, applier, gates, tools, seeds := defaultFakes()
	runner := newTestRunner(t, installer, applier, gates, tools, seeds)
	if _, err := runner.Run(Options{ProjectRoot: root, Capabilities: allCapabilities(t)}); err != nil {
		t.Fatalf("the run errored: %v", err)
	}

	for path, body := range consumer {
		if path == ".gitignore" {
			// The one file init APPENDS to. Every pre-existing byte must survive as a
			// prefix; that is CLM-034's contract, asserted here as non-clobbering.
			if !strings.HasPrefix(readFile(t, root, path), body) {
				t.Fatalf("%s lost its pre-existing bytes", path)
			}
			continue
		}
		if got := readFile(t, root, path); got != body {
			t.Fatalf("init overwrote the pre-existing file %s.\nbefore: %q\nafter:  %q", path, body, got)
		}
	}
}

// TestInit_SecondRunConvergesAndWritesNothing (SPEC-069 CLM-041).
//
// A second run immediately after a first writes NO new file, changes NO existing byte,
// and reports EVERY step as converged.
func TestInit_SecondRunConvergesAndWritesNothing(t *testing.T) {
	root := t.TempDir()
	installer, applier, gates, tools, seeds := defaultFakes()
	runner := newTestRunner(t, installer, applier, gates, tools, seeds)

	options := Options{ProjectRoot: root, Capabilities: allCapabilities(t)}
	if _, err := runner.Run(options); err != nil {
		t.Fatalf("the first run errored: %v", err)
	}
	after := snapshotTree(t, root)

	second, err := runner.Run(options)
	if err != nil {
		t.Fatalf("the second run errored: %v", err)
	}

	final := snapshotTree(t, root)
	if len(final) != len(after) {
		t.Fatalf("the second run changed the file count from %d to %d", len(after), len(final))
	}
	for path, body := range after {
		got, survived := final[path]
		if !survived {
			t.Fatalf("the second run removed %s", path)
		}
		if got != body {
			t.Fatalf("the second run changed %s.\nbefore: %q\nafter:  %q", path, body, got)
		}
	}

	// Every step that CAN converge reports converged. The steps that cannot — a
	// capability with no machinery, a flag the consumer did not pass — report their own
	// honest non-delivery instead, and neither is a change to the project.
	for _, step := range second.Steps {
		switch step.Outcome {
		case OutcomeConverged, OutcomeSkipped, OutcomeCapabilityAbsent, OutcomeDelivered:
		default:
			t.Fatalf("the second run's %q step reported %v (%s); a converged re-run has nothing left to fail at",
				step.Step, step.Outcome, step.Detail)
		}
	}
	for _, name := range []string{stepGitName, stepConfigName, stepLayoutName, stepGitignoreName} {
		step := requireStep(t, second.Steps, name)
		if step.Outcome != OutcomeConverged {
			t.Fatalf("the second run's %q step reported %v (%s), want OutcomeConverged", name, step.Outcome, step.Detail)
		}
	}
}

// TestInit_ReadsOnlyBackstopNeutralFactsToDecideWhatToDo (SPEC-069 CLM-042, denylist).
//
// The facts inspected are exactly `.git`, `backstop.yml`, the artifact directories and
// `.gitignore`. No other path in the project is read TO DECIDE anything.
//
// The assertion is behavioral rather than a source scan: two projects that differ ONLY
// in files outside that set must produce the same decisions, which is what "no other
// path is read to decide" means operationally.
func TestInit_ReadsOnlyBackstopNeutralFactsToDecideWhatToDo(t *testing.T) {
	decisionsFor := func(t *testing.T, extra map[string]string) []string {
		t.Helper()
		root := t.TempDir()
		for path, body := range extra {
			writeFile(t, root, path, body)
		}
		installer, applier, gates, tools, seeds := defaultFakes()
		runner := newTestRunner(t, installer, applier, gates, tools, seeds)
		result, err := runner.Run(Options{ProjectRoot: root, Capabilities: allCapabilities(t)})
		if err != nil {
			t.Fatalf("the run errored: %v", err)
		}

		decisions := make([]string, 0, len(result.Steps))
		for _, step := range result.Steps {
			decisions = append(decisions, step.Step+"="+step.Outcome.String())
		}
		return decisions
	}

	bare := decisionsFor(t, nil)
	cluttered := decisionsFor(t, map[string]string{
		"some-ecosystem.manifest":  "{\"declares\": \"a dependency graph\"}\n",
		"another.lockfile":         "locked = true\n",
		"src/deeply/nested.source": "content\n",
		"Makefile":                 "all:\n\techo hi\n",
		"tool.config.toml":         "[section]\nkey = 1\n",
	})

	if len(bare) != len(cluttered) {
		t.Fatalf("a project full of unrelated files produced a different number of decisions: %v vs %v", bare, cluttered)
	}
	for i := range bare {
		if bare[i] != cluttered[i] {
			t.Fatalf("decision %d differed between an empty project and one full of unrelated files: %q vs %q.\nSomething outside {.git, backstop.yml, the artifact directories, .gitignore} was read to decide.",
				i, bare[i], cluttered[i])
		}
	}
}

// TestInit_ReportUsesOnlyBackstopVocabulary (SPEC-069 CLM-043).
//
// Capability names, gate dimensions, artifact paths — and no ecosystem, language or
// framework noun anywhere in the report a consumer reads.
func TestInit_ReportUsesOnlyBackstopVocabulary(t *testing.T) {
	installer, applier, gates, tools, seeds := defaultFakes()
	gates.counts = []DimensionCount{{Dimension: "pack_engines", Count: 2}, {Dimension: "coverage_threshold", Count: 0}}
	tools.reports = []StepReport{{Step: stepToolchainName, Outcome: OutcomeDelivered, Detail: "ran the declared entrypoints"}}
	runner := newTestRunner(t, installer, applier, gates, tools, seeds)

	result, err := runner.Run(Options{ProjectRoot: t.TempDir(), Capabilities: allCapabilities(t)})
	if err != nil {
		t.Fatalf("the run errored: %v", err)
	}

	var report strings.Builder
	for _, step := range result.Steps {
		report.WriteString(step.Step)
		report.WriteString(" ")
		report.WriteString(step.Detail)
		report.WriteString("\n")
	}
	body := strings.ToLower(report.String())

	// The nouns live in this TEST, not in the source set, so naming them here is an
	// assertion rather than a bake.
	for _, noun := range []string{
		"golang", "typescript", "javascript", "python", "rust", "java",
		"npm", "yarn", "pnpm", "cargo", "maven", "gradle", "pip",
		"react", "django", "rails", "spring",
		"github actions", "gitlab ci", "jenkins", "circleci",
		"node_modules", "package.json", "go.mod", "cargo.toml", "requirements.txt",
	} {
		if strings.Contains(body, noun) {
			t.Fatalf("the report uses the ecosystem/language/framework noun %q; init's whole vocabulary is backstop's own.\n---\n%s", noun, report.String())
		}
	}
}

// TestInit_BehaviorIsIdenticalAcrossDifferingEcosystemMarkers (SPEC-069 CLM-045).
//
// Two fixture projects differing ONLY in which ecosystem marker files they contain
// produce identical reports, identical written files and identical exit-relevant
// verdicts. The marker filenames live in testdata, so they put no literal in the init
// source set the denylist claims scan.
func TestInit_BehaviorIsIdenticalAcrossDifferingEcosystemMarkers(t *testing.T) {
	runOver := func(t *testing.T, fixture string) (Result, map[string]string) {
		t.Helper()
		root := t.TempDir()
		copyFixtureInto(t, filepath.Join("testdata", fixture), root)

		installer, applier, gates, tools, seeds := defaultFakes()
		runner := newTestRunner(t, installer, applier, gates, tools, seeds)
		result, err := runner.Run(Options{ProjectRoot: root, Capabilities: allCapabilities(t)})
		if err != nil {
			t.Fatalf("the run over %s errored: %v", fixture, err)
		}
		return result, snapshotTree(t, root)
	}

	resultA, treeA := runOver(t, "ecosystem-marker-a")
	resultB, treeB := runOver(t, "ecosystem-marker-b")

	if resultA.BrokenPromise != resultB.BrokenPromise {
		t.Fatalf("the two ecosystem-marker projects produced different verdicts: %v vs %v", resultA.BrokenPromise, resultB.BrokenPromise)
	}
	if len(resultA.Steps) != len(resultB.Steps) {
		t.Fatalf("the two projects produced %d and %d step reports", len(resultA.Steps), len(resultB.Steps))
	}
	for i := range resultA.Steps {
		if resultA.Steps[i].Step != resultB.Steps[i].Step || resultA.Steps[i].Outcome != resultB.Steps[i].Outcome {
			t.Fatalf("step %d differed: %+v vs %+v", i, resultA.Steps[i], resultB.Steps[i])
		}
	}

	// The WRITTEN files must match too, modulo the marker each fixture carries and the
	// project name, which is the directory basename and therefore differs by
	// construction.
	writtenA := backstopWrittenFiles(treeA)
	writtenB := backstopWrittenFiles(treeB)
	if len(writtenA) != len(writtenB) {
		t.Fatalf("the two projects ended with different backstop-written file sets: %v vs %v", writtenA, writtenB)
	}
	for path := range writtenA {
		if _, present := writtenB[path]; !present {
			t.Fatalf("%s was written for one ecosystem-marker project and not the other", path)
		}
	}
}

// backstopWrittenFiles narrows a tree snapshot to the files init itself writes.
func backstopWrittenFiles(tree map[string]string) map[string]string {
	written := map[string]string{}
	for path, body := range tree {
		if strings.HasSuffix(path, ".manifest") {
			continue
		}
		written[path] = body
	}
	return written
}

// copyFixtureInto copies a testdata fixture project's files into root.
func copyFixtureInto(t *testing.T, fixture, root string) {
	t.Helper()
	entries, err := os.ReadDir(fixture)
	if err != nil {
		t.Fatalf("reading the fixture %s: %v", fixture, err)
	}
	if len(entries) == 0 {
		t.Fatalf("the fixture %s is empty; two projects that differ in nothing cannot falsify anything", fixture)
	}
	for _, entry := range entries {
		body, readErr := os.ReadFile(filepath.Join(fixture, entry.Name()))
		if readErr != nil {
			t.Fatalf("reading %s: %v", entry.Name(), readErr)
		}
		writeFile(t, root, entry.Name(), string(body))
	}
}

// TestInit_NewRunnerRefusesEachNilSeamByName pins the fail-closed constructor at the
// PACKAGE level. Its production-shape twin lives in cmd/backstop, over the real
// adapters; this one covers every seam, including the combinations a production test
// would not bother enumerating.
//
// The property is what makes a half-wired runner UNCONSTRUCTABLE rather than a runtime
// nil-deref — which is also why "no baseline seeder is available" had to become a
// sentinel VALUE instead of a nil field.
func TestInit_NewRunnerRefusesEachNilSeamByName(t *testing.T) {
	installer, applier, gates, tools, seeds := defaultFakes()

	cases := []struct {
		name    string
		build   func() (*Runner, error)
		mustSay string
	}{
		{"packs", func() (*Runner, error) { return NewRunner(nil, applier, gates, tools, seeds) }, "PackInstaller"},
		{"recipes", func() (*Runner, error) { return NewRunner(installer, nil, gates, tools, seeds) }, "RecipeApplier"},
		{"gates", func() (*Runner, error) { return NewRunner(installer, applier, nil, tools, seeds) }, "GateRunner"},
		{"tools", func() (*Runner, error) { return NewRunner(installer, applier, gates, nil, seeds) }, "ToolchainProber"},
		{"seeds", func() (*Runner, error) { return NewRunner(installer, applier, gates, tools, nil) }, "BaselineSeeder"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := tc.build()
			if err == nil {
				t.Fatalf("NewRunner accepted a nil %s", tc.mustSay)
			}
			if runner != nil {
				t.Fatal("NewRunner returned a runner alongside its error")
			}
			if !strings.Contains(err.Error(), tc.mustSay) {
				t.Fatalf("the refusal does not name %s.\ngot: %s", tc.mustSay, err.Error())
			}
		})
	}

	// EVERY nil is named, not just the first. An operator with a half-built binary
	// should learn the whole list from one message rather than one recompile at a time.
	_, err := NewRunner(nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("NewRunner accepted five nil seams")
	}
	for _, name := range []string{"PackInstaller", "RecipeApplier", "GateRunner", "ToolchainProber", "BaselineSeeder"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("the all-nil refusal does not name %s.\ngot: %s", name, err.Error())
		}
	}
}

// TestInit_ARefusingStepAbortsTheRunAsAnError covers the two places Run returns an
// ERROR rather than a report: a step that REFUSED the invocation.
//
// The distinction matters to the exit code. A refusal means nothing was done and the
// invocation must change (a config error, exit 2); a step that ran and failed to
// deliver is a broken promise reported in the run's own account (exit 1). Collapsing
// them would tell a consumer with a bad flag that their project has violations.
func TestInit_ARefusingStepAbortsTheRunAsAnError(t *testing.T) {
	t.Run("a local-path pack ref", func(t *testing.T) {
		installer, applier, gates, tools, seeds := defaultFakes()
		runner := newTestRunner(t, installer, applier, gates, tools, seeds)

		_, err := runner.Run(Options{
			ProjectRoot:  t.TempDir(),
			Capabilities: allCapabilities(t),
			PackRefs:     []string{"./a-local-pack"},
		})
		if err == nil {
			t.Fatal("a local-path --pack value did not abort the run")
		}
		if !strings.Contains(err.Error(), stepPacksName) {
			t.Fatalf("the abort does not name the step that refused.\ngot: %s", err.Error())
		}
		if gates.calls != 0 {
			t.Fatal("the gate ran after a refusal; a refused invocation must do nothing at all")
		}
	})

	t.Run("a refused entrypoint tool", func(t *testing.T) {
		installer, applier, gates, tools, seeds := defaultFakes()
		tools.err = errors.New("the trusted-tool allowlist refused this entrypoint")
		runner := newTestRunner(t, installer, applier, gates, tools, seeds)

		_, err := runner.Run(Options{ProjectRoot: t.TempDir(), Capabilities: allCapabilities(t)})
		if err == nil {
			t.Fatal("a refused entrypoint tool did not abort the run")
		}
		if !strings.Contains(err.Error(), stepToolchainName) {
			t.Fatalf("the abort does not name the step that refused.\ngot: %s", err.Error())
		}
		if !strings.Contains(err.Error(), "allowlist refused") {
			t.Fatalf("the abort does not carry the refusal's own reason.\ngot: %s", err.Error())
		}
	})
}

// TestInit_OutcomeAndPreserveClassRenderEveryValue pins the two report renderers.
//
// They are what a consumer reads on every line of every run, and their DEFAULT arms
// exist so an unrecognized value is visible as itself rather than silently rendering as
// some neighbouring state — which is exactly how a new outcome added later would
// otherwise masquerade as "delivered".
func TestInit_OutcomeAndPreserveClassRenderEveryValue(t *testing.T) {
	outcomes := map[Outcome]string{
		OutcomeDelivered:        "delivered",
		OutcomeConverged:        "converged",
		OutcomeSkipped:          "skipped",
		OutcomeCapabilityAbsent: "capability absent",
		OutcomeBrokenPromise:    "not delivered",
	}
	for outcome, want := range outcomes {
		if got := outcome.String(); got != want {
			t.Fatalf("Outcome(%d).String() = %q, want %q", int(outcome), got, want)
		}
	}
	if got := Outcome(99).String(); !strings.Contains(got, "99") {
		t.Fatalf("an unrecognized Outcome rendered as %q; it must name itself rather than borrow a neighbour's label", got)
	}

	classes := map[PreserveClass]string{
		PreserveWaiverCovered: "waiver-covered",
		PreserveUserOwned:     "user-owned",
		PreserveIndeterminate: "indeterminate",
	}
	for class, want := range classes {
		if got := class.String(); got != want {
			t.Fatalf("PreserveClass(%d).String() = %q, want %q", int(class), got, want)
		}
	}
	if got := PreserveClass(99).String(); !strings.Contains(got, "99") {
		t.Fatalf("an unrecognized PreserveClass rendered as %q", got)
	}

	// Only the accountable class is not a gap.
	if PreserveWaiverCovered.IsGap() {
		t.Fatal("a waiver-covered preserve is an accountable customization, not a gap")
	}
	for _, class := range []PreserveClass{PreserveUserOwned, PreserveIndeterminate} {
		if !class.IsGap() {
			t.Fatalf("%v is not reported as a gap; both of the empty-pair classes exit non-zero", class)
		}
	}
}
