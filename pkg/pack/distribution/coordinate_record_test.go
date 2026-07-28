package distribution_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-004 suite: the requested org/repository is recorded VERBATIM, and every command
// that rewrites a lock entry PRESERVES it.
//
// Recording it is only half the requirement. The moment REQ-003 keyed the lock by manifest
// name, a divergent-name pack became uninstallable from its own lock unless the coordinate
// survives — so a rewrite that silently drops the field is not a cosmetic loss, it breaks
// the pack. recordGitPackInLock REPLACES the whole LockEntry rather than updating it,
// which is exactly the shape that drops a field nobody remembered to copy.

// seedLockWithCoordinate writes a lock entry whose coordinate DIFFERS from its pack name,
// which is the only shape that can catch a rewrite dropping the field. Where the two
// strings coincide, a preservation test passes against an implementation that re-derived
// the coordinate from the name — the very derivation REQ-003 removes.
func seedLockWithCoordinate(t *testing.T, projectDir, packName, coordinate, version, hash string) {
	t.Helper()
	ref := "v" + version
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:             packName,
				Version:          version,
				GitRef:           &ref,
				ContentHash:      hash,
				SourceType:       "git",
				InstallDate:      "2026-01-01T00:00:00Z",
				SourceCoordinate: coordinate,
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatalf("seeding the lock: %v", err)
	}
}

func readLockEntry(t *testing.T, projectDir, packName string) distribution.LockEntry {
	t.Helper()
	lf, err := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	entry, ok := lf.Packs[packName]
	if !ok {
		t.Fatalf("no lock entry for %q; keys are %v", packName, lockKeys(lf))
	}
	return entry
}

// ── Recording ───────────────────────────────────────────────────────────────────

func TestAddCommand_Run_RecordsSourceCoordinate(t *testing.T) {
	projectDir := setupAddProject(t)

	if _, _, err := addWithManifest(t, projectDir, "acme/pack-repo@1.0.0",
		identityManifestYAML("acme/pack", "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entry := readLockEntry(t, projectDir, "acme/pack")
	if entry.SourceCoordinate != "acme/pack-repo" {
		t.Errorf("source_coordinate = %q, want the requested repository %q", entry.SourceCoordinate, "acme/pack-repo")
	}
}

// TestAddCommand_Run_RecordsMixedCaseCoordinateVerbatim asserts EQUALITY with the exact
// input string (CLM-041). A case-insensitive containment check would pass against a
// "helpful" fold, and this assertion is the only thing standing between here and one.
func TestAddCommand_Run_RecordsMixedCaseCoordinateVerbatim(t *testing.T) {
	projectDir := setupAddProject(t)
	const coordinate = "Backstop-AI/Go-Standards"

	if _, _, err := addWithManifest(t, projectDir, coordinate+"@1.0.0",
		identityManifestYAML("backstop/go-standards", "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entry := readLockEntry(t, projectDir, "backstop/go-standards")
	if entry.SourceCoordinate != coordinate {
		t.Errorf("source_coordinate = %q, want %q byte-for-byte — case-insensitivity is a GitHub property and packs may be hosted anywhere (DD-31)",
			entry.SourceCoordinate, coordinate)
	}
}

// TestAddCommand_Run_RecordsSuffixedCoordinateVerbatim uses the published fleet's real
// shape: the pack is NAMED backstop/harness-toolchain and LIVES at
// backstop-ai/backstop-harness-toolchain-pack (CLM-042).
func TestAddCommand_Run_RecordsSuffixedCoordinateVerbatim(t *testing.T) {
	projectDir := setupAddProject(t)
	const coordinate = "backstop-ai/backstop-harness-toolchain-pack"

	if _, _, err := addWithManifest(t, projectDir, coordinate+"@1.0.0",
		identityManifestYAML("backstop/harness-toolchain", "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entry := readLockEntry(t, projectDir, "backstop/harness-toolchain")
	if entry.SourceCoordinate != coordinate {
		t.Errorf("source_coordinate = %q, want %q with its -pack suffix intact", entry.SourceCoordinate, coordinate)
	}
}

// TestAddCommand_Run_RecordedCoordinateExcludesVersionSuffix pins that the @version
// suffix is removed and NOTHING ELSE is (CLM-043).
func TestAddCommand_Run_RecordedCoordinateExcludesVersionSuffix(t *testing.T) {
	projectDir := setupAddProject(t)

	if _, _, err := addWithManifest(t, projectDir, "acme/pack@1.0.0",
		identityManifestYAML("acme/pack", "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entry := readLockEntry(t, projectDir, "acme/pack")
	if entry.SourceCoordinate != "acme/pack" {
		t.Errorf("source_coordinate = %q, want %q — the version suffix is stripped and nothing else is", entry.SourceCoordinate, "acme/pack")
	}
	if strings.Contains(entry.SourceCoordinate, "@") {
		t.Errorf("source_coordinate %q still carries an @version suffix", entry.SourceCoordinate)
	}
}

// TestAddCommand_Run_LocalSourceRecordsNoCoordinate asserts on the emitted lock TEXT
// (CLM-044), not merely that the struct field is empty.
//
// The text is where phase 4's omitempty guard is exercised END TO END: an empty field
// that still emitted `source_coordinate: ""` would satisfy a struct assertion perfectly
// while putting a blank key into every local consumer's tracked lock.
func TestAddCommand_Run_LocalSourceRecordsNoCoordinate(t *testing.T) {
	projectDir := setupAddProject(t)

	localDir := filepath.Join(t.TempDir(), "local-src")
	writeFile(t, filepath.Join(localDir, "pack.yml"), identityManifestYAML("internal/local-rules", "1.0.0"))

	add := newTestAddCommand(t, defaultTestPackCloner(), &mockValidator{})
	if _, err := add.Run(localDir, distribution.AddOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("local Add: %v", err)
	}

	entry := readLockEntry(t, projectDir, "internal/local-rules")
	if entry.SourceCoordinate != "" {
		t.Errorf("a local-path add recorded source_coordinate %q; local_path already records its source", entry.SourceCoordinate)
	}

	lockText := string(mustReadFile(t, filepath.Join(projectDir, "backstop.lock")))
	if strings.Contains(lockText, "source_coordinate") {
		t.Errorf("the emitted lock carries a source_coordinate key for a local pack; the omitempty guard must keep it out entirely.\nGot:\n%s", lockText)
	}
}

// ── Preservation ────────────────────────────────────────────────────────────────

// TestUpdateCommand_Run_PreservesRecordedCoordinate (CLM-048).
//
// recordGitPackInLock builds a FRESH LockEntry rather than updating the existing one, so
// any field it does not explicitly carry is dropped on every update. The seeded
// coordinate differs from the pack name precisely so this can fail.
func TestUpdateCommand_Run_PreservesRecordedCoordinate(t *testing.T) {
	projectDir := setupUpdateProject(t)
	const coordinate = "acme/valid-pack-repo"

	existing := readLockEntry(t, projectDir, "acme/valid-pack")
	seedLockWithCoordinate(t, projectDir, "acme/valid-pack", coordinate, "1.0.0", existing.ContentHash)

	update := newTestUpdateCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		&mockValidator{},
		&mockVersionResolver{latestMinor: "1.1.0"},
	)
	if _, err := update.Run("acme/valid-pack", distribution.UpdateOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	entry := readLockEntry(t, projectDir, "acme/valid-pack")
	if entry.SourceCoordinate != coordinate {
		t.Errorf("source_coordinate = %q after an update, want the recorded %q preserved — a rewrite that drops it makes the pack uninstallable from its own lock",
			entry.SourceCoordinate, coordinate)
	}
}

// TestUpgradeCommand_Run_PreservesRecordedCoordinate (CLM-049). Same defect, same shape.
func TestUpgradeCommand_Run_PreservesRecordedCoordinate(t *testing.T) {
	projectDir := setupUpgradeProject(t)
	const coordinate = "acme/valid-pack-repo"

	existing := readLockEntry(t, projectDir, "acme/valid-pack")
	seedLockWithCoordinate(t, projectDir, "acme/valid-pack", coordinate, "1.0.0", existing.ContentHash)

	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		&mockScanner{},
		&mockRemediationGenerator{},
	)
	if _, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	entry := readLockEntry(t, projectDir, "acme/valid-pack")
	if entry.SourceCoordinate != coordinate {
		t.Errorf("source_coordinate = %q after an upgrade, want the recorded %q preserved", entry.SourceCoordinate, coordinate)
	}
}

// TestRelock_PreservesEveryFieldItDoesNotRefresh is a CHARACTERIZATION test and passes on
// arrival (CLM-050).
//
// Relock (relock.go:29-68) already does a READ-MODIFY-WRITE: it reads the entry, sets
// ContentHash and InstallDate, and writes the same struct back, so every other field
// survives by construction. Nothing in this spec changes that.
//
// What this test is FOR is the day someone converts relock to the whole-entry REPLACE that
// recordGitPackInLock uses — the same shape that drops source_coordinate on update and
// upgrade. It would look like a tidy-up and would silently strip provenance from every
// local pack.
func TestRelock_PreservesEveryFieldItDoesNotRefresh(t *testing.T) {
	projectDir := t.TempDir()

	const packName = "internal/local-rules"
	installedDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(packName))
	writeFile(t, filepath.Join(installedDir, "pack.yml"), identityManifestYAML(packName, "1.0.0"))

	sourceDir := filepath.Join(projectDir, "local-src")
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), identityManifestYAML(packName, "1.0.0"))

	// A local entry carrying a coordinate it has no business losing, plus the other
	// fields relock must leave alone.
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:             packName,
				Version:          "1.0.0",
				GitRef:           nil,
				ContentHash:      "sha256:stale",
				SourceType:       "local",
				InstallDate:      "2020-01-01T00:00:00Z",
				LocalPath:        "local-src",
				SourceCoordinate: "legacy/coordinate-kept",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	if _, err := distribution.Relock(projectDir, sourceDir); err != nil {
		t.Fatalf("Relock: %v", err)
	}

	entry := readLockEntry(t, projectDir, packName)

	// REFRESHED.
	if entry.ContentHash == "sha256:stale" {
		t.Error("relock did not refresh the content hash")
	}
	if entry.InstallDate == "2020-01-01T00:00:00Z" {
		t.Error("relock did not refresh the install date")
	}
	// EVERYTHING ELSE, untouched.
	if entry.SourceCoordinate != "legacy/coordinate-kept" {
		t.Errorf("source_coordinate = %q, want it left exactly as found; relock refreshes two fields and must preserve the rest", entry.SourceCoordinate)
	}
	if entry.LocalPath != "local-src" {
		t.Errorf("local_path = %q, want it preserved", entry.LocalPath)
	}
	if entry.Version != "1.0.0" {
		t.Errorf("version = %q, want it preserved", entry.Version)
	}
	if entry.SourceType != "local" {
		t.Errorf("source_type = %q, want it preserved", entry.SourceType)
	}
}
