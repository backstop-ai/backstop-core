package main

import (
	"fmt"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

// formatAddedLine renders the pack-add success line. For a versionless (local) pack the
// "@version" slot is omitted so there is no bare trailing `@`; a versioned pack renders
// `<name>@<version>` as before.
func formatAddedLine(name, version, hash string) string {
	if version == "" {
		return fmt.Sprintf("Added %s (hash: %s)", name, hash)
	}
	return fmt.Sprintf("Added %s@%s (hash: %s)", name, version, hash)
}

func newPackAddCommand(_ *bool) *cobra.Command {
	var versionFlag string

	cmd := &cobra.Command{
		Use:   "add [pack-ref]",
		Short: "Add an enforcement pack to the project",
		Long:  "Resolves, clones, validates, installs, merges config, and locks an enforcement pack. Accepts org/pack-name@version for git packs or a local filesystem path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packRef := args[0]

			add, err := newProductionAddCommand()
			if err != nil {
				// A command that cannot be assembled is a mis-built binary, not a
				// finding about the operator's project, so it exits as a
				// configuration error rather than as a violation.
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			result, err := add.Run(packRef, distribution.AddOptions{
				ProjectDir: ".",
				Version:    versionFlag,
			})
			if err != nil {
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
			}

			if result.AlreadyCurrent {
				cmd.Printf("Pack %s is already installed and up to date\n", result.PackName)
				return nil
			}

			cmd.Println(formatAddedLine(result.PackName, result.Version, result.ContentHash))
			return nil
		},
	}

	cmd.Flags().StringVar(&versionFlag, "version", "", "Version to install (overrides version in pack reference)")

	return cmd
}
