package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GitExecutor abstracts git operations for testability.
type GitExecutor interface {
	ListTags(pattern string) ([]string, error)
	CreateAnnotatedTag(name, message string) error
	PushTag(name string) error
	FetchTags() error
	IsGitRepo() bool
	IsGitAvailable() bool
}

// TagConflictError indicates a tag push failed because the tag already exists
// on the remote (another developer reserved the same ID).
type TagConflictError struct {
	Tag string
}

func (e *TagConflictError) Error() string {
	return fmt.Sprintf("tag conflict: %s already exists on remote", e.Tag)
}

// FallbackError is a sentinel error indicating the caller should fall back
// to local filesystem scan.
type FallbackError struct {
	Reason string
}

func (e *FallbackError) Error() string {
	return fmt.Sprintf("git unavailable, falling back to local scan: %s", e.Reason)
}

// RetriesExhaustedError indicates all tag conflict retries have been used.
type RetriesExhaustedError struct {
	Attempts int
}

func (e *RetriesExhaustedError) Error() string {
	return fmt.Sprintf("git tag reservation failed: %d retries exhausted due to tag conflicts", e.Attempts)
}

// IDOptions holds configuration for ID resolution.
type IDOptions struct {
	ProjectRoot string
	Executor    GitExecutor
	MaxRetries  int
}

// GitTagResolver resolves the next available ID via git annotated tags.
type GitTagResolver struct {
	executor   GitExecutor
	maxRetries int
	// pushFunc allows overriding push behavior for testing (e.g., simulating conflicts).
	pushFunc func(name string) error
	// createFunc allows overriding tag creation for testing (e.g., capturing messages).
	createFunc func(name, message string) error
}

// Resolve finds the next available ID for the given artifact type via git tags.
// Returns the zero-padded numeric ID string (e.g., "003").
// Returns FallbackError if git is unavailable or non-conflict remote ops fail.
// Returns RetriesExhaustedError if all conflict retries are used.
func (r *GitTagResolver) Resolve(artifactType, slug string) (string, error) {
	if !r.executor.IsGitAvailable() || !r.executor.IsGitRepo() {
		return "", &FallbackError{Reason: "git not available or not a git repository"}
	}

	if err := r.executor.FetchTags(); err != nil {
		return "", &FallbackError{Reason: fmt.Sprintf("fetch failed: %v", err)}
	}

	cfg := ValidArtifactTypes[artifactType]
	pattern := fmt.Sprintf("backstop/%s/", artifactType)

	tags, err := r.executor.ListTags(pattern + "*")
	if err != nil {
		return "", &FallbackError{Reason: fmt.Sprintf("list tags failed: %v", err)}
	}

	maxNum := 0
	for _, tag := range tags {
		parts := strings.Split(tag, "/")
		if len(parts) < 3 {
			continue
		}
		numStr := parts[len(parts)-1]
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		if num > maxNum {
			maxNum = num
		}
	}

	nextNum := maxNum + 1
	attempts := 0

	for attempts <= r.maxRetries {
		idStr := fmt.Sprintf("%0*d", cfg.DigitCount, nextNum)
		tagName := fmt.Sprintf("backstop/%s/%s", artifactType, idStr)
		message := fmt.Sprintf("Reserved %s at %s (slug: %s)", tagName, time.Now().Format(time.RFC3339), slug)

		// Create the annotated tag
		if r.createFunc != nil {
			if err := r.createFunc(tagName, message); err != nil {
				return "", fmt.Errorf("creating tag: %w", err)
			}
		} else {
			if err := r.executor.CreateAnnotatedTag(tagName, message); err != nil {
				return "", fmt.Errorf("creating tag: %w", err)
			}
		}

		// Push the specific tag
		var pushErr error
		if r.pushFunc != nil {
			pushErr = r.pushFunc(tagName)
		} else {
			pushErr = r.executor.PushTag(tagName)
		}

		if pushErr == nil {
			return idStr, nil
		}

		// Check if this is a tag conflict error
		if _, ok := pushErr.(*TagConflictError); ok {
			attempts++
			nextNum++
			if attempts > r.maxRetries {
				return "", &RetriesExhaustedError{Attempts: attempts}
			}
			continue
		}

		// Non-conflict push failure: trigger fallback
		return "", &FallbackError{Reason: fmt.Sprintf("push failed: %v", pushErr)}
	}

	return "", &RetriesExhaustedError{Attempts: attempts}
}

// LocalScanResolver resolves the next available ID by scanning the local filesystem.
type LocalScanResolver struct{}

// Resolve scans the target directory for existing artifacts and returns the next ID.
func (r *LocalScanResolver) Resolve(artifactType, targetDir string) (string, error) {
	cfg := ValidArtifactTypes[artifactType]

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist yet; start at 001 (or 0001 for adr)
			return fmt.Sprintf("%0*d", cfg.DigitCount, 1), nil
		}
		return "", fmt.Errorf("reading directory %s: %w", targetDir, err)
	}

	// Build a regex to match artifact filenames and extract numeric IDs.
	// Pattern: PREFIX-(\d+)-.*
	prefix := cfg.IDPrefix
	idPattern := regexp.MustCompile(fmt.Sprintf(`^%s-(\d+)`, regexp.QuoteMeta(prefix)))

	maxNum := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := idPattern.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if num > maxNum {
			maxNum = num
		}
	}

	nextNum := maxNum + 1
	return fmt.Sprintf("%0*d", cfg.DigitCount, nextNum), nil
}

// ResolveID resolves the next available ID for the given artifact type.
// It tries git tag reservation first, falling back to local scan on FallbackError.
// RetriesExhaustedError is returned directly (not eligible for fallback).
func ResolveID(artifactType string, opts IDOptions) (string, error) {
	maxRetries := opts.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	gitResolver := &GitTagResolver{
		executor:   opts.Executor,
		maxRetries: maxRetries,
	}

	id, err := gitResolver.Resolve(artifactType, "")
	if err == nil {
		return id, nil
	}

	// Tag conflict exhaustion is not eligible for fallback
	if _, ok := err.(*RetriesExhaustedError); ok {
		return "", err
	}

	// Fallback to local scan
	if _, ok := err.(*FallbackError); ok {
		cfg := ValidArtifactTypes[artifactType]
		targetDir := filepath.Join(opts.ProjectRoot, cfg.Directory)
		localResolver := &LocalScanResolver{}
		return localResolver.Resolve(artifactType, targetDir)
	}

	return "", err
}

