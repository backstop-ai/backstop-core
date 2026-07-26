package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// SPEC-055 REQ-014 / CLM-096..098. reportError is the extracted, testable half of
// main()'s error handling: main() calls os.Exit and can never be driven by a test, so
// the decision of WHICH errors print, to WHICH writer, and with WHICH exit code lives
// here instead. These tests drive it directly against a bytes.Buffer — do not try to
// reach it through main().

// TestReportError_WritesDiagnosticAndReturnsCode is CLM-096. Both halves are asserted
// on purpose: an implementation that returns the right exit code while writing nothing
// is precisely the defect under repair (main.go's blanket ExitViolations suppression),
// and it would pass a code-only assertion.
func TestReportError_WritesDiagnosticAndReturnsCode(t *testing.T) {
	var buf bytes.Buffer

	code := reportError(&buf, &ExitCodeError{Code: ExitViolations, Message: "boom"})

	if code != ExitViolations {
		t.Errorf("reportError returned exit code %d, want ExitViolations (%d)", code, ExitViolations)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Errorf("reportError wrote %q, want a diagnostic containing the message %q", buf.String(), "boom")
	}

	// The falsifying twin: "always write something" must not pass. An ExitCodeError
	// carrying no message has no diagnostic to surface, so it stays silent while still
	// carrying its code.
	var empty bytes.Buffer
	emptyCode := reportError(&empty, &ExitCodeError{Code: ExitViolations})
	if empty.Len() != 0 {
		t.Errorf("reportError wrote %q for an empty-message error, want nothing written", empty.String())
	}
	if emptyCode != ExitViolations {
		t.Errorf("reportError returned exit code %d for an empty-message error, want ExitViolations (%d)", emptyCode, ExitViolations)
	}

	// A WRAPPED ExitCodeError must still be classified — the spec mandates errors.As
	// rather than a bare type assertion, and a bare assertion would fall through to the
	// untyped branch and report exit 2 for a violation.
	var wrapped bytes.Buffer
	wrappedCode := reportError(&wrapped, fmt.Errorf("adding pack: %w", &ExitCodeError{Code: ExitViolations, Message: "clone failed"}))
	if wrappedCode != ExitViolations {
		t.Errorf("reportError returned exit code %d for a wrapped ExitCodeError, want ExitViolations (%d)", wrappedCode, ExitViolations)
	}
	if !strings.Contains(wrapped.String(), "clone failed") {
		t.Errorf("reportError wrote %q for a wrapped ExitCodeError, want a diagnostic containing %q", wrapped.String(), "clone failed")
	}
}

// TestReportError_ExplainedWritesNothing is CLM-097. Explained is an explicit opt-OUT,
// claimed only by the four commands that have already written structured findings to
// the consumer. The buffer must be EMPTY, not merely free of the message: a writer that
// emitted a bare "Error:" prefix would still double up on the findings already printed.
func TestReportError_ExplainedWritesNothing(t *testing.T) {
	var buf bytes.Buffer

	code := reportError(&buf, &ExitCodeError{Code: ExitViolations, Message: "boom", Explained: true})

	if buf.Len() != 0 {
		t.Errorf("reportError wrote %q for an Explained error, want nothing written", buf.String())
	}
	if code != ExitViolations {
		t.Errorf("reportError returned exit code %d for an Explained error, want ExitViolations (%d)", code, ExitViolations)
	}
}

// TestReportError_UntypedErrorMapsToConfigExit is CLM-098. An error that never reached
// an ExitCodeError is a failure the command did not classify, so it is a config error
// (exit 2) and it always prints — there is no other record of it anywhere.
func TestReportError_UntypedErrorMapsToConfigExit(t *testing.T) {
	var buf bytes.Buffer

	code := reportError(&buf, errors.New("config: no backstop.yml found"))

	if code != ExitConfigError {
		t.Errorf("reportError returned exit code %d for an untyped error, want ExitConfigError (%d)", code, ExitConfigError)
	}
	if !strings.Contains(buf.String(), "config: no backstop.yml found") {
		t.Errorf("reportError wrote %q for an untyped error, want a diagnostic containing the message", buf.String())
	}
}
