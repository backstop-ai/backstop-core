package main

import (
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackUpdateCommand(jsonFlag *bool) *cobra.Command {
	var acknowledgeFlag bool

	cmd := &cobra.Command{
		Use:   "update [pack-name]",
		Short: "Update a pack to the latest compatible minor/patch version",
		Long:  "Resolves the latest compatible version, validates, runs tamper detection, and updates backstop.yml and backstop.lock with the new exact pin.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]

			update, err := newProductionUpdateCommand()
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			result, err := update.Run(packName, distribution.UpdateOptions{
				ProjectDir:  ".",
				Acknowledge: acknowledgeFlag,
			})
			if err != nil {
				return packLifecycleFailure(cmd.OutOrStdout(), jsonFlag, "pack update", err)
			}

			// STDERR, before EITHER outcome line — a no-op update can still have fallen
			// back on a coordinate, and that diagnostic must not be swallowed by the
			// early return below.
			renderWarnings(cmd, result.Warnings)

			if result.NoOp {
				cmd.Println(result.Message)
				return nil
			}

			cmd.Printf("Updated %s: %s -> %s\n", packName, result.OldVersion, result.NewVersion)
			return nil
		},
	}

	cmd.Flags().BoolVar(&acknowledgeFlag, "acknowledge", false,
		"Acknowledge tamper detection findings and proceed")

	return cmd
}
