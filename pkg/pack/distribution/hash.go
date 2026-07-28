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
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return fmt.Errorf("computing relative path: %w", relErr)
		}
		// Normalize to forward slashes for cross-platform consistency.
		rel = filepath.ToSlash(rel)

		// Tested before the IsDir short-circuit so a metadata DIRECTORY is
		// reached at all: skipping the entry without SkipDir would descend into
		// it and hash the whole subtree.
		if isRootRepositoryMetadata(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			// A file or symlink shape returns nil, never SkipDir — SkipDir on a
			// non-directory abandons the rest of the parent, silently dropping
			// every sibling after it from the manifest. Returning here is also
			// what keeps ComputeFileHash from opening a .git symlink, so a
			// dangling one is skipped instead of failing the walk.
			return nil
		}

		if info.IsDir() {
			return nil
		}

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

// isRootRepositoryMetadata reports whether rel names repository-control metadata
// at the walked directory's own root. rel is slash-normalized and relative to
// that root, so the root itself arrives as "." and is never excluded.
//
// The match is exact name equality against gitDirectoryName and nothing else —
// no prefix, no suffix, no nesting. That scope is deliberate: ExecGitCloner
// strips exactly one path from a clone, so mirroring it here is what makes local
// and remote identity CONVERGE on one algorithm rather than merely both change.
// Matching at any depth would break that agreement, and it would also break
// install's rollback snapshot, whose walk root is the packs directory rather
// than a pack root.
//
// It knows nothing about the entry's shape, which is what makes a metadata
// directory, worktree pointer file, and symlink one rule rather than three.
func isRootRepositoryMetadata(rel string) bool {
	return rel == gitDirectoryName
}

// ComputeFileHash computes the SHA-256 hex digest of a single file's content.
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file %s: %w", path, err)
	}
	// Read-only handle: a close failure cannot affect the digest already read.
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
