package main

import (
	"fmt"

	"github.com/bmanson/backstop-core/pkg/packval"
	"github.com/spf13/cobra"
)

func newPackCheckCommand(jsonFlag *bool) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate pack manifest and constraints",
		Long:  "Runs pack validation phases 1,2,4,5,6 (manifest and metadata checks) and returns structured output.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Format resolution (ISSUE-032 Defect C / CLM-007): the default is human
			// TEXT; the global --json flag (shared jsonFlag pointer) selects JSON. The
			// prior dead branch assigned "json" in BOTH arms, so --json had no effect on
			// the default and the human default was never text.
			if format == "" {
				if jsonFlag != nil && *jsonFlag {
					format = "json"
				} else {
					format = "text"
				}
			}
			p := packval.NewPipeline(".", packval.PipelineOptions{Mode: "check", Format: format})
			result := p.Run()
			out, err := packval.FormatResult(result, format)
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}
			cmd.Print(out)
			if result.Status == "fail" {
				return &ExitCodeError{Code: ExitViolations, Message: fmt.Sprintf("%d pack validation errors", len(result.Errors))}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Output format: json|text")
	return cmd
}
