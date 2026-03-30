package validate

import (
	"fmt"
	"regexp"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

var (
	standardNumberRe = regexp.MustCompile(`^(STD-[A-Z]+-\d{3})-`)
	standardIDRe     = regexp.MustCompile(`^STD-[A-Z]+-\d{3}$`)
	ruleIDRe         = regexp.MustCompile(`^[A-Z]+-\d{3}$`)
	validLanguages   = map[string]bool{
		"go": true, "typescript": true, "python": true,
		"bash": true,
	}
	validScopes = map[string]bool{
		"universal": true, "language": true, "framework": true,
	}
	validCategories = map[string]bool{
		"structure": true, "error-handling": true, "naming": true,
		"testing": true, "security": true, "concurrency": true,
		"performance": true, "imports": true, "documentation": true,
	}
	validSeverities = map[string]bool{
		"error": true, "warning": true, "info": true,
	}
	validDetectionStrategies = map[string]bool{
		"pattern": true, "metric": true, "regex": true, "delegated": true,
	}
	validComplianceTiers = map[string]bool{
		"baseline": true, "standard": true, "strict": true,
	}
	validStatuses = map[string]bool{
		"draft": true, "active": true, "deprecated": true,
	}
)

// Standard composes base validation with standard-specific checks
// for code standard artifacts that define machine-readable rules.
func Standard(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := Base(art, sch)
	var violations []Violation

	// 1. Filename pattern
	if sch.FilenamePattern != "" {
		re, err := regexp.Compile(sch.FilenamePattern)
		if err == nil && !re.MatchString(art.Filename) {
			violations = append(violations, Violation{
				Rule:    "standard/filename-pattern",
				File:    art.Filename,
				Message: fmt.Sprintf("filename does not match pattern %s", sch.FilenamePattern),
			})
		}
	}

	// 2. Number format
	id := art.Metadata["number"]
	if id != "" && !standardIDRe.MatchString(id) {
		violations = append(violations, Violation{
			Rule:    "standard/number-format",
			File:    art.Filename,
			Message: fmt.Sprintf("number %q does not match STD-<LANG>-NNN pattern", id),
		})
	}

	// 3. Number-filename consistency
	if id != "" && art.Filename != "" {
		m := standardNumberRe.FindStringSubmatch(art.Filename)
		if len(m) > 1 && m[1] != id {
			violations = append(violations, Violation{
				Rule:    "standard/number-filename-mismatch",
				File:    art.Filename,
				Message: fmt.Sprintf("number %q does not match filename prefix %q", id, m[1]),
			})
		}
	}

	// 4. Status enum
	status := art.Metadata["status"]
	if status != "" && !validStatuses[status] {
		violations = append(violations, Violation{
			Rule:    "standard/invalid-status",
			File:    art.Filename,
			Message: fmt.Sprintf("status %q is not valid (draft, active, deprecated)", status),
		})
	}

	// 5. Language (optional — omit for language-agnostic standards)
	lang := getFrontmatterString(art, "language")
	if lang != "" && !validLanguages[lang] {
		violations = append(violations, Violation{
			Rule:    "standard/invalid-language",
			File:    art.Filename,
			Message: fmt.Sprintf("language %q is not recognized (go, typescript, python, bash)", lang),
		})
	}

	// 6. Scope (required)
	scope := getFrontmatterString(art, "scope")
	if scope == "" {
		violations = append(violations, Violation{
			Rule:    "standard/scope-required",
			File:    art.Filename,
			Message: "scope is required in frontmatter (universal, language, framework)",
		})
	} else if !validScopes[scope] {
		violations = append(violations, Violation{
			Rule:    "standard/invalid-scope",
			File:    art.Filename,
			Message: fmt.Sprintf("scope %q is not valid (universal, language, framework)", scope),
		})
	}

	// 7. Scope-language consistency: language scope requires language field
	if scope == "language" && lang == "" {
		violations = append(violations, Violation{
			Rule:    "standard/scope-language-missing",
			File:    art.Filename,
			Message: "scope 'language' requires a language field",
		})
	}

	// 8. Pack
	pack := getFrontmatterString(art, "pack")
	if pack == "" {
		violations = append(violations, Violation{
			Rule:    "standard/pack-required",
			File:    art.Filename,
			Message: "pack is required in frontmatter",
		})
	}

	// 9. Rules block
	violations = append(violations, validateRulesBlock(art, art.Filename, scope)...)

	// 10. Sources block (optional but validated if present)
	violations = append(violations, validateSourcesBlock(art, art.Filename)...)

	combined := make([]Violation, 0, len(base.Violations)+len(violations))
	combined = append(combined, base.Violations...)
	combined = append(combined, violations...)
	return ValidationResult{Violations: combined}
}

func validateRulesBlock(art *artifact.ParsedArtifact, filename, scope string) []Violation {
	var violations []Violation

	rulesRaw, ok := art.Frontmatter["rules"]
	if !ok {
		violations = append(violations, Violation{
			Rule:    "standard/rules-required",
			File:    filename,
			Message: "rules block is required in frontmatter",
		})
		return violations
	}

	rulesArr, ok := rulesRaw.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:    "standard/rules-format",
			File:    filename,
			Message: "rules must be an array",
		})
		return violations
	}

	if len(rulesArr) == 0 {
		violations = append(violations, Violation{
			Rule:    "standard/rules-empty",
			File:    filename,
			Message: "rules array must not be empty",
		})
		return violations
	}

	seenIDs := make(map[string]bool)
	seenNames := make(map[string]bool)

	for i, item := range rulesArr {
		rule, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:    "standard/rule-format",
				File:    filename,
				Message: fmt.Sprintf("rules[%d] must be a map", i),
			})
			continue
		}

		label := fmt.Sprintf("rules[%d]", i)
		violations = append(violations, validateSingleRule(rule, filename, label, seenIDs, seenNames, scope)...)
	}

	return violations
}

func validateSingleRule(rule map[string]interface{}, filename, label string, seenIDs, seenNames map[string]bool, scope string) []Violation {
	var violations []Violation

	// Required fields
	for _, field := range []string{"id", "name", "category", "severity", "description", "detection"} {
		val, ok := rule[field]
		if !ok {
			violations = append(violations, Violation{
				Rule:    fmt.Sprintf("standard/%s-%s-required", label, field),
				File:    filename,
				Message: fmt.Sprintf("%s.%s is required", label, field),
			})
			continue
		}
		if field != "detection" {
			str, isStr := val.(string)
			if !isStr || str == "" {
				violations = append(violations, Violation{
					Rule:    fmt.Sprintf("standard/%s-%s-empty", label, field),
					File:    filename,
					Message: fmt.Sprintf("%s.%s must be a non-empty string", label, field),
				})
			}
		}
	}

	// ID format
	if id, ok := rule["id"].(string); ok && id != "" {
		if !ruleIDRe.MatchString(id) {
			violations = append(violations, Violation{
				Rule:    "standard/rule-id-format",
				File:    filename,
				Message: fmt.Sprintf("%s.id %q does not match <PREFIX>-NNN pattern", label, id),
			})
		}
		if seenIDs[id] {
			violations = append(violations, Violation{
				Rule:    "standard/rule-id-duplicate",
				File:    filename,
				Message: fmt.Sprintf("%s.id %q is a duplicate", label, id),
			})
		}
		seenIDs[id] = true
	}

	// Name uniqueness
	if name, ok := rule["name"].(string); ok && name != "" {
		if seenNames[name] {
			violations = append(violations, Violation{
				Rule:    "standard/rule-name-duplicate",
				File:    filename,
				Message: fmt.Sprintf("%s.name %q is a duplicate", label, name),
			})
		}
		seenNames[name] = true
	}

	// Category
	if cat, ok := rule["category"].(string); ok && cat != "" && !validCategories[cat] {
		violations = append(violations, Violation{
			Rule:    "standard/rule-invalid-category",
			File:    filename,
			Message: fmt.Sprintf("%s.category %q is not valid", label, cat),
		})
	}

	// Severity
	if sev, ok := rule["severity"].(string); ok && sev != "" && !validSeverities[sev] {
		violations = append(violations, Violation{
			Rule:    "standard/rule-invalid-severity",
			File:    filename,
			Message: fmt.Sprintf("%s.severity %q is not valid (error, warning, info)", label, sev),
		})
	}

	// Compliance tier (optional)
	if tier, ok := rule["compliance_tier"].(string); ok && tier != "" && !validComplianceTiers[tier] {
		violations = append(violations, Violation{
			Rule:    "standard/rule-invalid-compliance-tier",
			File:    filename,
			Message: fmt.Sprintf("%s.compliance_tier %q is not valid (baseline, standard, strict)", label, tier),
		})
	}

	// Detection block
	violations = append(violations, validateDetectionBlock(rule, filename, label, scope)...)

	return violations
}

func validateDetectionBlock(rule map[string]interface{}, filename, label, scope string) []Violation {
	var violations []Violation

	detRaw, ok := rule["detection"]
	if !ok {
		return violations
	}

	det, ok := detRaw.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:    fmt.Sprintf("standard/%s-detection-format", label),
			File:    filename,
			Message: fmt.Sprintf("%s.detection must be a map", label),
		})
		return violations
	}

	strategy, ok := det["strategy"].(string)
	if !ok || strategy == "" {
		violations = append(violations, Violation{
			Rule:    fmt.Sprintf("standard/%s-detection-strategy-required", label),
			File:    filename,
			Message: fmt.Sprintf("%s.detection.strategy is required", label),
		})
		return violations
	}

	if !validDetectionStrategies[strategy] {
		violations = append(violations, Violation{
			Rule:    fmt.Sprintf("standard/%s-detection-invalid-strategy", label),
			File:    filename,
			Message: fmt.Sprintf("%s.detection.strategy %q is not valid (pattern, metric, regex, delegated)", label, strategy),
		})
		return violations
	}

	switch strategy {
	case "pattern":
		if _, hasSemgrep := det["semgrep"]; !hasSemgrep {
			if _, hasNote := det["note"]; !hasNote {
				violations = append(violations, Violation{
					Rule:    fmt.Sprintf("standard/%s-detection-pattern-semgrep", label),
					File:    filename,
					Message: fmt.Sprintf("%s.detection with strategy 'pattern' requires 'semgrep' or 'note' field", label),
				})
			}
		}
	case "metric":
		if _, hasMetric := det["metric"].(string); !hasMetric {
			violations = append(violations, Violation{
				Rule:    fmt.Sprintf("standard/%s-detection-metric-name", label),
				File:    filename,
				Message: fmt.Sprintf("%s.detection with strategy 'metric' requires 'metric' field", label),
			})
		}
	case "regex":
		if _, hasPattern := det["pattern"].(string); !hasPattern {
			violations = append(violations, Violation{
				Rule:    fmt.Sprintf("standard/%s-detection-regex-pattern", label),
				File:    filename,
				Message: fmt.Sprintf("%s.detection with strategy 'regex' requires 'pattern' field", label),
			})
		}
	case "delegated":
		if _, hasEnforcedBy := det["enforced_by"].(string); !hasEnforcedBy {
			violations = append(violations, Violation{
				Rule:    fmt.Sprintf("standard/%s-detection-delegated-enforced-by", label),
				File:    filename,
				Message: fmt.Sprintf("%s.detection with strategy 'delegated' requires 'enforced_by' field", label),
			})
		}
		if _, hasRule := det["rule"].(string); !hasRule {
			violations = append(violations, Violation{
				Rule:    fmt.Sprintf("standard/%s-detection-delegated-rule", label),
				File:    filename,
				Message: fmt.Sprintf("%s.detection with strategy 'delegated' requires 'rule' field", label),
			})
		}
	}

	// Universal-scope pattern/regex rules must specify per-rule languages
	if scope == "universal" && (strategy == "pattern" || strategy == "regex") {
		if _, hasLanguages := det["languages"]; !hasLanguages {
			violations = append(violations, Violation{
				Rule:    fmt.Sprintf("standard/%s-detection-universal-languages", label),
				File:    filename,
				Message: fmt.Sprintf("%s.detection: universal-scope %s rules must specify 'languages' field", label, strategy),
			})
		}
	}

	return violations
}

func validateSourcesBlock(art *artifact.ParsedArtifact, filename string) []Violation {
	var violations []Violation

	sourcesRaw, ok := art.Frontmatter["sources"]
	if !ok {
		return violations
	}

	sourcesArr, ok := sourcesRaw.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:    "standard/sources-format",
			File:    filename,
			Message: "sources must be an array",
		})
		return violations
	}

	for i, item := range sourcesArr {
		src, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:    "standard/source-format",
				File:    filename,
				Message: fmt.Sprintf("sources[%d] must be a map", i),
			})
			continue
		}

		title, hasTitle := src["title"].(string)
		if !hasTitle || title == "" {
			violations = append(violations, Violation{
				Rule:    "standard/source-title-required",
				File:    filename,
				Message: fmt.Sprintf("sources[%d].title is required", i),
			})
		}
	}

	return violations
}

func getFrontmatterString(art *artifact.ParsedArtifact, key string) string {
	val, ok := art.Frontmatter[key]
	if !ok {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}
