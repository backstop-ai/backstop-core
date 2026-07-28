package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

func containsPack(list []string, name string) bool {
	for _, p := range list {
		if p == name {
			return true
		}
	}
	return false
}

func warningsMention(warnings []string, needle string) bool {
	for _, w := range warnings {
		if strings.Contains(w, needle) {
			return true
		}
	}
	return false
}

// TestInstall_ManifestDrivenIgnoresStaleLockEntry proves Defect B is fixed: install is
// driven by the DECLARED backstop.yml manifest, not raw lf.Packs. A stale lock entry
// (in lock, absent from the manifest — e.g. a renamed slotly/go-standards) is called out
// LOUDLY in Warnings and NOT installed; the DECLARED backstop/go-standards is installed
// instead (CLM-004, CLM-005).
func TestInstall_ManifestDrivenIgnoresStaleLockEntry(t *testing.T) {
	projectDir := t.TempDir()

	declared := "backstop/go-standards"
	stale := "slotly/go-standards"

	sourceDir := t.TempDir()
	hash := writeLocalPackSource(t, sourceDir, declared)
	rel, err := filepath.Rel(projectDir, sourceDir)
	if err != nil {
		t.Fatal(err)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			declared: {
				Name:        declared,
				ContentHash: hash,
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   rel,
			},
			stale: {
				Name:        stale,
				ContentHash: "sha256:stalehash",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   "../renamed-away",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	// Manifest declares ONLY the current pack, not the stale one.
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+declared+": local\n")

	install := newTestInstallCommand(t, defaultTestPackCloner())

	result, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !containsPack(result.InstalledPacks, declared) {
		t.Errorf("expected declared pack %q to be installed, got %v", declared, result.InstalledPacks)
	}
	if containsPack(result.InstalledPacks, stale) {
		t.Errorf("stale lock-only pack %q should NOT be installed, got %v", stale, result.InstalledPacks)
	}
	if !warningsMention(result.Warnings, stale) {
		t.Errorf("expected a stale-lock warning naming %q, got warnings: %v", stale, result.Warnings)
	}

	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(declared))); statErr != nil {
		t.Errorf("declared pack not materialized: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(stale))); !os.IsNotExist(statErr) {
		t.Error("stale lock-only pack should not be materialized")
	}
}

// TestInstall_ManifestPackMissingFromLockSurfaced proves a manifest pack that is
// missing/diverged from the lock is surfaced (warning), not silently skipped (CLM-004).
func TestInstall_ManifestPackMissingFromLockSurfaced(t *testing.T) {
	projectDir := t.TempDir()

	// Lock holds a different pack than the manifest declares.
	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/other": {
				Name:        "acme/other",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:other",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  backstop/undeclared-in-lock: \"1.0.0\"\n")

	install := newTestInstallCommand(t, defaultTestPackCloner())

	result, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if containsPack(result.InstalledPacks, "backstop/undeclared-in-lock") {
		t.Error("a manifest pack absent from the lock cannot be installed")
	}
	if !warningsMention(result.Warnings, "backstop/undeclared-in-lock") {
		t.Errorf("expected a warning naming the manifest pack missing from the lock, got: %v", result.Warnings)
	}
}

// TestInstall_MalformedManifestErrors proves a malformed backstop.yml surfaces a parse
// error rather than being silently treated as absent.
func TestInstall_MalformedManifestErrors(t *testing.T) {
	projectDir := t.TempDir()

	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{}}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs: [not: valid: {{{")

	install := newTestInstallCommand(t, defaultTestPackCloner())

	_, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error for malformed backstop.yml")
	}
	if !strings.Contains(err.Error(), "backstop.yml") {
		t.Errorf("error should mention backstop.yml, got: %v", err)
	}
}

// TestInstall_ManifestReadErrorSurfaces proves a non-not-exist read error on backstop.yml
// (here: the manifest path is a directory) is surfaced, not swallowed as absent.
func TestInstall_ManifestReadErrorSurfaces(t *testing.T) {
	projectDir := t.TempDir()

	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{}}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	// backstop.yml as a directory makes os.ReadFile fail with a non-not-exist error.
	if err := os.MkdirAll(filepath.Join(projectDir, "backstop.yml"), 0o755); err != nil {
		t.Fatal(err)
	}

	install := newTestInstallCommand(t, defaultTestPackCloner())

	_, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error when backstop.yml cannot be read")
	}
}

// TestInstall_AbsentManifestInstallsNothing proves that with a backstop.lock but NO
// backstop.yml, Install installs NOTHING (empty InstalledPacks) and returns a clear
// message — there is NO silent lf.Packs fallback that would bypass Defect B (CLM-004).
func TestInstall_AbsentManifestInstallsNothing(t *testing.T) {
	projectDir := t.TempDir()

	sourceDir := t.TempDir()
	hash := writeLocalPackSource(t, sourceDir, "internal/local-rules")
	rel := mustRel(t, projectDir, sourceDir)

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local-rules": {
				Name:        "internal/local-rules",
				ContentHash: hash,
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   rel,
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	// Intentionally NO backstop.yml.

	install := newTestInstallCommand(t, defaultTestPackCloner())

	result, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if len(result.InstalledPacks) != 0 {
		t.Errorf("expected nothing installed with no manifest, got %v", result.InstalledPacks)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected a clear warning message when no manifest is declared")
	}

	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs")); !os.IsNotExist(statErr) {
		t.Error("nothing should be materialized when no manifest is declared")
	}
}
