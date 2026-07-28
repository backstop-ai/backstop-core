package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The run() seam exists because main() cannot be driven by a test: it calls
// os.Exit, so every statement inside it is permanently unreachable from the test
// binary. Phase 3 made that concrete by adding the sandbox helper gate's error
// handling to main — three statements that no test could ever execute, guarding
// the one failure mode (the Linux sandbox could not be installed) whose whole
// purpose is to stop pack-supplied code running unsandboxed.
//
// These tests are written against the seam BEFORE it exists. They are the red
// phase for TASK-022.

// fakeRootCommand builds a minimal Cobra command standing in for NewRootCommand.
// It mirrors the real root's error posture — SilenceErrors and SilenceUsage are
// both set on the real tree (root.go:34-35), so the seam is the sole printer and
// a test that let Cobra print would be asserting against the wrong writer.
func fakeRootCommand(runE func(cmd *cobra.Command, args []string) error) *cobra.Command {
	return &cobra.Command{
		Use:           "backstop",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          runE,
		Args:          cobra.ArbitraryArgs,
	}
}

// TestRun_ReturnsExitCodeFromCommandError asserts run returns the code
// reportError produces for a classified *ExitCodeError, and writes the
// diagnostic to the writer it was handed rather than to os.Stderr.
//
// The writer half is the load-bearing half. main() passed os.Stderr literally,
// so nothing forced the diagnostic through a parameter; a seam that kept writing
// to os.Stderr would return the right code and still be untestable.
func TestRun_ReturnsExitCodeFromCommandError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runWith(&stdout, &stderr,
		func() error { return nil },
		func() *cobra.Command {
			return fakeRootCommand(func(*cobra.Command, []string) error {
				return &ExitCodeError{Code: ExitViolations, Message: "three findings block this change"}
			})
		},
	)

	if code != ExitViolations {
		t.Fatalf("run returned %d for an *ExitCodeError carrying ExitViolations; want %d — the seam must "+
			"preserve the command's own classification rather than collapsing every failure to a config error",
			code, ExitViolations)
	}
	if !strings.Contains(stderr.String(), "three findings block this change") {
		t.Errorf("the diagnostic did not reach the injected stderr; got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("a failing command wrote %q to stdout; diagnostics belong on stderr so JSON consumers "+
			"reading stdout are not corrupted", stdout.String())
	}
}

// TestRun_SuccessReturnsZero asserts a command that succeeds returns 0 and
// writes no diagnostic.
//
// This case drives the REAL run() rather than runWith, which is what makes it
// more than an arithmetic check: it proves run is wired to the real helper gate
// and the real command tree, so a seam that compiled but delegated to nothing
// would fail here. os.Args is swapped for the duration because Cobra resolves
// argv itself; the test is deliberately not parallel for that reason.
func TestRun_SuccessReturnsZero(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"backstop", "version"}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr)

	if code != 0 {
		t.Fatalf("run returned %d for `backstop version`; a successful command must return 0. stderr: %q",
			code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("a successful command wrote %q to stderr; success must be silent there", stderr.String())
	}
	if !strings.Contains(stdout.String(), "backstop version") {
		t.Errorf("run did not route the real command tree's output to the injected stdout; got %q", stdout.String())
	}
}

// TestRun_SandboxHelperErrorExitsOneTwentySix covers the path Phase 3 could not
// cover at all: the helper gate reporting that the Linux sandbox could not be
// installed.
//
// This is the first time that path is exercised anywhere. Its three statements
// were dead in main() by construction, and they guard the defect ISSUE-020
// exists to fix — a helper that neither execs nor errors would fall through and
// run pack-supplied code with no sandbox. The gate is INJECTED: the test must
// not depend on this process actually having been spawned as a helper.
func TestRun_SandboxHelperErrorExitsOneTwentySix(t *testing.T) {
	var stdout, stderr bytes.Buffer
	gateErr := errors.New("landlock ABI 0: no filesystem confinement available — see ISSUE-020")

	code := runWith(&stdout, &stderr,
		func() error { return gateErr },
		func() *cobra.Command {
			t.Fatal("the command tree was constructed after the helper gate reported failure")
			return nil
		},
	)

	if code != sandboxHelperExitCode {
		t.Fatalf("run returned %d when the sandbox helper gate failed; want %d — neither ExitViolations "+
			"nor ExitConfigError, because a broken sandbox is not a finding and not a misconfiguration",
			code, sandboxHelperExitCode)
	}
	if !strings.Contains(stderr.String(), gateErr.Error()) {
		t.Errorf("the gate's diagnostic must reach stderr so a broken sandbox is legible; got %q", stderr.String())
	}
}

// TestRun_RoutesOutputToStdoutAndErrorsToStderr asserts the streams main wires
// via SetOut/SetErr stay wired the same way through the seam.
//
// Cobra's cmd.Print/Println default to STDERR when SetOut is never called, so
// dropping the SetOut call is a silent regression: JSON consumers reading stdout
// would see nothing and every command would look broken to a script.
func TestRun_RoutesOutputToStdoutAndErrorsToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runWith(&stdout, &stderr,
		func() error { return nil },
		func() *cobra.Command {
			return fakeRootCommand(func(cmd *cobra.Command, _ []string) error {
				cmd.Println("structured output for stdout consumers")
				return &ExitCodeError{Code: ExitConfigError, Message: "diagnostic for stderr"}
			})
		},
	)

	if code != ExitConfigError {
		t.Fatalf("run returned %d, want %d", code, ExitConfigError)
	}
	if !strings.Contains(stdout.String(), "structured output for stdout consumers") {
		t.Errorf("cmd.Println did not reach the injected stdout — SetOut is not wired; got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "diagnostic for stderr") {
		t.Errorf("the diagnostic did not reach the injected stderr; got %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "diagnostic for stderr") {
		t.Errorf("the diagnostic leaked onto stdout: %q", stdout.String())
	}
}

// TestRun_SandboxGateRunsBeforeAnyCommandWork is THE ORDERING LOCK, and without
// it the other four tests are all satisfied by a run() that gets this exactly
// wrong.
//
// A run() that built and executed the command tree BEFORE calling the gate would
// return the right codes, write to the right streams, and pass every assertion
// above — while a real helper process printed help text to stdout before
// trampolining. "First statement" is behavioral, not stylistic: the helper is
// invoked with NO arguments and keys only on an environment variable, so
// anything that parses argv first is already wrong.
//
// The falsifier is a command-construction spy plus an empty-stdout assertion. A
// test that only checked the return code would not catch the reordering.
func TestRun_SandboxGateRunsBeforeAnyCommandWork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	commandConstructed := false

	code := runWith(&stdout, &stderr,
		func() error { return errors.New("sandbox restrictions could not be installed") },
		func() *cobra.Command {
			commandConstructed = true
			return fakeRootCommand(func(cmd *cobra.Command, _ []string) error {
				cmd.Println("help text a reordered run would emit before trampolining")
				return nil
			})
		},
	)

	if commandConstructed {
		t.Error("the command tree was CONSTRUCTED even though the helper gate failed; the gate must be run's " +
			"first statement, because a helper process is invoked with no arguments and keys only on an " +
			"environment variable")
	}
	if stdout.Len() != 0 {
		t.Errorf("run wrote %q to stdout after a failed helper gate; a real helper would have emitted this "+
			"before trampolining into the pack's command", stdout.String())
	}
	if code != sandboxHelperExitCode {
		t.Errorf("run returned %d, want %d", code, sandboxHelperExitCode)
	}
}
