package distribution

// RemediationGenerator abstracts remediation bundle generation.
type RemediationGenerator interface {
	GenerateBundle(projectDir string, violations []string) (string, error)
}

// Scanner abstracts codebase scanning for violations.
type Scanner interface {
	ScanViolations(projectDir, packDir string) ([]string, error)
}

// UpgradeOptions configures the pack upgrade command.
//
// All four dependencies moved to NewUpgradeCommand. The scanner and remediation
// generator in particular were skipped when nil, which let an unwired upgrade
// cross a major version and report zero baselined violations.
type UpgradeOptions struct {
	ProjectDir string
}

// UpgradeResult holds the result of a pack upgrade operation.
type UpgradeResult struct {
	OldVersion          string `json:"old_version"`
	NewVersion          string `json:"new_version"`
	ContentHash         string `json:"content_hash"`
	RemediationBundle   string `json:"remediation_bundle"`
	BaselinedViolations int    `json:"baselined_violations"`
	// Warnings carries diagnostics the upgrade produced without failing: the
	// coordinate fallback (REQ-005) and the divergence diagnostic (REQ-006). Like
	// UpdateResult.Warnings it did not exist before SPEC-056 (REQ-011).
	Warnings []string `json:"warnings,omitempty"`
}
