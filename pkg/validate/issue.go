package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

var (
	issueNumberRe = regexp.MustCompile(`^(ISSUE-[0-9]{3})-`)
	issueIDRe     = regexp.MustCompile(`^ISSUE-[0-9]{3}$`)
	acIDRe        = regexp.MustCompile(`^AC-[0-9]{3}\.[0-9]+$`)
	issueTypes    = map[string]bool{
		"bug": true, "technical-debt": true, "enhancement": true,
		"question": true, "policy-violation": true,
	}
	issueStatuses = map[string]bool{
		"open": true, "in-progress": true, "blocked": true, "closed": true,
	}
	scopeEnum       = map[string]bool{"isolated": true, "contained": true, "cross-cutting": true}
	uncertaintyEnum = map[string]bool{"known": true, "exploratory": true, "novel": true}
	riskEnum        = map[string]bool{"safe": true, "moderate": true, "critical": true}
)

// ValidateIssue composes base validation with issue-specific checks.
// Issues are lighter-weight than specs — claims are optional but
// validated when the issue is closed.
func ValidateIssue(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := ValidateBase(art, sch)
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
	violations = append(violations, validateIssueBlock(art, filenameOK, &status)...)

	// 4. ID/filename consistency
	if filenameOK {
		violations = append(violations, validateIssueIDConsistency(art)...)
	}

	// 5. Status-gated rules
	violations = append(violations, validateIssueStatusRules(art, status)...)

	// 6. Complexity block (if present)
	violations = append(violations, validateComplexity(art)...)

	// 7. Claims validation (only enforced at close)
	violations = append(violations, validateIssueClaims(art, status)...)

	combined := append(base.Violations, violations...)
	return ValidationResult{Violations: combined}
}

// validateIssueBlock checks the required issue.* frontmatter fields.
func validateIssueBlock(art *artifact.ParsedArtifact, filenameOK bool, statusOut *string) []Violation {
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
			Message:  fmt.Sprintf("issue.status '%s' is not valid (allowed: open, in-progress, blocked, closed)", s),
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
	if m != nil && m[1] != id {
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

// validateIssueClaims checks acceptance criteria claims.
// Claims are optional but validated when the issue is closed.
func validateIssueClaims(art *artifact.ParsedArtifact, status string) []Violation {
	var violations []Violation

	claimsVal, ok := art.Frontmatter["claims"]
	if !ok {
		if status == "closed" {
			violations = append(violations, Violation{
				Rule:     "issue/claims-required-on-close",
				File:     art.Filename,
				Message:  "claims array is required when issue is closed",
				Severity: "error",
			})
		}
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

	if status == "closed" && len(claims) == 0 {
		violations = append(violations, Violation{
			Rule:     "issue/claims-required-on-close",
			File:     art.Filename,
			Message:  "claims array must not be empty when issue is closed",
			Severity: "error",
		})
		return violations
	}

	// Only validate claim contents when closed
	if status != "closed" {
		return violations
	}

	seenIDs := make(map[string]bool)
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

		// claim.id
		if id, ok := getStringField(claim, "id"); !ok {
			violations = append(violations, Violation{
				Rule:     "issue/claim-id-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'id'", label),
				Severity: "error",
			})
		} else if !acIDRe.MatchString(id) {
			violations = append(violations, Violation{
				Rule:     "issue/claim-id-pattern",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s id '%s' must match pattern AC-NNN.N", label, id),
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

		// claim.text
		if _, ok := getStringField(claim, "text"); !ok {
			violations = append(violations, Violation{
				Rule:     "issue/claim-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'text'", label),
				Severity: "error",
			})
		}
	}

	return violations
}
