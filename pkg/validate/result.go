package validate

// Violation represents a single validation failure.
type Violation struct {
	Rule     string
	File     string
	Message  string
	Severity string
}

// ValidationResult holds the outcome of validating an artifact.
type ValidationResult struct {
	Violations []Violation
}

// Pass returns true when there are no violations.
func (r ValidationResult) Pass() bool {
	return len(r.Violations) == 0
}
