package distribution

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// VersionResolver abstracts semver version resolution.
type VersionResolver interface {
	// ResolveLatestCompatible takes the repository COORDINATE, not the pack name: after
	// SPEC-056 REQ-003 the two differ for any pack whose manifest name is not its
	// repository, and ls-remote must run against the repository.
	ResolveLatestCompatible(coordinate, currentVersion string) (string, error)
	IsMajorBump(current, resolved string) bool
}

// UpdateOptions configures the pack update command.
//
// Acknowledge stays: it is the consumer's tamper-acknowledgment flag, not a
// dependency. The three dependencies moved to NewUpdateCommand, which turned the
// runtime "version resolver required for update" check into an assembly failure.
type UpdateOptions struct {
	ProjectDir  string
	Acknowledge bool
}

// UpdateResult holds the result of a pack update operation.
type UpdateResult struct {
	OldVersion  string `json:"old_version"`
	NewVersion  string `json:"new_version"`
	ContentHash string `json:"content_hash"`
	NoOp        bool   `json:"no_op"`
	Message     string `json:"message"`
	// Warnings carries diagnostics the update produced without failing: the
	// coordinate fallback (REQ-005) and the divergence diagnostic (REQ-006).
	//
	// It did not exist before SPEC-056, which is exactly why REQ-011 exists — a warning
	// computed inside update with no field to hold it is computed and silently DROPPED,
	// and the code that drops it looks correct.
	Warnings []string `json:"warnings,omitempty"`
}

// readPackVersion reads the current version and source type from backstop.yml.
func readPackVersion(projectDir, packName string) (string, bool, error) {
	path := filepath.Join(projectDir, "backstop.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, fmt.Errorf("reading %s: %w", path, err)
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return "", false, fmt.Errorf("parsing %s: %w", path, err)
	}

	version, exists := yml.Packs[packName]
	if !exists {
		return "", false, fmt.Errorf("pack %s not found in backstop.yml", packName)
	}
	if version == "local" {
		return "", true, nil
	}
	return version, false, nil
}

// updatePackVersion updates the version of a pack in backstop.yml.
func updatePackVersion(projectDir, packName, newVersion string) error {
	path := filepath.Join(projectDir, "backstop.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	yml.Packs[packName] = newVersion

	out, err := yaml.Marshal(&yml)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0o644)
}
