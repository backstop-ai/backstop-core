package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

const dogfoodPackName = "backstop/go-standards"

// TestDogfood_BackstopYmlDeclaresGoStandardsPack verifies backstop.yml declares
// the pack keyed "backstop/go-standards" in its packs map, the pack is installed
// under .backstop/packs/backstop/go-standards/ with a pack.yml whose name is
// backstop/go-standards, and the declaration resolves through loadInstalledPacks
// to a parseable manifest. (CLM-016)
func TestDogfood_BackstopYmlDeclaresGoStandardsPack(t *testing.T) {
	root := repoRoot(t)

	cfg, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatalf("loading backstop.yml: %v", err)
	}
	if _, ok := cfg.Packs[dogfoodPackName]; !ok {
		t.Fatalf("backstop.yml packs map does not declare %q; got %v", dogfoodPackName, cfg.Packs)
	}

	packDir := filepath.Join(root, ".backstop", "packs", "backstop", "go-standards")
	manifestPath := filepath.Join(packDir, "pack.yml")
	if _, statErr := os.Stat(manifestPath); statErr != nil {
		t.Fatalf("pack not installed at %s: %v", manifestPath, statErr)
	}
	manifest, parseErr := pack.ParseManifestFile(manifestPath)
	if parseErr != nil {
		t.Fatalf("parsing installed pack manifest: %v", parseErr)
	}
	if manifest.NormalizedName != dogfoodPackName {
		t.Errorf("installed pack name = %q, want %q", manifest.NormalizedName, dogfoodPackName)
	}

	// The declaration must resolve through the production loadInstalledPacks path.
	packs, loadErr := loadInstalledPacks(root)
	if loadErr != nil {
		t.Fatalf("loadInstalledPacks: %v", loadErr)
	}
	found := false
	for _, m := range packs {
		if m.NormalizedName == dogfoodPackName {
			found = true
		}
	}
	if !found {
		t.Errorf("loadInstalledPacks did not resolve %q; got %d packs", dogfoodPackName, len(packs))
	}
}

// TestDogfood_GoStandardsLockVerifies verifies backstop.lock contains the
// matching backstop/go-standards entry and VerifyLock passes. (CLM-017)
func TestDogfood_GoStandardsLockVerifies(t *testing.T) {
	root := repoRoot(t)

	lockPath := filepath.Join(root, "backstop.lock")
	lockfile, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("reading backstop.lock: %v", err)
	}
	if _, ok := lockfile.Packs[dogfoodPackName]; !ok {
		t.Fatalf("backstop.lock has no %q entry; got %v", dogfoodPackName, lockfile.Packs)
	}

	packsDir := filepath.Join(root, ".backstop", "packs")
	result, verifyErr := distribution.VerifyLock(lockfile, packsDir, []string{dogfoodPackName})
	if verifyErr != nil {
		t.Fatalf("VerifyLock: %v", verifyErr)
	}
	if !result.Pass {
		t.Errorf("VerifyLock failed: %+v", result.Failures)
	}
}

// TestDogfood_StaleSlotlyLockEntryRemoved verifies the stale slotly/go-standards
// entry is absent from backstop.lock (no declared-but-unlocked and no orphaned
// lock entry remains). (CLM-017)
func TestDogfood_StaleSlotlyLockEntryRemoved(t *testing.T) {
	root := repoRoot(t)

	lockfile, err := distribution.ReadLockfile(filepath.Join(root, "backstop.lock"))
	if err != nil {
		t.Fatalf("reading backstop.lock: %v", err)
	}
	if _, ok := lockfile.Packs["slotly/go-standards"]; ok {
		t.Errorf("backstop.lock still carries the stale slotly/go-standards entry; it must be removed")
	}
}
