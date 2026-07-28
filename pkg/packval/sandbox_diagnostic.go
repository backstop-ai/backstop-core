package packval

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// The stdout arm of the sandbox throws away the helper's reason unless something
// deliberately keeps it.
//
// WHY THIS EXISTS, recorded because the failure mode is invisible by nature. On
// 2026-07-28 the first real CI run of the Linux sandbox failed with exactly this
// to show for it:
//
//	convert step (scripts/build-to-sarif.sh) failed: sandboxed run (stdout) failed:
//	exit status 126
//
// 126 means the helper refused to run pack code it could not confine — fail-closed
// working correctly. The helper had ALSO written a precise diagnostic naming
// Landlock, the kernel release and ISSUE-020, exactly as CLM-015 requires. It went
// nowhere: platformSandboxedRunStdout set Stdin and Stdout and left Stderr nil,
// and a nil Stderr in os/exec sends the child's stderr to /dev/null. A guarantee
// that exists and cannot be observed is, for a diagnostic, the same as no
// guarantee at all.
//
// The fold is a PURE FUNCTION, and untagged, so the regression lock runs on
// darwin. Needing a Linux host to keep this caught would rebuild the blind spot
// that hid it: the bug lived on the one path no local test exercised.

// maxHelperDiagnosticBytes bounds how much helper stderr is folded into an error,
// so a runaway converter cannot produce an unbounded error string.
//
// The TAIL is what survives truncation. The helper writes its diagnostic LAST —
// after whatever the interpreter or the converter emitted on its way down — so
// keeping the head would preserve the noise and discard the reason.
const maxHelperDiagnosticBytes = 4096

// foldHelperStderrIntoError returns the command's stdout unchanged, plus an error
// carrying the helper's stderr when the run failed.
//
// The two halves are deliberately asymmetric. STDOUT IS RETURNED BYTE-IDENTICAL,
// always: this arm exists so a converter's stderr banner cannot corrupt the SARIF
// the gate parses, and folding stderr into the error must not undo that. STDERR
// goes into the ERROR only, and only when runErr is non-nil — a healthy converter
// that writes a deprecation notice and exits 0 must not be turned into a failure,
// so the run error decides, never the presence of stderr.
func foldHelperStderrIntoError(stdout, stderr []byte, runErr error) ([]byte, error) {
	if runErr == nil {
		return stdout, nil
	}

	diagnostic := strings.TrimSpace(string(stderr))
	if diagnostic == "" {
		// Absence of a diagnostic must not become absence of a failure. Say so
		// explicitly: "the helper died silently" is itself diagnostic information,
		// and it distinguishes this from the fold having dropped something.
		return stdout, fmt.Errorf("sandboxed run (stdout) failed: %w (the sandboxed command wrote no diagnostic)", runErr)
	}

	return stdout, fmt.Errorf("sandboxed run (stdout) failed: %w: %s", runErr, tailWithinLimit(diagnostic, maxHelperDiagnosticBytes))
}

// tailWithinLimit returns the last limit bytes of s, trimmed forward to a rune
// boundary so truncation cannot emit a broken UTF-8 sequence into an error
// message. A short string is returned unchanged.
func tailWithinLimit(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	tail := s[len(s)-limit:]
	// RuneStart, not ValidString on a single byte: the lead byte of a multi-byte
	// rune is not a valid string on its own, so testing validity byte-by-byte would
	// discard a perfectly good boundary.
	for len(tail) > 0 && !utf8.RuneStart(tail[0]) {
		tail = tail[1:]
	}
	return "…" + tail
}
