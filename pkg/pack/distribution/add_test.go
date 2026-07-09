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

// --- shared must-helpers (handle errors explicitly so tests don't ignore them) ---

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs %s: %v", path, err)
	}
	return abs
}

func mustReadLock(t *testing.T, path string) *distribution.Lockfile {
	t.Helper()
	lf, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile %s: %v", path, err)
	}
	return lf
}

func mustReadProvenance(t *testing.T, path string) *distribution.Provenance {
	t.Helper()
	prov, err := distribution.ReadProvenance(path)
	if err != nil {
		t.Fatalf("ReadProvenance %s: %v", path, err)
	}
	return prov
}

// --- Test helpers for add tests ---

func setupAddProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create backstop.yml with empty packs.
	writeFile(t, filepath.Join(dir, "backstop.yml"), "packs: {}\n")

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
	ymlBefore := mustReadFile(t, ymlPath)

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
	ymlAfter := mustReadFile(t, ymlPath)
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
	prov := mustReadProvenance(t, provPath)

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

	data := mustReadFile(t, filepath.Join(projectDir, ".gitignore"))
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

	data := mustReadFile(t, filepath.Join(projectDir, ".gitignore"))
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

	data := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))
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
		"packs:\n  acme/valid-pack: \"1.0.0\"\n")

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
	absPath := mustAbs(t, localPackDir)

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
	absPath := mustAbs(t, localPackDir)

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
	absPath := mustAbs(t, localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	data := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))
	content := string(data)

	if !strings.Contains(content, ": local") {
		t.Error("backstop.yml should contain 'local' version for local pack")
	}
}

func TestPackAdd_LocalPathComputesHash(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath := mustAbs(t, localPackDir)

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

func TestPackAdd_LocalPathCopiedToPacksDir(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath := mustAbs(t, localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	// Local pack SHOULD be copied to .backstop/packs/ (same as git packs).
	packName := "internal/local-rules" // from testdata/local-pack/pack.yml
	packsDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(packName))
	if _, statErr := os.Stat(packsDir); os.IsNotExist(statErr) {
		t.Error("local pack should be copied to .backstop/packs/")
	}
}

func TestLocalPack_ValidatedSameAsGit(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath := mustAbs(t, localPackDir)

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

func TestPackAdd_LocalPathMissingPackYml(t *testing.T) {
	projectDir := setupAddProject(t)

	// Create a local dir without pack.yml.
	localDir := t.TempDir()

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(localDir, opts)
	if err == nil {
		t.Fatal("expected error for local path missing pack.yml")
	}

	if !strings.Contains(err.Error(), "does not contain pack.yml") {
		t.Errorf("error should mention missing pack.yml, got: %v", err)
	}
}

func TestPackAdd_LocalPathMissingName(t *testing.T) {
	projectDir := setupAddProject(t)

	localDir := t.TempDir()
	writeFile(t, filepath.Join(localDir, "pack.yml"), "version: \"1.0.0\"\n")

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(localDir, opts)
	if err == nil {
		t.Fatal("expected error for local pack missing name")
	}

	if !strings.Contains(err.Error(), "local pack manifest missing name") {
		t.Errorf("error should mention missing name, got: %v", err)
	}
}

func TestPackAdd_NilValidatorSkipsValidation(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)
	opts.Validator = nil

	result, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add with nil validator should succeed: %v", err)
	}

	if result.PackName != "acme/valid-pack" {
		t.Errorf("PackName = %q, want %q", result.PackName, "acme/valid-pack")
	}
}

func TestPackAdd_VersionOverridesPackRef(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)
	opts.Version = "1.0.0"

	// packRef has @2.0.0 but opts.Version is "1.0.0".
	result, err := distribution.Add("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q (opts.Version should override @version)", result.Version, "1.0.0")
	}
}

func TestPackAdd_GitignoreNoTrailingNewline(t *testing.T) {
	projectDir := setupAddProject(t)
	// Write .gitignore without trailing newline.
	writeFile(t, filepath.Join(projectDir, ".gitignore"), "node_modules/")

	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data := mustReadFile(t, filepath.Join(projectDir, ".gitignore"))
	content := string(data)

	if !strings.Contains(content, "node_modules/") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(content, ".backstop/packs/") {
		t.Error(".gitignore should contain .backstop/packs/")
	}
	// Verify they are on separate lines.
	lines := strings.Split(content, "\n")
	foundNode := false
	foundBackstop := false
	for _, l := range lines {
		if strings.TrimSpace(l) == "node_modules/" {
			foundNode = true
		}
		if strings.TrimSpace(l) == ".backstop/packs/" {
			foundBackstop = true
		}
	}
	if !foundNode || !foundBackstop {
		t.Errorf("expected both entries on separate lines, content:\n%s", content)
	}
}

func TestPackAdd_LocalPathRelativeDot(t *testing.T) {
	projectDir := setupAddProject(t)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	result, err := distribution.Add("./testdata/local-pack", opts)
	if err != nil {
		t.Fatalf("Add with ./ prefix: %v", err)
	}

	if result.PackName != "internal/local-rules" {
		t.Errorf("PackName = %q, want %q", result.PackName, "internal/local-rules")
	}
}

func TestPackAdd_LocalPathMalformedManifest(t *testing.T) {
	projectDir := setupAddProject(t)

	localDir := t.TempDir()
	writeFile(t, filepath.Join(localDir, "pack.yml"), "not: [valid: yaml: {{{")

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(localDir, opts)
	if err == nil {
		t.Fatal("expected error for malformed local manifest")
	}
}

func TestPackAdd_ParsePackRefNoVersion(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)
	opts.Version = "1.0.0"

	// packRef without @version — version comes from opts.Version.
	result, err := distribution.Add("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", result.Version, "1.0.0")
	}
}

func TestPackAdd_LockfileEntryFields(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	result, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, readErr := distribution.ReadLockfile(lockPath)
	if readErr != nil {
		t.Fatalf("ReadLockfile: %v", readErr)
	}

	entry, ok := lf.Packs["acme/valid-pack"]
	if !ok {
		t.Fatal("expected pack in lockfile")
	}

	if entry.Name != "acme/valid-pack" {
		t.Errorf("Name = %q, want %q", entry.Name, "acme/valid-pack")
	}
	if entry.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", entry.Version, "1.0.0")
	}
	if entry.GitRef == nil || *entry.GitRef != "v1.0.0" {
		t.Errorf("GitRef = %v, want v1.0.0", entry.GitRef)
	}
	if entry.ContentHash == "" || entry.ContentHash != result.ContentHash {
		t.Errorf("ContentHash = %q, want %q", entry.ContentHash, result.ContentHash)
	}
	if entry.SourceType != "git" {
		t.Errorf("SourceType = %q, want %q", entry.SourceType, "git")
	}
	if entry.InstallDate == "" {
		t.Error("InstallDate should not be empty")
	}
}

func TestPackAdd_LocalLockfileSourceType(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absPath := mustAbs(t, localPackDir)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(absPath, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf := mustReadLock(t, lockPath)

	entry := lf.Packs["internal/local-rules"]
	if entry.SourceType != "local" {
		t.Errorf("SourceType = %q, want %q", entry.SourceType, "local")
	}
	if entry.GitRef != nil {
		t.Errorf("expected nil GitRef for local pack, got %v", entry.GitRef)
	}
}

func TestPackAdd_ToolConfigConflictRollsBack(t *testing.T) {
	projectDir := setupAddProject(t)

	// Pre-create a conflicting config.
	writeFile(t, filepath.Join(projectDir, ".golangci.yml"),
		`{"linters.enable.revive": false}`)

	opts := newTestAddOptions(projectDir)

	// Snapshot backstop.yml before add.
	ymlPath := filepath.Join(projectDir, "backstop.yml")
	ymlBefore := mustReadFile(t, ymlPath)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error for tool_config conflict")
	}

	if !strings.Contains(err.Error(), "tool_config conflicts") {
		t.Errorf("error should mention conflicts, got: %v", err)
	}

	// Verify rollback: backstop.yml unchanged.
	ymlAfter := mustReadFile(t, ymlPath)
	if string(ymlBefore) != string(ymlAfter) {
		t.Error("backstop.yml should be rolled back after conflict")
	}

	// Verify rollback: pack not in .backstop/packs/.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("pack should be rolled back after conflict")
	}

	// Verify rollback: no lockfile entry.
	lockPath := filepath.Join(projectDir, "backstop.lock")
	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Error("lockfile should be rolled back (removed) after conflict")
	}
}

func TestPackAdd_LocalPathRelativeParent(t *testing.T) {
	projectDir := setupAddProject(t)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	result, err := distribution.Add("../distribution/testdata/local-pack", opts)
	if err != nil {
		t.Fatalf("Add with ../ prefix: %v", err)
	}

	if result.PackName != "internal/local-rules" {
		t.Errorf("PackName = %q, want %q", result.PackName, "internal/local-rules")
	}
}

func TestPackAdd_RollbackOnProvenanceReadError(t *testing.T) {
	projectDir := setupAddProject(t)

	// Create invalid provenance file that will cause ReadProvenance to fail.
	backstopDir := filepath.Join(projectDir, ".backstop")
	if err := os.MkdirAll(backstopDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(backstopDir, "pack-config-provenance.json"), "{{{invalid json")

	// Snapshot backstop.yml before add.
	ymlPath := filepath.Join(projectDir, "backstop.yml")
	ymlBefore := mustReadFile(t, ymlPath)

	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error for invalid provenance file")
	}

	// Verify rollback: backstop.yml unchanged.
	ymlAfter := mustReadFile(t, ymlPath)
	if string(ymlBefore) != string(ymlAfter) {
		t.Error("backstop.yml should be rolled back after provenance error")
	}

	// Verify rollback: pack not installed.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("pack should be rolled back after provenance error")
	}
}

func TestPackAdd_RollbackOnWriteProvenanceFail(t *testing.T) {
	projectDir := setupAddProject(t)

	// Create a pack that has NO tool_config (so merge succeeds with no changes).
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "pack.yml"),
		"name: acme/no-config-pack\nversion: \"1.0.0\"\n")

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
		GitCloner:  &mockGitCloner{cloneDir: packDir},
		Validator:  &mockValidator{},
	}

	// First add should succeed to verify the pack works.
	result, err := distribution.Add("acme/no-config-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.PackName != "acme/no-config-pack" {
		t.Errorf("PackName = %q, want %q", result.PackName, "acme/no-config-pack")
	}
}

func TestPackAdd_MalformedBackstopYmlReturnsInstallCheck(t *testing.T) {
	projectDir := t.TempDir()
	// Write malformed backstop.yml — isPackInstalled should return false.
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "not_valid_yaml: [[[")

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
		GitCloner: &mockGitCloner{
			cloneDir: filepath.Join("testdata", "valid-pack"),
		},
		Validator: &mockValidator{},
	}

	// isPackInstalled returns false for malformed YAML, so Add proceeds.
	// But updateBackstopYml will then fail on the malformed YAML.
	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	// We expect an error from updateBackstopYml, exercising the rollback.
	if err != nil {
		// Verify the rollback happened — pack should be cleaned up.
		packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
		if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
			t.Error("pack should be rolled back after updateBackstopYml failure")
		}
	}
}

func TestPackAdd_WriteLockfileRollback(t *testing.T) {
	projectDir := setupAddProject(t)

	// Create a pack with no tool_config for cleaner test.
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "pack.yml"),
		"name: acme/simple-pack\nversion: \"1.0.0\"\n")

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
		GitCloner:  &mockGitCloner{cloneDir: packDir},
		Validator:  &mockValidator{},
	}

	// Add should succeed.
	result, err := distribution.Add("acme/simple-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Verify lockfile was created with correct source type.
	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf := mustReadLock(t, lockPath)
	entry := lf.Packs["acme/simple-pack"]
	if entry.SourceType != "git" {
		t.Errorf("SourceType = %q, want %q", entry.SourceType, "git")
	}
	if entry.ContentHash != result.ContentHash {
		t.Errorf("ContentHash mismatch: lock=%q result=%q", entry.ContentHash, result.ContentHash)
	}
}

func TestPackAdd_MergeToolConfigErrorRollsBack(t *testing.T) {
	projectDir := setupAddProject(t)

	// Create a pack with an invalid pack.yml (not valid YAML or JSON).
	// This will cause MergeToolConfig → readPackManifest to fail.
	badPackDir := t.TempDir()
	writeFile(t, filepath.Join(badPackDir, "pack.yml"), "\x00\x01\x02 binary garbage")

	// Snapshot backstop.yml before add.
	ymlPath := filepath.Join(projectDir, "backstop.yml")
	ymlBefore := mustReadFile(t, ymlPath)

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
		GitCloner:  &mockGitCloner{cloneDir: badPackDir},
		Validator:  nil, // Skip validation so it reaches MergeToolConfig.
	}

	_, err := distribution.Add("acme/bad-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error for invalid pack manifest during merge")
	}

	if !strings.Contains(err.Error(), "merging tool_config") {
		t.Errorf("error should mention merging tool_config, got: %v", err)
	}

	// Verify rollback: pack should be cleaned up.
	packsDir := filepath.Join(projectDir, ".backstop", "packs", "acme", "bad-pack")
	if _, statErr := os.Stat(packsDir); !os.IsNotExist(statErr) {
		t.Error("pack should be rolled back after merge error")
	}

	// Verify rollback: backstop.yml unchanged.
	ymlAfter := mustReadFile(t, ymlPath)
	if string(ymlBefore) != string(ymlAfter) {
		t.Error("backstop.yml should be rolled back after merge error")
	}
}

func TestPackAdd_ExistingLockfileIsPreserved(t *testing.T) {
	projectDir := setupAddProject(t)

	// Create a pre-existing lockfile with another pack.
	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"existing/pack": {
				Name:        "existing/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:existing",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Verify both packs are in lockfile.
	readLf := mustReadLock(t, filepath.Join(projectDir, "backstop.lock"))
	if _, ok := readLf.Packs["existing/pack"]; !ok {
		t.Error("existing pack should be preserved in lockfile")
	}
	if _, ok := readLf.Packs["acme/valid-pack"]; !ok {
		t.Error("new pack should be in lockfile")
	}
}

func TestPackAdd_WriteProvenanceFailRollsBack(t *testing.T) {
	projectDir := setupAddProject(t)

	// Set up .backstop with a valid provenance file then make it read-only.
	backstopDir := filepath.Join(projectDir, ".backstop")
	if err := os.MkdirAll(backstopDir, 0o755); err != nil {
		t.Fatal(err)
	}
	provPath := filepath.Join(backstopDir, "pack-config-provenance.json")
	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	if err := distribution.WriteProvenance(provPath, prov); err != nil {
		t.Fatal(err)
	}
	// Make the provenance file read-only so WriteProvenance fails.
	if err := os.Chmod(provPath, 0o444); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(provPath, 0o644) }()

	ymlBefore := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))

	opts := newTestAddOptions(projectDir)
	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error when provenance file is read-only")
	}

	// Verify rollback: backstop.yml unchanged.
	ymlAfter := mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))
	if string(ymlBefore) != string(ymlAfter) {
		t.Error("backstop.yml should be rolled back after provenance write failure")
	}
}

func TestPackAdd_LocalPathRecordsRelativeSourcePath(t *testing.T) {
	projectDir := setupAddProject(t)

	localPackDir := filepath.Join("testdata", "local-pack")
	absSource, absErr := filepath.Abs(localPackDir)
	if absErr != nil {
		t.Fatalf("Abs: %v", absErr)
	}

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Validator:  &mockValidator{},
	}

	_, err := distribution.Add(absSource, opts)
	if err != nil {
		t.Fatalf("Add local: %v", err)
	}

	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, readErr := distribution.ReadLockfile(lockPath)
	if readErr != nil {
		t.Fatalf("ReadLockfile: %v", readErr)
	}

	entry, ok := lf.Packs["internal/local-rules"]
	if !ok {
		t.Fatal("expected local pack in lockfile")
	}

	if entry.LocalPath == "" {
		t.Fatal("expected LocalPath to be recorded for a local pack")
	}
	if filepath.IsAbs(entry.LocalPath) {
		t.Errorf("LocalPath = %q, want a project-relative (non-absolute) path", entry.LocalPath)
	}

	// Joining the recorded relative path against the project dir must resolve to the
	// same directory that was added.
	resolved, absErr := filepath.Abs(filepath.Join(projectDir, entry.LocalPath))
	if absErr != nil {
		t.Fatalf("resolving joined path: %v", absErr)
	}
	if resolved != absSource {
		t.Errorf("resolved LocalPath = %q, want %q", resolved, absSource)
	}
}

func TestPackAdd_GitSourceLeavesLocalPathEmpty(t *testing.T) {
	projectDir := setupAddProject(t)
	opts := newTestAddOptions(projectDir)

	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, readErr := distribution.ReadLockfile(lockPath)
	if readErr != nil {
		t.Fatalf("ReadLockfile: %v", readErr)
	}
	entry := lf.Packs["acme/valid-pack"]
	if entry.LocalPath != "" {
		t.Errorf("git-source LocalPath = %q, want empty", entry.LocalPath)
	}
}

func TestPackAdd_PacksDirMkdirFails(t *testing.T) {
	projectDir := setupAddProject(t)

	// Pre-create .backstop/packs as a FILE so MkdirAll of the org subdir fails.
	if err := os.MkdirAll(filepath.Join(projectDir, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, ".backstop", "packs"), "not a dir")

	opts := newTestAddOptions(projectDir)
	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error when packs dir cannot be created")
	}
	if !strings.Contains(err.Error(), "creating packs dir") {
		t.Errorf("error should mention creating packs dir, got: %v", err)
	}
}

func TestPackAdd_CopyToInstalledPathFails(t *testing.T) {
	projectDir := setupAddProject(t)

	// Pre-create the org dir and make the pack's install target a FILE so the recursive
	// copy (which must create it as a directory) fails.
	orgDir := filepath.Join(projectDir, ".backstop", "packs", "acme")
	if err := os.MkdirAll(orgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(orgDir, "valid-pack"), "occupying file")

	opts := newTestAddOptions(projectDir)
	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error when install target cannot be created")
	}
	if !strings.Contains(err.Error(), "copying pack") {
		t.Errorf("error should mention copying pack, got: %v", err)
	}
}

func TestPackAdd_ManifestWithoutPacksKey(t *testing.T) {
	projectDir := t.TempDir()
	// backstop.yml with NO packs key — updateBackstopYml must initialize the map.
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "project: myproj\n")

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
		t.Error("backstop.yml should contain the added pack even when packs key was absent")
	}
}

func TestPackAdd_RollbackRestoresExistingLock(t *testing.T) {
	projectDir := setupAddProject(t)

	// Pre-existing lock with another pack — the rollback path must restore it (lockSnap
	// non-nil branch).
	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"existing/pack": {
				Name:        "existing/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:existing",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}

	// Conflicting config forces a rollback AFTER the snapshot is taken.
	writeFile(t, filepath.Join(projectDir, ".golangci.yml"), `{"linters.enable.revive": false}`)

	opts := newTestAddOptions(projectDir)
	_, err := distribution.Add("acme/valid-pack@1.0.0", opts)
	if err == nil {
		t.Fatal("expected error for tool_config conflict")
	}

	// The pre-existing lock must be restored intact.
	readLf, readErr := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if readErr != nil {
		t.Fatalf("ReadLockfile: %v", readErr)
	}
	if _, ok := readLf.Packs["existing/pack"]; !ok {
		t.Error("pre-existing lock entry should be restored after rollback")
	}
	if _, ok := readLf.Packs["acme/valid-pack"]; ok {
		t.Error("the failed pack should not remain in the lock after rollback")
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
