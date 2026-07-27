package distribution

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// strictSemverPattern is the corpus's strict version convention: three numeric
// components, nothing else. Prerelease and build-metadata suffixes are versions
// under the wider semver grammar but are NOT release versions, and a resolution
// that offered one would move a consumer onto an unreleased pack.
//
// It is the shape pkg/validate/bundle.go already applies to artifact versions, so
// a tag and a bundle requirement version mean the same thing. No module
// dependency is introduced for it.
const strictSemverPattern = `^\d+\.\d+\.\d+$`

// versionTagPrefix is the SINGLE optional leading character a release tag may
// carry. Exactly one is normalization; a second is not a version.
const versionTagPrefix = "v"

// versionComponentCount is how many numeric components a strict version has.
const versionComponentCount = 3

// TagVersionResolver is the production VersionResolver: it resolves versions from
// the tags a pack's repository publishes.
//
// The cloner field is UNEXPORTED so a resolver cannot be assembled by composite
// literal without one — the same structural rule the lifecycle command
// constructors apply. NewTagVersionResolver is the only way to build one.
type TagVersionResolver struct {
	git GitCloner
}

// NewTagVersionResolver assembles the production resolver over a cloner.
//
// It FAILS CLOSED: a nil cloner yields a *MissingDependencyError naming the
// dependency rather than a resolver that would nil-dereference at its first
// listing. It supplies no fallback cloner of its own — an internal default is
// exactly what makes a test double mistakable for production wiring.
func NewTagVersionResolver(git GitCloner) (*TagVersionResolver, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "TagVersionResolver", Dependency: "git cloner"}
	}

	return &TagVersionResolver{git: git}, nil
}

// ResolveLatestCompatible returns the highest released version sharing
// currentVersion's MAJOR component, or currentVersion unchanged when the
// repository publishes nothing newer.
//
// The repository URL is built from the COORDINATE it is PASSED, so resolution reaches the
// same repository every other lifecycle operation reaches (SPEC-056 REQ-005). It used to
// be derived from the pack NAME, which is wrong for any pack whose manifest name differs
// from its repository — exactly the divergence REQ-003 made expressible.
//
// A listing failure PROPAGATES. It must never come back as currentVersion with a
// nil error: an operator reads that as "already at the latest version", which is
// a lie when the truth is that the remote could not be reached.
func (r *TagVersionResolver) ResolveLatestCompatible(coordinate, currentVersion string) (string, error) {
	strictSemver := regexp.MustCompile(strictSemverPattern)

	current, ok := parseVersionComponents(currentVersion, strictSemver)
	if !ok {
		return "", fmt.Errorf("cannot resolve versions for %s: its current version %q is not a strict MAJOR.MINOR.PATCH version", coordinate, currentVersion)
	}

	url := resolveGitURL(coordinate)

	tags, err := r.git.ListTags(url)
	if err != nil {
		return "", fmt.Errorf("cannot resolve the latest compatible version of %s: %w", coordinate, err)
	}

	latest := current
	resolved := currentVersion

	for _, tag := range tags {
		candidate, parsed := parseVersionComponents(tag, strictSemver)
		switch {
		case !parsed:
			// Prerelease, build-metadata, peeled, and arbitrary tags land here.
			continue
		case candidate.major != current.major:
			// The same-major rule, applied LITERALLY — including at major zero,
			// where 0.1.0 and 0.2.0 are compatible. That is BUNDLE-006's
			// enforcement-semver reading, not an oversight of the 0.x convention.
			continue
		case !candidate.newerThan(latest):
			continue
		}

		latest = candidate
		resolved = candidate.String()
	}

	return resolved, nil
}

// IsMajorBump reports whether two versions differ in their major component.
//
// A version that does not parse is reported AS a bump: the update path uses this
// to demand an explicit acknowledgment, so an unreadable version must take the
// path that asks rather than the one that proceeds silently.
func (r *TagVersionResolver) IsMajorBump(current, resolved string) bool {
	strictSemver := regexp.MustCompile(strictSemverPattern)

	currentComponents, currentParsed := parseVersionComponents(current, strictSemver)
	resolvedComponents, resolvedParsed := parseVersionComponents(resolved, strictSemver)
	if !currentParsed || !resolvedParsed {
		return true
	}

	return currentComponents.major != resolvedComponents.major
}

// versionComponents is one parsed strict version.
type versionComponents struct {
	major int
	minor int
	patch int
}

// String renders the version in its normalized form — no tag prefix — so a
// resolved version is directly comparable to the version recorded in a manifest.
func (v versionComponents) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

// newerThan compares components NUMERICALLY, so 1.10.0 is newer than 1.2.0. A
// lexical comparison of tag strings gets that backwards.
func (v versionComponents) newerThan(other versionComponents) bool {
	switch {
	case v.major != other.major:
		return v.major > other.major
	case v.minor != other.minor:
		return v.minor > other.minor
	default:
		return v.patch > other.patch
	}
}

// parseVersionComponents parses a tag or version into its components, reporting
// whether it is a strict release version at all.
//
// TrimPrefix strips at most ONE leading prefix character, so "vv1.0.0" survives
// as "v1.0.0" and then fails the pattern — which is the intended reading: a
// doubled prefix is not a version.
func parseVersionComponents(value string, strictSemver *regexp.Regexp) (versionComponents, bool) {
	normalized := strings.TrimPrefix(value, versionTagPrefix)
	if !strictSemver.MatchString(normalized) {
		return versionComponents{}, false
	}

	parts := strings.SplitN(normalized, ".", versionComponentCount)
	numbers := make([]int, 0, versionComponentCount)
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			// The pattern already established these are digits, so this can only be
			// a value too large to hold. Such a tag is not a version anyone released.
			return versionComponents{}, false
		}
		numbers = append(numbers, number)
	}

	return versionComponents{major: numbers[0], minor: numbers[1], patch: numbers[2]}, true
}
