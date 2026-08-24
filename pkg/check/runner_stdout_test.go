package check

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestRunner_StdoutSeparateFromStderr proves the dispatch runner returns stdout
// uncontaminated by stderr (REQ-009 / CLM-028). An engine that writes a banner
// to stderr and SARIF to stdout must yield clean stdout via RunStdout, so a
// tool's progress banner cannot corrupt the SARIF the dispatch parses.
func TestRunner_StdoutSeparateFromStderr(t *testing.T) {
	r := &ExecCommandRunner{}
	// sh writes a banner to stderr and a SARIF-shaped line to stdout.
	out, err := r.RunStdout(context.Background(), "sh", "-c", "echo banner 1>&2; printf '{\"version\":\"2.1.0\"}'")
	if err != nil {
		t.Fatalf("RunStdout error: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "banner") {
		t.Fatalf("stdout contaminated by stderr banner: %q", got)
	}
	if !strings.Contains(got, `{"version":"2.1.0"}`) {
		t.Fatalf("stdout missing the engine payload: %q", got)
	}
}

func TestRunner_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for name, run := range map[string]func(context.Context, string, ...string) ([]byte, error){
		"Run":       (&ExecCommandRunner{}).Run,
		"RunStdout": (&ExecCommandRunner{}).RunStdout,
	} {
		t.Run(name, func(t *testing.T) {
			started := time.Now()
			_, err := run(ctx, "sh", "-c", "sleep 10")
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
			if elapsed := time.Since(started); elapsed > time.Second {
				t.Fatalf("cancellation took %v", elapsed)
			}
		})
	}
}

// TestRunner_RunStdoutNonZeroExitReturnsStdout proves a non-zero exit still
// returns the captured stdout (alongside the error) so the caller can attribute
// a failure to the engine's output rather than receiving an opaque error.
func TestRunner_RunStdoutNonZeroExitReturnsStdout(t *testing.T) {
	r := &ExecCommandRunner{}
	out, err := r.RunStdout(context.Background(), "sh", "-c", "printf 'partial'; exit 3")
	if err == nil {
		t.Fatal("expected a non-zero exit to return an error")
	}
	if !strings.Contains(string(out), "partial") {
		t.Fatalf("expected captured stdout on failure, got: %q", out)
	}
}

// TestRunner_RunUnchangedCombinesOutput guards the additive contract (Review
// Question 5): the existing Run (CombinedOutput) method is untouched, so
// build/test executors that rely on merged stderr do not regress.
func TestRunner_RunUnchangedCombinesOutput(t *testing.T) {
	r := &ExecCommandRunner{}
	out, _ := r.Run(context.Background(), "sh", "-c", "echo err 1>&2; echo out")
	got := string(out)
	if !strings.Contains(got, "err") || !strings.Contains(got, "out") {
		t.Fatalf("Run must combine stdout+stderr, got: %q", got)
	}
}
