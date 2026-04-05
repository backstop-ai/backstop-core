package scaffold

import (
	"fmt"
	"regexp"
	"strings"
)

// slugPattern is the compiled regex for valid slug format.
var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// sourcePattern matches SPEC-NNN or ISSUE-NNN format.
var sourcePattern = regexp.MustCompile(`^(SPEC|ISSUE)-\d{3}$`)

// ValidateSlug validates that the slug conforms to the required pattern and length.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("missing slug: --slug is required")
	}
	if len(slug) < 2 {
		return fmt.Errorf("slug too short: minimum length is 2, got %d", len(slug))
	}
	if len(slug) > 64 {
		return fmt.Errorf("slug too long: maximum length is 64, got %d", len(slug))
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must match pattern ^[a-z][a-z0-9]*(-[a-z0-9]+)*$", slug)
	}
	return nil
}

// ValidateSource validates that sourceID matches the SPEC-NNN or ISSUE-NNN format.
func ValidateSource(sourceID string) error {
	if sourceID == "" {
		return fmt.Errorf("missing --source: required for plan type")
	}
	if !sourcePattern.MatchString(sourceID) {
		return fmt.Errorf("invalid --source %q: must match SPEC-NNN or ISSUE-NNN", sourceID)
	}
	return nil
}

// ParseSourceKind returns "spec" or "issue" based on the source ID prefix.
func ParseSourceKind(sourceID string) (string, error) {
	if err := ValidateSource(sourceID); err != nil {
		return "", err
	}
	if strings.HasPrefix(sourceID, "SPEC-") {
		return "spec", nil
	}
	// ValidateSource ensures sourceID matches ^(SPEC|ISSUE)-\d{3}$,
	// so if we reach here, it must be ISSUE-prefixed.
	return "issue", nil
}
