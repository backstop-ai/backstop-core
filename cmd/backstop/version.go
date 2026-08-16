package main

import (
	"regexp"
	"runtime/debug"
	"strings"
)

// pseudoVersionSuffix matches the pre-release segment the Go toolchain stamps
// onto a module version derived from VCS state: a 14-digit UTC timestamp and a
// 12-character commit prefix. https://go.dev/ref/mod#pseudo-versions defines
// exactly three forms, and this pattern must see ALL of them:
//
//	1. vX.0.0-yyyymmddhhmmss-abcdefabcdef       (no known base version)
//	2. vX.Y.Z-pre.0.yyyymmddhhmmss-abcdefabcdef (base is a pre-release vX.Y.Z-pre)
//	3. vX.Y.(Z+1)-0.yyyymmddhhmmss-abcdefabcdef (base is a release vX.Y.Z)
//
// The distinction is ONE CHARACTER WIDE. Form 1 puts the timestamp straight
// after a "-", but forms 2 and 3 insert a counter segment ("pre.0", "0"), so the
// character immediately preceding the timestamp is "." instead. Matching only
// "-" therefore sees form 1 alone — and THIS repo has real release tags, so
// every plain `go build` here produces FORM 3.
//
// This is the load-bearing half of the rejection predicate. Since Go 1.24 a
// PLAIN `go build` records a pseudo-version in build info rather than the
// "(devel)" sentinel, so a fallback that accepts whatever Main.Version holds
// makes every local build report something that looks like a release and is not.
var pseudoVersionSuffix = regexp.MustCompile(`[-.]\d{14}-[0-9a-f]{12}`)

// releasedVersion matches a released module version: a leading "v", a strict
// X.Y.Z core, and an optional pre-release (so real tags like v1.0.0-rc.1 are
// accepted). Build metadata is deliberately NOT permitted here — it is rejected
// separately so the reason stays legible.
var releasedVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// resolveVersion decides which version string the CLI reports.
//
// Precedence:
//  1. a non-empty injected value other than "dev" wins outright — that is the
//     goreleaser `-ldflags -X main.version` path, and it is authoritative
//  2. absent build info yields "dev"
//  3. a RELEASED module version from build info is reported — this is what makes
//     `go install ...@vX.Y.Z` show the real tag
//  4. anything else yields "dev"
//
// It is a pure function of its arguments so the whole matrix is testable without
// building a binary per case; effectiveBuildIdentity supplies the real inputs.
func resolveVersion(injected string, info *debug.BuildInfo, ok bool) string {
	if injected != "" && injected != "dev" {
		return injected
	}
	if !ok || info == nil {
		return "dev"
	}
	if isReleasedModuleVersion(info.Main.Version) {
		return info.Main.Version
	}
	return "dev"
}

// isReleasedModuleVersion reports whether v names a published release rather than
// a version the toolchain synthesized from local state.
//
// Rejected, and why each matters:
//   - "" and "(devel)" — build info exists but names no version at all
//   - anything containing "+" — build metadata; a modified tree stamps "+dirty",
//     and "v0.11.0+dirty" is a real tag plus uncommitted changes, not a release
//   - a pseudo-version pre-release segment — synthesized from VCS state by a
//     plain `go build`, never published
func isReleasedModuleVersion(v string) bool {
	if v == "" || v == "(devel)" {
		return false
	}
	if strings.Contains(v, "+") {
		return false
	}
	if pseudoVersionSuffix.MatchString(v) {
		return false
	}
	return releasedVersion.MatchString(v)
}

// commit and buildDate are the link-time build stamps, injected the same way `version`
// is. Neither is populated by any release path today (Sharp Edge 8: .goreleaser.yml is
// deliberately not edited by SPEC-068), so both are empty in every build that exists —
// they are honored IF a release path ever supplies them.
var (
	commit    = "" // nosemgrep: go.core.no-global-mutable-state — link-time build stamp; -ldflags -X can only write a package-level var, and it is never mutated at runtime
	buildDate = "" // nosemgrep: go.core.no-global-mutable-state — link-time build stamp; -ldflags -X can only write a package-level var, and it is never mutated at runtime
)

// unknownBuildField is what a build identity reports for a field no source supplies.
// It is a LITERAL rather than an empty string so a report never carries a blank where a
// commit belongs — an empty field reads as "not rendered" and a caller cannot tell it
// apart from a surface that forgot to print it.
const unknownBuildField = "unknown"

// BuildIdentity is the full identity of one binary: what it reports as its version,
// which commit produced it, and when it was built.
type BuildIdentity struct {
	Version   string
	Commit    string
	BuildDate string
}

// resolveBuildIdentity decides the whole reported identity of a binary.
//
// The VERSION IS DELEGATED to resolveVersion, never reimplemented: the anti-spoofing
// precedence and rejections are that function's, and REQ-005 adds fields AROUND them.
// A copy of the "+" check or the pseudo-version regex here would be exactly the
// duplication CLM-026 exists to prevent.
//
// Commit and date come from the vcs.revision and vcs.time build settings, with
// vcs.modified appended to the COMMIT as a dirty marker — never to the version, which
// would let a modified tree describe itself as something other than the release it was
// cut from. A non-empty injected value wins PER FIELD, independently, so injecting only
// a commit leaves the recorded date intact.
//
// It is a pure function of its arguments, mirroring resolveVersion, so the whole matrix
// is testable without building a binary per case; effectiveBuildIdentity supplies the
// real inputs.
func resolveBuildIdentity(injectedVersion, injectedCommit, injectedDate string, info *debug.BuildInfo, ok bool) BuildIdentity {
	identity := BuildIdentity{
		Version:   resolveVersion(injectedVersion, info, ok),
		Commit:    injectedCommit,
		BuildDate: injectedDate,
	}

	var revision, buildTime string
	dirty := false
	if ok && info != nil {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.time":
				buildTime = setting.Value
			case "vcs.modified":
				dirty = setting.Value == "true"
			}
		}
	}

	if identity.Commit == "" && revision != "" {
		identity.Commit = revision
		if dirty {
			identity.Commit += "-dirty"
		}
	}
	if identity.BuildDate == "" {
		identity.BuildDate = buildTime
	}

	if identity.Commit == "" {
		identity.Commit = unknownBuildField
	}
	if identity.BuildDate == "" {
		identity.BuildDate = unknownBuildField
	}
	return identity
}

// effectiveBuildIdentity is the ONE resolved identity every output surface reads, so no
// two of them can report different versions than `backstop version` does. It is the only
// debug.ReadBuildInfo caller in the package.
func effectiveBuildIdentity() BuildIdentity {
	info, ok := debug.ReadBuildInfo()
	return resolveBuildIdentity(version, commit, buildDate, info, ok)
}
