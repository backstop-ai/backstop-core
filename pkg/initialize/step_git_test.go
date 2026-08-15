package initialize

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireGitBinary skips rather than fails when git is absent: this step drives the
// real git binary, and an environment without it cannot honestly run these claims.
func requireGitBinary(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH; the git step drives the real binary")
	}
}

// initGitRepository makes dir a real repository with one commit, so HEAD, config and
// refs all exist and a re-init would be observable.
func initGitRepository(t *testing.T, dir string) {
	t.Helper()
	requireGitBinary(t)
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=init step fixture", "GIT_AUTHOR_EMAIL=fixture@backstop.test",
		"GIT_COMMITTER_NAME=init step fixture", "GIT_COMMITTER_EMAIL=fixture@backstop.test",
	)
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "init step fixture"},
		{"config", "user.email", "fixture@backstop.test"},
		{"commit", "--allow-empty", "-m", "pre-existing history"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}
}

// TestInit_CreatesGitRepositoryWhenNoneExists (SPEC-069 CLM-036).
//
// In a directory with no `.git`, the step creates a git repository. The only fact the
// step inspects is the PRESENCE of `.git` — nothing else in the project is read.
func TestInit_CreatesGitRepositoryWhenNoneExists(t *testing.T) {
	requireGitBinary(t)
	root := t.TempDir()

	if exists(root, ".git") {
		t.Fatal("the fixture directory already carries a .git; the claim needs a directory with none")
	}

	report := stepGit(root)

	if report.Outcome != OutcomeDelivered {
		t.Fatalf("git step reported %v (%s), want OutcomeDelivered on a directory with no repository", report.Outcome, report.Detail)
	}
	if !exists(root, ".git") {
		t.Fatal("the git step reported delivered but no .git exists; the repository was not created")
	}
	// A real repository, not just a directory named .git: git itself must recognize it.
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git does not recognize the created directory as a repository: %v\n%s", err, out)
	}
}

// TestInit_ExistingGitRepositoryIsLeftUntouched (SPEC-069 CLM-037).
//
// In a directory that IS already a repository, no git initialization runs and HEAD,
// config and refs are byte-unchanged. The assertion is on all THREE, deliberately: a
// re-init that happened to be harmless still fails this claim, and "no error" would
// pass over it.
func TestInit_ExistingGitRepositoryIsLeftUntouched(t *testing.T) {
	requireGitBinary(t)
	root := t.TempDir()
	initGitRepository(t, root)

	before := snapshotGitState(t, root)

	report := stepGit(root)

	if report.Outcome != OutcomeConverged {
		t.Fatalf("git step reported %v (%s) over an existing repository, want OutcomeConverged", report.Outcome, report.Detail)
	}

	after := snapshotGitState(t, root)
	for _, key := range []string{"HEAD", "config", "refs"} {
		if before[key] != after[key] {
			t.Fatalf("the git step mutated %s of an existing repository.\nbefore: %q\nafter:  %q", key, before[key], after[key])
		}
	}
}

// snapshotGitState captures the three pieces of git state CLM-037 protects: the HEAD
// file, the config file, and the full ref listing.
func snapshotGitState(t *testing.T, root string) map[string]string {
	t.Helper()
	gitDir := filepath.Join(root, ".git")

	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		t.Fatalf("reading HEAD: %v", err)
	}
	config, err := os.ReadFile(filepath.Join(gitDir, "config"))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	cmd := exec.Command("git", "show-ref", "--head")
	cmd.Dir = root
	refs, _ := cmd.Output()

	return map[string]string{
		"HEAD":   string(head),
		"config": string(config),
		"refs":   string(refs),
	}
}

// TestInit_GitStepReportsAFailedInitAsABrokenPromise covers the failure path: git init
// itself refusing. It is additive — CLM-036/037 are the mandated claims — but the path
// is real, and an unreported git failure would leave a consumer with a project whose
// very first step silently did nothing.
func TestInit_GitStepReportsAFailedInitAsABrokenPromise(t *testing.T) {
	requireGitBinary(t)

	// A directory that does not exist: the child process cannot be started there, which
	// is the honest way to make `git init` fail without depending on git's own internals.
	missing := filepath.Join(t.TempDir(), "no-such-directory")

	report := stepGit(missing)

	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("a failed git init reported %v, want OutcomeBrokenPromise", report.Outcome)
	}
	if !strings.Contains(report.Detail, missing) {
		t.Fatalf("the failure report does not name the directory it tried to initialize.\ngot: %s", report.Detail)
	}
}

// TestInit_CapturedGitOutputContributesNothingWhenThereIsNone pins the empty case of
// the captured-output renderer: a command that said nothing must add nothing, or every
// clean report carries a trailing blank line.
func TestInit_CapturedGitOutputContributesNothingWhenThereIsNone(t *testing.T) {
	if got := formatCapturedOutput(nil); got != "" {
		t.Fatalf("formatCapturedOutput(nil) = %q, want the empty string", got)
	}
	if got := formatCapturedOutput([]byte("   \n\n ")); got != "" {
		t.Fatalf("formatCapturedOutput(whitespace) = %q, want the empty string", got)
	}
	if got := formatCapturedOutput([]byte("fatal: something went wrong\n")); got != "\nfatal: something went wrong" {
		t.Fatalf("formatCapturedOutput dropped or mangled real output: %q", got)
	}
}
