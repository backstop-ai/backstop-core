package validate

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// TestValidateResolvedBy_PRURLAccepted (CLM-005): a GitHub-style PR URL is a valid
// commit/PR ref — accepted shape-only, no existence check, no malformed.
func TestValidateResolvedBy_PRURLAccepted(t *testing.T) {
	art := &artifact.ParsedArtifact{Filename: "ISSUE-839-pr.issue.md", SourcePath: "issues/ISSUE-839-pr.issue.md"}
	if vs := validateResolvedBy(art, "https://github.com/backstop-ai/backstop-core/pull/42"); len(vs) != 0 {
		t.Fatalf("a PR URL must be accepted shape-only, got: %v", vs)
	}
}

// TestIsPullRequestURL covers the PR-URL shape check across its branches.
func TestIsPullRequestURL(t *testing.T) {
	cases := []struct {
		ref  string
		want bool
	}{
		{"https://github.com/org/repo/pull/42", true},
		{"https://example.com/x/pulls/7", true},
		{"http://github.com/org/repo/pull/42", false}, // not https
		{"https://github.com/org/repo/issues/42", false},
		{"a1b2c3d", false},
	}
	for _, c := range cases {
		if got := isPullRequestURL(c.ref); got != c.want {
			t.Errorf("isPullRequestURL(%q) = %v, want %v", c.ref, got, c.want)
		}
	}
}

// TestTypedRefArtifactExists_UnresolvablePaths covers the existence-check branches that
// return false without a filesystem hit: an empty SourcePath, a directory-less
// (base-only) SourcePath, and an unknown typed-ref prefix.
func TestTypedRefArtifactExists_UnresolvablePaths(t *testing.T) {
	// Empty SourcePath — cannot anchor a sibling dir.
	empty := &artifact.ParsedArtifact{Filename: "ISSUE-831-x.issue.md", SourcePath: ""}
	if typedRefArtifactExists(empty, "ISSUE-838") {
		t.Error("empty SourcePath must not resolve any sibling artifact")
	}

	// Directory-less (base-only) SourcePath — filepath.Dir == "." — cannot anchor.
	baseOnly := &artifact.ParsedArtifact{Filename: "ISSUE-831-x.issue.md", SourcePath: "ISSUE-831-x.issue.md"}
	if typedRefArtifactExists(baseOnly, "ISSUE-838") {
		t.Error("base-only SourcePath must not resolve any sibling artifact")
	}

	// Unknown typed-ref prefix — no sibling dir mapping (defensive; the caller's
	// typed-ref regex normally gates this, so this exercises the map-miss branch).
	anchored := &artifact.ParsedArtifact{Filename: "ISSUE-831-x.issue.md", SourcePath: "root/issues/ISSUE-831-x.issue.md"}
	if typedRefArtifactExists(anchored, "FOO-001") {
		t.Error("an unknown typed-ref prefix must not resolve any sibling dir")
	}
}
