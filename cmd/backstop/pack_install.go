package main

import (
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackInstallCommand(jsonFlag *bool) *cobra.Command {
	var cacheFlag string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Restore packs from backstop.lock",
		Long:  "Clones all packs at their locked versions and verifies content hashes. Does not run validation or merge tool_config. Use --cache for offline/airgapped environments.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			install, err := newProductionInstallCommand()
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			opts := distribution.InstallOptions{ProjectDir: "."}
			opts.CachePath = cacheFlag

			result, err := install.Run(opts)
			if err != nil {
				return packLifecycleFailure(cmd.OutOrStdout(), jsonFlag, "pack install", err)
			}

			// Surface reconciliation divergences loudly (stale lock entries, manifest
			// packs missing from the lock, absent manifest) before the installed summary,
			// so an install is never silently green over a diverged lock.
			// STDERR, not stdout: a warning is a DIAGNOSTIC, not part of this command's
			// output, and only a stream-separated assertion can tell the two apart
			// (SPEC-056 REQ-011). The "warning: " prefix is preserved verbatim — two
			// existing tests and any consumer grep match on that lowercase word.
			renderWarnings(cmd, result.Warnings)

			cmd.Printf("Installed %d packs\n", len(result.InstalledPacks))
			for _, p := range result.InstalledPacks {
				cmd.Printf("  %s\n", p)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cacheFlag, "cache", "", "Local directory to read packs from instead of cloning")

	return cmd
}
