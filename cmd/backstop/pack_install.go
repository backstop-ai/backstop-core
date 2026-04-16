package main

import (
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackInstallCommand(_ *bool) *cobra.Command {
	var cacheFlag string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Restore packs from backstop.lock",
		Long:  "Clones all packs at their locked versions and verifies content hashes. Does not run validation or merge tool_config. Use --cache for offline/airgapped environments.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := distribution.InstallOptions{
				ProjectDir: ".",
				CachePath:  cacheFlag,
			}

			result, err := distribution.Install(opts)
			if err != nil {
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
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
