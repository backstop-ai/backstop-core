package distribution_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// setupRelockProject installs a pack (backstop.yml + installed pack.yml + a lock entry
// carrying a DELIBERATELY stale hash) and returns the project dir and installed pack
// dir. sourceType selects git/local.
func setupRelockProject(t *testing.T, sourceType string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "backstop.yml"), "packs:\n  acme/relock-pack: local\n")
	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "relock-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"),
		"name: acme/relock-pack\nversion: 0.1.0\nlanguage: go\narchetype: enforcement\n")
	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{
		"acme/relock-pack": {
			Name:        "acme/relock-pack",
			ContentHash: "sha256:STALE",
			SourceType:  sourceType,
			InstallDate: "2026-01-01T00:00:00Z",
		},
	}}
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	return dir, packDir
}

// TestRelock_LocalPackRefreshesHash (ISSUE-032 Defect F / CLM-010): Relock re-reads a
// local pack, recomputes its content hash over the installed dir, and overwrites the
// backstop.lock entry (preserving SourceType local) — no remove+add.
func TestRelock_LocalPackRefreshesHash(t *testing.T) {
	dir, packDir := setupRelockProject(t, "local")

	result, err := distribution.Relock(dir, packDir)
	if err != nil {
		t.Fatalf("Relock: %v", err)
	}
	if result.PackName != "acme/relock-pack" {
		t.Errorf("PackName = %q, want acme/relock-pack", result.PackName)
	}

	want, _ := distribution.ComputeContentHash(packDir)
	if result.ContentHash != want {
		t.Errorf("ContentHash = %q, want recomputed %q", result.ContentHash, want)
	}
	if result.ContentHash == "sha256:STALE" {
		t.Error("Relock must recompute the hash, not preserve the stale one")
	}

	// The lockfile entry must be overwritten with the fresh hash, SourceType intact.
	lf, err := distribution.ReadLockfile(filepath.Join(dir, "backstop.lock"))
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	entry := lf.Packs["acme/relock-pack"]
	if entry.ContentHash != want {
		t.Errorf("lock entry hash = %q, want %q", entry.ContentHash, want)
	}
	if entry.SourceType != "local" {
		t.Errorf("SourceType = %q, want local (preserved)", entry.SourceType)
	}
}

// TestRelock_NonLocalPackErrors proves Relock refuses a git-source pack (CLM-010).
func TestRelock_NonLocalPackErrors(t *testing.T) {
	dir, packDir := setupRelockProject(t, "git")
	_, err := distribution.Relock(dir, packDir)
	if err == nil {
		t.Fatal("expected error relocking a non-local pack")
	}
}

// TestRelock_UnknownPackErrors proves Relock errors when the pack is absent from the
// lockfile (CLM-010).
func TestRelock_UnknownPackErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "backstop.yml"), "packs: {}\n")
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"),
		&distribution.Lockfile{Packs: map[string]distribution.LockEntry{}}); err != nil {
		t.Fatal(err)
	}
	packDir := filepath.Join(dir, "orphan")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/orphan\nversion: 0.1.0\n")
	_, err := distribution.Relock(dir, packDir)
	if err == nil {
		t.Fatal("expected error relocking a pack absent from the lockfile")
	}
}

// TestRelock_MissingManifestErrors proves Relock errors when the path has no pack.yml.
func TestRelock_MissingManifestErrors(t *testing.T) {
	dir := t.TempDir()
	_, err := distribution.Relock(dir, filepath.Join(dir, "nope"))
	if err == nil {
		t.Fatal("expected error when the pack path has no pack.yml")
	}
}

// TestRelock_NamelessManifestErrors proves Relock errors when pack.yml has no name.
func TestRelock_NamelessManifestErrors(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "nameless")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "version: 0.1.0\nlanguage: go\n")
	if _, err := distribution.Relock(dir, packDir); err == nil {
		t.Fatal("expected error when pack.yml has no name")
	}
}

// TestRelock_MalformedManifestErrors proves readPackName fails loud on invalid yaml.
func TestRelock_MalformedManifestErrors(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "bad")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: [unterminated\n:::{{{")
	if _, err := distribution.Relock(dir, packDir); err == nil {
		t.Fatal("expected error parsing malformed pack.yml")
	}
}

// TestRelock_NoLockfileErrors proves Relock errors when backstop.lock is absent.
func TestRelock_NoLockfileErrors(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/x\nversion: 0.1.0\n")
	// No backstop.lock written.
	if _, err := distribution.Relock(dir, packDir); err == nil {
		t.Fatal("expected error when backstop.lock is absent")
	}
}

// TestRelock_InstalledDirMissingErrors proves Relock errors when the lock entry is a
// local pack but its installed dir is absent (nothing to hash).
func TestRelock_InstalledDirMissingErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "backstop.yml"), "packs:\n  acme/gone: local\n")
	// pack.yml source exists, lock entry exists (local), but no installed dir.
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(src, "pack.yml"), "name: acme/gone\nversion: 0.1.0\n")
	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{
		"acme/gone": {Name: "acme/gone", ContentHash: "sha256:x", SourceType: "local"},
	}}
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	if _, err := distribution.Relock(dir, src); err == nil {
		t.Fatal("expected error when the installed pack dir is missing")
	}
}
