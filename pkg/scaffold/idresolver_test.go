package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockGitExecutor is a configurable mock for testing git tag operations.
type mockGitExecutor struct {
	tags          []string
	isRepo        bool
	isAvailable   bool
	fetchErr      error
	createTagErr  error
	pushTagErr    error
	pushCallCount int
	createdTags   []string
	pushedTags    []string
}

func (m *mockGitExecutor) ListTags(pattern string) ([]string, error) {
	var matched []string
	for _, t := range m.tags {
		// Simple glob match: replace * with anything
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(t, prefix) {
			matched = append(matched, t)
		}
	}
	return matched, nil
}

func (m *mockGitExecutor) CreateAnnotatedTag(name, message string) error {
	m.createdTags = append(m.createdTags, name)
	if m.createTagErr != nil {
		return m.createTagErr
	}
	return nil
}

func (m *mockGitExecutor) PushTag(name string) error {
	m.pushCallCount++
	m.pushedTags = append(m.pushedTags, name)
	if m.pushTagErr != nil {
		return m.pushTagErr
	}
	return nil
}

func (m *mockGitExecutor) FetchTags() error {
	return m.fetchErr
}

func (m *mockGitExecutor) IsGitRepo() bool {
	return m.isRepo
}

func (m *mockGitExecutor) IsGitAvailable() bool {
	return m.isAvailable
}

func TestArtifactNew_GitTagReservation_NextSequential(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001", "backstop/spec/002"},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	id, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "003" {
		t.Fatalf("expected id '003', got %q", id)
	}
}

func TestArtifactNew_GitTagReservation_CreatesAnnotatedTag(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001"},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.createdTags) == 0 {
		t.Fatal("expected annotated tag to be created")
	}
	expectedTag := "backstop/spec/002"
	if mock.createdTags[0] != expectedTag {
		t.Fatalf("expected tag %q, got %q", expectedTag, mock.createdTags[0])
	}
}

func TestArtifactNew_GitTagReservation_RetryOnConflict(t *testing.T) {
	callCount := 0
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001"},
		isRepo:      true,
		isAvailable: true,
	}
	// Override push to fail on first call with conflict, succeed on second
	origPush := mock.pushTagErr
	_ = origPush
	mock.pushTagErr = nil

	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		pushFunc: func(name string) error {
			callCount++
			if callCount == 1 {
				return &TagConflictError{Tag: name}
			}
			return nil
		},
	}
	id, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// First attempt: 002 (conflict), second attempt: 003 (success)
	if id != "003" {
		t.Fatalf("expected id '003' after retry, got %q", id)
	}
}

func TestArtifactNew_GitTagReservation_GapsPreserved(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001", "backstop/spec/003"},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	id, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Next after 003 is 004, not filling gap at 002
	if id != "004" {
		t.Fatalf("expected id '004' (gaps preserved), got %q", id)
	}
}

func TestArtifactNew_GitTag_IsAnnotated(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The mock's CreateAnnotatedTag was called (not a lightweight tag function)
	if len(mock.createdTags) != 1 {
		t.Fatalf("expected exactly 1 annotated tag creation, got %d", len(mock.createdTags))
	}
}

func TestArtifactNew_GitTag_Format(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/adr/0001"},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("adr", "my-adr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "backstop/adr/0002"
	if mock.createdTags[0] != expected {
		t.Fatalf("expected tag format %q, got %q", expected, mock.createdTags[0])
	}
}

func TestArtifactNew_GitTag_MessageContents(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	// We need to capture the message passed to CreateAnnotatedTag.
	var capturedMessage string
	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		createFunc: func(name, message string) error {
			capturedMessage = message
			mock.createdTags = append(mock.createdTags, name)
			return nil
		},
	}
	_, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedMessage, "my-spec") {
		t.Fatalf("expected tag message to contain slug 'my-spec', got %q", capturedMessage)
	}
	// Message should contain a timestamp-like string (at least a date)
	if len(capturedMessage) < 10 {
		t.Fatalf("expected tag message to contain timestamp, got %q", capturedMessage)
	}
}

func TestArtifactNew_GitTag_PushSpecificTag(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.pushedTags) != 1 {
		t.Fatalf("expected exactly 1 push call, got %d", len(mock.pushedTags))
	}
	expected := "backstop/spec/001"
	if mock.pushedTags[0] != expected {
		t.Fatalf("expected push of specific tag %q, got %q", expected, mock.pushedTags[0])
	}
}

func TestArtifactNew_GitTag_MaxRetries(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001"},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		pushFunc: func(name string) error {
			return &TagConflictError{Tag: name}
		},
	}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected error after max retries exhausted, got nil")
	}
	var exhausted *RetriesExhaustedError
	if !isRetriesExhausted(err) {
		t.Fatalf("expected RetriesExhaustedError, got %T: %v", err, err)
	}
	_ = exhausted
}

// isRetriesExhausted checks if err is a RetriesExhaustedError.
func isRetriesExhausted(err error) bool {
	_, ok := err.(*RetriesExhaustedError)
	return ok
}

func TestArtifactNew_ExitCode_2_RetriesExhausted(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		pushFunc: func(name string) error {
			return &TagConflictError{Tag: name}
		},
	}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected error for retries exhausted")
	}
	if !isRetriesExhausted(err) {
		t.Fatalf("expected RetriesExhaustedError, got %T: %v", err, err)
	}
}

func TestArtifactNew_ExitCode_NonConflictPushFallsBack(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		pushFunc: func(name string) error {
			return fmt.Errorf("network error: connection refused")
		},
	}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected fallback sentinel error")
	}
	if !isFallbackError(err) {
		t.Fatalf("expected FallbackError, got %T: %v", err, err)
	}
}

// isFallbackError checks if err is a FallbackError.
func isFallbackError(err error) bool {
	_, ok := err.(*FallbackError)
	return ok
}

// --- ResolveID integration tests ---

func TestArtifactNew_ResolveID_GitSuccess(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001", "backstop/spec/002"},
		isRepo:      true,
		isAvailable: true,
	}
	id, err := ResolveID("spec", IDOptions{
		ProjectRoot: t.TempDir(),
		Executor:    mock,
		MaxRetries:  3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "003" {
		t.Fatalf("expected id '003', got %q", id)
	}
}

func TestArtifactNew_ResolveID_FallbackToLocalScan(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "SPEC-002-foo.spec.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := &mockGitExecutor{
		isRepo:      false,
		isAvailable: false,
	}
	id, err := ResolveID("spec", IDOptions{
		ProjectRoot: tmpDir,
		Executor:    mock,
		MaxRetries:  3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "003" {
		t.Fatalf("expected id '003' from local scan, got %q", id)
	}
}

func TestArtifactNew_ResolveID_RetriesExhausted(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{},
		isRepo:      true,
		isAvailable: true,
	}
	// Override push to always conflict
	origResolve := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		pushFunc: func(name string) error {
			return &TagConflictError{Tag: name}
		},
	}
	_, err := origResolve.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected error")
	}
	if !isRetriesExhausted(err) {
		t.Fatalf("expected RetriesExhaustedError, got %T: %v", err, err)
	}
}

// --- Error() method coverage ---

func TestTagConflictError_Error(t *testing.T) {
	err := &TagConflictError{Tag: "backstop/spec/001"}
	msg := err.Error()
	if !strings.Contains(msg, "backstop/spec/001") {
		t.Fatalf("expected tag name in error message, got %q", msg)
	}
	if !strings.Contains(msg, "tag conflict") {
		t.Fatalf("expected 'tag conflict' in error message, got %q", msg)
	}
}

func TestFallbackError_Error(t *testing.T) {
	err := &FallbackError{Reason: "git not available"}
	msg := err.Error()
	if !strings.Contains(msg, "git not available") {
		t.Fatalf("expected reason in error message, got %q", msg)
	}
	if !strings.Contains(msg, "falling back") {
		t.Fatalf("expected 'falling back' in error message, got %q", msg)
	}
}

func TestRetriesExhaustedError_Error(t *testing.T) {
	err := &RetriesExhaustedError{Attempts: 4}
	msg := err.Error()
	if !strings.Contains(msg, "4") {
		t.Fatalf("expected attempt count in error message, got %q", msg)
	}
	if !strings.Contains(msg, "retries exhausted") {
		t.Fatalf("expected 'retries exhausted' in error message, got %q", msg)
	}
}
