package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ConfigError signals a hard stop — exit code 2.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string { return e.Message }

// DegradedError signals degraded mode — skip the check with a warning.
type DegradedError struct {
	Message string
}

func (e *DegradedError) Error() string { return e.Message }

// SemgrepInstaller abstracts semgrep discovery and installation for testability.
type SemgrepInstaller interface {
	LookPath(name string) (string, error)
	ExistsAt(backstopDir string) (string, bool)
	Install(targetDir, version string) (string, error)
	Version(binPath string) (string, error)
}

// DefaultSemgrepInstaller implements SemgrepInstaller by using real system calls.
type DefaultSemgrepInstaller struct{}

// LookPath checks if semgrep is on PATH.
func (d *DefaultSemgrepInstaller) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// ExistsAt checks if semgrep exists in .backstop/tools/.
func (d *DefaultSemgrepInstaller) ExistsAt(backstopDir string) (string, bool) {
	binPath := filepath.Join(backstopDir, "tools", "semgrep")
	if _, err := os.Stat(binPath); err == nil {
		return binPath, true
	}
	return "", false
}

// Install downloads semgrep to .backstop/tools/.
func (d *DefaultSemgrepInstaller) Install(targetDir, version string) (string, error) {
	toolsDir := filepath.Join(targetDir, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return "", fmt.Errorf("creating tools directory: %w", err)
	}

	// Download to temp file for atomic rename
	tmpFile := filepath.Join(toolsDir, ".semgrep.tmp")
	defer os.Remove(tmpFile) // clean up on failure

	// Use pip install semgrep or download binary depending on platform
	// For now, use pip as the install mechanism
	versionArg := "semgrep"
	if version != "" {
		versionArg = "semgrep==" + version
	}

	cmd := exec.Command("pip", "install", "--target", toolsDir, versionArg)
	if err := cmd.Run(); err != nil {
		return "", &DegradedError{
			Message: fmt.Sprintf("failed to install semgrep: %v", err),
		}
	}

	binPath := filepath.Join(toolsDir, "semgrep")
	if _, err := os.Stat(binPath); err != nil {
		return "", &DegradedError{
			Message: "semgrep binary not found after install",
		}
	}

	return binPath, nil
}

// Version returns the installed semgrep version.
func (d *DefaultSemgrepInstaller) Version(binPath string) (string, error) {
	cmd := exec.Command(binPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("checking semgrep version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureSemgrep returns the path to a semgrep binary, auto-installing if needed.
// Uses DefaultSemgrepInstaller for real system operations.
func EnsureSemgrep(backstopDir string, pinnedVersion string) (string, error) {
	return ensureSemgrepWith(backstopDir, pinnedVersion, &DefaultSemgrepInstaller{})
}

// ensureSemgrepWith is the testable core using an injected installer.
func ensureSemgrepWith(backstopDir string, pinnedVersion string, installer SemgrepInstaller) (string, error) {
	// Step 1: Check PATH for existing semgrep
	if binPath, err := installer.LookPath("semgrep"); err == nil {
		// Found on PATH — verify version if pin exists
		if pinnedVersion != "" {
			installedVersion, verErr := installer.Version(binPath)
			if verErr != nil {
				return "", &DegradedError{
					Message: fmt.Sprintf("semgrep found on PATH but cannot determine version: %v", verErr),
				}
			}
			if installedVersion != pinnedVersion {
				return "", &ConfigError{
					Message: fmt.Sprintf("semgrep version mismatch: installed %s, pinned %s", installedVersion, pinnedVersion),
				}
			}
		}
		return binPath, nil
	}

	// Step 2: Check .backstop/tools/semgrep
	if binPath, exists := installer.ExistsAt(backstopDir); exists {
		// Verify version if pin exists
		if pinnedVersion != "" {
			installedVersion, verErr := installer.Version(binPath)
			if verErr != nil {
				return "", &DegradedError{
					Message: fmt.Sprintf("semgrep at %s but cannot determine version: %v", binPath, verErr),
				}
			}
			if installedVersion != pinnedVersion {
				return "", &ConfigError{
					Message: fmt.Sprintf("semgrep version mismatch: installed %s, pinned %s", installedVersion, pinnedVersion),
				}
			}
		}
		return binPath, nil
	}

	// Step 3: Auto-install with lock for concurrent safety
	unlock, lockErr := acquireSemgrepLock(backstopDir)
	if lockErr != nil {
		// Another process is installing — re-check if it finished
		if binPath, exists := installer.ExistsAt(backstopDir); exists {
			if pinnedVersion != "" {
				installedVersion, verErr := installer.Version(binPath)
				if verErr != nil {
					return "", &DegradedError{
						Message: fmt.Sprintf("semgrep at %s but cannot determine version: %v", binPath, verErr),
					}
				}
				if installedVersion != pinnedVersion {
					return "", &ConfigError{
						Message: fmt.Sprintf("semgrep version mismatch: installed %s, pinned %s", installedVersion, pinnedVersion),
					}
				}
			}
			return binPath, nil
		}
		return "", &DegradedError{
			Message: fmt.Sprintf("semgrep auto-install failed (lock contention): %v", lockErr),
		}
	}
	defer unlock()

	binPath, err := installer.Install(backstopDir, pinnedVersion)
	if err != nil {
		// Install failure is degraded mode, not config error
		return "", &DegradedError{
			Message: fmt.Sprintf("semgrep auto-install failed: %v", err),
		}
	}

	return binPath, nil
}

// semgrepLockPath returns the path to the semgrep install lock file.
func semgrepLockPath(backstopDir string) string {
	return filepath.Join(backstopDir, "tools", ".semgrep.lock")
}

// acquireSemgrepLock attempts to acquire the semgrep install lock.
// Returns a cleanup function that releases the lock.
func acquireSemgrepLock(backstopDir string) (func(), error) {
	lockPath := semgrepLockPath(backstopDir)
	toolsDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating tools directory: %w", err)
	}

	// Check for stale lock (older than 5 minutes)
	if info, err := os.Stat(lockPath); err == nil {
		if time.Since(info.ModTime()) > 5*time.Minute {
			os.Remove(lockPath)
		}
	}

	// Try to create lock file with O_CREATE|O_EXCL for atomicity
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("acquiring semgrep lock: %w", err)
	}
	f.Close()

	cleanup := func() {
		os.Remove(lockPath)
	}
	return cleanup, nil
}
