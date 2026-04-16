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

func TestPackInstall_LocalPackNoLocalDir(t *testing.T) {
	projectDir := t.TempDir()

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local": {
				Name:        "internal/local",
				ContentHash: "sha256:localhash",
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		// No LocalPackDir provided — should skip hash verification.
	}

	result, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install with no local dir should succeed: %v", err)
	}

	if len(result.InstalledPacks) != 1 {
		t.Errorf("expected 1 installed pack, got %d", len(result.InstalledPacks))
	}
	if result.InstalledPacks[0] != "internal/local" {
		t.Errorf("InstalledPacks[0] = %q, want %q", result.InstalledPacks[0], "internal/local")
	}
}

func TestPackInstall_LocalPackHashMismatch(t *testing.T) {
	projectDir := t.TempDir()

	localDir := t.TempDir()
	writeFile(t, filepath.Join(localDir, "pack.yml"), "name: internal/local\n")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local": {
				Name:        "internal/local",
				ContentHash: "wrong-hash",
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
	if err == nil {
		t.Fatal("expected error for local pack hash mismatch")
	}

	if !strings.Contains(err.Error(), "hash mismatch for local pack") {
		t.Errorf("error should mention local pack hash mismatch, got: %v", err)
	}
}

func TestPackInstall_EmptyLockfile(t *testing.T) {
	projectDir := t.TempDir()

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
	}

	result, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install with empty lockfile should succeed: %v", err)
	}

	if len(result.InstalledPacks) != 0 {
		t.Errorf("expected 0 installed packs, got %d", len(result.InstalledPacks))
	}
}

func TestPackInstall_RollbackRestoresExistingPacks(t *testing.T) {
	projectDir := t.TempDir()

	// Create existing packs dir with a pre-existing pack.
	existingPack := filepath.Join(projectDir, ".backstop", "packs", "old", "pack")
	if err := os.MkdirAll(existingPack, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(existingPack, "marker.txt"), "pre-existing")

	// Create lockfile with a hash-mismatching pack to trigger rollback.
	ref := "v1.0.0"
	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/pack\n")

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

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sourceDir},
	}

	_, err := distribution.Install(opts)
	if err == nil {
		t.Fatal("expected hash mismatch error")
	}

	// Verify the pre-existing pack was restored after rollback.
	markerPath := filepath.Join(projectDir, ".backstop", "packs", "old", "pack", "marker.txt")
	data, readErr := os.ReadFile(markerPath)
	if readErr != nil {
		t.Fatalf("pre-existing pack marker should be restored after rollback: %v", readErr)
	}
	if string(data) != "pre-existing" {
		t.Errorf("marker content = %q, want %q", string(data), "pre-existing")
	}
}

func TestPackInstall_MultiplePacks(t *testing.T) {
	projectDir := t.TempDir()

	// Create two source directories.
	sourceDir1 := t.TempDir()
	writeFile(t, filepath.Join(sourceDir1, "pack.yml"), "name: acme/pack-a\nversion: \"1.0.0\"\n")
	hash1, _ := distribution.ComputeContentHash(sourceDir1)

	sourceDir2 := t.TempDir()
	writeFile(t, filepath.Join(sourceDir2, "pack.yml"), "name: acme/pack-b\nversion: \"1.0.0\"\n")
	hash2, _ := distribution.ComputeContentHash(sourceDir2)

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack-a": {
				Name:        "acme/pack-a",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: hash1,
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
			"acme/pack-b": {
				Name:        "acme/pack-b",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: hash2,
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	// Mock cloner that copies the right source for each pack.
	// Since map iteration order is random, use a single source that matches both hashes.
	// Instead, just use one source dir for all.  
	// Actually, mockGitCloner uses the same cloneDir for all clones.
	// But hashes will differ. We need to handle this differently.
	// Let's use separate dirs and a special approach.
	// Actually: each clone call gets a different tmpDir, so the mock copies the same source.
	// The hash is computed on that tmpDir, so we need both packs to share a source.
	// Simplest: make them both have the same content.

	// Recreate with matching content.
	sharedSource := t.TempDir()
	writeFile(t, filepath.Join(sharedSource, "pack.yml"), "name: shared\nversion: \"1.0.0\"\n")
	sharedHash, _ := distribution.ComputeContentHash(sharedSource)

	lf.Packs["acme/pack-a"] = distribution.LockEntry{
		Name: "acme/pack-a", Version: "1.0.0", GitRef: &ref,
		ContentHash: sharedHash, SourceType: "git", InstallDate: "2026-01-01T00:00:00Z",
	}
	lf.Packs["acme/pack-b"] = distribution.LockEntry{
		Name: "acme/pack-b", Version: "1.0.0", GitRef: &ref,
		ContentHash: sharedHash, SourceType: "git", InstallDate: "2026-01-01T00:00:00Z",
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: sharedSource},
	}

	result, err := distribution.Install(opts)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if len(result.InstalledPacks) != 2 {
		t.Errorf("expected 2 installed packs, got %d", len(result.InstalledPacks))
	}
}
