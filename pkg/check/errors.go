package check

// ConfigError and DegradedError are the cross-cutting check-pass error contract.
// They were relocated here from the deleted in-process semgrep file
// (ISSUE-018 Section B) because they are used across surviving code —
// parsers.go, manifest.go, registry.go, output.go, check.go, and cmd/backstop
// (via check.ConfigError) — and must outlive the semgrep.go deletion.

// ConfigError signals a hard stop — exit code 2. The cmd layer switches on
// *ConfigError to surface ExitConfigError; it must never be silently swallowed.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string { return e.Message }

// DegradedError signals degraded mode — skip the check with a warning rather
// than failing the whole run.
type DegradedError struct {
	Message string
}

func (e *DegradedError) Error() string { return e.Message }
