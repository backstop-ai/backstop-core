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

// TestResolveVersion_PseudoVersionFallsBackToDev rejects the shape Go 1.24+
// stamps from VCS state on a plain `go build`. Accepting it would make every
// local dev build report something that looks like a real version. (CLM-007)
func TestResolveVersion_PseudoVersionFallsBackToDev(t *testing.T) {
	const pseudo = "v0.0.0-20260727014125-1ccb2a60b2f7"
	if got := resolveVersion("dev", buildInfoWithVersion(pseudo), true); got != "dev" {
		t.Errorf("resolveVersion(dev, %q) = %q, want \"dev\" — a pseudo-version is not a release", pseudo, got)
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

// TestVersion_LdflagsInjectionReachesBuiltCLI is the only test that proves the
// SYMBOL PATH: that `-X main.version` writes the very var the CLI reads. The
// table tests above cannot prove it — a typo in the symbol name satisfies every
// one of them and still ships a binary reporting "dev".
//
// The plain-build half is the regression guard for the captured `+dirty` trap:
// since Go 1.24 an uninjected build carries a VCS-derived pseudo-version in build
// info, so "not dev" here would mean the fallback is leaking it. Slow by design
// (two real builds); it must not be skipped. (CLM-009)
func TestVersion_LdflagsInjectionReachesBuiltCLI(t *testing.T) {
	dir := t.TempDir()

	injected := runVersion(t, buildBackstop(t, dir, "backstop", "-ldflags", "-X main.version=v9.9.9"))
	if !strings.Contains(injected, "v9.9.9") {
		t.Errorf("ldflags-injected build does not report v9.9.9; -X main.version did not reach the var the CLI reads.\ngot:\n%s", injected)
	}

	plain := runVersion(t, buildBackstop(t, dir, "backstop-plain"))
	plainVersion := versionFromTextOutput(t, plain)
	if plainVersion != "dev" {
		t.Errorf("plain build reports %q, want \"dev\" — a VCS-derived pseudo-version is leaking through the fallback", plainVersion)
	}
	if strings.Contains(plainVersion, "+") || strings.Contains(plainVersion, "dirty") {
		t.Errorf("plain build reports build metadata %q", plainVersion)
	}
}
