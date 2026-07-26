package distribution

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// InstallOptions configures the pack install command.
//
// LocalPackDir stays: it is a source override for local packs, not a dependency.
// The cloner moved to NewInstallCommand, which also retired the nil-cloner guard
// that failed with no diagnostic at all.
type InstallOptions struct {
	ProjectDir   string
	CachePath    string
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
