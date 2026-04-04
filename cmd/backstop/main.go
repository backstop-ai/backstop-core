package main

import "os"

func main() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		// Determine exit code from error context
		os.Exit(ExitConfigError)
	}
}
