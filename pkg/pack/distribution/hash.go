package distribution

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ComputeContentHash computes a SHA-256 hash from a sorted manifest of
// relative-path:SHA-256-file-hash pairs covering every file in dir.
// Paths use forward slashes regardless of OS.
func ComputeContentHash(dir string) (string, error) {
	var entries []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return fmt.Errorf("computing relative path: %w", relErr)
		}
		// Normalize to forward slashes for cross-platform consistency.
		rel = filepath.ToSlash(rel)

		fileHash, hashErr := ComputeFileHash(path)
		if hashErr != nil {
			return fmt.Errorf("hashing file %s: %w", rel, hashErr)
		}

		entries = append(entries, rel+":"+fileHash)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking directory %s: %w", dir, err)
	}

	sort.Strings(entries)

	manifest := strings.Join(entries, "\n")
	h := sha256.Sum256([]byte(manifest))
	return hex.EncodeToString(h[:]), nil
}

// ComputeFileHash computes the SHA-256 hex digest of a single file's content.
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
