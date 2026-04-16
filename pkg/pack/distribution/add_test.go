package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// TestGitError_Message covers GitError.Error() method.
func TestGitError_Message(t *testing.T) {
	err := &distribution.GitError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test error")
	}
}

// TestValidationError_Message covers ValidationError.Error() method.
func TestValidationError_Message(t *testing.T) {
	err := &distribution.ValidationError{Message: "validation failed"}
	if err.Error() != "validation failed" {
		t.Errorf("Error() = %q, want %q", err.Error(), "validation failed")
	}
}

// --- Test helpers for add tests ---

func setupAddProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create backstop.yml with empty packs.
	writeFile(t, filepath.Join(dir, "backstop.yml"), "packs: []\n")

	return dir
}

func newTestAddOptions(projectDir string) distribution.AddOptions {
	return distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
		GitCloner: &mockGitCloner{
			cloneDir: filepath.Join("testdata", "valid-pack"),
		},
		Validator: &mockValidator{},
	}
}

// --- mock types ---

type mockGitCloner struct {
	cloneDir string
	failWith error
}

func (m *mockGitCloner) Clone(url, version, destDir string) error {
	if m.failWith != nil {
		return m.failWith
	}
	return copyDir(m.cloneDir, destDir)
}

func (m *mockGitCloner) ListTags(_ string) ([]string, error) {
	return []string{"v1.0.0", "v1.1.0", "v2.0.0"}, nil
}

type mockValidator struct {
	checkFail bool
	testFail  bool
}

func (m *mockValidator) RunPackCheck(_ string) error {
	if m.checkFail {
		return &distribution.ValidationError{Message: "pack check failed"}
	}
	return nil
}

func (m *mockValidator) RunPackTest(_ string) error {
	if m.testFail {
		return &distribution.ValidationError{Message: "pack test failed"}
	}
	return nil
}

// --- Tests ---

func TestPackAdd_ResolvesGitURL(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	result, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.PackName != "acme/valid-pack" {
		t.Errorf("PackName = %q, want %q", result.PackName, "acme/valid-pack")
	}
}

func TestPackAdd_ClonesAtVersionTag(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	result, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", result.Version, "1.0.0")
	}
}

func TestPackAdd_MissingTagExitsNonZero(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)
	opts.GitCloner = &mockGitCloner{
		failWith: &distribution.GitError{Message: "tag not found: v9.9.9"},
	}

	_, err := distribution.Add("acme/valid-pack@9.9.9", opts)
	if err == nil {
		t.Fatal("expected error for missing tag")
	}
}

func TestPackAdd_CloneFailureExitsNonZero(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)
	opts.GitCloner = &mockGitCloner{
		failWith: &distribution.GitError{Message: "network error"},
	}

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error for clone failure")
	}
}

func TestPackAdd_RunsPackCheckBeforeInstall(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	// Success case — pack check passes.
	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add should succeed when pack check passes: %v", err)
	}
}

func TestPackAdd_RunsPackTestBeforeInstall(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add should succeed when pack test passes: %v", err)
	}
}

func TestPackAdd_PackCheckFailureAbortsInstall(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)
	opts.Validator = &mockValidator{checkFail: true}

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error when pack check fails")
	}

	// Verify pack was NOT installed.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("pack should not be installed when check fails")
	}
}

func TestPackAdd_PackTestFailureAbortsInstall(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)
	opts.Validator = &mockValidator{testFail: true}

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error when pack test fails")
	}
}

func TestPackAdd_CopiesToPacksDir(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	result, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.InstalledPath == "" {
		t.Error("expected non-empty InstalledPath")
	}

	// Verify pack was copied.
	packYml := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack", "pack.yml")
	if _, err := os.Stat(packYml); err != nil {
		t.Errorf("pack.yml not found at installed path: %v", err)
	}
}

func TestPackAdd_ComputesContentHash(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	result, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.ContentHash == "" {
		t.Error("expected non-empty ContentHash")
	}
}

func TestPackAdd_UpdatesBackstopYml(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, readErr := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if readErr != nil {
		t.Fatalf("reading backstop.yml: %v", readErr)
	}

	if !strings.Contains(string(data), "acme/valid-pack") {
		t.Error("backstop.yml should contain the added pack")
	}
}

func TestPackAdd_UpdatesBackstopLock(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, readErr := distribution.ReadLockfile(lockPath)
	if readErr != nil {
		t.Fatalf("reading lockfile: %v", readErr)
	}

	if _, ok := lf.Packs["acme/valid-pack"]; !ok {
		t.Error("lockfile should contain the added pack")
	}
}

func TestPackAdd_RollbackOnPostCloneFailure(t *testing.T) {
	projectDir := setupAddProject(t)

	// Snapshot backstop.yml before add.
	ymlPath := filepath.Join(projectDir, "backstop.yml")
	ymlBefore, _ := os.ReadFile(ymlPath)

	opts := newTestAddOptions(projectDir)
	// Validator passes check but fails test — post-clone failure.
	opts.Validator = &mockValidator{testFail: true}

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error")
	}

	// Verify rollback: no pack in .backstop/packs/.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("pack should be rolled back after failure")
	}

	// Verify rollback: backstop.yml unchanged.
	ymlAfter, _ := os.ReadFile(ymlPath)
	if string(ymlBefore) != string(ymlAfter) {
		t.Error("backstop.yml should be rolled back after failure")
	}
}

func TestPackAdd_RecordsProvenance(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	prov, readErr := distribution.ReadProvenance(provPath)
	if readErr != nil {
		t.Fatalf("reading provenance: %v", readErr)
	}

	if len(prov.Entries) == 0 {
		t.Error("expected provenance entries after pack add with tool_config")
	}
}

func TestPackAdd_ProvenanceContainsAllFields(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	prov, _ := distribution.ReadProvenance(provPath)

	if len(prov.Entries) == 0 {
		t.Fatal("expected provenance entries")
	}

	entry := prov.Entries[0]
	if entry.ConfigFile == "" {
		t.Error("provenance missing ConfigFile")
	}
	if entry.SettingKey == "" {
		t.Error("provenance missing SettingKey")
	}
	if entry.SourcePack == "" {
		t.Error("provenance missing SourcePack")
	}
	if entry.ValueHash == "" {
		t.Error("provenance missing ValueHash")
	}
}

func TestPackAdd_CreatesGitignore(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, readErr := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if readErr != nil {
		t.Fatalf("reading .gitignore: %v", readErr)
	}

	if !strings.Contains(string(data), ".backstop/packs/") {
		t.Error(".gitignore should contain .backstop/packs/")
	}
}

func TestPackAdd_AppendsToGitignore(t *testing.T) {
	projectDir := setupAddProject(t)
	writeFile(t, filepath.Join(projectDir, ".gitignore"), "node_modules/\n")

	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	content := string(data)

	if !strings.Contains(content, "node_modules/") {
		t.Error("existing .gitignore content should be preserved")
	}
	if !strings.Contains(content, ".backstop/packs/") {
		t.Error(".gitignore should contain .backstop/packs/")
	}
}

func TestPackAdd_GitignoreAlreadyPresent(t *testing.T) {
	projectDir := setupAddProject(t)
	writeFile(t, filepath.Join(projectDir, ".gitignore"), ".backstop/packs/\n")

	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	count := strings.Count(string(data), ".backstop/packs/")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of .backstop/packs/ in .gitignore, got %d", count)
	}
}

func TestPackAdd_NoTransitiveDependencies(t *testing.T) {
	projectDir := setupAddProject(t)

	// Create a pack that declares dependencies on other packs.
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "pack.yml"),
		"name: acme/with-deps\nversion: \"1.0.0\"\ndependencies:\n  - acme/other-pack\n")

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
		GitCloner:  &mockGitCloner{cloneDir: packDir},
		Validator:  &mockValidator{},
	}

	result, err := distribution.Add("acme/with-deps@1.0.0", opts)

	// Should succeed but not install transitive dependencies.
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.PackName != "acme/with-deps" {
		t.Errorf("PackName = %q, want %q", result.PackName, "acme/with-deps")
	}

	// Verify no other pack was installed.
	otherPack := filepath.Join(projectDir, ".backstop", "packs", "acme", "other-pack")
	if _, err := os.Stat(otherPack); !os.IsNotExist(err) {
		t.Error("transitive dependency should not be installed")
	}
}

func TestBackstopYml_ExactVersionPins(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	content := string(data)

	// Should contain exact version, not a range.
	if strings.Contains(content, "^") || strings.Contains(content, "~") {
		t.Error("backstop.yml should use exact version pins, not ranges")
	}
	if !strings.Contains(content, "1.0.0") {
		t.Error("backstop.yml should contain exact version 1.0.0")
	}
}

func TestPackAdd_SkipsSDKDependencies(t *testing.T) {
	projectDir := setupAddProject(t)

	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "pack.yml"),
		"name: acme/sdk-pack\nversion: \"1.0.0\"\nsdk_dependencies:\n  - go:1.21\n")

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
		GitCloner:  &mockGitCloner{cloneDir: packDir},
		Validator:  &mockValidator{},
	}

	result, err := distribution.Add("acme/sdk-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Verify pack was installed (SDK deps skipped silently).
	if result.PackName != "acme/sdk-pack" {
		t.Errorf("PackName = %q, want %q", result.PackName, "acme/sdk-pack")
	}
}

func TestPackAdd_AlreadyInstalledExitsNonZero(t *testing.T) {
	projectDir := setupAddProject(t)

	// Write backstop.yml with the pack already listed.
	writeFile(t, filepath.Join(projectDir, "backstop.yml"),
		"packs:\n  - name: acme/valid-pack\n    version: \"1.0.0\"\n")

	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error for already-installed pack")
	}

	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("error should mention 'already installed', got: %v", err)
	}
}

func TestPackAdd_LocalPathSkipsGit(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	result, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	if result.PackName != "internal/local-rules" {
		t.Errorf("PackName = %q, want %q", result.PackName, "internal/local-rules")
	}
}

func TestPackAdd_LocalPathValidatesInPlace(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}
}

func TestPackAdd_LocalPathRegistersWithPathEntry(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	content := string(data)

	if !strings.Contains(content, "path:") {
		t.Error("backstop.yml should contain path: entry for local pack")
	}
}

func TestPackAdd_LocalPathComputesHash(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	result, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	if result.ContentHash == "" {
		t.Error("expected content hash for local pack")
	}
}

func TestPackAdd_LocalPathNotClonedToPacksDir(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	// Local pack should NOT be in .backstop/packs/.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "internal", "local-rules")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("local pack should not be cloned to .backstop/packs/")
	}
}

func TestLocalPack_ValidatedSameAsGit(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath, _ := filepath.Abs(localPackDir)

	// Validator that fails check — should abort for local packs too.
	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{checkFail: true},
	}

	_, err := distribution.Add(absPath, opts)
	if err == nil {
		t.Fatal("expected error when pack check fails for local pack")
	}
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(target, data, info.Mode())
	})
}
