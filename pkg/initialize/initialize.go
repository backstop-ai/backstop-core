package initialize

import (
	"fmt"
	"os"
	"strings"
)

// Options is one `backstop init` invocation's whole input.
//
// TWO REF-SHAPED FIELDS AND NO REF-SHAPED CONSTRUCTOR. CIRecipeRef and
// ScaffoldRecipeRef are whole OPAQUE strings — empty when the flag was omitted, which
// is the honest-skip case — and are NEVER decomposed into pack/recipe/version parts
// anywhere in core. A field holding a pack name or a version separately would be core
// holding half a ref, which is the bake REQ-016 v1.2.0 was corrected to remove.
// Neither has a param companion: init supplies no recipe param to either apply.
type Options struct {
	// ProjectRoot is the directory init operates on. Init writes beneath it and
	// nowhere else.
	ProjectRoot string
	// Capabilities is the RESOLVED set, produced by ResolveCapabilities and by
	// nothing else.
	Capabilities map[Capability]bool
	// PackRefs are the `--pack <ref>` values, in the order the consumer supplied
	// them. Init installs exactly these and nothing else.
	PackRefs []string
	// CIRecipeRef is the `--ci <pack>:<recipe>@<version>` value, verbatim.
	CIRecipeRef string
	// ScaffoldRecipeRef is the `--scaffold <pack>:<recipe>@<version>` value, verbatim.
	ScaffoldRecipeRef string
}

// Outcome is what one step did. It is the value the command's exit mapping reads, so
// the distinctions here are the distinctions the exit code carries.
//
// THE ORGANIZING RULE is REQ-014's precedence clause: pre-existing findings are NEVER
// an init failure, but an init STEP that failed to deliver what it promised ALWAYS
// is. Capability-absent outcomes are neither — nothing promised them.
type Outcome int

const (
	// OutcomeDelivered: the step did what it promised.
	OutcomeDelivered Outcome = iota
	// OutcomeConverged: the state the step promises was ALREADY present, so it
	// changed nothing. Distinct from Delivered because REQ-007's second-run claim is
	// precisely that every step reports THIS.
	OutcomeConverged
	// OutcomeSkipped: the consumer did not ask for this step, so it is a deliberate
	// no-op reported in words. This is `--ci` and `--scaffold` omitted.
	OutcomeSkipped
	// OutcomeCapabilityAbsent: the step was asked for, but there is no capability
	// behind it to exercise — no installed pack declares an entrypoint, no baseline
	// seeding machinery exists. An un-adopted capability is a MISSING BENEFIT, never
	// a broken promise, so this does not fail the run.
	OutcomeCapabilityAbsent
	// OutcomeBrokenPromise: the consumer ASKED for something and init did not
	// deliver it. This is the only outcome that carries a non-zero exit.
	OutcomeBrokenPromise
)

// String renders an Outcome for a report line.
func (o Outcome) String() string {
	switch o {
	case OutcomeDelivered:
		return "delivered"
	case OutcomeConverged:
		return "converged"
	case OutcomeSkipped:
		return "skipped"
	case OutcomeCapabilityAbsent:
		return "capability absent"
	case OutcomeBrokenPromise:
		return "not delivered"
	default:
		return fmt.Sprintf("Outcome(%d)", int(o))
	}
}

// StepReport is one step's account of itself: what it was, how it went, and the
// detail a human needs to act.
type StepReport struct {
	Step    string
	Outcome Outcome
	Detail  string
}

// PreserveClass names REQ-035's OBSERVABLE classes, which are deliberately NOT the
// code's three producers.
//
// preserveOrRegenerate (pkg/recipe/apply.go) has three branches, but its
// `!own.adopted` test returns BEFORE the kind test, so a never-adopted templating
// recipe emits a value BYTE-IDENTICAL to an adopted-and-materialized one. What init
// holds is exactly two observables — the Rule/CoveringWaiver pair per preserve, and
// the resolved recipe's declared kind — and they yield the three classes below.
// Naming the type for PRODUCERS would re-assert a discrimination init cannot make.
type PreserveClass int

const (
	// PreserveWaiverCovered: Rule and CoveringWaiver POPULATED, at any declared
	// kind. Unambiguously producer (c) — recipe-owned output the consumer
	// legitimately customized and accounted for with a valid @waiver token. The gate
	// IS wired and the customization is accountable: no gap, exit 0.
	PreserveWaiverCovered PreserveClass = iota
	// PreserveUserOwned: the pair is EMPTY and the declared kind is `scaffolding` or
	// `implementing`. Unambiguously producer (a), because branch (b) is unreachable
	// for those kinds. THIS is the REQ-035 brownfield gap: exit non-zero.
	PreserveUserOwned
	// PreserveIndeterminate: the pair is EMPTY and the declared kind is
	// `templating`. Producers (a) and (b) are INDISTINGUISHABLE from what init can
	// see. Claiming success would hide a real brownfield gap; asserting that no gate
	// was wired would be a false positive about a one-shot that already
	// materialized. DD-15's "on 'I cannot tell', REFUSE" posture governs, so it is
	// scored conservatively as a gap — named honestly, exit non-zero, and using NO
	// "no gate was wired" language, because that is the half init cannot know.
	PreserveIndeterminate
)

// String renders a PreserveClass for a report line and a test failure message.
func (c PreserveClass) String() string {
	switch c {
	case PreserveWaiverCovered:
		return "waiver-covered"
	case PreserveUserOwned:
		return "user-owned"
	case PreserveIndeterminate:
		return "indeterminate"
	default:
		return fmt.Sprintf("PreserveClass(%d)", int(c))
	}
}

// IsGap reports whether a class is a gap the consumer must act on. Only the
// accountable class is not.
func (c PreserveClass) IsGap() bool {
	return c != PreserveWaiverCovered
}

// ClassifiedPreserve is one preserved file together with everything init can say
// about it. Carrying the Rule/CoveringWaiver pair alongside the class is what lets
// the waiver-covered report NAME the rule and the covering token.
type ClassifiedPreserve struct {
	Path           string
	Class          PreserveClass
	Rule           string
	CoveringWaiver string
}

// DimensionCount is one gate dimension and how many findings it reported.
//
// Dimension names come from the gate's OWN step vocabulary, so grouping by them
// introduces no tool or language noun into init's report.
type DimensionCount struct {
	Dimension string
	Count     int
}

// Result is one init run's whole account of itself.
//
// Preserved carries CLASSIFIED preserves, not paths. A []string would discard the
// Rule/CoveringWaiver pair and the recipe kind, which are the only two observables
// separating an accountable customization from a brownfield gap from an unresolvable
// unknown — and reporting all three as the brownfield gap is the false "no gate was
// wired" this type exists to prevent.
//
// BrokenPromise is computed from THE REQUEST — did the consumer ASK for something
// init then failed to deliver — NOT from the resulting filesystem state. Two
// structurally identical no-ops carry different verdicts: `--ci` omitted and `--ci`
// supplied-and-unresolvable both end with no CI wired. A refactor that computes the
// verdict from the RESULT collapses that pair.
type Result struct {
	Steps         []StepReport
	Observations  []DimensionCount
	Preserved     []ClassifiedPreserve
	BrokenPromise bool
}

// Runner executes the ten-step sequence over the five seams.
//
// Every dependency is required. There is no partially-wired Runner, and there is no
// field a caller can leave nil and discover at run time.
type Runner struct {
	packs   PackInstaller
	recipes RecipeApplier
	gates   GateRunner
	tools   ToolchainProber
	seeds   BaselineSeeder
}

// NewRunner assembles a Runner and is FAIL-CLOSED: it errors naming any nil
// dependency, the shape pkg/pack/distribution's NewAddCommand uses.
//
// A half-wired runner is UNCONSTRUCTABLE rather than a runtime nil-deref, which is
// why "no baseline seeder is available" cannot be expressed as a nil field and is a
// VALUE the seam returns instead (ErrBaselineSeedingUnavailable). Special-casing
// BaselineSeeder here would make one of the five dependencies silently optional and
// reopen exactly the hole this constructor closes.
func NewRunner(packs PackInstaller, recipes RecipeApplier, gates GateRunner, tools ToolchainProber, seeds BaselineSeeder) (*Runner, error) {
	missing := []string{}
	if packs == nil {
		missing = append(missing, "PackInstaller")
	}
	if recipes == nil {
		missing = append(missing, "RecipeApplier")
	}
	if gates == nil {
		missing = append(missing, "GateRunner")
	}
	if tools == nil {
		missing = append(missing, "ToolchainProber")
	}
	if seeds == nil {
		missing = append(missing, "BaselineSeeder")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("cannot construct the init runner: no %s was supplied", strings.Join(missing, ", no "))
	}

	return &Runner{packs: packs, recipes: recipes, gates: gates, tools: tools, seeds: seeds}, nil
}

// pathExists reports whether anything is present at path.
//
// It lives here, once, because several steps ask the same question about their own
// backstop-neutral fact — `.git`, `backstop.yml`, an artifact directory, `.gitignore`
// — and a per-step copy would be four chances for one of them to start tolerating a
// stat error the others refuse.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// directoryExists reports whether path is present AND is a directory.
//
// The distinction matters for `.git`: a FILE named `.git` is what a git worktree or
// submodule checkout carries, and treating it as an absent repository would have init
// run `git init` inside one.
func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Run executes the TEN-STEP sequence and returns one account of the whole run
// (SPEC-069 REQ-001, REQ-002, REQ-007, REQ-008, REQ-019).
//
// THE ORDER IS FIXED, AND SUBTRACTING A CAPABILITY REMOVES ITS STEP AND REORDERS
// NOTHING:
//
//	 1  git        (git capability)        2  config    (UNCONDITIONAL)
//	 3  layout     (sdlc capability)       4  packs     (packs capability)
//	 5  gitignore  (gitignore capability)  6  scaffold  (--scaffold flag)
//	 7  toolchain  (toolchain capability)  8  baseline  (baseline capability)
//	 9  ci         (--ci flag)            10  observe   (observe capability)
//
// Steps 6 and 9 are FLAG-GOVERNED and are NOT capabilities: omission is the opt-out,
// so there is no `--no-scaffold` and no `--no-ci`.
//
// ★ TWO DEVIATIONS FROM THE TRANSCRIBED HAND-ONBOARDING ORDER, AND NEITHER MAY BE
// "TIDIED":
//
// DEVIATION 1 — gitignore AFTER packs. The entry set is a FUNCTION of the installed
// packs' declared stdout_artifact values, so emitting the file first is unsatisfiable:
// a `--pack` run on a fresh repo would write a `.gitignore` missing every pack-derived
// entry, which is the cross-repo ignore divergence this whole flow exists to end.
//
// DEVIATION 2 — scaffold at 6, not 1 and not beside ci at 9. Not at 1, because a
// recipe cannot resolve out of a pack that is not installed yet. NOT AT 9, because the
// toolchain step is EXACTLY the run that hits the empty-project failure a scaffolded
// source file prevents. Grouping "the two recipe steps" reads as a tidy-up and
// manufactures the very failure the scaffold step exists to prevent — which is why the
// claims that guard this assert the file is ON DISK at the toolchain step's boundary
// and that an entrypoint's verdict FLIPS, rather than asserting the order alone.
//
// ONLY BACKSTOP-NEUTRAL FACTS ARE INSPECTED to decide anything: the presence of
// `.git`, of `backstop.yml`, of the artifact directories, of `.gitignore`. No other
// path in the project is read to make a decision.
func (r *Runner) Run(opts Options) (Result, error) {
	var result Result

	record := func(report StepReport) {
		result.Steps = append(result.Steps, report)
	}

	// 1 — git.
	if opts.Capabilities[CapabilityGit] {
		record(stepGit(opts.ProjectRoot))
	}

	// 2 — config. UNCONDITIONAL: an init that does not write the config produces
	// nothing a consumer can use, so it is not a capability and cannot be subtracted.
	record(stepConfig(opts.ProjectRoot, opts.Capabilities))

	// 3 — layout.
	if opts.Capabilities[CapabilitySdlc] {
		record(stepLayout(opts.ProjectRoot, opts.Capabilities))
	}

	// 4 — packs. A local-path refusal is a CONFIG error and aborts the run: nothing has
	// been installed, and continuing would produce a report about a project the
	// consumer's own invocation cannot produce.
	if opts.Capabilities[CapabilityPacks] {
		report, err := stepPacks(opts.ProjectRoot, opts.PackRefs, r.packs)
		if err != nil {
			return Result{}, fmt.Errorf("the %s step refused this invocation: %w", stepPacksName, err)
		}
		record(report)
	}

	// 5 — gitignore. It reads the manifests of the packs step 4 just installed, which
	// is why it cannot run any earlier.
	if opts.Capabilities[CapabilityGitignore] {
		record(stepGitignore(opts.ProjectRoot, installedManifests(opts.ProjectRoot)))
	}

	// 6 — scaffold. Flag-governed, and BEFORE the toolchain step.
	scaffoldReport, scaffoldPreserves := stepScaffold(opts.ProjectRoot, opts.ScaffoldRecipeRef, r.recipes)
	record(scaffoldReport)
	result.Preserved = append(result.Preserved, scaffoldPreserves...)

	// 7 — toolchain. A refusal here is the allowlist trust gate saying no, which is a
	// CONFIG error rather than a toolchain verdict, so it aborts rather than being
	// reported as a pass or a fail.
	if opts.Capabilities[CapabilityToolchain] {
		reports, err := r.tools.Probe(opts.ProjectRoot)
		if err != nil {
			return Result{}, fmt.Errorf("the %s step refused this invocation: %w", stepToolchainName, err)
		}
		if len(reports) == 0 {
			record(StepReport{
				Step:    stepToolchainName,
				Outcome: OutcomeCapabilityAbsent,
				Detail:  "no installed pack declares a test or build entrypoint, so there was nothing to run. That is not a failure — it just means your packs do not describe how to build or test this project yet",
			})
		}
		for _, report := range reports {
			record(report)
		}
	}

	// 8 — baseline.
	if opts.Capabilities[CapabilityBaseline] {
		record(stepBaseline(opts.ProjectRoot, r.seeds))
	}

	// 9 — ci. Flag-governed.
	ciReport, ciPreserves := stepCI(opts.ProjectRoot, opts.CIRecipeRef, r.recipes)
	record(ciReport)
	result.Preserved = append(result.Preserved, ciPreserves...)

	// 10 — observe.
	if opts.Capabilities[CapabilityObserve] {
		report, observations := stepObserve(opts.ProjectRoot, r.gates)
		record(report)
		result.Observations = observations
	}

	// THE VERDICT IS COMPUTED FROM THE REQUEST, NOT FROM THE RESULTING FILESYSTEM
	// STATE. Two structurally identical no-ops carry different verdicts: `--ci` omitted
	// and `--ci` supplied-and-unresolvable both end with no CI wired, and "no packs
	// supplied" and "pack ref refused" both end with no packs. What separates them is
	// whether the consumer ASKED — and that is visible only in the invocation, which is
	// why a refactor that derives this from the RESULT collapses both pairs.
	//
	// OutcomeBrokenPromise is the ONLY outcome that sets it. Capability-absent and
	// skipped do not: nothing promised them.
	for _, step := range result.Steps {
		if step.Outcome == OutcomeBrokenPromise {
			result.BrokenPromise = true
		}
	}

	return result, nil
}

// StepToolchain is the report name for step 7.
//
// It is EXPORTED because the concrete ToolchainProber lives in `package main` under
// cmd/backstop — forced there by the unexported allowlist gate and command splitter it
// must bind — so the prober names its reports from here rather than from a second
// literal that could drift from the runner's own capability-absent report.
const StepToolchain = "toolchain"

// stepToolchainName is the package-internal spelling of the same name.
const stepToolchainName = StepToolchain
