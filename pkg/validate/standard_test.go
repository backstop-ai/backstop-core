package validate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

// --- Helpers ---

func validStandardArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "STD-GO-001-go-code-standards.standard.md",
		Title:    "Go Code Standards",
		Metadata: map[string]string{
			"title":          "Go Code Standards",
			"number":         "STD-GO-001",
			"created":        "2026-03-29",
			"status":         "active",
			"schema_version": "standard/v1",
		},
		Sections: []string{"Overview", "Rules", "Examples"},
		Frontmatter: map[string]interface{}{
			"title":          "Go Code Standards",
			"number":         "STD-GO-001",
			"created":        "2026-03-29",
			"status":         "active",
			"schema_version": "standard/v1",
			"language":       "go",
			"pack":           "go",
			"scope":          "language",
			"rules": []interface{}{
				map[string]interface{}{
					"id":          "GO-001",
					"name":        "max-file-length",
					"category":    "structure",
					"severity":    "error",
					"description": "Go source files must not exceed 500 lines",
					"detection": map[string]interface{}{
						"strategy":  "metric",
						"metric":    "file_lines",
						"operator":  ">",
						"threshold": 500,
					},
				},
			},
		},
	}
}

func validStandardSchema() *schema.Schema {
	return &schema.Schema{
		ArtifactType:      "standard",
		FilenamePattern:   `^STD-[A-Z]+-\d{3}-[a-z][a-z0-9]*(-[a-z0-9]+)*\.standard\.md$`,
		RequiredMetadata:  []string{"title", "number", "created", "status", "schema_version"},
		ExtensionMetadata: []string{"pack", "scope"},
		RequiredSections:  []string{"Overview", "Rules", "Examples"},
		StatusEnum:        []string{"draft", "active", "deprecated"},
	}
}

func stdAssertHasViolation(t *testing.T, result validate.ValidationResult, rule string) {
	t.Helper()
	for _, v := range result.Violations {
		if v.Rule == rule {
			return
		}
	}
	t.Errorf("expected violation with rule '%s', got none. Violations:", rule)
	for _, v := range result.Violations {
		t.Errorf("  [%s] %s", v.Rule, v.Message)
	}
}

func stdAssertNoViolation(t *testing.T, result validate.ValidationResult, rule string) {
	t.Helper()
	for _, v := range result.Violations {
		if v.Rule == rule {
			t.Errorf("expected no violation with rule '%s', but found: %s", rule, v.Message)
			return
		}
	}
}

// --- 1. Valid standard ---

func TestStandard_FullyValid(t *testing.T) {
	result := validate.Standard(validStandardArtifact(), validStandardSchema())
	if !result.Pass() {
		t.Errorf("expected valid standard to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestStandard_FullyValid_AllDetectionStrategies(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "GO-001",
			"name":        "metric-rule",
			"category":    "structure",
			"severity":    "error",
			"description": "A metric rule",
			"detection": map[string]interface{}{
				"strategy": "metric",
				"metric":   "file_lines",
			},
		},
		map[string]interface{}{
			"id":          "GO-002",
			"name":        "pattern-rule",
			"category":    "naming",
			"severity":    "warning",
			"description": "A pattern rule",
			"detection": map[string]interface{}{
				"strategy": "pattern",
				"semgrep":  "var $NAME = ...",
			},
		},
		map[string]interface{}{
			"id":          "GO-003",
			"name":        "regex-rule",
			"category":    "testing",
			"severity":    "info",
			"description": "A regex rule",
			"detection": map[string]interface{}{
				"strategy": "regex",
				"pattern":  `^func Test`,
			},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	if !result.Pass() {
		t.Errorf("expected valid standard with all strategies to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestStandard_ValidWithOptionalFields(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["compliance_tier"] = "baseline"
	rule["fix"] = "Split the file"
	art.Frontmatter["sources"] = []interface{}{
		map[string]interface{}{
			"title": "Effective Go",
			"url":   "https://go.dev/doc/effective_go",
		},
	}
	result := validate.Standard(art, validStandardSchema())
	if !result.Pass() {
		t.Errorf("expected valid standard with optional fields to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

// --- 2. Filename pattern ---

func TestStandard_InvalidFilename(t *testing.T) {
	art := validStandardArtifact()
	art.Filename = "bad-standard.md"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/filename-pattern")
}

func TestStandard_FilenameUppercaseSlug(t *testing.T) {
	art := validStandardArtifact()
	art.Filename = "STD-GO-001-GoCode.standard.md"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/filename-pattern")
}

func TestStandard_FilenameNoNumber(t *testing.T) {
	art := validStandardArtifact()
	art.Filename = "go-code-standards.standard.md"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/filename-pattern")
}

// --- 3. Number format ---

func TestStandard_InvalidNumberFormat(t *testing.T) {
	art := validStandardArtifact()
	art.Metadata["number"] = "GO-001"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/number-format")
}

func TestStandard_NumberMissingDigits(t *testing.T) {
	art := validStandardArtifact()
	art.Metadata["number"] = "STD-GO-1"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/number-format")
}

func TestStandard_NumberLowercase(t *testing.T) {
	art := validStandardArtifact()
	art.Metadata["number"] = "STD-go-001"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/number-format")
}

func TestStandard_EmptyNumber_NoFormatViolation(t *testing.T) {
	art := validStandardArtifact()
	art.Metadata["number"] = ""

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/number-format")
}

// --- 4. Number-filename mismatch ---

func TestStandard_NumberFilenameMismatch(t *testing.T) {
	art := validStandardArtifact()
	art.Metadata["number"] = "STD-GO-999"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/number-filename-mismatch")
}

func TestStandard_NumberFilenameMatch(t *testing.T) {
	art := validStandardArtifact()
	// number=STD-GO-001, filename=STD-GO-001-go-code-standards.standard.md — match
	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/number-filename-mismatch")
}

// --- 5. Status enum ---

func TestStandard_InvalidStatus(t *testing.T) {
	art := validStandardArtifact()
	art.Metadata["status"] = "archived"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/invalid-status")
}

func TestStandard_ValidStatuses(t *testing.T) {
	for _, status := range []string{"draft", "active", "deprecated"} {
		t.Run(status, func(t *testing.T) {
			art := validStandardArtifact()
			art.Metadata["status"] = status
			result := validate.Standard(art, validStandardSchema())
			stdAssertNoViolation(t, result, "standard/invalid-status")
		})
	}
}

func TestStandard_EmptyStatus_NoEnumViolation(t *testing.T) {
	art := validStandardArtifact()
	art.Metadata["status"] = ""

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/invalid-status")
}

// --- 6. Language (optional) ---

func TestStandard_LanguageOptional(t *testing.T) {
	art := validStandardArtifact()
	delete(art.Frontmatter, "language")
	art.Frontmatter["scope"] = "universal"

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/invalid-language")
}

func TestStandard_InvalidLanguage(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["language"] = "rust"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/invalid-language")
}

func TestStandard_ValidLanguages(t *testing.T) {
	for _, lang := range []string{"go", "typescript", "python", "bash"} {
		t.Run(lang, func(t *testing.T) {
			art := validStandardArtifact()
			art.Frontmatter["language"] = lang
			result := validate.Standard(art, validStandardSchema())
			stdAssertNoViolation(t, result, "standard/invalid-language")
		})
	}
}

// --- 6b. Scope (required) ---

func TestStandard_MissingScope(t *testing.T) {
	art := validStandardArtifact()
	delete(art.Frontmatter, "scope")

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/scope-required")
}

func TestStandard_InvalidScope(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["scope"] = "global"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/invalid-scope")
}

func TestStandard_ValidScopes(t *testing.T) {
	for _, scope := range []string{"universal", "language", "framework"} {
		t.Run(scope, func(t *testing.T) {
			art := validStandardArtifact()
			art.Frontmatter["scope"] = scope
			if scope != "language" {
				delete(art.Frontmatter, "language")
			}
			result := validate.Standard(art, validStandardSchema())
			stdAssertNoViolation(t, result, "standard/scope-required")
			stdAssertNoViolation(t, result, "standard/invalid-scope")
		})
	}
}

func TestStandard_ScopeLanguageRequiresLanguageField(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["scope"] = "language"
	delete(art.Frontmatter, "language")

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/scope-language-missing")
}

func TestStandard_ScopeUniversalNoLanguageOK(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["scope"] = "universal"
	delete(art.Frontmatter, "language")

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/scope-language-missing")
}

// --- 7. Pack ---

func TestStandard_MissingPack(t *testing.T) {
	art := validStandardArtifact()
	delete(art.Frontmatter, "pack")

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/pack-required")
}

func TestStandard_PackEmptyString(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["pack"] = ""

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/pack-required")
}

func TestStandard_PackNotString(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["pack"] = 123

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/pack-required")
}

// --- 8. Rules block ---

func TestStandard_MissingRules(t *testing.T) {
	art := validStandardArtifact()
	delete(art.Frontmatter, "rules")

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules-required")
}

func TestStandard_RulesNotArray(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = "not-an-array"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules-format")
}

func TestStandard_RulesEmptyArray(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules-empty")
}

func TestStandard_RuleNotMap(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{"not-a-map"}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-format")
}

func TestStandard_RuleMissingRequiredFields(t *testing.T) {
	fields := []string{"id", "name", "category", "severity", "description", "detection"}
	for _, field := range fields {
		t.Run("missing-"+field, func(t *testing.T) {
			art := validStandardArtifact()
			rule := map[string]interface{}{
				"id":          "GO-001",
				"name":        "max-file-length",
				"category":    "structure",
				"severity":    "error",
				"description": "Go source files must not exceed 500 lines",
				"detection": map[string]interface{}{
					"strategy": "metric",
					"metric":   "file_lines",
				},
			}
			delete(rule, field)
			art.Frontmatter["rules"] = []interface{}{rule}

			result := validate.Standard(art, validStandardSchema())
			expectedRule := "standard/rules[0]-" + field + "-required"
			stdAssertHasViolation(t, result, expectedRule)
		})
	}
}

func TestStandard_RuleEmptyStringFields(t *testing.T) {
	fields := []string{"id", "name", "category", "severity", "description"}
	for _, field := range fields {
		t.Run("empty-"+field, func(t *testing.T) {
			art := validStandardArtifact()
			rule := map[string]interface{}{
				"id":          "GO-001",
				"name":        "max-file-length",
				"category":    "structure",
				"severity":    "error",
				"description": "Go source files must not exceed 500 lines",
				"detection": map[string]interface{}{
					"strategy": "metric",
					"metric":   "file_lines",
				},
			}
			rule[field] = ""
			art.Frontmatter["rules"] = []interface{}{rule}

			result := validate.Standard(art, validStandardSchema())
			expectedRule := "standard/rules[0]-" + field + "-empty"
			stdAssertHasViolation(t, result, expectedRule)
		})
	}
}

// --- Rule ID format ---

func TestStandard_RuleIDInvalidFormat(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["id"] = "go-001" // lowercase, should be uppercase

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-id-format")
}

func TestStandard_RuleIDFormatVariants(t *testing.T) {
	badIDs := []string{"go-001", "GO001", "GO-1", "GO-0001", "123-GO", "GO_001"}
	for _, id := range badIDs {
		t.Run(id, func(t *testing.T) {
			art := validStandardArtifact()
			rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
			rule["id"] = id

			result := validate.Standard(art, validStandardSchema())
			stdAssertHasViolation(t, result, "standard/rule-id-format")
		})
	}
}

func TestStandard_RuleIDValidFormat(t *testing.T) {
	goodIDs := []string{"GO-001", "TS-123", "PY-999", "BASH-042"}
	for _, id := range goodIDs {
		t.Run(id, func(t *testing.T) {
			art := validStandardArtifact()
			rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
			rule["id"] = id

			result := validate.Standard(art, validStandardSchema())
			stdAssertNoViolation(t, result, "standard/rule-id-format")
		})
	}
}

// --- Rule ID duplicate ---

func TestStandard_RuleIDDuplicate(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "GO-001",
			"name":        "first-rule",
			"category":    "structure",
			"severity":    "error",
			"description": "First rule",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "x"},
		},
		map[string]interface{}{
			"id":          "GO-001",
			"name":        "second-rule",
			"category":    "naming",
			"severity":    "warning",
			"description": "Second rule with same ID",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "y"},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-id-duplicate")
}

// --- Rule name duplicate ---

func TestStandard_RuleNameDuplicate(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "GO-001",
			"name":        "same-name",
			"category":    "structure",
			"severity":    "error",
			"description": "First rule",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "x"},
		},
		map[string]interface{}{
			"id":          "GO-002",
			"name":        "same-name",
			"category":    "naming",
			"severity":    "warning",
			"description": "Second rule with same name",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "y"},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-name-duplicate")
}

// --- Invalid category ---

func TestStandard_RuleInvalidCategory(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["category"] = "unknown-category"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-invalid-category")
}

func TestStandard_RuleValidCategories(t *testing.T) {
	categories := []string{
		"structure", "error-handling", "naming", "testing", "security",
		"concurrency", "performance", "imports", "documentation",
	}
	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			art := validStandardArtifact()
			rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
			rule["category"] = cat

			result := validate.Standard(art, validStandardSchema())
			stdAssertNoViolation(t, result, "standard/rule-invalid-category")
		})
	}
}

// --- Invalid severity ---

func TestStandard_RuleInvalidSeverity(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["severity"] = "critical"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-invalid-severity")
}

func TestStandard_RuleValidSeverities(t *testing.T) {
	for _, sev := range []string{"error", "warning", "info"} {
		t.Run(sev, func(t *testing.T) {
			art := validStandardArtifact()
			rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
			rule["severity"] = sev

			result := validate.Standard(art, validStandardSchema())
			stdAssertNoViolation(t, result, "standard/rule-invalid-severity")
		})
	}
}

// --- Invalid compliance tier ---

func TestStandard_RuleInvalidComplianceTier(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["compliance_tier"] = "custom"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-invalid-compliance-tier")
}

func TestStandard_RuleValidComplianceTiers(t *testing.T) {
	for _, tier := range []string{"baseline", "standard", "strict"} {
		t.Run(tier, func(t *testing.T) {
			art := validStandardArtifact()
			rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
			rule["compliance_tier"] = tier

			result := validate.Standard(art, validStandardSchema())
			stdAssertNoViolation(t, result, "standard/rule-invalid-compliance-tier")
		})
	}
}

func TestStandard_RuleNoComplianceTier_OK(t *testing.T) {
	art := validStandardArtifact()
	// The default helper doesn't include compliance_tier — should be fine
	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rule-invalid-compliance-tier")
}

// --- 9. Detection block ---

func TestStandard_DetectionNotMap(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = "not-a-map"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-format")
}

func TestStandard_DetectionMissingStrategy(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"metric": "file_lines",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-strategy-required")
}

func TestStandard_DetectionEmptyStrategy(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-strategy-required")
}

func TestStandard_DetectionInvalidStrategy(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "manual",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-invalid-strategy")
}

// --- Detection: pattern strategy ---

func TestStandard_DetectionPatternWithoutSemgrepOrNote(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "pattern",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-pattern-semgrep")
}

func TestStandard_DetectionPatternWithSemgrep(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "pattern",
		"semgrep":  "var $NAME = ...",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-pattern-semgrep")
}

func TestStandard_DetectionPatternWithNote(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "pattern",
		"note":     "Requires manual review",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-pattern-semgrep")
}

// --- Detection: metric strategy ---

func TestStandard_DetectionMetricWithoutMetricField(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "metric",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-metric-name")
}

func TestStandard_DetectionMetricWithMetricField(t *testing.T) {
	art := validStandardArtifact()
	// Default helper already uses metric strategy with metric field — verify it passes
	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-metric-name")
}

func TestStandard_DetectionMetricFieldNotString(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "metric",
		"metric":   42,
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-metric-name")
}

// --- Detection: regex strategy ---

func TestStandard_DetectionRegexWithoutPattern(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "regex",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-regex-pattern")
}

func TestStandard_DetectionRegexWithPattern(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "regex",
		"pattern":  `^func Test`,
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-regex-pattern")
}

func TestStandard_DetectionRegexPatternNotString(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "regex",
		"pattern":  123,
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-regex-pattern")
}

// --- Delegated strategy ---

func TestStandard_DetectionDelegatedValid(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy":    "delegated",
		"enforced_by": "golangci-lint",
		"rule":        "revive/exported",
	}

	result := validate.Standard(art, validStandardSchema())
	if !result.Pass() {
		t.Errorf("expected pass, got %d violations", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  %s: %s", v.Rule, v.Message)
		}
	}
}

func TestStandard_DetectionDelegatedMissingEnforcedBy(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "delegated",
		"rule":     "revive/exported",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-delegated-enforced-by")
}

func TestStandard_DetectionDelegatedMissingRule(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy":    "delegated",
		"enforced_by": "golangci-lint",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-delegated-rule")
}

func TestStandard_DetectionDelegatedBothMissing(t *testing.T) {
	art := validStandardArtifact()
	rule := art.Frontmatter["rules"].([]interface{})[0].(map[string]interface{})
	rule["detection"] = map[string]interface{}{
		"strategy": "delegated",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-delegated-enforced-by")
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-delegated-rule")
}

// --- 10. Sources block (optional) ---

func TestStandard_SourcesNotArray(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["sources"] = "not-an-array"

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/sources-format")
}

func TestStandard_SourceNotMap(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["sources"] = []interface{}{"not-a-map"}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/source-format")
}

func TestStandard_SourceMissingTitle(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["sources"] = []interface{}{
		map[string]interface{}{
			"url": "https://example.com",
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/source-title-required")
}

func TestStandard_SourceEmptyTitle(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["sources"] = []interface{}{
		map[string]interface{}{
			"title": "",
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/source-title-required")
}

func TestStandard_SourceValid(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["sources"] = []interface{}{
		map[string]interface{}{
			"title": "Effective Go",
			"url":   "https://go.dev/doc/effective_go",
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/sources-format")
	stdAssertNoViolation(t, result, "standard/source-format")
	stdAssertNoViolation(t, result, "standard/source-title-required")
}

func TestStandard_NoSources_OK(t *testing.T) {
	art := validStandardArtifact()
	// No sources key — should be fine
	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/sources-format")
}

func TestStandard_MultipleSources_MixedValidity(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["sources"] = []interface{}{
		map[string]interface{}{
			"title": "Good Source",
		},
		map[string]interface{}{
			"url": "missing-title",
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/sources-format")
	stdAssertHasViolation(t, result, "standard/source-title-required")
}

// --- 11. Composition — base violations compose with standard violations ---

func TestStandard_BaseViolationsCompose(t *testing.T) {
	art := validStandardArtifact()
	art.Sections = []string{"Overview"} // missing "Rules" and "Examples"

	result := validate.Standard(art, validStandardSchema())
	// Should have base section violation
	stdAssertHasViolation(t, result, "base/section-required")
	// Should still pass standard-specific checks too
	stdAssertNoViolation(t, result, "standard/scope-required")
}

func TestStandard_BaseAndStandardViolationsBothPresent(t *testing.T) {
	art := validStandardArtifact()
	art.Sections = []string{"Overview"} // missing required sections (base violation)
	delete(art.Frontmatter, "scope")    // standard-specific violation

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "base/section-required")
	stdAssertHasViolation(t, result, "standard/scope-required")
}

// --- 13. Multiple rules with different indices ---

func TestStandard_MultipleRulesSecondRuleViolation(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "GO-001",
			"name":        "first-rule",
			"category":    "structure",
			"severity":    "error",
			"description": "Valid first rule",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "x"},
		},
		map[string]interface{}{
			"id":          "GO-002",
			"name":        "second-rule",
			"category":    "invalid-cat",
			"severity":    "error",
			"description": "Second rule with bad category",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "y"},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-invalid-category")
}

func TestStandard_RuleNotMapAtIndex1(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "GO-001",
			"name":        "first-rule",
			"category":    "structure",
			"severity":    "error",
			"description": "Valid first rule",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "x"},
		},
		"not-a-map",
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rule-format")
}

// --- 14. Detection for second rule index ---

func TestStandard_DetectionViolationAtIndex1(t *testing.T) {
	art := validStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "GO-001",
			"name":        "first-rule",
			"category":    "structure",
			"severity":    "error",
			"description": "Valid first rule",
			"detection":   map[string]interface{}{"strategy": "metric", "metric": "x"},
		},
		map[string]interface{}{
			"id":          "GO-002",
			"name":        "second-rule",
			"category":    "naming",
			"severity":    "warning",
			"description": "Missing metric in detection",
			"detection":   map[string]interface{}{"strategy": "metric"},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[1]-detection-metric-name")
}

// --- 15. Real file test ---

func TestStandard_RealGoStandard(t *testing.T) {
	root := standardRepoRoot(t)
	stdPath := filepath.Join(root, "standards", "go", "STD-GO-001-go-code-standards.standard.md")

	if _, err := os.Stat(stdPath); os.IsNotExist(err) {
		t.Skipf("real standard file not found at %s", stdPath)
	}

	art, err := artifact.ParseFile(stdPath)
	if err != nil {
		t.Skipf("skipping real file test — ParseFile: %v", err)
	}

	schemaRelPath, err := schema.ResolveSchemaPath(art)
	if err != nil {
		t.Fatalf("ResolveSchemaPath: %v", err)
	}

	artifactsRoot := filepath.Join(root, "artifacts")
	schemaFullPath := filepath.Join(root, schemaRelPath)

	sch, err := schema.LoadArtifactSchema(schemaFullPath, artifactsRoot)
	if err != nil {
		t.Fatalf("LoadArtifactSchema: %v", err)
	}

	result := validate.Standard(art, sch)
	if !result.Pass() {
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s: %s", v.Severity, v.Rule, v.Message)
		}
	}
}

// --- 16. Universal-scope per-rule languages validation ---

func validUniversalStandardArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "STD-CORE-001-universal-standards.standard.md",
		Title:    "Universal Standards",
		Metadata: map[string]string{
			"title":          "Universal Standards",
			"number":         "STD-CORE-001",
			"created":        "2026-03-30",
			"status":         "active",
			"schema_version": "standard/v1",
		},
		Sections: []string{"Overview", "Rules", "Examples"},
		Frontmatter: map[string]interface{}{
			"title":          "Universal Standards",
			"number":         "STD-CORE-001",
			"created":        "2026-03-30",
			"status":         "active",
			"schema_version": "standard/v1",
			"pack":           "core",
			"scope":          "universal",
			"rules": []interface{}{
				map[string]interface{}{
					"id":          "CORE-001",
					"name":        "test-file-exists",
					"category":    "testing",
					"severity":    "error",
					"description": "Every source file must have a test file",
					"detection": map[string]interface{}{
						"strategy": "metric",
						"metric":   "test_file_exists",
					},
				},
			},
		},
	}
}

func TestStandard_UniversalPatternRuleWithoutLanguages(t *testing.T) {
	art := validUniversalStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "CORE-001",
			"name":        "no-todo-comments",
			"category":    "documentation",
			"severity":    "warning",
			"description": "No TODO comments in production code",
			"detection": map[string]interface{}{
				"strategy": "pattern",
				"semgrep":  "// TODO ...",
			},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-universal-languages")
}

func TestStandard_UniversalPatternRuleWithLanguages(t *testing.T) {
	art := validUniversalStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "CORE-001",
			"name":        "no-todo-comments",
			"category":    "documentation",
			"severity":    "warning",
			"description": "No TODO comments in production code",
			"detection": map[string]interface{}{
				"strategy":  "pattern",
				"semgrep":   "// TODO ...",
				"languages": []interface{}{"go", "typescript"},
			},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-universal-languages")
}

func TestStandard_UniversalRegexRuleWithoutLanguages(t *testing.T) {
	art := validUniversalStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "CORE-001",
			"name":        "no-hardcoded-secrets",
			"category":    "security",
			"severity":    "error",
			"description": "No hardcoded secrets",
			"detection": map[string]interface{}{
				"strategy": "regex",
				"pattern":  "(?i)(password|secret|api_key)\\s*=\\s*[\"']",
			},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertHasViolation(t, result, "standard/rules[0]-detection-universal-languages")
}

func TestStandard_UniversalMetricRuleNoLanguagesRequired(t *testing.T) {
	art := validUniversalStandardArtifact()
	// Default artifact already has a metric rule with no languages — should pass
	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-universal-languages")
}

func TestStandard_UniversalDelegatedRuleNoLanguagesRequired(t *testing.T) {
	art := validUniversalStandardArtifact()
	art.Frontmatter["rules"] = []interface{}{
		map[string]interface{}{
			"id":          "CORE-001",
			"name":        "license-check",
			"category":    "documentation",
			"severity":    "warning",
			"description": "License header required",
			"detection": map[string]interface{}{
				"strategy":    "delegated",
				"enforced_by": "license-checker",
				"rule":        "has-header",
			},
		},
	}

	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-universal-languages")
}

func TestStandard_LanguageScopePatternRuleNoLanguagesRequired(t *testing.T) {
	// Language-scope standards have the language at the standard level, not per-rule
	art := validStandardArtifact()
	result := validate.Standard(art, validStandardSchema())
	stdAssertNoViolation(t, result, "standard/rules[0]-detection-universal-languages")
}

// standardRepoRoot walks up from the current working directory to find go.mod.
func standardRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod found)")
		}
		dir = parent
	}
}
