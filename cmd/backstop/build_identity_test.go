package main

import (
	"runtime/debug"
	"strings"
	"testing"
)

// buildInfoWith constructs a debug.BuildInfo carrying a main module version and the
// VCS settings the Go toolchain stamps. Settings are built here rather than read from
// the running test binary so every row of the matrix is reachable without building a
// binary per case.
func buildInfoWith(mainVersion string, settings map[string]string) *debug.BuildInfo {
	info := &debug.BuildInfo{}
	info.Main.Version = mainVersion
	for k, v := range settings {
		info.Settings = append(info.Settings, debug.BuildSetting{Key: k, Value: v})
	}
	return info
}

// TestResolveVersion_NonReleaseBuildStillReportsDev pins CLM-026: REQ-005 adds fields
// AROUND resolveVersion and must not alter its precedence or its rejections.
//
// Every row is driven through resolveBuildIdentity, NOT through resolveVersion
// directly, because the claim is about what the new wrapper reports. A wrapper that
// reimplemented the predicate instead of delegating would pass a resolveVersion test
// and fail here.
func TestResolveVersion_NonReleaseBuildStillReportsDev(t *testing.T) {
	cases := []struct {
		name        string
		injectedVer string
		mainVersion string
		infoOK      bool
		want        string
	}{
		{"devel sentinel", "", "(devel)", true, "dev"},
		{"dirty build metadata", "", "v0.11.0+dirty", true, "dev"},
		{"pseudo-version", "", "v0.0.0-20260727014125-1ccb2a60b2f7", true, "dev"},
		{"absent build info", "", "", false, "dev"},
		{"literal dev injection is not a release", "dev", "(devel)", true, "dev"},
		{"released tag from build info", "", "v0.1.0", true, "v0.1.0"},
		{"injected release beats build info", "v9.9.9", "v0.1.0", true, "v9.9.9"},
		{"injected release beats a devel tree", "v9.9.9", "(devel)", true, "v9.9.9"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var info *debug.BuildInfo
			if tc.infoOK {
				info = buildInfoWith(tc.mainVersion, nil)
			}
			got := resolveBuildIdentity(tc.injectedVer, "", "", info, tc.infoOK)
			if got.Version != tc.want {
				t.Errorf("resolveBuildIdentity version = %q, want %q", got.Version, tc.want)
			}
		})
	}

	// The precedence order itself, asserted as an order rather than as three
	// independent rows: an injected release outranks a released tag, which outranks
	// the dev fallback.
	injected := resolveBuildIdentity("v9.9.9", "", "", buildInfoWith("v0.1.0", nil), true)
	tagged := resolveBuildIdentity("", "", "", buildInfoWith("v0.1.0", nil), true)
	fallback := resolveBuildIdentity("", "", "", buildInfoWith("(devel)", nil), true)
	if injected.Version == tagged.Version || tagged.Version == fallback.Version {
		t.Errorf("precedence collapsed: injected=%q tagged=%q fallback=%q",
			injected.Version, tagged.Version, fallback.Version)
	}
}

// TestBuildIdentity_DirtyTreeMarkedOnCommitNotVersion pins CLM-027. A modified tree is
// reported on the COMMIT. Marking the VERSION is the spoofing failure this claim
// forbids, so both halves are asserted: the marker is present on one field and the
// other is byte-identical to what a clean tree reports.
func TestBuildIdentity_DirtyTreeMarkedOnCommitNotVersion(t *testing.T) {
	const rev = "1ccb2a60b2f70000000000000000000000000000"

	clean := resolveBuildIdentity("", "", "", buildInfoWith("v0.1.0", map[string]string{
		"vcs.revision": rev,
		"vcs.modified": "false",
	}), true)
	dirty := resolveBuildIdentity("", "", "", buildInfoWith("v0.1.0", map[string]string{
		"vcs.revision": rev,
		"vcs.modified": "true",
	}), true)

	if dirty.Commit == clean.Commit {
		t.Errorf("a modified tree reported the same commit as a clean one (%q); the dirty marker is missing", dirty.Commit)
	}
	if !strings.HasPrefix(dirty.Commit, rev) {
		t.Errorf("dirty commit %q does not start with the recorded revision %q", dirty.Commit, rev)
	}
	if dirty.Version != clean.Version {
		t.Errorf("a modified tree changed the reported VERSION: dirty=%q clean=%q; the dirty marker belongs on the commit", dirty.Version, clean.Version)
	}
	if clean.Commit != rev {
		t.Errorf("a clean tree reported commit %q, want the bare revision %q", clean.Commit, rev)
	}
}

// TestBuildIdentity_InjectedStampTakesPrecedence pins CLM-028. "When non-empty" is the
// whole meaning of the rule, and the two fields are independent: an injected commit
// with no injected date must not blank the date. A naive per-struct assignment gets
// exactly that wrong, so it is asserted directly.
func TestBuildIdentity_InjectedStampTakesPrecedence(t *testing.T) {
	info := buildInfoWith("v0.1.0", map[string]string{
		"vcs.revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"vcs.time":     "2026-01-01T00:00:00Z",
	})

	both := resolveBuildIdentity("", "deadbeef", "2026-08-14T00:00:00Z", info, true)
	if both.Commit != "deadbeef" {
		t.Errorf("injected commit did not win: got %q", both.Commit)
	}
	if both.BuildDate != "2026-08-14T00:00:00Z" {
		t.Errorf("injected build date did not win: got %q", both.BuildDate)
	}

	// An EMPTY injected value must not win — the recorded build info still shows.
	neither := resolveBuildIdentity("", "", "", info, true)
	if neither.Commit != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("an empty injected commit displaced the recorded revision: got %q", neither.Commit)
	}
	if neither.BuildDate != "2026-01-01T00:00:00Z" {
		t.Errorf("an empty injected date displaced the recorded vcs.time: got %q", neither.BuildDate)
	}

	// Independence: injecting ONLY the commit must leave the recorded date intact.
	commitOnly := resolveBuildIdentity("", "deadbeef", "", info, true)
	if commitOnly.Commit != "deadbeef" {
		t.Errorf("injected commit did not win when the date was left empty: got %q", commitOnly.Commit)
	}
	if commitOnly.BuildDate != "2026-01-01T00:00:00Z" {
		t.Errorf("injecting only the commit blanked the build date: got %q", commitOnly.BuildDate)
	}

	// And the mirror image: injecting ONLY the date must leave the recorded commit.
	dateOnly := resolveBuildIdentity("", "", "2026-08-14T00:00:00Z", info, true)
	if dateOnly.BuildDate != "2026-08-14T00:00:00Z" {
		t.Errorf("injected date did not win when the commit was left empty: got %q", dateOnly.BuildDate)
	}
	if dateOnly.Commit != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("injecting only the date blanked the commit: got %q", dateOnly.Commit)
	}
}

// TestBuildIdentity_UnknownWhenNoInjectedStampAndNoVCSData pins CLM-029, which is
// Sharp Edge 8's module-cache install: `go install` from the module cache records no
// VCS settings at all. The commit is the LITERAL "unknown" — not "", not omitted — so
// the reported identity never looks like a commit nobody can resolve.
func TestBuildIdentity_UnknownWhenNoInjectedStampAndNoVCSData(t *testing.T) {
	noVCS := resolveBuildIdentity("", "", "", buildInfoWith("v0.1.0", nil), true)
	if noVCS.Commit != "unknown" {
		t.Errorf("commit with no injected stamp and no VCS settings = %q, want the literal %q", noVCS.Commit, "unknown")
	}
	if noVCS.BuildDate != "unknown" {
		t.Errorf("build date with no injected stamp and no VCS settings = %q, want the literal %q", noVCS.BuildDate, "unknown")
	}

	// Absent build info entirely is the same story, and it must not panic.
	noInfo := resolveBuildIdentity("", "", "", nil, false)
	if noInfo.Commit != "unknown" || noInfo.BuildDate != "unknown" {
		t.Errorf("absent build info = commit %q date %q, want both %q", noInfo.Commit, noInfo.BuildDate, "unknown")
	}
	if noInfo.Version != "dev" {
		t.Errorf("absent build info version = %q, want %q", noInfo.Version, "dev")
	}
}
