package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func adrSchema() *schema.Schema {
	return &schema.Schema{
		ArtifactType:      "adr",
		FilenamePattern:   `^ADR-[0-9]{4}-[a-z][a-z0-9]*(-[a-z0-9]+)*\.adr\.md$`,
		SlugPattern:       `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`,
		SlugMinLength:     2,
		SlugMaxLength:     64,
		RequiredMetadata:  []string{"title", "number", "created", "status", "schema-version"},
		ExtensionMetadata: []string{"deciders", "decisions"},
		RequiredSections:  []string{"Context", "Decision", "Consequences", "Alternatives Considered", "References"},
		OptionalSections:  []string{"Thesis"},
		StatusEnum:        []string{"Proposed", "Accepted", "Deprecated", "Superseded"},
	}
}

func validADRArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "ADR-0001-test-slug.adr.md",
		Title:    "ADR-0001: Test Title",
		Metadata: map[string]string{
			"Number":         "ADR-0001",
			"Created":        "2026-03-17",
			"Status":         "Accepted",
			"Deciders":       "@bmanson",
			"Decisions":      "D-001",
			"Schema-Version": "adr/v1",
		},
		Sections: []string{"Context", "Decision", "Consequences", "Alternatives Considered", "References"},
	}
}

func TestValidateADR_InvalidFilename(t *testing.T) {
	art := validADRArtifact()
	art.Filename = "bad-name.md"

	result := validate.ValidateADR(art, adrSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "adr/filename-pattern" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'adr/filename-pattern' violation, got: %v", result.Violations)
	}
}

func TestValidateADR_NumberMismatch(t *testing.T) {
	art := validADRArtifact()
	art.Metadata["Number"] = "ADR-0002"

	result := validate.ValidateADR(art, adrSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "adr/number-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'adr/number-mismatch' violation, got: %v", result.Violations)
	}
}

func TestValidateADR_InvalidStatus(t *testing.T) {
	art := validADRArtifact()
	art.Metadata["Status"] = "Draft"

	result := validate.ValidateADR(art, adrSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "adr/invalid-status" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'adr/invalid-status' violation, got: %v", result.Violations)
	}
}

func TestValidateADR_MissingDeciders(t *testing.T) {
	art := validADRArtifact()
	delete(art.Metadata, "Deciders")

	result := validate.ValidateADR(art, adrSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "adr/metadata-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'adr/metadata-required' violation for Deciders, got: %v", result.Violations)
	}
}

func TestValidateADR_MissingDecisions(t *testing.T) {
	art := validADRArtifact()
	delete(art.Metadata, "Decisions")

	result := validate.ValidateADR(art, adrSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "adr/metadata-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'adr/metadata-required' violation for Decisions, got: %v", result.Violations)
	}
}

func TestValidateADR_MissingRequiredSection(t *testing.T) {
	art := validADRArtifact()
	art.Sections = []string{"Context", "Decision", "Consequences", "References"}

	result := validate.ValidateADR(art, adrSchema())
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

func TestValidateADR_FullyValid(t *testing.T) {
	result := validate.ValidateADR(validADRArtifact(), adrSchema())
	if !result.Pass() {
		t.Errorf("expected Pass, got violations: %v", result.Violations)
	}
}

func TestValidateADR_ComposesBaseAndADR(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename: "bad-name.md",
		Title:    "",
		Metadata: map[string]string{},
		Sections: []string{},
	}

	result := validate.ValidateADR(art, adrSchema())
	hasBase := false
	hasADR := false
	for _, v := range result.Violations {
		if v.Rule == "base/title-required" {
			hasBase = true
		}
		if v.Rule == "adr/filename-pattern" {
			hasADR = true
		}
	}
	if !hasBase {
		t.Error("expected base violation (base/title-required)")
	}
	if !hasADR {
		t.Error("expected ADR violation (adr/filename-pattern)")
	}
}

func TestValidateADR_ExtraSectionsAllowed(t *testing.T) {
	art := validADRArtifact()
	art.Sections = append([]string{"Thesis"}, art.Sections...)

	result := validate.ValidateADR(art, adrSchema())
	if !result.Pass() {
		t.Errorf("expected Pass with extra Thesis section, got violations: %v", result.Violations)
	}
}

func TestValidateADR_InvalidSlug(t *testing.T) {
	cases := []struct {
		name     string
		filename string
	}{
		{"uppercase", "ADR-0001-Test.adr.md"},
		{"underscore", "ADR-0001-test_slug.adr.md"},
		{"consecutive hyphens", "ADR-0001--double.adr.md"},
		{"single char", "ADR-0001-a.adr.md"},
		{"starts with digit", "ADR-0001-1start.adr.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			art := validADRArtifact()
			art.Filename = tc.filename

			result := validate.ValidateADR(art, adrSchema())
			found := false
			for _, v := range result.Violations {
				if v.Rule == "adr/invalid-slug" || v.Rule == "adr/filename-pattern" {
					found = true
				}
			}
			if !found {
				t.Errorf("expected slug or filename violation for %q, got: %v", tc.filename, result.Violations)
			}
		})
	}
}

func TestValidateADR_TitleNumberMismatch(t *testing.T) {
	art := validADRArtifact()
	art.Title = "ADR-0002: Wrong Number"

	result := validate.ValidateADR(art, adrSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "adr/title-number-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'adr/title-number-mismatch' violation, got: %v", result.Violations)
	}
}

func TestValidateADR_SchemaVersionTypeMismatch(t *testing.T) {
	art := validADRArtifact()
	art.Metadata["Schema-Version"] = "spec/v1"

	result := validate.ValidateADR(art, adrSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "adr/schema-version-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'adr/schema-version-mismatch' violation, got: %v", result.Violations)
	}
}
