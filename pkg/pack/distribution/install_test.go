package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func setupInstallProject(t *testing.T) (string, *distribution.Lockfile) {
	t.Helper()
	dir := t.TempDir()

	// Create a valid lockfile.
	ref := "v1.0.0"
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")
	hash, _ := distribution.ComputeContentHash(packDir)

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/valid-pack": {
				Name:        "acme/valid-pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: hash,
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}

	lockPath := filepath.Join(dir, "backstop.lock")
	if err := distribution.WriteLockfile(lockPath, lf); err != nil {
		t.Fatal(err)
	}

	return dir, lf
}

func TestPackInstall_RestoresFromLockfile(t *testing.T) {
	projectDir, lf := setupInstallProject(t)

	// Create a source directory matching the lockfile hash.
	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner: &mockGitCloner{
			cloneDir: sourceDir,
		},
	}
	_ = lf

	result, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if len(result.InstalledPacks) == 0 {
		t.Error("expected installed packs")
	}
}

func TestPackInstall_VerifiesContentHash(t *testing.T) {
	projectDir, _ := setupInstallProject(t)

	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Verify the installed pack hash matches.
	packDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if _, statErr := os.Stat(packDir); statErr != nil {
		t.Fatalf("pack not installed: %v", statErr)
	}
}

func TestPackInstall_SkipsValidation(t *testing.T) {
	projectDir, _ := setupInstallProject(t)

	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")

	// Install should NOT call validator at all.
	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install should succeed without validation: %v", err)
	}
}

func TestPackInstall_HashMismatchFailsHard(t *testing.T) {
	projectDir := t.TempDir()

	// Create lockfile with a hash that won't match.
	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "wrong-hash-that-wont-match",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/pack\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}

	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("error should mention hash mismatch, got: %v", err)
	}
}

func TestPackInstall_CacheReadsFromLocalDir(t *testing.T) {
	projectDir, _ := setupInstallProject(t)

	// Set up cache directory with the pack.
	cacheDir := t.TempDir()
	packCache := filepath.Join(cacheDir, "acme", "valid-pack")
	if err := os.MkdirAll(packCache, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packCache, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		CachePath:  cacheDir,
	}

	result, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install with cache: %v", err)
	}

	if len(result.InstalledPacks) == 0 {
		t.Error("expected installed packs from cache")
	}
}

func TestPackInstall_CacheStillVerifiesHash(t *testing.T) {
	projectDir := t.TempDir()

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "wrong-hash",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	cacheDir := t.TempDir()
	packCache := filepath.Join(cacheDir, "acme", "pack")
	if err := os.MkdirAll(packCache, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packCache, "pack.yml"), "name: acme/pack\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		CachePath:  cacheDir,
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected hash mismatch error even with cache")
	}
}

func TestPackInstall_SkipsToolConfigMerge(t *testing.T) {
	projectDir := t.TempDir()

	// Create source with tool_config.
	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\ntool_config:\n  - config_file: .golangci.yml\n    settings:\n      test: true\n")

	hash, _ := distribution.ComputeContentHash(sourceDir)
	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/valid-pack": {
				Name:        "acme/valid-pack",
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

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Verify no config file was merged into project.
	if _, statErr := os.Stat(filepath.Join(projectDir, ".golangci.yml")); !os.IsNotExist(statErr) {
		t.Error("install should NOT merge tool_config")
	}
}

func TestPackInstall_LocalPackHashVerification(t *testing.T) {
	projectDir := t.TempDir()

	localDir := t.TempDir()
	writeFile(t, filepath.Join(localDir, "pack.yml"), "name: internal/local\n")
	hash, _ := distribution.ComputeContentHash(localDir)

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local": {
				Name:        "internal/local",
				ContentHash: hash,
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.InstallOptions{
		ProjectDir:   projectDir,
		LocalPackDir: localDir,
	}

	_, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install with local pack: %v", err)
	}
}

func TestPackInstall_SkipsSDKDependencies(t *testing.T) {
	projectDir := t.TempDir()

	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"),
		"name: acme/valid-pack\nversion: \"1.0.0\"\nsdk_dependencies:\n  - go:1.21\n")

	hash, _ := distribution.ComputeContentHash(sourceDir)
	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/valid-pack": {
				Name:        "acme/valid-pack",
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

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
}

func TestPackInstall_MissingLockfileExitsNonZero(t *testing.T) {
	projectDir := t.TempDir()
	// No backstop.lock file.

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected error for missing lockfile")
	}

	if !strings.Contains(err.Error(), "backstop.lock") {
		t.Errorf("error should mention backstop.lock, got: %v", err)
	}
}

func TestPackInstall_AtomicRollbackOnCloneFailure(t *testing.T) {
	projectDir := t.TempDir()

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:abc",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{failWith: &distribution.GitError{Message: "clone failed"}},
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected error for clone failure")
	}

	// Verify no partial install.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "pack")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("should have rolled back on clone failure")
	}
}

func TestPackInstall_AtomicRollbackOnHashFailure(t *testing.T) {
	projectDir := t.TempDir()

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "intentionally-wrong-hash",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/pack\n")

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}

	// Verify rollback.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "pack")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("should have rolled back on hash failure")
	}
}

func TestPackInstall_WithExistingPacksDir(t *testing.T) {
	projectDir := t.TempDir()

	// Create existing packs dir to test snapshot path.
	existingPack := filepath.Join(projectDir, ".backstop", "packs", "old", "pack")
	if err := os.MkdirAll(existingPack, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(existingPack, "pack.yml"), "name: old/pack\n")

	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/pack\nversion: \"1.0.0\"\n")
	hash, _ := distribution.ComputeContentHash(sourceDir)

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
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

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
}

func TestPackInstall_NoGitClonerForGitPack(t *testing.T) {
	projectDir := t.TempDir()

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:abc",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		// No GitCloner provided.
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected error when no git cloner provided")
	}
}

func TestPackInstall_CacheMissingPack(t *testing.T) {
	projectDir := t.TempDir()
	cacheDir := t.TempDir()

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:abc",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		CachePath:  cacheDir,
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected error when pack not in cache")
	}
}
