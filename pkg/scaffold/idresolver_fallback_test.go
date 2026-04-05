package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactNew_OfflineFallback_NotGitRepo(t *testing.T) {
	mock := &mockGitExecutor{
		isRepo:      false,
		isAvailable: true,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected fallback error when not a git repo")
	}
	if !isFallbackError(err) {
		t.Fatalf("expected FallbackError, got %T: %v", err, err)
	}
}

func TestArtifactNew_OfflineFallback_NoGitBinary(t *testing.T) {
	mock := &mockGitExecutor{
		isRepo:      false,
		isAvailable: false,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected fallback error when git binary not available")
	}
	if !isFallbackError(err) {
		t.Fatalf("expected FallbackError, got %T: %v", err, err)
	}
}

func TestArtifactNew_OfflineFallback_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "specs")
	resolver := &LocalScanResolver{}
	id, err := resolver.Resolve("spec", nonExistentDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "001" {
		t.Fatalf("expected id '001' for non-existent dir, got %q", id)
	}
}

func TestArtifactNew_OfflineFallback_ScansExistingArtifacts(t *testing.T) {
	// Create temp dir with existing spec artifacts
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create SPEC-001 and SPEC-003 (gap at 002)
	for _, name := range []string{"SPEC-001-foo.spec.md", "SPEC-003-bar.spec.md"} {
		if err := os.WriteFile(filepath.Join(specsDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	resolver := &LocalScanResolver{}
	id, err := resolver.Resolve("spec", specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Next after 003 is 004
	if id != "004" {
		t.Fatalf("expected id '004', got %q", id)
	}
}

func TestArtifactNew_OfflineFallback_FetchNetworkFailure(t *testing.T) {
	mock := &mockGitExecutor{
		isRepo:      true,
		isAvailable: true,
		fetchErr:    fmt.Errorf("fatal: unable to access remote: network is unreachable"),
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected fallback error on fetch network failure")
	}
	if !isFallbackError(err) {
		t.Fatalf("expected FallbackError, got %T: %v", err, err)
	}
}

func TestArtifactNew_OfflineFallback_PushUnreachableRemote(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		pushFunc: func(name string) error {
			return fmt.Errorf("fatal: Could not read from remote repository")
		},
	}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected fallback error on unreachable remote")
	}
	if !isFallbackError(err) {
		t.Fatalf("expected FallbackError, got %T: %v", err, err)
	}
}

func TestArtifactNew_OfflineFallback_DirectoryWithSubdirs(t *testing.T) {
	// Test that local scan handles subdirectories correctly (skips them)
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a subdirectory that should be skipped
	if err := os.MkdirAll(filepath.Join(specsDir, "SPEC-999-subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a valid artifact file
	if err := os.WriteFile(filepath.Join(specsDir, "SPEC-001-foo.spec.md"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a file that doesn't match the pattern
	if err := os.WriteFile(filepath.Join(specsDir, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := &LocalScanResolver{}
	id, err := resolver.Resolve("spec", specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be 002 (next after 001), ignoring the subdir and non-matching file
	if id != "002" {
		t.Fatalf("expected id '002', got %q", id)
	}
}

func TestArtifactNew_OfflineFallback_PushPermissionError(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		pushFunc: func(name string) error {
			return fmt.Errorf("remote: Permission denied")
		},
	}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected fallback error on permission error")
	}
	if !isFallbackError(err) {
		t.Fatalf("expected FallbackError, got %T: %v", err, err)
	}
}
