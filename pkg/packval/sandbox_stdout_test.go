package packval

import (
	"runtime"
	"strings"
	"testing"
)

// TestSandboxedRunStdout_StdoutCleanOfStderr proves the clean-stdout sandbox
// capture (REQ-009 / CLM-065): a converter that writes a banner to stderr and
// SARIF-shaped bytes to stdout yields stdout uncontaminated by the stderr
// banner, so the convert step's final SARIF cannot be corrupted by a stderr
// banner. SandboxedRun's CombinedOutput would interleave the two.
func TestSandboxedRunStdout_StdoutCleanOfStderr(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only")
	}
	dir := t.TempDir()

	out, err := SandboxedRunStdout("sh", []string{"-c", "echo 'WARNING: banner' 1>&2; printf '{\"sarif\":true}'"}, dir, nil)
	if err != nil {
		t.Fatalf("SandboxedRunStdout error: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "WARNING") || strings.Contains(got, "banner") {
		t.Fatalf("stdout was contaminated by stderr banner: %q", got)
	}
	if !strings.Contains(got, `{"sarif":true}`) {
		t.Fatalf("stdout did not carry the converter payload: %q", got)
	}
}

// TestSandboxedRunStdout_PipesStdin proves the convert step can feed the
// engine's stdout into the converter's stdin (the REQ-007 two-process pipe):
// the command echoes its stdin to stdout, and the captured stdout equals the
// stdin we supplied.
func TestSandboxedRunStdout_PipesStdin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only")
	}
	dir := t.TempDir()

	payload := []byte(`{"engine":"ast-grep","findings":1}`)
	out, err := SandboxedRunStdout("cat", nil, dir, payload)
	if err != nil {
		t.Fatalf("SandboxedRunStdout error: %v", err)
	}
	if string(out) != string(payload) {
		t.Fatalf("stdin was not piped through to stdout: got %q want %q", out, payload)
	}
}

// TestSandboxedRunStdout_NonZeroExitReturnsStdoutSoFar closes the last unasserted
// half of SandboxedRunStdout's contract: on a non-zero exit it returns the stdout
// captured SO FAR alongside the error, so the caller can attribute the failure to
// the convert step rather than to the engine that fed it.
//
// The failure path is the one that matters here. A converter that emits partial
// SARIF and then dies is the realistic shape — a jq filter that hits malformed
// input halfway through — and an implementation that returned (nil, err) on that
// path would look correct in every success-path test while destroying the only
// evidence of where the pipeline broke. The stderr assertion is the same
// clean-stdout guarantee as the success path: failing must not start merging the
// two streams.
func TestSandboxedRunStdout_NonZeroExitReturnsStdoutSoFar(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only")
	}
	dir := t.TempDir()

	out, err := SandboxedRunStdout("sh",
		[]string{"-c", "printf '{\"partial\":true}'; echo 'converter blew up' 1>&2; exit 3"}, dir, nil)

	if err == nil {
		t.Fatalf("a converter exiting 3 must surface as an error; got output %q and no error", string(out))
	}
	if !strings.Contains(string(out), `{"partial":true}`) {
		t.Fatalf("the stdout captured before the failure was discarded; it must be returned alongside the "+
			"error so the caller can attribute the failure to the convert step. got %q", string(out))
	}
	if strings.Contains(string(out), "converter blew up") {
		t.Fatalf("stderr leaked into the returned stdout on the FAILURE path; the clean-stdout guarantee "+
			"holds there too. got %q", string(out))
	}
}

// TestSandboxedRun_StillCombinesOutput guards the additive contract: the
// existing CombinedOutput-based SandboxedRun is left in place for the exit-code
// sandbox-validator path (REQ-014), whose message body may legitimately include
// stderr. A stderr banner must appear in SandboxedRun's combined output.
func TestSandboxedRun_StillCombinesOutput(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only")
	}
	dir := t.TempDir()

	out, err := SandboxedRun("sh", []string{"-c", "echo 'stderr-line' 1>&2; echo 'stdout-line'"}, dir)
	if err != nil {
		t.Fatalf("SandboxedRun error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "stderr-line") {
		t.Fatalf("SandboxedRun must retain combined stderr, got: %q", got)
	}
}
