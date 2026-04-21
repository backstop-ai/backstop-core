package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmanson/backstop-core/pkg/check"
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
			return runGate(cmd, args)
		},
	}
	// No additional flags — gate accepts no scope flags (REQ-012)
	gateCmd = cmd
	return cmd
}

// runGate is the Cobra RunE handler that orchestrates all nine gate steps.
func runGate(cmd *cobra.Command, _ []string) error {
	// Load config via CLI foundation config loader.
	_, cfgErr := config.LoadConfig()

	// If config loading fails, return immediately with exit code 2.
	if cfgErr != nil {
		return &ExitCodeError{
			Code:    ExitConfigError,
			Message: fmt.Sprintf("config: %s", cfgErr),
		}
	}

	// Resolve project root from backstop.yml location.
	cfgPath, discoverErr := config.DiscoverConfigPath()
	projectRoot := "."
	if discoverErr == nil {
		projectRoot = filepath.Dir(cfgPath)
	}

	// Build gate with step implementations.
	var opts []gate.Option

	steps := buildGateSteps(projectRoot)
	opts = append(opts, gate.WithSteps(steps))

	g := gate.New(opts...)
	result, exitCode := g.Run(context.Background())

	// Format output based on --json flag.
	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
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
// implementations wired to real pkg/gate logic.
// Steps 1-2: delegate to real artifact validation and code check.
// Steps 3-6: mechanical verification using grep/AST parsing.
// Steps 7-9: deferred (baseline, waivers, ledger).
func buildGateSteps(projectRoot string) []gate.StepFunc {
	specDir := filepath.Join(projectRoot, "specs")
	packs, packErr := loadInstalledPacks(projectRoot)
	if packErr != nil {
		return []gate.StepFunc{
			func(context.Context) gate.StepResult {
				return gate.StepResult{
					StepName:   "pack_loading",
					Status:     "fail",
					ConfigErr:  true,
					Violations: []gate.Violation{{Rule: "pack_loading", Message: packErr.Error(), Severity: "error"}},
				}
			},
		}
	}

	extraSemgrepConfigs := []string{}
	if len(packs) > 0 {
		merged, mergeErr := mergePackRules(packs, filepath.Join(projectRoot, ".backstop", "packs"))
		if mergeErr != nil {
			return []gate.StepFunc{
				func(context.Context) gate.StepResult {
					return gate.StepResult{
						StepName:   "pack_rule_merge",
						Status:     "fail",
						ConfigErr:  true,
						Violations: []gate.Violation{{Rule: "pack_rule_merge", Message: mergeErr.Error(), Severity: "error"}},
					}
				},
			}
		}
		extraSemgrepConfigs = merged
	}

	// Step 1: Artifact validation — delegates to ValidateArtifacts.
	artifactValidator := &realArtifactValidator{projectRoot: projectRoot}

	// Step 2: Code check — delegates to pkg/check.Run with ScopeModeAll.
	codeChecker := &realCodeChecker{projectRoot: projectRoot, extraSemgrepConfigs: extraSemgrepConfigs}

	// Steps 3-4: Test verification and substantiveness need spec dir and code dir.
	// We use the project root as the code directory for walking test files.
	testVerifyStep := gate.StepTestVerificationFunc(specDir, projectRoot)

	// Step 4: Test substantiveness needs the resolved mandated tests with file paths.
	// We extract mandated tests and resolve their file paths, then pass to substantiveness.
	testSubstantivenessStep := buildTestSubstantivenessStep(specDir, projectRoot)

	// Step 5: Coverage threshold needs spec verifications and a command runner.
	coverageStep := buildCoverageStep(specDir, projectRoot)

	// Step 6: Contract signature needs contract entries extracted from specs.
	contractStep := buildContractStep(specDir, projectRoot)

	steps := []gate.StepFunc{
		gate.StepArtifactValidationFunc(artifactValidator),
		gate.StepCodeCheckFunc(codeChecker),
		testVerifyStep,
		testSubstantivenessStep,
		coverageStep,
		contractStep,
		// Steps 7-9: deferred
		gate.StepBaselineComparisonFunc(),
		gate.StepWaiverResolutionFunc(),
		gate.StepLedgerIntegrityFunc(),
	}

	if len(packs) == 0 {
		return steps
	}

	packNames := make([]string, 0, len(packs))
	for _, manifest := range packs {
		packNames = append(packNames, manifest.NormalizedName)
	}

	lockStep := func(context.Context) gate.StepResult {
		err := verifyPackLock(projectRoot, packNames)
		if err != nil {
			return gate.StepResult{
				StepName:   "pack_lock_verification",
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "pack_lock_verification", Message: err.Error(), Severity: "error"}},
			}
		}
		return gate.StepResult{
			StepName:   "pack_lock_verification",
			Status:     "pass",
			Violations: []gate.Violation{},
		}
	}

	packValidatorStep := func(context.Context) gate.StepResult {
		violations, err := runPackValidators(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot)
		if err != nil {
			return gate.StepResult{
				StepName:   "pack_validators",
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "pack_validators", Message: err.Error(), Severity: "error"}},
			}
		}
		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		return gate.StepResult{
			StepName:   "pack_validators",
			Status:     status,
			Violations: violations,
		}
	}

	packed := make([]gate.StepFunc, 0, len(steps)+2)
	packed = append(packed, lockStep)
	packed = append(packed, steps[0], steps[1], packValidatorStep)
	packed = append(packed, steps[2:]...)
	return packed
}

// buildTestSubstantivenessStep creates a StepFunc that extracts mandated tests
// from specs, resolves their file paths, and checks substantiveness.
func buildTestSubstantivenessStep(specDir, codeDir string) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		mandated, err := gate.ExtractMandatedTests(specDir)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: "test_substantiveness", Message: "failed to extract mandated tests: " + err.Error(), Severity: "error"}},
			}
		}

		// Resolve file paths for found tests.
		mandated = gate.ResolveMandatedTestPaths(mandated, codeDir)

		// Delegate to the real substantiveness checker.
		step := gate.StepTestSubstantivenessFunc(mandated)
		return step(ctx)
	}
}

// buildCoverageStep creates a StepFunc that extracts spec verifications
// and runs coverage checks using a real command runner.
func buildCoverageStep(specDir, projectRoot string) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		specs, err := gate.ExtractSpecVerifications(specDir)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepCoverageThreshold,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: "coverage_threshold", Message: "failed to extract spec verifications: " + err.Error(), Severity: "error"}},
			}
		}

		runner := &gate.ExecCommandRunner{Dir: projectRoot}
		step := gate.StepCoverageThresholdFunc(runner, specs)
		return step(ctx)
	}
}

// buildContractStep creates a StepFunc that extracts contract entries
// from specs and verifies them against actual code.
func buildContractStep(specDir, projectRoot string) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		contracts, err := gate.ExtractContractEntries(specDir, projectRoot)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepContractSignature,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: "contract_signature", Message: "failed to extract contracts: " + err.Error(), Severity: "error"}},
			}
		}

		step := gate.StepContractSignatureFunc(contracts)
		return step(ctx)
	}
}

// realArtifactValidator implements gate.ArtifactValidator by calling
// ValidateArtifacts with the embedded schema FS.
type realArtifactValidator struct {
	projectRoot string
}

func (v *realArtifactValidator) ValidateAll(_ context.Context) ([]gate.Violation, error) {
	cfg := ValidateConfig{
		ProjectRoot: v.projectRoot,
		All:         true,
		SchemaFS:    SchemaFS,
	}

	result, err := ValidateArtifacts(cfg)
	if err != nil {
		return nil, &gate.ConfigError{Err: err}
	}

	// Convert validate.Violation to gate.Violation.
	var violations []gate.Violation
	for _, vv := range result.Violations {
		violations = append(violations, gate.Violation{
			Rule:     vv.Rule,
			File:     vv.File,
			Message:  vv.Message,
			Severity: vv.Severity,
		})
	}
	return violations, nil
}

// realCodeChecker implements gate.CodeChecker by calling pkg/check.Run
// with ScopeModeAll.
type realCodeChecker struct {
	projectRoot         string
	extraSemgrepConfigs []string
}

func (c *realCodeChecker) CheckAll(_ context.Context) ([]gate.Violation, error) {
	cfgPath, cfgErr := config.DiscoverConfigPath()
	pRoot := c.projectRoot
	if cfgErr == nil {
		pRoot = filepath.Dir(cfgPath)
	}

	backstopDir := filepath.Join(pRoot, ".backstop")

	// Check .backstop/ directory validity. Missing .backstop is a step
	// failure, not a config error — gate should continue with other steps.
	if verr := check.ValidateBackstopDir(pRoot); verr != nil {
		return nil, verr
	}

	opts := check.Options{
		Mode:        check.ScopeModeAll,
		ManifestDir: filepath.Join(backstopDir, "rules"),
		BackstopDir: backstopDir,
		ProjectDir:  pRoot,
	}
	_ = c.extraSemgrepConfigs

	// Load config to extract semgrep version pin.
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Packs.Rules != nil {
		if v, ok := cfg.Packs.Rules["semgrep"]; ok {
			opts.PinnedSemgrepVersion = v
		}
	}

	result, runErr := check.Run(context.Background(), opts)
	if runErr != nil {
		return nil, &gate.ConfigError{Err: runErr}
	}

	// Convert check.Violation to gate.Violation.
	var violations []gate.Violation
	for _, cv := range result.AllViolations() {
		violations = append(violations, gate.Violation{
			Rule:     cv.Pass.String(),
			File:     cv.File,
			Message:  cv.Message,
			Severity: cv.Severity,
		})
	}
	return violations, nil
}

// Verify that specDir exists before building steps — avoid confusing
// errors when specs directory is missing.
func specsExist(specDir string) bool {
	info, err := os.Stat(specDir)
	return err == nil && info.IsDir()
}
