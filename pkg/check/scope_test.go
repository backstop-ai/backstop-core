package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCodeCheck_DefaultScope_UsesGitMergeBase verifies diff mode calls git
// merge-base with origin/main. (CLM-005)
func TestCodeCheck_DefaultScope_UsesGitMergeBase(t *testing.T) {
	var calledRemote string
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			calledRemote = remote
			return "abc123", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			return []string{"pkg/foo.go", "pkg/bar.go"}, nil
		},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledRemote != "origin/main" {
		t.Errorf("merge-base called with %q, want %q", calledRemote, "origin/main")
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0", len(warnings))
	}
}

// TestCodeCheck_DefaultScope_NonGitFallsBackToAll verifies non-git directory
// returns all-scope file list with a warning. (CLM-006)
func TestCodeCheck_DefaultScope_NonGitFallsBackToAll(t *testing.T) {
	// Create a temp directory with some files (not a git repo)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main"), 0o644)

	mock := &mockGitExecutor{
		isGitRepo: false,
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock, withProjectDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected files from all-scope fallback, got none")
	}
	if len(warnings) == 0 {
		t.Error("expected a warning about non-git fallback, got none")
	}
	foundWarning := false
	for _, w := range warnings {
		if contains(w, "git") || contains(w, "all") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("warnings %v do not mention git or all-scope fallback", warnings)
	}
}

// TestCodeCheck_AllFlag_FullCodebase verifies ScopeModeAll walks directory
// for all files, no git invocation. (CLM-007)
func TestCodeCheck_AllFlag_FullCodebase(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.MkdirAll(filepath.Join(dir, "pkg"), 0o755)
	os.WriteFile(filepath.Join(dir, "pkg", "util.go"), []byte("package pkg"), 0o644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0o644)

	files, warnings, err := resolveScopeWithGit(ScopeModeAll, "", nil, withProjectDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) < 3 {
		t.Errorf("got %d files, want at least 3", len(files))
	}
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0", len(warnings))
	}
}

// TestCodeCheck_AllFlag_NoGitCall verifies ScopeModeAll does not invoke git
// at all. (CLM-008)
func TestCodeCheck_AllFlag_NoGitCall(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	gitCalled := false
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			gitCalled = true
			return "", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			gitCalled = true
			return nil, nil
		},
		diffLocalFn: func() ([]string, error) {
			gitCalled = true
			return nil, nil
		},
	}

	_, _, err := resolveScopeWithGit(ScopeModeAll, "", mock, withProjectDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitCalled {
		t.Error("ScopeModeAll invoked git, but should not")
	}
}

// TestCodeCheck_DefaultScope_LocalStagedAndUnstaged verifies local scope
// detects staged and unstaged changes when no remote divergence exists. (CLM-041)
func TestCodeCheck_DefaultScope_LocalStagedAndUnstaged(t *testing.T) {
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "", errNoRemote
		},
		diffLocalFn: func() ([]string, error) {
			return []string{"staged.go", "unstaged.go"}, nil
		},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}
	if len(warnings) == 0 {
		t.Error("expected warning about local-only fallback, got none")
	}
}

// TestCodeCheck_ChangedFiles_MergeBaseOriginMain verifies step 1: merge-base
// with origin/main succeeds. (CLM-037)
func TestCodeCheck_ChangedFiles_MergeBaseOriginMain(t *testing.T) {
	calls := []string{}
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			calls = append(calls, "merge-base:"+remote)
			return "abc123", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			calls = append(calls, "diff:"+base)
			return []string{"changed.go"}, nil
		},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "changed.go" {
		t.Errorf("files = %v, want [changed.go]", files)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	if len(calls) < 1 || calls[0] != "merge-base:origin/main" {
		t.Errorf("calls = %v, want merge-base:origin/main first", calls)
	}
}

// TestCodeCheck_ChangedFiles_FallbackOriginMaster verifies step 2: origin/main
// fails, falls back to origin/master. (CLM-038)
func TestCodeCheck_ChangedFiles_FallbackOriginMaster(t *testing.T) {
	calls := []string{}
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			calls = append(calls, "merge-base:"+remote)
			if remote == "origin/main" {
				return "", errNoRemote
			}
			return "def456", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			return []string{"fallback.go"}, nil
		},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "fallback.go" {
		t.Errorf("files = %v, want [fallback.go]", files)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	// Verify both merge-base calls were made
	if len(calls) < 2 || calls[1] != "merge-base:origin/master" {
		t.Errorf("calls = %v, want origin/master as second attempt", calls)
	}
}

// TestCodeCheck_ChangedFiles_FallbackLocalStagedUnstaged verifies step 3:
// neither remote exists, falls back to local diff with warning. (CLM-039)
func TestCodeCheck_ChangedFiles_FallbackLocalStagedUnstaged(t *testing.T) {
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "", errNoRemote
		},
		diffLocalFn: func() ([]string, error) {
			return []string{"local-change.go"}, nil
		},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "local-change.go" {
		t.Errorf("files = %v, want [local-change.go]", files)
	}
	if len(warnings) == 0 {
		t.Error("expected warning about local fallback")
	}
}

// TestCodeCheck_DiffScope_IncludesUntrackedFiles verifies the merge-base path
// appends untracked files (git ls-files --others --exclude-standard) alongside
// the tracked diff result, matching the gate resolver. (CLM-001)
func TestCodeCheck_DiffScope_IncludesUntrackedFiles(t *testing.T) {
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "abc123", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			return []string{"tracked.go"}, nil
		},
		untrackedFiles: []string{"brand-new.go"},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsFile(files, "tracked.go") {
		t.Errorf("files = %v, want to contain tracked.go", files)
	}
	if !containsFile(files, "brand-new.go") {
		t.Errorf("files = %v, want to contain untracked brand-new.go", files)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
}

// TestCodeCheck_DiffScope_MergeBase_UntrackedErrorIsBestEffort verifies that
// an UntrackedFiles error on the merge-base path does not fail scope
// resolution: the tracked diff is returned unchanged with no error, mirroring
// pkg/gate/scope.go:117. (CLM-001)
func TestCodeCheck_DiffScope_MergeBase_UntrackedErrorIsBestEffort(t *testing.T) {
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "abc123", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			return []string{"tracked.go"}, nil
		},
		untrackedErr: fmt.Errorf("git ls-files exploded"),
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "tracked.go" {
		t.Errorf("files = %v, want [tracked.go] (untracked error must not append)", files)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
}

// TestCodeCheck_DiffScope_LocalFallback_IncludesUntrackedFiles verifies the
// local staged+unstaged fallback path (no remote base) appends untracked files
// and still emits the no-remote warning. (CLM-002)
func TestCodeCheck_DiffScope_LocalFallback_IncludesUntrackedFiles(t *testing.T) {
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "", errNoRemote
		},
		diffLocalFn: func() ([]string, error) {
			return []string{"staged.go", "unstaged.go"}, nil
		},
		untrackedFiles: []string{"brand-new.go"},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsFile(files, "staged.go") || !containsFile(files, "unstaged.go") {
		t.Errorf("files = %v, want to contain staged.go and unstaged.go", files)
	}
	if !containsFile(files, "brand-new.go") {
		t.Errorf("files = %v, want to contain untracked brand-new.go", files)
	}
	if len(warnings) == 0 {
		t.Error("expected no-remote warning, got none")
	}
}

// TestCodeCheck_DiffScope_LocalFallback_UntrackedErrorIsBestEffort verifies
// that an UntrackedFiles error on the local fallback path is ignored: scope
// still resolves to the local diff and the no-remote warning is still present,
// mirroring pkg/gate/scope.go:129. (CLM-002)
func TestCodeCheck_DiffScope_LocalFallback_UntrackedErrorIsBestEffort(t *testing.T) {
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "", errNoRemote
		},
		diffLocalFn: func() ([]string, error) {
			return []string{"staged.go", "unstaged.go"}, nil
		},
		untrackedErr: fmt.Errorf("git ls-files exploded"),
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 || !containsFile(files, "staged.go") || !containsFile(files, "unstaged.go") {
		t.Errorf("files = %v, want exactly [staged.go unstaged.go] (untracked error must not append)", files)
	}
	if len(warnings) == 0 {
		t.Error("expected no-remote warning even when untracked fetch fails, got none")
	}
}

// TestCodeCheck_ChangedFiles_NonGitFallbackToAll verifies step 4: not a git
// repo, falls back to --all with warning. (CLM-040)
func TestCodeCheck_ChangedFiles_NonGitFallbackToAll(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	mock := &mockGitExecutor{
		isGitRepo: false,
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock, withProjectDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected all-scope files, got none")
	}
	if len(warnings) == 0 {
		t.Error("expected warning about non-git fallback")
	}
}

// TestCodeCheck_ScopeModeFile_ExistingFile verifies ScopeModeFile returns
// the file when it exists.
func TestCodeCheck_ScopeModeFile_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.go")
	os.WriteFile(f, []byte("package main"), 0o644)

	files, warnings, err := resolveScopeWithGit(ScopeModeFile, f, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != f {
		t.Errorf("files = %v, want [%s]", files, f)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
}

// TestCodeCheck_ScopeModeFile_MissingFile verifies ScopeModeFile returns
// error for non-existent file.
func TestCodeCheck_ScopeModeFile_MissingFile(t *testing.T) {
	_, _, err := resolveScopeWithGit(ScopeModeFile, "/nonexistent/file.go", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestCodeCheck_ScopeModeFile_EmptyPath verifies ScopeModeFile returns error
// for empty path.
func TestCodeCheck_ScopeModeFile_EmptyPath(t *testing.T) {
	_, _, err := resolveScopeWithGit(ScopeModeFile, "", nil)
	if err == nil {
		t.Fatal("expected error for empty file path")
	}
}

// TestCodeCheck_SplitLines verifies splitLines filters empty lines.
func TestCodeCheck_SplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a\nb\nc\n", 3},
		{"", 0},
		{"\n\n\n", 0},
		{"single", 1},
	}
	for _, tc := range tests {
		got := splitLines(tc.input)
		if len(got) != tc.want {
			t.Errorf("splitLines(%q) = %d lines, want %d", tc.input, len(got), tc.want)
		}
	}
}

// TestCodeCheck_ScopeModeAll_EmptyProjectDir verifies resolveScopeAll returns
// error when project directory is empty string.
func TestCodeCheck_ScopeModeAll_EmptyProjectDir(t *testing.T) {
	_, _, err := resolveScopeWithGit(ScopeModeAll, "", nil)
	if err == nil {
		t.Fatal("expected error for empty project dir, got nil")
	}
}

// TestCodeCheck_ScopeModeAll_HiddenDirsSkipped verifies resolveScopeAll skips
// hidden directories (starting with dot).
func TestCodeCheck_ScopeModeAll_HiddenDirsSkipped(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	hiddenDir := filepath.Join(dir, ".hidden")
	os.MkdirAll(hiddenDir, 0o755)
	os.WriteFile(filepath.Join(hiddenDir, "secret.go"), []byte("package hidden"), 0o644)

	files, _, err := resolveScopeWithGit(ScopeModeAll, "", nil, withProjectDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range files {
		if filepath.Base(f) == "secret.go" {
			t.Error("hidden directory file should be skipped")
		}
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1 (only main.go)", len(files))
	}
}

// TestCodeCheck_ScopeDiff_MergeBaseSucceedsDiffFails verifies that when
// merge-base succeeds for origin/main but diff-name-only fails, falls through
// to origin/master.
func TestCodeCheck_ScopeDiff_MergeBaseSucceedsDiffFails(t *testing.T) {
	calls := []string{}
	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			calls = append(calls, "merge-base:"+remote)
			return "abc123", nil
		},
		diffNameOnlyFn: func(base string) ([]string, error) {
			calls = append(calls, "diff:"+base)
			if len(calls) <= 2 {
				// First diff call (for origin/main) fails
				return nil, fmt.Errorf("diff failed")
			}
			// Second diff call (for origin/master) succeeds
			return []string{"recovered.go"}, nil
		},
	}

	files, _, err := resolveScopeWithGit(ScopeModeDiff, "", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "recovered.go" {
		t.Errorf("files = %v, want [recovered.go]", files)
	}
}

// TestCodeCheck_ScopeDiff_AllRemotesFailDiffLocalFails verifies fallback to
// all-scope when both remotes and local diff all fail.
func TestCodeCheck_ScopeDiff_AllRemotesFailDiffLocalFails(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	mock := &mockGitExecutor{
		isGitRepo: true,
		mergeBaseFn: func(remote string) (string, error) {
			return "", errNoRemote
		},
		diffLocalFn: func() ([]string, error) {
			return nil, fmt.Errorf("local diff failed")
		},
	}

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", mock, withProjectDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected fallback to all-scope files, got none")
	}
	foundWarning := false
	for _, w := range warnings {
		if contains(w, "git diff failed") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected warning about git diff failure, got: %v", warnings)
	}
}

// TestCodeCheck_ScopeDiff_NilGitExecutor verifies that nil git executor
// triggers all-scope fallback.
func TestCodeCheck_ScopeDiff_NilGitExecutor(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	files, warnings, err := resolveScopeWithGit(ScopeModeDiff, "", nil, withProjectDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected files from all-scope fallback")
	}
	if len(warnings) == 0 {
		t.Error("expected warning about non-git fallback")
	}
}

// TestCodeCheck_Scope_UnknownMode verifies unknown scope mode returns error.
func TestCodeCheck_Scope_UnknownMode(t *testing.T) {
	_, _, err := resolveScopeWithGit(ScopeMode(99), "", nil)
	if err == nil {
		t.Fatal("expected error for unknown scope mode, got nil")
	}
}

// TestCodeCheck_DefaultGitExecutor_IsGitRepo verifies IsGitRepo against real
// git repos and non-repos.
func TestCodeCheck_DefaultGitExecutor_IsGitRepo(t *testing.T) {
	// The current project directory should be a git repo
	g := &DefaultGitExecutor{Dir: "."}
	if !g.IsGitRepo() {
		t.Skip("test requires running inside a git repo")
	}

	// A temp dir is NOT a git repo
	nonRepo := t.TempDir()
	g2 := &DefaultGitExecutor{Dir: nonRepo}
	if g2.IsGitRepo() {
		t.Error("temp dir should not be a git repo")
	}
}

// TestCodeCheck_DefaultGitExecutor_MergeBase verifies MergeBase with a
// non-existent remote returns error.
func TestCodeCheck_DefaultGitExecutor_MergeBase(t *testing.T) {
	g := &DefaultGitExecutor{Dir: "."}
	if !g.IsGitRepo() {
		t.Skip("test requires running inside a git repo")
	}

	// Non-existent remote should fail
	_, err := g.MergeBase("nonexistent-remote/nonexistent-branch")
	if err == nil {
		t.Error("expected error for non-existent remote, got nil")
	}
}

// TestCodeCheck_DefaultGitExecutor_DiffNameOnly verifies DiffNameOnly with
// a valid base commit.
func TestCodeCheck_DefaultGitExecutor_DiffNameOnly(t *testing.T) {
	g := &DefaultGitExecutor{Dir: "."}
	if !g.IsGitRepo() {
		t.Skip("test requires running inside a git repo")
	}

	// Use HEAD as base — should return empty diff
	files, err := g.DiffNameOnly("HEAD")
	if err != nil {
		t.Fatalf("DiffNameOnly(HEAD): %v", err)
	}
	// HEAD..HEAD should be empty
	_ = files
}

// TestCodeCheck_DefaultGitExecutor_DiffLocal verifies DiffLocal runs without error.
func TestCodeCheck_DefaultGitExecutor_DiffLocal(t *testing.T) {
	g := &DefaultGitExecutor{Dir: "."}
	if !g.IsGitRepo() {
		t.Skip("test requires running inside a git repo")
	}

	files, err := g.DiffLocal()
	if err != nil {
		t.Fatalf("DiffLocal: %v", err)
	}
	// Result is environment-dependent; just verify no error
	_ = files
}

// TestCodeCheck_DefaultGitExecutor_UntrackedFiles verifies UntrackedFiles runs
// without error against a real repo and returns a temp untracked file. (CLM-001)
func TestCodeCheck_DefaultGitExecutor_UntrackedFiles(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	g := &DefaultGitExecutor{Dir: dir}
	if !g.IsGitRepo() {
		t.Skip("test requires a working git repo")
	}

	untrackedPath := filepath.Join(dir, "brand-new.go")
	if err := os.WriteFile(untrackedPath, []byte("package main"), 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	files, err := g.UntrackedFiles()
	if err != nil {
		t.Fatalf("UntrackedFiles: %v", err)
	}
	if !containsFile(files, "brand-new.go") {
		t.Errorf("files = %v, want to contain brand-new.go", files)
	}
}

// TestCodeCheck_ResolveScope_ScopeModeFile verifies ResolveScope with
// ScopeModeFile works for an existing file.
func TestCodeCheck_ResolveScope_ScopeModeFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.go")
	os.WriteFile(f, []byte("package main"), 0o644)

	files, warnings, err := ResolveScope(ScopeModeFile, f)
	if err != nil {
		t.Fatalf("ResolveScope: %v", err)
	}
	if len(files) != 1 || files[0] != f {
		t.Errorf("files = %v, want [%s]", files, f)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
}

// --- Test helpers ---

// mockGitExecutor is a test double for GitExecutor.
type mockGitExecutor struct {
	isGitRepo      bool
	mergeBaseFn    func(remote string) (string, error)
	diffNameOnlyFn func(base string) ([]string, error)
	diffLocalFn    func() ([]string, error)
	untrackedFiles []string
	untrackedErr   error
}

func (m *mockGitExecutor) IsGitRepo() bool {
	return m.isGitRepo
}

func (m *mockGitExecutor) MergeBase(remote string) (string, error) {
	if m.mergeBaseFn != nil {
		return m.mergeBaseFn(remote)
	}
	return "", errNoRemote
}

func (m *mockGitExecutor) DiffNameOnly(base string) ([]string, error) {
	if m.diffNameOnlyFn != nil {
		return m.diffNameOnlyFn(base)
	}
	return nil, nil
}

func (m *mockGitExecutor) DiffLocal() ([]string, error) {
	if m.diffLocalFn != nil {
		return m.diffLocalFn()
	}
	return nil, nil
}

func (m *mockGitExecutor) UntrackedFiles() ([]string, error) {
	if m.untrackedErr != nil {
		return nil, m.untrackedErr
	}
	return m.untrackedFiles, nil
}

func containsFile(files []string, target string) bool {
	for _, f := range files {
		if f == target {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
