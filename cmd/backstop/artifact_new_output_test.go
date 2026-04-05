package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArtifactNew_Output_JSON_Fields(t *testing.T) {
	_, output := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"spec", "--slug", "my-spec", "--json"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})

	var result ArtifactNewResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if result.ArtifactType != "spec" {
		t.Errorf("expected artifact_type 'spec', got %q", result.ArtifactType)
	}
	if result.ID == "" {
		t.Error("expected non-empty ID in JSON output")
	}
	if result.FilePath == "" {
		t.Error("expected non-empty file_path in JSON output")
	}
	if result.SchemaVersion == "" {
		t.Error("expected non-empty schema_version in JSON output")
	}
}

func TestArtifactNew_Output_Human_Display(t *testing.T) {
	_, output := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"spec", "--slug", "my-spec"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})

	if !strings.Contains(output, "Created") {
		t.Errorf("human output should contain 'Created', got %q", output)
	}
	if !strings.Contains(output, "ID:") {
		t.Errorf("human output should contain 'ID:', got %q", output)
	}
	if !strings.Contains(output, ".spec.md") {
		t.Errorf("human output should contain file path, got %q", output)
	}
}

func TestArtifactNew_Output_DataParity(t *testing.T) {
	// Run once with JSON
	tmpDir1 := t.TempDir()
	_, jsonOutput := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "my-spec", "--json"},
		executor:   &noopGitExecutor{},
		projectDir: tmpDir1,
		wantExit:   0,
	})

	var jsonResult ArtifactNewResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonOutput)), &jsonResult); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Run once with human
	tmpDir2 := t.TempDir()
	_, humanOutput := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "my-spec"},
		executor:   &noopGitExecutor{},
		projectDir: tmpDir2,
		wantExit:   0,
	})

	// Verify data parity: human output should contain the same ID and type info
	if !strings.Contains(humanOutput, jsonResult.ID) {
		t.Errorf("human output should contain same ID %q as JSON", jsonResult.ID)
	}
	if jsonResult.ArtifactType != "spec" {
		t.Errorf("expected artifact_type 'spec' in JSON, got %q", jsonResult.ArtifactType)
	}
}
