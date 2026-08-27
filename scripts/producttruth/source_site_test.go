package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductTruth_SourceIncludesReferenceFragments(t *testing.T) {
	root := testRoot(t)
	manifest := testManifest(t)
	if err := VerifySourceIncludes(root, manifest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "docs/reference.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "{% include generated/") != 2 {
		t.Fatal("reference include count")
	}
}

func TestProductTruth_SourceIncludesInstalledPackFragment(t *testing.T) {
	root := testRoot(t)
	manifest := testManifest(t)
	if err := VerifySourceIncludes(root, manifest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "docs/packs.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "installed-pack-catalog.md") != 1 {
		t.Fatal("pack include count")
	}
}

func TestProductTruth_SourceIncludesReleaseHistoryFragment(t *testing.T) {
	root := testRoot(t)
	manifest := testManifest(t)
	if err := VerifySourceIncludes(root, manifest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "docs/status.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "release-history.md") != 1 {
		t.Fatal("release include count")
	}
}

func TestProductTruth_SourceRejectsParallelOrInvalidConsumption(t *testing.T) {
	root := t.TempDir()
	manifest := testManifest(t)
	for _, file := range []string{"docs/reference.md", "docs/packs.md", "docs/status.md"} {
		target := filepath.Join(root, file)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("parallel reader\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err := VerifySourceIncludes(root, manifest)
	if err == nil || !strings.Contains(err.Error(), "PT204_CONSUMPTION") {
		t.Fatalf("err=%v", err)
	}
}
