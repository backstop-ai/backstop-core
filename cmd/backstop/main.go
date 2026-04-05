package main

import "os"

func main() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		// Check if the error carries a specific exit code
		if exitErr, ok := err.(*ExitCodeError); ok {
			os.Exit(exitErr.Code)
		}
		// Default to config error for untyped errors
		os.Exit(ExitConfigError)
	}
}
