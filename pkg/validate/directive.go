package validate

import (
	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

// Directive validates a directive artifact against its schema.
func Directive(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	var violations []Violation

	// Directives are lightweight — skip base validation (which expects
	// an H1 markdown heading). The directive's title is in YAML frontmatter,
	// and the body is just a ## Description section.

	// Access nested directive block from full YAML frontmatter
	directiveRaw, hasDirective := art.Frontmatter["directive"]
	if !hasDirective {
		violations = append(violations, Violation{
			Rule:     "directive/directive-required",
			File:     art.Filename,
			Message:  "directive block is missing from frontmatter",
			Severity: "error",
		})
		return ValidationResult{Violations: violations}
	}

	dirMap, ok := directiveRaw.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "directive/directive-required",
			File:     art.Filename,
			Message:  "directive block must be a YAML mapping",
			Severity: "error",
		})
		return ValidationResult{Violations: violations}
	}

	// Check status
	statusVal, hasStatus := dirMap["status"]
	if !hasStatus {
		violations = append(violations, Violation{
			Rule:     "directive/status-required",
			File:     art.Filename,
			Message:  "directive.status is required",
			Severity: "error",
		})
	} else if status, ok := statusVal.(string); ok {
		// Terminal/end-of-life states (ISSUE-031): replaced, canceled. No "deprecated".
		validStatuses := map[string]bool{
			"queued": true, "active": true, "specced": true, "done": true,
			"replaced": true, "canceled": true,
		}
		if !validStatuses[status] {
			violations = append(violations, Violation{
				Rule:     "directive/invalid-status",
				File:     art.Filename,
				Message:  "directive.status must be one of: queued, active, specced, done, replaced, canceled",
				Severity: "error",
			})
		}
		// Retirement fields — replaced-by required+typed when status==replaced
		// (ISSUE-031 DQ-2).
		violations = append(violations, validateRetirementFields(art, status, "directive")...)
	}

	// Check source
	sourceVal, hasSource := dirMap["source"]
	if !hasSource {
		violations = append(violations, Violation{
			Rule:     "directive/source-required",
			File:     art.Filename,
			Message:  "directive.source is required (at least one source artifact)",
			Severity: "error",
		})
	} else if sourceList, ok := sourceVal.([]interface{}); ok && len(sourceList) == 0 {
		violations = append(violations, Violation{
			Rule:     "directive/source-required",
			File:     art.Filename,
			Message:  "directive.source must contain at least one source artifact",
			Severity: "error",
		})
	}

	return ValidationResult{Violations: violations}
}
