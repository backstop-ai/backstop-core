package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/spf13/cobra"
)

var (
	checkRunFn           = check.Run
	loadInstalledPacksFn = loadInstalledPacks
	mergePackRulesFn     = mergePackRules
	runPackValidatorsFn  = runPackValidators
)

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
			packs, packsErr := loadInstalledPacksFn(projectRoot)
			if packsErr != nil {
				return &ExitCodeError{
					Code:    ExitConfigError,
					Message: fmt.Sprintf("pack loading: %s", packsErr),
				}
			}

			extraSemgrepConfigs := []string{}
			if len(packs) > 0 {
				configs, mergeErr := mergePackRulesFn(packs, filepath.Join(projectRoot, ".backstop", "packs"))
				if mergeErr != nil {
					return &ExitCodeError{
						Code:    ExitConfigError,
						Message: fmt.Sprintf("pack rules: %s", mergeErr),
					}
				}
				extraSemgrepConfigs = configs
			}

			// Step 5: Determine scope mode
			mode := check.ScopeModeDiff
			if allFlag {
				mode = check.ScopeModeAll
			} else if fileFlag != "" {
				mode = check.ScopeModeFile
			}

			// Step 6: Build check options
			opts := check.Options{
				Mode:                mode,
				FilePath:            fileFlag,
				ManifestDir:         filepath.Join(projectRoot, ".backstop", "rules"),
				BackstopDir:         filepath.Join(projectRoot, ".backstop"),
				ProjectDir:          projectRoot,
				ExtraSemgrepConfigs: extraSemgrepConfigs,
			}

			// Extract semgrep version pin from config
			if cfg.Enforcement.SemgrepVersion != "" {
				opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion
			}

			// Step 7: Set 2-second timeout for --file mode
			ctx := cmd.Context()
			if mode == check.ScopeModeFile {
				opts.Timeout = 2 * time.Second
			}

			// Step 8: Run checks
			result, runErr := checkRunFn(ctx, opts)
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

			// Step 9: Run pack validators on full project (always full scope).
			if len(packs) > 0 {
				packViolations, validatorErr := runPackValidatorsFn(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot)
				if validatorErr != nil {
					return &ExitCodeError{
						Code:    ExitConfigError,
						Message: fmt.Sprintf("pack validators: %s", validatorErr),
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
