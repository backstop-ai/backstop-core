package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// TestArtifactValidate_Route_Spec verifies that an artifact with
// schema_version "spec/v1" routes to validate.Spec. (CLM-001)
func TestArtifactValidate_Route_Spec(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "spec/v1"},
	}
	fn, err := RouteValidator(art)
	if err != nil {
		t.Fatalf("RouteValidator: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil validator function for spec/v1")
	}
	// Call the validator on a known-bad artifact to verify it produces violations
	badArt := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "spec/v1", "title": "Bad"},
	}
	schPath, _ := schema.ResolveSchemaPath(badArt)
	sch, _ := loadSchemaFromFS(SchemaFS, schPath)
	result := fn(badArt, sch)
	if len(result.Violations) == 0 {
		t.Error("expected violations from known-bad spec artifact")
	}
}

// TestArtifactValidate_Route_Plan verifies that an artifact with
// schema_version "plan/v1" routes to validate.Plan. (CLM-002)
func TestArtifactValidate_Route_Plan(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "plan/v1"},
	}
	fn, err := RouteValidator(art)
	if err != nil {
		t.Fatalf("RouteValidator: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil validator function for plan/v1")
	}
	// Call the validator on a known-bad plan artifact (missing required fields)
	badArt := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "plan/v1"},
	}
	result := fn(badArt, nil) // Plans don't use schemas
	if len(result.Violations) == 0 {
		t.Error("expected violations from known-bad plan artifact")
	}
}

// TestArtifactValidate_Route_ADR verifies that an artifact with
// schema_version "adr/v1" routes to validate.ADR. (CLM-003)
func TestArtifactValidate_Route_ADR(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "adr/v1"},
	}
	fn, err := RouteValidator(art)
	if err != nil {
		t.Fatalf("RouteValidator: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil validator function for adr/v1")
	}
	// Call the validator on a known-bad ADR artifact
	badArt := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "adr/v1"},
	}
	schPath, _ := schema.ResolveSchemaPath(badArt)
	sch, _ := loadSchemaFromFS(SchemaFS, schPath)
	result := fn(badArt, sch)
	if len(result.Violations) == 0 {
		t.Error("expected violations from known-bad ADR artifact")
	}
}

// TestArtifactValidate_Route_Bundle verifies that an artifact with
// schema_version "bundle/v1" routes to validate.Bundle. (CLM-004)
func TestArtifactValidate_Route_Bundle(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "bundle/v1"},
	}
	fn, err := RouteValidator(art)
	if err != nil {
		t.Fatalf("RouteValidator: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil validator function for bundle/v1")
	}
	// Call the validator on a known-bad bundle artifact
	badArt := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "bundle/v1"},
	}
	schPath, _ := schema.ResolveSchemaPath(badArt)
	sch, _ := loadSchemaFromFS(SchemaFS, schPath)
	result := fn(badArt, sch)
	if len(result.Violations) == 0 {
		t.Error("expected violations from known-bad bundle artifact")
	}
}

// TestArtifactValidate_Route_Issue verifies that an artifact with
// schema_version "issue/v1" routes to validate.Issue. (CLM-005)
func TestArtifactValidate_Route_Issue(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "issue/v1"},
	}
	fn, err := RouteValidator(art)
	if err != nil {
		t.Fatalf("RouteValidator: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil validator function for issue/v1")
	}
	// Call the validator on a known-bad issue artifact
	badArt := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "issue/v1"},
	}
	schPath, _ := schema.ResolveSchemaPath(badArt)
	sch, _ := loadSchemaFromFS(SchemaFS, schPath)
	result := fn(badArt, sch)
	if len(result.Violations) == 0 {
		t.Error("expected violations from known-bad issue artifact")
	}
}

// Standard artifact type removed — standards live inside packs now (DD-18/DD-45).

// TestArtifactValidate_Route_UnknownType_Exit2 verifies that an artifact with
// an unrecognized schema_version prefix produces a config error. (CLM-007)
func TestArtifactValidate_Route_UnknownType_Exit2(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{"schema_version": "unknown/v1"},
	}
	fn, err := RouteValidator(art)
	if err == nil {
		t.Fatal("expected error for unknown schema_version prefix, got nil")
	}
	if fn != nil {
		t.Fatal("expected nil validator for unknown schema_version prefix")
	}
	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
	}
}

// TestArtifactValidate_Route_MissingSchemaVersion_Exit2 verifies that an
// artifact with no schema_version produces a config error. (CLM-008)
func TestArtifactValidate_Route_MissingSchemaVersion_Exit2(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Metadata: map[string]string{"title": "No Schema"},
	}
	fn, err := RouteValidator(art)
	if err == nil {
		t.Fatal("expected error for missing schema_version, got nil")
	}
	if fn != nil {
		t.Fatal("expected nil validator for missing schema_version")
	}
	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
	}
}
