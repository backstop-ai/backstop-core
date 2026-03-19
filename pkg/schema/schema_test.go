package schema_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

// repoRoot walks up from the current working directory to find go.mod.
func repoRoot(t *testing.T) string {
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

func TestLoadSchema_BaseSchema(t *testing.T) {
	root := repoRoot(t)
	sch, err := schema.LoadSchema(filepath.Join(root, "artifacts", "base", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}

	wantKeys := []string{"title", "number", "created", "status", "schema_version"}
	for _, key := range wantKeys {
		found := false
		for _, k := range sch.RequiredMetadata {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RequiredMetadata missing %q", key)
		}
	}

	if sch.Extends != "" {
		t.Errorf("Extends = %q, want empty for base schema", sch.Extends)
	}
}

func TestLoadArtifactSchema_ADR(t *testing.T) {
	root := repoRoot(t)
	artifactsRoot := filepath.Join(root, "artifacts")
	schemaPath := filepath.Join(artifactsRoot, "adr", "v2", "schema.json")

	sch, err := schema.LoadArtifactSchema(schemaPath, artifactsRoot)
	if err != nil {
		t.Fatal(err)
	}

	if sch.ArtifactType != "adr" {
		t.Errorf("ArtifactType = %q, want %q", sch.ArtifactType, "adr")
	}
	if sch.FilenamePattern == "" {
		t.Error("FilenamePattern is empty")
	}
	if sch.SlugPattern != "^[a-z][a-z0-9]*(-[a-z0-9]+)*$" {
		t.Errorf("SlugPattern = %q", sch.SlugPattern)
	}
	if sch.SlugMinLength != 2 {
		t.Errorf("SlugMinLength = %d, want 2", sch.SlugMinLength)
	}
	if sch.SlugMaxLength != 64 {
		t.Errorf("SlugMaxLength = %d, want 64", sch.SlugMaxLength)
	}

	// Base keys should be in RequiredMetadata
	for _, key := range []string{"title", "number", "created", "status", "schema_version"} {
		found := false
		for _, k := range sch.RequiredMetadata {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RequiredMetadata missing base key %q", key)
		}
	}

	// Extension keys should be in ExtensionMetadata, NOT RequiredMetadata
	for _, key := range []string{"deciders", "decisions"} {
		found := false
		for _, k := range sch.ExtensionMetadata {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ExtensionMetadata missing %q", key)
		}
		for _, k := range sch.RequiredMetadata {
			if k == key {
				t.Errorf("RequiredMetadata should NOT contain extension key %q", key)
			}
		}
	}

	wantSections := []string{"Context", "Decision", "Consequences", "Alternatives Considered", "References"}
	if len(sch.RequiredSections) != len(wantSections) {
		t.Fatalf("RequiredSections = %v, want %v", sch.RequiredSections, wantSections)
	}
	for i, s := range wantSections {
		if sch.RequiredSections[i] != s {
			t.Errorf("RequiredSections[%d] = %q, want %q", i, sch.RequiredSections[i], s)
		}
	}

	hasThesis := false
	for _, s := range sch.OptionalSections {
		if s == "Thesis" {
			hasThesis = true
		}
	}
	if !hasThesis {
		t.Error("OptionalSections missing 'Thesis'")
	}

	wantStatus := []string{"Proposed", "Accepted", "Deprecated", "Superseded"}
	if len(sch.StatusEnum) != len(wantStatus) {
		t.Fatalf("StatusEnum = %v, want %v", sch.StatusEnum, wantStatus)
	}
	for i, s := range wantStatus {
		if sch.StatusEnum[i] != s {
			t.Errorf("StatusEnum[%d] = %q, want %q", i, sch.StatusEnum[i], s)
		}
	}
}

func TestResolveSchemaPath_Valid(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{
			"schema_version": "adr/v2",
		},
	}

	path, err := schema.ResolveSchemaPath(art)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("artifacts", "adr", "v2", "schema.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestResolveSchemaPath_MissingVersion(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{},
	}

	_, err := schema.ResolveSchemaPath(art)
	if err == nil {
		t.Error("expected error for missing schema_version, got nil")
	}
}

func TestResolveSchemaPath_InvalidFormat(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{
			"schema_version": "not-valid-format",
		},
	}

	_, err := schema.ResolveSchemaPath(art)
	if err == nil {
		t.Error("expected error for invalid schema_version format, got nil")
	}
}

func TestLoadSchema_FileNotFound(t *testing.T) {
	_, err := schema.LoadSchema("/nonexistent/path/schema.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadSchema_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := schema.LoadSchema(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoadArtifactSchema_NoExtends(t *testing.T) {
	root := repoRoot(t)
	basePath := filepath.Join(root, "artifacts", "base", "schema.json")

	sch, err := schema.LoadArtifactSchema(basePath, filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	// Base schema has no extends — should return as-is with no ExtensionMetadata
	if len(sch.ExtensionMetadata) != 0 {
		t.Errorf("ExtensionMetadata = %v, want empty for base schema", sch.ExtensionMetadata)
	}
}

func TestLoadArtifactSchema_BadBasePath(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "artifacts", "adr", "v2", "schema.json")

	// Point to nonexistent artifacts root so base schema can't be found
	_, err := schema.LoadArtifactSchema(schemaPath, "/nonexistent/artifacts")
	if err == nil {
		t.Error("expected error when base schema not found, got nil")
	}
}
