package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

func TestGateScope_IncludesUntrackedFiles(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(root, "tracked.go"), "package main\n")
	runGit(t, root, "add", "tracked.go")
	runGit(t, root, "commit", "-m", "initial")
	writeFile(t, filepath.Join(root, "tracked.go"), "package main\nfunc changed() {}\n")
	writeFile(t, filepath.Join(root, "new.go"), "package main\n")

	scope, err := ComputeGateScope(root, GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	if !scope.Contains("tracked.go") || !scope.Contains("new.go") {
		t.Fatalf("expected tracked and untracked files in scope, got %#v", scope.Files)
	}
}

func TestGateScope_ComputedOnce(t *testing.T) {
	scope := newGateScope(t.TempDir(), GateScopeModeFile, []string{"b.go", "a.go", "a.go"}, nil)
	if len(scope.Files) != 2 || scope.Files[0] != "a.go" || scope.Files[1] != "b.go" {
		t.Fatalf("expected stable deduplicated file list, got %#v", scope.Files)
	}
}

func TestGate_EmptyDiff(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(root, "tracked.go"), "package main\n")
	runGit(t, root, "add", "tracked.go")
	runGit(t, root, "commit", "-m", "initial")

	scope, err := ComputeGateScope(root, GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	if !scope.Empty() {
		t.Fatalf("expected empty diff scope, got %#v", scope.Files)
	}
}

func TestGateScope_ModeFromCheck(t *testing.T) {
	if GateScopeModeFromCheck(check.ScopeModeAll) != GateScopeModeAll {
		t.Fatal("expected all mode mapping")
	}
	if GateScopeModeFromCheck(check.ScopeModeFile) != GateScopeModeFile {
		t.Fatal("expected file mode mapping")
	}
	if GateScopeModeFromCheck(check.ScopeModeDiff) != GateScopeModeDiff {
		t.Fatal("expected diff mode mapping")
	}
}

func TestGateScope_AllModeWalksFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "visible.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden", "ignored.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	scope, err := ComputeGateScope(dir, GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("compute all scope: %v", err)
	}
	if !scope.Contains("anything.go") {
		t.Fatal("all scope should contain every file")
	}
	if len(scope.Files) != 1 || scope.Files[0] != "visible.go" {
		t.Fatalf("expected only visible file, got %#v", scope.Files)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
