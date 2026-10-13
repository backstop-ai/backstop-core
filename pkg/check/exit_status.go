package check

import "errors"

// ExitStatus extracts the exit status a STARTED process reported from a run error
// (ISSUE-207 D-5). It is the single authority both findings dispatch paths use
// (cmd/backstop's runFindingsEngine and pkg/packval's DefaultExecutor.RunEngine),
// sitting alongside NeverStarted, which answers the earlier question of whether
// the process started at all.
//
// It matches any error carrying an ExitCode() method via errors.As, which
// *exec.ExitError satisfies, so a wrapped error still classifies. It reports false
// for a nil error, for an error with no exit status (a stub or I/O failure), and
// for a negative code (a process killed by a signal reports -1). A caller must
// treat false as "no status was observed", never as success.
func ExitStatus(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var coded interface{ ExitCode() int }
	if !errors.As(err, &coded) {
		return 0, false
	}
	code := coded.ExitCode()
	if code < 0 {
		return 0, false
	}
	return code, true
}
