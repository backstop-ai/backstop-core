package check

import (
	"fmt"
	"testing"
)

// SPEC-034 cutover (REQ-008): EnsureSemgrep's bespoke install ladder (PATH-probe
// -> .backstop/tools probe -> pip Install, with an install lock) was retired.
// semgrep is now provisioned through the declared provisioning model
// (provisionEngines); EnsureSemgrep/ensureSemgrepWith only RESOLVE the provisioned
// binary on PATH and verify the pinned version — they never install. These tests
// cover that surviving resolution behavior.

// TestCodeCheck_Semgrep_UsesPathWhenAvailable verifies that when semgrep is on
// PATH, it is resolved directly and its path returned.
func TestCodeCheck_Semgrep_UsesPathWhenAvailable(t *testing.T) {
	resolver := &mockSemgrepInstaller{
		lookPathFn: func(string) (string, error) { return "/usr/local/bin/semgrep", nil },
		versionFn:  func(string) (string, error) { return "1.50.0", nil },
	}

	path, err := ensureSemgrepWith("/fake/.backstop", "1.50.0", resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/usr/local/bin/semgrep" {
		t.Errorf("path = %q, want /usr/local/bin/semgrep", path)
	}
}

// TestCodeCheck_Semgrep_NoPinSkipsVersionCheck verifies that when no version is
// pinned, the version check is skipped and the resolved path is returned.
func TestCodeCheck_Semgrep_NoPinSkipsVersionCheck(t *testing.T) {
	versionCalled := false
	resolver := &mockSemgrepInstaller{
		lookPathFn: func(string) (string, error) { return "/usr/local/bin/semgrep", nil },
		versionFn: func(string) (string, error) {
			versionCalled = true
			return "1.40.0", nil
		},
	}

	path, err := ensureSemgrepWith("/fake/.backstop", "", resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/usr/local/bin/semgrep" {
		t.Errorf("path = %q, want /usr/local/bin/semgrep", path)
	}
	if versionCalled {
		t.Error("version check must be skipped when no version is pinned")
	}
}

// TestCodeCheck_Semgrep_WrongVersionExitCode2 verifies that a provisioned semgrep
// at the wrong version is a fail-loud ConfigError (exit code 2), not degraded
// mode — the version pin is a hard contract.
func TestCodeCheck_Semgrep_WrongVersionExitCode2(t *testing.T) {
	resolver := &mockSemgrepInstaller{
		lookPathFn: func(string) (string, error) { return "/usr/local/bin/semgrep", nil },
		versionFn:  func(string) (string, error) { return "1.40.0", nil }, // wrong version
	}

	_, err := ensureSemgrepWith("/fake/.backstop", "1.50.0", resolver)
	if err == nil {
		t.Fatal("expected error for wrong version, got nil")
	}
	var cfgErr *ConfigError
	if !asConfigError(err, &cfgErr) {
		t.Errorf("expected ConfigError (exit 2), got %T: %v", err, err)
	}
}

// TestCodeCheck_Semgrep_MissingFromPathDegraded verifies that when semgrep is not
// on PATH (the declared provisioning has not materialized it), resolution is a
// DegradedError — skip with a warning, not a hard stop, and definitely not an ad
// hoc install.
func TestCodeCheck_Semgrep_MissingFromPathDegraded(t *testing.T) {
	installAttempted := false
	resolver := &mockSemgrepInstaller{
		lookPathFn: func(name string) (string, error) {
			return "", &execNotFoundError{name: name}
		},
		versionFn: func(string) (string, error) {
			installAttempted = true // a version probe would imply we found/installed something
			return "1.50.0", nil
		},
	}

	_, err := ensureSemgrepWith("/fake/.backstop", "1.50.0", resolver)
	if err == nil {
		t.Fatal("expected a DegradedError when semgrep is absent from PATH, got nil")
	}
	var degradedErr *DegradedError
	if !asDegradedError(err, &degradedErr) {
		t.Errorf("expected DegradedError, got %T: %v", err, err)
	}
	if installAttempted {
		t.Error("retired ladder must not run: no version probe / install when semgrep is absent — it is just degraded")
	}
}

// TestCodeCheck_Semgrep_VersionParseFailureOnPath verifies that when semgrep is on
// PATH but its version cannot be determined, resolution is a DegradedError.
func TestCodeCheck_Semgrep_VersionParseFailureOnPath(t *testing.T) {
	resolver := &mockSemgrepInstaller{
		lookPathFn: func(string) (string, error) { return "/usr/local/bin/semgrep", nil },
		versionFn:  func(string) (string, error) { return "", fmt.Errorf("cannot parse version output") },
	}

	_, err := ensureSemgrepWith("/fake/.backstop", "1.50.0", resolver)
	if err == nil {
		t.Fatal("expected error when version check fails, got nil")
	}
	var degradedErr *DegradedError
	if !asDegradedError(err, &degradedErr) {
		t.Errorf("expected DegradedError, got %T: %v", err, err)
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

// TestCodeCheck_DefaultSemgrepInstaller_LookPath verifies LookPath delegates to
// exec.LookPath.
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

// TestCodeCheck_DefaultSemgrepInstaller_Version verifies Version shells the binary
// with --version and trims the output (and surfaces an error for a missing binary).
func TestCodeCheck_DefaultSemgrepInstaller_Version(t *testing.T) {
	d := &DefaultSemgrepInstaller{}
	if _, err := d.Version("/nonexistent/semgrep-backstop-test"); err == nil {
		t.Error("expected an error resolving the version of a nonexistent binary")
	}
}

// --- Test helpers ---

type execNotFoundError struct {
	name string
}

func (e *execNotFoundError) Error() string {
	return "executable file not found: " + e.name
}

// mockSemgrepInstaller implements SemgrepResolver (LookPath + Version) for the
// post-cutover resolution tests. The bespoke ExistsAt/Install ladder methods were
// retired with the install path.
type mockSemgrepInstaller struct {
	lookPathFn func(name string) (string, error)
	versionFn  func(binPath string) (string, error)
}

func (m *mockSemgrepInstaller) LookPath(name string) (string, error) {
	if m.lookPathFn != nil {
		return m.lookPathFn(name)
	}
	return "", &execNotFoundError{name: name}
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
