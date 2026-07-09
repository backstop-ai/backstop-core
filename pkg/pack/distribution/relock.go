package distribution

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// RelockResult holds the outcome of a pack relock operation.
type RelockResult struct {
	PackName    string `json:"pack_name"`
	ContentHash string `json:"content_hash"`
}

// Relock re-reads a locally-installed pack and refreshes its backstop.lock entry
// (ISSUE-032 Defect F / CLM-010). It reads the pack name from the pack.yml at packPath,
// recomputes the content hash over the INSTALLED pack directory
// (.backstop/packs/<name>), and overwrites the matching lock entry's content_hash and
// install_date — preserving SourceType "local". It is the clean refresh path for a
// locally-edited pack, replacing the remove+add workaround (`pack add` refuses an
// already-installed pack and `pack update` is a no-op for a local source).
//
// It errors clearly when the pack path has no pack.yml, when the pack is absent from
// the lockfile, or when the lock entry is not a local-source pack (git packs refresh
// through pack update/upgrade, not relock).
func Relock(projectDir, packPath string) (*RelockResult, error) {
	name, err := readPackName(packPath)
	if err != nil {
		return nil, err
	}

	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, err := ReadLockfile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("reading lockfile: %w", err)
	}

	entry, ok := lf.Packs[name]
	if !ok {
		return nil, fmt.Errorf("pack %q is not in backstop.lock; add it first with pack add", name)
	}
	if entry.SourceType != "local" {
		return nil, fmt.Errorf("pack %q is not a local pack (source_type %q); relock only refreshes local packs — use pack update/upgrade for git packs", name, entry.SourceType)
	}

	installedDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(name))
	if _, statErr := os.Stat(installedDir); statErr != nil {
		return nil, fmt.Errorf("installed pack directory %s not found: %w", installedDir, statErr)
	}

	hash, err := ComputeContentHash(installedDir)
	if err != nil {
		return nil, fmt.Errorf("computing content hash for %q: %w", name, err)
	}

	entry.ContentHash = hash
	entry.InstallDate = time.Now().UTC().Format(time.RFC3339)
	lf.Packs[name] = entry

	if err := WriteLockfile(lockPath, lf); err != nil {
		return nil, fmt.Errorf("writing lockfile: %w", err)
	}

	return &RelockResult{PackName: name, ContentHash: hash}, nil
}

// readPackName reads the `name:` field from a pack.yml at packDir, failing loud when
// the manifest is missing or nameless.
func readPackName(packDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(packDir, "pack.yml"))
	if err != nil {
		return "", fmt.Errorf("reading pack manifest at %s: %w", packDir, err)
	}
	var manifest struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("parsing pack manifest at %s: %w", packDir, err)
	}
	if manifest.Name == "" {
		return "", fmt.Errorf("pack manifest at %s has no name", packDir)
	}
	return manifest.Name, nil
}
