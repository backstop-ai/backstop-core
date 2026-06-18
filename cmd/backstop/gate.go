package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
func newGateCommand(jsonFlag *bool) *cobra.Command {
	var allFlag bool
	var fileFlag string
	cmd := &cobra.Command{
		Use:   "gate [--all | --file FILE [FILE...]]",
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
	cmd.Flags().BoolVar(&allFlag, "all", false, "run the full project sweep")
	cmd.Flags().StringVar(&fileFlag, "file", "", "scope gate to one or more explicit files")
	gateCmd = cmd
	return cmd
}

// runGate is the Cobra RunE handler that orchestrates all nine gate steps.
func runGate(cmd *cobra.Command, args []string) error {
	// Load config via CLI foundation config loader.
	cfg, cfgErr := config.LoadConfig()

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

	allFlag, _ := cmd.Flags().GetBool("all")
	fileValue, _ := cmd.Flags().GetString("file")
	if allFlag && fileValue != "" {
		return &ExitCodeError{Code: ExitConfigError, Message: "config: --all and --file are mutually exclusive"}
	}

	scopeMode := gate.GateScopeModeDiff
	explicitFiles := []string{}
	if allFlag {
		scopeMode = gate.GateScopeModeAll
	}
	if fileValue != "" {
		scopeMode = gate.GateScopeModeFile
		explicitFiles = append([]string{fileValue}, args...)
	} else if len(args) > 0 {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: unexpected gate arguments: %s", strings.Join(args, " "))}
	}

	scope, scopeErr := gate.ComputeGateScope(projectRoot, scopeMode, explicitFiles)
	if scopeErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", scopeErr)}
	}

	// Build gate with step implementations.
	var opts []gate.Option

	steps := buildGateSteps(projectRoot, scope)
	opts = append(opts, gate.WithSteps(steps))
	opts = append(opts, gate.WithScope(scope))

	baselinePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	ttl, ttlErr := cfg.BaselineTTLDuration()
	if ttlErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %v", ttlErr)}
	}
	baselineArtifact, baselineWarning, baselineModTime := resolveBaselineCache(baselinePath, ttl)
	opts = append(opts, gate.WithBaseline(baselineArtifact), gate.WithBaselineWarning(baselineWarning), gate.WithBaselineCacheMeta(baselinePath, ttl, baselineModTime))
	allowSeeding, changedFiles := ruleSetChangeSeedingContext(projectRoot, scope)
	opts = append(opts, gate.WithRuleSetChangeSeedingAllowed(allowSeeding), gate.WithRuleSetChangeFiles(changedFiles))

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

func resolveBaselineCache(path string, ttl time.Duration) (*gate.BaselineArtifact, string, time.Time) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		refreshed, refreshErr := refreshBaselineFromRemote(path)
		if refreshErr == nil {
			return refreshed, "baseline cache fetched from remote main baseline artifact", time.Now()
		}
		return nil, fmt.Sprintf("baseline unavailable: no cached baseline found at .backstop/baseline.json and remote baseline fetch failed (%v); run `backstop baseline pull`", refreshErr), time.Time{}
	}
	baseline, loadErr := gate.LoadBaseline(path)
	if loadErr != nil {
		refreshed, refreshErr := refreshBaselineFromRemote(path)
		if refreshErr == nil {
			return refreshed, "baseline cache was unreadable and refreshed from remote main baseline artifact", time.Now()
		}
		return nil, fmt.Sprintf("baseline unavailable: cached baseline at .backstop/baseline.json is unreadable (%v) and remote baseline fetch failed (%v); run `backstop baseline pull`", loadErr, refreshErr), info.ModTime()
	}
	if time.Since(info.ModTime()) <= ttl {
		return baseline, "", info.ModTime()
	}
	refreshed, refreshErr := refreshBaselineFromRemote(path)
	if refreshErr == nil {
		return refreshed, "baseline cache refreshed from remote main baseline artifact", time.Now()
	}
	return baseline, fmt.Sprintf("baseline refresh failed; using stale cached baseline from .backstop/baseline.json (%v)", refreshErr), info.ModTime()
}

func refreshBaselineFromRemote(path string) (*gate.BaselineArtifact, error) {
	if err := runBaselinePull(nil, nil); err != nil {
		return nil, err
	}
	artifact, err := gate.LoadBaseline(path)
	if err != nil {
		return nil, fmt.Errorf("refreshed baseline cache is unreadable: %w", err)
	}
	return artifact, nil
}

func ruleSetChangeSeedingContext(projectRoot string, scope *gate.GateScope) (bool, []string) {
	if scope == nil || scope.Mode != gate.GateScopeModeAll {
		return false, nil
	}
	changed, err := changedFilesAgainstOriginMain(projectRoot)
	if err != nil {
		return false, nil
	}
	for _, file := range changed {
		if file == "backstop.yml" || file == "backstop.lock" || strings.HasPrefix(file, ".backstop/packs/") || strings.HasPrefix(file, ".backstop/rules/") {
			return true, changed
		}
	}
	return false, changed
}

func changedFilesAgainstOriginMain(projectRoot string) ([]string, error) {
	baseCmd := exec.Command("git", "merge-base", "HEAD", "origin/main")
	baseCmd.Dir = projectRoot
	baseOut, err := baseCmd.Output()
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(string(baseOut))
	if base == "" {
		return nil, fmt.Errorf("empty merge-base")
	}
	diffCmd := exec.Command("git", "diff", "--name-only", base)
	diffCmd.Dir = projectRoot
	out, err := diffCmd.Output()
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files, nil
}

// buildGateSteps constructs the nine ordered step functions with concrete
// implementations wired to real pkg/gate logic.
// Steps 1-2: delegate to real artifact validation and code check.
// Steps 3-6: mechanical verification using grep/AST parsing.
// Steps 7-9: deferred (baseline, waivers, ledger).
func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc {
	var activeScope *gate.GateScope
	if len(scope) > 0 {
		activeScope = scope[0]
	}
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

	// Step 1: Artifact validation — delegates to ValidateArtifacts.
	artifactValidator := &realArtifactValidator{projectRoot: projectRoot}

	// Step 2: Code check — delegates to pkg/check.Run with ScopeModeAll. Pack
	// rule findings no longer feed the in-process semgrepExecutor via
	// ExtraSemgrepConfigs; they are dispatched group-by-engine in the pack
	// engine step below (SPEC-031 REQ-011).
	codeChecker := &realCodeChecker{projectRoot: projectRoot}

	// Steps 3-4: Test verification and substantiveness need spec dir and code dir.
	// We use the project root as the code directory for walking test files.
	testVerifyStep := gate.StepTestVerificationScopedFunc(specDir, projectRoot, activeScope)

	// Step 4: Test substantiveness needs the resolved mandated tests with file paths.
	// We extract mandated tests and resolve their file paths, then pass to substantiveness.
	testSubstantivenessStep := buildTestSubstantivenessStep(specDir, projectRoot, activeScope)

	// Step 5: Coverage threshold needs spec verifications and a command runner.
	coverageStep := buildCoverageStep(specDir, projectRoot, activeScope)

	// Step 6: Contract signature needs contract entries extracted from specs.
	contractStep := buildContractStep(specDir, projectRoot, activeScope)

	steps := []gate.StepFunc{
		gate.StepArtifactValidationScopedFunc(artifactValidator, activeScope),
		gate.StepCodeCheckScopedFunc(codeChecker, activeScope),
		testVerifyStep,
		testSubstantivenessStep,
		coverageStep,
		contractStep,
		// Steps 7-9: deferred
		gate.StepBaselineComparisonScopedFunc(activeScope),
		gate.StepWaiverResolutionScopedFunc(activeScope),
		gate.StepLedgerIntegrityScopedFunc(activeScope),
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
		runner := &check.ExecCommandRunner{Dir: projectRoot}
		violations, err := dispatchPackEngines(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, runner)
		if err != nil {
			return gate.StepResult{
				StepName:   "pack_engines",
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "pack_engines", Message: err.Error(), Severity: "error"}},
			}
		}
		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		return gate.StepResult{
			StepName:   "pack_engines",
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
func buildTestSubstantivenessStep(specDir, codeDir string, scope *gate.GateScope) gate.StepFunc {
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
		step := gate.StepTestSubstantivenessScopedFunc(mandated, scope)
		return step(ctx)
	}
}

// buildCoverageStep creates a StepFunc that extracts spec verifications
// and runs coverage checks using a real command runner.
func buildCoverageStep(specDir, projectRoot string, scope *gate.GateScope) gate.StepFunc {
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
		step := gate.StepCoverageThresholdScopedFunc(runner, specs, scope)
		return step(ctx)
	}
}

// buildContractStep creates a StepFunc that extracts contract entries
// from specs and verifies them against actual code.
func buildContractStep(specDir, projectRoot string, scope *gate.GateScope) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		contracts, err := gate.ExtractContractEntries(specDir, projectRoot)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepContractSignature,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: "contract_signature", Message: "failed to extract contracts: " + err.Error(), Severity: "error"}},
			}
		}

		step := gate.StepContractSignatureScopedFunc(contracts, scope)
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

// realCodeChecker implements gate.CodeChecker by calling pkg/check.Run.
type realCodeChecker struct {
	projectRoot string
	// runnerForTest / ensurerForTest are test-only injection seams: when set,
	// runCheck routes through check.RunWith with these hermetic dependencies so
	// gate-layer scope-semantics tests can drive CheckScoped without shelling
	// out to live tools or installing semgrep. Nil in production (check.Run).
	runnerForTest  check.CommandRunner
	ensurerForTest check.SemgrepEnsurer
}

func (c *realCodeChecker) CheckAll(_ context.Context) ([]gate.Violation, error) {
	return c.runCheck(context.Background(), check.ScopeModeAll, nil)
}

func (c *realCodeChecker) CheckScoped(ctx context.Context, scope *gate.GateScope) ([]gate.Violation, error) {
	if scope == nil || scope.Mode == gate.GateScopeModeAll {
		return c.runCheck(ctx, check.ScopeModeAll, nil)
	}
	// Carry the ENTIRE scoped file list through ONE runCheck — no per-file loop.
	// Per-pass ScopeKind then shapes the arg list: lint gets all scoped files in
	// one invocation; build/typecheck runs project-wide once ignoring the list;
	// test is dependency-mapped once with full-suite fallback. The old per-file
	// loop invoked lint N×(1 file) and build N× project-wide, which both
	// violated the per-pass scope semantics (Constraint 2/3, CLM-008).
	return c.runCheck(ctx, check.ScopeModeDiff, scope.Files)
}

func (c *realCodeChecker) runCheck(ctx context.Context, mode check.ScopeMode, files []string) ([]gate.Violation, error) {
	// The checker is bound to the project root it was constructed with;
	// CWD-based discovery is only a fallback for an unset root. Re-discovering
	// from CWD here would silently retarget the check at whatever repo the
	// process happens to run inside (and lets gate tests recurse into the
	// host repo's own test suite).
	pRoot := c.projectRoot
	if pRoot == "" || pRoot == "." {
		if cfgPath, cfgErr := config.DiscoverConfigPath(); cfgErr == nil {
			pRoot = filepath.Dir(cfgPath)
		}
	}

	backstopDir := filepath.Join(pRoot, ".backstop")

	// Check .backstop/ directory validity. Missing .backstop is a step
	// failure, not a config error — gate should continue with other steps.
	if verr := check.ValidateBackstopDir(pRoot); verr != nil {
		return nil, verr
	}

	opts := check.Options{
		Mode:        mode,
		BackstopDir: backstopDir,
		ProjectDir:  pRoot,
	}
	// An explicit scoped file list (diff/file gate scope) is carried via the
	// Files branch — a SINGLE Run covering all scoped files, not a per-file
	// loop. Paths are project-relative as they arrive from the gate scope.
	if len(files) > 0 {
		opts.Files = files
	}

	// Load config to select the toolchain stack (language) and extract the
	// semgrep version pin. The Language/Config fields drive registry selection;
	// a declared language with no toolchain surfaces as a *check.ConfigError
	// from check.Run, wrapped below into gate.ConfigError for exit 2.
	cfg, err := config.LoadConfig()
	if err == nil {
		opts.Language = cfg.Language
		opts.Config = cfg
		if cfg.Enforcement.SemgrepVersion != "" {
			opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion
		}
	}

	result, runErr := c.runWithOpts(ctx, opts)
	if runErr != nil {
		return nil, &gate.ConfigError{Err: runErr}
	}

	return checkViolationsToGate(result.AllViolations()), nil
}

// runWithOpts dispatches to check.Run in production, or to check.RunWith with
// the injected hermetic runner/ensurer when the test seams are set. This keeps
// the gate's scope-semantics tests bounded (no live tool, no semgrep install)
// while production keeps the exact check.Run path.
func (c *realCodeChecker) runWithOpts(ctx context.Context, opts check.Options) (*check.Result, error) {
	if c.runnerForTest != nil || c.ensurerForTest != nil {
		return check.RunWith(ctx, check.RunOptions{
			Options:        opts,
			Runner:         c.runnerForTest,
			SemgrepEnsurer: c.ensurerForTest,
		})
	}
	return check.Run(ctx, opts)
}

// checkViolationsToGate converts check.Violations to gate.Violations, carrying
// the structured rule ID across the bridge. A pack-namespaced semgrep check_id
// (pack.NamespacedRuleID format "org/pack/rule-id") is preserved on gate
// Violation.Rule; SourcePack is derived as everything before the LAST "/" —
// the two-segment pack NormalizedName "org/pack" — matching the layer-3
// convention at pack_gate.go (SourcePack = manifest.NormalizedName). When a
// violation carries no Rule (built-in lint/build/test passes), Rule falls back
// to the pass name and SourcePack is empty.
func checkViolationsToGate(cvs []check.Violation) []gate.Violation {
	var violations []gate.Violation
	for _, cv := range cvs {
		rule := cv.Rule
		sourcePack := ""
		if rule == "" {
			rule = cv.Pass.String()
		} else if idx := strings.LastIndex(rule, "/"); idx >= 0 {
			sourcePack = rule[:idx]
		}
		violations = append(violations, gate.Violation{
			Rule:     rule,
			File:     cv.File,
			Message:  cv.Message,
			Severity: cv.Severity,
			// ProjectWide is set from the originating CheckType, INDEPENDENT of
			// the parser-populated Rule: the build pass (Go go-build and TS tsc/
			// typecheck) runs project-wide, so its violations are exempt from
			// gate scope-filtering even when Rule is non-empty (e.g. "TS2304").
			// Keying off cv.Pass — not the Rule string — is what makes the
			// exemption correct for the tsc/sarif parsers (Constraint 3).
			ProjectWide: cv.Pass == check.CheckTypeBuild,
			SourcePack:  sourcePack,
		})
	}
	return violations
}

// Verify that specDir exists before building steps — avoid confusing
// errors when specs directory is missing.
func specsExist(specDir string) bool {
	info, err := os.Stat(specDir)
	return err == nil && info.IsDir()
}
