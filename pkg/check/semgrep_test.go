package check

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCodeCheck_Semgrep_AutoInstallWhenMissing verifies semgrep auto-installs
// to .backstop/tools/ when not on PATH. (CLM-018)
func TestCodeCheck_Semgrep_AutoInstallWhenMissing(t *testing.T) {
	dir := t.TempDir()
	backstopDir := filepath.Join(dir, ".backstop")
	os.MkdirAll(backstopDir, 0o755)

	installer := &mockSemgrepInstaller{
		lookPathFn: func(name string) (string, error) {
			return "", &execNotFoundError{name: name}
		},
		installFn: func(targetDir, version string) (string, error) {
			// Simulate successful install
			toolsDir := filepath.Join(targetDir, "tools")
			os.MkdirAll(toolsDir, 0o755)
			binPath := filepath.Join(toolsDir, "semgrep")
			os.WriteFile(binPath, []byte("#!/bin/sh\necho 1.50.0"), 0o755)
			return binPath, nil
		},
		versionFn: func(binPath string) (string, error) {
			return "1.50.0", nil
		},
	}

	path, err := ensureSemgrepWith(backstopDir, "1.50.0", installer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty semgrep path")
	}
}

// TestCodeCheck_Semgrep_WrongVersionExitCode2 verifies that installed semgrep
// at wrong version produces a config error (exit code 2), not degraded mode.
// (CLM-019)
func TestCodeCheck_Semgrep_WrongVersionExitCode2(t *testing.T) {
	dir := t.TempDir()
	backstopDir := filepath.Join(dir, ".backstop")
	toolsDir := filepath.Join(backstopDir, "tools")
	os.MkdirAll(toolsDir, 0o755)
	os.WriteFile(filepath.Join(toolsDir, "semgrep"), []byte("fake"), 0o755)

	installer := &mockSemgrepInstaller{
		lookPathFn: func(name string) (string, error) {
			return "", &execNotFoundError{name: name}
		},
		existsAtFn: func(backstopDir string) (string, bool) {
			return filepath.Join(backstopDir, "tools", "semgrep"), true
		},
		versionFn: func(binPath string) (string, error) {
			return "1.40.0", nil // wrong version
		},
	}

	_, err := ensureSemgrepWith(backstopDir, "1.50.0", installer)
	if err == nil {
		t.Fatal("expected error for wrong version, got nil")
	}

	var cfgErr *ConfigError
	if !asConfigError(err, &cfgErr) {
		t.Errorf("expected ConfigError, got %T: %v", err, err)
	}
}

// TestCodeCheck_Semgrep_InstallFailureDegradedMode verifies that when semgrep
// is not installed and auto-install fails, it skips with warning (not exit code 2).
// (CLM-020)
func TestCodeCheck_Semgrep_InstallFailureDegradedMode(t *testing.T) {
	dir := t.TempDir()
	backstopDir := filepath.Join(dir, ".backstop")
	os.MkdirAll(backstopDir, 0o755)

	installer := &mockSemgrepInstaller{
		lookPathFn: func(name string) (string, error) {
			return "", &execNotFoundError{name: name}
		},
		existsAtFn: func(backstopDir string) (string, bool) {
			return "", false
		},
		installFn: func(targetDir, version string) (string, error) {
			return "", &installError{msg: "network timeout"}
		},
	}

	_, err := ensureSemgrepWith(backstopDir, "1.50.0", installer)
	if err == nil {
		t.Fatal("expected error for install failure, got nil")
	}

	var degradedErr *DegradedError
	if !asDegradedError(err, &degradedErr) {
		t.Errorf("expected DegradedError (degraded mode), got %T: %v", err, err)
	}

	// Verify it is NOT a ConfigError
	var cfgErr *ConfigError
	if asConfigError(err, &cfgErr) {
		t.Error("install failure should NOT produce ConfigError (exit 2)")
	}
}

// TestCodeCheck_Semgrep_UsesPathWhenAvailable verifies that when semgrep is
// on PATH, it is used directly without auto-install. (CLM-021)
func TestCodeCheck_Semgrep_UsesPathWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	backstopDir := filepath.Join(dir, ".backstop")

	installCalled := false
	installer := &mockSemgrepInstaller{
		lookPathFn: func(name string) (string, error) {
			return "/usr/local/bin/semgrep", nil
		},
		versionFn: func(binPath string) (string, error) {
			return "1.50.0", nil
		},
		installFn: func(targetDir, version string) (string, error) {
			installCalled = true
			return "", nil
		},
	}

	path, err := ensureSemgrepWith(backstopDir, "1.50.0", installer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/usr/local/bin/semgrep" {
		t.Errorf("path = %q, want /usr/local/bin/semgrep", path)
	}
	if installCalled {
		t.Error("auto-install was called, but semgrep is on PATH")
	}
}

// TestCodeCheck_Semgrep_NoPinSkipsVersionCheck verifies that when no version
// is pinned, version check is skipped.
func TestCodeCheck_Semgrep_NoPinSkipsVersionCheck(t *testing.T) {
	dir := t.TempDir()
	backstopDir := filepath.Join(dir, ".backstop")

	installer := &mockSemgrepInstaller{
		lookPathFn: func(name string) (string, error) {
			return "/usr/local/bin/semgrep", nil
		},
		versionFn: func(binPath string) (string, error) {
			return "1.40.0", nil // any version
		},
	}

	path, err := ensureSemgrepWith(backstopDir, "", installer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/usr/local/bin/semgrep" {
		t.Errorf("path = %q, want /usr/local/bin/semgrep", path)
	}
}

// TestCodeCheck_Semgrep_LockAcquireRelease verifies lock file creation and cleanup.
func TestCodeCheck_Semgrep_LockAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	backstopDir := filepath.Join(dir, ".backstop")

	cleanup, err := acquireSemgrepLock(backstopDir)
	if err != nil {
		t.Fatalf("acquireSemgrepLock: %v", err)
	}

	lockPath := semgrepLockPath(backstopDir)
	if _, statErr := os.Stat(lockPath); statErr != nil {
		t.Errorf("lock file should exist: %v", statErr)
	}

	// Second acquire should fail (lock already held)
	_, err2 := acquireSemgrepLock(backstopDir)
	if err2 == nil {
		t.Error("expected error on second lock acquire")
	}

	// Release
	cleanup()

	// After cleanup, lock file should be gone
	if _, statErr := os.Stat(lockPath); statErr == nil {
		t.Error("lock file should be removed after cleanup")
	}
}

// TestCodeCheck_Semgrep_StaleLockRemoval verifies stale locks are removed.
func TestCodeCheck_Semgrep_StaleLockRemoval(t *testing.T) {
	dir := t.TempDir()
	backstopDir := filepath.Join(dir, ".backstop")
	toolsDir := filepath.Join(backstopDir, "tools")
	os.MkdirAll(toolsDir, 0o755)

	lockPath := semgrepLockPath(backstopDir)
	// Create a stale lock file with old mtime
	os.WriteFile(lockPath, []byte{}, 0o644)
	// Set mtime to 10 minutes ago
	oldTime := time.Now().Add(-10 * time.Minute)
	os.Chtimes(lockPath, oldTime, oldTime)

	// Acquire should succeed because the lock is stale
	cleanup, err := acquireSemgrepLock(backstopDir)
	if err != nil {
		t.Fatalf("acquireSemgrepLock should handle stale lock: %v", err)
	}
	cleanup()
}

// TestCodeCheck_Semgrep_LockPath verifies semgrepLockPath.
func TestCodeCheck_Semgrep_LockPath(t *testing.T) {
	got := semgrepLockPath("/project/.backstop")
	want := "/project/.backstop/tools/.semgrep.lock"
	if got != want {
		t.Errorf("semgrepLockPath = %q, want %q", got, want)
	}
}

// TestCodeCheck_Semgrep_ConfigError_Message verifies ConfigError.Error returns message.
func TestCodeCheck_Semgrep_ConfigError_Message(t *testing.T) {
	err := &ConfigError{Message: "version mismatch"}
	if err.Error() != "version mismatch" {
		t.Errorf("Error() = %q, want %q", err.Error(), "version mismatch")
	}
}

// TestCodeCheck_Semgrep_DegradedError_Message verifies DegradedError.Error returns message.
func TestCodeCheck_Semgrep_DegradedError_Message(t *testing.T) {
	err := &DegradedError{Message: "network failure"}
	if err.Error() != "network failure" {
		t.Errorf("Error() = %q, want %q", err.Error(), "network failure")
	}
}

// TestCodeCheck_DefaultSemgrepInstaller_ExistsAt verifies ExistsAt for real filesystem.
func TestCodeCheck_DefaultSemgrepInstaller_ExistsAt(t *testing.T) {
	d := &DefaultSemgrepInstaller{}

	// Non-existent backstop dir — should return false
	path, exists := d.ExistsAt("/nonexistent/.backstop")
	if exists {
		t.Errorf("expected false for non-existent dir, got path=%q", path)
	}

	// Create a fake semgrep binary
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0o755)
	os.WriteFile(filepath.Join(toolsDir, "semgrep"), []byte("fake"), 0o755)

	path, exists = d.ExistsAt(dir)
	if !exists {
		t.Error("expected true for existing semgrep binary")
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

// TestCodeCheck_DefaultSemgrepInstaller_LookPath verifies LookPath delegates to exec.LookPath.
func TestCodeCheck_DefaultSemgrepInstaller_LookPath(t *testing.T) {
	d := &DefaultSemgrepInstaller{}
	// Look for a binary that definitely exists (go)
	path, err := d.LookPath("go")
	if err != nil {
		t.Skipf("go not on PATH (unexpected in Go test): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for 'go'")
	}

	// Look for a binary that doesn't exist
	_, err = d.LookPath("nonexistent-binary-backstop-test")
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

// --- Test helpers ---

type execNotFoundError struct {
	name string
}

func (e *execNotFoundError) Error() string {
	return "executable file not found: " + e.name
}

type installError struct {
	msg string
}

func (e *installError) Error() string {
	return e.msg
}

type mockSemgrepInstaller struct {
	lookPathFn func(name string) (string, error)
	existsAtFn func(backstopDir string) (string, bool)
	installFn  func(targetDir, version string) (string, error)
	versionFn  func(binPath string) (string, error)
}

func (m *mockSemgrepInstaller) LookPath(name string) (string, error) {
	if m.lookPathFn != nil {
		return m.lookPathFn(name)
	}
	return "", &execNotFoundError{name: name}
}

func (m *mockSemgrepInstaller) ExistsAt(backstopDir string) (string, bool) {
	if m.existsAtFn != nil {
		return m.existsAtFn(backstopDir)
	}
	return "", false
}

func (m *mockSemgrepInstaller) Install(targetDir, version string) (string, error) {
	if m.installFn != nil {
		return m.installFn(targetDir, version)
	}
	return "", &installError{msg: "not implemented"}
}

func (m *mockSemgrepInstaller) Version(binPath string) (string, error) {
	if m.versionFn != nil {
		return m.versionFn(binPath)
	}
	return "", nil
}

// asConfigError checks if err is a *ConfigError.
func asConfigError(err error, target **ConfigError) bool {
	if e, ok := err.(*ConfigError); ok {
		*target = e
		return true
	}
	return false
}

// asDegradedError checks if err is a *DegradedError.
func asDegradedError(err error, target **DegradedError) bool {
	if e, ok := err.(*DegradedError); ok {
		*target = e
		return true
	}
	return false
}
