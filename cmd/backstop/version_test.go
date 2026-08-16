package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

// buildInfoWithVersion returns a *debug.BuildInfo whose Main.Version is v, the
// only field resolveVersion reads. Built here rather than inline so each table
// row stays a single readable line.
func buildInfoWithVersion(v string) *debug.BuildInfo {
	return &debug.BuildInfo{Main: debug.Module{Version: v}}
}

// TestResolveVersion_LdflagsInjectionWins pins the goreleaser path as
// authoritative: an injected version is returned unchanged and is never
// second-guessed by build info, present or absent. (CLM-005)
func TestResolveVersion_LdflagsInjectionWins(t *testing.T) {
	cases := []struct {
		name     string
		injected string
		info     *debug.BuildInfo
		ok       bool
		want     string
	}{
		{"injection beats a released module version", "v0.11.0", buildInfoWithVersion("v0.9.0"), true, "v0.11.0"},
		{"injection stands with no build info at all", "v0.11.0", nil, false, "v0.11.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.injected, tc.info, tc.ok); got != tc.want {
				t.Errorf("resolveVersion(%q, %v, %v) = %q, want %q", tc.injected, tc.info, tc.ok, got, tc.want)
			}
		})
	}
}

// TestResolveVersion_ModuleVersionFallbackForGoInstall covers the
// `go install ...@vX.Y.Z` path: with no injection, a RELEASED module version
// recorded in build info is what the CLI reports. (CLM-006)
func TestResolveVersion_ModuleVersionFallbackForGoInstall(t *testing.T) {
	cases := []struct {
		name     string
		injected string
		modVer   string
		want     string
	}{
		{"dev sentinel yields to a released version", "dev", "v0.11.0", "v0.11.0"},
		{"pre-release tags are released versions", "dev", "v1.0.0-rc.1", "v1.0.0-rc.1"},
		{"empty injection yields to a released version", "", "v0.11.0", "v0.11.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.injected, buildInfoWithVersion(tc.modVer), true); got != tc.want {
				t.Errorf("resolveVersion(%q, Main.Version=%q, true) = %q, want %q", tc.injected, tc.modVer, got, tc.want)
			}
		})
	}
}

// documentedPseudoVersions returns one row per pseudo-version form defined at
// https://go.dev/ref/mod#pseudo-versions, plus the two shapes actually captured
// from this repo. It is shared by the regex table and the resolveVersion table
// so the two can never drift apart on which shapes count as pseudo-versions.
//
// The load-bearing distinction lives in a single character: form 1 puts the
// 14-digit timestamp straight after a "-", while forms 2 and 3 insert a counter
// segment ("pre.0", "0") so the character immediately before the timestamp is
// "." instead. A pattern that hard-codes "-" sees form 1 only.
func documentedPseudoVersions() []struct {
	name  string
	value string
} {
	return []struct {
		name  string
		value string
	}{
		{"form 1, no known base version", "v0.0.0-20191109021931-daa7c04131f5"},
		{"form 1, captured from this repo by `go version -m ./bin/backstop`", "v0.0.0-20260727014125-1ccb2a60b2f7"},
		{"form 2, pre-release base version vX.Y.Z-pre", "v1.2.3-pre.0.20191109021931-daa7c04131f5"},
		{"form 3, release base version — the reference's own worked example for base v1.2.3", "v1.2.4-0.20191109021931-daa7c04131f5"},
		{"form 3, captured from this repo in ISSUE-130's reproduction", "v0.1.3-0.20260815231324-cf17746b5df5"},
	}
}

// TestPseudoVersionSuffix_MatchesAllDocumentedGoPseudoVersionForms drives the
// regex DIRECTLY rather than through resolveVersion, so no input carries a "+"
// and the build-metadata short-circuit in isReleasedModuleVersion is
// structurally out of the picture. That is what makes this guard hold in a dirty
// working tree, where the integration test's bare "dev" check cannot.
//
// The must-match half proves the pattern sees all three forms defined at
// https://go.dev/ref/mod#pseudo-versions; the must-NOT-match half proves the
// widening does not swallow genuine release tags, which would make
// `go install ...@vX.Y.Z` report "dev" — a worse bug than the one being fixed.
// (CLM-001, CLM-002, CLM-003)
func TestPseudoVersionSuffix_MatchesAllDocumentedGoPseudoVersionForms(t *testing.T) {
	for _, tc := range documentedPseudoVersions() {
		t.Run("matches/"+tc.name, func(t *testing.T) {
			if !pseudoVersionSuffix.MatchString(tc.value) {
				t.Errorf("pseudoVersionSuffix does not match %q (%s) — this pseudo-version would be reported as a release", tc.value, tc.name)
			}
		})
	}

	released := []string{
		"v0.11.0",
		"v1.0.0-rc.1",
		"v0.1.3",
		"v2.0.0-beta.2",
		"v1.0.0-alpha",
	}
	for _, v := range released {
		t.Run("does not match/"+v, func(t *testing.T) {
			if pseudoVersionSuffix.MatchString(v) {
				t.Errorf("pseudoVersionSuffix matches released tag %q — a real release would be rejected and reported as \"dev\"", v)
			}
		})
	}
}

// TestResolveVersion_PseudoVersionFallsBackToDev rejects every shape Go stamps
// from VCS state on a plain `go build`. Accepting any of them would make a local
// dev build report something that looks like a real version.
//
// This is the end-to-end complement to
// TestPseudoVersionSuffix_MatchesAllDocumentedGoPseudoVersionForms: that test
// proves the pattern matches, this one proves the fallback actually fires.
// (CLM-007, CLM-001, CLM-003)
func TestResolveVersion_PseudoVersionFallsBackToDev(t *testing.T) {
	for _, tc := range documentedPseudoVersions() {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion("dev", buildInfoWithVersion(tc.value), true); got != "dev" {
				t.Errorf("resolveVersion(dev, %q) = %q, want \"dev\" — a pseudo-version is not a release", tc.value, got)
			}
		})
	}
}

// TestResolveVersion_DirtyBuildFallsBackToDev covers build metadata. The first
// string is REAL captured output from `go version -m ./bin/backstop` against a
// `make build` binary on 2026-07-27 — a modified tree stamps `+dirty`, and a
// naive fallback reports it verbatim. (CLM-007)
func TestResolveVersion_DirtyBuildFallsBackToDev(t *testing.T) {
	cases := []struct {
		name   string
		modVer string
	}{
		{"captured dirty pseudo-version from a local build", "v0.0.0-20260727014125-1ccb2a60b2f7+dirty"},
		{"a real tag with dirty build metadata is still not a clean release", "v0.11.0+dirty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion("dev", buildInfoWithVersion(tc.modVer), true); got != "dev" {
				t.Errorf("resolveVersion(dev, %q) = %q, want \"dev\"", tc.modVer, got)
			}
		})
	}
}

// TestResolveVersion_DevelAndAbsentBuildInfoFallBackToDev covers every way build
// info can fail to name a release: the (devel) sentinel, an empty version, a nil
// BuildInfo, and ReadBuildInfo reporting ok==false. (CLM-007)
func TestResolveVersion_DevelAndAbsentBuildInfoFallBackToDev(t *testing.T) {
	cases := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
	}{
		{"devel sentinel", buildInfoWithVersion("(devel)"), true},
		{"empty module version", buildInfoWithVersion(""), true},
		{"nil build info", nil, true},
		{"ReadBuildInfo reported not ok", buildInfoWithVersion("v0.11.0"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion("dev", tc.info, tc.ok); got != "dev" {
				t.Errorf("resolveVersion(dev, %v, %v) = %q, want \"dev\"", tc.info, tc.ok, got)
			}
		})
	}
}

// versionFromTextOutput extracts x from a "backstop version x" line.
func versionFromTextOutput(t *testing.T, out string) string {
	t.Helper()
	const prefix = "backstop version "
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	t.Fatalf("no %q line in output:\n%s", prefix, out)
	return ""
}

// TestVersionCommand_TextAndJSONAgree drives the REAL cobra command both ways and
// asserts the two rendering branches report the same resolved value. Without this
// the JSON map and the Printf line can drift apart silently. (CLM-008)
func TestVersionCommand_TextAndJSONAgree(t *testing.T) {
	textOut, err := executeCommand(NewRootCommand(), "version")
	if err != nil {
		t.Fatalf("version (text): %v", err)
	}
	textVersion := versionFromTextOutput(t, textOut)

	jsonOut, err := executeCommand(NewRootCommand(), "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var payload struct {
		Version string `json:"version"`
	}
	if unmarshalErr := json.Unmarshal([]byte(jsonOut), &payload); unmarshalErr != nil {
		t.Fatalf("parsing --json payload: %v\noutput:\n%s", unmarshalErr, jsonOut)
	}

	if payload.Version == "" {
		t.Fatal("--json payload has an empty version field")
	}
	if payload.Version != textVersion {
		t.Errorf("version disagrees between renderings: text %q, json %q", textVersion, payload.Version)
	}
}

// buildBackstop compiles ./cmd/backstop into dir with the supplied extra build
// flags and returns the binary path.
func buildBackstop(t *testing.T, dir, name string, extraArgs ...string) string {
	t.Helper()
	bin := filepath.Join(dir, name)
	args := append([]string{"build"}, extraArgs...)
	args = append(args, "-o", bin, "./cmd/backstop")
	// nosemgrep: backstop.packs.backstop-ai.backstop-self.rules.no-baked-tool-exec — binary-building test harness: proving `-ldflags -X main.version` reaches the CLI REQUIRES invoking the real Go toolchain, the same founder-ratified rationale that exempts tests/smoke in the self pack (1.1.1). This execs the compiler under test; it is not routing/dispatch inside the binary.
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %v: %v\n%s", args, err, out)
	}
	return bin
}

// runVersion executes the built binary's version subcommand.
func runVersion(t *testing.T, bin string) string {
	t.Helper()
	// nosemgrep: backstop.packs.backstop-ai.backstop-self.rules.no-baked-tool-exec — executes the just-built backstop binary by absolute path, not a baked tool name.
	out, err := exec.Command(bin, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("running %s version: %v\n%s", bin, err, out)
	}
	return string(out)
}

// recordedModuleVersion returns the Main.Version the Go toolchain stamped into
// bin, read from the third tab-separated field of the "mod" line of
// `go version -m`. This is the raw input isReleasedModuleVersion is handed at
// runtime, which is why the test reads it rather than inferring it.
func recordedModuleVersion(t *testing.T, bin string) string {
	t.Helper()
	// nosemgrep: backstop.packs.backstop-ai.backstop-self.rules.no-baked-tool-exec — binary-inspecting test harness: reading what the Go toolchain stamped into build info REQUIRES invoking the real Go toolchain, the same founder-ratified rationale that exempts tests/smoke in the self pack (1.1.1). This execs the compiler under test; it is not routing/dispatch inside the binary.
	out, err := exec.Command("go", "version", "-m", bin).CombinedOutput()
	if err != nil {
		t.Fatalf("go version -m %s: %v\n%s", bin, err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		if len(fields) >= 3 && fields[0] == "mod" {
			return fields[2]
		}
	}
	t.Fatalf("no \"mod\" line in `go version -m %s` output:\n%s", bin, out)
	return ""
}

// TestVersion_LdflagsInjectionReachesBuiltCLI is the only test that proves the
// SYMBOL PATH: that `-X main.version` writes the very var the CLI reads. The
// table tests above cannot prove it — a typo in the symbol name satisfies every
// one of them and still ships a binary reporting "dev".
//
// The plain-build half is the regression guard for the captured `+dirty` trap:
// since Go 1.24 an uninjected build carries a VCS-derived pseudo-version in build
// info, so "not dev" here would mean the fallback is leaking it. Slow by design
// (two real builds); it must not be skipped.
//
// LIMITATION, STATED SO IT IS NOT RELIED ON: the bare `plainVersion != "dev"`
// check is SHORT-CIRCUIT-DEPENDENT. Any uncommitted or untracked file makes Go
// stamp "+dirty" onto the recorded module version, and isReleasedModuleVersion
// rejects on "+" BEFORE it consults the pseudo-version pattern — so in a dirty
// tree that check reports "dev" for the right answer via the wrong reason and
// never exercises the pattern at all. It discriminates only on a pristine
// checkout. The assertions that hold in EVERY tree are the metadata-stripped
// isReleasedModuleVersion check below, which asks "would this still be rejected
// if the tree were clean?", and
// TestPseudoVersionSuffix_MatchesAllDocumentedGoPseudoVersionForms.
// (CLM-009, CLM-004)
func TestVersion_LdflagsInjectionReachesBuiltCLI(t *testing.T) {
	dir := t.TempDir()

	injected := runVersion(t, buildBackstop(t, dir, "backstop", "-ldflags", "-X main.version=v9.9.9"))
	if !strings.Contains(injected, "v9.9.9") {
		t.Errorf("ldflags-injected build does not report v9.9.9; -X main.version did not reach the var the CLI reads.\ngot:\n%s", injected)
	}

	plainBin := buildBackstop(t, dir, "backstop-plain")
	plain := runVersion(t, plainBin)
	plainVersion := versionFromTextOutput(t, plain)
	if plainVersion != "dev" {
		t.Errorf("plain build reports %q, want \"dev\" — a VCS-derived pseudo-version is leaking through the fallback", plainVersion)
	}
	if strings.Contains(plainVersion, "+") || strings.Contains(plainVersion, "dirty") {
		t.Errorf("plain build reports build metadata %q", plainVersion)
	}

	recorded := recordedModuleVersion(t, plainBin)
	stripped := recorded
	if i := strings.IndexByte(stripped, '+'); i >= 0 {
		stripped = stripped[:i]
	}
	if isReleasedModuleVersion(stripped) {
		t.Errorf("the module version Go stamped into the plain build, %q, is accepted as a release once its build metadata is stripped (%q) — this build would report a fake release on any clean checkout, and the bare \"dev\" check above cannot see it in a dirty tree", recorded, stripped)
	}
}
