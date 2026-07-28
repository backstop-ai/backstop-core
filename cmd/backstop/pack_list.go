package main

import (
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackListCommand(jsonFlag *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed enforcement packs",
		Long:  "Displays installed pack name, version, lock status, archetype, rule count, and scaffold count. Use --json for structured output.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := distribution.ListOptions{
				ProjectDir: ".",
				JSON:       jsonFlag != nil && *jsonFlag,
			}

			result, err := distribution.List(opts)
			if err != nil {
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
			}

			cmd.Print(result.FormattedOutput)
			return nil
		},
	}
}
