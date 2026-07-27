package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// The SPEC-056 end-to-end suite: real git, the real binary, hermetic throughout, and
// stdout/stderr captured INDEPENDENTLY.
//
// Every test here drives runBackstopStreams, and NOTHING ELSE. The two merged-output
// helpers this package also offers are both disqualified: one uses CombinedOutput, and the
// cobra helper in root_test.go:17-23 points SetOut and SetErr at the SAME buffer. Against
// either, "renders to stderr" passes for a command that printed to stdout, and every
// stream claim in this file would be vacuous.
//
// The mechanical check is that neither helper is NAMED in this file — including here,
// which is why they are described rather than spelled.
//
// HERMETICITY IS LOAD-BEARING AND FRAGILE. The mechanism is a
// url.<file://repo>.insteadOf rewrite installed through GIT_CONFIG_GLOBAL, and it survives
// the process boundary only because runBackstopStreams passes os.Environ() through. Do not
// hand any child here a filtered environment: a scrubbed env sends the clone to GitHub and
// the suite goes silently network-dependent with nothing turning red.
//
// For a DIVERGENT pack the redirect is keyed on the COORDINATE the ref names
// (hermetic/divergent-name-pack), never on the manifest name. A redirect registered
// against the manifest name simply MISSES, and the clone reaches the network.

// driftE2ESetup publishes a fixture WITHOUT rewriting its manifest version, which is the
// only way a tag-versus-manifest divergence can exist at all.
//
// remoteE2ESetup cannot be used for this: it builds through newHermeticRemote, which
// rewrites pack.yml's version to match each tag — repairing the drift before the test runs
// and making CLM-022 pass over a fixture that cannot fail (spec Review Question 7).
func driftE2ESetup(t *testing.T, fixture string, tags ...string) (packName, projectDir string) {
	t.Helper()

	source, err := filepath.Abs(filepath.Join("testdata", "hermetic-remote", fixture))
	if err != nil {
		t.Fatalf("resolving the %s fixture: %v", fixture, err)
	}

	remote := newHermeticRemoteKeepingManifestVersion(t, source, tags...)
	redirectPackURL(t, remoteE2EOrg, fixture, remote.Path)
	assertPackURLRedirected(t, remoteE2EOrg, fixture, remote)

	return remoteE2EOrg + "/" + fixture, newConsumerProject(t)
}

// TestE2E_PackAdd_VersionlessReferenceRefusesWithGuidance (CLM-011).
//
// Before REQ-001, parsePackRef returned an empty version for a bare org/name and the
// pipeline cloned the ref "v" — so the operator's diagnostic was git complaining about a
// nonexistent branch. That specific regression is what the last assertion guards.
func TestE2E_PackAdd_VersionlessReferenceRefusesWithGuidance(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := remoteE2ESetup(t, "valid-pack", "v1.0.0")

	stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName)

	if code == 0 {
		t.Fatalf("a bare org/name with no version must exit non-zero\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "--version") {
		t.Errorf("the diagnostic must name the --version flag; stderr was:\n%s", stderr)
	}
	if !strings.Contains(stderr, "@") {
		t.Errorf("the diagnostic must name the @version suffix; stderr was:\n%s", stderr)
	}
	// NOT a git error about a branch — that is the pre-REQ-001 symptom.
	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "remote branch") || strings.Contains(lower, "not found in upstream") {
		t.Errorf("the operator got a GIT error about a nonexistent branch instead of a version diagnostic; the refusal must happen before git runs.\nstderr: %s", stderr)
	}
}

// TestE2E_PackAdd_ManifestVersionDriftRefusesLoudly (CLM-022) — the published harness
// pack's exact shape: manifest 0.1.3, tags stopping at v0.1.1.
func TestE2E_PackAdd_ManifestVersionDriftRefusesLoudly(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := driftE2ESetup(t, "version-drift-pack", "v0.1.0", "v0.1.1")

	stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@0.1.1")

	if code == 0 {
		t.Fatalf("a manifest version disagreeing with the requested tag must exit non-zero\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "0.1.3") {
		t.Errorf("the diagnostic must name the manifest's declared 0.1.3; stderr was:\n%s", stderr)
	}
	if !strings.Contains(stderr, "0.1.1") {
		t.Errorf("the diagnostic must name the requested 0.1.1; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "added") || strings.Contains(strings.ToLower(stdout), "installed") {
		t.Errorf("a refused add must print no success line on stdout; stdout was:\n%s", stdout)
	}
}

// TestE2E_PackAdd_DivergentIdentityWarnsOnStderrAndSucceeds (CLM-070) — all three
// properties on separated streams.
func TestE2E_PackAdd_DivergentIdentityWarnsOnStderrAndSucceeds(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := remoteE2ESetup(t, "divergent-name-pack", "v1.0.0")

	stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@1.0.0")

	if code != 0 {
		t.Fatalf("divergence must never change the exit code of an otherwise-successful add; exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "hermetic/renamed-pack") {
		t.Errorf("the divergence diagnostic must reach STDERR naming the manifest name; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Errorf("the diagnostic must not appear on stdout; stdout was:\n%s", stdout)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Errorf("the success line must still reach stdout; stdout was empty")
	}
}

// TestE2E_DivergentNamePack_AddThenInstallRoundTrip (CLM-062) is the claim that proves
// REQ-005 makes REQ-003 survivable.
//
// The fresh consumer carries ONLY backstop.yml and backstop.lock. Without the recorded
// source_coordinate it would try to clone `hermetic/renamed-pack` — the manifest name —
// and find nothing there.
func TestE2E_DivergentNamePack_AddThenInstallRoundTrip(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := remoteE2ESetup(t, "divergent-name-pack", "v1.0.0")

	if stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("the seeding add failed: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	const manifestName = "hermetic/renamed-pack"
	added := remoteE2ELockEntry(t, projectDir, manifestName)
	if added.SourceCoordinate != packName {
		t.Fatalf("source_coordinate = %q, want the requested %q — without it the fresh install cannot find the repository", added.SourceCoordinate, packName)
	}

	fresh := remoteE2EFreshConsumer(t, projectDir)
	stdout, stderr, code := runBackstopStreams(t, bin, fresh, "pack", "install")
	if code != 0 {
		t.Fatalf("installing a divergent pack from its lock must succeed: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	installed := remoteE2ELockEntry(t, fresh, manifestName)
	if installed.ContentHash != added.ContentHash {
		t.Errorf("content hash differs: add=%q install=%q — the restored bytes are not the bytes that were locked", added.ContentHash, installed.ContentHash)
	}
	installedTree := filepath.Join(fresh, ".backstop", "packs", filepath.FromSlash(manifestName))
	if _, err := distribution.ComputeContentHash(installedTree); err != nil {
		t.Errorf("the pack must be materialized under the MANIFEST name at %s: %v", installedTree, err)
	}
}

// TestE2E_ScaffoldConfigPack_AddThenInstallHashesMatch (CLM-090) is THE claim REQ-008
// exists for, and it must drive a REAL pack through REAL packval.
//
// scaffold-config-pack declares the corpus's only tier:complete scaffold with a
// sample_config. Before the scratch copy, packval's phase-3 rendering left files present
// at add-hash time and absent at install-hash time, and the install failed with a mismatch
// that looks exactly like tampering. A mock-validator proof cannot reach packval at all,
// which is why this one is end to end.
func TestE2E_ScaffoldConfigPack_AddThenInstallHashesMatch(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := remoteE2ESetup(t, "scaffold-config-pack", "v1.0.0")

	if stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("adding the scaffold pack failed: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	const manifestName = "hermetic/scaffold-config-pack"
	added := remoteE2ELockEntry(t, projectDir, manifestName)

	fresh := remoteE2EFreshConsumer(t, projectDir)
	stdout, stderr, code := runBackstopStreams(t, bin, fresh, "pack", "install")
	if code != 0 {
		t.Fatalf("installing a tier:complete scaffold pack from its lock must succeed — a hash mismatch here is validator contamination, not tampering: exit %d\nstdout: %s\nstderr: %s",
			code, stdout, stderr)
	}

	restored := remoteE2ELockEntry(t, fresh, manifestName)
	if restored.ContentHash != added.ContentHash {
		t.Errorf("the hash pack add recorded (%q) differs from what a fresh clone restores (%q); validation contaminated the tree that was hashed",
			added.ContentHash, restored.ContentHash)
	}
}

// TestE2E_IdentityRefusalNeverExitsSilently (CLM-097).
//
// An exit-1 with empty stderr satisfies `if code != 1` and is a defect class this repo has
// shipped before, so every refusal SHAPE is checked for a non-empty diagnostic.
func TestE2E_IdentityRefusalNeverExitsSilently(t *testing.T) {
	bin := buildBackstopBinary(t)

	t.Run("unresolvable version", func(t *testing.T) {
		packName, projectDir := remoteE2ESetup(t, "valid-pack", "v1.0.0")
		stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName)
		assertLoudRefusal(t, stdout, stderr, code)
	})

	t.Run("version mismatch", func(t *testing.T) {
		packName, projectDir := driftE2ESetup(t, "version-drift-pack", "v0.1.0", "v0.1.1")
		stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@0.1.1")
		assertLoudRefusal(t, stdout, stderr, code)
	})

	t.Run("absent tag", func(t *testing.T) {
		packName, projectDir := remoteE2ESetup(t, "valid-pack", "v1.0.0")
		stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@9.9.9")
		assertLoudRefusal(t, stdout, stderr, code)
	})
}

func assertLoudRefusal(t *testing.T, stdout, stderr string, code int) {
	t.Helper()
	if code == 0 {
		t.Fatalf("expected a non-zero exit\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Errorf("exited %d with an EMPTY stderr; a silent non-zero exit tells the operator nothing.\nstdout: %s", code, stdout)
	}
}

// TestE2E_IdentityRefusalEmitsNoStackTrace (CLM-098). pack add's options once carried a
// nil GitCloner that nil-dereferenced (ISSUE-073); a panic is not a diagnostic.
func TestE2E_IdentityRefusalEmitsNoStackTrace(t *testing.T) {
	bin := buildBackstopBinary(t)

	for _, tc := range []struct {
		name   string
		ref    func(t *testing.T) (string, string)
		suffix string
	}{
		{"unresolvable version", func(t *testing.T) (string, string) { return remoteE2ESetup(t, "valid-pack", "v1.0.0") }, ""},
		{"version mismatch", func(t *testing.T) (string, string) {
			return driftE2ESetup(t, "version-drift-pack", "v0.1.0", "v0.1.1")
		}, "@0.1.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			packName, projectDir := tc.ref(t)
			stdout, stderr, _ := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+tc.suffix)

			for stream, body := range map[string]string{"stdout": stdout, "stderr": stderr} {
				if strings.Contains(body, "goroutine ") {
					t.Errorf("%s carries a goroutine dump; a panic is not a diagnostic:\n%s", stream, body)
				}
				if strings.Contains(body, "panic:") {
					t.Errorf("%s carries a panic:\n%s", stream, body)
				}
			}
		})
	}
}

// TestE2E_WarningDoesNotChangeExitCode (CLM-113) reads the PROCESS exit code, both for a
// divergent add and for a fallback install.
func TestE2E_WarningDoesNotChangeExitCode(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := remoteE2ESetup(t, "divergent-name-pack", "v1.0.0")

	stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@1.0.0")
	if code != 0 {
		t.Fatalf("a divergent add emitted a warning AND changed the exit code (%d)\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatal("this test is only meaningful if a warning was actually emitted; stderr was empty")
	}

	// THE FALLBACK HALF USES A CONVERGENT PACK, AND THAT IS NOT A CONVENIENCE.
	//
	// The coordinate fallback resolves to the PACK NAME. For a DIVERGENT pack that name is
	// not a repository at all, so a stripped-coordinate install is unresolvable BY
	// CONSTRUCTION — driving it here sent the clone to github.com, which is both a wrong
	// assertion and a hermeticity leak. That is precisely REQ-005's point: the fallback is
	// a compatibility path that works only while the name == coordinate fleet convention
	// holds, which is why REQ-004 records the coordinate at all.
	convergent, convergentProject := remoteE2ESetup(t, "valid-pack", "v1.0.0")
	if out, errOut, addCode := runBackstopStreams(t, bin, convergentProject, "pack", "add", convergent+"@1.0.0"); addCode != 0 {
		t.Fatalf("seeding the convergent add failed: exit %d\nstdout: %s\nstderr: %s", addCode, out, errOut)
	}
	stripRecordedCoordinate(t, convergentProject, convergent)
	fresh := remoteE2EFreshConsumer(t, convergentProject)

	stdout, stderr, code = runBackstopStreams(t, bin, fresh, "pack", "install")
	if code != 0 {
		t.Errorf("a fallback install emitted a warning AND changed the exit code (%d)\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "warning") {
		t.Errorf("this half is only meaningful if the fallback actually warned; stderr was:\n%s", stderr)
	}
}

// TestE2E_WarningsAndOutputOccupySeparateStreams (CLM-114) asserts BOTH directions: a
// renderer writing to both streams satisfies either half alone.
func TestE2E_WarningsAndOutputOccupySeparateStreams(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := remoteE2ESetup(t, "divergent-name-pack", "v1.0.0")

	stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@1.0.0")
	if code != 0 {
		t.Fatalf("the add must succeed: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	if !strings.Contains(strings.ToLower(stderr), "warning") {
		t.Errorf("the warning must appear on STDERR; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Errorf("the warning must NOT appear on stdout; stdout was:\n%s", stdout)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Errorf("command output must still occupy stdout; it was empty")
	}
	if strings.Contains(stderr, "hermetic/renamed-pack at") && strings.Contains(stdout, "hermetic/renamed-pack at") {
		t.Error("the same line reached BOTH streams")
	}
}

// TestE2E_GateResolvesPackAssetsUnderManifestName (CLM-031) is the claim that ties this
// whole spec back to the defect that motivated it.
//
// The harness consumer's failure was `missing convert script`: pack add reported success
// while installing under the requested COORDINATE, and the gate then resolved the pack's
// rules, producers and converters under its MANIFEST name — looking for assets in a
// directory nothing had written. After REQ-003 both are the manifest name, so the gate
// finds the pack's declared rule file where it expects it.
func TestE2E_GateResolvesPackAssetsUnderManifestName(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, projectDir := remoteE2ESetup(t, "divergent-name-pack", "v1.0.0")

	if stdout, stderr, code := runBackstopStreams(t, bin, projectDir, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("the divergent add must succeed: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	const manifestName = "hermetic/renamed-pack"

	// The pack's DECLARED rule file must exist under the MANIFEST-name asset root — the
	// exact path the gate resolves engine assets from.
	assetRoot := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(manifestName))
	declaredRule := filepath.Join(assetRoot, "rules", "forbidden-marker.yml")
	if _, err := os.Stat(declaredRule); err != nil {
		t.Fatalf("the pack's declared rule file is not under the manifest-name asset root at %s: %v — this is the shape that produced the consumer's missing-asset failure", declaredRule, err)
	}

	// And nothing was written under the coordinate, which is where the old behavior put it.
	underCoordinate := filepath.Join(projectDir, ".backstop", "packs", filepath.FromSlash(packName))
	if _, err := os.Stat(underCoordinate); !os.IsNotExist(err) {
		t.Errorf("assets were installed under the COORDINATE at %s (stat error: %v); the gate would not look there", underCoordinate, err)
	}

	// Now run the real gate in the consumer and require no missing-asset finding for it.
	stdout, stderr, _ := runBackstopStreams(t, bin, projectDir, "gate")
	combined := stdout + stderr
	for _, symptom := range []string{"missing convert script", "missing rule file", "no such file"} {
		if strings.Contains(strings.ToLower(combined), symptom) && strings.Contains(combined, manifestName) {
			t.Errorf("the gate reported %q for %s; its assets must resolve under the manifest name.\nstdout: %s\nstderr: %s", symptom, manifestName, stdout, stderr)
		}
	}
}
