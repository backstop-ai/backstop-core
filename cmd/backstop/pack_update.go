package main

import (
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackUpdateCommand(_ *bool) *cobra.Command {
	var acknowledgeFlag bool

	cmd := &cobra.Command{
		Use:   "update [pack-name]",
		Short: "Update a pack to the latest compatible minor/patch version",
		Long:  "Resolves the latest compatible version, validates, runs tamper detection, and updates backstop.yml and backstop.lock with the new exact pin.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]

			opts := distribution.UpdateOptions{
				ProjectDir:  ".",
				Acknowledge: acknowledgeFlag,
			}

			result, err := distribution.Update(packName, opts)
			if err != nil {
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
			}

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
