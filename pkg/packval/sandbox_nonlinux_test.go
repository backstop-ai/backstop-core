//go:build !linux

package packval

import (
	"go/build/constraint"
	"os"
	"strings"
	"testing"
)

// These two assertions are the WIRING LOCK for the helper gate on the machine
// where the Linux sandbox is written but cannot run.
//
// Platform DISPATCH is build-tagged now — sandbox_linux.go for Landlock+seccomp,
// sandbox_nonlinux.go for sandbox-exec on darwin and the fail-closed refusal
// everywhere else — so sandbox_nonlinux.go no longer carries linuxSandboxedRun*
// stub arms. What these two assertions guard is the symbol that must exist on
// EVERY platform, and the tag that keeps it that way. Both failures they prevent
// are silent.

// TestNonLinuxSandboxHelperTagIsNotNarrowed asserts sandbox_nonlinux.go's build
// constraint is still exactly `!linux`.
//
// This is the trap TASK-023 names, and it is worth a test rather than a comment
// because the mistake looks like tidying: split the darwin arm out into its own
// `darwin` file and this file's `!linux` starts reading like an overlap someone
// should narrow to `!linux && !darwin`. It is not an overlap. MaybeRunSandboxHelper
// is not a dispatch arm — cmd/backstop's run() calls it unconditionally on every
// platform, so narrowing the tag costs the darwin build the symbol, and the obvious
// fix at that point is to delete the call site, which makes the build green and
// silently disarms the Linux sandbox.
//
// Asserting the constraint STRUCTURALLY (parsed, not substring-matched) is what
// makes the failure message land on the real cause instead of on a compile error
// three files away.
func TestNonLinuxSandboxHelperTagIsNotNarrowed(t *testing.T) {
	source, err := os.ReadFile("sandbox_nonlinux.go")
	if err != nil {
		t.Fatalf("reading sandbox_nonlinux.go: %v", err)
	}

	var got string
	for _, line := range strings.Split(string(source), "\n") {
		if !constraint.IsGoBuild(line) {
			continue
		}
		expr, parseErr := constraint.Parse(line)
		if parseErr != nil {
			t.Fatalf("parsing build constraint %q: %v", line, parseErr)
		}
		got = expr.String()
		break
	}

	if got == "" {
		t.Fatal("sandbox_nonlinux.go carries no //go:build constraint; without one it compiles on linux too " +
			"and collides with the linux MaybeRunSandboxHelper")
	}
	if got != "!linux" {
		t.Fatalf("sandbox_nonlinux.go is tagged %q, want \"!linux\". MaybeRunSandboxHelper is NOT a dispatch "+
			"arm — run() calls it unconditionally on every platform, so narrowing this tag costs the darwin "+
			"build the symbol and invites deleting the call site, which silently disarms the Linux sandbox", got)
	}
}

// TestNonLinuxMaybeRunSandboxHelperReturns asserts the helper gate is present on
// this platform and RETURNS.
//
// On Linux this function never returns when the process was spawned as a sandbox
// helper, and cmd/backstop/run() plus this package's TestMain both call it
// unconditionally as their first statement. The non-Linux build has no trampoline
// to re-enter, so the correct behaviour is to do nothing and return — and the test
// reaching its final line is the assertion. If the darwin arm ever grew an exit or
// a block, every test binary in this package would die before running a single
// test, and every backstop invocation on macOS would die before parsing argv.
func TestNonLinuxMaybeRunSandboxHelperReturns(t *testing.T) {
	if err := MaybeRunSandboxHelper(); err != nil {
		t.Fatalf("the non-linux helper gate reported %v; nil is its only correct answer, and any error "+
			"would make cmd/backstop's run() return 126 before parsing a single argument", err)
	}
}
