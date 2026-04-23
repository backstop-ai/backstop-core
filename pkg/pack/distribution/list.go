package distribution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ListOptions configures the pack list command.
type ListOptions struct {
	ProjectDir string
	JSON       bool
}

// ListResult holds the result of a pack list operation.
type ListResult struct {
	Packs           []PackInfo `json:"packs"`
	FormattedOutput string     `json:"-"`
}

// PackInfo describes a single installed pack.
type PackInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	LockStatus    string `json:"lock_status"`
	Archetype     string `json:"archetype"`
	RuleCount     int    `json:"rule_count"`
	ScaffoldCount int    `json:"scaffold_count"`
}

// listManifest is a minimal pack.yml for list metadata extraction.
type listManifest struct {
	Archetype string        `yaml:"archetype"`
	Rules     []interface{} `yaml:"rules"`
	Scaffolds []interface{} `yaml:"scaffolds"`
}

// List implements the pack list command: reads backstop.yml and backstop.lock,
// computes lock status, and formats output.
func List(opts ListOptions) (*ListResult, error) {
	// Read backstop.yml.
	ymlPath := filepath.Join(opts.ProjectDir, "backstop.yml")
	data, err := os.ReadFile(ymlPath)
	if err != nil {
		return nil, fmt.Errorf("reading backstop.yml: %w", err)
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return nil, fmt.Errorf("parsing backstop.yml: %w", err)
	}

	// Read backstop.lock.
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	lf, _ := ReadLockfile(lockPath)

	result := &ListResult{
		Packs: []PackInfo{},
	}

	for ref, version := range yml.Packs {
		info := PackInfo{
			Name:    ref,
			Version: version,
		}

		// Determine lock status.
		if lf != nil {
			if entry, exists := lf.Packs[ref]; exists {
				info.LockStatus = computeLockStatus(opts.ProjectDir, ref, version, entry)
				if info.Version == "" || info.Version == "local" {
					info.Version = entry.Version
				}
			} else {
				info.LockStatus = "missing"
			}
		}

		// Read pack metadata.
		packDir := filepath.Join(opts.ProjectDir, ".backstop", "packs", filepath.FromSlash(ref))
		if manifest, readErr := readListManifest(packDir); readErr == nil {
			info.Archetype = manifest.Archetype
			info.RuleCount = len(manifest.Rules)
			info.ScaffoldCount = len(manifest.Scaffolds)
		}

		result.Packs = append(result.Packs, info)
	}

	// Format output.
	if opts.JSON {
		out, err := json.MarshalIndent(result.Packs, "", "  ")
		if err != nil {
			return nil, err
		}
		result.FormattedOutput = string(out)
	} else {
		result.FormattedOutput = formatListTable(result.Packs)
	}

	return result, nil
}

// computeLockStatus determines the lock status: locked, stale, or missing.
func computeLockStatus(projectDir, ref, version string, entry LockEntry) string {
	packDir := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(ref))

	if _, err := os.Stat(packDir); os.IsNotExist(err) {
		return "missing"
	}

	hash, err := ComputeContentHash(packDir)
	if err != nil {
		return "missing"
	}

	if hash == entry.ContentHash {
		return "locked"
	}
	return "stale"
}

// readListManifest reads minimal pack manifest for list metadata.
func readListManifest(packDir string) (*listManifest, error) {
	data, err := os.ReadFile(filepath.Join(packDir, "pack.yml"))
	if err != nil {
		return nil, err
	}

	var manifest listManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// formatListTable formats pack info as a human-readable table.
func formatListTable(packs []PackInfo) string {
	var sb strings.Builder

	// Header.
	sb.WriteString(fmt.Sprintf("%-30s %-12s %-12s %-15s %-8s %-10s\n",
		"NAME", "VERSION", "LOCK STATUS", "ARCHETYPE", "RULES", "SCAFFOLDS"))
	sb.WriteString(strings.Repeat("-", 90) + "\n")

	for _, p := range packs {
		version := p.Version
		if version == "" {
			version = "-"
		}
		sb.WriteString(fmt.Sprintf("%-30s %-12s %-12s %-15s %-8d %-10d\n",
			p.Name, version, p.LockStatus, p.Archetype, p.RuleCount, p.ScaffoldCount))
	}

	return sb.String()
}
