package main

import (
	"bytes"
	"context"
	"os/exec"
	"sync"
)

// sharedTestRunner deduplicates the whole-module `go test ./...` execution that
// both the code_check step (asks "did any test fail?") and the
// coverage_threshold step (asks "what is each package's coverage?") would
// otherwise run separately. A single `go test ./... -coverprofile=/dev/null`
// run answers BOTH questions: its output carries the per-test FAIL blocks AND
// the per-package `coverage: N% of statements` lines. Running it twice was ~94s
// of pure duplicate work (the test suite is the gate's dominant cost).
//
// It implements both check.CommandRunner (Run + RunStdout) and
// gate.CommandRunner (Run) so the same instance can be injected into both
// steps. Any command that is NOT a whole-module `go test ./...` is delegated
// unchanged to a plain exec runner, so lint/build/semgrep/pack passes behave
// exactly as before.
type sharedTestRunner struct {
	dir string

	once sync.Once
	out  []byte
	err  error
}

func newSharedTestRunner(dir string) *sharedTestRunner {
	return &sharedTestRunner{dir: dir}
}

// isWholeModuleGoTest reports whether (name, args) is a `go test ./... [...]`
// invocation — the whole-module test run shared between code_check and
// coverage. A scoped `go test ./pkg/x` is NOT matched and runs normally.
func isWholeModuleGoTest(name string, args []string) bool {
	if name != "go" || len(args) < 2 || args[0] != "test" {
		return false
	}
	for _, a := range args {
		if a == "./..." {
			return true
		}
	}
	return false
}

// wholeModuleTest runs `go test ./... -coverprofile=/dev/null` exactly once,
// memoizing its combined output. -coverprofile forces the per-package coverage
// lines into the output (which the coverage step parses) without changing the
// FAIL blocks the code_check step parses.
func (r *sharedTestRunner) wholeModuleTest(ctx context.Context) ([]byte, error) {
	r.once.Do(func() {
		cmd := exec.CommandContext(ctx, "go", "test", "./...", "-coverprofile=/dev/null")
		cmd.Dir = r.dir
		r.out, r.err = cmd.CombinedOutput()
	})
	return r.out, r.err
}

// Run serves the memoized whole-module test output for any whole-module
// `go test ./...` request, and delegates everything else to a plain exec.
func (r *sharedTestRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if isWholeModuleGoTest(name, args) {
		return r.wholeModuleTest(ctx)
	}
	// name/args originate from the gate's own pass executors (go/golangci/
	// semgrep), not user input.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, name, args...)
	if r.dir != "" {
		cmd.Dir = r.dir
	}
	return cmd.CombinedOutput()
}

// RunStdout returns ONLY stdout for non-go-test commands (the engine-dispatch
// SARIF contract). Whole-module go test is captured via combined output, so a
// RunStdout request for it falls through to a normal stdout-only exec rather
// than the shared cache — the shared path is the CombinedOutput one used by the
// test/coverage passes.
func (r *sharedTestRunner) RunStdout(ctx context.Context, name string, args ...string) ([]byte, error) {
	// See Run: gate-supplied command, not user input.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, name, args...)
	if r.dir != "" {
		cmd.Dir = r.dir
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}
