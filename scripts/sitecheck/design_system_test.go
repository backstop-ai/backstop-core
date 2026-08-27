package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesignSystem_CopyTreeAndManifestHelpers(t *testing.T) {
	source := t.TempDir()
	writeTestFile(t, filepath.Join(source, "nested", "asset.css"), "token bytes\n")
	destination := filepath.Join(t.TempDir(), "copy")
	if err := copyTree(source, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "nested", "asset.css"))
	if err != nil || string(data) != "token bytes\n" {
		t.Fatalf("copied bytes=%q err=%v", data, err)
	}
	if err := copyTree(filepath.Join(source, "missing"), filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing copy source passed")
	}
	blockedDestination := filepath.Join(t.TempDir(), "blocked")
	writeTestFile(t, blockedDestination, "not a directory")
	if err := copyTree(source, blockedDestination); err == nil {
		t.Fatal("blocked copy destination passed")
	}
	danglingSource := t.TempDir()
	if err := os.Symlink(filepath.Join(danglingSource, "absent"), filepath.Join(danglingSource, "dangling")); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(danglingSource, filepath.Join(t.TempDir(), "dangling-copy")); err == nil {
		t.Fatal("dangling copy source passed")
	}

	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	isolated := t.TempDir()
	if err := writeIsolatedManifest(isolated, repositoryRoot, "unit"); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(isolated, "backstop.yml"))
	if err != nil || !strings.Contains(string(manifest), "seed4-design-system-unit") {
		t.Fatalf("manifest=%q err=%v", manifest, err)
	}
	if _, err := os.Stat(filepath.Join(isolated, "backstop.lock")); err != nil {
		t.Fatal(err)
	}
	if err := writeIsolatedManifest(t.TempDir(), t.TempDir(), "missing-lock"); err == nil {
		t.Fatal("missing source lock passed")
	}
	withoutPack := t.TempDir()
	writeTestFile(t, filepath.Join(withoutPack, "backstop.lock"), "packs: {}\n")
	if err := writeIsolatedManifest(t.TempDir(), withoutPack, "missing-pack"); err == nil || !strings.Contains(err.Error(), "lock entry missing") {
		t.Fatalf("missing pack error=%v", err)
	}
	invalidLock := t.TempDir()
	writeTestFile(t, filepath.Join(invalidLock, "backstop.lock"), "packs: [\n")
	if err := writeIsolatedManifest(t.TempDir(), invalidLock, "invalid-lock"); err == nil {
		t.Fatal("invalid source lock passed")
	}
	blockedRoot := filepath.Join(t.TempDir(), "blocked-root")
	writeTestFile(t, blockedRoot, "not a directory")
	if err := writeIsolatedManifest(blockedRoot, repositoryRoot, "blocked-root"); err == nil {
		t.Fatal("blocked manifest root passed")
	}
}

func TestDesignSystem_IsolatedCorpusRejectsMissingInputs(t *testing.T) {
	root := t.TempDir()
	blockedMatrix := t.TempDir()
	writeTestFile(t, filepath.Join(blockedMatrix, "blocked"), "not a directory")
	if err := runIsolatedCorpus(root, t.TempDir(), blockedMatrix, "blocked", nil); err == nil {
		t.Fatal("blocked matrix root passed")
	}
	if err := runIsolatedCorpus(root, filepath.Join(root, "missing-site"), t.TempDir(), "missing", nil); err == nil {
		t.Fatal("missing built corpus passed")
	}
	if findings := VerifyEightIsolatedCorpora(root, t.TempDir(), OwnerAcceptanceExport{}); len(findings) != 1 || findings[0].Identity != "clean-plus-seven" {
		t.Fatalf("empty corpus findings=%#v", findings)
	}
}
