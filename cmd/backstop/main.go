package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		// Check if the error carries a specific exit code
		if exitErr, ok := err.(*ExitCodeError); ok {
			fmt.Fprintln(os.Stderr, "Error:", exitErr.Message)
			os.Exit(exitErr.Code)
		}
		// Default to config error for untyped errors
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(ExitConfigError)
	}
}
