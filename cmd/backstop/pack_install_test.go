package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func TestPackInstallCommand_MissingLockfile(t *testing.T) {
	root := NewRootCommand()

	// Run in temp dir with no backstop.lock.
	root.SetArgs([]string{"pack", "install"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error running pack install without lockfile")
	}
}

// TestPackInstallCommand_PrintsStaleLockWarning proves the CLI SURFACES reconciliation
// warnings (CLM-007): a stale lock entry diverging from the manifest is printed as a
// warning line naming the stale entry, before the installed summary.
func TestPackInstallCommand_PrintsStaleLockWarning(t *testing.T) {
	projectDir := t.TempDir()

	declared := "backstop/go-standards"
	stale := "slotly/go-standards"

	// A real local pack source inside the project.
	srcRel := "gostd-src"
	writeFileForTest(t, projectDir, filepath.Join(srcRel, "pack.yml"),
		"name: "+declared+"\nversion: \"1.0.0\"")
	writeFileForTest(t, projectDir, filepath.Join(srcRel, "rules", "r.yml"), "rules: []")
	hash, hashErr := distribution.ComputeContentHash(filepath.Join(projectDir, srcRel))
	if hashErr != nil {
		t.Fatal(hashErr)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			declared: {
				Name:        declared,
				ContentHash: hash,
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   srcRel,
			},
			stale: {
				Name:        stale,
				ContentHash: "sha256:stale",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   "renamed-away",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, projectDir, "backstop.yml", "packs:\n  "+declared+": local")

	restore := chdirForTest(t, projectDir)
	defer restore()

	root := NewRootCommand()
	out, err := executeCommand(root, "pack", "install")
	if err != nil {
		t.Fatalf("pack install should succeed: %v (out: %s)", err, out)
	}

	if !strings.Contains(out, stale) {
		t.Errorf("expected output to name the stale lock entry %q, got:\n%s", stale, out)
	}
	if !strings.Contains(strings.ToLower(out), "warning") {
		t.Errorf("expected a loud warning marker in output, got:\n%s", out)
	}
}

// TestPackInstallCommand_UnresolvableLocalPackExitsNonZero proves the CLI exits NON-ZERO
// on an unresolvable local pack instead of printing a bare "Installed N packs" green
// (CLM-007) — the invisible-failure fix.
func TestPackInstallCommand_UnresolvableLocalPackExitsNonZero(t *testing.T) {
	projectDir := t.TempDir()
	packName := "internal/local-rules"

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
				ContentHash: "sha256:whatever",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				// Empty LocalPath — unresolvable.
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, projectDir, "backstop.yml", "packs:\n  "+packName+": local")

	restore := chdirForTest(t, projectDir)
	defer restore()

	root := NewRootCommand()
	out, err := executeCommand(root, "pack", "install")
	if err == nil {
		t.Fatal("expected non-zero exit for unresolvable local pack")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected *ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitViolations {
		t.Errorf("exit code = %d, want %d", exitErr.Code, ExitViolations)
	}
	if strings.Contains(out, "Installed ") {
		t.Errorf("should NOT print a bare installed summary on failure, got:\n%s", out)
	}

	// Nothing materialized.
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", "internal", "local-rules")); !os.IsNotExist(statErr) {
		t.Error("nothing should be materialized on failure")
	}
}
