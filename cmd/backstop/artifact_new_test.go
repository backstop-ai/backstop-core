package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/scaffold"
)

// testArtifactNewCmd is a helper that executes the artifact new command with
// a mock git executor and temp directory, returning the exit code and output.
type artifactNewTestCase struct {
	args       []string
	executor   scaffold.GitExecutor
	projectDir string
	wantExit   int
}

func runArtifactNewTest(t *testing.T, tc artifactNewTestCase) (int, string) {
	t.Helper()

	projectDir := tc.projectDir
	if projectDir == "" {
		projectDir = t.TempDir()
	}

	cmd := newArtifactNewCommandWithDeps(scaffold.ArtifactNewDeps{
		Executor:    tc.executor,
		ProjectRoot: projectDir,
		DateFunc:    func() string { return "2026-04-04" },
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(tc.args)

	err := cmd.Execute()
	exitCode := 0
	if err != nil {
		if ece, ok := err.(*ExitCodeError); ok {
			exitCode = ece.Code
		} else {
			exitCode = 1 // default error exit
		}
	}

	return exitCode, buf.String()
}

// noopGitExecutor always triggers fallback to local scan.
type noopGitExecutor struct{}

func (n *noopGitExecutor) ListTags(string) ([]string, error) { return nil, nil }
func (n *noopGitExecutor) CreateAnnotatedTag(string, string) error {
	return nil
}
func (n *noopGitExecutor) PushTag(string) error { return nil }
func (n *noopGitExecutor) FetchTags() error     { return nil }
func (n *noopGitExecutor) IsGitRepo() bool      { return false }
func (n *noopGitExecutor) IsGitAvailable() bool { return false }

// --- Type validation tests ---

func TestArtifactNew_ValidType_Spec(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"spec", "--slug", "my-spec"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_ValidType_Plan(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"plan", "--slug", "my-plan", "--source", "SPEC-002"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_ValidType_Issue(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"issue", "--slug", "my-issue"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_ValidType_ADR(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"adr", "--slug", "my-adr"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_ValidType_Directive(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"directive", "--slug", "my-dir"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_ValidType_Bundle(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"bundle", "--slug", "my-bundle"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_ValidType_Capability(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"capability", "--slug", "my-cap"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_InvalidType_Exit2(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"bogus", "--slug", "my-thing"},
		executor: &noopGitExecutor{},
		wantExit: 2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestArtifactNew_MissingType_Exit2(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{},
		executor: &noopGitExecutor{},
		wantExit: 2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

// --- Directory and file tests ---

func TestArtifactNew_Directory_CreatedIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "my-spec"},
		executor:   &noopGitExecutor{},
		projectDir: tmpDir,
		wantExit:   0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	specsDir := filepath.Join(tmpDir, "specs")
	info, err := os.Stat(specsDir)
	if err != nil {
		t.Fatalf("expected specs/ dir to be created, got error: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected specs/ to be a directory")
	}
}

func TestArtifactNew_Directory_ExistingFileRefused(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Use a git executor that returns tags so resolver produces ID 003.
	// Pre-create the conflict file at that exact path.
	existingFile := filepath.Join(specsDir, "SPEC-003-my-spec.spec.md")
	if err := os.WriteFile(existingFile, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "my-spec"},
		executor:   &fixedIDGitExecutor{tags: []string{"backstop/spec/001", "backstop/spec/002"}},
		projectDir: tmpDir,
		wantExit:   1,
	})
	if code != 1 {
		t.Fatalf("expected exit 1 (file exists), got %d", code)
	}
}

// --- Exit code tests ---

func TestArtifactNew_ExitCode_0_Success(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"spec", "--slug", "my-spec"},
		executor: &noopGitExecutor{},
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestArtifactNew_ExitCode_1_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Git resolver returns tags 001,002 so next ID is 003. Pre-create conflict file.
	existingFile := filepath.Join(specsDir, "SPEC-003-my-spec.spec.md")
	if err := os.WriteFile(existingFile, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "my-spec"},
		executor:   &fixedIDGitExecutor{tags: []string{"backstop/spec/001", "backstop/spec/002"}},
		projectDir: tmpDir,
		wantExit:   1,
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestArtifactNew_ExitCode_2_InvalidType(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"bogus", "--slug", "my-thing"},
		executor: &noopGitExecutor{},
		wantExit: 2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestArtifactNew_ExitCode_2_InvalidSlug(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"spec", "--slug", "BadSlug"},
		executor: &noopGitExecutor{},
		wantExit: 2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestArtifactNew_ExitCode_2_RetriesExhausted(t *testing.T) {
	// Use an executor that always causes tag conflicts
	mock := &conflictGitExecutor{}
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"spec", "--slug", "my-spec"},
		executor: mock,
		wantExit: 2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2 (retries exhausted), got %d", code)
	}
}

func TestArtifactNew_ExitCode_NonConflictPushFallsBack(t *testing.T) {
	// Executor where push fails with non-conflict error, triggering fallback
	mock := &networkFailGitExecutor{}
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"spec", "--slug", "my-spec"},
		executor: mock,
		wantExit: 0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0 (fallback succeeds), got %d", code)
	}
}

func TestArtifactNew_ExitCode_2_MissingSourceForPlan(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"plan", "--slug", "my-plan"},
		executor: &noopGitExecutor{},
		wantExit: 2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

// TestArtifactNew_ExitCode_2_InvalidSourceForPlan verifies that a plan whose
// --source is present but malformed (not SPEC-NNN / ISSUE-NNN) is rejected with
// exit 2, exercising the ValidateSource failure branch distinct from the
// missing-source branch.
func TestArtifactNew_ExitCode_2_InvalidSourceForPlan(t *testing.T) {
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:     []string{"plan", "--slug", "my-plan", "--source", "not-a-valid-source"},
		executor: &noopGitExecutor{},
		wantExit: 2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2 for malformed --source, got %d", code)
	}
}

func TestArtifactNew_ExitCode_2_PrecedesOne(t *testing.T) {
	// Both invalid slug (exit 2) and existing file (exit 1) conditions.
	// Exit 2 should take precedence.
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existingFile := filepath.Join(specsDir, "SPEC-001-BadSlug.spec.md")
	if err := os.WriteFile(existingFile, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "BadSlug"},
		executor:   &noopGitExecutor{},
		projectDir: tmpDir,
		wantExit:   2,
	})
	if code != 2 {
		t.Fatalf("expected exit 2 (precedence over 1), got %d", code)
	}
}

// --- Thin adapter tests ---

func TestArtifactNew_ThinAdapter_DelegatesRendering(t *testing.T) {
	// Verify the command produces content from scaffold.Scaffold, not inline
	tmpDir := t.TempDir()
	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "my-spec"},
		executor:   &noopGitExecutor{},
		projectDir: tmpDir,
		wantExit:   0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	// Read the written file and verify it has scaffold-generated content
	content, err := os.ReadFile(filepath.Join(tmpDir, "specs", "SPEC-001-my-spec.spec.md"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	// The content should contain frontmatter from Scaffold function
	if len(content) == 0 {
		t.Fatal("generated file is empty — adapter must delegate to scaffold.Scaffold")
	}
	s := string(content)
	if !contains(s, "title:") || !contains(s, "schema_version:") {
		t.Fatal("generated content doesn't match scaffold.Scaffold output")
	}
}

func TestArtifactNew_ThinAdapter_DelegatesIDResolution(t *testing.T) {
	// Verify the command uses scaffold.ResolveID for ID assignment
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create an existing artifact so local scan returns 002
	if err := os.WriteFile(
		filepath.Join(specsDir, "SPEC-001-existing.spec.md"),
		[]byte("test"), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	code, _ := runArtifactNewTest(t, artifactNewTestCase{
		args:       []string{"spec", "--slug", "new-spec"},
		executor:   &noopGitExecutor{},
		projectDir: tmpDir,
		wantExit:   0,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	// The file should be SPEC-002 (next after 001 from local scan)
	_, err := os.Stat(filepath.Join(specsDir, "SPEC-002-new-spec.spec.md"))
	if err != nil {
		t.Fatalf("expected SPEC-002 file (ID from ResolveID), got error: %v", err)
	}
}

// contains is a simple helper for string containment checks.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// fixedIDGitExecutor returns a specific set of tags so the resolver
// produces a known next ID.
type fixedIDGitExecutor struct {
	tags []string
}

func (f *fixedIDGitExecutor) ListTags(string) ([]string, error) { return f.tags, nil }
func (f *fixedIDGitExecutor) CreateAnnotatedTag(string, string) error {
	return nil
}
func (f *fixedIDGitExecutor) PushTag(string) error { return nil }
func (f *fixedIDGitExecutor) FetchTags() error     { return nil }
func (f *fixedIDGitExecutor) IsGitRepo() bool      { return true }
func (f *fixedIDGitExecutor) IsGitAvailable() bool { return true }

// conflictGitExecutor simulates an executor where git is available but
// tag push always fails with a tag conflict.
type conflictGitExecutor struct{}

func (c *conflictGitExecutor) ListTags(string) ([]string, error) { return nil, nil }
func (c *conflictGitExecutor) CreateAnnotatedTag(string, string) error {
	return nil
}
func (c *conflictGitExecutor) PushTag(string) error {
	return &scaffold.TagConflictError{Tag: "test"}
}
func (c *conflictGitExecutor) FetchTags() error     { return nil }
func (c *conflictGitExecutor) IsGitRepo() bool      { return true }
func (c *conflictGitExecutor) IsGitAvailable() bool { return true }

// networkFailGitExecutor simulates an executor where git is available
// but push fails with a non-conflict network error (should trigger fallback).
type networkFailGitExecutor struct{}

func (n *networkFailGitExecutor) ListTags(string) ([]string, error) { return nil, nil }
func (n *networkFailGitExecutor) CreateAnnotatedTag(string, string) error {
	return nil
}
func (n *networkFailGitExecutor) PushTag(string) error {
	return fmt.Errorf("network error: connection refused")
}
func (n *networkFailGitExecutor) FetchTags() error     { return nil }
func (n *networkFailGitExecutor) IsGitRepo() bool      { return true }
func (n *networkFailGitExecutor) IsGitAvailable() bool { return true }
