package distribution_test

// Behavior suite for the production VersionResolver (SPEC-055 REQ-004).
//
// These tests drive resolution over a CONTROLLED tag slice, so each selection
// rule is stated against a listing built to falsify exactly it. The real-git half
// — resolution over an actual remote tag listing — is the companion suite in
// versionresolver_real_test.go; neither substitutes for the other, because a stub
// cannot prove the resolver reaches a repository and a repository cannot be
// arranged to hold every adversarial tag shape cheaply.

import (
	"errors"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// tagListingCloner is a GitCloner that answers a tag listing from a fixed slice
// and records the URLs it was asked about.
//
// Recording the URL is what lets a test assert the resolver resolved the PACK
// COORDINATE through the package's own URL construction rather than being handed
// a URL — a resolver that took a URL directly would never exercise that path.
type tagListingCloner struct {
	t            *testing.T
	tags         []string
	listingError error
	askedURLs    []string
}

// Clone fails the test: a version resolution reads tags and clones nothing. A
// resolver that cloned to inspect versions would be doing remote work no
// resolution needs, and a silent stub would hide it.
func (c *tagListingCloner) Clone(url, version, destDir string) error {
	c.t.Helper()
	c.t.Errorf("the resolver cloned %s at %s into %s; resolving a version must only list tags", url, version, destDir)

	return errors.New("the version resolver must not clone")
}

func (c *tagListingCloner) ListTags(url string) ([]string, error) {
	c.askedURLs = append(c.askedURLs, url)
	if c.listingError == nil {
		return c.tags, nil
	}

	// The configured failure is returned VERBATIM. It is not a cause propagated
	// from a callee — it is the listing failure this stub exists to simulate — and
	// the test asserts the RESOLVER's wrap carries it, so adding a wrap here would
	// move the thing under test into the stub.
	return nil, c.listingError
}

// resolverOverTags builds a resolver over a listing stub, failing the test if
// assembly does not succeed — assembly failure is a separate claim, and letting
// it surface as a nil-pointer panic inside a resolution test would misattribute
// the defect.
func resolverOverTags(t *testing.T, cloner *tagListingCloner) *distribution.TagVersionResolver {
	t.Helper()

	resolver, err := distribution.NewTagVersionResolver(cloner)
	if err != nil {
		t.Fatalf("assembling a resolver over a non-nil cloner failed: %v", err)
	}

	return resolver
}

// resolve runs a resolution and fails the test on error, returning the resolved
// version.
func resolve(t *testing.T, resolver *distribution.TagVersionResolver, packName, currentVersion string) string {
	t.Helper()

	resolved, err := resolver.ResolveLatestCompatible(packName, currentVersion)
	if err != nil {
		t.Fatalf("resolving the latest compatible version of %s from %s: %v", packName, currentVersion, err)
	}

	return resolved
}

// TestTagVersionResolver_ResolvesHighestSameMajor proves resolution selects the
// highest tag sharing the current major (CLM-020).
//
// The listing carries 1.10.0 alongside 1.2.0 and is presented out of order, so a
// resolver comparing tags as STRINGS picks 1.2.0 and a resolver returning the
// last matching tag picks 1.1.0. Only numeric component comparison passes.
func TestTagVersionResolver_ResolvesHighestSameMajor(t *testing.T) {
	packName := "hermetic/resolution"
	cloner := &tagListingCloner{t: t, tags: []string{"v1.2.0", "v1.10.0", "v1.0.0", "v1.1.0"}}
	resolver := resolverOverTags(t, cloner)

	if got := resolve(t, resolver, packName, "1.0.0"); got != "1.10.0" {
		t.Errorf("resolved %q, want the highest same-major tag 1.10.0", got)
	}

	// The pack COORDINATE, not a URL, is what the resolver was given, so the
	// listing it asked for must be the one production's URL construction yields.
	if len(cloner.askedURLs) != 1 {
		t.Fatalf("the resolver made %d tag listings, want exactly 1: %v", len(cloner.askedURLs), cloner.askedURLs)
	}
	if !strings.Contains(cloner.askedURLs[0], packName) {
		t.Errorf("the resolver listed tags for %q, which does not name the pack %q; it did not resolve the pack's git URL", cloner.askedURLs[0], packName)
	}
}

// TestTagVersionResolver_DoesNotCrossMajorBoundary proves a higher-major tag is
// present and is NOT selected (CLM-021). Without the higher-major tag in the
// listing, "never crosses a major" would be untestable.
func TestTagVersionResolver_DoesNotCrossMajorBoundary(t *testing.T) {
	cloner := &tagListingCloner{t: t, tags: []string{"v1.0.0", "v1.1.0", "v2.0.0", "v3.5.1"}}
	resolver := resolverOverTags(t, cloner)

	if got := resolve(t, resolver, "hermetic/resolution", "1.0.0"); got != "1.1.0" {
		t.Errorf("resolved %q from a listing containing 2.0.0 and 3.5.1, want the same-major 1.1.0", got)
	}
}

// TestTagVersionResolver_ZeroMajorMinorIsCompatible proves the same-major rule
// applies LITERALLY at major zero: 0.1.0 with a v0.2.0 available resolves to
// 0.2.0 (CLM-022).
//
// This is a DECISION, not a default. Strict semver convention would call a 0.x
// minor bump breaking, but BUNDLE-006's enforcement-semver model is about what a
// version change means for RULES and states no 0.x caveat. Changing this test to
// treat 0.x minors as major-equivalent is a spec violation, not a fix — the
// disagreement belongs in a bundle revision.
func TestTagVersionResolver_ZeroMajorMinorIsCompatible(t *testing.T) {
	cloner := &tagListingCloner{t: t, tags: []string{"v0.1.0", "v0.2.0", "v1.0.0"}}
	resolver := resolverOverTags(t, cloner)

	if got := resolve(t, resolver, "hermetic/resolution", "0.1.0"); got != "0.2.0" {
		t.Errorf("resolved %q from 0.1.0, want 0.2.0 — the same-major rule applies literally at major zero", got)
	}
}

// TestTagVersionResolver_NormalizesOptionalVPrefix proves a SINGLE optional
// leading "v" is stripped and that the resolved version is returned without it
// (CLM-023).
//
// The listing mixes prefixed and bare tags, so a resolver that only handled one
// form silently ignores half the listing. "vv1.9.0" is present and must be
// ignored: exactly one leading v is normalization, two is not a version.
func TestTagVersionResolver_NormalizesOptionalVPrefix(t *testing.T) {
	cloner := &tagListingCloner{t: t, tags: []string{"v1.1.0", "1.2.0", "vv1.9.0"}}
	resolver := resolverOverTags(t, cloner)

	got := resolve(t, resolver, "hermetic/resolution", "1.0.0")
	if got == "1.9.0" || got == "vv1.9.0" {
		t.Fatalf("resolved %q; a doubled prefix is not a version and must be ignored", got)
	}
	if got != "1.2.0" {
		t.Errorf("resolved %q, want the normalized 1.2.0 — both prefixed and bare tags count, and the result carries no prefix", got)
	}
}

// TestTagVersionResolver_IgnoresNonStrictSemverTags proves prerelease, build
// metadata, peeled refs, and arbitrary tags are all ignored (CLM-024).
//
// Every ignorable shape is HIGHER than the answer, so a resolver with a loose
// filter selects one of them and fails. The peeled "1.9.0^{}" entry is included
// even though the production cloner already drops peeled refs: the resolver's own
// filter is what this test covers, and a listing from any other source would
// carry them.
func TestTagVersionResolver_IgnoresNonStrictSemverTags(t *testing.T) {
	cloner := &tagListingCloner{t: t, tags: []string{
		"1.9.0-rc1",
		"1.8.0+build",
		"1.9.0^{}",
		"release-2",
		"latest",
		"1.1.0",
	}}
	resolver := resolverOverTags(t, cloner)

	if got := resolve(t, resolver, "hermetic/resolution", "1.0.0"); got != "1.1.0" {
		t.Errorf("resolved %q, want 1.1.0 — every higher tag in the listing is non-strict semver and must be ignored", got)
	}
}

// TestTagVersionResolver_NoNewerCompatibleReturnsCurrent proves the current
// version comes back unchanged when nothing newer is compatible (CLM-025).
//
// The listing is not empty: it holds an OLDER same-major tag and a higher tag of
// a different major, so this distinguishes "nothing newer" from "nothing at all".
func TestTagVersionResolver_NoNewerCompatibleReturnsCurrent(t *testing.T) {
	cloner := &tagListingCloner{t: t, tags: []string{"v1.0.0", "v1.1.0", "v2.4.0"}}
	resolver := resolverOverTags(t, cloner)

	if got := resolve(t, resolver, "hermetic/resolution", "1.1.0"); got != "1.1.0" {
		t.Errorf("resolved %q from 1.1.0 with nothing newer in-major, want 1.1.0 unchanged", got)
	}
}

// TestTagVersionResolver_ListTagsFailurePropagatesAsError proves a listing
// failure propagates and is NEVER reported as already-at-the-latest (CLM-026).
//
// Both halves matter: the error must be non-nil AND the returned version must not
// be the current one dressed up as a successful no-op. A resolver that swallowed
// the failure and returned currentVersion would tell an operator their pack is up
// to date while the remote was unreachable.
func TestTagVersionResolver_ListTagsFailurePropagatesAsError(t *testing.T) {
	listingFailure := errors.New("the remote refused the tag listing")
	cloner := &tagListingCloner{t: t, listingError: listingFailure}
	resolver := resolverOverTags(t, cloner)

	resolved, err := resolver.ResolveLatestCompatible("hermetic/resolution", "1.0.0")
	if err == nil {
		t.Fatalf("resolution returned %q with no error; a failed listing must never surface as a version", resolved)
	}
	if resolved == "1.0.0" {
		t.Errorf("resolution returned the current version %q alongside its error; that reads as already-at-the-latest", resolved)
	}
	if !errors.Is(err, listingFailure) {
		t.Errorf("the propagated error %v does not wrap the underlying listing failure, so the cause is lost", err)
	}
}

// TestTagVersionResolver_IsMajorBump_TrueAcrossMajors proves IsMajorBump reports
// true for versions differing in major (CLM-027), including the 0 → 1 crossing
// that the literal zero-major rule makes easy to get wrong.
func TestTagVersionResolver_IsMajorBump_TrueAcrossMajors(t *testing.T) {
	resolver := resolverOverTags(t, &tagListingCloner{t: t})

	cases := [][2]string{
		{"1.2.3", "2.0.0"},
		{"0.9.0", "1.0.0"},
		{"2.0.0", "1.0.0"},
	}
	for _, pair := range cases {
		if !resolver.IsMajorBump(pair[0], pair[1]) {
			t.Errorf("IsMajorBump(%q, %q) = false, want true — the major components differ", pair[0], pair[1])
		}
	}
}

// TestTagVersionResolver_IsMajorBump_FalseWithinMajor proves IsMajorBump reports
// false for versions sharing a major (CLM-028).
//
// The 0.1.0 → 0.2.0 pair is the one that pins the literal zero-major reading:
// under a strict-semver reading it would be a breaking change and this would be
// true, which would make every 0.x minor update demand an acknowledgment.
func TestTagVersionResolver_IsMajorBump_FalseWithinMajor(t *testing.T) {
	resolver := resolverOverTags(t, &tagListingCloner{t: t})

	cases := [][2]string{
		{"1.2.3", "1.9.0"},
		{"1.2.3", "1.2.3"},
		{"0.1.0", "0.2.0"},
	}
	for _, pair := range cases {
		if resolver.IsMajorBump(pair[0], pair[1]) {
			t.Errorf("IsMajorBump(%q, %q) = true, want false — the major components match", pair[0], pair[1])
		}
	}
}

// TestNewTagVersionResolver_CompleteAssemblySucceeds proves a complete assembly
// yields a WORKING resolver (CLM-090).
//
// It resolves after constructing rather than only checking for a non-nil pointer:
// a constructor that returned a value whose cloner field was never populated
// would satisfy a nil check and then panic at the first listing.
func TestNewTagVersionResolver_CompleteAssemblySucceeds(t *testing.T) {
	cloner := &tagListingCloner{t: t, tags: []string{"v1.1.0"}}

	resolver, err := distribution.NewTagVersionResolver(cloner)
	if err != nil {
		t.Fatalf("NewTagVersionResolver with a cloner returned %v, want a resolver", err)
	}
	if resolver == nil {
		t.Fatal("NewTagVersionResolver returned a nil resolver and a nil error")
	}

	if got := resolve(t, resolver, "hermetic/resolution", "1.0.0"); got != "1.1.0" {
		t.Errorf("the assembled resolver resolved %q, want 1.1.0; its cloner was not wired", got)
	}
}

// TestNewTagVersionResolver_NilGitClonerNamesDependency proves the constructor
// fails closed on a nil cloner with a *MissingDependencyError that NAMES the git
// cloner (CLM-047).
//
// Asserting the NAME is the point: "an error occurred" passes even when the
// constructor names the wrong dependency, and the whole value of a typed
// assembly error is that it identifies the wiring site.
func TestNewTagVersionResolver_NilGitClonerNamesDependency(t *testing.T) {
	resolver, err := distribution.NewTagVersionResolver(nil)
	if err == nil {
		t.Fatal("NewTagVersionResolver(nil) returned no error; a partially assembled resolver would nil-dereference at its first listing")
	}
	if resolver != nil {
		t.Errorf("NewTagVersionResolver(nil) returned a resolver alongside its error; a failed assembly must yield nothing usable")
	}

	var missing *distribution.MissingDependencyError
	if !errors.As(err, &missing) {
		t.Fatalf("NewTagVersionResolver(nil) returned %T (%v), want a *distribution.MissingDependencyError", err, err)
	}

	if !strings.Contains(strings.ToLower(missing.Dependency), "git cloner") {
		t.Errorf("the error names the missing dependency as %q, which does not identify the git cloner", missing.Dependency)
	}
	if !strings.Contains(missing.Error(), missing.Dependency) {
		t.Errorf("the rendered message %q does not carry the dependency name %q", missing.Error(), missing.Dependency)
	}
	if missing.Command == "" {
		t.Error("the error names no command, so the diagnostic does not identify the wiring site being assembled")
	}
}

// TestCapabilityUnavailableError_NamesCapabilityAndTrackingReference covers the
// second typed assembly error declared alongside MissingDependencyError.
//
// It lives in this suite because both errors land together with the fail-closed
// constructor that needed one of them, while the production implementations that
// RETURN a CapabilityUnavailableError arrive with the upgrade pipeline in a later
// phase. Leaving its rendering unexercised until then would ship a diagnostic
// nothing has ever read.
//
// The assertion is that the message carries BOTH data: the capability alone reads
// as a defect report, and it is the tracking reference that tells an operator the
// capability is scheduled work rather than something broken.
func TestCapabilityUnavailableError_NamesCapabilityAndTrackingReference(t *testing.T) {
	unavailable := &distribution.CapabilityUnavailableError{
		Capability: "violation scanning",
		Reference:  "pack-distribution-lifecycle:REQ-014",
	}

	message := unavailable.Error()
	if !strings.Contains(message, unavailable.Capability) {
		t.Errorf("the message %q does not name the capability %q", message, unavailable.Capability)
	}
	if !strings.Contains(message, unavailable.Reference) {
		t.Errorf("the message %q does not name the tracking reference %q, so it reads as a defect rather than as scheduled work", message, unavailable.Reference)
	}

	var capability *distribution.CapabilityUnavailableError
	if !errors.As(error(unavailable), &capability) {
		t.Fatal("a CapabilityUnavailableError is not recoverable with errors.As; the consumers that classify it could not")
	}
}
