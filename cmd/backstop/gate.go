package main

import (
	"bufio"
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
	// SUBSTANTIVENESS-ONLY RE-KEY (SPEC-037 REQ-009 / CLM-035 / CLM-036). The baked
	// Go substantiveness analyzer is DELETED, so the substantiveness capability is now
	// "the substantiveness pack is INSTALLED / resolvable" — NOT cfg.Language +
	// baked-analyzer presence, and NOT a built-in tier. Present/Working iff the
	// substantiveness pack is installed (declared in backstop.yml's packs map). This
	// arm is DIMENSION-ASYMMETRIC: the coverage and contracts arms below stay on the
	// existing baked-Go keying UNCHANGED (HARD FENCE — coverage descoped, contracts
	// keeps its baked analyzer until SPEC-038/Seed 4 ships its pack).
	if dim == gate.DimensionSubstantiveness {
		if substantivenessPackInstalled(cfg) {
			return gate.CapabilityState{
				Present:       true,
				Working:       true,
				PackOrCommand: "the installed " + substantivenessPackName() + " pack",
			}
		}
		return gate.CapabilityState{
			Present:       false,
			Working:       false,
			PackOrCommand: "the " + substantivenessPackName() + " pack (install it: `backstop pack add`)",
		}
	}

	// CONTRACTS-ONLY RE-KEY (SPEC-038 REQ-015 / CLM-050 / CLM-051). The baked Go
	// contract analyzer is DELETED (REQ-001/Phase 6), so the contracts capability is
	// now "the contracts pack is INSTALLED / resolvable" — NOT cfg.Language +
	// baked-analyzer presence, and NOT a built-in tier. Present/Working iff the
	// contracts pack is installed (declared in backstop.yml's packs map). This arm is
	// DIMENSION-ASYMMETRIC: the SUBSTANTIVENESS arm above was re-keyed by Seed 3 (LEFT
	// AS-IS); the COVERAGE arm below STAYS baked-Go (coverage descoped, no pack — HARD
	// FENCE). Absent+undeclared lands class-2 (warn, exit 0) and absent+declared
	// class-3 (block) via the SPEC-036 classifier upstream.
	if dim == gate.DimensionContracts {
		if contractsPackInstalled(cfg) {
			return gate.CapabilityState{
				Present:       true,
				Working:       true,
				PackOrCommand: "the installed " + contractsPackName() + " pack",
			}
		}
		return gate.CapabilityState{
			Present:       false,
			Working:       false,
			PackOrCommand: "the " + contractsPackName() + " pack (install it: `backstop pack add`)",
		}
	}

	// COVERAGE arm — UNCHANGED baked-Go keying (the asymmetry fence; coverage descoped).
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

// substantivenessPackInstalled reports whether the substantiveness pack is INSTALLED
// (recorded in backstop.yml's packs map — a local pack records the value "local"). It
// is the installed-pack-resolvable signal the substantiveness capability keys on after
// the baked analyzer's deletion (REQ-009 / CLM-035). It reads ONLY the declaration
// surface (cfg.Packs), not the binary — the rules live in an installed pack, never
// compiled in.
func substantivenessPackInstalled(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	_, ok := cfg.Packs[substantivenessPackName()]
	return ok
}

// contractsPackName is the stable normalized name of the contracts pack (SPEC-038).
// It is the installed-pack key the contracts capability re-keys on after the baked
// analyzer's deletion. A function (not a var/const) to keep the file free of
// package-level mutable state.
func contractsPackName() string { return "backstop/contracts" }

// contractsPackInstalled reports whether the contracts pack is INSTALLED (recorded
// in backstop.yml's packs map — a local pack records the value "local"). It is the
// installed-pack-resolvable signal the contracts capability keys on after the baked
// go/parser analyzer's deletion (REQ-015 / CLM-050), MIRRORING the live
// substantivenessPackInstalled. It reads ONLY the declaration surface (cfg.Packs),
// never the binary — the contract rules live in an installed pack, never compiled in.
func contractsPackInstalled(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	_, ok := cfg.Packs[contractsPackName()]
	return ok
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
	testSubstantivenessStep := buildTestSubstantivenessStep(specDir, projectRoot, projectRoot, activeScope)

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
		// The generic pack_engines findings dispatch runs only the generic stages
		// (lint/build/test/findings). Rules bound to a dedicated-step gate_type
		// (substantiveness/contracts/coverage) are dispatched by their own gate step;
		// running them here too would scan context-free and emit garbage findings.
		dispatchPacks := excludeDedicatedStepRules(append(append([]*pack.Manifest{}, bridged...), packs...))
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

// substantivenessPackName is the stable normalized name of the substantiveness pack
// (SPEC-037). It is BOTH the routing anchor (the gate selects substantiveness findings
// out of the flat pack_engines stream by this pack's namespaced rule IDs) AND the
// installed-pack key: the substantiveness capability is INSTALLED-pack-resolvable, so
// resolveSubstantivenessPacks filters loadInstalledPacks by this name. It is a function
// (not a package-level var/const) to keep the file free of package-level mutable state.
func substantivenessPackName() string { return "backstop/substantiveness" }

// substantivenessHollowRuleID / substantivenessExtractionRuleID are the namespaced rule
// IDs the substantiveness pack declares, used by RouteSubstantivenessFindings to
// partition the flat pack_engines stream (REQ-007). Functions, not globals.
func substantivenessHollowRuleID() string {
	return pack.NamespacedRuleID(substantivenessPackName(), "hollow-test-go")
}

func substantivenessExtractionRuleID() string {
	return pack.NamespacedRuleID(substantivenessPackName(), "referenced-symbol-go")
}

// resolveSubstantivenessPacksFn is a test seam: nil in production (the resolver below
// returns the INSTALLED substantiveness pack manifest set), overridden by the wiring
// tests that inject a manifest set so the dispatch-seam spy can observe routing without
// a real on-disk install. Declared WITHOUT an initializer so it holds no package-level
// mutable default — production resolves lazily via resolveSubstantivenessPacks.
var resolveSubstantivenessPacksFn func(projectRoot string) ([]*pack.Manifest, error)

// resolveSubstantivenessPacks returns the substantiveness pack manifest set the
// substantiveness step dispatches. In production it filters the INSTALLED packs
// (loadInstalledPacks) to the substantiveness pack — the capability is
// installed-pack-resolvable, NOT built into the binary and NOT resolved from testdata
// (REQ-009 / CLM-030 / CLM-035). A test seam may override it. An empty result means the
// substantiveness pack is not installed; the caller treats that as a capability-absent
// no-op (no findings, no violation), which the capability classifier governs upstream.
func resolveSubstantivenessPacks(projectRoot string) ([]*pack.Manifest, error) {
	if resolveSubstantivenessPacksFn != nil {
		return resolveSubstantivenessPacksFn(projectRoot)
	}
	installed, err := loadInstalledPacks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving substantiveness packs: loading installed packs: %w", err)
	}
	out := make([]*pack.Manifest, 0, 1)
	for _, m := range installed {
		if m.NormalizedName == substantivenessPackName() {
			out = append(out, m)
		}
	}
	return out, nil
}

// buildTestSubstantivenessStep re-implements the substantiveness step (SPEC-037 REQ-005)
// to CONSUME the substantiveness pack's Q1 findings + Q2 extraction through the EXISTING
// resolveDispatchPackEngines() / dispatchPackEnginesFn seam (the same dispatcher code
// check and the pack_engines step use) — NOT a re-implemented dispatcher and NOT the
// deleted go/parser analyzer. It then routes the flat []gate.Violation result by
// namespaced rule ID, keys extraction findings per mandated test, and runs the
// language-agnostic gate set-join (NoTargetViolation) + hollow → test_substantiveness
// conversion. A spy on the real dispatchPackEnginesFn seam mechanically proves the step
// reached the dispatcher (CLM-015..017). When the substantiveness pack is not installed,
// the step is a no-op (the capability classifier governs the warn/block polarity).
func buildTestSubstantivenessStep(specDir, codeDir, projectRoot string, scope *gate.GateScope) gate.StepFunc {
	return func(_ context.Context) gate.StepResult {
		mandated, err := gate.ExtractMandatedTests(specDir)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "failed to extract mandated tests: " + err.Error(), Severity: "error"}},
			}
		}

		// Resolve file paths for found tests (the keying join's FilePath side).
		mandated = gate.ResolveMandatedTestPaths(mandated, codeDir)

		packs, err := resolveSubstantivenessPacks(projectRoot)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "resolving substantiveness pack: " + err.Error(), Severity: "error"}},
			}
		}
		if len(packs) == 0 {
			// The substantiveness pack is not installed — no-op (capability-absent is
			// governed by the SPEC-036 classifier upstream, not a silent pass here).
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "pass",
				Violations: []gate.Violation{},
			}
		}

		// Reach the substantiveness pack's findings + extraction through the SAME
		// dispatch seam code check and the pack_engines step use (REQ-005). NOT a
		// re-implemented dispatcher; NOT the deleted analyzer.
		runner := &check.ExecCommandRunner{Dir: projectRoot}
		flat, err := resolveDispatchPackEngines()(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, scope, runner)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "dispatching substantiveness pack: " + err.Error(), Severity: "error"}},
			}
		}

		// Route the flat stream by namespaced rule ID (no gate_type field exists).
		hollow, extraction := gate.RouteSubstantivenessFindings(flat, substantivenessHollowRuleID(), substantivenessExtractionRuleID())

		var violations []gate.Violation
		// Q1 hollow → one test_substantiveness violation per routed hollow finding,
		// scope-filtered so out-of-scope test files are suppressed (REQ-008/CLM-029).
		for _, v := range gate.HollowFindingsToViolations(hollow) {
			if scope != nil && scope.Mode != gate.GateScopeModeAll && v.File != "" && !scope.Contains(v.File) {
				continue
			}
			violations = append(violations, v)
		}

		// Q2 noTarget set-join, keyed per mandated test from the extraction findings.
		for _, mt := range mandated {
			if mt.FilePath == "" {
				continue // not found — already reported by the verification step
			}
			if scope != nil && scope.Mode != gate.GateScopeModeAll && !scope.Contains(mt.FilePath) {
				continue
			}
			referenced := gate.ReferencedSetForTest(extraction, mt)
			samePackage := goFilePackageMatchesTarget(mt.FilePath, mt.TargetPkg)
			if v, raised := gate.NoTargetViolation(mt.FuncName, mt.TargetPkg, referenced, samePackage); raised {
				v.File = mt.FilePath
				violations = append(violations, v)
			}
		}

		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		if violations == nil {
			violations = []gate.Violation{}
		}
		return gate.StepResult{
			StepName:   gate.StepTestSubstantiveness,
			Status:     status,
			Violations: violations,
		}
	}
}

// goFilePackageMatchesTarget reads a Go test file's `package <name>` clause as a string
// and reports whether it identifies the target package (allowing the `_test` external
// variant) — the language-agnostic samePackage derivation the gate set-join consumes,
// mirroring the deleted analyzer's same-package short-circuit WITHOUT an AST walk.
func goFilePackageMatchesTarget(filePath, targetPkg string) bool {
	if targetPkg == "" {
		return false
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "package ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, "package "))
		if i := strings.IndexAny(name, " \t/"); i >= 0 {
			name = name[:i]
		}
		return name == targetPkg || name == targetPkg+"_test" || strings.TrimSuffix(name, "_test") == targetPkg
	}
	return false
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

// contractEngineResultsFn is a test seam: nil in production (the producer below
// runs the PACK path — real ast-grep signature presence + real grep symbol
// absence over the in-repo traceability pack — to build one ContractEngineResult
// per ContractEntry), overridden by the wiring/E2E tests so a spy can observe that
// buildContractStep consumes the PACK-produced results rather than the deleted
// go/parser analyzer (REQ-006/CLM-020/021/022). Declared WITHOUT an initializer so
// it holds no package-level mutable default.
var contractEngineResultsFn func(projectRoot string, contracts []gate.ContractEntry) ([]gate.ContractEngineResult, error)

// produceContractEngineResults builds the []ContractEngineResult buildContractStep
// verdicts off. In production it runs the PACK path (dispatchContractEntry ->
// resolveDispatchPackEngines -> sandboxed convert -> SARIF) over
// each declared contract entry; it NEVER routes to the deleted go/parser analyzer
// (CLM-021). A test seam may override it so the wiring spy can assert the pack path
// is the one consumed (CLM-020) and an unwired path fails (CLM-022).
func produceContractEngineResults(projectRoot string, contracts []gate.ContractEntry) ([]gate.ContractEngineResult, error) {
	if contractEngineResultsFn != nil {
		return contractEngineResultsFn(projectRoot, contracts)
	}
	// Resolve the contracts pack from the INSTALLED declaration (loadInstalledPacks) —
	// NOT from testdata and NOT from a baked path. When the pack is not installed, the
	// step produces NO results — a capability-absent no-op the SPEC-036 classifier
	// governs upstream (warn/exit 0 undeclared, block declared), which keeps an
	// uninstalled workspace from passing vacuously (REQ-014/CLM-048).
	packs, err := resolveContractsPacks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving contracts pack: %w", err)
	}
	if len(packs) == 0 {
		return []gate.ContractEngineResult{}, nil
	}
	manifest := packs[0]

	results := make([]gate.ContractEngineResult, 0, len(contracts))
	for _, c := range contracts {
		r, err := dispatchContractEntry(projectRoot, manifest, c)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// resolveContractsPacksFn is a test seam: nil in production (resolveContractsPacks below
// filters the INSTALLED packs to the contracts pack), overridable by tests. Declared
// WITHOUT an initializer so it holds no package-level mutable default.
var resolveContractsPacksFn func(projectRoot string) ([]*pack.Manifest, error)

// resolveContractsPacks returns the contracts pack manifest set the contract step
// dispatches. In production it filters the INSTALLED packs (loadInstalledPacks) to the
// contracts pack — the capability is installed-pack-resolvable, NOT built into the binary
// and NOT resolved from testdata (REQ-013). An empty result means the pack is not
// installed (capability-absent, governed upstream).
func resolveContractsPacks(projectRoot string) ([]*pack.Manifest, error) {
	if resolveContractsPacksFn != nil {
		return resolveContractsPacksFn(projectRoot)
	}
	installed, err := loadInstalledPacks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading installed packs: %w", err)
	}
	out := make([]*pack.Manifest, 0, 1)
	for _, m := range installed {
		if m.NormalizedName == contractsPackName() {
			out = append(out, m)
		}
	}
	return out, nil
}

// dispatchContractEntry runs ONE contract entry through the REAL pack-engine dispatch
// seam (resolveDispatchPackEngines → dispatchPackEngines → runFindingsEngine), so the
// engine command clears the trusted-tool allowlist (CheckToolAllowed — REQ-005) and the
// convert script runs under the REAL sandbox (packval.SandboxedRunStdout — CLM-049). It
// builds a SINGLE-RULE in-memory manifest from the contracts pack's DECLARED engines: a
// signature entry compiles its Signature via the pack's compiler script (the COMPILER
// stays pack-side — the binary never compiles a pattern) and feeds the COMPILED pattern
// as the pattern-arg of the declared ast-grep engine; an absence entry feeds the forbidden
// symbol as the pattern-arg of the declared grep engine. Matched = the dispatch produced a
// finding for that contract's file; Scanned = the declared file/scope exists on disk (the
// file-scanned guard signal). NO raw exec, NO sandbox bypass.
func dispatchContractEntry(projectRoot string, manifest *pack.Manifest, c gate.ContractEntry) (gate.ContractEngineResult, error) {
	// Determine the scan target (file for a present signature; file-OR-path scope for an
	// absence) and the file-scanned guard signal.
	target := c.File
	if c.Absent && c.Scope != "" {
		target = c.Scope
	}
	if _, statErr := os.Stat(target); statErr != nil {
		// Unscanned/missing scope — no probe runs; the gate verdict raises a loud config
		// error for an absence entry (file-scanned guard) and a no-match for a present one.
		return gate.ContractEngineResult{Entry: c, Matched: false, Scanned: false}, nil
	}

	// Build the single rule the dispatch runs for this contract.
	var rule pack.Rule
	if c.Absent {
		rule = pack.Rule{ID: "contract-absence", Engine: "grep", Pattern: c.Name}
	} else {
		pattern, compileErr := compileContractSignature(projectRoot, manifest, c.Signature)
		if compileErr != nil {
			return gate.ContractEngineResult{}, fmt.Errorf("compiling signature for %s: %w", c.Name, compileErr)
		}
		rule = pack.Rule{ID: "contract-signature", Engine: contractSignatureEngine(manifest), Pattern: pattern}
	}

	// A single-rule manifest carrying the SAME declared engines as the installed pack, so
	// dispatch resolves the pack's engine bindings + convert scripts from packRoot.
	single := &pack.Manifest{
		Name:           manifest.Name,
		NormalizedName: manifest.NormalizedName,
		Language:       manifest.Language,
		Engines:        manifest.Engines,
		Content:        pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{rule}}},
	}

	// Scope the dispatch to exactly this contract's target via a FILE-mode gate scope, so
	// the engine scans only the declared file/path.
	scope, scopeErr := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{target})
	if scopeErr != nil {
		return gate.ContractEngineResult{}, fmt.Errorf("computing scope for %s: %w", c.Name, scopeErr)
	}

	runner := &check.ExecCommandRunner{Dir: projectRoot}
	packDir := filepath.Join(projectRoot, ".backstop", "packs")
	violations, dispErr := resolveDispatchPackEngines()([]*pack.Manifest{single}, packDir, projectRoot, scope, runner)
	if dispErr != nil {
		return gate.ContractEngineResult{}, fmt.Errorf("dispatching contract engine for %s: %w", c.Name, dispErr)
	}

	locs := make([]gate.SarifLocation, 0, len(violations))
	for _, v := range violations {
		locs = append(locs, gate.SarifLocation{File: v.File})
	}
	return gate.ContractEngineResult{Entry: c, Matched: len(violations) > 0, Scanned: true, Locations: locs}, nil
}

// contractSignatureEngine returns the pattern-arg ast-grep engine name the contracts pack
// declares for signature presence. It prefers a pack-declared "ast-grep-contracts" engine
// (the traceability dispatch pack) and falls back to the built-in "ast-grep" (the
// installable packs/contracts/ pack, which rides the DefaultRegistry ast-grep binding).
func contractSignatureEngine(manifest *pack.Manifest) string {
	if _, ok := manifest.Engines["ast-grep-contracts"]; ok {
		return "ast-grep-contracts"
	}
	return "ast-grep"
}

// compileContractSignature invokes the contracts pack's pack-relative signature compiler
// (scripts/compile-signature.sh) to turn the declared human-readable Signature into an
// ast-grep pattern. The COMPILER is pack-side (CLM-006): the binary shells the pack script
// and reads back the pattern; it never compiles or renders a signature itself. The
// resulting pattern is fed to the real ast-grep dispatch as the pattern-arg.
func compileContractSignature(projectRoot string, manifest *pack.Manifest, signature string) (string, error) {
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", filepath.FromSlash(manifest.NormalizedName))
	script := filepath.Join(packRoot, "scripts", "compile-signature.sh")
	out, err := exec.Command("/bin/sh", script, signature).Output()
	if err != nil {
		return "", fmt.Errorf("running pack signature compiler %s: %w", script, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// buildContractStep extracts contract entries from specs, routes them to the
// PACK-produced contracts SARIF path (ast-grep signature + grep absence) to build
// []ContractEngineResult, then feeds the rewritten gate.StepContractSignatureScopedFunc
// pack-SARIF consumer (REQ-006). It MUST NOT route to the deleted go/parser analyzer.
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

		results, err := produceContractEngineResults(projectRoot, contracts)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepContractSignature,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "contract_signature", Message: "dispatching contract pack: " + err.Error(), Severity: "error"}},
			}
		}

		step := gate.StepContractSignatureScopedFunc(results, scope)
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
