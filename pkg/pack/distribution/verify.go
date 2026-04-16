package distribution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VerifyResult holds the result of a gate-time lock verification.
type VerifyResult struct {
	Pass     bool          `json:"pass"`
	Failures []LockFailure `json:"failures"`
}

// LockFailure describes a single lock verification failure.
type LockFailure struct {
	Pack    string `json:"pack"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// VerifyLock verifies that all locked packs are present with matching hashes,
// detects extra unlocked packs, and handles missing lockfile.
func VerifyLock(lockfile *Lockfile, packsDir string, ymlPacks []string) (*VerifyResult, error) {
	result := &VerifyResult{
		Pass:     true,
		Failures: []LockFailure{},
	}

	// If lockfile is nil but packs are declared: missing_lockfile failure.
	if lockfile == nil {
		if len(ymlPacks) > 0 {
			result.Pass = false
			result.Failures = append(result.Failures, LockFailure{
				Pack:    "",
				Kind:    "missing_lockfile",
				Message: "backstop.lock is absent but packs are declared in backstop.yml",
			})
		}
		return result, nil
	}

	// Verify each locked pack.
	for name, entry := range lockfile.Packs {
		// Skip local packs — they are not in packsDir.
		if entry.SourceType == "local" {
			continue
		}

		packDir := filepath.Join(packsDir, name)
		info, err := os.Stat(packDir)
		if err != nil || !info.IsDir() {
			result.Pass = false
			result.Failures = append(result.Failures, LockFailure{
				Pack:    name,
				Kind:    "missing_pack",
				Message: fmt.Sprintf("locked pack %s is not present in %s; run pack install to restore", name, packsDir),
			})
			continue
		}

		// Compute current hash and compare to locked hash.
		currentHash, hashErr := ComputeContentHash(packDir)
		if hashErr != nil {
			return nil, fmt.Errorf("computing hash for %s: %w", name, hashErr)
		}

		if currentHash != entry.ContentHash {
			result.Pass = false
			result.Failures = append(result.Failures, LockFailure{
				Pack:    name,
				Kind:    "hash_mismatch",
				Message: fmt.Sprintf("pack %s content hash mismatch: installed=%s locked=%s; run pack install to restore", name, currentHash, entry.ContentHash),
			})
		}
	}

	// Check for extra unlocked packs in packsDir.
	if err := detectExtraUnlocked(lockfile, packsDir, result); err != nil {
		// If packsDir doesn't exist, that's fine — no extra packs.
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("checking for extra packs: %w", err)
		}
	}

	return result, nil
}

// detectExtraUnlocked walks the packs directory and reports packs not in the lockfile.
func detectExtraUnlocked(lockfile *Lockfile, packsDir string, result *VerifyResult) error {
	if _, err := os.Stat(packsDir); os.IsNotExist(err) {
		return nil
	}

	// Walk two levels deep: org/pack-name
	orgs, err := os.ReadDir(packsDir)
	if err != nil {
		return err
	}

	for _, org := range orgs {
		if !org.IsDir() {
			continue
		}
		orgPath := filepath.Join(packsDir, org.Name())
		packs, readErr := os.ReadDir(orgPath)
		if readErr != nil {
			continue
		}

		for _, pack := range packs {
			if !pack.IsDir() {
				continue
			}
			fullName := org.Name() + "/" + pack.Name()
			fullName = strings.ReplaceAll(fullName, "\\", "/")

			if _, locked := lockfile.Packs[fullName]; !locked {
				result.Pass = false
				result.Failures = append(result.Failures, LockFailure{
					Pack:    fullName,
					Kind:    "extra_unlocked",
					Message: fmt.Sprintf("pack %s exists in %s but is not in backstop.lock", fullName, packsDir),
				})
			}
		}
	}

	return nil
}
