package check

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCheck_NeverStartedMatchesBothRealShapes pins CLM-001: NeverStarted is true for
// BOTH real never-started shapes, and both fixtures drive a REAL exec.Command against
// a REAL failed exec. A synthetic &fs.PathError{Op: "fork/exec"} would make this test
// greenable without the classification ever facing a real one — the vacuous shape
// ISSUE-140 exists to remove.
func TestCheck_NeverStartedMatchesBothRealShapes(t *testing.T) {
	// A PATH-FUL command never consults LookPath, so its failure can never be an
	// *exec.Error. Write a real non-executable file and guard the fixture's own
	// premise: if it ever became executable this case would silently stop testing
	// the fork/exec shape while still passing.
	pathful := filepath.Join(t.TempDir(), "unstartable.sh")
	if err := os.WriteFile(pathful, []byte("#!/bin/sh\necho should-never-run\n"), 0o644); err != nil {
		t.Fatalf("write non-executable script: %v", err)
	}
	info, err := os.Stat(pathful)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode().Perm()&0o111 != 0 {
		t.Fatalf("fixture invariant: %s must NOT be executable, got mode %v", pathful, info.Mode().Perm())
	}

	cases := []struct {
		name string
		// command is a BARE name (routed through LookPath) or a PATH-FUL path.
		command string
		// assertShape pins the concrete Go error type this run must produce, so a
		// future Go change that alters it is caught here rather than as a confusing
		// behavioral failure somewhere downstream.
		assertShape func(t *testing.T, runErr error)
	}{
		{
			// A fixed nonsense bare name: exec.Command routes it through LookPath,
			// which misses. PATH is never mutated and no real tool is assumed absent.
			name:    "bare absent name yields *exec.Error",
			command: "backstop-absent-engine-140",
			assertShape: func(t *testing.T, runErr error) {
				t.Helper()
				var execErr *exec.Error
				if !errors.As(runErr, &execErr) {
					t.Fatalf("expected *exec.Error from a bare absent name, got %T: %v", runErr, runErr)
				}
			},
		},
		{
			name:    "path-ful non-executable file yields *fs.PathError with Op fork/exec",
			command: pathful,
			assertShape: func(t *testing.T, runErr error) {
				t.Helper()
				var pathErr *fs.PathError
				if !errors.As(runErr, &pathErr) {
					t.Fatalf("expected *fs.PathError from a path-ful unstartable command, got %T: %v", runErr, runErr)
				}
				if pathErr.Op != "fork/exec" {
					t.Fatalf("expected Op %q, got %q", "fork/exec", pathErr.Op)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runErr := exec.Command(tc.command).Run()
			if runErr == nil {
				t.Fatalf("fixture invariant: %q unexpectedly ran successfully", tc.command)
			}
			tc.assertShape(t, runErr)
			if !NeverStarted(runErr) {
				t.Fatalf("NeverStarted(%T: %v) = false, want true", runErr, runErr)
			}
		})
	}
}

// TestCheck_NeverStartedRejectsStartedProcess pins CLM-002 — the assertion that keeps
// CLM-001 honest. An implementation reduced to `runErr != nil` passes the shapes test
// above and fails here, which is exactly what this test is for: a rule-fed findings
// engine exits non-zero precisely WHEN it reports findings, so treating every run
// error as never-started would fail every real finding.
func TestCheck_NeverStartedRejectsStartedProcess(t *testing.T) {
	if NeverStarted(nil) {
		t.Fatal("NeverStarted(nil) = true, want false")
	}

	// Resolve a real shell rather than skipping: a skip here would silently remove
	// the only guard against the `runErr != nil` mistake.
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("resolving sh for the started-process fixture: %v", err)
	}
	runErr := exec.Command(sh, "-c", "exit 3").Run()
	if runErr == nil {
		t.Fatal("fixture invariant: `sh -c \"exit 3\"` must fail")
	}
	// Pin the fixture's own premise first: this process DID start, and reported the
	// only shape that says so.
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("expected *exec.ExitError from a started process, got %T: %v", runErr, runErr)
	}
	if NeverStarted(runErr) {
		t.Fatalf("NeverStarted(*exec.ExitError: %v) = true, want false — the process STARTED", runErr)
	}
}
