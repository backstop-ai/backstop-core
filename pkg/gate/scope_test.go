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

// TestGate_FilterViolations_BuildPassExemptFromScopeFilter pins CLM-008's
// gate-layer half (Ratified Design Constraint 3): a project-wide build/typecheck
// violation must NEVER be scope-filtered, even when it references an
// out-of-scope file. Using a NON-EMPTY parser-populated Rule (e.g. "TS2304") is
// load-bearing — a Rule=="build" string-match exemption would PASS against the
// broken impl, so the test pins the STRUCTURAL ProjectWide field, not the Rule
// string. A lint/semgrep violation (ProjectWide==false) on an out-of-scope file
// IS still filtered.
func TestGate_FilterViolations_BuildPassExemptFromScopeFilter(t *testing.T) {
	scope := newGateScope("/repo", GateScopeModeDiff, []string{"a.ts"}, nil)

	buildViolation := Violation{
		Rule:        "TS2304", // non-empty, parser-populated — NOT "build"
		File:        "b.ts",   // OUT of scope (only a.ts is in scope)
		Message:     "Cannot find name 'foo'.",
		Severity:    "error",
		ProjectWide: true,
	}
	lintViolation := Violation{
		Rule:        "no-undef",
		File:        "c.ts", // OUT of scope
		Message:     "'x' is not defined.",
		Severity:    "error",
		ProjectWide: false,
	}
	inScopeLint := Violation{
		Rule:        "no-undef",
		File:        "a.ts", // IN scope
		Message:     "'y' is not defined.",
		Severity:    "error",
		ProjectWide: false,
	}

	filtered := filterViolations(scope, []Violation{buildViolation, lintViolation, inScopeLint})

	var sawBuild, sawOutOfScopeLint, sawInScopeLint bool
	for _, v := range filtered {
		switch {
		case v.Rule == "TS2304":
			sawBuild = true
		case v.Rule == "no-undef" && v.File == "c.ts":
			sawOutOfScopeLint = true
		case v.Rule == "no-undef" && v.File == "a.ts":
			sawInScopeLint = true
		}
	}

	if !sawBuild {
		t.Error("project-wide build violation (Rule=TS2304, out-of-scope b.ts) was filtered out; it must be exempt via the structural ProjectWide field")
	}
	if sawOutOfScopeLint {
		t.Error("out-of-scope lint violation (ProjectWide=false) survived the filter; it must be filtered")
	}
	if !sawInScopeLint {
		t.Error("in-scope lint violation was filtered out; in-scope violations must be retained")
	}
}
