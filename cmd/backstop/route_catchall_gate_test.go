package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
)

// SPEC-039 REQ-007 gate-result behavior-preserving test. RED is not expected
// here (the assertion holds before AND after the deletion — the routed findings
// pass for non-Go files was always a Skipped no-op); it GUARDS that deleting the
// catch-all does not change the violation set or exit code.

// TestCodeCheck_NonGoFile_GateResultUnchangedAfterCatchAllRemoval pins CLM-005
// (sharp edge): a code-check run with non-Go files in scope yields the SAME
// violation set and exit/error outcome before and after the catch-all deletion.
//
// CRITICAL: this asserts on VIOLATIONS + ERROR (exit) ONLY — it does NOT assert
// an exact PassResults list, because the Skipped("no executor configured")
// findings entry for non-Go files intentionally disappears post-deletion (the
// observable, intended PassResults-inventory delta the spec's Sharp Edge 1
// warns about).
func TestCodeCheck_NonGoFile_GateResultUnchangedAfterCatchAllRemoval(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: catchall\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	// Non-Go files only. Their findings pass had no executor (Skipped no-op);
	// lint/build/test are not applicable to non-Go files. So the violation set is
	// empty both before and after the catch-all deletion.
	nonGo := []string{"README.md", "config.yml", "notes.txt"}
	for _, f := range nonGo {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	restore := chdirTemp(t, dir)
	defer restore()

	// A recording runner so no real tool executes; the non-Go scope routes no
	// pass to an executor anyway.
	runner := &recordingRunner{}
	checker := &realCodeChecker{projectRoot: dir, runnerForTest: runner}

	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: nonGo}
	violations, err := checker.CheckScoped(context.Background(), scope)
	if err != nil {
		t.Fatalf("CheckScoped over a non-Go scope returned error %v; want a clean (nil-error) run", err)
	}
	// VIOLATION SET unchanged: empty. (No executor ran for any routed pass.)
	if len(violations) != 0 {
		t.Errorf("non-Go scope yielded %d violations, want 0 (the findings catch-all was always a no-op): %+v", len(violations), violations)
	}
}
