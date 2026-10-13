package check

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"
)

// ExitStatus unit tests (ISSUE-207 D-5). The dispatch paths in cmd/backstop and
// pkg/packval exercise ExitStatus end to end, but coverage is measured per package,
// so the classifier's own branches are pinned here.

// codedErr is an error that reports an arbitrary exit code, standing in for a
// process killed by a signal (which reports -1) without needing to send one.
type codedErr struct{ code int }

func (e codedErr) Error() string { return fmt.Sprintf("coded %d", e.code) }
func (e codedErr) ExitCode() int { return e.code }

// TestExitStatus_ClassifiesRunErrors proves ExitStatus reports a status only for a
// started process with a non-negative exit code, and never treats "no status" as 0.
func TestExitStatus_ClassifiesRunErrors(t *testing.T) {
	realErr := exec.Command("/bin/sh", "-c", "exit 3").Run()
	if realErr == nil {
		t.Fatal("fixture: sh exit 3 returned nil error")
	}

	cases := []struct {
		name     string
		err      error
		wantCode int
		wantOK   bool
	}{
		{"nil error has no status", nil, 0, false},
		{"plain error has no status", errors.New("stub failure"), 0, false},
		{"real exit error", realErr, 3, true},
		{"wrapped exit error still classifies", fmt.Errorf("dispatch: %w", realErr), 3, true},
		{"exit code zero is a status", codedErr{code: 0}, 0, true},
		{"signal-killed negative code has no status", codedErr{code: -1}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, ok := ExitStatus(tc.err)
			if code != tc.wantCode || ok != tc.wantOK {
				t.Errorf("ExitStatus(%v) = (%d, %v), want (%d, %v)", tc.err, code, ok, tc.wantCode, tc.wantOK)
			}
		})
	}
}
