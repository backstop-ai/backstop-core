package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStandardScaffolder_Untouched pins CLM-018: the .standard.md scaffolder
// (pkg/pack/scaffold.go) is out of scope — it still exists and this change does
// not reference or remove it (scope fence for ISSUE-030). The former
// TestCodeCheck_MissingToolchain assertion drove the deleted `backstop code
// check` command (ISSUE-018) and was removed with it.
func TestStandardScaffolder_Untouched(t *testing.T) {
	// The scaffolder file lives at <repo>/pkg/pack/scaffold.go. From the
	// cmd/backstop package dir (CWD during tests), that is ../../pkg/pack.
	p := filepath.Join("..", "..", "pkg", "pack", "scaffold.go")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("pkg/pack/scaffold.go must remain (ISSUE-030 scope fence); stat error: %v", err)
	}
}
