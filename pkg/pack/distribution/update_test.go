package distribution_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// --- Version resolver mock ---

type mockVersionResolver struct {
	latestMinor string
	latestMajor string
}

func (m *mockVersionResolver) ResolveLatestCompatible(_, currentVersion string) (string, error) {
	if m.latestMinor != "" {
		return m.latestMinor, nil
	}
	return currentVersion, nil
}

func (m *mockVersionResolver) IsMajorBump(current, resolved string) bool {
	if len(current) > 0 && len(resolved) > 0 {
		return current[0] != resolved[0]
	}
	return false
}

func setupUpdateProject(t *testing.T) string {
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

func TestPackUpdate_ResolvesLatestMinorPatch(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir: projectDir,
		GitCloner: &mockGitCloner{
			cloneDir: filepath.Join("testdata", "valid-pack-v2"),
		},
		Validator: &mockValidator{},
		VersionResolver: &mockVersionResolver{
			latestMinor: "1.1.0",
		},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if result.NewVersion != "1.1.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "1.1.0")
	}
}

func TestPackUpdate_ValidatesBeforeUpdate(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir: projectDir,
		GitCloner: &mockGitCloner{
			cloneDir: filepath.Join("testdata", "valid-pack-v2"),
		},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestPackUpdate_WritesExactPin(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	content := string(data)

	if !strings.Contains(content, "1.1.0") {
		t.Error("backstop.yml should contain exact pin 1.1.0")
	}
	if strings.Contains(content, "^") || strings.Contains(content, "~") {
		t.Error("should not contain range syntax")
	}
}

func TestPackUpdate_UpdatesLockfile(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	lf, _ := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	entry := lf.Packs["acme/valid-pack"]
	if entry.Version != "1.1.0" {
		t.Errorf("lockfile version = %q, want %q", entry.Version, "1.1.0")
	}
}

func TestPackUpdate_AbortsOnValidationFailure(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{checkFail: true},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error when validation fails")
	}

	// Verify old version retained.
	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "1.0.0") {
		t.Error("should retain old version on validation failure")
	}
}

func TestPackUpdate_AcknowledgeBypassesTamperBlock(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		Acknowledge:     true,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "tamper-pack-severity-downgrade")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.0.1"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update with --acknowledge should proceed: %v", err)
	}

	if result.NoOp {
		t.Error("expected update to proceed with --acknowledge")
	}
}

func TestPackUpdate_LocalPackNoOp(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  - name: internal/local\n    path: /some/local/path\n")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local": {
				Name:       "internal/local",
				SourceType: "local",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := distribution.UpdateOptions{
		ProjectDir: projectDir,
	}

	result, err := distribution.Update("internal/local", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !result.NoOp {
		t.Error("expected no-op for local pack")
	}
}

func TestPackUpdate_WritesResolvedExactPin(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if result.NewVersion != "1.1.0" {
		t.Errorf("resolved pin = %q, want exact %q", result.NewVersion, "1.1.0")
	}
}

func TestPackUpdate_AppliesPatchVersion(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.0.1"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if result.NewVersion != "1.0.1" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "1.0.1")
	}
}

func TestPackUpdate_AppliesMinorVersion(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if result.NewVersion != "1.1.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "1.1.0")
	}
}

func TestPackUpdate_RefusesMajorVersion(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "2.0.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error for major version bump")
	}

	if !strings.Contains(err.Error(), "pack upgrade") {
		t.Errorf("error should suggest pack upgrade, got: %v", err)
	}
}

func TestPackUpdate_AlreadyLatestNoOp(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		VersionResolver: &mockVersionResolver{latestMinor: "1.0.0"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !result.NoOp {
		t.Error("expected no-op when already at latest")
	}
}

func TestPackUpdate_TamperDetectedWithoutAcknowledge(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		Acknowledge:     false,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "tamper-pack-rule-removed")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.0.1"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error for tamper detection without --acknowledge")
	}

	if !strings.Contains(err.Error(), "tamper") {
		t.Errorf("error should mention tamper, got: %v", err)
	}
}

func TestPackUpdate_NoVersionResolver(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir: projectDir,
		// No VersionResolver provided.
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error when no version resolver provided")
	}
}

func TestPackUpdate_PackNotFound(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs: []\n")

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		VersionResolver: &mockVersionResolver{latestMinor: "1.0.0"},
	}

	_, err := distribution.Update("acme/nonexistent", opts)
	if err == nil {
		t.Fatal("expected error for nonexistent pack")
	}
}

func TestPackUpdate_NonTamperChangesAccepted(t *testing.T) {
	projectDir := setupUpdateProject(t)

	// valid-pack-v2 adds a new rule and updates descriptions — non-tamper changes.
	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update with non-tamper changes should proceed: %v", err)
	}

	if result.NoOp {
		t.Error("expected update to proceed with non-tamper changes")
	}
	if result.NewVersion != "1.1.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "1.1.0")
	}
}

func TestPackUpdate_VersionResolverError(t *testing.T) {
	projectDir := setupUpdateProject(t)

	resolver := &mockVersionResolverWithError{
		err: fmt.Errorf("resolution failed"),
	}

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		VersionResolver: resolver,
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error when version resolver fails")
	}

	if !strings.Contains(err.Error(), "resolving version") {
		t.Errorf("error should mention resolving version, got: %v", err)
	}
}

func TestPackUpdate_GitCloneFailure(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{failWith: &distribution.GitError{Message: "clone failed"}},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error for clone failure")
	}

	// Verify old version retained.
	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if !strings.Contains(string(data), "1.0.0") {
		t.Error("should retain old version on clone failure")
	}
}

func TestPackUpdate_RunPackTestFailure(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{testFail: true},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error when pack test fails")
	}
}

func TestPackUpdate_NilValidatorSkipsValidation(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       nil,
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update with nil validator should succeed: %v", err)
	}

	if result.NewVersion != "1.1.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "1.1.0")
	}
}

func TestPackUpdate_NoExistingPackDir(t *testing.T) {
	projectDir := setupUpdateProject(t)
	// Remove the installed pack directory so tamper detection is skipped.
	os.RemoveAll(filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack"))

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update should succeed without existing pack dir: %v", err)
	}

	if result.NewVersion != "1.1.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "1.1.0")
	}
}

func TestPackUpdate_LockfileCreatedWhenMissing(t *testing.T) {
	projectDir := setupUpdateProject(t)
	// Delete lockfile.
	os.Remove(filepath.Join(projectDir, "backstop.lock"))

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update should succeed and create lockfile: %v", err)
	}

	// Verify lockfile was created.
	lf, readErr := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if readErr != nil {
		t.Fatalf("lockfile should exist after update: %v", readErr)
	}

	entry := lf.Packs["acme/valid-pack"]
	if entry.Version != "1.1.0" {
		t.Errorf("lockfile version = %q, want %q", entry.Version, "1.1.0")
	}
}

func TestPackUpdate_ResultFields(t *testing.T) {
	projectDir := setupUpdateProject(t)

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	result, err := distribution.Update("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if result.OldVersion != "1.0.0" {
		t.Errorf("OldVersion = %q, want %q", result.OldVersion, "1.0.0")
	}
	if result.ContentHash == "" {
		t.Error("expected non-empty ContentHash")
	}
	if result.NewVersion != "1.1.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "1.1.0")
	}
}

func TestPackUpdate_BackstopYmlMalformed(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs: [invalid: yaml: {{{")

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		VersionResolver: &mockVersionResolver{latestMinor: "1.0.0"},
	}

	_, err := distribution.Update("acme/pack", opts)
	if err == nil {
		t.Fatal("expected error for malformed backstop.yml")
	}
}

func TestPackUpdate_TamperDetectionError(t *testing.T) {
	projectDir := setupUpdateProject(t)

	// Corrupt the installed pack's pack.yml so DetectTamper fails.
	packDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	writeFile(t, filepath.Join(packDir, "pack.yml"), "{{{invalid yaml")

	opts := distribution.UpdateOptions{
		ProjectDir:      projectDir,
		GitCloner:       &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		Validator:       &mockValidator{},
		VersionResolver: &mockVersionResolver{latestMinor: "1.1.0"},
	}

	_, err := distribution.Update("acme/valid-pack", opts)
	if err == nil {
		t.Fatal("expected error for tamper detection failure")
	}

	if !strings.Contains(err.Error(), "tamper detection") {
		t.Errorf("error should mention tamper detection, got: %v", err)
	}
}

// mockVersionResolverWithError returns an error from ResolveLatestCompatible.
type mockVersionResolverWithError struct {
	err error
}

func (m *mockVersionResolverWithError) ResolveLatestCompatible(_, _ string) (string, error) {
	return "", m.err
}

func (m *mockVersionResolverWithError) IsMajorBump(_, _ string) bool {
	return false
}
