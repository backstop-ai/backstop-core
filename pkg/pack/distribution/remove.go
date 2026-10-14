package distribution

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RemoveOptions configures the pack remove command.
type RemoveOptions struct {
	ProjectDir string
}

// RemoveResult holds the result of a pack remove operation.
type RemoveResult struct {
	RevertedSettings []string `json:"reverted_settings"`
	Warnings         []string `json:"warnings"`
}

// Remove implements the pack remove pipeline: provenance-based config revert,
// cleanup from packs dir, backstop.yml, backstop.lock, and provenance.
func Remove(packName string, opts RemoveOptions) (*RemoveResult, error) {
	result := &RemoveResult{
		RevertedSettings: []string{},
		Warnings:         []string{},
	}

	// Verify pack is installed.
	if !isPackInstalled(opts.ProjectDir, packName) {
		return nil, fmt.Errorf("pack %s is not installed", packName)
	}

	// Read provenance and check settings.
	provPath := filepath.Join(opts.ProjectDir, ".backstop", "pack-config-provenance.json")
	prov, err := ReadProvenance(provPath)
	if err != nil {
		return nil, fmt.Errorf("removing %s: %w", packName, err)
	}

	for _, entry := range prov.Entries {
		if entry.SourcePack != packName {
			continue
		}

		// Check if setting was modified since install.
		configPath := filepath.Join(opts.ProjectDir, entry.ConfigFile)
		currentConfig, readErr := readConfigFile(configPath)
		if readErr != nil || currentConfig == nil {
			result.RevertedSettings = append(result.RevertedSettings, entry.SettingKey)
			continue
		}

		if currentVal, exists := currentConfig[entry.SettingKey]; exists {
			currentHash := computeValueHash(currentVal)
			if currentHash != entry.ValueHash {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("%s: %s was modified since install (not reverting)", entry.ConfigFile, entry.SettingKey))
				continue
			}
		}

		result.RevertedSettings = append(result.RevertedSettings, entry.SettingKey)
	}

	// Read backstop.lock before changing anything. A missing lockfile is
	// allowed; an unreadable one stops the remove with nothing changed.
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	lf, err := ReadLockfile(lockPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("removing %s: %w", packName, err)
	}

	// Delete from .backstop/packs/. A failed delete stops the remove before
	// backstop.yml, backstop.lock or provenance change, so a pack whose files
	// are still installed is never reported as removed.
	packDir := filepath.Join(opts.ProjectDir, ".backstop", "packs", packName)
	if err := os.RemoveAll(packDir); err != nil {
		return nil, fmt.Errorf("removing pack directory %s: %w", packDir, err)
	}

	// Remove from backstop.yml.
	if err := removeFromBackstopYml(opts.ProjectDir, packName); err != nil {
		return nil, fmt.Errorf("removing %s from backstop.yml: %w", packName, err)
	}

	// Remove from backstop.lock.
	if lf != nil {
		delete(lf.Packs, packName)
		if err := WriteLockfile(lockPath, lf); err != nil {
			return nil, fmt.Errorf("removing %s from backstop.lock: %w", packName, err)
		}
	}

	// Remove from provenance.
	var filtered []ProvenanceEntry
	for _, entry := range prov.Entries {
		if entry.SourcePack != packName {
			filtered = append(filtered, entry)
		}
	}
	prov.Entries = filtered
	if err := WriteProvenance(provPath, prov); err != nil {
		return nil, fmt.Errorf("removing %s from provenance: %w", packName, err)
	}

	return result, nil
}

// removeFromBackstopYml removes a pack entry from backstop.yml.
func removeFromBackstopYml(projectDir, packName string) error {
	path := filepath.Join(projectDir, "backstop.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return err
	}

	delete(yml.Packs, packName)

	out, err := yaml.Marshal(&yml)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0o644)
}
