package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
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
//
// It RETURNS the rendered content and the path it wrote, so each per-kind test
// asserts on that kind's own result rather than delegating its whole verdict here.
// A test whose body is nothing but a helper call asserts nothing the substantiveness
// dimension can see, and — more to the point — cannot say anything kind-specific.
func sourceIgnoredTestCase(t *testing.T, artifactType string) (content string, writtenPath string) {
	t.Helper()

	// Scaffold the artifact with a source that would be relevant for plans
	rendered, err := Scaffold(artifactType, "001", "test-slug", "2026-04-04", "SPEC-002")
	if err != nil {
		t.Fatalf("Scaffold(%q) with source returned unexpected error: %v", artifactType, err)
	}

	s := string(rendered)

	// The output should not reference SPEC-002 since source is ignored for non-plan types
	if contains(s, "SPEC-002") {
		t.Fatalf("expected source SPEC-002 to be ignored for %s, but it appears in output:\n%s", artifactType, s)
	}
	if contains(s, "spec_id") {
		t.Fatalf("expected no spec_id in %s scaffold, but found it in output:\n%s", artifactType, s)
	}

	// Verify the file would be written to the correct directory
	dir := TargetDir(artifactType, rootAt("/root"))
	layout, layoutOK := artifact.LayoutFor(artifact.Kind(artifactType))
	if !layoutOK {
		t.Fatalf("artifact.LayoutFor(%q) returned ok=false", artifactType)
	}
	expectedDir := filepath.Join("/root", layout.Directory)
	if dir != expectedDir {
		t.Fatalf("expected target dir %q, got %q", expectedDir, dir)
	}

	// Verify a file can be created (filename doesn't include source)
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, layout.Directory)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filename := Filename(artifactType, "001", "test-slug", "SPEC-002")
	filePath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(filePath, rendered, 0o644); err != nil {
		t.Fatalf("writing scaffold for %s: %v", artifactType, err)
	}

	// Verify file was written successfully
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file to exist at %s: %v", filePath, err)
	}

	return s, filePath
}

// assertSourceIgnored is the per-kind assertion each Ignored-for-<kind> test makes in
// its OWN body: the plan-only source ID is absent from that kind's rendered artifact,
// and the file the scaffolder produced carries that kind's declared extension. The
// expected extension is passed as a literal rather than read back from the layout table
// so the assertion cannot agree with a table that has drifted.
func assertSourceIgnored(t *testing.T, content, path, wantExtension string) {
	t.Helper()
	if contains(content, "SPEC-002") {
		t.Errorf("plan-only source SPEC-002 leaked into the rendered artifact:\n%s", content)
	}
	if !strings.HasSuffix(path, wantExtension) {
		t.Errorf("expected the scaffolded file to end in %q, got %q", wantExtension, path)
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
	content, path := sourceIgnoredTestCase(t, "spec")
	assertSourceIgnored(t, content, path, ".spec.md")
	if !contains(content, "number: SPEC-001") {
		t.Errorf("expected the scaffolded spec to carry its OWN id, got:\n%s", content)
	}
}

func TestArtifactNew_Source_IgnoredForIssue(t *testing.T) {
	content, path := sourceIgnoredTestCase(t, "issue")
	assertSourceIgnored(t, content, path, ".issue.md")
	if !contains(content, "id: ISSUE-001") {
		t.Errorf("expected the scaffolded issue to carry its OWN id, got:\n%s", content)
	}
}

func TestArtifactNew_Source_IgnoredForADR(t *testing.T) {
	content, path := sourceIgnoredTestCase(t, "adr")
	assertSourceIgnored(t, content, path, ".adr.md")
	if !contains(content, "number: ADR-001") {
		t.Errorf("expected the scaffolded ADR to carry its OWN id, got:\n%s", content)
	}
}

func TestArtifactNew_Source_IgnoredForDirective(t *testing.T) {
	content, path := sourceIgnoredTestCase(t, "directive")
	assertSourceIgnored(t, content, path, ".directive.md")
	if !contains(content, "number: DIR-001") {
		t.Errorf("expected the scaffolded directive to carry its OWN id, got:\n%s", content)
	}
}

func TestArtifactNew_Source_IgnoredForBundle(t *testing.T) {
	content, path := sourceIgnoredTestCase(t, "bundle")
	assertSourceIgnored(t, content, path, ".bundle.md")
	if !contains(content, "number: BUNDLE-001") {
		t.Errorf("expected the scaffolded bundle to carry its OWN id, got:\n%s", content)
	}
}

func TestArtifactNew_Source_IgnoredForCapability(t *testing.T) {
	content, path := sourceIgnoredTestCase(t, "capability")
	assertSourceIgnored(t, content, path, ".capability.yml")
	if !contains(content, "id: CAP-001") {
		t.Errorf("expected the scaffolded capability to carry its OWN id, got:\n%s", content)
	}
}
