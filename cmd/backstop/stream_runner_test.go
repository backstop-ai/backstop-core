package main

import (
	"strings"
	"testing"
)

// SPEC-055 REQ-014 / CLM-099. The existing runBackstop helper returns CombinedOutput,
// a MERGED stream. Against it, "prints its diagnostic to stderr" passes for a command
// that printed to stdout, and "stdout holds exactly one JSON document" passes for a run
// that also wrote a human line to stderr — so every REQ-011 and REQ-012 stream claim
// would pass vacuously. This test proves the replacement genuinely separates the two.
//
// The driving invocation is `artifact validate --all` in a project with a backstop.yml
// and no artifacts: the human formatter writes the pass line to STDOUT while the
// zero-artifact warning goes to STDERR (artifact_validate.go:342). Neither half routes
// through reportError, so this fixture is stable across the per-site rewiring that
// follows — it is not measuring the error seam, it is measuring the runner.
func TestRunBackstopStreams_SeparatesStdoutFromStderr(t *testing.T) {
	// No -short skip: this is the only test driving CLM-099, and it is the substrate
	// every REQ-011/REQ-012 stream claim rests on. A run that quietly skips it leaves
	// the merged-vs-separated distinction unproven.
	bin := buildBackstopBinary(t)
	proj := newConsumerProject(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "artifact", "validate", "--all")

	const (
		stdoutOnly = "All checks passed"
		stderrOnly = "no artifacts found to validate"
	)

	if code != 0 {
		t.Fatalf("artifact validate --all on an artifact-free project exited %d, want 0\nstdout: %q\nstderr: %q", code, stdout, stderr)
	}

	// The four assertions below are the whole point. The two "contains" halves pass
	// against a merged buffer too; the two "does NOT contain" halves are what a
	// CombinedOutput runner FAILS, because in a merged stream both strings appear in
	// both return values.
	if !strings.Contains(stdout, stdoutOnly) {
		t.Errorf("stdout = %q, want it to contain the report line %q", stdout, stdoutOnly)
	}
	if strings.Contains(stdout, stderrOnly) {
		t.Errorf("stdout = %q, want it NOT to contain the stderr-only warning %q — the streams are merged", stdout, stderrOnly)
	}
	if !strings.Contains(stderr, stderrOnly) {
		t.Errorf("stderr = %q, want it to contain the warning %q", stderr, stderrOnly)
	}
	if strings.Contains(stderr, stdoutOnly) {
		t.Errorf("stderr = %q, want it NOT to contain the stdout-only report line %q — the streams are merged", stderr, stdoutOnly)
	}

	// A non-zero run: the runner must recover the child's exit code from the
	// *exec.ExitError rather than reporting the success it did not get. Running in a
	// directory with no backstop.yml is a config error (exit 2) whose diagnostic goes
	// to stderr and leaves stdout empty.
	failStdout, failStderr, failCode := runBackstopStreams(t, bin, t.TempDir(), "artifact", "validate", "--all")
	if failCode != ExitConfigError {
		t.Errorf("artifact validate --all with no backstop.yml exited %d, want ExitConfigError (%d)\nstdout: %q\nstderr: %q", failCode, ExitConfigError, failStdout, failStderr)
	}
	if failStdout != "" {
		t.Errorf("stdout = %q for a config failure, want it empty", failStdout)
	}
	if !strings.Contains(failStderr, "backstop.yml") {
		t.Errorf("stderr = %q for a config failure, want the diagnostic naming backstop.yml", failStderr)
	}
}
