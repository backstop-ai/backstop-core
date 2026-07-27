package distribution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/pack"
	"gopkg.in/yaml.v3"
)

// The identity gate (SPEC-056 REQ-001 through REQ-003, REQ-005).
//
// Everything here is a PURE FUNCTION over strings and a materialized directory. Nothing
// clones, nothing writes, and nothing touches the consumer project — which is precisely
// what lets REQ-007 place the whole gate before the first byte of consumer state is
// written. A refusal from this file means nothing has happened yet.
//
// THE FIVE CHECKS RUN IN A FIXED ORDER, and a refusal at any one means none of the later
// ones ran:
//
//	1. a version is available at all      → *VersionUnresolvedError
//	2. that version is strict X.Y.Z       → *VersionUnresolvedError
//	3. the manifest reads, with a name    → *IdentityError
//	4. the manifest version equals it     → *VersionMismatchError
//	5. the name is a usable identity      → *IdentityError
//
// Divergence between the manifest name and the requested coordinate is computed AFTER
// check five and is a DIAGNOSTIC, never a sixth refusal (REQ-006 / OQ-9 option (b)).
// Requiring equality was considered and rejected by name: the ten packs published under
// backstop-ai hold name == coordinate as a fleet CONVENTION, and a convention is
// something you notice breaking, not something the tool enforces.

// RemoteIdentity is the resolved identity of a pack about to be installed.
// @waiver:backstop/go-standards/backstop.packs.backstop.go-standards.rules.core.go.core.error-type-suffix:false-positive:2026-10-25 pack rule fix pending — RemoteIdentity is not an error type; the rule's dotall regex spans declarations and anchors on whatever struct precedes the file's first Error() method, which here is the correctly-suffixed VersionUnresolvedError
type RemoteIdentity struct {
	// Coordinate is the requested org/repository VERBATIM — the @version suffix
	// removed and nothing else changed.
	Coordinate string
	// EffectiveVersion is the normalized X.Y.Z carrying NO prefix. It is what gets
	// recorded.
	EffectiveVersion string
	// Tag is EffectiveVersion re-prefixed with exactly one "v", and is what Clone is
	// called with. Keeping the prefixed and unprefixed forms in separate fields is what
	// makes a doubled prefix unrepresentable downstream.
	Tag string
	// ManifestName is the name the cloned tree's pack.yml declares.
	ManifestName string
	// InstallName aliases ManifestName. It exists so the field that builds the install
	// path, the backstop.yml key, the lock key and the engine asset root READS as an
	// identity rather than as a name that happens to be reused (REQ-003 / DD-31).
	InstallName string
	// Diverged reports whether ManifestName differs from Coordinate. It is a FIELD
	// rather than something each caller recomputes, so the comparison rule — byte-exact,
	// never case-folded — lives in exactly one place.
	Diverged bool
}

// VersionUnresolvedError refuses a reference from which no single strict release version
// could be resolved, before any git subprocess runs.
type VersionUnresolvedError struct {
	Reference string
	Problem   string
}

func (e *VersionUnresolvedError) Error() string {
	return fmt.Sprintf(
		"cannot resolve a version for pack reference %q: %s; supply one as an @version suffix (%s@1.2.3) or with the --version flag",
		e.Reference, e.Problem, e.Reference)
}

// VersionMismatchError refuses a cloned tree whose manifest disagrees with the tag it was
// cloned from. It names both versions so an operator can see the fix is a RETAG of the
// repository rather than a reinstall.
type VersionMismatchError struct {
	Coordinate      string
	Tag             string
	ManifestVersion string
	ExpectedVersion string
}

func (e *VersionMismatchError) Error() string {
	return fmt.Sprintf(
		"pack %s at tag %s declares version %q in its pack.yml but %q was requested; the repository's manifest and its tag disagree — retag the repository, or request the version the manifest declares",
		e.Coordinate, e.Tag, e.ManifestVersion, e.ExpectedVersion)
}

// IdentityError refuses a cloned tree whose pack.yml cannot supply a usable identity. It
// names the COORDINATE and the TAG rather than the directory, because at the point this
// fires the pack.yml lives in a temporary clone the operator cannot inspect.
type IdentityError struct {
	Coordinate string
	Tag        string
	Field      string
	Problem    string
}

func (e *IdentityError) Error() string {
	return fmt.Sprintf("pack %s at tag %s has an unusable pack.yml %s: %s",
		e.Coordinate, e.Tag, e.Field, e.Problem)
}

// errManifestNameMissing distinguishes "this manifest declares no name" from "this
// manifest could not be read at all", so ValidateRemoteIdentity can attribute the failure
// to the right pack.yml field.
var errManifestNameMissing = errors.New("manifest declares no name") // nosemgrep: go.core.no-global-mutable-state — immutable sentinel error, never reassigned

// ResolveEffectiveVersion resolves exactly ONE version from the two places it may be
// supplied, and refuses BEFORE any git subprocess runs (REQ-001).
//
// The --version flag wins over an @version suffix when both are given and they disagree,
// which is the flag's own documented contract ("overrides version in pack reference").
// Both supplying the SAME version is not ambiguity and is not an error.
//
// The returned coordinate has ONLY the @version suffix removed: no case folding, no
// suffix stripping, no host normalization. The returned version is the normalized X.Y.Z
// with no prefix, so re-prefixing it yields exactly one "v".
func ResolveEffectiveVersion(reference, overrideVersion string) (string, string, error) {
	coordinate, refVersion := parsePackRef(reference)

	supplied := refVersion
	if overrideVersion != "" {
		supplied = overrideVersion
	}
	if supplied == "" {
		return "", "", &VersionUnresolvedError{
			Reference: reference,
			Problem:   "the reference carries no @version suffix and no --version flag was given",
		}
	}

	normalized, ok := normalizeReleaseVersion(supplied)
	if !ok {
		return "", "", &VersionUnresolvedError{
			Reference: reference,
			Problem:   fmt.Sprintf("%q is not a strict MAJOR.MINOR.PATCH release version", supplied),
		}
	}

	return coordinate, normalized, nil
}

// ValidateRemoteIdentity is the gate itself: it reads a cloned tree's pack.yml and
// decides whether the pack may be installed, without writing anything anywhere (REQ-002,
// REQ-003, REQ-007).
//
// It MUST NOT mutate packDir. Every caller invokes it on a directory it is about to copy
// and hash, so a write here would silently change the recorded content hash.
func ValidateRemoteIdentity(coordinate, effectiveVersion, packDir string) (*RemoteIdentity, error) {
	normalizedEffective, ok := normalizeReleaseVersion(effectiveVersion)
	if !ok {
		return nil, &VersionUnresolvedError{
			Reference: coordinate,
			Problem:   fmt.Sprintf("%q is not a strict MAJOR.MINOR.PATCH release version", effectiveVersion),
		}
	}
	tag := versionTagPrefix + normalizedEffective

	manifestName, manifestVersion, err := ReadManifestIdentity(packDir)
	if err != nil {
		field := "pack.yml"
		if errors.Is(err, errManifestNameMissing) {
			field = "name"
		}
		return nil, &IdentityError{Coordinate: coordinate, Tag: tag, Field: field, Problem: err.Error()}
	}

	if manifestVersion == "" {
		return nil, &IdentityError{
			Coordinate: coordinate, Tag: tag, Field: "version",
			Problem: "the manifest declares no version, so it cannot be matched against the tag it was cloned from",
		}
	}

	// DELIBERATELY NARROWER THAN pkg/pack's manifest validation. semverPattern
	// (manifest.go:460) accepts prerelease and build-metadata suffixes, so `1.0.0-rc1` is
	// a VALID manifest that `pack check` passes — but it is not installable BY TAG,
	// because no strict release tag can ever equal it. Reusing pack.validateSemver here
	// would delete this check.
	normalizedManifest, ok := normalizeReleaseVersion(manifestVersion)
	if !ok {
		return nil, &IdentityError{
			Coordinate: coordinate, Tag: tag, Field: "version",
			Problem: fmt.Sprintf("%q is not a strict MAJOR.MINOR.PATCH release version, so no release tag can equal it", manifestVersion),
		}
	}

	if normalizedManifest != normalizedEffective {
		return nil, &VersionMismatchError{
			Coordinate:      coordinate,
			Tag:             tag,
			ManifestVersion: manifestVersion,
			ExpectedVersion: normalizedEffective,
		}
	}

	// The name rule arrives from pkg/pack rather than being restated here, so there is
	// ONE authority for what a usable pack identity is (REQ-003 / CLM-037).
	if nameErr := pack.ValidatePackName(manifestName); nameErr != nil {
		return nil, &IdentityError{
			Coordinate: coordinate, Tag: tag, Field: "name",
			Problem: nameErr.Error(),
		}
	}

	return &RemoteIdentity{
		Coordinate:       coordinate,
		EffectiveVersion: normalizedEffective,
		Tag:              tag,
		ManifestName:     manifestName,
		InstallName:      manifestName,
		// BYTE-EXACT, and it must stay that way. Any case-insensitive comparison here
		// would silently reintroduce the GitHub host assumption DD-31 removed —
		// case-insensitivity is a property of ONE host, and packs may be hosted
		// anywhere. Folding case would also make a genuinely divergent pair look
		// identical, which turns REQ-006's diagnostic off without failing anything.
		Diverged: manifestName != coordinate,
	}, nil
}

// ReadManifestIdentity reads ONLY the two identity fields out of a pack directory's
// pack.yml.
//
// IT TOLERATES A MISSING VERSION ON PURPOSE. It errors for an absent, unparseable or
// NAMELESS manifest and returns "" for an absent version, because local-path packs
// legitimately declare no version — testdata/local-pack/pack.yml is one, and several
// local-path add tests expect it to install. Version strictness is decided in
// ValidateRemoteIdentity, which only the remote tag-cloning path calls. Putting the
// version check here would refuse every local-path add in the corpus.
//
// The fields are read as yaml.Node so the RAW scalar text survives: a manifest declaring
// `version: 1.0` must be reported as "1.0", not silently coerced through a float.
func ReadManifestIdentity(packDir string) (string, string, error) {
	manifestPath := filepath.Join(packDir, "pack.yml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", "", fmt.Errorf("reading pack manifest: %w", err)
	}

	var fields struct {
		Name    yaml.Node `yaml:"name"`
		Version yaml.Node `yaml:"version"`
	}
	if unmarshalErr := yaml.Unmarshal(data, &fields); unmarshalErr != nil {
		return "", "", fmt.Errorf("parsing pack manifest: %w", unmarshalErr)
	}

	name := fields.Name.Value
	if name == "" {
		return "", "", errManifestNameMissing
	}

	return name, fields.Version.Value, nil
}

// CoordinateForEntry is the ONE accessor every remote operation on an already-locked pack
// resolves its repository through (REQ-005). It returns the coordinate and, when it had
// to fall back, a warning.
//
// There is exactly one accessor because the two consumers need different things from the
// same decision: TagVersionResolver builds its own URL from a coordinate while the clone
// paths need a URL, so pack update resolves the coordinate TWICE in one invocation. Two
// independent resolutions would emit the fallback warning twice for one command, which
// teaches operators to ignore it.
//
// The fallback is a COMPATIBILITY path, not the primary one, and is never silent: every
// lock entry written before this spec carries no coordinate, and using the pack name as a
// repository is a guess that happens to be right only while name == coordinate holds.
func CoordinateForEntry(packName string, entry LockEntry) (string, string) {
	if entry.SourceCoordinate != "" {
		return entry.SourceCoordinate, ""
	}
	return packName, fmt.Sprintf(
		"pack %s has no source_coordinate recorded in backstop.lock; falling back to its pack name as the repository. Re-add or relock the pack to record where it came from.",
		packName)
}

// RemoteURLForEntry is LAYERED on CoordinateForEntry rather than duplicating its
// fallback, so the resolver and the cloner cannot disagree about which repository a pack
// came from. It passes the warning straight through — it does not produce a second one.
func RemoteURLForEntry(packName string, entry LockEntry) (string, string) {
	coordinate, warning := CoordinateForEntry(packName, entry)
	return resolveGitURL(coordinate), warning
}

// normalizeReleaseVersion strips at most ONE leading prefix and reports whether what
// remains is a strict release version.
//
// It delegates to parseVersionComponents (versionresolver.go) rather than adding a second
// normalizer — that helper is what TagVersionResolver already applies to tags, so a tag,
// a manifest version and a resolved version all mean the same thing everywhere. Its
// single TrimPrefix is exactly the one-prefix rule: "vv1.0.0" survives as "v1.0.0" and
// then fails the pattern, because a doubled prefix is not a version.
func normalizeReleaseVersion(value string) (string, bool) {
	strictSemver := regexp.MustCompile(strictSemverPattern)
	if _, ok := parseVersionComponents(value, strictSemver); !ok {
		return "", false
	}
	return strings.TrimPrefix(value, versionTagPrefix), true
}
