package packval

import (
	"fmt"
	"os"
	"testing"
)

// TestMain exists for ONE reason: the sandbox trampoline re-execs
// /proc/self/exe, and under `go test` that is the TEST BINARY rather than the
// backstop binary.
//
// The Linux sandbox (sandbox_linux.go) cannot apply Landlock and seccomp
// in-process — they are self-applied, per-thread and irrevocable, so installing
// them inside SandboxedRun would permanently confine the calling process. It
// therefore spawns this executable in a hidden helper mode, and the helper
// restricts itself before exec'ing the pack's command. MaybeRunSandboxHelper is
// the gate that recognises helper mode, and it has to run before anything else
// consumes argv.
//
// This is the test-side half of a WIRING PAIR whose other half is
// packval.MaybeRunSandboxHelper() as the first statement of
// cmd/backstop/main.go. Either half missing is a silent hole, and they fail in
// different directions:
//
//   - missing in main.go — every Linux test here still passes while the SHIPPED
//     BINARY sandboxes nothing.
//   - missing here — the suite cannot reach the helper at all, so every Linux
//     sandbox test fails for a reason that has nothing to do with the sandbox.
//
// On darwin the call is a no-op (sandbox_nonlinux.go) and this TestMain is
// therefore inert, but it must not be made linux-only: `go test` builds one
// binary per package for whatever platform it targets, and a build-tagged
// TestMain would leave the linux build without one.
func TestMain(m *testing.M) {
	// FIRST STATEMENT. When this process was spawned as a sandbox helper a
	// successful call never returns; otherwise it returns nil immediately, having
	// done nothing.
	//
	// A non-nil error means this test binary IS a helper whose sandbox failed to
	// install. Running the suite at that point would be worse than useless: the
	// parent would read the suite's output as the sandboxed command's output.
	if err := MaybeRunSandboxHelper(); err != nil {
		fmt.Fprintf(os.Stderr, "backstop sandbox helper: %v\n", err)
		os.Exit(126)
	}

	os.Exit(m.Run())
}
