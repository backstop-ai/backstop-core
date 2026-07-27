package distribution_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// --- Upgrade mocks ---

type mockRemediationGenerator struct {
	failWith error
}

func (m *mockRemediationGenerator) GenerateBundle(_ string, _ []string) (string, error) {
	if m.failWith != nil {
		return "", fmt.Errorf("generating remediation bundle: %w", m.failWith)
	}
	return "remediation-bundle.md", nil
}

type mockScanner struct {
	violations []string
}

func (m *mockScanner) ScanViolations(_, _ string) ([]string, error) {
	return m.violations, nil
}

func setupUpgradeProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "backstop.yml"),
		"packs:\n  acme/valid-pack: \"1.0.0\"\n")

	packDir := filepath.Join(dir, ".backstop", "packs", "acme", "valid-pack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("testdata", "valid-pack", "pack.yml")
	data := mustReadFile(t, src)
	writeFile(t, filepath.Join(packDir, "pack.yml"), string(data))

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

	backstopDir := filepath.Join(dir, ".backstop")
	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	if err := distribution.WriteProvenance(filepath.Join(backstopDir, "pack-config-provenance.json"), prov); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestPackUpgrade_AcceptsMajorVersionTarget(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{violations: []string{"violation-1"}}, &mockRemediationGenerator{})

	result, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.NewVersion != "2.0.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "2.0.0")
	}
}

func TestPackUpgrade_GeneratesRemediationBundle(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{violations: []string{"v1"}}, &mockRemediationGenerator{})

	result, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.RemediationBundle == "" {
		t.Error("expected remediation bundle to be generated")
	}
}

func TestPackUpgrade_BaselinesNewViolations(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{violations: []string{"v1", "v2", "v3"}}, &mockRemediationGenerator{})

	result, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.BaselinedViolations != 3 {
		t.Errorf("BaselinedViolations = %d, want 3", result.BaselinedViolations)
	}
}

func TestPackUpgrade_ValidatesBeforeInstall(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
}

func TestPackUpgrade_AbortsOnValidationFailure(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{checkFail: true}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when validation fails")
	}
}

func TestPackUpgrade_UpdatesToolConfig(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	// Verify provenance was updated.
	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	prov := mustReadProvenance(t, provPath)
	if len(prov.Entries) == 0 {
		t.Error("expected provenance entries after upgrade with tool_config")
	}
}

func TestPackUpgrade_ToolConfigConflictExitsNonZero(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	// Write conflicting config.
	writeFile(t, filepath.Join(projectDir, ".golangci.yml"),
		`{"linters.enable.revive": false, "linters.enable.errcheck": false}`)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error for tool_config conflict")
	}

	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("error should mention conflict, got: %v", err)
	}
}

func TestPackUpgrade_UpdatesBackstopYml(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	data := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "2.0.0") {
		t.Error("backstop.yml should contain new version 2.0.0")
	}
}

func TestPackUpgrade_UpdatesBackstopLock(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	lf := mustReadLock(t, filepath.Join(projectDir, "backstop.lock"))
	entry := lf.Packs["acme/valid-pack"]
	if entry.Version != "2.0.0" {
		t.Errorf("lockfile version = %q, want %q", entry.Version, "2.0.0")
	}
}

func TestPackUpgrade_RemediationBundleCoversAllViolations(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{violations: []string{"v1", "v2"}}, &mockRemediationGenerator{})

	result, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.BaselinedViolations != 2 {
		t.Errorf("expected 2 baselined violations, got %d", result.BaselinedViolations)
	}
}

func TestPackUpgrade_RollbackOnRemediationFailure(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{violations: []string{"v1"}}, &mockRemediationGenerator{failWith: fmt.Errorf("remediation failed")})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when remediation fails")
	}

	// Verify old version retained.
	data := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "1.0.0") {
		t.Error("should retain old version on remediation failure")
	}
}

func TestPackUpgrade_FailsWhenCloneFails(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{failWith: &distribution.GitError{Message: "clone failed"}}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when clone fails")
	}

	// Verify old version retained.
	data := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "1.0.0") {
		t.Error("should retain old version on clone failure")
	}
}

func TestPackUpgrade_FailsWhenRunPackTestFails(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{testFail: true}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when pack test fails")
	}
}

func TestPackUpgrade_NoRemediationWhenNoViolations(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{violations: []string{}}, &mockRemediationGenerator{})

	result, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.RemediationBundle != "" {
		t.Errorf("expected empty remediation bundle when no violations, got %q", result.RemediationBundle)
	}
	if result.BaselinedViolations != 0 {
		t.Errorf("BaselinedViolations = %d, want 0", result.BaselinedViolations)
	}
}

func TestPackUpgrade_ResultOldVersion(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	result, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.OldVersion != "1.0.0" {
		t.Errorf("OldVersion = %q, want %q", result.OldVersion, "1.0.0")
	}
}

func TestPackUpgrade_ResultContentHash(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	result, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.ContentHash == "" {
		t.Error("expected non-empty ContentHash")
	}
}

func TestPackUpgrade_CreatesLockfileWhenAbsent(t *testing.T) {
	projectDir := setupUpgradeProject(t)
	// Delete lockfile.
	if err := os.Remove(filepath.Join(projectDir, "backstop.lock")); err != nil {
		t.Fatal(err)
	}

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade should create lockfile: %v", err)
	}

	lf, readErr := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if readErr != nil {
		t.Fatalf("lockfile should exist: %v", readErr)
	}

	entry := lf.Packs["acme/valid-pack"]
	if entry.Version != "2.0.0" {
		t.Errorf("version = %q, want %q", entry.Version, "2.0.0")
	}
}

func TestPackUpgrade_FailsWhenBackstopYmlMissing(t *testing.T) {
	projectDir := t.TempDir()
	// No backstop.yml.

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when backstop.yml is missing")
	}
}

func TestPackUpgrade_ScanViolationsError(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScannerWithError{err: fmt.Errorf("scan failed")}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when scan fails")
	}

	if !strings.Contains(err.Error(), "scanning violations") {
		t.Errorf("error should mention scanning violations, got: %v", err)
	}
}

func TestPackUpgrade_ReadProvenanceError(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	// Corrupt the provenance file.
	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	writeFile(t, provPath, "{{{invalid json")

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error for corrupted provenance")
	}
}

// TestPackUpgrade_MergeToolConfigError keeps its identifier and its assertion; only the
// way it REACHES the merge failure changed (SPEC-056 edit-set item TWELVE).
//
// It is the UPGRADE twin of TestPackAdd_MergeToolConfigErrorRollsBack. Both used to
// provoke the failure with an unparseable pack.yml, relying on MergeToolConfig ->
// readPackManifest being the first thing to choke on it. After the identity gate reached
// upgrade (TASK-038), such a pack is refused far earlier by ReadManifestIdentity, so the
// test's PREMISE no longer held and it would have gone on asserting a refusal it never
// actually reached.
//
// WHY THE PLAN'S SWEEP MISSED THIS ONE: the edit-set method paired every cloneDir literal
// with its Run() ref, which finds every test pointing at a NAMED fixture. This test builds
// its manifest INLINE in a t.TempDir(), so it has no fixture literal to pair and the sweep
// was blind to it by construction.
//
// The manifest now PARSES and identity-validates (name and version both real, the version
// matching the @2.0.0 ref the gate compares against). The failure comes from the merge
// itself: .golangci.yml exists as a DIRECTORY, so writeConfigFile cannot write the merged
// settings (config_merge.go:107), reached from upgrade at command.go:699.
func TestPackUpgrade_MergeToolConfigError(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	// Identity-VALID, but its tool_config cannot be applied.
	badPackDir := t.TempDir()
	writeFile(t, filepath.Join(badPackDir, "pack.yml"),
		"name: acme/valid-pack\nversion: \"2.0.0\"\narchetype: rule-pack\n"+
			"tool_config:\n  - config_file: \".golangci.yml\"\n    settings:\n      linters.enable.revive: true\n")

	// The merge target is a DIRECTORY, so writing the merged config fails.
	if mkErr := os.MkdirAll(filepath.Join(projectDir, ".golangci.yml"), 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	upgrade := newTestUpgradeCommand(t, &mockGitCloner{cloneDir: badPackDir}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error for invalid pack manifest during merge")
	}

	if !strings.Contains(err.Error(), "merging tool_config") {
		t.Errorf("error should mention merging tool_config, got: %v", err)
	}
}

// mockScannerRemovingManifest deletes the consumer's backstop.yml as a side
// effect of scanning, so the upgrade reaches its MANIFEST WRITE with the file
// gone. The scan is the last step before any consumer mutation, which makes it
// the only seam from which a late write failure can be staged.
type mockScannerRemovingManifest struct{}

func (m *mockScannerRemovingManifest) ScanViolations(projectDir, _ string) ([]string, error) {
	if err := os.Remove(filepath.Join(projectDir, "backstop.yml")); err != nil {
		return nil, fmt.Errorf("staging the scenario: removing the manifest mid-upgrade: %w", err)
	}
	return nil, nil
}

// TestPackUpgrade_ManifestWriteFailureNamesTheFile asserts a manifest write that
// fails LATE — after validation, the scan, the merge, and the install have all
// succeeded — surfaces a diagnostic naming backstop.yml.
//
// The write is the second-to-last step, so its failure is the easiest one to
// return bare; a bare os.ReadFile error at that point tells an operator only
// "no such file or directory" with no indication of which file or which command.
func TestPackUpgrade_ManifestWriteFailureNamesTheFile(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		&mockScannerRemovingManifest{},
		&mockRemediationGenerator{},
	)

	_, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("upgrade must fail when it cannot record the new version in the manifest")
	}
	if !strings.Contains(err.Error(), "backstop.yml") {
		t.Errorf("error = %v, want it to name backstop.yml so an operator knows which file could not be written", err)
	}
}

// mockScannerWithError returns an error from ScanViolations.
type mockScannerWithError struct {
	err error
}

func (m *mockScannerWithError) ScanViolations(_, _ string) ([]string, error) {
	return nil, m.err
}
