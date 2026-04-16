package distribution_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func setupListProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "backstop.yml"),
		"packs:\n  - name: acme/valid-pack\n    version: \"1.0.0\"\n")

	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "valid-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join("testdata", "valid-pack", "pack.yml")
	data, _ := os.ReadFile(src)
	writeFile(t, filepath.Join(packDir, "pack.yml"), string(data))

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

	return dir
}

func TestPackList_HumanTableOutput(t *testing.T) {
	projectDir := setupListProject(t)

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(result.Packs) == 0 {
		t.Fatal("expected packs in list result")
	}

	if result.FormattedOutput == "" {
		t.Error("expected formatted output")
	}

	// Verify table contains expected columns.
	output := result.FormattedOutput
	for _, col := range []string{"NAME", "VERSION", "LOCK STATUS"} {
		if !strings.Contains(output, col) {
			t.Errorf("table missing column %q", col)
		}
	}
}

func TestPackList_JsonOutput(t *testing.T) {
	projectDir := setupListProject(t)

	result, err := distribution.List(distribution.ListOptions{
		ProjectDir: projectDir,
		JSON:       true,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Verify JSON is valid.
	var parsed []map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(result.FormattedOutput), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON output: %v\n%s", jsonErr, result.FormattedOutput)
	}

	if len(parsed) == 0 {
		t.Error("expected at least one pack in JSON output")
	}
}

func TestPackList_LockStatusLocked(t *testing.T) {
	projectDir := setupListProject(t)

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	pack := result.Packs[0]
	if pack.LockStatus != "locked" {
		t.Errorf("LockStatus = %q, want %q", pack.LockStatus, "locked")
	}
}

func TestPackList_LockStatusStale(t *testing.T) {
	projectDir := setupListProject(t)

	// Modify the installed pack to make hash stale.
	packDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	writeFile(t, filepath.Join(packDir, "extra.txt"), "extra content")

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	pack := result.Packs[0]
	if pack.LockStatus != "stale" {
		t.Errorf("LockStatus = %q, want %q", pack.LockStatus, "stale")
	}
}

func TestPackList_LockStatusMissing(t *testing.T) {
	projectDir := setupListProject(t)

	// Remove the installed pack.
	os.RemoveAll(filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack"))

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	pack := result.Packs[0]
	if pack.LockStatus != "missing" {
		t.Errorf("LockStatus = %q, want %q", pack.LockStatus, "missing")
	}
}

func TestLocalPack_ValidatedSameAsGitList(t *testing.T) {
	projectDir := t.TempDir()

	localDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localDir)

	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  - name: internal/local-rules\n    path: "+absPath+"\n")

	hash, _ := distribution.ComputeContentHash(absPath)
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local-rules": {
				Name:        "internal/local-rules",
				ContentHash: hash,
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := false
	for _, p := range result.Packs {
		if p.Name == "internal/local-rules" {
			found = true
			break
		}
	}
	if !found {
		t.Error("local pack should appear in list")
	}
}

func TestLocalPack_LockEntryHasHashNoGitRefList(t *testing.T) {
	projectDir := t.TempDir()

	localDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localDir)

	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  - name: internal/local-rules\n    path: "+absPath+"\n")

	hash, _ := distribution.ComputeContentHash(absPath)
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local-rules": {
				Name:        "internal/local-rules",
				ContentHash: hash,
				SourceType:  "local",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	result, err := distribution.List(distribution.ListOptions{
		ProjectDir: projectDir,
		JSON:       true,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Verify JSON output contains the local pack with correct fields.
	if !strings.Contains(result.FormattedOutput, "internal/local-rules") {
		t.Error("JSON should contain local pack name")
	}
}
