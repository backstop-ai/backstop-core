package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPackCompile_DiscoversStandardFiles verifies discoverStandards finds
// .standard.md files recursively in configured directories. (CLM-001)
func TestPackCompile_DiscoversStandardFiles(t *testing.T) {
	dir := t.TempDir()

	// Create standards at root and nested levels
	writeFixture(t, filepath.Join(dir, "alpha.standard.md"), "")
	writeFixture(t, filepath.Join(dir, "subdir", "beta.standard.md"), "")
	writeFixture(t, filepath.Join(dir, "subdir", "deep", "gamma.standard.md"), "")

	paths, err := discoverStandards([]string{dir})
	if err != nil {
		t.Fatalf("discoverStandards error: %v", err)
	}

	if len(paths) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(paths), paths)
	}

	// Verify all paths are absolute
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			t.Errorf("expected absolute path, got %q", p)
		}
	}
}

// TestPackCompile_IgnoresNonStandardFiles verifies discoverStandards ignores
// files not matching *.standard.md. (CLM-002)
func TestPackCompile_IgnoresNonStandardFiles(t *testing.T) {
	dir := t.TempDir()

	writeFixture(t, filepath.Join(dir, "valid.standard.md"), "")
	writeFixture(t, filepath.Join(dir, "readme.md"), "")
	writeFixture(t, filepath.Join(dir, "config.yml"), "")
	writeFixture(t, filepath.Join(dir, "notes.txt"), "")
	writeFixture(t, filepath.Join(dir, "standard.md"), "") // missing prefix

	paths, err := discoverStandards([]string{dir})
	if err != nil {
		t.Fatalf("discoverStandards error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(paths), paths)
	}

	if filepath.Base(paths[0]) != "valid.standard.md" {
		t.Errorf("expected valid.standard.md, got %q", paths[0])
	}
}

// TestPackCompile_DefaultStandardsDir verifies the command uses "standards/"
// as the default directory when backstop.yml does not specify standards_dirs. (CLM-003)
func TestPackCompile_DefaultStandardsDir(t *testing.T) {
	// This tests the defaultStandardsDirs function which provides the
	// default when config has no StandardsDirs set.
	dirs := defaultStandardsDirs(nil)
	if len(dirs) != 1 || dirs[0] != "standards/" {
		t.Errorf("expected [standards/], got %v", dirs)
	}

	// Empty slice should also default
	dirs = defaultStandardsDirs([]string{})
	if len(dirs) != 1 || dirs[0] != "standards/" {
		t.Errorf("expected [standards/], got %v", dirs)
	}

	// Non-empty should pass through
	custom := []string{"custom/", "other/"}
	dirs = defaultStandardsDirs(custom)
	if len(dirs) != 2 || dirs[0] != "custom/" || dirs[1] != "other/" {
		t.Errorf("expected custom dirs, got %v", dirs)
	}
}

// TestPackCompile_DeterministicOrder verifies standards are returned in
// sorted file path order across multiple calls. (CLM-026)
func TestPackCompile_DeterministicOrder(t *testing.T) {
	dir := t.TempDir()

	// Create files with names that would sort differently than creation order
	writeFixture(t, filepath.Join(dir, "z-last.standard.md"), "")
	writeFixture(t, filepath.Join(dir, "a-first.standard.md"), "")
	writeFixture(t, filepath.Join(dir, "m-middle.standard.md"), "")

	// Run multiple times and verify same order
	for i := 0; i < 5; i++ {
		paths, err := discoverStandards([]string{dir})
		if err != nil {
			t.Fatalf("run %d: discoverStandards error: %v", i, err)
		}

		if len(paths) != 3 {
			t.Fatalf("run %d: expected 3 files, got %d", i, len(paths))
		}

		if filepath.Base(paths[0]) != "a-first.standard.md" {
			t.Errorf("run %d: expected a-first.standard.md first, got %q", i, filepath.Base(paths[0]))
		}
		if filepath.Base(paths[1]) != "m-middle.standard.md" {
			t.Errorf("run %d: expected m-middle.standard.md second, got %q", i, filepath.Base(paths[1]))
		}
		if filepath.Base(paths[2]) != "z-last.standard.md" {
			t.Errorf("run %d: expected z-last.standard.md third, got %q", i, filepath.Base(paths[2]))
		}
	}
}

// writeFixture creates a file at path with content, creating parent dirs as needed.
func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
