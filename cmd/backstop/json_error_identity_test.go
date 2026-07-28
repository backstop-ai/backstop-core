package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-009 classification suite: SPEC-056's three typed refusals get stable wire
// kinds under --json.
//
// THREE TYPES, TWO KINDS, DELIBERATELY. VersionUnresolvedError and VersionMismatchError
// share "version" because both mean the version is wrong; IdentityError is separate
// because its REMEDY differs — fix the manifest versus retag the repository — and a
// consumer switching on kind has to be able to tell those apart.
//
// EVERY CASE IS ASSERTED THROUGH A WRAP. The pipelines wrap their typed errors for
// context (fmt.Errorf("pack add %s: %w", …)), so the bare value is the rare shape and the
// wrapped one is what a real failure looks like. A classifier written with a type
// ASSERTION rather than errors.As classifies the bare values correctly and sends every
// wrapped one to "unknown" — which is exactly the bug this file's own doc comment warns
// about, and only a wrapped fixture can catch it.

func TestClassifyJSONErrorKind_VersionUnresolvedIsVersionKind(t *testing.T) {
	bare := &distribution.VersionUnresolvedError{Reference: "acme/pack", Problem: "no version supplied"}

	if got := classifyJSONErrorKind(bare); got != "version" {
		t.Errorf("bare *VersionUnresolvedError classified as %q, want %q", got, "version")
	}
	wrapped := fmt.Errorf("pack add %s: %w", "acme/pack", bare)
	if got := classifyJSONErrorKind(wrapped); got != "version" {
		t.Errorf("WRAPPED *VersionUnresolvedError classified as %q, want %q — a type assertion instead of errors.As sends every wrapped failure to unknown, and wrapped is what a real failure looks like", got, "version")
	}
	// Two levels deep: the pipelines sometimes wrap twice on the way out.
	if got := classifyJSONErrorKind(fmt.Errorf("outer: %w", wrapped)); got != "version" {
		t.Errorf("doubly-wrapped *VersionUnresolvedError classified as %q, want %q", got, "version")
	}
}

func TestClassifyJSONErrorKind_VersionMismatchIsVersionKind(t *testing.T) {
	bare := &distribution.VersionMismatchError{
		Coordinate: "acme/pack", Tag: "v1.0.0", ManifestVersion: "1.0.1", ExpectedVersion: "1.0.0",
	}

	if got := classifyJSONErrorKind(bare); got != "version" {
		t.Errorf("bare *VersionMismatchError classified as %q, want %q", got, "version")
	}
	if got := classifyJSONErrorKind(fmt.Errorf("pack update %s: %w", "acme/pack", bare)); got != "version" {
		t.Errorf("WRAPPED *VersionMismatchError classified as %q, want %q", got, "version")
	}
}

func TestClassifyJSONErrorKind_IdentityErrorIsIdentityKind(t *testing.T) {
	bare := &distribution.IdentityError{
		Coordinate: "acme/pack", Tag: "v1.0.0", Field: "name", Problem: "must contain exactly one slash",
	}

	if got := classifyJSONErrorKind(bare); got != "identity" {
		t.Errorf("bare *IdentityError classified as %q, want %q", got, "identity")
	}
	if got := classifyJSONErrorKind(fmt.Errorf("pack add %s: %w", "acme/pack", bare)); got != "identity" {
		t.Errorf("WRAPPED *IdentityError classified as %q, want %q", got, "identity")
	}
	// It must NOT collapse into the version kind: the two have different remedies.
	if got := classifyJSONErrorKind(bare); got == "version" {
		t.Error("*IdentityError classified as the version kind; a consumer cannot tell 'fix the manifest' from 'retag the repository'")
	}
}

// TestClassifyJSONErrorKind_ExistingKindsUnchanged pins the four kinds SPEC-055 built, so
// a new arm inserted in the wrong position — ahead of a more specific one, or shadowing
// it — reds here rather than silently reclassifying an existing consumer's failures.
func TestClassifyJSONErrorKind_ExistingKindsUnchanged(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"git", &distribution.GitError{Message: "clone failed"}, "git"},
		{"validation", &distribution.ValidationError{Message: "pack check failed"}, "validation"},
		{"dependency", &distribution.MissingDependencyError{Command: "AddCommand", Dependency: "git cloner"}, "dependency"},
		{"capability", &distribution.CapabilityUnavailableError{Capability: "scan", Reference: "BUNDLE-006 REQ-014"}, "capability"},
		{"unknown", errors.New("something else entirely"), "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyJSONErrorKind(tc.err); got != tc.want {
				t.Errorf("classified as %q, want the unchanged %q", got, tc.want)
			}
			if got := classifyJSONErrorKind(fmt.Errorf("wrapped: %w", tc.err)); got != tc.want {
				t.Errorf("wrapped, classified as %q, want the unchanged %q", got, tc.want)
			}
		})
	}
}
