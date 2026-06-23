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
	"github.com/bmanson/backstop-core/pkg/pack"
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

	allFlag, allErr := cmd.Flags().GetBool("all")
	fileValue, fileErr := cmd.Flags().GetString("file")
	if flagErr := firstNonNil(allErr, fileErr); flagErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", flagErr)}
	}
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
	jsonFlag, jsonErr := cmd.Flags().GetBool("json")
	if jsonErr != nil {
		return fmt.Errorf("reading --json flag: %w", jsonErr)
	}
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
		return nil, fmt.Errorf("pulling remote baseline: %w", err)
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
		return nil, fmt.Errorf("resolving merge-base against origin/main: %w", err)
	}
	base := strings.TrimSpace(string(baseOut))
	if base == "" {
		return nil, fmt.Errorf("empty merge-base")
	}
	diffCmd := exec.Command("git", "diff", "--name-only", base)
	diffCmd.Dir = projectRoot
	out, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing changed files against merge-base: %w", err)
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
// gateLanguage reads the project's declared language from backstop.yml so the
// bridge can select the matching native toolchain pack. An unreadable config
// defaults to the Go stack, mirroring check.Options' empty-language default.
func gateLanguage(projectRoot string) string {
	cfg, err := config.LoadConfigFromPath(filepath.Join(projectRoot, "backstop.yml"))
	if err != nil || cfg == nil {
		return "go"
	}
	if cfg.Language == "" {
		return "go"
	}
	return cfg.Language
}

// nativeToolchainPackName is the on-disk name of the reusable Go
// native-toolchain MECHANISM pack (SPEC-034 REQ-007) the bridge dispatches. It
// is a function (not a package-level var/const) to keep the bridge free of
// package-level mutable state.
func nativeToolchainPackName() string { return "backstop/go-toolchain" }

// gateConfig loads the project's backstop.yml for the gate wiring, returning a
// zero-value-safe config. An unreadable config yields a minimal config defaulting
// to the Go stack (mirroring gateLanguage), so the traceability classifier still
// derives a concrete CapabilityState rather than panicking.
func gateConfig(projectRoot string) *config.Config {
	cfg, err := config.LoadConfigFromPath(filepath.Join(projectRoot, "backstop.yml"))
	if err != nil || cfg == nil {
		return &config.Config{Language: "go"}
	}
	return cfg
}

// deriveCapabilityState computes the CapabilityState for a traceability
// dimension on the EXISTING binary from cfg.Language + baked-Go-analyzer presence
// ONLY — no pack, no engine (SPEC-036 REQ-003/CLM-029). The only traceability
// capability that exists on the existing binary is the baked Go analyzers
// (step_testverify / step_coverage / step_contract), which are Go-specific. So a
// dimension is Present/Working iff cfg.Language == "go" (the baked analyzer
// applies); for any non-Go stack the capability is Absent, so an undeclared
// dimension lands in class 2 (warn, exit 0) — not a silent pass and not a
// mis-applied Go analyzer.
func deriveCapabilityState(cfg *config.Config, dim gate.TraceabilityDimension) gate.CapabilityState {
	lang := "go"
	if cfg != nil && cfg.Language != "" {
		lang = cfg.Language
	}
	if lang == "go" {
		// The baked Go analyzer for this dimension is compiled into the binary
		// and applies to a Go project. Present and Working.
		return gate.CapabilityState{
			Present:       true,
			Working:       true,
			PackOrCommand: "the baked Go " + string(dim) + " analyzer",
		}
	}
	// Non-Go stack: no baked Go analyzer applies and no pack provides the
	// dimension on the existing binary — capability Absent.
	return gate.CapabilityState{
		Present:       false,
		Working:       false,
		PackOrCommand: "a " + lang + " " + string(dim) + " pack",
	}
}

// wrapTraceabilityStep wraps a traceability analyzer step (the delegate) with
// the SPEC-036 polarity classifier so the classifier runs IN FRONT OF the
// analyzer: it derives the CapabilityState (cfg.Language + baked-analyzer
// presence), classifies the dimension, and for class 1/2/3 returns
// PolarityStepResult WITHOUT reaching the analyzer (intercept). Only the
// none/proceed outcome (declared-and-working or undeclared-but-present) falls
// through to the unchanged delegate (REQ-008). The delegate is taken as a
// parameter so the wiring test can spy on whether the analyzer is reached
// (CLM-028).
func wrapTraceabilityStep(cfg *config.Config, dim gate.TraceabilityDimension, stepName string, delegate gate.StepFunc) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		cap := deriveCapabilityState(cfg, dim)
		class := gate.ClassifyDimension(cfg, dim, cap)
		if class != gate.ClassNone {
			// Intercept — do NOT reach the analyzer.
			return gate.PolarityStepResult(stepName, dim, class, cfg, cap)
		}
		// Fall through to the unchanged analyzer.
		return delegate(ctx)
	}
}

// loadBridgedToolchainPacks resolves the native <lang>-toolchain mechanism pack
// the bridge routes the native lint/build/test passes through (SPEC-034
// REQ-001/REQ-007). It is loaded from .backstop/packs/<name> on disk (so its
// pack-relative convert scripts resolve for SandboxedRunStdout) and dispatched
// through the SAME dispatchPackEngines step the declared packs use — NOT a
// parallel dispatcher and NOT a pkg/check->pkg/pack/engine import. The pack and
// this wiring land in lockstep (REQ-007): the bridge dispatches the pack's
// engine bindings, so there is never a commit where the bridge is live but the
// pack is absent. A missing pack directory yields no bridged packs (the bespoke
// path is still live in phase 1 — no enforcement lapse), while a PRESENT but
// malformed pack fails loud through the dispatch step like any broken pack.
func loadBridgedToolchainPacks(projectRoot, language string, declared []*pack.Manifest) ([]*pack.Manifest, error) {
	if language != "" && language != "go" {
		return nil, nil
	}
	// Dedupe: when the native toolchain pack is ALSO a declared+locked project
	// pack (as it is when backstop-core dogfoods it), the existing pack dispatch
	// already runs it — the bridge must not re-add it and double-run lint/build/
	// test. The bridge's value is dispatching the native pack even when a Go
	// project has NOT declared it; when it is declared, dispatch already covers it.
	for _, m := range declared {
		if m.NormalizedName == strings.ToLower(nativeToolchainPackName()) {
			return nil, nil
		}
	}
	packPath := filepath.Join(projectRoot, ".backstop", "packs", filepath.FromSlash(nativeToolchainPackName()))
	info, statErr := os.Stat(packPath)
	if statErr != nil || !info.IsDir() {
		return nil, nil
	}
	manifest, parseErr := pack.ParseManifestFile(filepath.Join(packPath, "pack.yml"))
	if parseErr != nil {
		return nil, fmt.Errorf("parsing native toolchain pack %s: %w", nativeToolchainPackName(), parseErr)
	}
	return []*pack.Manifest{manifest}, nil
}

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

	// The BRIDGE (SPEC-034 REQ-001): resolve the native <lang>-toolchain
	// mechanism pack so its lint/build/test engine bindings dispatch through the
	// SAME dispatchPackEngines step the declared packs use. Computed here, before
	// the no-declared-packs early return, so a Go project with no declared packs
	// still gets its native toolchain enforced via the engine path (the pack
	// engine step is built whenever there are declared OR bridged packs). A
	// malformed native pack fails loud as a single config-error step.
	bridged, bridgeErr := loadBridgedToolchainPacks(projectRoot, gateLanguage(projectRoot), packs)
	if bridgeErr != nil {
		return []gate.StepFunc{
			func(context.Context) gate.StepResult {
				return gate.StepResult{
					StepName:   "pack_loading",
					Status:     "fail",
					ConfigErr:  true,
					Violations: []gate.Violation{{Rule: "pack_loading", Message: bridgeErr.Error(), Severity: "error"}},
				}
			},
		}
	}

	// Step 1: Artifact validation — delegates to ValidateArtifacts.
	artifactValidator := &realArtifactValidator{projectRoot: projectRoot}

	// Step 2: Code check — delegates to pkg/check.Run with ScopeModeAll. Pack
	// rule findings are dispatched group-by-engine in the pack engine step below
	// (SPEC-031 REQ-011), not through an in-process rule-config feed.
	// One shared runner so the whole-module `go test ./...` executes ONCE and
	// feeds both code_check (test FAILs) and coverage_threshold (per-package
	// coverage), instead of running the suite twice (~94s of duplicate work).
	sharedTest := newSharedTestRunner(projectRoot)
	codeChecker := &realCodeChecker{projectRoot: projectRoot, sharedRunner: sharedTest}

	// Steps 3-4: Test verification and substantiveness need spec dir and code dir.
	// We use the project root as the code directory for walking test files.
	testVerifyStep := gate.StepTestVerificationScopedFunc(specDir, projectRoot, activeScope)

	// Step 4: Test substantiveness needs the resolved mandated tests with file paths.
	// We extract mandated tests and resolve their file paths, then pass to substantiveness.
	testSubstantivenessStep := buildTestSubstantivenessStep(specDir, projectRoot, activeScope)

	// Step 5: Coverage threshold needs spec verifications and a command runner.
	// It shares sharedTest so its whole-module coverage read reuses code_check's
	// already-executed `go test ./...` instead of running the suite again.
	coverageStep := buildCoverageStep(specDir, projectRoot, activeScope, sharedTest)

	// Step 6: Contract signature needs contract entries extracted from specs.
	contractStep := buildContractStep(specDir, projectRoot, activeScope)

	// SPEC-036: wrap the three traceability analyzer steps with the polarity
	// classifier so it runs IN FRONT OF each analyzer. The classifier derives the
	// CapabilityState from cfg.Language + baked-Go-analyzer presence (no pack, no
	// engine), classifies the dimension, and intercepts for class 1/2/3 (analyzer
	// not reached); only declared-and-working / undeclared-but-present fall
	// through to the UNCHANGED analyzer (REQ-008). The analyzer files themselves
	// are untouched — the wrapper intercepts at the wiring boundary.
	traceabilityCfg := gateConfig(projectRoot)
	testSubstantivenessStep = wrapTraceabilityStep(traceabilityCfg, gate.DimensionSubstantiveness, gate.StepTestSubstantiveness, testSubstantivenessStep)
	coverageStep = wrapTraceabilityStep(traceabilityCfg, gate.DimensionCoverage, gate.StepCoverageThreshold, coverageStep)
	contractStep = wrapTraceabilityStep(traceabilityCfg, gate.DimensionContracts, gate.StepContractSignature, contractStep)

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

	if len(packs) == 0 && len(bridged) == 0 {
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
		// The bridge dispatches the native toolchain pack's engine bindings
		// alongside the declared packs through the SAME substrate (SPEC-034
		// REQ-001/CLM-001/CLM-003), standing the engine path up ALONGSIDE the
		// still-live bespoke path (phase 1, no enforcement lapse).
		dispatchPacks := append(append([]*pack.Manifest{}, bridged...), packs...)
		// Split-provisioning fail-loud (SPEC-034 REQ-008): before dispatch, verify
		// the assume-present Layer-0 native tools (go/golangci-lint) resolve on PATH.
		// A missing one is a *check.ConfigError surfaced as a config-error step (exit
		// 2) naming the tool — backstop never installs it. Backstop-introduced
		// engines (semgrep) carry a Provision record and are skipped here (pinned +
		// auto-provisioned, CLM-026/CLM-027/CLM-028).
		if provErr := provisionEngines(dispatchPacks); provErr != nil {
			return gate.StepResult{
				StepName:   "pack_engines",
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "pack_engines", Message: provErr.Error(), Severity: "error"}},
			}
		}
		// Thread the gate's diff scope (activeScope) so rule-fed findings engines
		// scan only the changed files, not the whole repository (ISSUE-010). A nil
		// activeScope or GateScopeModeAll restores the whole-repo sweep; the
		// project-wide toolchain passes stay project-wide regardless. Routed through
		// the dispatchPackEnginesFn seam (the same one code check uses) so the
		// gate-wiring test can assert activeScope reaches the engine without a live
		// tool — it is the existing dispatchPackEngines, not a parallel dispatcher.
		violations, err := resolveDispatchPackEngines()(dispatchPacks, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, activeScope, runner)
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
func buildCoverageStep(specDir, projectRoot string, scope *gate.GateScope, runner gate.CommandRunner) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		specs, err := gate.ExtractSpecVerifications(specDir)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepCoverageThreshold,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: "coverage_threshold", Message: "failed to extract spec verifications: " + err.Error(), Severity: "error"}},
			}
		}

		if runner == nil {
			runner = &gate.ExecCommandRunner{Dir: projectRoot}
		}
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
	// runnerForTest is a test-only injection seam: when set, runCheck routes
	// through check.RunWith with this hermetic runner so gate-layer
	// scope-semantics tests can drive CheckScoped without shelling out to live
	// tools. Nil in production (check.Run).
	runnerForTest check.CommandRunner
	// sharedRunner, if set, is a PRODUCTION runner injected so the whole-module
	// `go test ./...` pass is shared with the coverage step (run once, not
	// twice). Non-go-test commands delegate to a plain exec, so lint/build/
	// semgrep behave exactly as the default check.Run path.
	sharedRunner check.CommandRunner
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
		return nil, fmt.Errorf("validating .backstop directory: %w", verr)
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

	// Load config to select the toolchain stack (language). The Language/Config
	// fields drive registry selection; a declared language with no toolchain
	// surfaces as a *check.ConfigError from check.Run, wrapped below into
	// gate.ConfigError for exit 2.
	cfg, err := config.LoadConfig()
	if err == nil {
		opts.Language = cfg.Language
		opts.Config = cfg
	}

	result, runErr := c.runWithOpts(ctx, opts)
	if runErr != nil {
		return nil, &gate.ConfigError{Err: runErr}
	}

	return checkViolationsToGate(result.AllViolations()), nil
}

// runWithOpts dispatches to check.Run in production, or to check.RunWith with
// the injected hermetic runner when the test seam is set. This keeps the gate's
// scope-semantics tests bounded (no live tool) while production keeps the exact
// check.Run path.
func (c *realCodeChecker) runWithOpts(ctx context.Context, opts check.Options) (*check.Result, error) {
	if c.runnerForTest != nil {
		return check.RunWith(ctx, check.RunOptions{
			Options: opts,
			Runner:  c.runnerForTest,
		})
	}
	// Production with a shared runner: route through RunWith so the whole-module
	// go test pass goes through sharedRunner (deduped with coverage).
	if c.sharedRunner != nil {
		return check.RunWith(ctx, check.RunOptions{Options: opts, Runner: c.sharedRunner})
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

// firstNonNil returns the first non-nil error from errs, or nil if all are nil.
// Used to collapse several independent flag-lookup errors into one guard.
func firstNonNil(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Verify that specDir exists before building steps — avoid confusing
// errors when specs directory is missing.
func specsExist(specDir string) bool {
	info, err := os.Stat(specDir)
	return err == nil && info.IsDir()
}
