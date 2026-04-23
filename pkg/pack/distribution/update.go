package distribution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// VersionResolver abstracts semver version resolution.
type VersionResolver interface {
	ResolveLatestCompatible(packName, currentVersion string) (string, error)
	IsMajorBump(current, resolved string) bool
}

// UpdateOptions configures the pack update command.
type UpdateOptions struct {
	ProjectDir      string
	Acknowledge     bool
	GitCloner       GitCloner
	Validator       Validator
	VersionResolver VersionResolver
}

// UpdateResult holds the result of a pack update operation.
type UpdateResult struct {
	OldVersion  string `json:"old_version"`
	NewVersion  string `json:"new_version"`
	ContentHash string `json:"content_hash"`
	NoOp        bool   `json:"no_op"`
	Message     string `json:"message"`
}

// Update implements the pack update pipeline: resolve latest compatible version,
// validate, tamper detect, and update manifest/lockfile.
func Update(packName string, opts UpdateOptions) (*UpdateResult, error) {
	// Read current version from backstop.yml.
	currentVersion, isLocal, err := readPackVersion(opts.ProjectDir, packName)
	if err != nil {
		return nil, err
	}

	// Local packs: no-op.
	if isLocal {
		return &UpdateResult{
			NoOp:    true,
			Message: fmt.Sprintf("pack %s is a local path pack; it updates when its source files change", packName),
		}, nil
	}

	// Resolve latest compatible version.
	if opts.VersionResolver == nil {
		return nil, fmt.Errorf("version resolver required for update")
	}

	resolved, err := opts.VersionResolver.ResolveLatestCompatible(packName, currentVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving version: %w", err)
	}

	// Already at latest?
	if resolved == currentVersion {
		return &UpdateResult{
			OldVersion: currentVersion,
			NewVersion: currentVersion,
			NoOp:       true,
			Message:    fmt.Sprintf("pack %s is already at latest compatible version %s", packName, currentVersion),
		}, nil
	}

	// Major version bump? Refuse.
	if opts.VersionResolver.IsMajorBump(currentVersion, resolved) {
		return nil, fmt.Errorf("version %s is a major version bump from %s; use pack upgrade instead", resolved, currentVersion)
	}

	// Clone new version.
	tmpDir, err := os.MkdirTemp("", "backstop-update-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	gitURL := resolveGitURL(packName)
	if err := opts.GitCloner.Clone(gitURL, "v"+resolved, tmpDir); err != nil {
		return nil, err
	}

	// Validate.
	if opts.Validator != nil {
		if err := opts.Validator.RunPackCheck(tmpDir); err != nil {
			return nil, err
		}
		if err := opts.Validator.RunPackTest(tmpDir); err != nil {
			return nil, err
		}
	}

	// Tamper detection.
	currentPackDir := filepath.Join(opts.ProjectDir, ".backstop", "packs", packName)
	if _, statErr := os.Stat(currentPackDir); statErr == nil {
		tamperResult, tamperErr := DetectTamper(currentPackDir, tmpDir)
		if tamperErr != nil {
			return nil, fmt.Errorf("tamper detection: %w", tamperErr)
		}

		if tamperResult.HasTamper && !opts.Acknowledge {
			var descriptions []string
			for _, c := range tamperResult.Changes {
				descriptions = append(descriptions, fmt.Sprintf("  %s: %s", c.Kind, c.Description))
			}
			return nil, fmt.Errorf("tamper detected in %s update:\n%s\nre-run with --acknowledge to proceed",
				packName, strings.Join(descriptions, "\n"))
		}
	}

	// Install new version.
	installedPath := filepath.Join(opts.ProjectDir, ".backstop", "packs", packName)
	os.RemoveAll(installedPath)
	if err := os.MkdirAll(filepath.Dir(installedPath), 0o755); err != nil {
		return nil, err
	}
	if err := copyDirRecursive(tmpDir, installedPath); err != nil {
		return nil, err
	}

	// Compute hash.
	contentHash, err := ComputeContentHash(installedPath)
	if err != nil {
		return nil, err
	}

	// Update backstop.yml.
	if err := updatePackVersion(opts.ProjectDir, packName, resolved); err != nil {
		return nil, err
	}

	// Update backstop.lock.
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	lf, _ := ReadLockfile(lockPath)
	if lf == nil {
		lf = &Lockfile{Packs: make(map[string]LockEntry)}
	}

	ref := "v" + resolved
	lf.Packs[packName] = LockEntry{
		Name:        packName,
		Version:     resolved,
		GitRef:      &ref,
		ContentHash: contentHash,
		SourceType:  "git",
		InstallDate: time.Now().UTC().Format(time.RFC3339),
	}

	if err := WriteLockfile(lockPath, lf); err != nil {
		return nil, err
	}

	return &UpdateResult{
		OldVersion:  currentVersion,
		NewVersion:  resolved,
		ContentHash: contentHash,
	}, nil
}

// readPackVersion reads the current version and source type from backstop.yml.
func readPackVersion(projectDir, packName string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if err != nil {
		return "", false, err
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return "", false, err
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
		return err
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return err
	}

	yml.Packs[packName] = newVersion

	out, err := yaml.Marshal(&yml)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0o644)
}
