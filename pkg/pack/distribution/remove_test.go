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
		"packs:\n  - name: acme/valid-pack\n    version: \"1.0.0\"\n")

	// Create installed pack.
	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "valid-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")

	// Create lockfile.
	hash, _ := distribution.ComputeContentHash(packDir)
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

	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
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
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs: []\n")

	_, err := distribution.Remove("acme/nonexistent", distribution.RemoveOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error for pack not installed")
	}

	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error should mention 'not installed', got: %v", err)
	}
}
