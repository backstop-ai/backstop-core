package check

import (
	"fmt"
	"os/exec"
	"strings"
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

// SemgrepResolver abstracts semgrep discovery on PATH for testability. After the
// SPEC-034 cutover (REQ-008) semgrep is provisioned through the declared
// provisioning model (the engine registry's pinned Provision record, materialized
// by provisionEngines before the gate runs) — pkg/check no longer carries a
// bespoke install ladder. This resolver only LOCATES the provisioned binary and
// verifies its pinned version; it never installs.
type SemgrepResolver interface {
	LookPath(name string) (string, error)
	Version(binPath string) (string, error)
}

// DefaultSemgrepInstaller implements SemgrepResolver via real system calls. The
// name is retained for the existing call sites; after the cutover it resolves
// (LookPath) and version-checks only — the install/probe ladder was retired.
type DefaultSemgrepInstaller struct{}

// LookPath checks if semgrep is on PATH.
func (d *DefaultSemgrepInstaller) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// Version returns the located semgrep version.
func (d *DefaultSemgrepInstaller) Version(binPath string) (string, error) {
	cmd := exec.Command(binPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("checking semgrep version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureSemgrep returns the path to the provisioned semgrep binary, verifying the
// pinned version if one is given. Provisioning is owned by the declared model
// (REQ-008); EnsureSemgrep resolves the already-provisioned binary on PATH and
// fails loud on a version mismatch — it does NOT install. backstopDir is retained
// in the signature for the in-process executor's call site but is no longer a
// search location (the bespoke .backstop/tools ladder was retired).
func EnsureSemgrep(backstopDir string, pinnedVersion string) (string, error) {
	return ensureSemgrepWith(backstopDir, pinnedVersion, &DefaultSemgrepInstaller{})
}

// ensureSemgrepWith is the testable core using an injected resolver. It resolves
// semgrep on PATH (where the declared provisioning materializes it) and verifies
// the pinned version. A missing binary is DegradedMode (skip with a warning, not
// a hard error) so a project without the provisioned binary degrades rather than
// failing the whole gate; a version mismatch is a hard ConfigError.
func ensureSemgrepWith(_ string, pinnedVersion string, resolver SemgrepResolver) (string, error) {
	binPath, err := resolver.LookPath("semgrep")
	if err != nil {
		return "", &DegradedError{
			Message: "semgrep not found on PATH: ensure the declared semgrep provisioning has run",
		}
	}
	if pinnedVersion != "" {
		installedVersion, verErr := resolver.Version(binPath)
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
