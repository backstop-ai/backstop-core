package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// sandboxHelperExitCode is what this binary exits with when it was spawned as a
// Linux sandbox helper and the restrictions could not be installed. 126 is the
// shell convention for "the command was found but could not be executed", which is
// precisely what happened: the pack's command was never exec'd because the sandbox
// it was to run inside did not come up. It is deliberately NOT ExitViolations or
// ExitConfigError — neither a finding nor a misconfiguration, and a consumer that
// sees it should read a broken sandbox, not a failed check.
const sandboxHelperExitCode = 126

func main() {
	// FIRST STATEMENT, before NewRootCommand and before Cobra sees argv.
	//
	// The Linux sandbox is a re-exec trampoline: SandboxedRun spawns /proc/self/exe
	// — this binary — in a hidden helper mode that restricts ITSELF with Landlock and
	// seccomp and then execs the pack's command. The helper gate therefore has to run
	// before any argument parsing, because the helper is invoked with no arguments and
	// keys only on an environment variable.
	//
	// When this process is not a helper the call returns nil immediately and does
	// nothing. When it IS one, a successful call never returns — the helper execs the
	// pack's command. A non-nil error means this process IS a helper and the sandbox
	// could NOT be installed, so main() owns the exit: falling through would run
	// pack-supplied code unsandboxed, which is the defect ISSUE-020 exists to fix.
	//
	// Removing these lines does not fail any test in cmd/backstop — it makes the
	// SHIPPED BINARY sandbox nothing, silently. Its test-side twin is
	// packval.MaybeRunSandboxHelper() in pkg/packval's TestMain.
	if err := packval.MaybeRunSandboxHelper(); err != nil {
		writeDiagnostic(os.Stderr, err.Error())
		os.Exit(sandboxHelperExitCode)
	}

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
