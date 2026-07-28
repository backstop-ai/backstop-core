package main

import (
	"regexp"
	"runtime/debug"
	"strings"
)

// pseudoVersionSuffix matches the pre-release segment the Go toolchain stamps
// onto a module version derived from VCS state: a 14-digit UTC timestamp and a
// 12-character commit prefix, e.g. "-20260727014125-1ccb2a60b2f7".
//
// This is the load-bearing half of the rejection predicate. Since Go 1.24 a
// PLAIN `go build` records a pseudo-version in build info rather than the
// "(devel)" sentinel, so a fallback that accepts whatever Main.Version holds
// makes every local build report something that looks like a release and is not.
var pseudoVersionSuffix = regexp.MustCompile(`-\d{14}-[0-9a-f]{12}`)

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
// building a binary per case; effectiveVersion supplies the real inputs.
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

// effectiveVersion is what the CLI reports: the link-time stamp when goreleaser
// injected one, otherwise a released module version recorded by `go install`,
// otherwise "dev".
func effectiveVersion() string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(version, info, ok)
}
