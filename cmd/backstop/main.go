package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd := NewRootCommand()
	// Cobra's cmd.Print/Println default to stderr when SetOut is never
	// called. Route success output to stdout so JSON consumers (the
	// runtime, scripts) see what they expect on stdout.
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	if err := rootCmd.Execute(); err != nil {
		// Check if the error carries a specific exit code
		if exitErr, ok := err.(*ExitCodeError); ok {
			// Only print error for config errors (exit 2), not violations (exit 1).
			// Violation details are already in the command's output (JSON or human).
			if exitErr.Code != ExitViolations {
				fmt.Fprintln(os.Stderr, "Error:", exitErr.Message)
			}
			os.Exit(exitErr.Code)
		}
		// Default to config error for untyped errors
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(ExitConfigError)
	}
}
