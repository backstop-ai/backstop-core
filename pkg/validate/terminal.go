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
// that status. "obsoleted" (ISSUE-048) is a delivered-then-removed terminal —
// distinct from replaced (no successor equivalent) and canceled (obsoleted DID
// ship) — and is retired for exemption purposes exactly like the others. A
// terminal artifact is retired work and is exempt from the live-work
// completeness/maturity/traceability rules that only apply to active work.
func isTerminalStatus(status string) bool {
	switch status {
	case "replaced", "canceled", "deprecated", "obsoleted":
		return true
	default:
		return false
	}
}

// validateRetirementFields enforces the conditional retirement-field rules
// (ISSUE-031 DQ-2, ISSUE-048) for any artifact type, given its resolved
// status/maturity, the frontmatter to read the pointer from, and the
// type-specific rule prefix (e.g. "spec", "bundle"):
//
//   - status == "replaced": a replaced-by field is REQUIRED. It may be a single
//     string OR a []string; every value must match the typed-ref pattern.
//     Emits <prefix>/replaced-by-required (absent) or <prefix>/replaced-by-malformed
//     (bad ref) at error severity — fail-loud.
//   - status == "obsoleted": an obsoleted-by field is REQUIRED, with the SAME
//     typed-ref shape as replaced-by. Emits <prefix>/obsoleted-by-required or
//     <prefix>/obsoleted-by-malformed. Obsoleted = delivered-then-removed with no
//     1:1 successor; the pointer names the removing work (ISSUE-048).
//   - status == "canceled" | "deprecated": a free-text "reason" MAY be present
//     but is NEVER required; absence emits nothing.
//
// Non-terminal statuses produce no violations. Both required-pointer terminals
// share validateTypedRetirementRef, so replaced-by and obsoleted-by cannot drift
// apart. The obsoleted-by trust level matches replaced-by: SHAPE-ONLY, no
// existence check (unlike resolved-by's typed-ref existence check).
func validateRetirementFields(art *artifact.ParsedArtifact, status, rulePrefix string) []Violation {
	switch status {
	case "replaced":
		refs, present := extractReplacedBy(art.Frontmatter)
		return validateTypedRetirementRef(art, rulePrefix, "replaced-by", status, refs, present)
	case "obsoleted":
		refs, present := extractObsoletedBy(art.Frontmatter)
		return validateTypedRetirementRef(art, rulePrefix, "obsoleted-by", status, refs, present)
	default:
		// canceled / deprecated: reason is optional; nothing to enforce.
		return nil
	}
}

// validateTypedRetirementRef enforces that a required retirement pointer (field,
// e.g. "replaced-by" or "obsoleted-by") is present and every value is a typed
// artifact id. It emits <prefix>/<field>-required (absent) or
// <prefix>/<field>-malformed (empty/non-string/non-typed) at error severity.
func validateTypedRetirementRef(art *artifact.ParsedArtifact, rulePrefix, field, status string, refs []string, present bool) []Violation {
	var violations []Violation

	if !present {
		return []Violation{{
			Rule:     rulePrefix + "/" + field + "-required",
			File:     art.Filename,
			Message:  fmt.Sprintf("status '%s' requires a %s reference (typed artifact id, e.g. ISSUE-018, or an array of them)", status, field),
			Severity: "error",
		}}
	}

	if len(refs) == 0 {
		return []Violation{{
			Rule:     rulePrefix + "/" + field + "-malformed",
			File:     art.Filename,
			Message:  fmt.Sprintf("%s must be a typed artifact id (BUNDLE/SPEC/ISSUE/PLAN/DIR-NNN) or a non-empty array of them", field),
			Severity: "error",
		}}
	}

	refRe := regexp.MustCompile(replacedByRefPattern)
	for _, ref := range refs {
		if !refRe.MatchString(ref) {
			violations = append(violations, Violation{
				Rule:     rulePrefix + "/" + field + "-malformed",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s value '%s' must match a typed artifact id (BUNDLE|SPEC|ISSUE|PLAN|DIR)-NNN", field, ref),
				Severity: "error",
			})
		}
	}

	return violations
}

// extractReplacedBy reads the "replaced-by" frontmatter field. See extractRefField.
func extractReplacedBy(fm map[string]interface{}) (refs []string, present bool) {
	return extractRefField(fm, "replaced-by")
}

// extractObsoletedBy reads the "obsoleted-by" frontmatter field, mirroring
// extractReplacedBy (ISSUE-048). Same present/malformed semantics.
func extractObsoletedBy(fm map[string]interface{}) (refs []string, present bool) {
	return extractRefField(fm, "obsoleted-by")
}

// extractRefField reads a typed-artifact-ref frontmatter field by key, returning
// the list of referenced ids and whether the field was present at all. A present
// field that is neither a string nor a list of strings yields (nil-or-empty,
// true) so the caller can flag it as malformed. Non-string array entries are
// NOT dropped — they are stringified so the caller's typed-ref pattern check
// flags them as malformed (every value must be a typed ref; DQ-2).
func extractRefField(fm map[string]interface{}, key string) (refs []string, present bool) {
	val, ok := fm[key]
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
