package distribution_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// TestMain exists for ONE reason (ISSUE-180): the Linux sandbox is a RE-EXEC
// TRAMPOLINE, and under `go test` its target is THIS PACKAGE'S OWN TEST BINARY.
//
// FIRST: newSandboxHelperInvocation (pkg/packval/sandbox_linux.go) cannot apply
// Landlock and seccomp in-process — they are self-applied, per-thread and
// irrevocable — so it spawns os.Executable() in a hidden helper mode with
// BACKSTOP_SANDBOX_HELPER_SPEC set and helper.Dir pointed at the pack directory.
// Under `go test`, os.Executable() is this package's compiled test binary.
//
// SECOND: without this gate, Go's DEFAULT generated test main does not recognise
// that env var, so the re-exec'd process reruns this package's ENTIRE suite from
// a directory off any go.mod ancestry, dies, and exits 1. And because Go's
// testing framework writes to STDOUT while foldHelperStderrIntoError
// (pkg/packval/sandbox_diagnostic.go) reads only STDERR, the parent reports "the
// sandboxed command wrote no diagnostic" and the real output vanishes. That is
// ISSUE-180, and it is what broke
// TestInstallContractsLocalPack_InstallsWithSuppliedCommand on Linux CI — 14
// times, once per fixture in packs/contracts.
//
// THIRD: a successful call in helper mode NEVER RETURNS. A non-nil error means
// this process IS a helper whose sandbox failed to install; running the suite at
// that point would hand the parent the suite's output as the sandboxed command's
// output.
//
// FOURTH: this package is a re-exec target at all because
// pkg/pack/distribution/validator.go reaches the real pipeline through
// packval.NewPipeline(packDir, ...).Run().
//
// The other members of this wiring family are pkg/packval/main_test.go's
// TestMain (the original precedent) and cmd/backstop/main.go's runWith (the
// shipped binary's half), with cmd/backstop/integration_test.go's TestMain as
// the sibling PLAN-ISSUE-163 added.
//
// NO BUILD TAG, deliberately. MaybeRunSandboxHelper resolves on every platform by
// design: on Linux it is the real gate, on darwin it is
// pkg/packval/sandbox_nonlinux.go's unconditional `return nil`. That stub is what
// makes this edit provably inert off Linux — the call returns nil immediately and
// control falls straight into m.Run(). A build-tagged TestMain would leave one
// platform's build without one.
func TestMain(m *testing.M) {
	// FIRST STATEMENT, and it must stay first: the helper has to run before
	// anything else consumes argv or does work.
	//
	// os.Exit takes the BARE LITERAL 126, not an identifier. sandboxHelperExitCode
	// is unexported and lives in cmd/backstop's `package main`, so it cannot be
	// referenced from here, and pkg/packval exports no equivalent —
	// pkg/packval/main_test.go spells this same literal for this same reason. 126 is
	// the fail-closed "refused to run pack code it could not confine" code
	// documented on pkg/packval/sandbox_diagnostic.go.
	if err := packval.MaybeRunSandboxHelper(); err != nil {
		var completion interface{ ExitCode() int }
		if errors.As(err, &completion) {
			os.Exit(completion.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "backstop sandbox helper: %v\n", err)
		os.Exit(126)
	}

	os.Exit(m.Run())
}
