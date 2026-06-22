package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/spf13/cobra"
)

// checkRunFn / loadInstalledPacksFn / dispatchPackEnginesFn are test seams:
// nil in production (the resolvers below fall back to the concrete functions),
// and overridden by tests to inject hermetic stubs. They are declared WITHOUT
// initializers so they hold no package-level mutable default — the real
// implementation is resolved lazily via the resolveXxx helpers, which keeps the
// production behavior identical while leaving an injectable hook for tests.
var (
	checkRunFn            func(context.Context, check.Options) (*check.Result, error)
	loadInstalledPacksFn  func(string) ([]*pack.Manifest, error)
	dispatchPackEnginesFn func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error)
)

// resolveCheckRun returns the injected check-run seam or the concrete check.Run.
func resolveCheckRun() func(context.Context, check.Options) (*check.Result, error) {
	if checkRunFn != nil {
		return checkRunFn
	}
	return check.Run
}

// resolveLoadInstalledPacks returns the injected pack-loader seam or the
// concrete loadInstalledPacks.
func resolveLoadInstalledPacks() func(string) ([]*pack.Manifest, error) {
	if loadInstalledPacksFn != nil {
		return loadInstalledPacksFn
	}
	return loadInstalledPacks
}

// resolveDispatchPackEngines returns the injected dispatch seam or the concrete
// dispatchPackEngines.
func resolveDispatchPackEngines() func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
	if dispatchPackEnginesFn != nil {
		return dispatchPackEnginesFn
	}
	return dispatchPackEngines
}

// codeCheckCmd is the top-level Cobra command for backstop code check,
// registered under the code namespace. It mirrors gateCmd so the command is
// addressable as a package-level symbol (SPEC-008 contract).
var codeCheckCmd *cobra.Command

// newCodeCheckCommand creates the Cobra command for backstop code check.
// This is a thin adapter — all enforcement logic lives in pkg/check.
func newCodeCheckCommand(jsonFlag *bool) *cobra.Command {
	var (
		allFlag  bool
		fileFlag string
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run implementation validation",
		Long: `Runs implementation validation passes (lint, build, test, semgrep)
against changed files by default, the full codebase with --all, or a
single file with --file. The --file mode is designed for runtime hook
dispatch with a 2-second execution budget.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Step 1: Validate flag combinations
			if fileFlag != "" && allFlag {
				return &ExitCodeError{
					Code:    ExitConfigError,
					Message: "--file and --all are mutually exclusive",
				}
			}

			// Step 2: Load backstop.yml via shared config loader
			cfg, err := config.LoadConfig()
			if err != nil {
				return &ExitCodeError{
					Code:    ExitConfigError,
					Message: fmt.Sprintf("config: %s", err),
				}
			}

			// Resolve project root from config file location
			cfgPath, cfgErr := config.DiscoverConfigPath()
			projectRoot := "."
			if cfgErr == nil {
				projectRoot = filepath.Dir(cfgPath)
			}

			// Step 3: Validate .backstop/ directory
			if verr := check.ValidateBackstopDir(projectRoot); verr != nil {
				return &ExitCodeError{
					Code:    ExitConfigError,
					Message: verr.Error(),
				}
			}

			// Step 4: Load installed packs
			packs, packsErr := resolveLoadInstalledPacks()(projectRoot)
			if packsErr != nil {
				return &ExitCodeError{
					Code:    ExitConfigError,
					Message: fmt.Sprintf("pack loading: %s", packsErr),
				}
			}

			// Step 5: Determine scope mode
			mode := check.ScopeModeDiff
			if allFlag {
				mode = check.ScopeModeAll
			} else if fileFlag != "" {
				mode = check.ScopeModeFile
			}

			// Step 6: Build check options. Pack rule findings are dispatched
			// group-by-engine in step 9 (SPEC-031 REQ-011) through the declared
			// engine path, not fed into any in-process semgrep executor.
			opts := check.Options{
				Mode:        mode,
				FilePath:    fileFlag,
				BackstopDir: filepath.Join(projectRoot, ".backstop"),
				ProjectDir:  projectRoot,
				Language:    cfg.Language,
				Config:      cfg,
			}

			// Step 7: Set 2-second timeout for --file mode
			ctx := cmd.Context()
			if mode == check.ScopeModeFile {
				opts.Timeout = 2 * time.Second
			}

			// Step 8: Run checks
			result, runErr := resolveCheckRun()(ctx, opts)
			if runErr != nil {
				// Check for config errors from the check engine
				if cfgErr, ok := runErr.(*check.ConfigError); ok {
					return &ExitCodeError{
						Code:    ExitConfigError,
						Message: cfgErr.Message,
					}
				}
				return &ExitCodeError{
					Code:    ExitConfigError,
					Message: runErr.Error(),
				}
			}

			// Step 9: Dispatch pack engines on full project (always full scope).
			// This is the consolidated group-by-engine path (SPEC-031 REQ-011):
			// it runs findings engines (semgrep, ast-grep, config-file) through
			// convert+parseSarif AND the sandbox engine through the exit-code
			// branch, replacing both the layer-2 mergePackRules feeder and the
			// layer-3 runPackValidators feeder.
			//
			// `code check` is intentionally NOT diff-scoped for pack engines
			// (CLM-008, ISSUE-010): it passes the full-scope sentinel (nil scope)
			// so rule-fed findings engines scan the whole project via the
			// projectRoot escape hatch. ONLY the gate's pack_engines step threads a
			// real diff scope. Narrowing code check here would silently drop
			// pack-rule coverage on the unchanged-but-still-relevant codebase.
			if len(packs) > 0 {
				runner := &check.ExecCommandRunner{Dir: projectRoot}
				packViolations, validatorErr := resolveDispatchPackEngines()(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, nil, runner)
				if validatorErr != nil {
					return &ExitCodeError{
						Code:    ExitConfigError,
						Message: fmt.Sprintf("pack engines: %s", validatorErr),
					}
				}
				if len(packViolations) > 0 {
					result.PassResults = append(result.PassResults, check.PassResult{
						Pass:       check.CheckTypeSemgrep,
						Violations: gateViolationsToCheck(packViolations),
					})
				}
			}

			// Step 10: Format output
			outputMode := check.OutputModeHuman
			if jsonFlag != nil && *jsonFlag {
				outputMode = check.OutputModeJSON
			}

			out, fmtErr := check.FormatResult(result, outputMode)
			if fmtErr != nil {
				return fmt.Errorf("formatting output: %w", fmtErr)
			}
			cmd.Print(out)

			// Step 11: Set exit code
			exitCode := check.DetermineExitCode(result, nil, false)
			if exitCode != 0 {
				return &ExitCodeError{
					Code:    exitCode,
					Message: fmt.Sprintf("%d violation(s) found", result.ViolationCount()),
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Check all files in the codebase")
	cmd.Flags().StringVar(&fileFlag, "file", "", "Check a single file (for hook dispatch)")

	codeCheckCmd = cmd
	return cmd
}

func gateViolationsToCheck(violations []gate.Violation) []check.Violation {
	out := make([]check.Violation, 0, len(violations))
	for _, violation := range violations {
		out = append(out, check.Violation{
			Pass:     check.CheckTypeSemgrep,
			File:     violation.File,
			Message:  violation.Message,
			Severity: violation.Severity,
		})
	}
	return out
}

func packNamesFromManifests(packs []*pack.Manifest) []string {
	names := make([]string, 0, len(packs))
	for _, manifest := range packs {
		names = append(names, manifest.NormalizedName)
	}
	return names
}
