package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func specSchema() *schema.Schema {
	return &schema.Schema{
		ArtifactType:      "spec",
		FilenamePattern:   `^SPEC-[0-9]{3}-[a-z][a-z0-9]*(-[a-z0-9]+)*\.impl\.spec\.md$`,
		SlugPattern:       `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`,
		SlugMinLength:     2,
		SlugMaxLength:     64,
		RequiredMetadata:  []string{"title", "number", "created", "status", "schema_version"},
		ExtensionMetadata: []string{"spec_version"},
		RequiredSections:  []string{"Overview", "Requirements", "Implementation", "Verification"},
		StatusEnum:        []string{"draft", "ready-for-implementation", "implemented"},
	}
}

func validSpecArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "SPEC-023-backstop-base-validator.impl.spec.md",
		Title:    "SPEC-023: Backstop Base Validator",
		Metadata: map[string]string{
			"number":         "SPEC-023",
			"created":        "2026-03-17",
			"status":         "implemented",
			"schema_version": "spec/v1",
			"spec_version":   "1.3.0",
		},
		Frontmatter: map[string]interface{}{
			"number":         "SPEC-023",
			"created":        "2026-03-17",
			"status":         "implemented",
			"schema_version": "spec/v1",
			"spec_version":   "1.3.0",
			"implementation": map[string]interface{}{
				"summary": "Bootstrap validator for backstop artifacts",
				"package": "pkg/artifact, pkg/schema, pkg/validate",
			},
			"verification": map[string]interface{}{
				"level":              "unit",
				"coverage_threshold": 90,
				"test_command":       "go test ./...",
			},
			"requirements": []interface{}{
				map[string]interface{}{
					"id":   "REQ-001",
					"text": "Artifact parser must extract YAML frontmatter metadata",
				},
				map[string]interface{}{
					"id":   "REQ-002",
					"text": "Schema loader must resolve versioned schemas",
				},
			},
			"claims": []interface{}{
				map[string]interface{}{
					"id":          "CLM-001",
					"requirement": "REQ-001",
					"text":        "ParseFile extracts H1 title",
					"tests": []interface{}{
						map[string]interface{}{"test_name": "TestParseFile_ValidADR"},
					},
				},
				map[string]interface{}{
					"id":          "CLM-002",
					"requirement": "REQ-002",
					"text":        "Parse extracts metadata from YAML frontmatter",
					"tests": []interface{}{
						map[string]interface{}{"test_name": "TestParse_ExtractsMetadata"},
					},
				},
			},
		},
		Sections: []string{"Overview", "Requirements", "Implementation", "Verification"},
	}
}

func TestValidateSpec_FullyValid(t *testing.T) {
	result := validate.ValidateSpec(validSpecArtifact(), specSchema())
	if !result.Pass() {
		for _, v := range result.Violations {
			t.Errorf("[%s] %s: %s", v.Severity, v.Rule, v.Message)
		}
	}
}

func TestValidateSpec_InvalidFilename(t *testing.T) {
	art := validSpecArtifact()
	art.Filename = "bad-spec.md"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/filename-pattern" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/filename-pattern' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_NumberMismatch(t *testing.T) {
	art := validSpecArtifact()
	art.Metadata["number"] = "SPEC-999"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/number-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/number-mismatch' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_TitleNumberMismatch(t *testing.T) {
	art := validSpecArtifact()
	art.Title = "SPEC-999: Wrong Number"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/title-number-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/title-number-mismatch' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_InvalidStatus(t *testing.T) {
	art := validSpecArtifact()
	art.Metadata["status"] = "abandoned"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/invalid-status" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/invalid-status' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingSpecVersion(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Metadata, "spec_version")

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/metadata-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/metadata-required' violation for spec_version, got: %v", result.Violations)
	}
}

func TestValidateSpec_SchemaVersionMismatch(t *testing.T) {
	art := validSpecArtifact()
	art.Metadata["schema_version"] = "adr/v2"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/schema-version-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/schema-version-mismatch' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingVerification(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "verification")

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/verification-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/verification-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_InvalidVerificationLevel(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":        "manual",
		"test_command": "go test ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/verification-level-invalid" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/verification-level-invalid' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ThresholdWrongForUnit(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "unit",
		"coverage_threshold": 80,
		"test_command":       "go test ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/threshold-value" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/threshold-value' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ThresholdMissingForUnit(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":        "unit",
		"test_command": "go test ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/threshold-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/threshold-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ThresholdNotAllowedForStatic(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "static",
		"coverage_threshold": 90,
		"test_command":       "backstop lint ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/threshold-not-allowed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/threshold-not-allowed' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingTestCommand(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "unit",
		"coverage_threshold": 90,
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/test-command-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/test-command-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingImplementation(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "implementation")

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/implementation-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/implementation-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingImplementationSummary(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["implementation"] = map[string]interface{}{
		"package": "pkg/validate",
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/implementation-summary-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/implementation-summary-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingImplementationPackage(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["implementation"] = map[string]interface{}{
		"summary": "some summary",
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/implementation-package-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/implementation-package-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingClaims(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "claims")

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claims-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claims-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_EmptyClaims(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claims-empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claims-empty' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimMissingID(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"requirement": "REQ-001",
			"text":        "some claim",
			"tests":       []interface{}{map[string]interface{}{"test_name": "TestFoo"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-id-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-id-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimBadIDFormat(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":          "REQ-001",
			"requirement": "REQ-001",
			"text":        "some claim",
			"tests":       []interface{}{map[string]interface{}{"test_name": "TestFoo"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-id-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-id-format' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimDuplicateID(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":          "CLM-001",
			"requirement": "REQ-001",
			"text":        "first",
			"tests":       []interface{}{map[string]interface{}{"test_name": "TestA"}},
		},
		map[string]interface{}{
			"id":          "CLM-001",
			"requirement": "REQ-001",
			"text":        "duplicate",
			"tests":       []interface{}{map[string]interface{}{"test_name": "TestB"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-id-duplicate" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-id-duplicate' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimMissingText(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":          "CLM-001",
			"requirement": "REQ-001",
			"tests":       []interface{}{map[string]interface{}{"test_name": "TestFoo"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-text-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-text-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimEmptyTests(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":          "CLM-001",
			"requirement": "REQ-001",
			"text":        "some claim",
			"tests":       []interface{}{},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-tests-empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-tests-empty' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_MissingSection(t *testing.T) {
	art := validSpecArtifact()
	art.Sections = []string{"Overview", "Requirements", "Implementation"}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "base/section-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'base/section-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ComposesBaseAndSpec(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename:    "bad-name.md",
		Title:       "",
		Metadata:    map[string]string{},
		Frontmatter: map[string]interface{}{},
		Sections:    []string{},
	}

	result := validate.ValidateSpec(art, specSchema())
	hasBase := false
	hasSpec := false
	for _, v := range result.Violations {
		if v.Rule == "base/title-required" {
			hasBase = true
		}
		if v.Rule == "spec/filename-pattern" {
			hasSpec = true
		}
	}
	if !hasBase {
		t.Error("expected base violation (base/title-required)")
	}
	if !hasSpec {
		t.Error("expected spec violation (spec/filename-pattern)")
	}
}

func TestValidateSpec_VerificationIntegrationThreshold(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "integration",
		"coverage_threshold": 80,
		"test_command":       "go test -tags integration ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	if !result.Pass() {
		t.Errorf("expected pass for integration level with threshold 80, got: %v", result.Violations)
	}
}

func TestValidateSpec_InvalidSlug(t *testing.T) {
	art := validSpecArtifact()
	art.Filename = "SPEC-023-A.impl.spec.md"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/invalid-slug" || v.Rule == "spec/filename-pattern" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected slug or filename violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_VerificationNotAMap(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = "not a map"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/verification-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/verification-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ImplementationNotAMap(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["implementation"] = "not a map"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/implementation-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/implementation-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimsNotAnArray(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = "not an array"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claims-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claims-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimNotAMap(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{"not a map"}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-format' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimEmptyText(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":    "CLM-001",
			"text":  "   ",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestFoo"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-text-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-text-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_VerificationMissingLevel(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"coverage_threshold": 90,
		"test_command":       "go test ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/verification-level-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/verification-level-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_SecurityThreshold(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "security",
		"coverage_threshold": 90,
		"test_command":       "go test -tags security ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	// Filter out only spec violations (not base violations from threshold change)
	for _, v := range result.Violations {
		if v.Rule == "spec/threshold-value" || v.Rule == "spec/threshold-required" {
			t.Errorf("unexpected threshold violation for security/90: %v", v)
		}
	}
}

func TestValidateSpec_BuildLevelNoThreshold(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":        "build",
		"test_command": "go build ./...",
	}

	result := validate.ValidateSpec(art, specSchema())
	for _, v := range result.Violations {
		if v.Rule == "spec/threshold-required" || v.Rule == "spec/threshold-not-allowed" {
			t.Errorf("unexpected threshold violation for build level: %v", v)
		}
	}
}

func TestValidateSpec_MissingRequirements(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "requirements")

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirements-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirements-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_EmptyRequirements(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["requirements"] = []interface{}{}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirements-empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirements-empty' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_RequirementBadIDFormat(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{
			"id":   "BAD-1",
			"text": "some requirement",
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-id-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirement-id-format' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_RequirementDuplicateID(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"id": "REQ-001", "text": "first"},
		map[string]interface{}{"id": "REQ-001", "text": "duplicate"},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-id-duplicate" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirement-id-duplicate' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_RequirementMissingText(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"id": "REQ-001"},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-text-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirement-text-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimMissingRequirement(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":    "CLM-001",
			"text":  "some claim",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestFoo"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-requirement-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-requirement-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_ClaimReferencesInvalidREQ(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":          "CLM-001",
			"requirement": "REQ-999",
			"text":        "some claim",
			"tests":       []interface{}{map[string]interface{}{"test_name": "TestFoo"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/claim-requirement-invalid" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/claim-requirement-invalid' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_UncoveredRequirement(t *testing.T) {
	art := validSpecArtifact()
	// REQ-002 exists but no claim references it
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":          "CLM-001",
			"requirement": "REQ-001",
			"text":        "only covers REQ-001",
			"tests":       []interface{}{map[string]interface{}{"test_name": "TestFoo"}},
		},
	}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-uncovered" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirement-uncovered' violation for REQ-002, got: %v", result.Violations)
	}
}

func TestValidateSpec_RequirementsNotAnArray(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["requirements"] = "not an array"

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirements-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirements-required' violation, got: %v", result.Violations)
	}
}

func TestValidateSpec_RequirementNotAMap(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["requirements"] = []interface{}{"not a map"}

	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirement-format' violation, got: %v", result.Violations)
	}
}

// --- Supports field tests ---

func TestValidateSpec_RequirementValidSupports(t *testing.T) {
	art := validSpecArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	reqs[0].(map[string]interface{})["supports"] = "my-feature:REQ-001"
	result := validate.ValidateSpec(art, specSchema())
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-supports-format" {
			t.Errorf("unexpected supports format violation: %s", v.Message)
		}
	}
}

func TestValidateSpec_RequirementBadSupportsFormat(t *testing.T) {
	art := validSpecArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	reqs[0].(map[string]interface{})["supports"] = "bad format here"
	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-supports-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirement-supports-format' violation")
	}
}

func TestValidateSpec_RequirementEmptySupports(t *testing.T) {
	art := validSpecArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	reqs[0].(map[string]interface{})["supports"] = ""
	result := validate.ValidateSpec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-supports-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/requirement-supports-format' violation for empty supports")
	}
}

func TestValidateSpec_RequirementNoSupports_OK(t *testing.T) {
	art := validSpecArtifact()
	// No supports field — should be fine (it's optional)
	result := validate.ValidateSpec(art, specSchema())
	for _, v := range result.Violations {
		if v.Rule == "spec/requirement-supports-format" {
			t.Errorf("unexpected supports violation when field absent: %s", v.Message)
		}
	}
}
