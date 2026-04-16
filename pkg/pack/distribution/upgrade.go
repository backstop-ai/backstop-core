package distribution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RemediationGenerator abstracts remediation bundle generation.
type RemediationGenerator interface {
	GenerateBundle(projectDir string, violations []string) (string, error)
}

// Scanner abstracts codebase scanning for violations.
type Scanner interface {
	ScanViolations(projectDir, packDir string) ([]string, error)
}

// UpgradeOptions configures the pack upgrade command.
type UpgradeOptions struct {
	ProjectDir           string
	GitCloner            GitCloner
	Validator            Validator
	RemediationGenerator RemediationGenerator
	Scanner              Scanner
}

// UpgradeResult holds the result of a pack upgrade operation.
type UpgradeResult struct {
	OldVersion          string `json:"old_version"`
	NewVersion          string `json:"new_version"`
	ContentHash         string `json:"content_hash"`
	RemediationBundle   string `json:"remediation_bundle"`
	BaselinedViolations int    `json:"baselined_violations"`
}

// Upgrade implements the pack upgrade pipeline: major version target,
// validate, scan, generate remediation, update config and lockfile.
func Upgrade(packRef string, opts UpgradeOptions) (*UpgradeResult, error) {
	// Parse pack reference with explicit major version.
	packName, targetVersion := parsePackRef(packRef)

	// Read current version.
	currentVersion, _, err := readPackVersion(opts.ProjectDir, packName)
	if err != nil {
		return nil, err
	}

	// Clone new version.
	tmpDir, err := os.MkdirTemp("", "backstop-upgrade-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	gitURL := resolveGitURL(packName)
	if err := opts.GitCloner.Clone(gitURL, "v"+targetVersion, tmpDir); err != nil {
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

	// Merge tool_config with conflict escalation.
	backstopDir := filepath.Join(opts.ProjectDir, ".backstop")
	if err := os.MkdirAll(backstopDir, 0o755); err != nil {
		return nil, err
	}

	provPath := filepath.Join(backstopDir, "pack-config-provenance.json")
	prov, err := ReadProvenance(provPath)
	if err != nil {
		return nil, err
	}

	mergeResult, err := MergeToolConfig(tmpDir, opts.ProjectDir, prov)
	if err != nil {
		return nil, fmt.Errorf("merging tool_config: %w", err)
	}

	if len(mergeResult.Conflicts) > 0 {
		var msgs []string
		for _, c := range mergeResult.Conflicts {
			msgs = append(msgs, fmt.Sprintf("%s: %s (pack=%s, current=%s)", c.ConfigFile, c.SettingKey, c.PackValue, c.CurrentValue))
		}
		return nil, fmt.Errorf("tool_config conflict during upgrade:\n%s", strings.Join(msgs, "\n"))
	}

	// Scan for violations.
	var violations []string
	if opts.Scanner != nil {
		violations, err = opts.Scanner.ScanViolations(opts.ProjectDir, tmpDir)
		if err != nil {
			return nil, fmt.Errorf("scanning violations: %w", err)
		}
	}

	// Generate remediation bundle if there are violations.
	var remediationBundle string
	if len(violations) > 0 && opts.RemediationGenerator != nil {
		bundle, genErr := opts.RemediationGenerator.GenerateBundle(opts.ProjectDir, violations)
		if genErr != nil {
			// Rollback: don't update anything.
			return nil, fmt.Errorf("generating remediation bundle: %w", genErr)
		}
		remediationBundle = bundle
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

	// Record provenance.
	for i := range mergeResult.Merged {
		mergeResult.Merged[i].SourcePack = packName
	}
	prov.Entries = append(prov.Entries, mergeResult.Merged...)
	if err := WriteProvenance(provPath, prov); err != nil {
		return nil, err
	}

	// Update backstop.yml.
	if err := updatePackVersion(opts.ProjectDir, packName, targetVersion); err != nil {
		return nil, err
	}

	// Update backstop.lock.
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	lf, _ := ReadLockfile(lockPath)
	if lf == nil {
		lf = &Lockfile{Packs: make(map[string]LockEntry)}
	}

	ref := "v" + targetVersion
	lf.Packs[packName] = LockEntry{
		Name:        packName,
		Version:     targetVersion,
		GitRef:      &ref,
		ContentHash: contentHash,
		SourceType:  "git",
		InstallDate: time.Now().UTC().Format(time.RFC3339),
	}

	if err := WriteLockfile(lockPath, lf); err != nil {
		return nil, err
	}

	return &UpgradeResult{
		OldVersion:          currentVersion,
		NewVersion:          targetVersion,
		ContentHash:         contentHash,
		RemediationBundle:   remediationBundle,
		BaselinedViolations: len(violations),
	}, nil
}
