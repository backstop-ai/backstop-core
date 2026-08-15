package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/initialize"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// init_seams.go holds the PRODUCTION implementations of four of init's five seams.
//
// Each does exactly ONE thing: translate between a pkg/initialize seam and an assembly
// that ALREADY EXISTS in package main. If any of them grows logic of its own, it has
// become a second implementation of something and the wrong thing is being built.
//
// THE FILE IS IN cmd/backstop, AND FOR EVERY ADAPTER THE LOCATION IS FORCED — the same
// forcing the spec states for the toolchain prober, applied consistently. It is also
// inside the `cmd/backstop/init*.go` glob the structural claims scan, which is
// deliberate: these adapters ARE init source and must answer to the no-second-execution-
// path and no-dependency-installation claims like the rest of it.
//
// NO SECOND EXECUTION PATH IS INTRODUCED BY ANY OF THE FOUR. None constructs an
// exec.Command, invokes a shell, or calls checkEngineToolAllowed / splitCommand — the
// only command execution on init's path is the shared entrypoint prober's, and the only
// pack-command execution these four reach is whatever runRecipeApply and buildGateSteps
// already do inside the shipped assemblies.

// initPackInstaller is the production initialize.PackInstaller.
//
// It installs through newProductionAddCommand, which is THE production assembly seam
// for the pack lifecycle. Assembling a second AddCommand here — or handing it a nil
// validator "because init already validated" — is exactly the partial-wiring defect
// that file exists to end: `pack add` once shipped with no cloner in it at all and
// nil-dereferenced on the first remote pack anyone tried.
type initPackInstaller struct{}

// Install adds one pack ref into the project.
//
// Init supplies NO Version: the whole pinned ref is the consumer's, and splitting a
// version out of it would be core holding half a ref.
func (initPackInstaller) Install(projectRoot string, ref string) error {
	add, err := newProductionAddCommand()
	if err != nil {
		return fmt.Errorf("assembling the pack installer: %w", err)
	}
	if _, runErr := add.Run(ref, distribution.AddOptions{ProjectDir: projectRoot}); runErr != nil {
		return fmt.Errorf("installing pack %s: %w", ref, runErr)
	}
	return nil
}

// initRecipeApplier is the production initialize.RecipeApplier.
//
// It applies through runRecipeApply, the SHIPPED resolve+apply path — the same one
// `backstop recipe apply` drives. A pkg/initialize re-implementation would be a SECOND
// apply path, and that path depends on loadInstalledPacks, provisionedEngineBinding,
// checkEngineToolAllowed, transformDispatch and recordRecipeAdoption, all unexported
// here.
type initRecipeApplier struct{}

// Apply applies one pinned recipe ref and reports what it produced.
//
// THE PARAM ARGUMENT IS NIL, AND THAT IS THE REQUIREMENT RATHER THAN AN OVERSIGHT. Init
// supplies no recipe param to EITHER apply, and the most tempting derivation — the
// project basename already computed for `project:` — would be core constructing recipe
// INPUT, the same defect one layer in as core constructing the pack half of a ref. No
// params map is built here, empty or otherwise, that a later reader could fill.
//
// THE ERROR SURFACES UNCHANGED. Every fail-loud resolve error is produced INSIDE this
// path and must reach the step VERBATIM: no wrapping that prepends init's own guidance,
// no re-classification, no ConfigError/plain-error reshuffling. The step attributes it;
// this adapter does not interpret it.
//
// Rule and CoveringWaiver are carried through INTACT. Dropping either collapses
// REQ-035's three preserve classes into one, which is the false "no gate was wired"
// report the whole classification exists to prevent.
func (initRecipeApplier) Apply(projectRoot string, ref string) (initialize.ApplyOutcome, error) {
	result, kind, err := runRecipeApply(ref, projectRoot, nil)

	// THE OUTCOME IS BUILT ONLY ON SUCCESS AND THE ERROR IS RETURNED WHATEVER IT IS.
	//
	// The inverted condition is how "this error passes through UNTOUCHED" is said
	// structurally rather than in a comment alone. The usual shape — an early return
	// under `if err != nil` — invites the ordinary Go habit of wrapping with context on
	// the way out, and here that habit is WRONG: every fail-loud resolve error is
	// produced inside runRecipeApply and must reach the step VERBATIM, so that the step
	// can attribute it without init having interpreted it. A zero outcome on failure is
	// the applier's own contract: an apply either produces a verdict or it fails, never
	// both.
	outcome := initialize.ApplyOutcome{}
	if err == nil {
		outcome = initialize.ApplyOutcome{
			Written:    result.Written,
			Preserved:  result.Preserved,
			RecipeKind: kind,
		}
	}
	return outcome, err
}

// initGateRunner is the production initialize.GateRunner.
//
// It assembles the kill chain through buildGateSteps — the SAME assembly `backstop
// gate` uses — rather than hand-building a step list. buildGateSteps is what loads the
// installed packs and derives the SourceClassifier and TestNameMatcher, and a second
// assembly would make init OBSERVE a different gate than `backstop gate` RUNS. An
// observation that disagrees with the command it is previewing is worse than no
// observation at all.
type initGateRunner struct{}

// Run runs the gate once and reduces it to one DimensionCount per gate dimension.
//
// THE GATE'S VERDICT IS NOT CONSULTED. Findings are OBSERVATION: a non-passing gate is
// not an error return, it is counts. Returning an error on a red gate would make
// pre-existing findings an init failure, which REQ-014 forbids in the one sentence the
// whole exit-code matrix is organized around.
//
// The scope and artifact root are resolved the way runGate resolves them, from the
// config init itself just wrote — so the run init previews is the run the consumer's
// next bare `backstop gate` will perform.
func (initGateRunner) Run(projectRoot string) ([]initialize.DimensionCount, error) {
	cfg, err := config.LoadConfigFromPath(filepath.Join(projectRoot, "backstop.yml"))
	if err != nil {
		return nil, fmt.Errorf("reading the configuration the gate runs against: %w", err)
	}

	root, rootErr := artifact.ResolveRoot(projectRoot, cfg.ArtifactRoot)
	if rootErr != nil {
		return nil, fmt.Errorf("resolving the artifact root the gate scans: %w", rootErr)
	}

	// The SHIPPED DEFAULT scope, and no flag of init's own. REQ-015 is satisfied by not
	// regressing it, so init observes exactly the scope a bare `backstop gate` would.
	scope, scopeErr := gate.ComputeGateScope(projectRoot, gate.GateScopeModeDiff, nil)
	if scopeErr != nil {
		return nil, fmt.Errorf("computing the gate scope: %w", scopeErr)
	}

	options := []gate.Option{
		gate.WithSteps(buildGateSteps(projectRoot, root, scope)),
		gate.WithScope(scope),
	}
	if policy := gatePolicyFromConfig(cfg); len(policy) > 0 {
		options = append(options, gate.WithPolicy(policy))
	}

	// THE SECOND RETURN IS THE GATE'S EXIT CODE, AND IT IS DELIBERATELY NOT CONSULTED.
	// Findings are OBSERVATION: a non-passing gate is not an init failure, it is counts.
	// Reading the verdict here and turning it into an error would make PRE-EXISTING
	// findings — violations a consumer inherited in a project init has only just started
	// governing — an init failure, which REQ-014 forbids in the one sentence the whole
	// exit-code matrix is organized around. It is bound to a named value rather than
	// discarded so that intent is legible instead of looking like an oversight.
	result, gateExitCode := gate.New(options...).Run(context.Background())
	_ = gateExitCode

	counts := make([]initialize.DimensionCount, 0, len(result.Steps))
	for _, step := range result.Steps {
		counts = append(counts, initialize.DimensionCount{
			Dimension: step.StepName,
			Count:     len(step.Violations),
		})
	}
	return counts, nil
}

// unavailableBaselineSeeder is the production initialize.BaselineSeeder, and its whole
// body is the sentinel.
//
// THIS SPEC BUILDS NO SEEDING MACHINERY. The gitignored local baseline is owned
// elsewhere, and nothing in the init source set writes `.backstop/baseline.json` or
// computes a fingerprint. The type exists ONLY because NewRunner is fail-closed and a
// nil seam is therefore unconstructable, so "capability absent" has to be a VALUE.
//
// Do not grow it into a real seeder here, and do not have it consult anything on disk
// to decide: a seeder that looked around before saying "unavailable" would be the first
// half of the machinery this boundary exists to keep out.
type unavailableBaselineSeeder struct{}

// Seed reports that no seeding implementation is available.
func (unavailableBaselineSeeder) Seed(projectRoot string) (string, error) {
	return "", initialize.ErrBaselineSeedingUnavailable
}
