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
			for _, w := range result.Warnings {
				cmd.Printf("warning: %s\n", w)
			}

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
