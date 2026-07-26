package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func setupRemoveProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Install a pack first.
	writeFile(t, filepath.Join(dir, "backstop.yml"),
		"packs:\n  acme/valid-pack: \"1.0.0\"\n")

	// Create installed pack.
	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "valid-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")

	// Create lockfile.
	hash := mustContentHash(t, packDir)
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
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	// Create provenance.
	prov := &distribution.Provenance{
		Entries: []distribution.ProvenanceEntry{
			{
				ConfigFile: ".golangci.yml",
				SettingKey: "linters.enable.revive",
				SourcePack: "acme/valid-pack",
				ValueHash:  computeSettingHash(true),
			},
		},
	}
	provPath := filepath.Join(dir, ".backstop", "pack-config-provenance.json")
	if err := distribution.WriteProvenance(provPath, prov); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestPackRemove_RevertsProvenanceSettings(t *testing.T) {
	projectDir := setupRemoveProject(t)

	result, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if len(result.RevertedSettings) == 0 {
		t.Error("expected reverted settings")
	}
}

func TestPackRemove_WarnsOnModifiedSetting(t *testing.T) {
	projectDir := setupRemoveProject(t)

	// Modify the setting value so hash no longer matches.
	writeFile(t, filepath.Join(projectDir, ".golangci.yml"), `{"linters.enable.revive": false}`)

	result, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Error("expected warnings for modified setting")
	}
}

func TestPackRemove_DeletesFromPacksDir(t *testing.T) {
	projectDir := setupRemoveProject(t)

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	packDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if _, statErr := os.Stat(packDir); !os.IsNotExist(statErr) {
		t.Error("pack directory should be deleted")
	}
}

func TestPackRemove_RemovesFromBackstopYml(t *testing.T) {
	projectDir := setupRemoveProject(t)

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	data := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))
	if strings.Contains(string(data), "acme/valid-pack") {
		t.Error("backstop.yml should not contain removed pack")
	}
}

func TestPackRemove_RemovesFromBackstopLock(t *testing.T) {
	projectDir := setupRemoveProject(t)

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	lf, readErr := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if readErr != nil {
		t.Fatalf("reading lockfile: %v", readErr)
	}

	if _, ok := lf.Packs["acme/valid-pack"]; ok {
		t.Error("lockfile should not contain removed pack")
	}
}

func TestPackRemove_RemovesFromProvenance(t *testing.T) {
	projectDir := setupRemoveProject(t)

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	prov, readErr := distribution.ReadProvenance(provPath)
	if readErr != nil {
		t.Fatalf("reading provenance: %v", readErr)
	}

	for _, entry := range prov.Entries {
		if entry.SourcePack == "acme/valid-pack" {
			t.Error("provenance should not contain entries for removed pack")
		}
	}
}

func TestPackRemove_NotInstalledExitsNonZero(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs: {}\n")

	_, err := distribution.Remove("acme/nonexistent", distribution.RemoveOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error for pack not installed")
	}

	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error should mention 'not installed', got: %v", err)
	}
}

// TestPackRemove_AbsentManifestExitsNonZero asserts a project with no
// backstop.yml at all refuses the removal rather than treating the missing
// manifest as "nothing declares this pack, so removing it is a no-op".
func TestPackRemove_AbsentManifestExitsNonZero(t *testing.T) {
	projectDir := t.TempDir()

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error when the project has no backstop.yml to read")
	}

	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error should mention 'not installed', got: %v", err)
	}
}

// TestPackRemove_MalformedManifestExitsNonZero asserts an unparseable
// backstop.yml is treated as declaring nothing, so the removal refuses instead
// of proceeding to delete files on the strength of a manifest it could not read.
func TestPackRemove_MalformedManifestExitsNonZero(t *testing.T) {
	projectDir := setupRemoveProject(t)
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  - this is not a mapping\n\t bad indent")

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error when backstop.yml cannot be parsed")
	}

	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error should mention 'not installed', got: %v", err)
	}

	// The installed pack must survive: a removal that could not read the manifest
	// must not delete content on the strength of it.
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")); statErr != nil {
		t.Errorf("installed pack was deleted despite the manifest being unreadable: %v", statErr)
	}
}

func TestPackRemove_NoLockfile(t *testing.T) {
	projectDir := setupRemoveProject(t)
	// Delete the lockfile.
	if err := os.Remove(filepath.Join(projectDir, "backstop.lock")); err != nil {
		t.Fatalf("deleting the lockfile the scenario needs absent: %v", err)
	}

	result, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove should succeed without lockfile: %v", err)
	}

	if len(result.RevertedSettings) == 0 {
		t.Error("expected reverted settings even without lockfile")
	}
}

func TestPackRemove_PreservesOtherPackProvenance(t *testing.T) {
	projectDir := setupRemoveProject(t)

	// Add a second pack's provenance entries.
	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	prov := mustReadProvenance(t, provPath)
	prov.Entries = append(prov.Entries, distribution.ProvenanceEntry{
		ConfigFile: ".eslintrc.json",
		SettingKey: "rules.no-console",
		SourcePack: "other/pack",
		ValueHash:  "sha256:other",
	})
	if err := distribution.WriteProvenance(provPath, prov); err != nil {
		t.Fatal(err)
	}

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Verify other pack's provenance entries survive.
	prov = mustReadProvenance(t, provPath)
	found := false
	for _, entry := range prov.Entries {
		if entry.SourcePack == "other/pack" {
			found = true
		}
		if entry.SourcePack == "acme/valid-pack" {
			t.Error("removed pack's provenance should be gone")
		}
	}
	if !found {
		t.Error("other pack's provenance entries should be preserved")
	}
}

func TestPackRemove_MultiplePacksInYml(t *testing.T) {
	projectDir := setupRemoveProject(t)

	// Add another pack to backstop.yml.
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  acme/valid-pack: \"1.0.0\"\n  other/pack: \"2.0.0\"\n")

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	content := string(mustReadFile(t, filepath.Join(projectDir, "backstop.yml")))

	if strings.Contains(content, "acme/valid-pack") {
		t.Error("removed pack should not be in backstop.yml")
	}
	if !strings.Contains(content, "other/pack") {
		t.Error("other pack should be preserved in backstop.yml")
	}
}

func TestPackRemove_SettingKeyNotFound(t *testing.T) {
	projectDir := setupRemoveProject(t)

	// Create config file with different keys than what provenance tracks.
	writeFile(t, filepath.Join(projectDir, ".golangci.yml"),
		`{"linters.enable.govet": true}`)

	result, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Setting should still be reverted (key not found → falls through to revert).
	if len(result.RevertedSettings) == 0 {
		t.Error("expected reverted settings when key not found in config")
	}

	// Should not generate any warnings since the key wasn't found (not modified).
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings when key not found, got %d", len(result.Warnings))
	}
}

func TestPackRemove_ReadProvenanceError(t *testing.T) {
	projectDir := setupRemoveProject(t)

	// Corrupt the provenance file.
	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	writeFile(t, provPath, "{{{invalid json")

	_, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error for corrupted provenance")
	}
}

func TestPackRemove_SettingHashMatches(t *testing.T) {
	projectDir := setupRemoveProject(t)

	// Write config with the exact same value that was tracked in provenance.
	// computeSettingHash(true) should match the provenance ValueHash.
	writeFile(t, filepath.Join(projectDir, ".golangci.yml"),
		`{"linters.enable.revive": true}`)

	result, err := distribution.Remove("acme/valid-pack", distribution.RemoveOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Hash matches → reverted, not warned.
	if len(result.RevertedSettings) == 0 {
		t.Error("expected reverted settings when hash matches")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings when hash matches, got %d", len(result.Warnings))
	}
}
