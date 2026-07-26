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
}
