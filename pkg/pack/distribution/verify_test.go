package distribution_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

func TestGate_HashMismatchFails(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs", "acme", "pack")
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packsDir, "pack.yml"), "name: acme/pack\n")

	hash, err := distribution.ComputeContentHash(packsDir)
	if err != nil {
		t.Fatal(err)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				ContentHash: hash + "-wrong",
				SourceType:  "git",
			},
		},
	}

	result, err := distribution.VerifyLock(lf, filepath.Join(dir, ".backstop", "packs"), []string{"acme/pack"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if result.Pass {
		t.Fatal("expected verification to fail for hash mismatch")
	}

	found := false
	for _, f := range result.Failures {
		if f.Kind == "hash_mismatch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected hash_mismatch failure")
	}
}

func TestGate_MissingPackFails(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/missing": {
				Name:        "acme/missing",
				ContentHash: "sha256:abc",
				SourceType:  "git",
			},
		},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{"acme/missing"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if result.Pass {
		t.Fatal("expected verification to fail for missing pack")
	}

	found := false
	for _, f := range result.Failures {
		if f.Kind == "missing_pack" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing_pack failure")
	}
}

func TestGate_ExtraUnlockedPackFails(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")

	// Create an extra pack not in lockfile.
	extraDir := filepath.Join(packsDir, "extra", "pack")
	if err := os.MkdirAll(extraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(extraDir, "pack.yml"), "name: extra/pack\n")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if result.Pass {
		t.Fatal("expected verification to fail for extra unlocked pack")
	}

	found := false
	for _, f := range result.Failures {
		if f.Kind == "extra_unlocked" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected extra_unlocked failure")
	}
}

func TestGate_AllPacksMatchPasses(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")
	packDir := filepath.Join(packsDir, "acme", "pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/pack\nversion: 1.0.0\n")

	hash, err := distribution.ComputeContentHash(packDir)
	if err != nil {
		t.Fatal(err)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				ContentHash: hash,
				SourceType:  "git",
			},
		},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{"acme/pack"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if !result.Pass {
		t.Errorf("expected verification to pass, got failures: %v", result.Failures)
	}
}

func TestGate_MissingLockfileWithPacksFails(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// nil lockfile with packs declared in backstop.yml.
	result, err := distribution.VerifyLock(nil, packsDir, []string{"acme/pack"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if result.Pass {
		t.Fatal("expected verification to fail for missing lockfile with packs")
	}

	found := false
	for _, f := range result.Failures {
		if f.Kind == "missing_lockfile" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing_lockfile failure")
	}
}

func TestGate_NoPacksNoLockfilePasses(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")

	// nil lockfile, no packs declared.
	result, err := distribution.VerifyLock(nil, packsDir, []string{})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if !result.Pass {
		t.Errorf("expected verification to pass with no packs and no lockfile")
	}
}

func TestGate_LocalPackVerifiedByHash(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a local pack at its source path.
	localDir := filepath.Join(dir, "local-rules")
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localDir, "pack.yml"), "name: internal/local\n")

	hash, err := distribution.ComputeContentHash(localDir)
	if err != nil {
		t.Fatal(err)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local": {
				Name:        "internal/local",
				ContentHash: hash,
				SourceType:  "local",
			},
		},
	}

	// For local packs, we verify at source path. Use a custom verifier
	// or override. Here we test that VerifyLock handles local source type.
	result, err := distribution.VerifyLock(lf, packsDir, []string{"internal/local"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	// Local packs are NOT expected in packsDir, so this should not fail
	// as "missing_pack" — the verifier should recognize local source type.
	if !result.Pass {
		for _, f := range result.Failures {
			t.Errorf("unexpected failure: %s - %s - %s", f.Pack, f.Kind, f.Message)
		}
		t.Fatal("expected pass for local pack")
	}
}

func TestGate_PackPathIsFileFails(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file instead of directory at the pack path.
	packPath := filepath.Join(packsDir, "acme", "pack")
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, packPath, "I am a file, not a directory")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				ContentHash: "sha256:abc",
				SourceType:  "git",
			},
		},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{"acme/pack"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if result.Pass {
		t.Fatal("expected fail when pack path is a file")
	}

	found := false
	for _, f := range result.Failures {
		if f.Kind == "missing_pack" {
			found = true
		}
	}
	if !found {
		t.Error("expected missing_pack failure when path is a file")
	}
}

func TestGate_MultipleLockedPacksAllPass(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")

	// Create two packs.
	pack1 := filepath.Join(packsDir, "acme", "pack-a")
	if err := os.MkdirAll(pack1, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pack1, "pack.yml"), "name: acme/pack-a\n")
	hash1, _ := distribution.ComputeContentHash(pack1)

	pack2 := filepath.Join(packsDir, "acme", "pack-b")
	if err := os.MkdirAll(pack2, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pack2, "pack.yml"), "name: acme/pack-b\n")
	hash2, _ := distribution.ComputeContentHash(pack2)

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack-a": {
				Name: "acme/pack-a", ContentHash: hash1, SourceType: "git",
			},
			"acme/pack-b": {
				Name: "acme/pack-b", ContentHash: hash2, SourceType: "git",
			},
		},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{"acme/pack-a", "acme/pack-b"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if !result.Pass {
		t.Errorf("expected pass, got failures: %v", result.Failures)
	}
}

func TestGate_NonDirEntryInPacksDirIgnored(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")
	if err := os.MkdirAll(packsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Place a regular file at org level (not a directory).
	writeFile(t, filepath.Join(packsDir, "README.md"), "not an org dir")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if !result.Pass {
		t.Errorf("expected pass, file at org level should be ignored, got: %v", result.Failures)
	}
}

func TestGate_NonDirEntryInOrgDirIgnored(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")

	orgDir := filepath.Join(packsDir, "acme")
	if err := os.MkdirAll(orgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Place a file inside the org directory (not a pack directory).
	writeFile(t, filepath.Join(orgDir, "notes.txt"), "not a pack")

	// Also add a valid pack so the org dir is walked.
	packDir := filepath.Join(orgDir, "valid-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/valid-pack\n")
	hash, _ := distribution.ComputeContentHash(packDir)

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/valid-pack": {
				Name: "acme/valid-pack", ContentHash: hash, SourceType: "git",
			},
		},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{"acme/valid-pack"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if !result.Pass {
		t.Errorf("expected pass, file in org dir should be ignored, got: %v", result.Failures)
	}
}

func TestGate_EmptyLockfileNoPacks(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if !result.Pass {
		t.Errorf("expected pass for empty lockfile and no packs, got: %v", result.Failures)
	}
}

func TestGate_MultipleFailuresCombined(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, ".backstop", "packs")

	// Create pack-b with wrong hash for hash_mismatch.
	packBDir := filepath.Join(packsDir, "acme", "pack-b")
	if err := os.MkdirAll(packBDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packBDir, "pack.yml"), "name: acme/pack-b\n")

	// Create pack-c on disk but NOT in lockfile for extra_unlocked.
	packCDir := filepath.Join(packsDir, "extra", "pack-c")
	if err := os.MkdirAll(packCDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packCDir, "pack.yml"), "name: extra/pack-c\n")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack-a": {
				Name: "acme/pack-a", ContentHash: "sha256:abc", SourceType: "git",
			},
			"acme/pack-b": {
				Name: "acme/pack-b", ContentHash: "wrong-hash", SourceType: "git",
			},
		},
	}

	result, err := distribution.VerifyLock(lf, packsDir, []string{"acme/pack-a", "acme/pack-b"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}

	if result.Pass {
		t.Fatal("expected failures")
	}

	kinds := make(map[string]bool)
	for _, f := range result.Failures {
		kinds[f.Kind] = true
	}

	if !kinds["missing_pack"] {
		t.Error("expected missing_pack failure for pack-a")
	}
	if !kinds["hash_mismatch"] {
		t.Error("expected hash_mismatch failure for pack-b")
	}
	if !kinds["extra_unlocked"] {
		t.Error("expected extra_unlocked failure for pack-c")
	}
}
