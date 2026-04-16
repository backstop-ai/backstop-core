package distribution_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func TestContentHash_SortedManifest(t *testing.T) {
	dir := t.TempDir()

	// Create files in non-alphabetical order to verify sorting.
	writeFile(t, filepath.Join(dir, "z_file.txt"), "content z")
	writeFile(t, filepath.Join(dir, "a_file.txt"), "content a")
	writeFile(t, filepath.Join(dir, "m_file.txt"), "content m")

	hash, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	// Hash must be deterministic regardless of file creation order.
	hash2, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash second call: %v", err)
	}
	if hash != hash2 {
		t.Errorf("hash not deterministic: %s != %s", hash, hash2)
	}
}

func TestContentHash_CoversAllFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "file1.txt"), "content 1")
	writeFile(t, filepath.Join(dir, "file2.txt"), "content 2")

	hash1, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash: %v", err)
	}

	// Adding a file must change the hash.
	writeFile(t, filepath.Join(dir, "file3.txt"), "content 3")

	hash2, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash after adding file: %v", err)
	}

	if hash1 == hash2 {
		t.Error("hash should change when a new file is added")
	}
}

func TestContentHash_DeterministicOutput(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "pack.yml"), "name: test\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, "rule.semgrep.yml"), "rules:\n  - id: test\n")

	results := make([]string, 5)
	for i := range results {
		h, err := distribution.ComputeContentHash(dir)
		if err != nil {
			t.Fatalf("ComputeContentHash iteration %d: %v", i, err)
		}
		results[i] = h
	}

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("hash not deterministic: iteration %d (%s) != iteration 0 (%s)", i, results[i], results[0])
		}
	}
}

func TestComputeFileHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	writeFile(t, path, "hello world")

	hash, err := distribution.ComputeFileHash(path)
	if err != nil {
		t.Fatalf("ComputeFileHash: %v", err)
	}

	// SHA-256 of "hello world" is well-known.
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("ComputeFileHash = %s, want %s", hash, expected)
	}
}

func TestContentHash_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	hash, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash on empty dir: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash for empty directory")
	}
}

func TestContentHash_NestedDirs(t *testing.T) {
	dir := t.TempDir()

	nested := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "top.txt"), "top")
	writeFile(t, filepath.Join(nested, "deep.txt"), "deep")

	hash, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	// Changing nested file must change hash.
	writeFile(t, filepath.Join(nested, "deep.txt"), "modified")
	hash2, err := distribution.ComputeContentHash(dir)
	if err != nil {
		t.Fatalf("ComputeContentHash after modification: %v", err)
	}
	if hash == hash2 {
		t.Error("hash should change when nested file is modified")
	}
}

func TestContentHash_MixedLineEndings(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Write identical content with LF and CRLF.
	writeFile(t, filepath.Join(dir1, "file.txt"), "line1\nline2\n")
	writeFile(t, filepath.Join(dir2, "file.txt"), "line1\nline2\n")

	hash1, err := distribution.ComputeContentHash(dir1)
	if err != nil {
		t.Fatalf("ComputeContentHash dir1: %v", err)
	}
	hash2, err := distribution.ComputeContentHash(dir2)
	if err != nil {
		t.Fatalf("ComputeContentHash dir2: %v", err)
	}

	// Same content should produce same hash.
	if hash1 != hash2 {
		t.Errorf("same content produced different hashes: %s != %s", hash1, hash2)
	}

	// Different line endings should produce different hash (byte-level hashing).
	dir3 := t.TempDir()
	writeFile(t, filepath.Join(dir3, "file.txt"), "line1\r\nline2\r\n")
	hash3, err := distribution.ComputeContentHash(dir3)
	if err != nil {
		t.Fatalf("ComputeContentHash dir3: %v", err)
	}
	if hash1 == hash3 {
		t.Log("CRLF and LF produce different hashes (expected byte-level hashing)")
	}
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
