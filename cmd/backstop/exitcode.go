package main

import (
	"github.com/bmanson/backstop-core/pkg/validate"
)

// Exit code constants for consistent CLI behavior.
const (
	// ExitPass indicates all checks passed or the command completed successfully.
	ExitPass = 0

	// ExitViolations indicates violations were found.
	ExitViolations = 1

	// ExitConfigError indicates a configuration error occurred.
	ExitConfigError = 2
)

// ExitWithResult determines the exit code for a command execution.
// Config error (2) takes precedence over violations (1).
func ExitWithResult(result validate.ValidationResult, configErr error) int {
	if configErr != nil {
		return ExitConfigError
	}
	if !result.Pass() {
		return ExitViolations
	}
	return ExitPass
}
