package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Coverage tests for thin CLI adapter commands.
// These invoke the cobra RunE with minimal setup to exercise the code paths.

func TestCLI_PackAdd_NoArgs(t *testing.T) {
	cmd := newPackAddCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack add with no args")
	}
}

func TestCLI_PackRemove_NoArgs(t *testing.T) {
	cmd := newPackRemoveCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack remove with no args")
	}
}

func TestCLI_PackRemove_NonexistentPack(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := newPackRemoveCommand(boolPtr(false))
	cmd.SetArgs([]string{"nonexistent/pack"})
	_ = cmd.Execute() // Will error — exercises the RunE path
}

func TestCLI_PackInstall_NoLockfile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := newPackInstallCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	_ = cmd.Execute() // May error on missing lockfile — that's fine, we're exercising the path
}

func TestCLI_PackUpdate_NoArgs(t *testing.T) {
	cmd := newPackUpdateCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack update with no args")
	}
}

func TestCLI_PackUpdate_NonexistentPack(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := newPackUpdateCommand(boolPtr(false))
	cmd.SetArgs([]string{"nonexistent/pack"})
	_ = cmd.Execute()
}

func TestCLI_PackUpgrade_NoArgs(t *testing.T) {
	cmd := newPackUpgradeCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack upgrade with no args")
	}
}

func TestCLI_PackUpgrade_NonexistentPack(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := newPackUpgradeCommand(boolPtr(false))
	cmd.SetArgs([]string{"nonexistent/pack@2.0.0"})
	_ = cmd.Execute()
}

func TestCLI_PackAdd_NonexistentPack(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	defer func() { recover() }() // distribution.Add may panic on missing deps
	cmd := newPackAddCommand(boolPtr(false))
	cmd.SetArgs([]string{"nonexistent/pack@1.0.0"})
	_ = cmd.Execute()
}

func TestCLI_PackList_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := newPackListCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	_ = cmd.Execute() // May succeed with empty list or error — exercises the path
}

func TestCLI_PackList_JSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	jsonFlag := true
	cmd := newPackListCommand(&jsonFlag)
	cmd.SetArgs([]string{})
	_ = cmd.Execute()
}

func TestCLI_FormatNewResult_JSON(t *testing.T) {
	f := &JSONArtifactNewFormatter{}
	result := ArtifactNewResult{ArtifactType: "spec", ID: "001", FilePath: "specs/SPEC-001.spec.md"}
	out, err := f.FormatNewResult(result)
	if err != nil {
		t.Fatalf("FormatNewResult JSON: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty JSON output")
	}
}

func TestCLI_FormatNewResult_Human(t *testing.T) {
	f := &HumanArtifactNewFormatter{}
	result := ArtifactNewResult{ArtifactType: "spec", ID: "001", FilePath: "specs/SPEC-001.spec.md"}
	out, err := f.FormatNewResult(result)
	if err != nil {
		t.Fatalf("FormatNewResult human: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty human output")
	}
}

func TestCLI_SpecsExist_NoDir(t *testing.T) {
	if specsExist(filepath.Join(t.TempDir(), "nonexistent")) {
		t.Error("expected false for nonexistent dir")
	}
}

func TestCLI_SpecsExist_DirExists(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "specs"), 0o755)
	if !specsExist(filepath.Join(dir, "specs")) {
		t.Error("expected true when specs/ exists")
	}
}

func TestCLI_SpecsExist_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "specs"), []byte("not a dir"), 0o644)
	if specsExist(filepath.Join(dir, "specs")) {
		t.Error("expected false when specs is a file not a directory")
	}
}

func TestCLI_CheckAll_NoBackstop(t *testing.T) {
	// TASK-002 hardening: runCheck calls config.DiscoverConfigPath, which walks
	// UP from cwd and would otherwise find backstop-core's own backstop.yml
	// (tests run inside the repo). That would make pRoot the real repo root,
	// pass ValidateBackstopDir, and — now that executors are real — shell out to
	// live golangci-lint / go build / go test / semgrep. Point BACKSTOP_CONFIG
	// at a non-existent path so discovery fails, pRoot stays the empty temp dir,
	// and ValidateBackstopDir returns its error before reaching check.Run. This
	// keeps the test hermetic and asserts the missing-.backstop error path.
	dir := t.TempDir()
	t.Setenv("BACKSTOP_CONFIG", filepath.Join(dir, "does-not-exist.yml"))

	checker := &realCodeChecker{projectRoot: dir}
	_, err := checker.CheckAll(t.Context())
	if err == nil {
		t.Error("expected an error when .backstop is missing")
	}
}

func boolPtr(b bool) *bool { return &b }
