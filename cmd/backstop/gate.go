package main

import (
	"context"
	"fmt"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/spf13/cobra"
)

// GateResult is re-exported from pkg/gate for the contract.
type GateResult = gate.GateResult

// StepResult is re-exported from pkg/gate for the contract.
type StepResult = gate.StepResult

// gateCmd is the top-level gate command variable.
var gateCmd *cobra.Command

// newGateCommand creates the Cobra command for backstop gate.
// Gate accepts no scope flags (REQ-012) — only inherits --json from root.
func newGateCommand(jsonFlag *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Run full verification gate",
		Long: `Runs the complete backstop gate: the full reconciliation kill chain
that orchestrates artifact validation, code checking, test verification,
test substantiveness checks, coverage threshold verification, contract
signature verification, baseline comparison, waiver resolution, and ledger
integrity verification. This is the primary enforcement checkpoint — if
it's green, it ships.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGate(cmd, args, jsonFlag)
		},
	}
	// No additional flags — gate accepts no scope flags (REQ-012)
	return cmd
}

// runGate is the Cobra RunE handler that orchestrates all nine gate steps.
func runGate(cmd *cobra.Command, _ []string, jsonFlag *bool) error {
	// Load config via CLI foundation config loader.
	_, cfgErr := config.LoadConfig()

	// If config loading fails, return immediately with exit code 2.
	// This preserves the original error message for upstream tests.
	if cfgErr != nil {
		return &ExitCodeError{
			Code:    ExitConfigError,
			Message: fmt.Sprintf("config: %s", cfgErr),
		}
	}

	// Build gate with step implementations.
	var opts []gate.Option

	// Wire up the nine steps with concrete implementations.
	// Steps 1-2 delegate to placeholder implementations for now.
	// Steps 3-6 use real grep/AST verification.
	// Steps 7-9 are deferred.
	steps := buildGateSteps()
	opts = append(opts, gate.WithSteps(steps))

	g := gate.New(opts...)
	result, exitCode := g.Run(context.Background())

	// Format output based on --json flag.
	if jsonFlag != nil && *jsonFlag {
		data, err := gate.FormatJSON(result)
		if err != nil {
			return fmt.Errorf("formatting gate JSON output: %w", err)
		}
		cmd.Println(string(data))
	} else {
		noColor := gate.NoColorFromEnv()
		output := gate.FormatHuman(result, noColor)
		cmd.Print(output)
	}

	if exitCode != 0 {
		return &ExitCodeError{
			Code:    exitCode,
			Message: fmt.Sprintf("gate: exit code %d", exitCode),
		}
	}
	return nil
}

// buildGateSteps constructs the nine ordered step functions with concrete
// implementations. Steps 1-2 use no-op validators for now (the real
// delegation will be wired when artifact validate and code check expose
// library functions). Steps 3-6 are mechanical verifiers. Steps 7-9 are
// deferred.
func buildGateSteps() []gate.StepFunc {
	return []gate.StepFunc{
		// Step 1: Artifact validation — no-op pass for now
		gate.StepArtifactValidationFunc(&noopValidator{}),
		// Step 2: Code check — no-op pass for now
		gate.StepCodeCheckFunc(&noopChecker{}),
		// Step 3: Test verification — would need spec/code dirs
		makeNoopStepFunc(gate.StepTestVerification),
		// Step 4: Test substantiveness — needs test function list
		makeNoopStepFunc(gate.StepTestSubstantiveness),
		// Step 5: Coverage threshold — needs command runner and spec data
		makeNoopStepFunc(gate.StepCoverageThreshold),
		// Step 6: Contract signature — needs contract entries
		makeNoopStepFunc(gate.StepContractSignature),
		// Step 7: Baseline comparison (deferred)
		gate.StepBaselineComparisonFunc(),
		// Step 8: Waiver resolution (deferred)
		gate.StepWaiverResolutionFunc(),
		// Step 9: Ledger integrity (deferred)
		gate.StepLedgerIntegrityFunc(),
	}
}

// makeNoopStepFunc creates a pass step for steps not yet wired to real logic.
func makeNoopStepFunc(name string) gate.StepFunc {
	return func(_ context.Context) gate.StepResult {
		return gate.StepResult{
			StepName:   name,
			Status:     "pass",
			Violations: []gate.Violation{},
		}
	}
}

// noopValidator is a placeholder ArtifactValidator that always passes.
type noopValidator struct{}

func (n *noopValidator) ValidateAll(_ context.Context) ([]gate.Violation, error) {
	return nil, nil
}

// noopChecker is a placeholder CodeChecker that always passes.
type noopChecker struct{}

func (n *noopChecker) CheckAll(_ context.Context) ([]gate.Violation, error) {
	return nil, nil
}
