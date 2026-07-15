package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

var (
	issueNumberRe = regexp.MustCompile(`^(ISSUE-\d{3})-`) // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	issueIDRe     = regexp.MustCompile(`^ISSUE-\d{3}$`)   // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	issueTypes    = map[string]bool{ // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
		"bug": true, "technical-debt": true, "enhancement": true,
		"question": true, "policy-violation": true,
	}
	issueStatuses = map[string]bool{ // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
		"open": true, "ready": true, "in-progress": true, "blocked": true, "closed": true,
		// Retirement terminal states: replaced, canceled (ISSUE-031); obsoleted
		// (ISSUE-048, delivered-then-removed). No "deprecated" for issues.
		"replaced": true, "canceled": true, "obsoleted": true,
	}
	// Statuses that require full traceability (REQ → CLM → tests)
	traceabilityRequired = map[string]bool{ // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
		"ready": true, "in-progress": true, "blocked": true, "closed": true,
	}
	scopeEnum       = map[string]bool{"isolated": true, "contained": true, "cross-cutting": true} // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
	uncertaintyEnum = map[string]bool{"known": true, "exploratory": true, "novel": true}          // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
	riskEnum        = map[string]bool{"safe": true, "moderate": true, "critical": true}           // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
)

// Issue composes base validation with issue-specific checks.
// Issues have full traceability parity with specs — requirements and
// claims are optional when open but required and fully validated on close.
func Issue(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := Base(art, sch)
	var violations []Violation

	// 1. Filename pattern
	filenameOK := false
	if sch.FilenamePattern != "" {
		re, err := regexp.Compile(sch.FilenamePattern)
		if err == nil {
			filenameOK = re.MatchString(art.Filename)
		}
		if !filenameOK {
			violations = append(violations, Violation{
				Rule:     "issue/filename-pattern",
				File:     art.Filename,
				Message:  fmt.Sprintf("filename does not match pattern %s", sch.FilenamePattern),
				Severity: "error",
			})
		}
	}

	// 2. schema_version cross-check
	if sv, ok := art.Metadata["schema_version"]; ok && sv != "" {
		parts := strings.SplitN(sv, "/", 2)
		if len(parts) == 2 && sch.ArtifactType != "" && parts[0] != sch.ArtifactType {
			violations = append(violations, Violation{
				Rule:     "issue/schema-version-mismatch",
				File:     art.Filename,
				Message:  fmt.Sprintf("schema_version type '%s' does not match artifact type '%s'", parts[0], sch.ArtifactType),
				Severity: "error",
			})
		}
	}

	// 3. Issue block validation
	status := ""
	violations = append(violations, validateIssueBlock(art, &status)...)

	// 4. ID/filename consistency
	if filenameOK {
		violations = append(violations, validateIssueIDConsistency(art)...)
	}

	// 5. Status-gated rules (blocked → blocked_by, closed → date)
	violations = append(violations, validateIssueStatusRules(art, status)...)

	// 5b. Retirement fields — replaced-by required+typed when status==replaced
	// (ISSUE-031 DQ-2). No "deprecated" state for issues.
	violations = append(violations, validateRetirementFields(art, status, "issue")...)

	// 6. Complexity block (if present)
	violations = append(violations, validateComplexity(art)...)

	// 7. Capabilities — optional UC-NNN references (validated at all statuses)
	violations = append(violations, validateCapabilities(art.Frontmatter, art.Filename, "issue")...)

	// 8. Requirements + Claims traceability (validated from ready onward)
	violations = append(violations, validateIssueTraceability(art, status)...)

	combined := make([]Violation, 0, len(base.Violations)+len(violations))
	combined = append(combined, base.Violations...)
	combined = append(combined, violations...)
	return ValidationResult{Violations: combined}
}

// validateIssueBlock checks the required issue.* frontmatter fields.
func validateIssueBlock(art *artifact.ParsedArtifact, statusOut *string) []Violation {
	var violations []Violation

	issueVal, ok := art.Frontmatter["issue"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "issue/block-required",
			File:     art.Filename,
			Message:  "issue block is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	issue, ok := issueVal.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "issue/block-required",
			File:     art.Filename,
			Message:  "issue block is not a valid map",
			Severity: "error",
		})
		return violations
	}

	// issue.id
	if id, ok := getStringField(issue, "id"); !ok {
		violations = append(violations, Violation{
			Rule:     "issue/id-required",
			File:     art.Filename,
			Message:  "issue.id is required",
			Severity: "error",
		})
	} else if !issueIDRe.MatchString(id) {
		violations = append(violations, Violation{
			Rule:     "issue/id-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("issue.id '%s' must match pattern ISSUE-NNN", id),
			Severity: "error",
		})
	}

	// issue.title
	if _, ok := getStringField(issue, "title"); !ok {
		violations = append(violations, Violation{
			Rule:     "issue/title-required",
			File:     art.Filename,
			Message:  "issue.title is required",
			Severity: "error",
		})
	}

	// issue.type
	if typ, ok := getStringField(issue, "type"); !ok {
		violations = append(violations, Violation{
			Rule:     "issue/type-required",
			File:     art.Filename,
			Message:  "issue.type is required",
			Severity: "error",
		})
	} else if !issueTypes[typ] {
		violations = append(violations, Violation{
			Rule:     "issue/type-enum",
			File:     art.Filename,
			Message:  fmt.Sprintf("issue.type '%s' is not valid (allowed: bug, technical-debt, enhancement, question, policy-violation)", typ),
			Severity: "error",
		})
	}

	// issue.status
	if s, ok := getStringField(issue, "status"); !ok {
		violations = append(violations, Violation{
			Rule:     "issue/status-required",
			File:     art.Filename,
			Message:  "issue.status is required",
			Severity: "error",
		})
	} else if !issueStatuses[s] {
		violations = append(violations, Violation{
			Rule:     "issue/status-enum",
			File:     art.Filename,
			Message:  fmt.Sprintf("issue.status '%s' is not valid (allowed: open, ready, in-progress, blocked, closed, replaced, canceled, obsoleted)", s),
			Severity: "error",
		})
	} else {
		*statusOut = s
	}

	// issue.created
	if created, ok := getStringField(issue, "created"); !ok {
		violations = append(violations, Violation{
			Rule:     "issue/created-required",
			File:     art.Filename,
			Message:  "issue.created is required",
			Severity: "error",
		})
	} else if !dateRe.MatchString(created) {
		violations = append(violations, Violation{
			Rule:     "issue/created-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("issue.created '%s' must be YYYY-MM-DD", created),
			Severity: "error",
		})
	}

	return violations
}

// validateIssueIDConsistency checks that issue.id matches the filename prefix.
func validateIssueIDConsistency(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	issueVal, ok := art.Frontmatter["issue"]
	if !ok {
		return violations
	}
	issue, ok := issueVal.(map[string]interface{})
	if !ok {
		return violations
	}
	id, ok := getStringField(issue, "id")
	if !ok {
		return violations
	}

	m := issueNumberRe.FindStringSubmatch(art.Filename)
	if len(m) > 1 && m[1] != id {
		violations = append(violations, Violation{
			Rule:     "issue/id-filename-mismatch",
			File:     art.Filename,
			Message:  fmt.Sprintf("issue.id '%s' does not match filename prefix '%s'", id, m[1]),
			Severity: "error",
		})
	}

	return violations
}

// validateIssueStatusRules enforces status-gated requirements:
// - blocked → context.blocked_by must be non-empty
// - closed → issue.closed date required
func validateIssueStatusRules(art *artifact.ParsedArtifact, status string) []Violation {
	var violations []Violation

	if status == "blocked" {
		hasBlockedBy := false
		if ctxVal, ok := art.Frontmatter["context"]; ok {
			if ctx, ok := ctxVal.(map[string]interface{}); ok {
				if bbVal, ok := ctx["blocked_by"]; ok {
					if bb, ok := bbVal.([]interface{}); ok && len(bb) > 0 {
						hasBlockedBy = true
					}
				}
			}
		}
		if !hasBlockedBy {
			violations = append(violations, Violation{
				Rule:     "issue/blocked-requires-blocked-by",
				File:     art.Filename,
				Message:  "issues with status 'blocked' must have non-empty context.blocked_by",
				Severity: "error",
			})
		}
	}

	if status == "closed" {
		hasClosed := false
		if issueVal, ok := art.Frontmatter["issue"]; ok {
			if issue, ok := issueVal.(map[string]interface{}); ok {
				if _, ok := getStringField(issue, "closed"); ok {
					hasClosed = true
				}
			}
		}
		if !hasClosed {
			violations = append(violations, Violation{
				Rule:     "issue/closed-requires-date",
				File:     art.Filename,
				Message:  "issues with status 'closed' must have issue.closed date",
				Severity: "error",
			})
		}
	}

	return violations
}

// validateComplexity checks enum values in the optional complexity block.
func validateComplexity(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	compVal, ok := art.Frontmatter["complexity"]
	if !ok {
		return violations
	}

	comp, ok := compVal.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "issue/complexity-format",
			File:     art.Filename,
			Message:  "complexity block is not a valid map",
			Severity: "error",
		})
		return violations
	}

	if scope, ok := getStringField(comp, "scope"); ok && !scopeEnum[scope] {
		violations = append(violations, Violation{
			Rule:     "issue/complexity-scope-enum",
			File:     art.Filename,
			Message:  fmt.Sprintf("complexity.scope '%s' is not valid (allowed: isolated, contained, cross-cutting)", scope),
			Severity: "error",
		})
	}

	if unc, ok := getStringField(comp, "uncertainty"); ok && !uncertaintyEnum[unc] {
		violations = append(violations, Violation{
			Rule:     "issue/complexity-uncertainty-enum",
			File:     art.Filename,
			Message:  fmt.Sprintf("complexity.uncertainty '%s' is not valid (allowed: known, exploratory, novel)", unc),
			Severity: "error",
		})
	}

	if risk, ok := getStringField(comp, "risk"); ok && !riskEnum[risk] {
		violations = append(violations, Violation{
			Rule:     "issue/complexity-risk-enum",
			File:     art.Filename,
			Message:  fmt.Sprintf("complexity.risk '%s' is not valid (allowed: safe, moderate, critical)", risk),
			Severity: "error",
		})
	}

	return violations
}

// validateIssueTraceability validates requirements → claims → tests verification chain.
// At open: requirements and claims are optional (no validation).
// At ready/in-progress/blocked/closed: both are required with full cross-validation.
func validateIssueTraceability(art *artifact.ParsedArtifact, status string) []Violation {
	var violations []Violation

	// Terminal-state exemption (ISSUE-031 CLM-014): a retired issue (replaced/
	// canceled) is not held to REQ→CLM→tests traceability even if a future edit
	// were to add a terminal state to traceabilityRequired.
	if isTerminalStatus(status) {
		return violations
	}

	if !traceabilityRequired[status] {
		return violations
	}

	// A `closed` issue may satisfy traceability by TRACING to the resolving work
	// via ONE of two mutually-exclusive close pointers, instead of re-authoring
	// requirements/claims onto the issue:
	//   - delivered_by (ISSUE-043): a completed backing plan (PLAN-ISSUE-NNN).
	//   - resolved-by (ISSUE-048): a structured ref to a DIRECT fix (a commit/PR
	//     or a typed artifact ref), requiring NEITHER a backing plan NOR a test.
	// Both relaxations fire ONLY at `closed`. A closed issue with NEITHER pointer,
	// and every pre-close status, falls through to the full
	// REQ→CLM→verification→implementation→contracts rigor below (CLM-007).
	if status == "closed" {
		deliveredBy := getFrontmatterString(art, "delivered_by")
		_, deliveredPresent := art.Frontmatter["delivered_by"]
		_, resolvedPresent := art.Frontmatter["resolved-by"]

		// Mutual exclusivity (CLM-007): at most ONE close pointer. Carrying both is
		// ambiguous — fail loud, no silent precedence, no double-counting.
		if deliveredPresent && resolvedPresent {
			violations = append(violations, Violation{
				Rule:     "issue/close-pointer-conflict",
				File:     art.Filename,
				Message:  "a closed issue may carry at most one close pointer, not both delivered_by and resolved-by",
				Severity: "error",
			})
			return violations
		}

		if deliveredBy != "" {
			violations = append(violations, validateDeliveredBy(art, deliveredBy, getIssueID(art))...)
			// Minimum standalone content: a delivered_by close must still carry a
			// Resolution section so the issue is independently readable (CLM-008).
			if !hasSection(art, "Resolution") {
				violations = append(violations, Violation{
					Rule:     "issue/delivered-by-resolution-required",
					File:     art.Filename,
					Message:  "a delivered_by close must still include a '## Resolution' section (minimum standalone content)",
					Severity: "error",
				})
			}
			// Skip the own-REQ/CLM/verification/implementation/contracts chain —
			// the completed backing plan carries the delivered claims (CLM-001).
			return violations
		}

		if resolvedPresent {
			// resolved-by close (ISSUE-048): the structured ref must be valid, and
			// the close must still carry a Resolution section for standalone
			// readability. It requires NO backing plan and NO mandated test.
			violations = append(violations, validateResolvedBy(art, getFrontmatterString(art, "resolved-by"))...)
			if !hasSection(art, "Resolution") {
				violations = append(violations, Violation{
					Rule:     "issue/resolved-by-resolution-required",
					File:     art.Filename,
					Message:  "a resolved-by close must still include a '## Resolution' section (minimum standalone content)",
					Severity: "error",
				})
			}
			// Skip the own-REQ/CLM/verification/implementation/contracts chain — the
			// resolving work named by resolved-by is the record (CLM-004).
			return violations
		}
	}

	// Verification block — level, threshold, test_command (required from ready onward)
	violations = append(violations, validateVerification(art, "issue")...)

	// Implementation block — summary, package (required from ready onward)
	violations = append(violations, validateImplementation(art, "issue")...)

	// Requirements (required from ready onward)
	validReqs := make(map[string]bool)
	violations = append(violations, validateIssueRequirements(art, validReqs)...)

	// Claims with full spec parity (required from ready onward)
	violations = append(violations, validateIssueClaims(art, validReqs)...)

	// Contracts — provides/consumes API surface (required from ready onward)
	violations = append(violations, validateContracts(art.Frontmatter, art.Filename, "issue")...)

	return violations
}

// getIssueID returns the issue.id from frontmatter, or "" when absent. Used by
// the delivered_by trace to back-match the plan's spec_id against this issue.
func getIssueID(art *artifact.ParsedArtifact) string {
	issueVal, ok := art.Frontmatter["issue"]
	if !ok {
		return ""
	}
	issue, ok := issueVal.(map[string]interface{})
	if !ok {
		return ""
	}
	id, ok := getStringField(issue, "id")
	if !ok {
		return ""
	}
	return id
}

// hasSection reports whether the artifact declares the given H2 section.
func hasSection(art *artifact.ParsedArtifact, name string) bool {
	for _, s := range art.Sections {
		if s == name {
			return true
		}
	}
	return false
}

// validateIssueRequirements checks the requirements array — REQ-NNN pattern,
// optional supports field for bundle traceability.
func validateIssueRequirements(art *artifact.ParsedArtifact, validReqs map[string]bool) []Violation {
	var violations []Violation

	reqsVal, ok := art.Frontmatter["requirements"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "issue/requirements-required",
			File:     art.Filename,
			Message:  "requirements array is required from 'ready' status onward",
			Severity: "error",
		})
		return violations
	}

	reqs, ok := reqsVal.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "issue/requirements-format",
			File:     art.Filename,
			Message:  "requirements is not a valid array",
			Severity: "error",
		})
		return violations
	}

	if len(reqs) == 0 {
		violations = append(violations, Violation{
			Rule:     "issue/requirements-required",
			File:     art.Filename,
			Message:  "requirements array must not be empty from 'ready' status onward",
			Severity: "error",
		})
		return violations
	}

	for i, item := range reqs {
		req, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:     "issue/requirement-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirements[%d] is not a valid map", i),
				Severity: "error",
			})
			continue
		}

		label := fmt.Sprintf("requirements[%d]", i)

		// id
		if idVal, ok := req["id"]; !ok {
			violations = append(violations, Violation{
				Rule:     "issue/requirement-id-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'id'", label),
				Severity: "error",
			})
		} else if id, ok := idVal.(string); !ok || !reqIDRe.MatchString(id) {
			violations = append(violations, Violation{
				Rule:     "issue/requirement-id-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s id '%v' does not match REQ-NNN pattern", label, idVal),
				Severity: "error",
			})
		} else if validReqs[id] {
			violations = append(violations, Violation{
				Rule:     "issue/requirement-id-duplicate",
				File:     art.Filename,
				Message:  fmt.Sprintf("duplicate requirement id '%s'", id),
				Severity: "error",
			})
		} else {
			validReqs[id] = true
		}

		// text
		if textVal, ok := req["text"]; !ok {
			violations = append(violations, Violation{
				Rule:     "issue/requirement-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'text'", label),
				Severity: "error",
			})
		} else if text, ok := textVal.(string); ok && strings.TrimSpace(text) == "" {
			violations = append(violations, Violation{
				Rule:     "issue/requirement-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s 'text' is empty", label),
				Severity: "error",
			})
		}

		// Optional supports — traces back to bundle requirement
		if supVal, ok := req["supports"]; ok {
			sup, ok := supVal.(string)
			if !ok || strings.TrimSpace(sup) == "" {
				violations = append(violations, Violation{
					Rule:     "issue/requirement-supports-format",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s 'supports' is empty", label),
					Severity: "error",
				})
			} else if !supportsRe.MatchString(sup) {
				violations = append(violations, Violation{
					Rule:     "issue/requirement-supports-format",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s 'supports' value '%s' must match format bundle-name:REQ-NNN@MAJOR.MINOR.PATCH", label, sup),
					Severity: "error",
				})
			}
		}
	}

	return violations
}

// validateIssueClaims checks the claims array — full spec parity with CLM-NNN
// pattern, requirement back-references, mandated test names, and coverage check.
func validateIssueClaims(art *artifact.ParsedArtifact, validReqs map[string]bool) []Violation {
	var violations []Violation

	claimsVal, ok := art.Frontmatter["claims"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "issue/claims-required",
			File:     art.Filename,
			Message:  "claims array is required from 'ready' status onward",
			Severity: "error",
		})
		return violations
	}

	claims, ok := claimsVal.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "issue/claims-format",
			File:     art.Filename,
			Message:  "claims is not a valid array",
			Severity: "error",
		})
		return violations
	}

	if len(claims) == 0 {
		violations = append(violations, Violation{
			Rule:     "issue/claims-required",
			File:     art.Filename,
			Message:  "claims array must not be empty from 'ready' status onward",
			Severity: "error",
		})
		return violations
	}

	seenIDs := make(map[string]bool)
	coveredReqs := make(map[string]bool)
	for i, item := range claims {
		claim, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:     "issue/claim-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] is not a valid map", i),
				Severity: "error",
			})
			continue
		}

		label := fmt.Sprintf("claims[%d]", i)

		// id (CLM-NNN)
		if idVal, ok := claim["id"]; !ok {
			violations = append(violations, Violation{
				Rule:     "issue/claim-id-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'id'", label),
				Severity: "error",
			})
		} else if id, ok := idVal.(string); !ok || !claimIDRe.MatchString(id) {
			violations = append(violations, Violation{
				Rule:     "issue/claim-id-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s id '%v' does not match CLM-NNN pattern", label, idVal),
				Severity: "error",
			})
		} else if seenIDs[id] {
			violations = append(violations, Violation{
				Rule:     "issue/claim-id-duplicate",
				File:     art.Filename,
				Message:  fmt.Sprintf("duplicate claim id '%s'", id),
				Severity: "error",
			})
		} else {
			seenIDs[id] = true
		}

		// text
		if textVal, ok := claim["text"]; !ok {
			violations = append(violations, Violation{
				Rule:     "issue/claim-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'text'", label),
				Severity: "error",
			})
		} else if text, ok := textVal.(string); ok && strings.TrimSpace(text) == "" {
			violations = append(violations, Violation{
				Rule:     "issue/claim-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s 'text' is empty", label),
				Severity: "error",
			})
		}

		// requirement back-reference
		if reqVal, ok := claim["requirement"]; !ok {
			violations = append(violations, Violation{
				Rule:     "issue/claim-requirement-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'requirement' field", label),
				Severity: "error",
			})
		} else if reqID, ok := reqVal.(string); !ok || reqID == "" {
			violations = append(violations, Violation{
				Rule:     "issue/claim-requirement-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s 'requirement' is empty", label),
				Severity: "error",
			})
		} else if len(validReqs) > 0 && !validReqs[reqID] {
			violations = append(violations, Violation{
				Rule:     "issue/claim-requirement-invalid",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s references unknown requirement '%s'", label, reqID),
				Severity: "error",
			})
		} else if len(validReqs) > 0 {
			coveredReqs[reqID] = true
		}

		// tests (mandated test names)
		if testsVal, ok := claim["tests"]; !ok {
			violations = append(violations, Violation{
				Rule:     "issue/claim-tests-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'tests'", label),
				Severity: "error",
			})
		} else if tests, ok := testsVal.([]interface{}); !ok || len(tests) == 0 {
			violations = append(violations, Violation{
				Rule:     "issue/claim-tests-empty",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s must have at least one test", label),
				Severity: "error",
			})
		}
	}

	// Requirement coverage — every REQ must have at least one claim
	for reqID := range validReqs {
		if !coveredReqs[reqID] {
			violations = append(violations, Violation{
				Rule:     "issue/requirement-uncovered",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirement '%s' has no claims referencing it", reqID),
				Severity: "error",
			})
		}
	}

	return violations
}
