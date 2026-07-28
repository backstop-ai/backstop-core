//go:build !linux

package packval

import (
	"strings"
	"testing"
)

// The non-Linux stubs are a WIRING LOCK, and this file is what keeps them honest
// on the machine where the Linux sandbox is written but cannot run.
//
// sandbox.go switches on runtime.GOOS at RUN time while the Landlock
// implementation is selected at BUILD time, so the darwin build has to carry
// non-Linux counterparts for the entry points the linux arm names. That makes
// them look like dead weight — and dead-looking code with no test is what gets
// "cleaned up". The comment in sandbox_nonlinux.go says exactly what that costs:
// delete MaybeRunSandboxHelper and the shipped darwin build stops compiling,
// whereupon the obvious fix is to remove the call from cmd/backstop/main.go,
// which silently disarms the Linux sandbox on the platform that actually has one.
//
// These assertions are cheap. The failure they prevent is not.

// TestNonLinuxSandboxStubsFailClosed asserts the non-Linux arms return an ERROR
// rather than an empty success.
//
// Fail-closed is the whole point. A stub returning (nil, nil) would hand
// SandboxedRun's caller zero bytes and no error — which the gate reads as a
// convert step that produced no findings, i.e. a silent pass. That is the vacuous
// green this plan exists to eliminate, arriving through the back door of a
// platform stub nobody looks at.
func TestNonLinuxSandboxStubsFailClosed(t *testing.T) {
	out, err := linuxSandboxedRun("/bin/echo", []string{"hello"}, t.TempDir())
	if err == nil {
		t.Fatalf("linuxSandboxedRun returned no error on a non-linux build; a caller would read %q as a "+
			"successful sandboxed run that produced no output", string(out))
	}
	if out != nil {
		t.Errorf("linuxSandboxedRun returned %q alongside its error; the non-linux arm executes nothing "+
			"and must return no output", string(out))
	}
	if !strings.Contains(err.Error(), "linux") {
		t.Errorf("the refusal must say which platform arm was reached; got %q", err)
	}

	out, err = linuxSandboxedRunStdout("/bin/echo", []string{"hello"}, t.TempDir(), []byte("stdin"))
	if err == nil {
		t.Fatalf("linuxSandboxedRunStdout returned no error on a non-linux build; the convert path would "+
			"treat %q as clean SARIF bytes", string(out))
	}
	if out != nil {
		t.Errorf("linuxSandboxedRunStdout returned %q alongside its error; stdout must be empty when "+
			"nothing ran", string(out))
	}
	if !strings.Contains(err.Error(), "linux") {
		t.Errorf("the refusal must say which platform arm was reached; got %q", err)
	}
}

// TestNonLinuxMaybeRunSandboxHelperReturns asserts the helper gate is present on
// this platform and RETURNS.
//
// On Linux this function never returns when the process was spawned as a sandbox
// helper, and cmd/backstop/main.go plus this package's TestMain both call it
// unconditionally as their first statement. The non-Linux build has no trampoline
// to re-enter, so the correct behaviour is to do nothing and return — and the test
// reaching its final line is the assertion. If the darwin arm ever grew an exit or
// a block, every test binary in this package would die before running a single
// test, and every backstop invocation on macOS would die before parsing argv.
func TestNonLinuxMaybeRunSandboxHelperReturns(t *testing.T) {
	if err := MaybeRunSandboxHelper(); err != nil {
		t.Fatalf("the non-linux helper gate reported %v; nil is its only correct answer, and any error "+
			"would make cmd/backstop/main.go exit 126 before parsing a single argument", err)
	}
}
