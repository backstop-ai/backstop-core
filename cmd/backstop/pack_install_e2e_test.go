package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// TestPackInstallE2E_LocalPackMaterializesAndResolves is the launch-bar acceptance
// (CLM-008): declare a local pack via the REAL `pack add`, simulate a fresh checkout
// (the gitignored .backstop/packs is gone), run the REAL `pack install`, and assert the
// pack directory + representative rule files ACTUALLY exist on disk and the pack-consume
// resolver (List) reads the installed pack from disk. Sibling `../go-standards` layout —
// the exact layout the two launching projects use.
func TestPackInstallE2E_LocalPackMaterializesAndResolves(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFileForTest(t, projectDir, "backstop.yml", "packs: {}")
	writeLocalPackSource(t, parent, "go-standards", "backstop/go-standards")

	restore := chdirForTest(t, projectDir)
	defer restore()

	// Real `pack add ../go-standards` (sibling local pack).
	addOut, addErr := executeCommand(NewRootCommand(), "pack", "add", "../go-standards")
	if addErr != nil {
		t.Fatalf("pack add: %v (out: %s)", addErr, addOut)
	}

	// Simulate a fresh checkout: .backstop/packs is gitignored, so the materialized copy
	// is gone. install must re-materialize from the lock's recorded local_path.
	if err := os.RemoveAll(filepath.Join(projectDir, ".backstop", "packs")); err != nil {
		t.Fatal(err)
	}

	instOut, instErr := executeCommand(NewRootCommand(), "pack", "install")
	if instErr != nil {
		t.Fatalf("pack install: %v (out: %s)", instErr, instOut)
	}

	// The pack dir + files ACTUALLY exist on disk.
	dest := filepath.Join(projectDir, ".backstop", "packs", "backstop", "go-standards")
	if _, err := os.Stat(filepath.Join(dest, "pack.yml")); err != nil {
		t.Errorf("pack.yml not materialized on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "rules", "r1.yml")); err != nil {
		t.Errorf("rule file not materialized on disk: %v", err)
	}

	// The pack-consume resolver reads the installed pack FROM DISK (archetype + rule count
	// only come from reading the on-disk pack.yml).
	listRes, listErr := distribution.List(distribution.ListOptions{ProjectDir: "."})
	if listErr != nil {
		t.Fatalf("List: %v", listErr)
	}
	found := false
	for _, p := range listRes.Packs {
		if p.Name == "backstop/go-standards" {
			found = true
			if p.Archetype != "enforcement" {
				t.Errorf("archetype not resolved from disk: %q", p.Archetype)
			}
			if p.RuleCount < 1 {
				t.Errorf("rule count not resolved from disk: %d", p.RuleCount)
			}
		}
	}
	if !found {
		t.Error("pack-consume resolver (List) did not find the installed pack")
	}
}

// TestPackInstallE2E_StaleLockWarnsInstallsDeclared drives the stale-lock divergence case
// through the REAL CLI (CLM-008): the DECLARED pack installs, the stale lock-only entry
// warns and is NOT installed.
func TestPackInstallE2E_StaleLockWarnsInstallsDeclared(t *testing.T) {
	projectDir := t.TempDir()

	declared := "backstop/go-standards"
	stale := "slotly/go-standards"

	srcRel := "gostd-src"
	writeFileForTest(t, projectDir, filepath.Join(srcRel, "pack.yml"),
		"name: "+declared+"\nversion: \"1.0.0\"")
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

	out, err := executeCommand(NewRootCommand(), "pack", "install")
	if err != nil {
		t.Fatalf("pack install should succeed: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, stale) || !strings.Contains(strings.ToLower(out), "warning") {
		t.Errorf("expected a warning naming the stale entry %q, got:\n%s", stale, out)
	}

	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", "backstop", "go-standards")); statErr != nil {
		t.Errorf("declared pack not materialized: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", "slotly", "go-standards")); !os.IsNotExist(statErr) {
		t.Error("stale lock-only pack should not be materialized")
	}
}

// TestPackInstallE2E_UnresolvableSourceFailsLoud drives the unresolvable-source case
// through the REAL CLI (CLM-008): a local pack added then whose source vanishes (and the
// gitignored materialized copy is gone) fails loud on install — non-zero, nothing on disk.
func TestPackInstallE2E_UnresolvableSourceFailsLoud(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFileForTest(t, projectDir, "backstop.yml", "packs: {}")
	sourceDir := writeLocalPackSource(t, parent, "go-standards", "backstop/go-standards")

	restore := chdirForTest(t, projectDir)
	defer restore()

	if addOut, addErr := executeCommand(NewRootCommand(), "pack", "add", "../go-standards"); addErr != nil {
		t.Fatalf("pack add: %v (out: %s)", addErr, addOut)
	}

	// Fresh checkout where the sibling source has also moved/vanished.
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(projectDir, ".backstop", "packs")); err != nil {
		t.Fatal(err)
	}

	out, err := executeCommand(NewRootCommand(), "pack", "install")
	if err == nil {
		t.Fatal("expected non-zero exit when the local source is gone")
	}
	if strings.Contains(out, "Installed ") {
		t.Errorf("should not print a bare installed summary on failure, got:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", "backstop", "go-standards")); !os.IsNotExist(statErr) {
		t.Error("nothing should be materialized when the source is unresolvable")
	}
}
