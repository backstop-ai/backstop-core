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

// TestGateScope_FilterViolations_ExportedWrapper covers the exported
// (*GateScope).FilterViolations bridge (ISSUE-070) — the entry point
// packValidatorStep uses so out-of-package callers apply the SAME diff-scope filter.
// It exercises every routing branch: a nil scope and an ModeAll scope pass violations
// through unchanged (whole-repo sweep), a ProjectWide (exempt) violation survives even
// out-of-scope, an out-of-scope non-exempt violation is dropped, and an in-scope
// violation is kept.
func TestGateScope_FilterViolations_ExportedWrapper(t *testing.T) {
	vs := []Violation{
		{Rule: "no-undef", File: "a.ts"},
		{Rule: "no-undef", File: "b.ts"},
	}

	// Nil scope: no filtering — the whole-repo sweep returns the input unchanged.
	if got := (*GateScope)(nil).FilterViolations(vs); len(got) != 2 {
		t.Errorf("nil scope must pass all violations through, got %d", len(got))
	}

	// ModeAll scope: no filtering either.
	all, err := ComputeGateScope(t.TempDir(), GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("compute all scope: %v", err)
	}
	if got := all.FilterViolations(vs); len(got) != 2 {
		t.Errorf("ModeAll scope must pass all violations through, got %d", len(got))
	}

	// Diff scope with only a.ts in scope: keep in-scope + ProjectWide-exempt, drop the
	// out-of-scope non-exempt one.
	scope := newGateScope("/repo", GateScopeModeDiff, []string{"a.ts"}, nil)
	mixed := []Violation{
		{Rule: "no-undef", File: "a.ts"},                    // in scope -> keep
		{Rule: "no-undef", File: "z.ts"},                    // out of scope -> drop
		{Rule: "TS2304", File: "z.ts", ProjectWide: true},   // exempt -> keep despite out of scope
	}
	got := scope.FilterViolations(mixed)
	var keptInScope, keptExempt, droppedPresent bool
	for _, v := range got {
		switch {
		case v.Rule == "no-undef" && v.File == "a.ts":
			keptInScope = true
		case v.Rule == "TS2304":
			keptExempt = true
		case v.Rule == "no-undef" && v.File == "z.ts":
			droppedPresent = true
		}
	}
	if !keptInScope {
		t.Error("in-scope violation must be kept by the exported wrapper")
	}
	if !keptExempt {
		t.Error("ProjectWide-exempt violation must be kept even when out of scope")
	}
	if droppedPresent {
		t.Error("out-of-scope non-exempt violation must be dropped by the exported wrapper")
	}
}

// TestComputeGateScope_FileAndErrorModes covers the ComputeGateScope ModeFile branch
// (success + the empty-files error) and the unknown-mode default error.
func TestComputeGateScope_FileAndErrorModes(t *testing.T) {
	root := t.TempDir()

	scope, err := ComputeGateScope(root, GateScopeModeFile, []string{"x.go", "y.go"})
	if err != nil {
		t.Fatalf("ModeFile with files: %v", err)
	}
	if scope.Mode != GateScopeModeFile || len(scope.Files) != 2 {
		t.Fatalf("ModeFile scope = %#v, want file mode with 2 files", scope)
	}

	if _, err := ComputeGateScope(root, GateScopeModeFile, nil); err == nil {
		t.Error("ModeFile with no files must error")
	}

	if _, err := ComputeGateScope(root, GateScopeMode("bogus"), nil); err == nil {
		t.Error("unknown scope mode must error")
	}
}

// TestComputeGateScope_DiffAgainstRemoteBranch covers the resolveGateScopeDiff
// remote-branch success path: when a refs/remotes/origin/main exists, the scope is the
// merge-base diff plus untracked files (the real-CI path). It plants an origin/main
// ref, modifies a tracked file, and adds an untracked file, asserting both land in scope.
func TestComputeGateScope_DiffAgainstRemoteBranch(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(root, "tracked.go"), "package main\n")
	runGit(t, root, "add", "tracked.go")
	runGit(t, root, "commit", "-m", "initial")
	// Materialize the remote-tracking ref the diff path looks for (origin/main).
	runGit(t, root, "update-ref", "refs/remotes/origin/main", "HEAD")

	// A tracked change plus an untracked add, both after the merge-base.
	writeFile(t, filepath.Join(root, "tracked.go"), "package main\nfunc changed() {}\n")
	writeFile(t, filepath.Join(root, "brand_new.go"), "package main\n")

	scope, err := ComputeGateScope(root, GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope diff (remote branch): %v", err)
	}
	if !scope.Contains("tracked.go") {
		t.Errorf("merge-base diff must include the modified tracked file, got %#v", scope.Files)
	}
	if !scope.Contains("brand_new.go") {
		t.Errorf("remote-branch diff must include untracked files, got %#v", scope.Files)
	}
	// The remote-branch path records no warning (unlike the local-only fallback).
	if len(scope.Warnings) != 0 {
		t.Errorf("remote-branch diff should record no warning, got %#v", scope.Warnings)
	}
}

// TestComputeGateScope_DiffNotGitRepoFallsBackToAll covers the resolveGateScopeDiff
// not-a-git-repository fallback (it delegates to resolveGateScopeAll and records a
// warning) and, via the plain subdirectory it plants, the resolveGateScopeAll
// non-hidden directory recursion branch.
func TestComputeGateScope_DiffNotGitRepoFallsBackToAll(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "top.go"), "package main\n")
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "sub", "nested.go"), "package main\n")

	scope, err := ComputeGateScope(root, GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope diff (non-git): %v", err)
	}
	// Fell back to a full walk: both the top-level and nested files are present.
	if !scope.Contains("top.go") || !scope.Contains(filepath.Join("sub", "nested.go")) {
		t.Fatalf("non-git diff fallback should walk all files, got %#v", scope.Files)
	}
	if len(scope.Warnings) == 0 {
		t.Error("non-git diff fallback must record a warning")
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
