package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
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

// resolvedRootAt resolves an UNCONFIGURED artifact root at dir through the REAL
// ResolveRoot, rather than composing an artifact.Root literal. These fixtures declare no
// artifact_root, so their project root IS their artifact root and what the tests assert
// is unchanged — but going through the resolver is what makes them exercise the
// absolute-path guarantee the ID fallback now depends on.
func resolvedRootAt(t *testing.T, dir string) artifact.Root {
	t.Helper()
	root, err := artifact.ResolveRoot(dir, "")
	if err != nil {
		t.Fatalf("resolving an unconfigured artifact root at %s: %v", dir, err)
	}
	return root
}

// --- ResolveID integration tests ---

func TestArtifactNew_ResolveID_GitSuccess(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001", "backstop/spec/002"},
		isRepo:      true,
		isAvailable: true,
	}
	id, err := ResolveID("spec", IDOptions{
		Root:       resolvedRootAt(t, t.TempDir()),
		Executor:   mock,
		MaxRetries: 3,
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
		Root:       resolvedRootAt(t, tmpDir),
		Executor:   mock,
		MaxRetries: 3,
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

// --- Failure classification and defensive parsing ---
//
// The tests below cover the error and edge paths of ID resolution. They matter beyond the
// coverage number: ResolveID's contract IS a classification — which failures are eligible
// for the local-scan fallback and which must surface — and the local scan is what decides
// the next number. A misclassified failure, or a tag/filename the scan adopts as the
// high-water mark when it should skip it, hands out a wrong ID silently. Each test below
// therefore asserts the classification AND that nothing was reserved or renumbered.

// rawTagExecutor returns a fixed tag list verbatim, or an error, bypassing the shared
// mock's glob filter. Resolve parses tags defensively — segment counts and numbers it did
// not itself produce — and the glob filter would otherwise make those inputs unreachable
// from a test, since anything matching "backstop/<type>/*" already has three segments.
type rawTagExecutor struct {
	*mockGitExecutor
	rawTags []string
	listErr error
}

func (e *rawTagExecutor) ListTags(string) ([]string, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	return e.rawTags, nil
}

func TestArtifactNew_GitTagReservation_ListTagsFailureFallsBack(t *testing.T) {
	exec := &rawTagExecutor{
		mockGitExecutor: &mockGitExecutor{isRepo: true, isAvailable: true},
		listErr:         fmt.Errorf("fatal: could not read from remote repository"),
	}
	resolver := &GitTagResolver{executor: exec, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected an error when listing tags fails")
	}
	if !isFallbackError(err) {
		t.Fatalf("a failed tag listing is a remote problem and must be fallback-eligible, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "list tags failed") {
		t.Fatalf("expected the reason to name the failing step, got %q", err.Error())
	}
	// Never reserve on top of a high-water mark that was never read.
	if len(exec.createdTags) != 0 || exec.pushCallCount != 0 {
		t.Fatalf("expected no reservation attempt, got created=%v pushes=%d", exec.createdTags, exec.pushCallCount)
	}
}

func TestArtifactNew_GitTagReservation_SkipsUnparseableTags(t *testing.T) {
	exec := &rawTagExecutor{
		mockGitExecutor: &mockGitExecutor{isRepo: true, isAvailable: true},
		rawTags: []string{
			"backstop/spec/004", // the only real high-water mark here
			"backstop-spec-009", // too few slash-separated segments to carry an ID
			"backstop/spec/draft",
			"backstop/spec/007x",
		},
	}
	resolver := &GitTagResolver{executor: exec, maxRetries: 3}
	id, err := resolver.Resolve("spec", "my-spec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 005, not 010 or 008: unparseable tags are skipped, never adopted as the maximum.
	if id != "005" {
		t.Fatalf("expected id '005' from the one parseable tag, got %q", id)
	}
	if len(exec.createdTags) != 1 || exec.createdTags[0] != "backstop/spec/005" {
		t.Fatalf("expected exactly one reservation of backstop/spec/005, got %v", exec.createdTags)
	}
}

func TestArtifactNew_GitTag_CreateFuncFailureIsNotFallbackEligible(t *testing.T) {
	mock := &mockGitExecutor{isRepo: true, isAvailable: true}
	resolver := &GitTagResolver{
		executor:   mock,
		maxRetries: 3,
		createFunc: func(name, _ string) error {
			return fmt.Errorf("fatal: tag %q already exists", name)
		},
	}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected an error when tag creation fails")
	}
	if isFallbackError(err) || isRetriesExhausted(err) {
		t.Fatalf("a local tag-creation failure is neither fallback nor exhaustion, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "creating tag") {
		t.Fatalf("expected the error to name the failing step, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected the underlying git error to be wrapped, got %q", err.Error())
	}
	if mock.pushCallCount != 0 {
		t.Fatalf("expected no push after tag creation failed, got %d", mock.pushCallCount)
	}
}

func TestArtifactNew_GitTag_CreateAnnotatedTagFailureIsNotFallbackEligible(t *testing.T) {
	mock := &mockGitExecutor{
		isRepo:       true,
		isAvailable:  true,
		createTagErr: fmt.Errorf("fatal: gpg failed to sign the data"),
	}
	resolver := &GitTagResolver{executor: mock, maxRetries: 3}
	_, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatal("expected an error when the executor cannot create the annotated tag")
	}
	if isFallbackError(err) || isRetriesExhausted(err) {
		t.Fatalf("an executor tag-creation failure is neither fallback nor exhaustion, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "gpg failed to sign") {
		t.Fatalf("expected the underlying git error to be wrapped, got %q", err.Error())
	}
	if mock.pushCallCount != 0 {
		t.Fatalf("expected no push after tag creation failed, got %d", mock.pushCallCount)
	}
}

func TestArtifactNew_GitTag_NegativeRetryBudgetReservesNothing(t *testing.T) {
	mock := &mockGitExecutor{
		tags:        []string{"backstop/spec/001"},
		isRepo:      true,
		isAvailable: true,
	}
	// A negative budget makes the reservation loop unenterable. The resolver must still
	// refuse rather than fall out of the loop returning an empty ID with a nil error —
	// the one outcome the caller cannot detect.
	resolver := &GitTagResolver{executor: mock, maxRetries: -1}
	id, err := resolver.Resolve("spec", "my-spec")
	if err == nil {
		t.Fatalf("expected a refusal for a negative retry budget, got id %q", id)
	}
	if !isRetriesExhausted(err) {
		t.Fatalf("expected RetriesExhaustedError, got %T: %v", err, err)
	}
	if id != "" {
		t.Fatalf("expected no id alongside the refusal, got %q", id)
	}
	if len(mock.createdTags) != 0 || mock.pushCallCount != 0 {
		t.Fatalf("expected nothing reserved, got created=%v pushes=%d", mock.createdTags, mock.pushCallCount)
	}
}

func TestArtifactNew_OfflineFallback_UnreadableDirectoryIsNotAnEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// A regular file where the artifact directory should be: ReadDir fails with ENOTDIR,
	// which is NOT os.IsNotExist. The distinction is load-bearing — treating it as "no
	// artifacts yet" would restart numbering at 001 over a directory it never read.
	notADir := filepath.Join(tmpDir, "specs")
	if err := os.WriteFile(notADir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := &LocalScanResolver{}
	id, err := resolver.Resolve("spec", notADir)
	if err == nil {
		t.Fatalf("expected an error for an unreadable artifact directory, got id %q", id)
	}
	if id == "001" {
		t.Fatal("an unreadable directory must not be mistaken for an empty one and restart numbering")
	}
	if id != "" {
		t.Fatalf("expected no id alongside the error, got %q", id)
	}
	if !strings.Contains(err.Error(), notADir) {
		t.Fatalf("expected the error to name the directory it could not read, got %q", err.Error())
	}
}

func TestArtifactNew_OfflineFallback_SkipsOutOfRangeIDs(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The filename pattern matches any run of digits, so a run too large for an int gets
	// past the regex and fails at conversion. Skipping it keeps the scan on the real
	// high-water mark instead of adopting a number no artifact could ever use.
	for _, name := range []string{
		"SPEC-99999999999999999999-overflow.spec.md",
		"SPEC-002-real.spec.md",
	} {
		if err := os.WriteFile(filepath.Join(specsDir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	resolver := &LocalScanResolver{}
	id, err := resolver.Resolve("spec", specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "003" {
		t.Fatalf("expected id '003' from the one usable ID, got %q", id)
	}
}

func TestArtifactNew_ResolveID_DefaultsToThreeRetries(t *testing.T) {
	// MaxRetries omitted. Every push conflicts, so the default budget is observable in the
	// push count: one initial attempt plus three retries.
	mock := &mockGitExecutor{
		isRepo:      true,
		isAvailable: true,
		pushTagErr:  &TagConflictError{Tag: "backstop/spec/001"},
	}
	id, err := ResolveID("spec", IDOptions{
		Root:     resolvedRootAt(t, t.TempDir()),
		Executor: mock,
	})
	if err == nil {
		t.Fatalf("expected exhaustion when every push conflicts, got id %q", id)
	}
	if !isRetriesExhausted(err) {
		t.Fatalf("expected RetriesExhaustedError, got %T: %v", err, err)
	}
	if mock.pushCallCount != 4 {
		t.Fatalf("expected 4 push attempts (1 initial + 3 default retries), got %d", mock.pushCallCount)
	}
	// Each retry must advance the number rather than re-push the tag that just conflicted.
	want := []string{"backstop/spec/001", "backstop/spec/002", "backstop/spec/003", "backstop/spec/004"}
	for i, tag := range want {
		if mock.pushedTags[i] != tag {
			t.Fatalf("push %d: expected %q, got %q (all: %v)", i, tag, mock.pushedTags[i], mock.pushedTags)
		}
	}
}

func TestArtifactNew_ResolveID_ExhaustionDoesNotFallBackToLocalScan(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A populated directory the local scan would happily answer "008" from. If exhaustion
	// were ever made fallback-eligible, this test sees a successful "008" instead of the
	// refusal — which is exactly why it asserts against a stocked directory rather than
	// an empty one.
	if err := os.WriteFile(filepath.Join(specsDir, "SPEC-007-foo.spec.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := &mockGitExecutor{
		isRepo:      true,
		isAvailable: true,
		pushTagErr:  &TagConflictError{Tag: "backstop/spec/001"},
	}
	id, err := ResolveID("spec", IDOptions{
		Root:       resolvedRootAt(t, tmpDir),
		Executor:   mock,
		MaxRetries: 1,
	})
	if err == nil {
		t.Fatalf("expected exhaustion to surface, got id %q", id)
	}
	if !isRetriesExhausted(err) {
		t.Fatalf("expected RetriesExhaustedError, got %T: %v", err, err)
	}
	if id != "" {
		t.Fatalf("expected no id: a contested reservation must not be answered from the local scan, got %q", id)
	}
}

func TestArtifactNew_ResolveID_UnclassifiedGitErrorSurfacesInsteadOfFallingBack(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "SPEC-007-foo.spec.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Tag creation failed locally: the error is neither FallbackError nor
	// RetriesExhaustedError. Git itself is working, so quietly scanning the filesystem
	// instead would reserve nothing on the remote while reporting success.
	mock := &mockGitExecutor{
		isRepo:       true,
		isAvailable:  true,
		createTagErr: fmt.Errorf("fatal: gpg failed to sign the data"),
	}
	id, err := ResolveID("spec", IDOptions{
		Root:       resolvedRootAt(t, tmpDir),
		Executor:   mock,
		MaxRetries: 3,
	})
	if err == nil {
		t.Fatalf("expected the tag-creation failure to surface, got id %q", id)
	}
	if isFallbackError(err) || isRetriesExhausted(err) {
		t.Fatalf("expected an unclassified error passed through verbatim, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "gpg failed to sign") {
		t.Fatalf("expected the underlying git error to survive to the caller, got %q", err.Error())
	}
	if id != "" {
		t.Fatalf("expected no id: the local scan must not answer for a working git, got %q", id)
	}
}
