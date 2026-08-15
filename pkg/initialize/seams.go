package initialize

import (
	"errors"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// The five seams init reaches the rest of backstop through.
//
// EVERY ONE OF THEM HAS ITS PRODUCTION IMPLEMENTATION IN `package main` UNDER
// cmd/backstop, AND THE LOCATION IS FORCED FOR EACH:
//
//	PackInstaller   -> newProductionAddCommand is THE production assembly seam for
//	                   the pack lifecycle; assembling a second AddCommand elsewhere is
//	                   the partial-wiring defect that file exists to end.
//	RecipeApplier   -> runRecipeApply is the shipped resolve+apply path, and it
//	                   depends on loadInstalledPacks, provisionedEngineBinding,
//	                   checkEngineToolAllowed, transformDispatch and
//	                   recordRecipeAdoption — all unexported in package main. A
//	                   re-implementation here would be a SECOND apply path.
//	GateRunner      -> buildGateSteps assembles the whole ordered kill chain from the
//	                   installed packs and derived classifiers. It is unexported, and
//	                   a second gate assembly would make init OBSERVE a different gate
//	                   than `backstop gate` RUNS.
//	ToolchainProber -> the allowlist trust gate and the command splitter REQ-011 binds
//	                   execution to are unexported in package main, so an
//	                   implementation here would need a second copy of both — the
//	                   exact second execution path REQ-011 forbids.
//	BaselineSeeder  -> there is no seeding machinery and this spec builds none.
//
// pkg/initialize holds the INTERFACES only.

// PackInstaller installs one pack ref into a project.
//
// It takes the WHOLE ref as an opaque string. Init supplies no version separately:
// the pinned ref is the consumer's, and splitting a version out of it would be core
// holding half a ref.
type PackInstaller interface {
	Install(projectRoot string, ref string) error
}

// ApplyOutcome is what one recipe apply produced, in the shape REQ-035's classifier
// needs.
//
// It surfaces the applier's OWN recipe.PreservedDivergence values with Rule and
// CoveringWaiver INTACT, plus the resolved recipe's declared kind. All three are
// already exported at HEAD, so this widening touches no file under pkg/recipe —
// which REQ-009 forbids editing.
//
// THESE TWO FIELDS ARE THE WHOLE OF WHAT INIT CAN CLASSIFY ON. The applier's
// recipe-level adoption bit is keyed by an UNEXPORTED derivation and is not carried
// here; reconstructing it would be a second derivation of a recipe's adoption
// identity, so the empty-pair-plus-templating case stays INDETERMINATE by design.
type ApplyOutcome struct {
	Written    []string
	Preserved  []recipe.PreservedDivergence
	RecipeKind string
}

// RecipeApplier applies one pinned recipe ref into a project.
//
// ONE seam, TWO callers: the CI step and the scaffold step both apply through this
// single interface with the ref their own flag supplied, and both run the returned
// preserves through the single shared classifier. The interface takes the ref as an
// OPAQUE STRING precisely so adding the scaffold step required no new method, no
// per-step variant, and no place for core to construct a ref part.
type RecipeApplier interface {
	Apply(projectRoot string, ref string) (ApplyOutcome, error)
}

// GateRunner runs the gate once and reduces it to per-dimension counts.
//
// The gate's VERDICT is deliberately not part of this seam. Findings are
// OBSERVATION: a non-passing gate is not an error, it is counts. Returning an error
// on a red gate would make pre-existing findings an init failure, which REQ-014
// forbids in the one sentence the whole exit-code matrix is organized around.
type GateRunner interface {
	Run(projectRoot string) ([]DimensionCount, error)
}

// ToolchainProber executes every installed pack's declared test/build entrypoint
// once and reports what each one did.
type ToolchainProber interface {
	Probe(projectRoot string) ([]StepReport, error)
}

// BaselineSeeder seeds the gitignored local baseline and returns where it landed.
//
// A seeder with no machinery behind it returns ErrBaselineSeedingUnavailable.
type BaselineSeeder interface {
	Seed(projectRoot string) (string, error)
}

// ErrBaselineSeedingUnavailable is the sentinel a BaselineSeeder returns to say the
// capability has no machinery behind it. It is capability-ABSENT, never a broken
// promise.
//
// WHY IT MUST EXIST. REQ-012 requires the baseline step to report a gap "when no
// seeding implementation is available", and NewRunner is FAIL-CLOSED — it errors
// naming any nil dependency. A nil BaselineSeeder is therefore UNCONSTRUCTABLE, so
// "unavailable" cannot be a nil field and must be a VALUE the seam returns. The
// sentinel is the smallest thing that satisfies both declared behaviors at once, and
// it adds no capability and no machinery: production wires a seeder whose whole body
// returns it.
//
// The baseline step matches it with errors.Is, never a string compare, and treats
// every OTHER error as a step that genuinely failed to deliver.
var ErrBaselineSeedingUnavailable = errors.New("no baseline seeding implementation is available: the gitignored local baseline at .backstop/baseline.json is owned by ISSUE-056, and this command builds none of that machinery") // nosemgrep: go.core.no-global-mutable-state — immutable sentinel error, never reassigned
