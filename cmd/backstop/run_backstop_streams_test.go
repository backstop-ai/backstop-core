package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"testing"
)

// runBackstopStreams runs the built binary with the given working dir and args,
// returning stdout and stderr as INDEPENDENT captures alongside the process exit code
// (SPEC-055 REQ-014 / CLM-099).
//
// It mirrors runBackstop's shape (pack_authoring_loop_test.go:27) so migrating a test
// between them is a one-line change, but it never calls CombinedOutput. That is the
// entire reason it exists: no REQ-011 or REQ-012 claim may be driven by the merged
// helper, because against a merged buffer "prints its diagnostic to stderr" passes for
// a command that printed to stdout, and "stdout holds exactly one JSON document"
// passes for a run that also wrote a human line to stderr. That blind spot is what let
// the silent-exit-1 defect survive several specs' worth of CLI tests.
//
// runBackstop STAYS for its existing callers, who legitimately do not care about
// streams.
//
// The child INHERITS the test process's environment on purpose. A hermetic test
// installs its git redirect with t.Setenv, which mutates os.Environ in this process
// only; a runner that handed the child a scrubbed or hand-built env would send it to
// GitHub instead, and the suite would go silently network-dependent without anything
// turning red.
func runBackstopStreams(t *testing.T, bin, dir string, args ...string) (string, string, int) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			// The binary never ran (missing, not executable, bad working dir). That is a
			// harness failure, not a result the caller should assert on.
			t.Fatalf("running %s %v: %v\nstdout: %s\nstderr: %s", bin, args, err, stdout.String(), stderr.String())
		}
		code = exitErr.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}
