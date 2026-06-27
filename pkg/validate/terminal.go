package validate

import (
	"fmt"
	"regexp"

	"github.com/bmanson/backstop-core/pkg/artifact"
)

// replacedByRefPattern is the typed-artifact-reference pattern a replaced-by
// value must match (DQ-2). An array of these is also accepted (multi-absorber).
const replacedByRefPattern = `^(BUNDLE|SPEC|ISSUE|PLAN|DIR)-[0-9]{3}$`

// isTerminalStatus reports whether a status/maturity value is an end-of-life
// state (ISSUE-031, DQ-1). The gate exclusion and the live-work exemptions key
// on this predicate. "deprecated" is only meaningful for spec + bundle, but
// recognizing it here is harmless for the other types because they never carry
// that status. A terminal artifact is retired work and is exempt from the
// live-work completeness/maturity/traceability rules that only apply to active
// work.
func isTerminalStatus(status string) bool {
	switch status {
	case "replaced", "canceled", "deprecated":
		return true
	default:
		return false
	}
}

// validateRetirementFields enforces the conditional retirement-field rules
// (ISSUE-031 DQ-2) for any artifact type, given its resolved status/maturity,
// the frontmatter to read replaced-by/reason from, and the type-specific rule
// prefix (e.g. "spec", "bundle"):
//
//   - status == "replaced": a replaced-by field is REQUIRED. It may be a single
//     string OR a []string; every value must match the typed-ref pattern.
//     Emits <prefix>/replaced-by-required (absent) or <prefix>/replaced-by-malformed
//     (bad ref) at error severity — fail-loud.
//   - status == "canceled" | "deprecated": a free-text "reason" MAY be present
//     but is NEVER required; absence emits nothing.
//
// Non-terminal statuses produce no violations.
func validateRetirementFields(art *artifact.ParsedArtifact, status, rulePrefix string) []Violation {
	var violations []Violation

	if status != "replaced" {
		// canceled / deprecated: reason is optional; nothing to enforce.
		return violations
	}

	refs, present := extractReplacedBy(art.Frontmatter)
	if !present {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/replaced-by-required",
			File:     art.Filename,
			Message:  "status 'replaced' requires a replaced-by reference (typed artifact id, e.g. BUNDLE-011, or an array of them)",
			Severity: "error",
		})
		return violations
	}

	if len(refs) == 0 {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/replaced-by-malformed",
			File:     art.Filename,
			Message:  "replaced-by must be a typed artifact id (BUNDLE/SPEC/ISSUE/PLAN/DIR-NNN) or a non-empty array of them",
			Severity: "error",
		})
		return violations
	}

	replacedByRefRe := regexp.MustCompile(replacedByRefPattern)
	for _, ref := range refs {
		if !replacedByRefRe.MatchString(ref) {
			violations = append(violations, Violation{
				Rule:     rulePrefix + "/replaced-by-malformed",
				File:     art.Filename,
				Message:  fmt.Sprintf("replaced-by value '%s' must match a typed artifact id (BUNDLE|SPEC|ISSUE|PLAN|DIR)-NNN", ref),
				Severity: "error",
			})
		}
	}

	return violations
}

// extractReplacedBy reads the "replaced-by" frontmatter field, returning the
// list of referenced ids and whether the field was present at all. A present
// field that is neither a string nor a list of strings yields (nil-or-empty,
// true) so the caller can flag it as malformed. Non-string array entries are
// NOT dropped — they are stringified so the caller's typed-ref pattern check
// flags them as malformed (every value must be a typed ref; DQ-2).
func extractReplacedBy(fm map[string]interface{}) (refs []string, present bool) {
	val, ok := fm["replaced-by"]
	if !ok {
		return nil, false
	}
	switch v := val.(type) {
	case string:
		return []string{v}, true
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				refs = append(refs, s)
			} else {
				// A non-string entry can never match the typed-ref pattern;
				// surface its value so the malformed check flags it loudly.
				refs = append(refs, fmt.Sprint(item))
			}
		}
		return refs, true
	default:
		return nil, true
	}
}
