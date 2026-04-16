package main

import (
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackAddCommand(_ *bool) *cobra.Command {
	var versionFlag string

	cmd := &cobra.Command{
		Use:   "add [pack-ref]",
		Short: "Add an enforcement pack to the project",
		Long:  "Resolves, clones, validates, installs, merges config, and locks an enforcement pack. Accepts org/pack-name@version for git packs or a local filesystem path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packRef := args[0]

			opts := distribution.AddOptions{
				ProjectDir: ".",
				Version:    versionFlag,
			}

			result, err := distribution.Add(packRef, opts)
			if err != nil {
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
			}

			cmd.Printf("Added %s@%s (hash: %s)\n", result.PackName, result.Version, result.ContentHash)
			return nil
		},
	}

	cmd.Flags().StringVar(&versionFlag, "version", "", "Version to install (overrides version in pack reference)")

	return cmd
}
