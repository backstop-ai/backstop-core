package main

import (
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackUpgradeCommand(jsonFlag *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade [pack-ref@version]",
		Short: "Upgrade a pack to a new major version",
		Long:  "Accepts an explicit major version target, validates, scans for new violations, generates a remediation bundle, and updates backstop.yml and backstop.lock.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packRef := args[0]

			upgrade, err := newProductionUpgradeCommand()
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			result, err := upgrade.Run(packRef, distribution.UpgradeOptions{
				ProjectDir: ".",
			})
			if err != nil {
				return packLifecycleFailure(cmd.OutOrStdout(), jsonFlag, "pack upgrade", err)
			}

			cmd.Printf("Upgraded %s -> %s\n", result.OldVersion, result.NewVersion)
			if result.RemediationBundle != "" {
				cmd.Printf("  remediation bundle: %s\n", result.RemediationBundle)
			}
			if result.BaselinedViolations > 0 {
				cmd.Printf("  baselined %d violations\n", result.BaselinedViolations)
			}

			return nil
		},
	}
}
