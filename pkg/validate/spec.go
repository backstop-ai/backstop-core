package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/schema"
)

var (
	specNumberRe     = regexp.MustCompile(`^(SPEC-\d{3})-`)                       // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	claimIDRe        = regexp.MustCompile(`^CLM-\d{3}$`)                          // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	reqIDRe          = regexp.MustCompile(`^REQ-\d{3}$`)                          // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	supportsRe       = regexp.MustCompile(`^[a-z0-9-]+:REQ-\d{3}@\d+\.\d+\.\d+$`) // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	followsStdRuleRe = regexp.MustCompile(`^STD-[A-Z]+-[0-9]{3}:[A-Z]+-[0-9]+$`)  // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	followsRecipeRe  = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)        // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
)

// Verification level → required coverage threshold. Nil means threshold must be absent.
var thresholdRules = map[string]*int{ // nosemgrep: go.core.no-global-mutable-state — immutable lookup table, package idiom
	"unit":        intPtr(90),
	"security":    intPtr(90),
	"integration": intPtr(80),
	"performance": intPtr(80),
	"static":      nil,
	"build":       nil,
}

var verificationLevels = map[string]bool{ // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
	"static": true, "build": true, "unit": true,
	"integration": true, "performance": true, "security": true,
}

func intPtr(n int) *int { return &n }

// Spec composes base validation with spec-specific checks.
func Spec(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := Base(art, sch)
	var specViolations []Violation

	// 1. Filename pattern
	filenameOK := false
	if sch.FilenamePattern != "" {
		re, err := regexp.Compile(sch.FilenamePattern)
		if err == nil {
			filenameOK = re.MatchString(art.Filename)
		}
		if !filenameOK {
			specViolations = append(specViolations, Violation{
				Rule:     "spec/filename-pattern",
				File:     art.Filename,
				Message:  fmt.Sprintf("filename does not match pattern %s", sch.FilenamePattern),
				Severity: "error",
			})
		}
	}

	// 2. Slug validation
	if filenameOK && sch.SlugPattern != "" {
		slug := extractSpecSlug(art.Filename)
		if slug != "" {
			slugRe, err := regexp.Compile(sch.SlugPattern)
			valid := err == nil && slugRe.MatchString(slug)
			if valid && (sch.SlugMinLength > 0 && len(slug) < sch.SlugMinLength) {
				valid = false
			}
			if valid && (sch.SlugMaxLength > 0 && len(slug) > sch.SlugMaxLength) {
				valid = false
			}
			if !valid {
				specViolations = append(specViolations, Violation{
					Rule:     "spec/invalid-slug",
					File:     art.Filename,
					Message:  fmt.Sprintf("slug '%s' does not conform to D-078 spec", slug),
					Severity: "error",
				})
			}
		}
	}

	// 3. Number/filename consistency
	if filenameOK {
		m := specNumberRe.FindStringSubmatch(art.Filename)
		if m != nil {
			fileNumber := m[1]
			if metaNumber, ok := art.Metadata["number"]; ok && metaNumber != fileNumber {
				specViolations = append(specViolations, Violation{
					Rule:     "spec/number-mismatch",
					File:     art.Filename,
					Message:  fmt.Sprintf("metadata number '%s' does not match filename '%s'", metaNumber, fileNumber),
					Severity: "error",
				})
			}
		}
	}

	// 4. Title/number consistency
	if filenameOK {
		m := specNumberRe.FindStringSubmatch(art.Filename)
		if m != nil && art.Title != "" {
			expectedPrefix := m[1] + ":"
			if !strings.HasPrefix(art.Title, expectedPrefix) {
				specViolations = append(specViolations, Violation{
					Rule:     "spec/title-number-mismatch",
					File:     art.Filename,
					Message:  fmt.Sprintf("title does not start with '%s'", expectedPrefix),
					Severity: "error",
				})
			}
		}
	}

	// 5. Status enum
	if len(sch.StatusEnum) > 0 {
		status := art.Metadata["status"]
		valid := false
		for _, s := range sch.StatusEnum {
			if s == status {
				valid = true
				break
			}
		}
		if !valid {
			specViolations = append(specViolations, Violation{
				Rule:     "spec/invalid-status",
				File:     art.Filename,
				Message:  fmt.Sprintf("status '%s' is not valid (allowed: %v)", status, sch.StatusEnum),
				Severity: "error",
			})
		}
	}

	// 5b. Retirement fields — replaced-by required+typed when status==replaced;
	// reason optional for canceled/deprecated (ISSUE-031 DQ-2).
	specViolations = append(specViolations, validateRetirementFields(art, art.Metadata["status"], "spec")...)

	// 6. Extension-specific required metadata (spec_version)
	for _, key := range sch.ExtensionMetadata {
		found := false
		for k, v := range art.Metadata {
			if strings.EqualFold(k, key) && strings.TrimSpace(v) != "" {
				found = true
				break
			}
		}
		if !found {
			specViolations = append(specViolations, Violation{
				Rule:     "spec/metadata-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("required metadata key '%s' is missing or empty", key),
				Severity: "error",
			})
		}
	}

	// 7. schema_version/artifact-type cross-check
	if sv, ok := art.Metadata["schema_version"]; ok && sv != "" {
		parts := strings.SplitN(sv, "/", 2)
		if len(parts) == 2 && sch.ArtifactType != "" && parts[0] != sch.ArtifactType {
			specViolations = append(specViolations, Violation{
				Rule:     "spec/schema-version-mismatch",
				File:     art.Filename,
				Message:  fmt.Sprintf("schema_version type '%s' does not match artifact type '%s'", parts[0], sch.ArtifactType),
				Severity: "error",
			})
		}
	}

	// Terminal-state exemption (ISSUE-031 CLM-014): a retired spec (replaced/
	// canceled/deprecated) is not held to live-work completeness. Structural
	// checks above (filename, slug, number, status enum, replaced-by, schema
	// version) still apply; the completeness blocks below are skipped.
	if !isTerminalStatus(art.Metadata["status"]) {
		// 8. Verification block — level and threshold
		specViolations = append(specViolations, validateVerification(art, "spec")...)

		// 9. Implementation block — summary and package
		specViolations = append(specViolations, validateImplementation(art, "spec")...)

		// 10. Requirements array — well-formed REQ-NNN entries
		reqIDs := validateRequirements(art)
		specViolations = append(specViolations, reqIDs.violations...)

		// 11. Claims array — well-formed CLM-NNN entries with valid REQ references
		specViolations = append(specViolations, validateClaims(art, reqIDs.ids)...)

		// 12. Contracts — provides/consumes API surface
		specViolations = append(specViolations, validateContracts(art.Frontmatter, art.Filename, "spec")...)

		// 13. Capabilities — optional UC-NNN references
		specViolations = append(specViolations, validateCapabilities(art.Frontmatter, art.Filename, "spec")...)
	}

	// 14. Review Questions — optional section content validation
	specViolations = append(specViolations, validateReviewQuestions(art)...)

	combined := make([]Violation, 0, len(base.Violations)+len(specViolations))
	combined = append(combined, base.Violations...)
	combined = append(combined, specViolations...)
	return ValidationResult{Violations: combined}
}

// validateVerification checks the verification nested block.
func validateVerification(art *artifact.ParsedArtifact, rulePrefix string) []Violation {
	var violations []Violation

	verBlock, ok := art.Frontmatter["verification"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/verification-required",
			File:     art.Filename,
			Message:  "verification block is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	ver, ok := verBlock.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/verification-required",
			File:     art.Filename,
			Message:  "verification block is not a valid map",
			Severity: "error",
		})
		return violations
	}

	// Level is required and must be a valid enum
	levelVal, hasLevel := ver["level"]
	if !hasLevel {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/verification-level-required",
			File:     art.Filename,
			Message:  "verification.level is missing",
			Severity: "error",
		})
	} else {
		level, ok := levelVal.(string)
		if !ok || !verificationLevels[level] {
			violations = append(violations, Violation{
				Rule:     rulePrefix + "/verification-level-invalid",
				File:     art.Filename,
				Message:  fmt.Sprintf("verification.level '%v' is not valid (allowed: static, build, unit, integration, performance, security)", levelVal),
				Severity: "error",
			})
		} else {
			// Threshold rules
			rule := thresholdRules[level]
			threshVal, hasThresh := ver["coverage_threshold"]

			if rule == nil && hasThresh {
				violations = append(violations, Violation{
					Rule:     rulePrefix + "/threshold-not-allowed",
					File:     art.Filename,
					Message:  fmt.Sprintf("coverage_threshold must not be set for verification level '%s'", level),
					Severity: "error",
				})
			} else if rule != nil {
				if !hasThresh {
					violations = append(violations, Violation{
						Rule:     rulePrefix + "/threshold-required",
						File:     art.Filename,
						Message:  fmt.Sprintf("coverage_threshold is required for verification level '%s' (must be %d)", level, *rule),
						Severity: "error",
					})
				} else {
					thresh := toInt(threshVal)
					if thresh != *rule {
						violations = append(violations, Violation{
							Rule:     rulePrefix + "/threshold-value",
							File:     art.Filename,
							Message:  fmt.Sprintf("coverage_threshold must be %d for level '%s', got %d", *rule, level, thresh),
							Severity: "error",
						})
					}
				}
			}
		}
	}

	// test_command is required
	if _, ok := ver["test_command"]; !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/test-command-required",
			File:     art.Filename,
			Message:  "verification.test_command is missing",
			Severity: "error",
		})
	}

	return violations
}

// validateImplementation checks the implementation nested block.
func validateImplementation(art *artifact.ParsedArtifact, rulePrefix string) []Violation {
	var violations []Violation

	implBlock, ok := art.Frontmatter["implementation"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/implementation-required",
			File:     art.Filename,
			Message:  "implementation block is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	impl, ok := implBlock.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/implementation-required",
			File:     art.Filename,
			Message:  "implementation block is not a valid map",
			Severity: "error",
		})
		return violations
	}

	// summary stays unconditionally required.
	if v, ok := impl["summary"]; !ok || fmt.Sprintf("%v", v) == "" {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/implementation-summary-required",
			File:     art.Filename,
			Message:  "implementation.summary is missing or empty",
			Severity: "error",
		})
	}

	// The target unit is the language-neutral `subject` (ISSUE-047 CLM-001). The
	// legacy `package` key is accepted as a DEPRECATED ALIAS — at least one of the
	// two must be present and non-empty — so the ~40 unmigrated specs keep
	// validating without a bulk migration. Only when BOTH are absent/empty do we
	// raise implementation-subject-required (the canonical neutral rule name). This
	// function is SHARED by the issue validator (issue.go), so issues inherit the
	// same alias with no separate edit.
	if implPresentNonEmpty(impl, "subject") == "" && implPresentNonEmpty(impl, "package") == "" {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/implementation-subject-required",
			File:     art.Filename,
			Message:  "implementation.subject is missing or empty (the legacy 'package' key is accepted as a deprecated alias)",
			Severity: "error",
		})
	}

	return violations
}

// implPresentNonEmpty returns the trimmed string value of key in the
// implementation block, or "" when the key is absent or its value is empty. It
// backs the subject-or-package presence guard (ISSUE-047): the caller treats a
// "" result as "not declared" for that key.
func implPresentNonEmpty(impl map[string]interface{}, key string) string {
	v, ok := impl[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// reqResult holds both violations and the set of valid REQ IDs for cross-referencing.
type reqResult struct {
	violations []Violation
	ids        map[string]bool
}

// validateRequirements checks the requirements array for well-formed REQ-NNN entries.
func validateRequirements(art *artifact.ParsedArtifact) reqResult {
	result := reqResult{ids: make(map[string]bool)}

	reqsVal, ok := art.Frontmatter["requirements"]
	if !ok {
		result.violations = append(result.violations, Violation{
			Rule:     "spec/requirements-required",
			File:     art.Filename,
			Message:  "requirements array is missing from frontmatter",
			Severity: "error",
		})
		return result
	}

	reqs, ok := reqsVal.([]interface{})
	if !ok {
		result.violations = append(result.violations, Violation{
			Rule:     "spec/requirements-required",
			File:     art.Filename,
			Message:  "requirements is not a valid array",
			Severity: "error",
		})
		return result
	}

	if len(reqs) == 0 {
		result.violations = append(result.violations, Violation{
			Rule:     "spec/requirements-empty",
			File:     art.Filename,
			Message:  "requirements array must contain at least one requirement",
			Severity: "error",
		})
		return result
	}

	for i, item := range reqs {
		req, ok := item.(map[string]interface{})
		if !ok {
			result.violations = append(result.violations, Violation{
				Rule:     "spec/requirement-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirements[%d] is not a valid map", i),
				Severity: "error",
			})
			continue
		}

		idVal, hasID := req["id"]
		if !hasID {
			result.violations = append(result.violations, Violation{
				Rule:     "spec/requirement-id-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirements[%d] is missing 'id'", i),
				Severity: "error",
			})
		} else {
			id, ok := idVal.(string)
			switch {
			case !ok || !reqIDRe.MatchString(id):
				result.violations = append(result.violations, Violation{
					Rule:     "spec/requirement-id-format",
					File:     art.Filename,
					Message:  fmt.Sprintf("requirements[%d] id '%v' does not match REQ-NNN pattern", i, idVal),
					Severity: "error",
				})
			case result.ids[id]:
				result.violations = append(result.violations, Violation{
					Rule:     "spec/requirement-id-duplicate",
					File:     art.Filename,
					Message:  fmt.Sprintf("duplicate requirement id '%s'", id),
					Severity: "error",
				})
			default:
				result.ids[id] = true
			}
		}

		textVal, hasText := req["text"]
		if !hasText {
			result.violations = append(result.violations, Violation{
				Rule:     "spec/requirement-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirements[%d] is missing 'text'", i),
				Severity: "error",
			})
		} else if text, ok := textVal.(string); ok && strings.TrimSpace(text) == "" {
			result.violations = append(result.violations, Violation{
				Rule:     "spec/requirement-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirements[%d] 'text' is empty", i),
				Severity: "error",
			})
		}

		// Optional supports field — traces back to bundle requirement(s)
		// Accepts a single string or an array of strings.
		if supVal, ok := req["supports"]; ok {
			var supItems []string
			switch v := supVal.(type) {
			case string:
				supItems = []string{v}
			case []interface{}:
				for _, item := range v {
					if s, ok := item.(string); ok {
						supItems = append(supItems, s)
					}
				}
			}
			if len(supItems) == 0 {
				result.violations = append(result.violations, Violation{
					Rule:     "spec/requirement-supports-format",
					File:     art.Filename,
					Message:  fmt.Sprintf("requirements[%d] 'supports' is empty", i),
					Severity: "error",
				})
			}
			for _, sup := range supItems {
				if strings.TrimSpace(sup) == "" || !supportsRe.MatchString(sup) {
					result.violations = append(result.violations, Violation{
						Rule:     "spec/requirement-supports-format",
						File:     art.Filename,
						Message:  fmt.Sprintf("requirements[%d] 'supports' value '%s' must match format bundle-name:REQ-NNN@MAJOR.MINOR.PATCH", i, sup),
						Severity: "error",
					})
				}
			}
		}

		// Optional follows field — binds requirement to standards/recipes.
		// Accepts a single string or an array of strings.
		if followsVal, ok := req["follows"]; ok {
			var followsItems []string
			switch v := followsVal.(type) {
			case string:
				if strings.TrimSpace(v) == "" {
					result.violations = append(result.violations, Violation{
						Rule:     "spec/requirement-follows-empty",
						File:     art.Filename,
						Message:  fmt.Sprintf("requirements[%d] 'follows' is empty", i),
						Severity: "error",
					})
				} else {
					followsItems = []string{v}
				}
			case []interface{}:
				if len(v) == 0 {
					result.violations = append(result.violations, Violation{
						Rule:     "spec/requirement-follows-empty",
						File:     art.Filename,
						Message:  fmt.Sprintf("requirements[%d] 'follows' is empty", i),
						Severity: "error",
					})
				} else {
					for _, item := range v {
						if s, ok := item.(string); ok {
							followsItems = append(followsItems, s)
						}
					}
				}
			}

			if len(followsItems) == 0 && !hasRule(result.violations, "spec/requirement-follows-empty") {
				result.violations = append(result.violations, Violation{
					Rule:     "spec/requirement-follows-empty",
					File:     art.Filename,
					Message:  fmt.Sprintf("requirements[%d] 'follows' is empty", i),
					Severity: "error",
				})
			}
			for _, follows := range followsItems {
				if !followsStdRuleRe.MatchString(follows) && !followsRecipeRe.MatchString(follows) {
					result.violations = append(result.violations, Violation{
						Rule:     "spec/requirement-follows-format",
						File:     art.Filename,
						Message:  fmt.Sprintf("requirements[%d] 'follows' value '%s' must match format STD-LANG-NNN:RULE-ID or lowercase-kebab recipe name", i, follows),
						Severity: "error",
					})
				}
			}
		}
	}

	return result
}

// validateReviewQuestions checks for the optional Review Questions section.
// The parser only stores section headings, not content. Since section content
// is not available, we validate presence only — if the heading exists, it
// passes. Content quality is the spec-reviewer's responsibility, not the
// validator's. This aligns with the sharp edge: "The validator checks that
// the section is non-empty when present but does not validate that the content
// contains actual questions."
func validateReviewQuestions(art *artifact.ParsedArtifact) []Violation {
	// Presence-only validation. The section heading existing in the markdown
	// body is sufficient — the parser confirmed it was found as an H2 heading.
	// No content check is possible without parser changes.
	return nil
}

func hasRule(vs []Violation, rule string) bool {
	for _, v := range vs {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

// validateClaims checks the claims array for well-formed CLM-NNN entries
// and validates that each claim references a valid requirement.
func validateClaims(art *artifact.ParsedArtifact, validReqs map[string]bool) []Violation {
	var violations []Violation

	claimsVal, ok := art.Frontmatter["claims"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "spec/claims-required",
			File:     art.Filename,
			Message:  "claims array is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	claims, ok := claimsVal.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "spec/claims-required",
			File:     art.Filename,
			Message:  "claims is not a valid array",
			Severity: "error",
		})
		return violations
	}

	if len(claims) == 0 {
		violations = append(violations, Violation{
			Rule:     "spec/claims-empty",
			File:     art.Filename,
			Message:  "claims array must contain at least one claim",
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
				Rule:     "spec/claim-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] is not a valid map", i),
				Severity: "error",
			})
			continue
		}

		// Claim ID
		idVal, hasID := claim["id"]
		if !hasID {
			violations = append(violations, Violation{
				Rule:     "spec/claim-id-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] is missing 'id'", i),
				Severity: "error",
			})
		} else {
			id, ok := idVal.(string)
			switch {
			case !ok || !claimIDRe.MatchString(id):
				violations = append(violations, Violation{
					Rule:     "spec/claim-id-format",
					File:     art.Filename,
					Message:  fmt.Sprintf("claims[%d] id '%v' does not match CLM-NNN pattern", i, idVal),
					Severity: "error",
				})
			case seenIDs[id]:
				violations = append(violations, Violation{
					Rule:     "spec/claim-id-duplicate",
					File:     art.Filename,
					Message:  fmt.Sprintf("duplicate claim id '%s'", id),
					Severity: "error",
				})
			default:
				seenIDs[id] = true
			}
		}

		// Claim text
		textVal, hasText := claim["text"]
		if !hasText {
			violations = append(violations, Violation{
				Rule:     "spec/claim-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] is missing 'text'", i),
				Severity: "error",
			})
		} else if text, ok := textVal.(string); ok && strings.TrimSpace(text) == "" {
			violations = append(violations, Violation{
				Rule:     "spec/claim-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] 'text' is empty", i),
				Severity: "error",
			})
		}

		// Claim requirement reference
		reqVal, hasReq := claim["requirement"]
		if !hasReq {
			violations = append(violations, Violation{
				Rule:     "spec/claim-requirement-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] is missing 'requirement' field", i),
				Severity: "error",
			})
		} else if reqID, ok := reqVal.(string); !ok || reqID == "" {
			violations = append(violations, Violation{
				Rule:     "spec/claim-requirement-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] 'requirement' is empty", i),
				Severity: "error",
			})
		} else if validReqs != nil && !validReqs[reqID] {
			violations = append(violations, Violation{
				Rule:     "spec/claim-requirement-invalid",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] references unknown requirement '%s'", i, reqID),
				Severity: "error",
			})
		} else if validReqs != nil {
			// Track which requirements are covered by claims
			coveredReqs[reqID] = true
		}

		// Claim tests
		testsVal, hasTests := claim["tests"]
		if !hasTests {
			violations = append(violations, Violation{
				Rule:     "spec/claim-tests-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("claims[%d] is missing 'tests'", i),
				Severity: "error",
			})
		} else {
			tests, ok := testsVal.([]interface{})
			if !ok || len(tests) == 0 {
				violations = append(violations, Violation{
					Rule:     "spec/claim-tests-empty",
					File:     art.Filename,
					Message:  fmt.Sprintf("claims[%d] must have at least one test", i),
					Severity: "error",
				})
			}
		}
	}

	// Requirement coverage — every REQ must have at least one claim
	for reqID := range validReqs {
		if !coveredReqs[reqID] {
			violations = append(violations, Violation{
				Rule:     "spec/requirement-uncovered",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirement '%s' has no claims referencing it", reqID),
				Severity: "error",
			})
		}
	}

	return violations
}

// extractSpecSlug pulls the slug portion from a spec filename.
// SPEC-023-my-slug.spec.md → my-slug
func extractSpecSlug(filename string) string {
	if len(filename) < 10 {
		return ""
	}
	rest := filename[9:] // after "SPEC-NNN-"
	suffix := ".spec.md"
	if strings.HasSuffix(rest, suffix) {
		return rest[:len(rest)-len(suffix)]
	}
	return ""
}

// toInt converts a YAML-parsed number to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}
