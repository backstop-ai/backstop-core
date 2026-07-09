package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// TestPackAddDeclaredE2E_EmptyPacksMaterializes is the launch-bar acceptance (CLM-006)
// reproducing the ISSUE-026 scenario against the REAL CLI: backstop.yml DECLARES a local
// pack but `.backstop/packs/` is EMPTY, so under the old false equivalence `pack add`
// short-circuited ("already installed") and installed nothing. Under the fix the declared-
// but-absent pack must actually MATERIALIZE on disk, update the lock, and print a real,
// non-empty success line with NO bare `@`. A second `pack add` on the now genuinely-current
// pack must be an HONEST no-op ("already installed" on stdout, exit 0) — never silent.
func TestPackAddDeclaredE2E_EmptyPacksMaterializes(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	sourceDir := filepath.Join(parent, "go-standards")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	packName := "backstop/go-standards"
	// backstop.yml DECLARES the pack, but nothing is materialized on disk.
	writeFileForTest(t, projectDir, "backstop.yml", "packs:\n  "+packName+": local")
	writeFileForTest(t, sourceDir, "pack.yml",
		"name: "+packName+"\narchetype: enforcement\ncontent:\n  ruleset:\n    rules:\n      - id: R1")
	writeFileForTest(t, sourceDir, filepath.Join("rules", "r1.yml"), "rules:\n  - id: R1")

	restore := chdirForTest(t, projectDir)
	defer restore()

	out, err := executeCommand(NewRootCommand(), "pack", "add", "../go-standards")
	if err != nil {
		t.Fatalf("pack add (declared-but-absent) should install, got error: %v (out: %s)", err, out)
	}

	// Non-empty success line with NO bare `@`.
	if !strings.Contains(out, "Added") || !strings.Contains(out, packName) {
		t.Errorf("expected a non-empty Added line naming %q, got: %q", packName, out)
	}
	if strings.Contains(out, packName+"@") {
		t.Errorf("versionless local pack must not render a bare `@`, got: %q", out)
	}

	// The pack MATERIALIZED on disk (not a no-op) with a representative rule file.
	dest := filepath.Join(projectDir, ".backstop", "packs", packName)
	if _, statErr := os.Stat(filepath.Join(dest, "pack.yml")); statErr != nil {
		t.Errorf("pack.yml not materialized on disk: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "rules", "r1.yml")); statErr != nil {
		t.Errorf("rule file not materialized on disk: %v", statErr)
	}

	// The lock now holds a consistent entry.
	lf, lockErr := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if lockErr != nil {
		t.Fatalf("reading lock: %v", lockErr)
	}
	if _, ok := lf.Packs[packName]; !ok {
		t.Errorf("lock entry not written for %q", packName)
	}

	// Second add: genuinely-current -> honest no-op, exit 0, never silent.
	out2, err2 := executeCommand(NewRootCommand(), "pack", "add", "../go-standards")
	if err2 != nil {
		t.Fatalf("second pack add (already current) should exit 0, got error: %v (out: %s)", err2, out2)
	}
	if !strings.Contains(strings.ToLower(out2), "already installed") {
		t.Errorf("expected an honest already-installed message on stdout, got: %q", out2)
	}
}

// TestPackAddDeclaredE2E_DivergedLockReinstalls covers the issue's rename/divergence
// variant against the REAL CLI (CLM-006): backstop.yml declares the pack, `.backstop/packs/`
// is empty, and the lock holds only a STALE entry under the OLD name (no entry for the
// declared name). The declared pack must still materialize and gain a consistent lock entry.
func TestPackAddDeclaredE2E_DivergedLockReinstalls(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	sourceDir := filepath.Join(parent, "go-standards")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	declared := "backstop/go-standards"
	stale := "slotly/go-standards"

	writeFileForTest(t, projectDir, "backstop.yml", "packs:\n  "+declared+": local")
	writeFileForTest(t, sourceDir, "pack.yml",
		"name: "+declared+"\narchetype: enforcement\ncontent:\n  ruleset:\n    rules:\n      - id: R1")
	writeFileForTest(t, sourceDir, filepath.Join("rules", "r1.yml"), "rules:\n  - id: R1")

	// Diverged lock: only a stale entry under the OLD name, no entry for the declared pack.
	staleLock := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			stale: {
				Name:        stale,
				ContentHash: "sha256:stale",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   "renamed-away",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), staleLock); err != nil {
		t.Fatal(err)
	}

	restore := chdirForTest(t, projectDir)
	defer restore()

	out, err := executeCommand(NewRootCommand(), "pack", "add", "../go-standards")
	if err != nil {
		t.Fatalf("pack add (diverged lock) should install, got error: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, "Added") {
		t.Errorf("expected a real Added line, got: %q", out)
	}

	// The declared pack materialized on disk.
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", declared, "pack.yml")); statErr != nil {
		t.Errorf("declared pack not materialized on disk: %v", statErr)
	}

	// The lock now holds a consistent entry for the DECLARED pack.
	lf, lockErr := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if lockErr != nil {
		t.Fatalf("reading lock: %v", lockErr)
	}
	if _, ok := lf.Packs[declared]; !ok {
		t.Errorf("lock entry not written for declared pack %q", declared)
	}
}
