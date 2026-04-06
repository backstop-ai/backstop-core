package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/spf13/cobra"
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

			// Step 4: Determine scope mode
			mode := check.ScopeModeDiff
			if allFlag {
				mode = check.ScopeModeAll
			} else if fileFlag != "" {
				mode = check.ScopeModeFile
			}

			// Step 5: Build check options
			opts := check.Options{
				Mode:        mode,
				FilePath:    fileFlag,
				ManifestDir: filepath.Join(projectRoot, ".backstop", "rules"),
				BackstopDir: filepath.Join(projectRoot, ".backstop"),
				ProjectDir:  projectRoot,
			}

			// Extract semgrep version pin from config
			if cfg.Packs.Rules != nil {
				if v, ok := cfg.Packs.Rules["semgrep"]; ok {
					opts.PinnedSemgrepVersion = v
				}
			}

			// Step 6: Set 2-second timeout for --file mode
			ctx := cmd.Context()
			if mode == check.ScopeModeFile {
				opts.Timeout = 2 * time.Second
			}

			// Step 7: Run checks
			result, runErr := check.Run(ctx, opts)
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

			// Step 8: Format output
			outputMode := check.OutputModeHuman
			if jsonFlag != nil && *jsonFlag {
				outputMode = check.OutputModeJSON
			}

			out, fmtErr := check.FormatResult(result, outputMode)
			if fmtErr != nil {
				return fmt.Errorf("formatting output: %w", fmtErr)
			}
			cmd.Print(out)

			// Step 9: Set exit code
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
