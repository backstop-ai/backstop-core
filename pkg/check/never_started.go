package check

import (
	"errors"
	"io/fs"
	"os/exec"
)

// NeverStarted reports whether a run error means THE PROCESS NEVER STARTED, as
// opposed to started-and-exited-non-zero (ISSUE-112, widened to this shared home
// by ISSUE-140). It is the SINGLE authority for that question: BOTH consumers
// call it, because two copies are how the two branches drift apart — and they
// did, which is what ISSUE-140 reports.
//
//   - cmd/backstop's gate dispatch (runFindingsEngine and runCoverageEngine): the
//     coverage producer command is always filepath.Join(packRoot, …) and is already
//     os.Stat-guarded, so *exec.Error is UNREACHABLE on that branch and a narrow
//     check there would be greenable only by a stub runner — i.e. vacuously.
//   - pkg/packval's fixture executor (DefaultExecutor.RunEngine, behind `backstop
//     pack test` / `pack check`): its command comes from binding.Command, which is
//     pack-declared DATA that may carry a path separator, so the path-ful shape is
//     an ordinary pack declaration rather than an edge case.
//
// It must NOT be replaced by `runErr != nil`. A rule-fed findings engine exits
// non-zero precisely WHEN it reports findings, so treating every run error as
// fatal would red the gate on every real finding. A started process reports an
// *exec.ExitError, which is neither shape below.
//
// "Never started" is TWO Go types because exec.Command reaches LookPath only for a
// BARE command name (filepath.Base(name) == name):
//
//   - *exec.Error — a bare name that LookPath could not resolve.
//   - *fs.PathError with Op == "fork/exec" — a PATH-FUL command that could not be
//     exec'd: absent, not executable, or carrying a bad interpreter line. Such a
//     command never consults LookPath, so it can never produce an *exec.Error.
//
// It keys on Op, never on the errno (ENOENT / EACCES / ENOEXEC), which would bake
// OS knowledge into a thin executor. It matches via errors.As so a wrapped error
// still classifies.
func NeverStarted(runErr error) bool {
	var execErr *exec.Error
	if errors.As(runErr, &execErr) {
		return true
	}
	var pathErr *fs.PathError
	if errors.As(runErr, &pathErr) && pathErr.Op == "fork/exec" {
		return true
	}
	return false
}
