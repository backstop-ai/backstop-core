package main

import (
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackRemoveCommand(_ *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "remove [pack-name]",
		Short: "Remove an enforcement pack from the project",
		Long:  "Reverts pack-contributed config settings, deletes pack files, and removes entries from backstop.yml, backstop.lock, and provenance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]

			result, err := distribution.Remove(packName, distribution.RemoveOptions{
				ProjectDir: ".",
			})
			if err != nil {
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
			}

			cmd.Printf("Removed %s\n", packName)
			if len(result.Warnings) > 0 {
				for _, w := range result.Warnings {
					cmd.Printf("  warning: %s\n", w)
				}
			}
			if len(result.RevertedSettings) > 0 {
				cmd.Printf("  reverted %d settings\n", len(result.RevertedSettings))
			}

			return nil
		},
	}
}
