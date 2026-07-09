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
		"packs:\n  acme/valid-pack: \"1.0.0\"\n")

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
		"packs:\n  internal/local-rules: local\n")

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
		"packs:\n  internal/local-rules: local\n")

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

func TestPackList_MissingBackstopYml(t *testing.T) {
	projectDir := t.TempDir()
	// No backstop.yml file.

	_, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error for missing backstop.yml")
	}

	if !strings.Contains(err.Error(), "reading backstop.yml") {
		t.Errorf("error should mention reading backstop.yml, got: %v", err)
	}
}

func TestPackList_MalformedBackstopYml(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs: [invalid: yaml: {{{")

	_, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected error for malformed backstop.yml")
	}

	if !strings.Contains(err.Error(), "parsing backstop.yml") {
		t.Errorf("error should mention parsing backstop.yml, got: %v", err)
	}
}

func TestPackList_PackNotInLockfile(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  acme/unlocked-pack: \"1.0.0\"\n")

	// Lockfile with a different pack.
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"other/pack": {
				Name:        "other/pack",
				ContentHash: "sha256:abc",
				SourceType:  "git",
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

	if len(result.Packs) == 0 {
		t.Fatal("expected at least one pack")
	}

	pack := result.Packs[0]
	if pack.LockStatus != "missing" {
		t.Errorf("LockStatus = %q, want %q for pack not in lockfile", pack.LockStatus, "missing")
	}
}

func TestPackList_VersionBackfillFromLock(t *testing.T) {
	projectDir := t.TempDir()
	// backstop.yml with no version field.
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  acme/valid-pack: local\n")

	// Create installed pack dir for lock status computation.
	packDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "name: acme/valid-pack\n")

	hash, _ := distribution.ComputeContentHash(packDir)
	ref := "v2.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/valid-pack": {
				Name:        "acme/valid-pack",
				Version:     "2.0.0",
				GitRef:      &ref,
				ContentHash: hash,
				SourceType:  "git",
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

	pack := result.Packs[0]
	if pack.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q (should backfill from lockfile)", pack.Version, "2.0.0")
	}
}

func TestPackList_EmptyPacksList(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs: {}\n")

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(result.Packs) != 0 {
		t.Errorf("expected empty packs list, got %d", len(result.Packs))
	}

	// Table output should contain header only.
	if !strings.Contains(result.FormattedOutput, "NAME") {
		t.Error("formatted output should contain header even for empty list")
	}
}

func TestPackList_NoLockfile(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  acme/some-pack: \"1.0.0\"\n")
	// No backstop.lock.

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	pack := result.Packs[0]
	// Without a lockfile, LockStatus should be empty.
	if pack.LockStatus != "" {
		t.Errorf("LockStatus = %q, want empty when no lockfile", pack.LockStatus)
	}
}

func TestPackList_TableEmptyVersion(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  acme/no-version: local\n")
	// No lockfile either, so version stays empty.

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Table output should have "-" for empty version.
	if !strings.Contains(result.FormattedOutput, "-") {
		t.Error("table should show '-' for empty version")
	}
}

func TestPackList_ManifestInvalidYaml(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  acme/bad-pack: \"1.0.0\"\n")

	// Create pack dir with invalid pack.yml.
	packDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "bad-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), "not: [valid: yaml: {{{")

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List should succeed even with invalid manifest: %v", err)
	}

	// Pack should be listed but without metadata.
	pack := result.Packs[0]
	if pack.Archetype != "" {
		t.Error("expected empty archetype for invalid manifest")
	}
	if pack.RuleCount != 0 {
		t.Error("expected zero rule count for invalid manifest")
	}
}

func TestPackList_ManifestMissing(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  acme/no-manifest: \"1.0.0\"\n")

	// Create pack dir but no pack.yml inside.
	packDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "no-manifest")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List should succeed without manifest: %v", err)
	}

	pack := result.Packs[0]
	if pack.Archetype != "" {
		t.Error("expected empty archetype when manifest is missing")
	}
}

// installListPack writes backstop.yml + an installed pack.yml + a matching lockfile
// entry, and returns the project dir. sourceType selects git/local.
func installListPack(t *testing.T, packYml, sourceType, versionField string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "backstop.yml"),
		"packs:\n  acme/modern-pack: "+versionField+"\n")
	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "modern-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), packYml)
	hash, _ := distribution.ComputeContentHash(packDir)
	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{
		"acme/modern-pack": {
			Name: "acme/modern-pack", ContentHash: hash, SourceType: sourceType,
			InstallDate: "2026-01-01T00:00:00Z",
		},
	}}
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestPackList_RuleCountFromContentRuleset (ISSUE-032 Defect D / CLM-008): a modern
// pack nesting its rules under content.ruleset.rules yields a real RuleCount, not 0.
func TestPackList_RuleCountFromContentRuleset(t *testing.T) {
	modern := "name: acme/modern-pack\nversion: 1.0.0\nlanguage: go\narchetype: enforcement\n" +
		"content:\n  ruleset:\n    rules:\n      - id: a\n        risk_class: style\n      - id: b\n        risk_class: style\n"
	projectDir := installListPack(t, modern, "git", "\"1.0.0\"")
	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Packs[0].RuleCount != 2 {
		t.Errorf("RuleCount = %d, want 2 (from content.ruleset.rules)", result.Packs[0].RuleCount)
	}
}

// TestPackList_RuleCountLegacyTopLevel guards the fallback: a legacy pack with
// top-level rules: is still counted (CLM-008).
func TestPackList_RuleCountLegacyTopLevel(t *testing.T) {
	legacy := "name: acme/modern-pack\nversion: 1.0.0\narchetype: rule-pack\n" +
		"rules:\n  - id: RULE-001\n  - id: RULE-002\n  - id: RULE-003\n"
	projectDir := installListPack(t, legacy, "git", "\"1.0.0\"")
	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Packs[0].RuleCount != 3 {
		t.Errorf("RuleCount = %d, want 3 (legacy top-level rules fallback)", result.Packs[0].RuleCount)
	}
}

// TestPackList_LocalPackUnchangedIsCurrent (ISSUE-032 Defect D / CLM-008): a
// local-source pack whose installed content matches its lock entry reads "current",
// not "stale".
func TestPackList_LocalPackUnchangedIsCurrent(t *testing.T) {
	modern := "name: acme/modern-pack\nversion: 1.0.0\nlanguage: go\narchetype: enforcement\n" +
		"content:\n  ruleset:\n    rules:\n      - id: a\n        risk_class: style\n"
	projectDir := installListPack(t, modern, "local", "local")
	result, err := distribution.List(distribution.ListOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := result.Packs[0].LockStatus; got != "current" {
		t.Errorf("LockStatus = %q, want %q for an unchanged local pack", got, "current")
	}
}
