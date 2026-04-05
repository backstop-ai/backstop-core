package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactNew_Source_SpecBacked(t *testing.T) {
	err := ValidateSource("SPEC-002")
	if err != nil {
		t.Fatalf("expected valid spec source to be accepted, got error: %v", err)
	}
	kind, err := ParseSourceKind("SPEC-002")
	if err != nil {
		t.Fatalf("expected ParseSourceKind to succeed, got error: %v", err)
	}
	if kind != "spec" {
		t.Fatalf("expected kind 'spec', got %q", kind)
	}
}

func TestArtifactNew_Source_IssueBacked(t *testing.T) {
	err := ValidateSource("ISSUE-005")
	if err != nil {
		t.Fatalf("expected valid issue source to be accepted, got error: %v", err)
	}
	kind, err := ParseSourceKind("ISSUE-005")
	if err != nil {
		t.Fatalf("expected ParseSourceKind to succeed, got error: %v", err)
	}
	if kind != "issue" {
		t.Fatalf("expected kind 'issue', got %q", kind)
	}
}

func TestArtifactNew_Source_MissingForPlan_Exit2(t *testing.T) {
	err := ValidateSource("")
	if err == nil {
		t.Fatal("expected error for empty source, got nil")
	}
}

func TestArtifactNew_Source_InvalidFormat_Exit2(t *testing.T) {
	err := ValidateSource("BADFORMAT")
	if err == nil {
		t.Fatal("expected error for invalid source format, got nil")
	}
	// Also verify ParseSourceKind returns error for invalid input
	_, err = ParseSourceKind("BADFORMAT")
	if err == nil {
		t.Fatal("expected ParseSourceKind error for invalid format, got nil")
	}
}

// sourceIgnoredTestCase scaffolds a non-plan artifact with a --source flag
// and verifies the source is silently ignored: no error, and the scaffolded
// content does not reference the source ID.
func sourceIgnoredTestCase(t *testing.T, artifactType string) {
	t.Helper()

	// Scaffold the artifact with a source that would be relevant for plans
	content, err := Scaffold(artifactType, "001", "test-slug", "2026-04-04", "SPEC-002")
	if err != nil {
		t.Fatalf("Scaffold(%q) with source returned unexpected error: %v", artifactType, err)
	}

	s := string(content)

	// The output should not reference SPEC-002 since source is ignored for non-plan types
	if contains(s, "SPEC-002") {
		t.Fatalf("expected source SPEC-002 to be ignored for %s, but it appears in output:\n%s", artifactType, s)
	}
	if contains(s, "spec_id") {
		t.Fatalf("expected no spec_id in %s scaffold, but found it in output:\n%s", artifactType, s)
	}

	// Verify the file would be written to the correct directory
	dir := TargetDir(artifactType, "/root")
	cfg := ValidArtifactTypes[artifactType]
	expectedDir := filepath.Join("/root", cfg.Directory)
	if dir != expectedDir {
		t.Fatalf("expected target dir %q, got %q", expectedDir, dir)
	}

	// Verify a file can be created (filename doesn't include source)
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, cfg.Directory)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filename := Filename(artifactType, "001", "test-slug", "SPEC-002")
	filePath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("writing scaffold for %s: %v", artifactType, err)
	}

	// Verify file was written successfully
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file to exist at %s: %v", filePath, err)
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestArtifactNew_Source_IgnoredForSpec(t *testing.T) {
	sourceIgnoredTestCase(t, "spec")
}

func TestArtifactNew_Source_IgnoredForIssue(t *testing.T) {
	sourceIgnoredTestCase(t, "issue")
}

func TestArtifactNew_Source_IgnoredForADR(t *testing.T) {
	sourceIgnoredTestCase(t, "adr")
}

func TestArtifactNew_Source_IgnoredForDirective(t *testing.T) {
	sourceIgnoredTestCase(t, "directive")
}

func TestArtifactNew_Source_IgnoredForBundle(t *testing.T) {
	sourceIgnoredTestCase(t, "bundle")
}

func TestArtifactNew_Source_IgnoredForCapability(t *testing.T) {
	sourceIgnoredTestCase(t, "capability")
}
