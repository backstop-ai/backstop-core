package validate

import (
	"fmt"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

// ValidateBase checks universal artifact requirements: non-empty title,
// required metadata keys present, and required sections present.
func ValidateBase(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	var violations []Violation

	if art.Title == "" {
		violations = append(violations, Violation{
			Rule:     "base/title-required",
			File:     art.Filename,
			Message:  "artifact title is missing",
			Severity: "error",
		})
	}

	for _, key := range sch.RequiredMetadata {
		// "title" is the H1 heading, not a metadata key — checked above
		if strings.EqualFold(key, "title") {
			continue
		}
		found := false
		for k := range art.Metadata {
			if strings.EqualFold(k, key) {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, Violation{
				Rule:     "base/metadata-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("required metadata key '%s' is missing", key),
				Severity: "error",
			})
		}
	}

	for _, section := range sch.RequiredSections {
		found := false
		for _, s := range art.Sections {
			if s == section {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, Violation{
				Rule:     "base/section-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("required section '%s' is missing", section),
				Severity: "error",
			})
		}
	}

	return ValidationResult{Violations: violations}
}
