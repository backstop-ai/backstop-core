package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

func mustRel(t *testing.T, base, target string) string {
	t.Helper()
	rel, err := filepath.Rel(base, target)
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}
	return rel
}

func mustHash(t *testing.T, dir string) string {
	t.Helper()
	hash, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash: %v", err)
	}
	return hash
}

// writeLocalPackSource writes a representative local pack (pack.yml + a rule file)
// into sourceDir and returns its content hash.
func writeLocalPackSource(t *testing.T, sourceDir, name string) string {
	t.Helper()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: "+name+"\nversion: \"1.0.0\"\n")
	writeFile(t, filepath.Join(sourceDir, "rules", "r1.yml"), "rules:\n  - id: R1\n")
	hash, err := distribution.ComputeContentHash(sourceDir)
	if err != nil {
		t.Fatalf("ComputeContentHash: %v", err)
	}
	return hash
}

// TestInstall_MaterializesLocalPackToDisk proves Defect A is fixed: the real Install
// (no LocalPackDir override) resolves a local pack's source from the lock's local_path,
// COPIES it into .backstop/packs/<name>/, and the pack dir + representative files
// genuinely exist on disk afterward (CLM-002).
func TestInstall_MaterializesLocalPackToDisk(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()
	packName := "internal/local-rules"
	hash := writeLocalPackSource(t, sourceDir, packName)

	rel, err := filepath.Rel(projectDir, sourceDir)
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
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
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+packName+": local\n")

	// No LocalPackDir override — the real/CLI resolution path from local_path.
	opts := distribution.InstallOptions{ProjectDir: projectDir}

	install := newTestInstallCommand(t, defaultTestPackCloner())

	result, err := install.Run(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	destDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(packName))
	if _, statErr := os.Stat(filepath.Join(destDir, "pack.yml")); statErr != nil {
		t.Errorf("pack.yml not materialized on disk: %v", statErr)
	}
	ruleData, ruleErr := os.ReadFile(filepath.Join(destDir, "rules", "r1.yml"))
	if ruleErr != nil {
		t.Fatalf("rule file not materialized on disk: %v", ruleErr)
	}
	if !strings.Contains(string(ruleData), "id: R1") {
		t.Errorf("materialized rule file content = %q, want it to contain 'id: R1'", string(ruleData))
	}

	found := false
	for _, p := range result.InstalledPacks {
		if p == packName {
			found = true
		}
	}
	if !found {
		t.Errorf("InstalledPacks = %v, want it to contain %q", result.InstalledPacks, packName)
	}
}

// TestInstall_LocalPackEmptyLocalPathFailsLoud proves a local pack with an EMPTY
// local_path (and no override) FAILS LOUD naming the pack and materializes nothing —
// never a silent success (CLM-003).
func TestInstall_LocalPackEmptyLocalPathFailsLoud(t *testing.T) {
	projectDir := t.TempDir()
	packName := "internal/no-path"

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
				ContentHash: "sha256:whatever",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				// LocalPath intentionally empty.
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+packName+": local\n")

	opts := distribution.InstallOptions{ProjectDir: projectDir}

	install := newTestInstallCommand(t, defaultTestPackCloner())

	_, err := install.Run(opts)
	if err == nil {
		t.Fatal("expected error for local pack with empty local_path")
	}
	if !strings.Contains(err.Error(), packName) {
		t.Errorf("error should name the pack %q, got: %v", packName, err)
	}

	destDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(packName))
	if _, statErr := os.Stat(destDir); !os.IsNotExist(statErr) {
		t.Error("nothing should be materialized when local_path is empty")
	}
}

// TestInstall_LocalPackMissingSourceDirFailsLoud proves a local pack whose recorded
// local_path resolves to a directory MISSING on disk FAILS LOUD naming the pack and
// materializes nothing (CLM-003) — the fresh-checkout hard error.
func TestInstall_LocalPackMissingSourceDirFailsLoud(t *testing.T) {
	projectDir := t.TempDir()
	packName := "internal/gone"

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
				ContentHash: "sha256:whatever",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   "../does-not-exist-anywhere",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+packName+": local\n")

	opts := distribution.InstallOptions{ProjectDir: projectDir}

	install := newTestInstallCommand(t, defaultTestPackCloner())

	_, err := install.Run(opts)
	if err == nil {
		t.Fatal("expected error for local pack with missing source dir")
	}
	if !strings.Contains(err.Error(), packName) {
		t.Errorf("error should name the pack %q, got: %v", packName, err)
	}

	destDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(packName))
	if _, statErr := os.Stat(destDir); !os.IsNotExist(statErr) {
		t.Error("nothing should be materialized when source dir is missing")
	}
}

// TestInstall_LocalPackSourceIsFileFailsLoud proves a local pack whose recorded
// local_path resolves to a regular FILE (not a directory) fails loud naming the pack.
func TestInstall_LocalPackSourceIsFileFailsLoud(t *testing.T) {
	projectDir := t.TempDir()
	packName := "internal/is-a-file"

	// Create a plain file the local_path will point at.
	writeFile(t, filepath.Join(projectDir, "notadir"), "hello")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
				ContentHash: "sha256:whatever",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
				LocalPath:   "notadir",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+packName+": local\n")

	install := newTestInstallCommand(t, defaultTestPackCloner())

	_, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error when local source is a file")
	}
	if !strings.Contains(err.Error(), packName) {
		t.Errorf("error should name the pack %q, got: %v", packName, err)
	}
}

// TestInstall_LocalPackDestMkdirFails proves the local materialize path rolls back and
// errors when the destination parent cannot be created (here: .backstop/packs exists as
// a regular file).
func TestInstall_LocalPackDestMkdirFails(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()
	packName := "internal/local-rules"
	hash := writeLocalPackSource(t, sourceDir, packName)
	rel := mustRel(t, projectDir, sourceDir)

	// Make .backstop/packs a FILE so MkdirAll of a subdirectory fails.
	if err := os.MkdirAll(filepath.Join(projectDir, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, ".backstop", "packs"), "not a dir")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
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
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+packName+": local\n")

	install := newTestInstallCommand(t, defaultTestPackCloner())

	_, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error when destination parent cannot be created")
	}
}

// TestInstall_GitPackDestMkdirFails proves the git materialize path errors when the
// destination parent cannot be created (.backstop/packs exists as a regular file).
func TestInstall_GitPackDestMkdirFails(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()
	packName := "acme/git-pack"
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: "+packName+"\nversion: \"1.0.0\"\n")
	hash := mustHash(t, sourceDir)

	if err := os.MkdirAll(filepath.Join(projectDir, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, ".backstop", "packs"), "not a dir")

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: hash,
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+packName+": \"1.0.0\"\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
	}
	install := newTestInstallCommand(t, &mockGitCloner{cloneDir: sourceDir})

	_, err := install.Run(opts)
	if err == nil {
		t.Fatal("expected error when destination parent cannot be created")
	}
}

// TestInstall_GitPathStillMaterializes is the regression guard proving the local-pack
// changes do NOT regress the git-source install branch: a git pack still materializes
// its cloned files into .backstop/packs/<name>/ (CLM-006).
func TestInstall_GitPathStillMaterializes(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()
	packName := "acme/git-pack"
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: "+packName+"\nversion: \"1.0.0\"\n")
	writeFile(t, filepath.Join(sourceDir, "rules", "g.yml"), "rules:\n  - id: G1\n")
	hash := mustHash(t, sourceDir)

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:        packName,
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: hash,
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+packName+": \"1.0.0\"\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
	}

	install := newTestInstallCommand(t, &mockGitCloner{cloneDir: sourceDir})

	_, err := install.Run(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	destDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(packName))
	if _, statErr := os.Stat(filepath.Join(destDir, "pack.yml")); statErr != nil {
		t.Errorf("git pack.yml not materialized: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(destDir, "rules", "g.yml")); statErr != nil {
		t.Errorf("git rule file not materialized: %v", statErr)
	}
}
