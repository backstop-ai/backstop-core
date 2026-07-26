package distribution_test

// Diagnostic-quality suite for the production GitCloner (SPEC-055 REQ-002).
//
// REQ-002 is about what a failure TELLS the operator, so every test here asserts on
// MESSAGE CONTENT, never merely that an error came back. A cloner that returned a
// bare exit status for each of these would satisfy an "err != nil" suite and leave
// every real failure unactionable.
//
// Nothing here reaches a remote host: the unreachable case is a local path that does
// not exist, and the timeout case drives the harness's hanging shell stand-in.

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// diagnosticTimeout bounds the real-git invocations in this suite.
const diagnosticTimeout = 60 * time.Second

// timeoutTestBudget bounds the TEST around a call that must return on its own. It is
// far larger than the cloner timeout under test, so a regression that hangs fails
// this test instead of wedging the run.
const timeoutTestBudget = 30 * time.Second

func TestExecGitCloner_Clone_MissingTagReturnsTypedDiagnostic(t *testing.T) {
	repo := newTaggedRepo(t, taggedRepoSpec{
		Revisions: []repoRevision{
			{
				Files: packTreeFiles("harness-pack", "1.0.0"),
				Tags:  []repoTag{{Name: "v1.0.0", Annotated: true}},
			},
		},
	})

	cloner := &distribution.ExecGitCloner{Timeout: diagnosticTimeout}

	err := cloner.Clone(repo.URL, "v9.9.9", newCloneDestination(t))

	message := requireGitError(t, err, "cloning a tag that does not exist")
	if !strings.Contains(message, repo.URL) {
		t.Errorf("the diagnostic %q does not name the repository %q; an operator cannot tell WHICH pack failed", message, repo.URL)
	}
	if !strings.Contains(message, "v9.9.9") {
		t.Errorf("the diagnostic %q does not name the requested tag v9.9.9", message)
	}
}

func TestExecGitCloner_Clone_UnreachableURLCarriesStderr(t *testing.T) {
	// A local path that does not exist, NOT a network host: the failure is real and
	// the test stays hermetic.
	unreachableURL := "file://" + filepath.ToSlash(filepath.Join(t.TempDir(), "repository-that-was-never-created"))

	cloner := &distribution.ExecGitCloner{Timeout: diagnosticTimeout}

	err := cloner.Clone(unreachableURL, "v1.0.0", newCloneDestination(t))

	message := requireGitError(t, err, "cloning an unreachable repository")
	// "fatal:" is git's OWN stderr prefix. Asserting on it means a hand-written
	// message that merely says "clone failed" cannot pass this test.
	if !strings.Contains(message, "fatal:") {
		t.Errorf("the diagnostic %q carries no captured git stderr; want git's own output so the operator sees what git actually said", message)
	}
	if !strings.Contains(message, unreachableURL) {
		t.Errorf("the diagnostic %q does not name the repository %q", message, unreachableURL)
	}
}

func TestExecGitCloner_Clone_MissingGitBinaryReturnsTypedDiagnostic(t *testing.T) {
	// GitBinary exists precisely so this is testable without mutating the test
	// process's PATH.
	absentBinary := missingGitBinaryPath(t)

	cloner := &distribution.ExecGitCloner{GitBinary: absentBinary, Timeout: diagnosticTimeout}

	err := cloner.Clone("file:///harness/repo", "v1.0.0", newCloneDestination(t))

	message := requireGitError(t, err, "invoking a git binary that is not installed")
	if !strings.Contains(message, absentBinary) {
		t.Errorf("the diagnostic %q does not name the executable %q; a bare exec error leaves the operator guessing what is missing", message, absentBinary)
	}
}

func TestExecGitCloner_Clone_TimeoutReturnsTypedDiagnostic(t *testing.T) {
	hangingGit := newFakeGitBinary(t, fakeGitHangs)

	cloner := &distribution.ExecGitCloner{GitBinary: hangingGit, Timeout: 250 * time.Millisecond}
	destDir := newCloneDestination(t)

	// The call must RETURN. Bounding the test means a cloner that forgot to bound
	// its subprocess fails here rather than wedging CI. The destination is built on
	// the test goroutine, so nothing below touches testing.T off it.
	outcome := make(chan error, 1)
	go func() {
		outcome <- cloner.Clone("file:///harness/repo", "v1.0.0", destDir)
	}()

	select {
	case cloneErr := <-outcome:
		message := requireGitError(t, cloneErr, "a git invocation that exceeds its timeout")
		if !strings.Contains(message, "250ms") {
			t.Errorf("the diagnostic %q does not name the timeout that fired; the operator cannot tell a slow remote from a broken one", message)
		}
	case <-time.After(timeoutTestBudget):
		t.Fatalf("Clone did not return within %v; the invocation is not bounded", timeoutTestBudget)
	}
}

func TestExecGitCloner_ListTags_UnreachableURLErrorsRatherThanEmpty(t *testing.T) {
	// The falsifier for the whole resolver chain: an empty list on failure would
	// surface later as "already at the latest version", which is a lie.
	unreachableURL := "file://" + filepath.ToSlash(filepath.Join(t.TempDir(), "repository-that-was-never-created"))

	cloner := &distribution.ExecGitCloner{Timeout: diagnosticTimeout}

	tags, err := cloner.ListTags(unreachableURL)

	if err == nil {
		t.Fatalf("ListTags(%s) returned %v with no error; a failing listing must never be reported as an empty tag set", unreachableURL, tags)
	}
	if len(tags) != 0 {
		t.Errorf("ListTags returned %v alongside its error; want no tags", tags)
	}

	message := requireGitError(t, err, "listing tags on an unreachable repository")
	if !strings.Contains(message, unreachableURL) {
		t.Errorf("the diagnostic %q does not name the repository %q", message, unreachableURL)
	}
}

// requireGitError asserts err is a typed *GitError and returns its message for
// content assertions. The context describes the operation under test so a failure
// here reads as a sentence.
func requireGitError(t *testing.T, err error, context string) string {
	t.Helper()

	if err == nil {
		t.Fatalf("%s returned no error; want a *distribution.GitError", context)
	}

	var gitErr *distribution.GitError
	if !errors.As(err, &gitErr) {
		t.Fatalf("%s returned %T (%v); want *distribution.GitError so callers can classify it", context, err, err)
	}

	return gitErr.Error()
}
