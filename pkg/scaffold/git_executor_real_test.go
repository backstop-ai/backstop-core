package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTempGitRepo creates a temporary git repository for testing.
func initTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init setup %v failed: %v\n%s", args, err, out)
		}
	}

	// Create an initial commit so tags can be created
	f := filepath.Join(dir, "README.md")
	if err := os.WriteFile(f, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git commit setup %v failed: %v\n%s", args, err, out)
		}
	}

	return dir
}

func TestRealGitExecutor_IsGitAvailable(t *testing.T) {
	r := &RealGitExecutor{}
	// git should be available in CI/local dev
	if !r.IsGitAvailable() {
		t.Skip("git not available on this system")
	}
}

func TestRealGitExecutor_IsGitRepo(t *testing.T) {
	dir := initTempGitRepo(t)
	// Change to the temp repo directory for the check
	t.Chdir(dir)

	r := &RealGitExecutor{}
	if !r.IsGitRepo() {
		t.Fatal("expected IsGitRepo to return true in a git repo")
	}
}

func TestRealGitExecutor_IsGitRepo_NotARepo(t *testing.T) {
	dir := t.TempDir() // plain directory, not a git repo
	t.Chdir(dir)

	r := &RealGitExecutor{}
	if r.IsGitRepo() {
		t.Fatal("expected IsGitRepo to return false outside a git repo")
	}
}

func TestRealGitExecutor_ListTags(t *testing.T) {
	dir := initTempGitRepo(t)
	t.Chdir(dir)

	// Create a test tag
	cmd := exec.Command("git", "tag", "-a", "backstop/spec/001", "-m", "test tag")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("creating test tag: %v\n%s", err, out)
	}

	r := &RealGitExecutor{}
	tags, err := r.ListTags("backstop/spec/*")
	if err != nil {
		t.Fatalf("ListTags error: %v", err)
	}
	if len(tags) != 1 || tags[0] != "backstop/spec/001" {
		t.Fatalf("expected [backstop/spec/001], got %v", tags)
	}
}

func TestRealGitExecutor_ListTags_Empty(t *testing.T) {
	dir := initTempGitRepo(t)
	t.Chdir(dir)

	r := &RealGitExecutor{}
	tags, err := r.ListTags("backstop/spec/*")
	if err != nil {
		t.Fatalf("ListTags error: %v", err)
	}
	if tags != nil {
		t.Fatalf("expected nil tags for empty result, got %v", tags)
	}
}

func TestRealGitExecutor_CreateAnnotatedTag(t *testing.T) {
	dir := initTempGitRepo(t)
	t.Chdir(dir)

	r := &RealGitExecutor{}
	err := r.CreateAnnotatedTag("backstop/spec/001", "test reservation")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag error: %v", err)
	}

	// Verify the tag was created
	tags, _ := r.ListTags("backstop/spec/*")
	if len(tags) != 1 || tags[0] != "backstop/spec/001" {
		t.Fatalf("expected tag to exist after creation, got %v", tags)
	}
}

func TestRealGitExecutor_PushTag_NoRemote(t *testing.T) {
	dir := initTempGitRepo(t)
	t.Chdir(dir)

	// Create a tag first
	r := &RealGitExecutor{}
	_ = r.CreateAnnotatedTag("backstop/spec/001", "test")

	// Push should fail (no remote configured)
	err := r.PushTag("backstop/spec/001")
	if err == nil {
		t.Fatal("expected error pushing to non-existent remote")
	}
}

func TestRealGitExecutor_FetchTags_InRepo(t *testing.T) {
	dir := initTempGitRepo(t)
	t.Chdir(dir)

	// Give the repo a local origin that holds a tag the repo lacks, so the
	// fetch has a known, observable effect.
	origin := initTempGitRepo(t)
	for _, args := range [][]string{
		{"git", "-C", origin, "tag", "-a", "backstop/spec/007", "-m", "origin tag"},
		{"git", "-C", dir, "remote", "add", "origin", origin},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("remote setup %v failed: %v\n%s", args, err, out)
		}
	}

	r := &RealGitExecutor{}
	if err := r.FetchTags(); err != nil {
		t.Fatalf("FetchTags error: %v", err)
	}
	tags, err := r.ListTags("backstop/spec/*")
	if err != nil {
		t.Fatalf("ListTags error: %v", err)
	}
	if len(tags) != 1 || tags[0] != "backstop/spec/007" {
		t.Fatalf("expected the origin tag [backstop/spec/007] after FetchTags, got %v", tags)
	}
}
