package main

import (
	"errors"
	"fmt"
	"io"
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
		os.Exit(reportError(os.Stderr, err))
	}
}

// reportError renders a failed command's diagnostic to w and returns the process exit
// code (SPEC-055 REQ-014). It exists as a separate function because main calls os.Exit
// and can therefore never be driven by a test, while the decision it carries — which
// errors print, and with which code — is exactly what SPEC-055 REQ-011 changes.
//
// The default is LOUD. Every *ExitCodeError carrying a message prints, and silence is
// an explicit opt-out via Explained, claimed only by the commands that have already
// written structured findings to the consumer. This inverts the previous rule, which
// suppressed the message for EVERY ExitViolations error and so discarded the
// diagnostic of every command that uses exit 1 to mean "this operation failed"
// (ISSUE-074 pack relock, ISSUE-080 recipe apply).
//
// Cobra's own error printing is suppressed by the root command's SilenceErrors
// (root.go:30), so this is the sole printer.
func reportError(w io.Writer, err error) int {
	// errors.As, not a type assertion: a command that wraps its ExitCodeError for
	// context must still be classified by its code rather than falling through to the
	// untyped branch and reporting a config error for a violation.
	var exitErr *ExitCodeError
	if errors.As(err, &exitErr) {
		if !exitErr.Explained && exitErr.Message != "" {
			writeDiagnostic(w, exitErr.Message)
		}
		return exitErr.Code
	}

	// An error that never reached an ExitCodeError is one the command did not
	// classify: report it as a config error, and always print it — nothing else in the
	// process has recorded it.
	writeDiagnostic(w, err.Error())
	return ExitConfigError
}

// writeDiagnostic writes one "Error: <message>" line to w and reports whether it
// landed. reportError cannot act on the answer — the stream it would report a broken
// diagnostic stream on is the one that just failed, and the caller is already exiting
// non-zero — but keeping the write checked in one documented place is what stops the
// error being silently discarded at each call site.
func writeDiagnostic(w io.Writer, message string) bool {
	_, err := fmt.Fprintln(w, "Error:", message)
	return err == nil
}
