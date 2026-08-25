package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

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

// main is deliberately ONE statement. Everything it used to do lives in run,
// which returns a code instead of calling os.Exit and is therefore reachable
// from a test — including the sandbox helper gate's error handling, which was
// unreachable by construction while it sat here.
func main() { os.Exit(run(os.Stdout, os.Stderr)) }

// run is main's body with the process exit removed: it returns the code main
// exits with. It supplies the REAL dependencies to runWith, which is the seam
// tests drive.
//
// The split is what makes both halves honest. run proves the production wiring —
// the real helper gate, the real command tree — while runWith proves the
// behavior against injected doubles, including the one failure mode the real
// gate cannot be made to produce on demand.
func run(stdout, stderr io.Writer) int {
	return runWith(stdout, stderr, packval.MaybeRunSandboxHelper, NewRootCommand)
}

// runWith carries main's logic with the helper gate and the command-tree
// constructor injected.
//
// They are PARAMETERS rather than package-level function variables on purpose:
// a `var sandboxGate = packval.MaybeRunSandboxHelper` is package-level mutable
// state (go.core.no-global-mutable-state), and the rule's own remedy is the
// dependency injection used here.
func runWith(stdout, stderr io.Writer, sandboxGate func() error, newRoot func() *cobra.Command) int {
	// FIRST STATEMENT, before the command tree is built and before Cobra sees argv.
	//
	// The Linux sandbox is a re-exec trampoline: SandboxedRun spawns /proc/self/exe
	// — this binary — in a hidden helper mode that restricts ITSELF with Landlock and
	// seccomp and then execs the pack's command. The helper gate therefore has to run
	// before any argument parsing, because the helper is invoked with no arguments and
	// keys only on an environment variable. Anything that constructs or executes the
	// command tree first is already wrong, which is why the ordering has its own test
	// rather than resting on this comment.
	//
	// When this process is not a helper the call returns nil immediately and does
	// nothing. When it IS one, a successful call never returns — the helper execs the
	// pack's command. A non-nil error means this process IS a helper and the sandbox
	// could NOT be installed, so the process exits 126: falling through would run
	// pack-supplied code unsandboxed, which is the defect ISSUE-020 exists to fix.
	//
	// Removing these lines does not fail any test in cmd/backstop that asserts on a
	// command's behavior — it makes the SHIPPED BINARY sandbox nothing, silently. Its
	// test-side twin is packval.MaybeRunSandboxHelper() in pkg/packval's TestMain.
	if err := sandboxGate(); err != nil {
		var completion interface{ ExitCode() int }
		if errors.As(err, &completion) {
			return completion.ExitCode()
		}
		writeDiagnostic(stderr, err.Error())
		return sandboxHelperExitCode
	}

	rootCmd := newRoot()
	// Cobra's cmd.Print/Println default to stderr when SetOut is never
	// called. Route success output to stdout so JSON consumers (the
	// runtime, scripts) see what they expect on stdout.
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	if err := rootCmd.Execute(); err != nil {
		return reportError(stderr, err)
	}
	return 0
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
