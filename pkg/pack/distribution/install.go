package distribution

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
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
	// Warnings surfaces manifest-vs-lock reconciliation divergences (stale lock entries,
	// manifest packs missing from the lock, or an absent manifest) so the CLI can report
	// them loudly instead of exiting vacuously green.
	Warnings []string `json:"warnings,omitempty"`
}

// readManifestPacks reads the DECLARED packs from backstop.yml (`packs:`), reusing the
// package's existing backstopYml model (which tolerates the `"local"` value). The bool
// reports whether the manifest is PRESENT; an absent manifest means nothing is declared.
func readManifestPacks(projectDir string) (map[string]string, bool, error) {
	path := filepath.Join(projectDir, "backstop.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("reading backstop.yml: %w", err)
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return nil, false, fmt.Errorf("parsing backstop.yml: %w", err)
	}

	if yml.Packs == nil {
		yml.Packs = make(map[string]string)
	}
	return yml.Packs, true, nil
}

// Install restores the packs DECLARED in backstop.yml, reconciling them against
// backstop.lock (which supplies version/source/hash). The manifest is the source of
// truth for WHAT to install; a lock entry absent from the manifest is a stale entry
// (surfaced, not installed), and a manifest pack absent from the lock is surfaced too.
// Local packs are materialized by copying their recorded source into
// .backstop/packs/<name>/; git packs are cloned or read from cache. Content hashes are
// verified. Does NOT run validation or merge tool_config.
func Install(opts InstallOptions) (*InstallResult, error) {
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	lf, err := ReadLockfile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("backstop.lock not found: %w", err)
	}

	result := &InstallResult{
		InstalledPacks: []string{},
		Warnings:       []string{},
	}

	// The DECLARED manifest is the source of truth for WHAT to install (Defect B). An
	// absent backstop.yml means nothing is declared: install NOTHING, no lf.Packs fallback.
	manifestPacks, manifestPresent, err := readManifestPacks(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving declared packs: %w", err)
	}
	if !manifestPresent {
		result.Warnings = append(result.Warnings,
			"no backstop.yml manifest found: nothing is declared, so nothing was installed")
		return result, nil
	}

	// Surface stale lock entries: present in the lock but NOT declared in the manifest
	// (e.g. a renamed slotly/go-standards). These are called out and NOT installed.
	staleNames := make([]string, 0, len(lf.Packs))
	for name := range lf.Packs {
		if _, declared := manifestPacks[name]; !declared {
			staleNames = append(staleNames, name)
		}
	}
	sort.Strings(staleNames)
	for _, name := range staleNames {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"stale lock entry %q is not declared in backstop.yml; skipping it (run 'backstop pack remove' to clean it up)", name))
	}

	packsDir := filepath.Join(opts.ProjectDir, ".backstop", "packs")

	// Snapshot current state for atomic rollback.
	var snapshotDir string
	if info, statErr := os.Stat(packsDir); statErr == nil && info.IsDir() {
		snapshotDir, err = os.MkdirTemp("", "backstop-snapshot-*")
		if err != nil {
			return nil, fmt.Errorf("creating snapshot: %w", err)
		}
		defer func() { _ = os.RemoveAll(snapshotDir) }()
		if copyErr := copyDirRecursive(packsDir, snapshotDir); copyErr != nil {
			return nil, fmt.Errorf("snapshotting packs: %w", copyErr)
		}
	}

	// rollback best-effort restores the pre-install packs dir; its cleanup calls are
	// genuinely fire-and-forget (there is no recovery if undo itself fails).
	rollback := func() {
		_ = os.RemoveAll(packsDir)
		if snapshotDir != "" {
			_ = os.MkdirAll(filepath.Dir(packsDir), 0o755)
			_ = copyDirRecursive(snapshotDir, packsDir)
		}
	}

	// Install exactly what the manifest declares, in deterministic order.
	declaredNames := make([]string, 0, len(manifestPacks))
	for name := range manifestPacks {
		declaredNames = append(declaredNames, name)
	}
	sort.Strings(declaredNames)

	for _, name := range declaredNames {
		entry, inLock := lf.Packs[name]
		if !inLock {
			// A declared pack absent from the lock is surfaced, not silently skipped: we
			// have no version/source/hash to install it from.
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"declared pack %q is missing from backstop.lock; skipping it (run 'backstop pack add' or 'pack relock' to lock it)", name))
			continue
		}

		if entry.SourceType == "local" {
			if matErr := materializeLocalPack(opts, name, entry, packsDir); matErr != nil {
				rollback()
				return nil, matErr
			}
			result.InstalledPacks = append(result.InstalledPacks, name)
			continue
		}

		// Git packs: clone or read from cache.
		var sourceDir string
		if opts.CachePath != "" {
			sourceDir = filepath.Join(opts.CachePath, name)
			if _, statErr := os.Stat(sourceDir); statErr != nil {
				rollback()
				return nil, fmt.Errorf("pack %s not found in cache %s", name, opts.CachePath)
			}
		} else {
			tmpDir, mkErr := os.MkdirTemp("", "backstop-install-*")
			if mkErr != nil {
				rollback()
				return nil, mkErr
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

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

// materializeLocalPack resolves a local pack's source directory and COPIES it into
// .backstop/packs/<name>/ (mirroring the git branch). The source is opts.LocalPackDir
// when set (an explicit test override), otherwise the lock's project-relative local_path
// resolved against opts.ProjectDir. An unresolvable source — empty local_path with no
// override, or a resolved dir missing on disk — FAILS LOUD naming the pack; there is no
// silent no-op. This helper is reusable (ISSUE-026 pack add will call it too).
func materializeLocalPack(opts InstallOptions, name string, entry LockEntry, packsDir string) error {
	sourceDir := opts.LocalPackDir
	if sourceDir == "" {
		if entry.LocalPath == "" {
			return fmt.Errorf("local pack %q has no recorded source path (local_path) in backstop.lock; cannot materialize it (re-add the pack)", name)
		}
		sourceDir = filepath.Join(opts.ProjectDir, entry.LocalPath)
	}

	info, statErr := os.Stat(sourceDir)
	if statErr != nil {
		return fmt.Errorf("local pack %q source %q does not exist on disk; cannot materialize it", name, sourceDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("local pack %q source %q is not a directory; cannot materialize it", name, sourceDir)
	}

	hash, hashErr := ComputeContentHash(sourceDir)
	if hashErr != nil {
		return fmt.Errorf("computing hash for local pack %s: %w", name, hashErr)
	}
	if hash != entry.ContentHash {
		return fmt.Errorf("hash mismatch for local pack %s: computed=%s locked=%s", name, hash, entry.ContentHash)
	}

	destDir := filepath.Join(packsDir, name)
	if mkErr := os.MkdirAll(filepath.Dir(destDir), 0o755); mkErr != nil {
		return mkErr
	}
	if copyErr := copyDirRecursive(sourceDir, destDir); copyErr != nil {
		return fmt.Errorf("copying local pack %s: %w", name, copyErr)
	}
	return nil
}
