package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func baseSchema() *schema.Schema {
	return &schema.Schema{
		RequiredMetadata: []string{"title", "number", "status", "schema-version"},
		RequiredSections: []string{"Context", "Decision"},
	}
}

func TestValidateBase_MissingTitle(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename: "test.md",
		Title:    "",
		Metadata: map[string]string{
			"Number":         "ADR-0001",
			"Status":         "Accepted",
			"Schema-Version": "adr/v1",
		},
		Sections: []string{"Context", "Decision"},
	}

	result := validate.ValidateBase(art, baseSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "base/title-required" {
			found = true
		}
	}
	if !found {
		t.Error("expected violation with Rule 'base/title-required'")
	}
}

func TestValidateBase_MissingMetadata(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename: "test.md",
		Title:    "Test Title",
		Metadata: map[string]string{
			"Number": "ADR-0001",
		},
		Sections: []string{"Context", "Decision"},
	}

	result := validate.ValidateBase(art, baseSchema())
	count := 0
	for _, v := range result.Violations {
		if v.Rule == "base/metadata-required" {
			count++
		}
	}
	// Missing: status, schema-version (title is the H1, not metadata)
	if count < 2 {
		t.Errorf("expected at least 2 metadata violations, got %d", count)
	}
}

func TestValidateBase_MissingSections(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename: "test.md",
		Title:    "Test Title",
		Metadata: map[string]string{
			"Number":         "ADR-0001",
			"Status":         "Accepted",
			"Schema-Version": "adr/v1",
		},
		Sections: []string{"Context"},
	}

	result := validate.ValidateBase(art, baseSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "base/section-required" {
			found = true
		}
	}
	if !found {
		t.Error("expected violation with Rule 'base/section-required' for missing 'Decision'")
	}
}

func TestValidateBase_AllPass(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename: "test.md",
		Title:    "Test Title",
		Metadata: map[string]string{
			"Number":         "ADR-0001",
			"Status":         "Accepted",
			"Schema-Version": "adr/v1",
		},
		Sections: []string{"Context", "Decision"},
	}

	result := validate.ValidateBase(art, baseSchema())
	if !result.Pass() {
		t.Errorf("expected Pass, got violations: %v", result.Violations)
	}
}
