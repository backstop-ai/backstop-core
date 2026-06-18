package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitCmd runs a git subcommand in dir, failing the test on error.
func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
}

// TestGitSHA_ReturnsHeadOfRealRepo verifies gitSHA returns the resolved HEAD
// commit hash for a real repository with at least one commit.
func TestGitSHA_ReturnsHeadOfRealRepo(t *testing.T) {
	dir := t.TempDir()
	gitCmd(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", ".")
	gitCmd(t, dir, "commit", "-m", "initial")

	sha := gitSHA(dir)
	if len(sha) != 40 {
		t.Fatalf("expected a 40-char HEAD sha, got %q (len %d)", sha, len(sha))
	}
	// Cross-check against git rev-parse executed independently.
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	if got, want := sha, string(out[:40]); got != want {
		t.Errorf("gitSHA = %q, want %q", got, want)
	}
}

// TestGitSHA_ReturnsEmptyOutsideRepo verifies gitSHA returns an empty string
// when the directory is not a git repository (rev-parse errors).
func TestGitSHA_ReturnsEmptyOutsideRepo(t *testing.T) {
	dir := t.TempDir() // no git init
	if sha := gitSHA(dir); sha != "" {
		t.Errorf("expected empty sha outside a repo, got %q", sha)
	}
}
