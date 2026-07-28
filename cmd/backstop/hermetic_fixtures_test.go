package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// Characterization tests for the SPEC-056 REQ-010 fixture corpus and the harness
// variant that publishes it.
//
// THESE ARE GREEN ON ARRIVAL AND THAT IS THE POINT. They assert properties of DATA
// authored in phase 1, not behavior of code written after them. Their job is to RED
// later, when someone "tidies" a fixture into uselessness — because every one of these
// properties is load-bearing for a claim elsewhere in the spec, and every one of them
// fails SILENTLY if it is lost. A drift fixture whose manifest was repaired to match
// its tags still adds and installs cleanly; it just stops testing anything. A
// sample_config target that someone authored into the tree still validates; it just
// makes a rendered file indistinguishable from an authored one.
//
// The fixtures are validated as COPIES in t.TempDir() rather than in place. `pack test`
// renders the scaffold fixture's sample_config into whatever directory it is handed, so
// validating the committed tree would leave a file behind that reds the next run's
// CLM-102 assertion. Copying first is also, not coincidentally, the same move REQ-008
// makes for exactly the same reason.

// hermeticFixtureNames are the three fixtures REQ-010 adds. CLM-104 drives all of them
// through the real validator; the other tests characterize one apiece.
var hermeticFixtureNames = []string{ // nosemgrep: go.core.no-global-mutable-state — immutable test fixture roster, never reassigned
	"version-drift-pack",
	"divergent-name-pack",
	"scaffold-config-pack",
}

const (
	// driftFixtureManifestVersion is what version-drift-pack's manifest declares.
	driftFixtureManifestVersion = "0.1.3"
	// driftFixtureTags are the tags a test publishes for it. Neither equals the
	// manifest version, which is the entire fixture.
	driftFixtureEarlierTag = "v0.1.0"
	driftFixtureLatestTag  = "v0.1.1"
)

// hermeticFixtureDir resolves a fixture source directory to an absolute path. The
// harness constructors and the built binary both need absolute paths — a relative one
// resolves against whatever working directory the subprocess was given.
func hermeticFixtureDir(t *testing.T, fixture string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "hermetic-remote", fixture))
	if err != nil {
		t.Fatalf("resolving the %s fixture: %v", fixture, err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "pack.yml")); statErr != nil {
		t.Fatalf("the %s fixture has no pack.yml at %s: %v", fixture, dir, statErr)
	}
	return dir
}

// hermeticFixtureManifest parses a fixture's manifest through packval's model — the
// same model `pack check` uses, so a fixture that parses here parses there.
func hermeticFixtureManifest(t *testing.T, fixtureDir string) *packval.PackManifest {
	t.Helper()
	manifest, err := packval.ParseManifest(filepath.Join(fixtureDir, "pack.yml"))
	if err != nil {
		t.Fatalf("parsing %s/pack.yml: %v", fixtureDir, err)
	}
	return manifest
}

// manifestVersionAtTag reads pack.yml OUT OF THE REPOSITORY at the given tag, rather
// than off the working tree. The working tree is whatever the last commit left there;
// only the tagged blob answers "what would a clone of this tag receive".
func manifestVersionAtTag(t *testing.T, repoPath, tag string) string {
	t.Helper()
	blob := mustGit(t, repoPath, "show", tag+":pack.yml")
	version := manifestVersionFromYAML(blob)
	if version == "" {
		t.Fatalf("no version line in pack.yml at tag %s of %s:\n%s", tag, repoPath, blob)
	}
	return version
}

// manifestVersionFromYAML pulls the manifest's top-level version out of raw YAML text.
// It reads the TEXT rather than unmarshalling because the assertion is about the bytes
// a clone would receive; a model that normalized or defaulted the field would hide
// exactly the drift these tests exist to observe.
func manifestVersionFromYAML(text string) string {
	for _, line := range strings.Split(text, "\n") {
		rest, found := strings.CutPrefix(line, "version:")
		if !found {
			continue
		}
		return strings.Trim(strings.TrimSpace(rest), `"'`)
	}
	return ""
}

// TestHermeticRemote_TagsWithoutRewritingManifestVersion proves the harness can publish
// tags while leaving the manifest's declared version alone (CLM-099).
//
// newHermeticRemote rewrites pack.yml's version to match every tag it creates. Built
// through it, a drift fixture is REPAIRED before any test can observe the drift, and
// CLM-016 passes over a fixture that cannot fail. The contrast arm below is what makes
// this test able to detect that: without it, a constructor that silently rewrote anyway
// would still satisfy the first half.
func TestHermeticRemote_TagsWithoutRewritingManifestVersion(t *testing.T) {
	source := hermeticFixtureDir(t, "version-drift-pack")

	kept := newHermeticRemoteKeepingManifestVersion(t, source, driftFixtureEarlierTag, driftFixtureLatestTag)
	for _, tag := range []string{driftFixtureEarlierTag, driftFixtureLatestTag} {
		if got := manifestVersionAtTag(t, kept.Path, tag); got != driftFixtureManifestVersion {
			t.Errorf("newHermeticRemoteKeepingManifestVersion rewrote the manifest at %s: version = %q, want the fixture's own %q — a rewritten manifest repairs the drift and CLM-016 stops testing anything",
				tag, got, driftFixtureManifestVersion)
		}
	}

	// THE CONTRAST ARM. The same source through the DEFAULT constructor must be
	// rewritten to match its tag. This is what distinguishes "the constructor honored
	// the fixture" from "nothing rewrites anything anymore".
	rewritten := newHermeticRemote(t, source, driftFixtureEarlierTag, driftFixtureLatestTag)
	wantRewritten := strings.TrimPrefix(driftFixtureLatestTag, "v")
	if got := manifestVersionAtTag(t, rewritten.Path, driftFixtureLatestTag); got != wantRewritten {
		t.Errorf("newHermeticRemote did not rewrite the manifest at %s: version = %q, want %q — if the default constructor no longer rewrites, the keeping-version constructor is not the thing under test and this suite proves nothing",
			driftFixtureLatestTag, got, wantRewritten)
	}
}

// TestHermeticFixture_VersionDriftPackHasDriftingManifest pins all three values —
// both published tags and the manifest version — rather than their inequality (CLM-100).
//
// Asserting only that they differ would let a fixture edited to 9.9.9 pass, and 9.9.9
// is not the shape REQ-002's first customer has. The published harness toolchain pack
// declares 0.1.3 against tags that stop at v0.1.1 (DIR-027 item 2); that specific
// relationship — a manifest AHEAD of every tag — is what the identity gate must refuse.
func TestHermeticFixture_VersionDriftPackHasDriftingManifest(t *testing.T) {
	source := hermeticFixtureDir(t, "version-drift-pack")

	manifest := hermeticFixtureManifest(t, source)
	if manifest.Version != driftFixtureManifestVersion {
		t.Errorf("version-drift-pack manifest version = %q, want %q", manifest.Version, driftFixtureManifestVersion)
	}

	remote := newHermeticRemoteKeepingManifestVersion(t, source, driftFixtureEarlierTag, driftFixtureLatestTag)
	wantTags := []string{driftFixtureEarlierTag, driftFixtureLatestTag}
	if len(remote.Tags) != len(wantTags) {
		t.Fatalf("published tags = %v, want %v", remote.Tags, wantTags)
	}
	for i, want := range wantTags {
		if remote.Tags[i] != want {
			t.Errorf("published tag %d = %q, want %q", i, remote.Tags[i], want)
		}
		if got := manifestVersionAtTag(t, remote.Path, want); got != driftFixtureManifestVersion {
			t.Errorf("manifest at %s = %q, want %q", want, got, driftFixtureManifestVersion)
		}
	}

	// The drift itself, stated directly: the manifest matches NEITHER tag.
	for _, tag := range wantTags {
		if strings.TrimPrefix(tag, "v") == driftFixtureManifestVersion {
			t.Errorf("tag %s equals the manifest version %q — the fixture no longer drifts and every claim that consumes it is vacuous", tag, driftFixtureManifestVersion)
		}
	}
}

// TestHermeticFixture_DivergentNamePackNameDiffersFromRepository proves the manifest
// name is not derivable from the repository directory (CLM-101).
//
// remoteE2ESetup builds the requested coordinate as `hermetic/<directory>`, so the
// directory name IS the coordinate a test asks for. The manifest declaring something
// else is what makes REQ-003's install-path/lock-key/asset-root claims falsifiable.
//
// The difference must not be case-only. CLM-066 owns the case-only shape and gets its
// divergence from the caller varying the ref's case; a case-only fixture here would
// silently turn every general divergence claim into a duplicate of that one.
func TestHermeticFixture_DivergentNamePackNameDiffersFromRepository(t *testing.T) {
	const fixture = "divergent-name-pack"
	source := hermeticFixtureDir(t, fixture)

	manifest := hermeticFixtureManifest(t, source)
	coordinate := remoteE2EOrg + "/" + fixture

	if manifest.Name == coordinate {
		t.Fatalf("manifest name %q equals the coordinate remoteE2ESetup builds for this fixture — the fixture has stopped diverging and CLM-028..031 test nothing", manifest.Name)
	}
	if strings.EqualFold(manifest.Name, coordinate) {
		t.Errorf("manifest name %q differs from coordinate %q only in letter case; the divergence must be byte-level, because CLM-066 owns the case-only shape and a case-only fixture makes the general divergence claims duplicates of it",
			manifest.Name, coordinate)
	}

	// The pack part alone must differ too, not merely the org: an identical pack part
	// under a different org would still resolve assets to the same leaf directory.
	_, manifestLeaf, ok := strings.Cut(manifest.Name, "/")
	if !ok {
		t.Fatalf("manifest name %q carries no slash; it cannot be a pack identity", manifest.Name)
	}
	if manifestLeaf == fixture {
		t.Errorf("manifest name's pack part %q equals the fixture directory %q — the install path would land in the same place either way", manifestLeaf, fixture)
	}
}

// TestHermeticFixture_ScaffoldConfigPackDeclaresUnauthoredSampleConfig proves every
// declared sample_config target is ABSENT from the authored tree (CLM-102).
//
// This is what makes a rendered file DETECTABLE. packval phase 3 writes each entry to
// <packDir>/<scaffold.path>/<relPath>; if a file is already authored there, a rendered
// one is indistinguishable from it, and CLM-081/082's "validator writes do not reach
// the installed tree" assertions cannot fail no matter how broken the seam is.
//
// If this test reds after a local `pack test` run against the committed fixture, the
// fix is to DELETE the rendered file, never to commit it.
func TestHermeticFixture_ScaffoldConfigPackDeclaresUnauthoredSampleConfig(t *testing.T) {
	source := hermeticFixtureDir(t, "scaffold-config-pack")
	manifest := hermeticFixtureManifest(t, source)

	checked := 0
	for _, scaffold := range manifest.Content.Scaffolds {
		if scaffold.Tier != "complete" {
			continue
		}
		if len(scaffold.SampleConfig) == 0 {
			t.Errorf("scaffold %q is tier complete but declares no sample_config; it provokes no mutation and REQ-008 has nothing to contain", scaffold.ID)
			continue
		}
		for relPath := range scaffold.SampleConfig {
			target := filepath.Join(source, scaffold.Path, relPath)
			if _, err := os.Stat(target); err == nil {
				t.Errorf("sample_config target %s EXISTS in the authored tree; a file rendered there by packval would be indistinguishable from an authored one, which disarms CLM-081/082", target)
			} else if !os.IsNotExist(err) {
				t.Errorf("stat %s: %v", target, err)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no tier:complete scaffold with sample_config found; this fixture is the corpus's only one and REQ-008's whole observable contract rests on it")
	}
}

// TestHermeticFixture_ScaffoldTestCommandIsAShellBuiltin proves the declared command
// resolves to NO executable on PATH (CLM-103).
//
// WHY THE ASSERTION IS SHAPED THIS WAY. A shell subprocess is UNAVOIDABLE on this path:
// DefaultExecutor.RunScaffoldTest runs exec.Command("sh", "-c", testCommand)
// unconditionally for a tier:complete scaffold (executor.go:107-112, reached from
// phase3.go:173) and PackvalValidator supplies no executor. So the hermetic property
// this fixture can honestly claim is NETWORK-FREE AND TOOLCHAIN-FREE, not process-free
// — writing it the other way would be false by construction. "Resolves to nothing on
// PATH" is the mechanical statement of "reaches nothing outside the process".
//
// THIS IS WHY THE COMMAND IS ":" AND NOT "true". Both are shell builtins, but `true` is
// ALSO a real binary — /usr/bin/true exists on macOS and on every coreutils system — so
// exec.LookPath("true") SUCCEEDS and a fixture declaring it would fail this assertion.
// ":" is a POSIX SPECIAL built-in: a conforming shell resolves it before any PATH
// search and it has no external counterpart to fall through to.
func TestHermeticFixture_ScaffoldTestCommandIsAShellBuiltin(t *testing.T) {
	source := hermeticFixtureDir(t, "scaffold-config-pack")
	manifest := hermeticFixtureManifest(t, source)

	checked := 0
	for _, scaffold := range manifest.Content.Scaffolds {
		if scaffold.Tier != "complete" {
			continue
		}
		if strings.TrimSpace(scaffold.TestCommand) == "" {
			t.Errorf("scaffold %q declares no test_command; validateScaffold requires one and phase 3 would fail for a structural reason", scaffold.ID)
			continue
		}
		word := strings.Fields(scaffold.TestCommand)[0]
		if path, err := exec.LookPath(word); err == nil {
			t.Errorf("scaffold %q test_command %q resolves to the executable %s on PATH; the fixture must run a command PATH cannot supply, or the sh subprocess reaches a real binary and the fixture stops being toolchain-free",
				scaffold.ID, scaffold.TestCommand, path)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no tier:complete scaffold found; CLM-103 has nothing to characterize")
	}
}

// TestHermeticFixture_NewFixturesPassPackCheckAndPackTest drives the BUILT binary
// against every new fixture and requires exit 0 from the PROCESS exit code (CLM-104).
//
// A hand-written fixture that merely satisfies whatever the validator happened to
// accept is a mirror, not a fixture. This test is what makes it a fixture — and it
// matters most for scaffold-config-pack, the corpus's first tier:complete scaffold
// declaration and therefore the one most likely to be structurally wrong in a way no
// existing test would notice.
//
// Each fixture is validated as a COPY in t.TempDir(). `pack test` renders the scaffold
// fixture's sample_config INTO the directory it validates, so running it against the
// committed tree would leave a file behind that reds CLM-102 on the next run.
func TestHermeticFixture_NewFixturesPassPackCheckAndPackTest(t *testing.T) {
	bin := buildBackstopBinary(t)

	for _, fixture := range hermeticFixtureNames {
		t.Run(fixture, func(t *testing.T) {
			source := hermeticFixtureDir(t, fixture)
			workDir := t.TempDir()
			copyDir := filepath.Join(workDir, fixture)
			if err := os.MkdirAll(copyDir, 0o755); err != nil {
				t.Fatalf("creating the fixture copy dir: %v", err)
			}
			copyTree(t, source, copyDir)

			for _, sub := range []string{"check", "test"} {
				stdout, stderr, code := runBackstopStreams(t, bin, workDir, "pack", sub, copyDir)
				if code != 0 {
					t.Errorf("pack %s %s exited %d, want 0\nstdout: %s\nstderr: %s", sub, fixture, code, stdout, stderr)
				}
			}
		})
	}
}
