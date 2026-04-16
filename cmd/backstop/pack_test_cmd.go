package main

import (
	"fmt"

	"github.com/bmanson/backstop-core/pkg/packval"
	"github.com/spf13/cobra"
)

func newPackTestCommand(jsonFlag *bool) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run full pack validation including fixture execution",
		Long:  "Runs all six pack validation phases, including fixture execution in phase 3.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format == "" {
				if jsonFlag != nil && *jsonFlag {
					format = "json"
				} else {
					format = "json"
				}
			}
			p := packval.NewPipeline(".", packval.PipelineOptions{Mode: "test", Format: format})
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
