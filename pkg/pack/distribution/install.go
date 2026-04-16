package distribution

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallOptions configures the pack install command.
type InstallOptions struct {
	ProjectDir   string
	CachePath    string
	GitCloner    GitCloner
	LocalPackDir string
}

// InstallResult holds the result of a pack install operation.
type InstallResult struct {
	InstalledPacks []string `json:"installed_packs"`
}

// Install restores all packs from backstop.lock by cloning at pinned version
// and verifying content hash. Does NOT run validation or merge tool_config.
func Install(opts InstallOptions) (*InstallResult, error) {
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	lf, err := ReadLockfile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("backstop.lock not found: %w", err)
	}

	result := &InstallResult{
		InstalledPacks: []string{},
	}

	packsDir := filepath.Join(opts.ProjectDir, ".backstop", "packs")

	// Snapshot current state for atomic rollback.
	var snapshotDir string
	if info, statErr := os.Stat(packsDir); statErr == nil && info.IsDir() {
		snapshotDir, err = os.MkdirTemp("", "backstop-snapshot-*")
		if err != nil {
			return nil, fmt.Errorf("creating snapshot: %w", err)
		}
		defer os.RemoveAll(snapshotDir)
		if copyErr := copyDirRecursive(packsDir, snapshotDir); copyErr != nil {
			return nil, fmt.Errorf("snapshotting packs: %w", copyErr)
		}
	}

	rollback := func() {
		os.RemoveAll(packsDir)
		if snapshotDir != "" {
			os.MkdirAll(filepath.Dir(packsDir), 0o755)
			copyDirRecursive(snapshotDir, packsDir)
		}
	}

	for name, entry := range lf.Packs {
		if entry.SourceType == "local" {
			// Local packs: verify hash at source path.
			sourceDir := opts.LocalPackDir
			if sourceDir == "" {
				// Skip if no local dir provided — will be verified at gate time.
				result.InstalledPacks = append(result.InstalledPacks, name)
				continue
			}
			hash, hashErr := ComputeContentHash(sourceDir)
			if hashErr != nil {
				rollback()
				return nil, fmt.Errorf("computing hash for local pack %s: %w", name, hashErr)
			}
			if hash != entry.ContentHash {
				rollback()
				return nil, fmt.Errorf("hash mismatch for local pack %s: computed=%s locked=%s", name, hash, entry.ContentHash)
			}
			result.InstalledPacks = append(result.InstalledPacks, name)
			continue
		}

		// Git packs: clone or read from cache.
		var sourceDir string
		if opts.CachePath != "" {
			// Read from cache.
			sourceDir = filepath.Join(opts.CachePath, name)
			if _, statErr := os.Stat(sourceDir); statErr != nil {
				rollback()
				return nil, fmt.Errorf("pack %s not found in cache %s", name, opts.CachePath)
			}
		} else {
			// Clone from git.
			tmpDir, mkErr := os.MkdirTemp("", "backstop-install-*")
			if mkErr != nil {
				rollback()
				return nil, mkErr
			}
			defer os.RemoveAll(tmpDir)

			if opts.GitCloner == nil {
				rollback()
				return nil, fmt.Errorf("no git cloner provided for pack %s", name)
			}

			gitURL := resolveGitURL(name)
			if cloneErr := opts.GitCloner.Clone(gitURL, *entry.GitRef, tmpDir); cloneErr != nil {
				rollback()
				return nil, cloneErr
			}
			sourceDir = tmpDir
		}

		// Verify content hash.
		hash, hashErr := ComputeContentHash(sourceDir)
		if hashErr != nil {
			rollback()
			return nil, fmt.Errorf("computing hash for %s: %w", name, hashErr)
		}
		if hash != entry.ContentHash {
			rollback()
			return nil, fmt.Errorf("hash mismatch for pack %s: computed=%s locked=%s", name, hash, entry.ContentHash)
		}

		// Copy to .backstop/packs/.
		destDir := filepath.Join(packsDir, name)
		if mkErr := os.MkdirAll(filepath.Dir(destDir), 0o755); mkErr != nil {
			rollback()
			return nil, mkErr
		}
		if copyErr := copyDirRecursive(sourceDir, destDir); copyErr != nil {
			rollback()
			return nil, fmt.Errorf("copying pack %s: %w", name, copyErr)
		}

		result.InstalledPacks = append(result.InstalledPacks, name)
	}

	return result, nil
}
