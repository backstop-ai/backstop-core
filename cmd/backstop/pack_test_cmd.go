package main

import (
	"fmt"

	"github.com/backstop-ai/backstop-core/pkg/packval"
	"github.com/spf13/cobra"
)

func newPackTestCommand(jsonFlag *bool) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "test [pack-dir]",
		Short: "Run full pack validation including fixture execution",
		Long:  "Runs all six pack validation phases, including fixture execution in phase 3. Give the pack directory as an argument (e.g. `pack test ./my-pack`); defaults to the current directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Format resolution (ISSUE-032 Defect C / CLM-007): the default is human
			// TEXT; the global --json flag selects JSON. The prior dead branch assigned
			// "json" in BOTH arms.
			if format == "" {
				if jsonFlag != nil && *jsonFlag {
					format = "json"
				} else {
					format = "text"
				}
			}
			// ISSUE-049: accept an optional pack-dir path arg (default cwd), same as pack check.
			packDir := "."
			if len(args) == 1 {
				packDir = args[0]
			}
			p := packval.NewPipeline(packDir, packval.PipelineOptions{Mode: "test", Format: format})
			result := p.Run()
			out, err := packval.FormatResult(result, format)
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}
			cmd.Print(out)
			if result.Status == "fail" {
				// Explained for the same reason as pack check (SPEC-055 REQ-011 /
				// CLM-080): the report is already on stdout by the time this returns.
				return &ExitCodeError{Code: ExitViolations, Message: fmt.Sprintf("%d pack validation errors", len(result.Errors)), Explained: true}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Output format: json|text")
	return cmd
}
