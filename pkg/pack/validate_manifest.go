package pack

import (
	"strconv"
	"strings"
)

// ValidationError describes a manifest validation violation.
type ValidationError struct {
	Field   string
	Message string
	Rule    string
}

// ValidateManifest validates manifest constraints and returns all violations.
func ValidateManifest(m *Manifest) []ValidationError {
	if m == nil {
		return []ValidationError{{
			Field:   "manifest",
			Message: "manifest is required",
			Rule:    "CLM-001",
		}}
	}

	var errs []ValidationError
	errs = append(errs, validateContentTypes(m)...)
	errs = append(errs, validateLayerFields(m)...)
	errs = append(errs, validateSecurityFixtures(m)...)
	errs = append(errs, validateToolConfigTrace(m)...)
	errs = append(errs, validateCoOccurrence(m)...)
	errs = append(errs, validateFixtureProof(m)...)
	errs = append(errs, validateFixtureDirNaming(m)...)
	return errs
}

// ExpectedLayout returns the expected pack layout.
func ExpectedLayout(m *Manifest) []string {
	seen := map[string]struct{}{}
	layout := make([]string, 0, 6)
	add := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		layout = append(layout, path)
	}

	add("pack.yml")
	add("go.mod")
	add("fixtures/rules/")

	hasLayer2 := false
	hasLayer3 := false
	if m != nil {
		for _, rule := range m.Content.Ruleset.Rules {
			if rule.Layer == 2 {
				hasLayer2 = true
			}
			if rule.Layer == 3 {
				hasLayer3 = true
			}
		}
		if m.Archetype == "code" {
			add("scaffolds/")
		}
	}
	if hasLayer2 {
		add("rules/")
	}
	if hasLayer3 {
		add("validators/")
	}
	return layout
}

func validateContentTypes(m *Manifest) []ValidationError {
	var errs []ValidationError

	if m.Archetype == "enforcement" && len(m.Content.Scaffolds) > 0 {
		errs = append(errs, ValidationError{
			Field:   "content.scaffolds",
			Message: "enforcement packs must not define scaffolds",
			Rule:    "CLM-002",
		})
	}
	if m.Archetype == "enforcement" && m.Content.SDK != nil {
		errs = append(errs, ValidationError{
			Field:   "content.sdk",
			Message: "enforcement packs must not define sdk",
			Rule:    "CLM-003",
		})
	}

	if len(m.Content.Ruleset.Rules) == 0 && len(m.Content.Scaffolds) == 0 && m.Content.SDK == nil {
		errs = append(errs, ValidationError{
			Field:   "content",
			Message: "unknown or empty content type",
			Rule:    "CLM-005",
		})
	}

	return errs
}

func validateLayerFields(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, rule := range m.Content.Ruleset.Rules {
		fieldPrefix := "content.ruleset.rules[" + strconv.Itoa(i) + "]"
		switch rule.Layer {
		case 1:
			if rule.RulePath != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".rule_path",
					Message: "layer 1 must not define rule_path",
					Rule:    "CLM-009",
				})
			}
			if rule.Category != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".category",
					Message: "layer 1 must not define category",
					Rule:    "CLM-017",
				})
			}
			if rule.InputScope != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".input_scope",
					Message: "layer 1 must not define input_scope",
					Rule:    "CLM-024",
				})
			}
			if rule.Validator != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".validator",
					Message: "layer 1 must not define validator",
					Rule:    "CLM-026",
				})
			}
		case 2:
			if rule.RulePath == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".rule_path",
					Message: "layer 2 requires rule_path",
					Rule:    "CLM-007",
				})
			}
			if rule.Standard == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".standard",
					Message: "layer 2 requires standard",
					Rule:    "CLM-008",
				})
			}
			if rule.Category != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".category",
					Message: "layer 2 must not define category",
					Rule:    "CLM-018",
				})
			}
			if rule.InputScope != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".input_scope",
					Message: "layer 2 must not define input_scope",
					Rule:    "CLM-025",
				})
			}
			if rule.Validator != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".validator",
					Message: "layer 2 must not define validator",
					Rule:    "CLM-027",
				})
			}
		case 3:
			if rule.RulePath != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".rule_path",
					Message: "layer 3 must not define rule_path",
					Rule:    "CLM-010",
				})
			}
			if rule.Category == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".category",
					Message: "layer 3 requires category",
					Rule:    "CLM-015",
				})
			} else if !isValidLayer3Category(rule.Category) {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".category",
					Message: "layer 3 category must be structural, semantic, or other",
					Rule:    "CLM-016",
				})
			}
			if rule.Category == "other" && strings.TrimSpace(rule.Justification) == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".justification",
					Message: "layer 3 category other requires justification",
					Rule:    "CLM-014",
				})
			}
			if rule.InputScope == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".input_scope",
					Message: "layer 3 requires input_scope",
					Rule:    "CLM-021",
				})
			} else if rule.InputScope != "single_file" && rule.InputScope != "multi_file" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".input_scope",
					Message: "layer 3 input_scope must be single_file or multi_file",
					Rule:    "CLM-023",
				})
			}
			if rule.Validator == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + ".validator",
					Message: "layer 3 requires validator",
					Rule:    "CLM-022",
				})
			}
			if rule.InputScope == "" || rule.Validator == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix,
					Message: "layer 3 requires isolation fields",
					Rule:    "CLM-040",
				})
			}
		}
	}

	return errs
}

func isValidLayer3Category(category string) bool {
	switch category {
	case "structural", "semantic", "other":
		return true
	default:
		return false
	}
}

func validateSecurityFixtures(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, rule := range m.Content.Ruleset.Rules {
		if rule.RiskClass != "security" {
			continue
		}
		if hasBypassFixture(rule.Claims) {
			continue
		}
		errs = append(errs, ValidationError{
			Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims.fixtures",
			Message: "security rules require at least one bypass fixture",
			Rule:    "CLM-029",
		})
	}
	return errs
}

func hasBypassFixture(claims []Claim) bool {
	for _, claim := range claims {
		for _, fixture := range claim.Fixtures.Positive {
			if fixture.BypassAttempt {
				return true
			}
		}
		for _, fixture := range claim.Fixtures.Negative {
			if fixture.BypassAttempt {
				return true
			}
		}
	}
	return false
}

func validateToolConfigTrace(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, tc := range m.ToolConfig {
		if tc.ID == "" && tc.RequiredBy == "" {
			errs = append(errs, ValidationError{
				Field:   "tool_config[" + strconv.Itoa(i) + "]",
				Message: "tool_config requires id or required_by",
				Rule:    "CLM-033",
			})
		}
	}
	return errs
}

func validateCoOccurrence(m *Manifest) []ValidationError {
	var errs []ValidationError

	if m.Archetype == "enforcement" && len(m.Content.Scaffolds) > 0 {
		errs = append(errs, ValidationError{
			Field:   "content.scaffolds",
			Message: "enforcement packs must not include scaffolds",
			Rule:    "CLM-047",
		})
	}
	if m.Archetype == "enforcement" && m.Content.SDK != nil {
		errs = append(errs, ValidationError{
			Field:   "content.sdk",
			Message: "enforcement packs must not include sdk",
			Rule:    "CLM-048",
		})
	}

	if m.Archetype != "code" {
		return errs
	}

	ruleSet := map[string]struct{}{}
	for _, rule := range m.Content.Ruleset.Rules {
		ruleSet[rule.ID] = struct{}{}
	}
	scaffoldSet := map[string]struct{}{}
	for _, scaffold := range m.Content.Scaffolds {
		scaffoldSet[scaffold.ID] = struct{}{}
	}

	for i, scaffold := range m.Content.Scaffolds {
		for _, ruleID := range scaffold.PairsWith.Rules {
			if _, ok := ruleSet[ruleID]; !ok {
				errs = append(errs, ValidationError{
					Field:   "content.scaffolds[" + strconv.Itoa(i) + "].pairs_with.rules",
					Message: "scaffold references missing rule",
					Rule:    "CLM-045",
				})
				break
			}
		}
	}

	for i, rule := range m.Content.Ruleset.Rules {
		for _, scaffoldID := range rule.PairsWith.Scaffolds {
			if _, ok := scaffoldSet[scaffoldID]; !ok {
				errs = append(errs, ValidationError{
					Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].pairs_with.scaffolds",
					Message: "rule references missing scaffold",
					Rule:    "CLM-046",
				})
				break
			}
		}
	}

	return errs
}

func validateFixtureProof(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, rule := range m.Content.Ruleset.Rules {
		if len(rule.Claims) == 0 {
			errs = append(errs, ValidationError{
				Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims",
				Message: "rule requires at least one claim",
				Rule:    "CLM-051",
			})
			continue
		}
		for j, claim := range rule.Claims {
			if len(claim.Fixtures.Positive) == 0 || len(claim.Fixtures.Negative) == 0 {
				errs = append(errs, ValidationError{
					Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims[" + strconv.Itoa(j) + "].fixtures",
					Message: "claim requires positive and negative fixtures",
					Rule:    "CLM-049",
				})
			}
		}
	}
	return errs
}

func validateFixtureDirNaming(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, rule := range m.Content.Ruleset.Rules {
		want := "fixtures/rules/" + strings.ToLower(rule.ID) + "/"
		for j, claim := range rule.Claims {
			for k, fixture := range claim.Fixtures.Positive {
				if !strings.HasPrefix(fixture.Path, want) {
					errs = append(errs, ValidationError{
						Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims[" + strconv.Itoa(j) + "].fixtures.positive[" + strconv.Itoa(k) + "]",
						Message: "fixture path must use lowercase rule id directory",
						Rule:    "CLM-035",
					})
				}
			}
			for k, fixture := range claim.Fixtures.Negative {
				if !strings.HasPrefix(fixture.Path, want) {
					errs = append(errs, ValidationError{
						Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims[" + strconv.Itoa(j) + "].fixtures.negative[" + strconv.Itoa(k) + "]",
						Message: "fixture path must use lowercase rule id directory",
						Rule:    "CLM-035",
					})
				}
			}
		}
	}
	return errs
}
