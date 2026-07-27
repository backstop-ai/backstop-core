package distribution_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// The REQ-001 unit suite: exactly ONE effective version is resolved before git runs.
//
// NO GIT ANYWHERE IN THIS FILE, and that is a claim rather than a convenience.
// ResolveEffectiveVersion is a free function precisely because refusing a malformed
// reference needs no dependency — no cloner, no resolver, no network. A test that had to
// build a cloner to reach this behavior would be evidence the refusal happens too late,
// which is the defect REQ-001 exists to fix: today parsePackRef returns an empty version
// for a bare org/name and the pipeline clones the ref "v", so the operator's diagnostic
// is a git error about a nonexistent branch.
//
// COORDINATE PURITY IS ASSERTED IN EVERY PASSING CASE. The returned coordinate must have
// ONLY the @version suffix removed — mixed case preserved, a -pack suffix preserved, no
// host normalization of any kind. CLM-043 re-asserts this at the lock level in a later
// phase; asserting it here too is what localizes a regression to one function instead of
// leaving it to surface as a wrong lock entry.

// mixedCaseRef is a coordinate carrying BOTH properties a normalization would destroy:
// upper-case letters and a -pack repository suffix. It is the real shape of the pack that
// motivated this spec.
const (
	mixedCaseRef        = "Backstop-AI/backstop-harness-toolchain-pack"
	mixedCaseRefVersion = mixedCaseRef + "@1.2.3"
)

// assertCoordinateVerbatim requires the coordinate to be byte-identical to want. Case
// folding is a GitHub property; packs may be hosted anywhere, and DD-31 removed that
// assumption deliberately.
func assertCoordinateVerbatim(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("coordinate = %q, want %q byte-for-byte — no case folding, no suffix stripping, no host normalization", got, want)
	}
}

// assertPrefixFree requires the RECORDED version to carry no leading prefix, which is
// what makes re-prefixing produce exactly one. A recorded "v1.2.3" would clone "vv1.2.3".
func assertPrefixFree(t *testing.T, version string) {
	t.Helper()
	if strings.HasPrefix(version, "v") {
		t.Errorf("recorded version %q carries a leading prefix; the recorded form must be bare X.Y.Z so the cloned tag carries EXACTLY one", version)
	}
}

func TestResolveEffectiveVersion_RefSuffixSuppliesVersion(t *testing.T) {
	coordinate, version, err := distribution.ResolveEffectiveVersion(mixedCaseRefVersion, "")
	if err != nil {
		t.Fatalf("ResolveEffectiveVersion(%q, \"\") = %v, want the @suffix to supply the version", mixedCaseRefVersion, err)
	}

	assertCoordinateVerbatim(t, coordinate, mixedCaseRef)
	if version != "1.2.3" {
		t.Errorf("version = %q, want %q", version, "1.2.3")
	}
	assertPrefixFree(t, version)

	// The tag the caller hands to Clone is the recorded version re-prefixed with exactly
	// one "v" (Clone(url, "v"+version, dst)). Asserting it here pins the pairing that
	// makes a doubled prefix impossible downstream.
	if tag := "v" + version; tag != "v1.2.3" {
		t.Errorf("cloned tag = %q, want %q — exactly one prefix", tag, "v1.2.3")
	}
}

func TestResolveEffectiveVersion_FlagSuppliesVersionWhenRefHasNone(t *testing.T) {
	coordinate, version, err := distribution.ResolveEffectiveVersion("acme/pack", "1.2.3")
	if err != nil {
		t.Fatalf("ResolveEffectiveVersion(\"acme/pack\", \"1.2.3\") = %v, want the flag to supply the version", err)
	}
	assertCoordinateVerbatim(t, coordinate, "acme/pack")
	if version != "1.2.3" {
		t.Errorf("version = %q, want %q", version, "1.2.3")
	}
}

func TestResolveEffectiveVersion_FlagOverridesDisagreeingRefSuffix(t *testing.T) {
	coordinate, version, err := distribution.ResolveEffectiveVersion("acme/pack@2.0.0", "1.2.3")
	if err != nil {
		t.Fatalf("ResolveEffectiveVersion with a disagreeing flag = %v, want the flag to win", err)
	}
	assertCoordinateVerbatim(t, coordinate, "acme/pack")
	// --version documents itself as "overrides version in pack reference"
	// (pack_add.go:57). The flag winning is that contract, not a tiebreak invented here.
	if version != "1.2.3" {
		t.Errorf("version = %q, want the flag's %q to override the ref's %q", version, "1.2.3", "2.0.0")
	}
}

// TestResolveEffectiveVersion_AgreeingRefAndFlagResolveOnce is claimed SEPARATELY from
// CLM-003 on purpose: an implementation that ERRORED on any dual supply — treating "both
// given" as ambiguous — would still satisfy CLM-003's "the flag wins" when they disagree.
// Only this case distinguishes "the flag wins" from "two sources are an error".
func TestResolveEffectiveVersion_AgreeingRefAndFlagResolveOnce(t *testing.T) {
	coordinate, version, err := distribution.ResolveEffectiveVersion("acme/pack@1.2.3", "1.2.3")
	if err != nil {
		t.Fatalf("ResolveEffectiveVersion with an AGREEING ref and flag = %v, want no error — dual supply that agrees is not ambiguity", err)
	}
	assertCoordinateVerbatim(t, coordinate, "acme/pack")
	if version != "1.2.3" {
		t.Errorf("version = %q, want %q", version, "1.2.3")
	}
}

func TestResolveEffectiveVersion_NoVersionAnywhereIsTypedRefusal(t *testing.T) {
	const ref = "acme/pack"
	_, _, err := distribution.ResolveEffectiveVersion(ref, "")
	if err == nil {
		t.Fatal("a bare org/name with no --version must refuse; today it clones the ref \"v\" and the operator sees a git error about a nonexistent branch")
	}

	var unresolved *distribution.VersionUnresolvedError
	if !errors.As(err, &unresolved) {
		t.Fatalf("error is %T (%v), want *VersionUnresolvedError — the refusal must be typed so --json can classify it", err, err)
	}

	msg := err.Error()
	if !strings.Contains(msg, ref) {
		t.Errorf("diagnostic %q does not name the reference the operator typed", msg)
	}
	// BOTH remedies, because an operator who is told only "no version" does not know
	// which of the two ways to supply one this command accepts.
	if !strings.Contains(msg, "@") {
		t.Errorf("diagnostic %q does not mention the @version suffix as a way to supply a version", msg)
	}
	if !strings.Contains(msg, "--version") {
		t.Errorf("diagnostic %q does not mention the --version flag as a way to supply a version", msg)
	}
}

func TestResolveEffectiveVersion_NonStrictVersionIsTypedRefusal(t *testing.T) {
	_, _, err := distribution.ResolveEffectiveVersion("acme/pack@1.0", "")
	var unresolved *distribution.VersionUnresolvedError
	if !errors.As(err, &unresolved) {
		t.Fatalf("ResolveEffectiveVersion(\"acme/pack@1.0\") = %v (%T), want *VersionUnresolvedError — MAJOR.MINOR is not a release version", err, err)
	}
}

// TestResolveEffectiveVersion_PrereleaseVersionIsTypedRefusal pins REFUSAL, not
// resolution. An implementation that quietly resolved 1.0.0-rc1 to some nearby released
// tag would move a consumer onto a pack they did not ask for.
func TestResolveEffectiveVersion_PrereleaseVersionIsTypedRefusal(t *testing.T) {
	_, _, err := distribution.ResolveEffectiveVersion("acme/pack@1.0.0-rc1", "")
	var unresolved *distribution.VersionUnresolvedError
	if !errors.As(err, &unresolved) {
		t.Fatalf("ResolveEffectiveVersion(\"acme/pack@1.0.0-rc1\") = %v (%T), want *VersionUnresolvedError", err, err)
	}
}

// TestResolveEffectiveVersion_DoubledPrefixIsTypedRefusal pins the ONE-prefix rule.
// parseVersionComponents (versionresolver.go:151) strips at most one prefix via
// TrimPrefix, so "vv1.0.0" survives as "v1.0.0" and then fails the strict pattern. That
// is the intended reading, not an accident of the helper: exactly one leading character
// is normalization, and a second one means the string is not a version.
func TestResolveEffectiveVersion_DoubledPrefixIsTypedRefusal(t *testing.T) {
	_, _, err := distribution.ResolveEffectiveVersion("acme/pack@vv1.0.0", "")
	var unresolved *distribution.VersionUnresolvedError
	if !errors.As(err, &unresolved) {
		t.Fatalf("ResolveEffectiveVersion(\"acme/pack@vv1.0.0\") = %v (%T), want *VersionUnresolvedError", err, err)
	}
}

// TestResolveEffectiveVersion_SingleLeadingPrefixNormalizes asserts BOTH halves. Checking
// only the recorded form would let an implementation that recorded "v1.2.3" through, and
// that implementation clones "vv1.2.3" — a tag no repository has.
func TestResolveEffectiveVersion_SingleLeadingPrefixNormalizes(t *testing.T) {
	coordinate, version, err := distribution.ResolveEffectiveVersion("acme/pack@v1.2.3", "")
	if err != nil {
		t.Fatalf("ResolveEffectiveVersion(\"acme/pack@v1.2.3\") = %v, want a single prefix to normalize away", err)
	}
	assertCoordinateVerbatim(t, coordinate, "acme/pack")

	if version != "1.2.3" {
		t.Errorf("recorded version = %q, want %q with NO prefix", version, "1.2.3")
	}
	assertPrefixFree(t, version)
	if tag := "v" + version; tag != "v1.2.3" {
		t.Errorf("cloned tag = %q, want %q — exactly one prefix, never two", tag, "v1.2.3")
	}
}
