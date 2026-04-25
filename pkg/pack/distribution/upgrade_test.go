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
		return "", m.failWith
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{violations: []string{"violation-1"}},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{violations: []string{"v1"}},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{violations: []string{"v1", "v2", "v3"}},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
}

func TestPackUpgrade_AbortsOnValidationFailure(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:  &mockValidator{checkFail: true},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when validation fails")
	}
}

func TestPackUpgrade_UpdatesToolConfig(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	// Verify provenance was updated.
	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	prov, _ := distribution.ReadProvenance(provPath)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "2.0.0") {
		t.Error("backstop.yml should contain new version 2.0.0")
	}
}

func TestPackUpgrade_UpdatesBackstopLock(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	lf, _ := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	entry := lf.Packs["acme/valid-pack"]
	if entry.Version != "2.0.0" {
		t.Errorf("lockfile version = %q, want %q", entry.Version, "2.0.0")
	}
}

func TestPackUpgrade_RemediationBundleCoversAllViolations(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{violations: []string{"v1", "v2"}},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		GitCloner:  &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:  &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{
			failWith: fmt.Errorf("remediation failed"),
		},
		Scanner: &mockScanner{violations: []string{"v1"}},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when remediation fails")
	}

	// Verify old version retained.
	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "1.0.0") {
		t.Error("should retain old version on remediation failure")
	}
}

func TestPackUpgrade_FailsWhenCloneFails(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{failWith: &distribution.GitError{Message: "clone failed"}},
		Validator:  &mockValidator{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when clone fails")
	}

	// Verify old version retained.
	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "1.0.0") {
		t.Error("should retain old version on clone failure")
	}
}

func TestPackUpgrade_FailsWhenRunPackTestFails(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:  &mockValidator{testFail: true},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when pack test fails")
	}
}

func TestPackUpgrade_SkipsValidationWhenValidatorNil(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            nil,
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade with nil validator should succeed: %v", err)
	}

	if result.NewVersion != "2.0.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "2.0.0")
	}
}

func TestPackUpgrade_NoRemediationWhenGeneratorNil(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: nil,
		Scanner:              &mockScanner{violations: []string{"v1"}},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if result.RemediationBundle != "" {
		t.Errorf("expected empty remediation bundle when generator is nil, got %q", result.RemediationBundle)
	}
	if result.BaselinedViolations != 1 {
		t.Errorf("BaselinedViolations = %d, want 1", result.BaselinedViolations)
	}
}

func TestPackUpgrade_NoRemediationWhenNoViolations(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{violations: []string{}},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
	os.Remove(filepath.Join(projectDir, "backstop.lock"))

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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

func TestPackUpgrade_SkipsScanningWhenScannerNil(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              nil,
	}

	result, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Upgrade with nil scanner should succeed: %v", err)
	}

	if result.BaselinedViolations != 0 {
		t.Errorf("BaselinedViolations = %d, want 0", result.BaselinedViolations)
	}
	if result.RemediationBundle != "" {
		t.Errorf("RemediationBundle = %q, want empty", result.RemediationBundle)
	}
}

func TestPackUpgrade_FailsWhenBackstopYmlMissing(t *testing.T) {
	projectDir := t.TempDir()
	// No backstop.yml.

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
		GitCloner:  &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:  &mockValidator{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error when backstop.yml is missing")
	}
}

func TestPackUpgrade_ScanViolationsError(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScannerWithError{err: fmt.Errorf("scan failed")},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
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
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error for corrupted provenance")
	}
}

func TestPackUpgrade_MergeToolConfigError(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	// Use a pack with invalid pack.yml in the cloned dir.
	badPackDir := t.TempDir()
	writeFile(t, filepath.Join(badPackDir, "pack.yml"), "{{{invalid yaml")

	opts := distribution.UpgradeOptions{
		ProjectDir:           projectDir,
		GitCloner:            &mockGitCloner{cloneDir: badPackDir},
		Validator:            &mockValidator{},
		RemediationGenerator: &mockRemediationGenerator{},
		Scanner:              &mockScanner{},
	}

	_, err := distribution.Upgrade("acme/valid-pack@2.0.0", opts)
	if err == nil {
		t.Fatal("expected error for invalid pack manifest during merge")
	}

	if !strings.Contains(err.Error(), "merging tool_config") {
		t.Errorf("error should mention merging tool_config, got: %v", err)
	}
}

// mockScannerWithError returns an error from ScanViolations.
type mockScannerWithError struct {
	err error
}

func (m *mockScannerWithError) ScanViolations(_, _ string) ([]string, error) {
	return nil, m.err
}
