package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// TestPackRelock_Command (ISSUE-032 Defect F / CLM-010): `pack relock <path>` refreshes
// a local pack's lock entry and reports the fresh content hash.
func TestPackRelock_Command(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "backstop.yml", "packs:\n  acme/relock-cmd: local\n")
	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "relock-cmd")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, packDir, "pack.yml",
		"name: acme/relock-cmd\nversion: 0.1.0\nlanguage: go\narchetype: enforcement")
	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{
		"acme/relock-cmd": {Name: "acme/relock-cmd", ContentHash: "sha256:STALE", SourceType: "local"},
	}}
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	defer chdirForTest(t, dir)()
	root := NewRootCommand()
	out, err := executeCommand(root, "pack", "relock", packDir)
	if err != nil {
		t.Fatalf("pack relock errored: %v (out=%s)", err, out)
	}
	if !strings.Contains(out, "acme/relock-cmd") {
		t.Errorf("relock output should name the pack, got: %s", out)
	}

	updated, err := distribution.ReadLockfile(filepath.Join(dir, "backstop.lock"))
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	if updated.Packs["acme/relock-cmd"].ContentHash == "sha256:STALE" {
		t.Error("relock should have overwritten the stale lock hash")
	}
}

// TestPackRelock_CommandError proves `pack relock` surfaces an error (non-zero) when
// the pack is not relockable — here a git-source pack (CLM-010).
func TestPackRelock_CommandError(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "backstop.yml", "packs:\n  acme/gitpack: \"1.0.0\"\n")
	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "gitpack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, packDir, "pack.yml", "name: acme/gitpack\nversion: 1.0.0")
	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{
		"acme/gitpack": {Name: "acme/gitpack", ContentHash: "sha256:x", SourceType: "git"},
	}}
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	defer chdirForTest(t, dir)()
	root := NewRootCommand()
	_, err := executeCommand(root, "pack", "relock", packDir)
	if err == nil {
		t.Fatal("expected an error relocking a non-local (git) pack")
	}
}
