package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// substantiveness_helpers_test.go covers the language-agnostic helper edge cases the
// set-join and dispatch glue rely on: file-join robustness (sameFile), the fail-loud
// severity default (nonEmptySeverity), and the string-only same-package derivation
// (goFilePackageMatchesTarget). These are real behavior assertions, not coverage filler:
// each pins a disposition the gate verdict depends on.

// TestSameFile_EmptyAndBasenameJoin — the (FilePath, FuncName) join's file side: empty
// paths never match (so a finding with no file can't leak into a test), an exact clean
// match joins, and a basename match joins absolute-vs-relative shapes.
func TestSameFile_EmptyAndBasenameJoin(t *testing.T) {
	if sameFile("", "x_test.go") {
		t.Errorf("an empty finding path must NOT join any test")
	}
	if sameFile("x_test.go", "") {
		t.Errorf("an empty test path must NOT be joined by any finding")
	}
	if !sameFile("/abs/dir/x_test.go", "dir/x_test.go") {
		t.Errorf("basename-equal paths must join (absolute finding vs relative test)")
	}
	if sameFile("a/x_test.go", "b/y_test.go") {
		t.Errorf("different basenames must NOT join")
	}
}

// TestNonEmptySeverity_DefaultsToError — an empty severity defaults to "error"
// (fail-loud), while a present severity is preserved verbatim.
func TestNonEmptySeverity_DefaultsToError(t *testing.T) {
	if got := nonEmptySeverity(""); got != "error" {
		t.Errorf("empty severity must default to error (fail-loud); got %q", got)
	}
	if got := nonEmptySeverity("warning"); got != "warning" {
		t.Errorf("present severity must be preserved; got %q", got)
	}
}

// TestGoFilePackageMatchesTarget_EdgeCases — the string-only same-package derivation:
// an empty target is never same-package, a missing file is never same-package, an
// external _test package matches its target, and an unrelated package does not.
func TestGoFilePackageMatchesTarget_EdgeCases(t *testing.T) {
	if goFilePackageMatchesTarget("/nonexistent/x_test.go", "") {
		t.Errorf("empty target package is never same-package")
	}
	if goFilePackageMatchesTarget("/nonexistent/missing_test.go", "gate") {
		t.Errorf("a missing file is never same-package (open fails → false)")
	}

	dir := t.TempDir()
	// External test package `gate_test` for target `gate` → same-package.
	ext := filepath.Join(dir, "ext_test.go")
	if err := os.WriteFile(ext, []byte("package gate_test\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatalf("writing ext fixture: %v", err)
	}
	if !goFilePackageMatchesTarget(ext, "gate") {
		t.Errorf("package gate_test must match target gate (external test variant)")
	}

	// Same-name internal package → same-package.
	internal := filepath.Join(dir, "internal_test.go")
	if err := os.WriteFile(internal, []byte("package gate\n\nimport \"testing\"\n\nfunc TestY(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatalf("writing internal fixture: %v", err)
	}
	if !goFilePackageMatchesTarget(internal, "gate") {
		t.Errorf("package gate must match target gate (same package)")
	}

	// Unrelated package → not same-package.
	other := filepath.Join(dir, "other_test.go")
	if err := os.WriteFile(other, []byte("package other_test\n\nimport \"testing\"\n\nfunc TestZ(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatalf("writing other fixture: %v", err)
	}
	if goFilePackageMatchesTarget(other, "gate") {
		t.Errorf("package other_test must NOT match target gate")
	}
}
