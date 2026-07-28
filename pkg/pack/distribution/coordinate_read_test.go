package distribution_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-005 read-side suite: every remote operation on an already-locked pack resolves
// its repository from the RECORDED coordinate, through ONE accessor, warning at most once.
//
// EVERY FIXTURE HERE SEPARATES THE KEY FROM THE COORDINATE. A lock whose key and
// coordinate coincide cannot falsify any of these claims: a URL built from either string
// looks correct, and an implementation that ignored the coordinate entirely would pass.
// The pair used throughout is the real one — the pack is NAMED backstop/harness-toolchain
// and LIVES at backstop-ai/backstop-harness-toolchain-pack.

const (
	readLockedPackName = "backstop/harness-toolchain"
	readCoordinate     = "backstop-ai/backstop-harness-toolchain-pack"
)

// urlRecordingCloner records every URL it is handed, so a test can assert WHICH
// repository a command reached for rather than only that it reached for something.
type urlRecordingCloner struct {
	urls     []string
	refs     []string
	listed   []string
	sourcefs string
}

func (c *urlRecordingCloner) Clone(url, version, destDir string) error {
	c.urls = append(c.urls, url)
	c.refs = append(c.refs, version)
	if c.sourcefs == "" {
		return os.MkdirAll(destDir, 0o755)
	}
	return copyDir(c.sourcefs, destDir)
}

func (c *urlRecordingCloner) ListTags(url string) ([]string, error) {
	c.listed = append(c.listed, url)
	return []string{"v1.0.0", "v1.1.0"}, nil
}

// coordinateRecordingResolver records the first argument it was handed — the thing this
// suite is about — while behaving like the ordinary resolver double.
type coordinateRecordingResolver struct {
	seen        []string
	latestMinor string
}

func (r *coordinateRecordingResolver) ResolveLatestCompatible(coordinate, _ string) (string, error) {
	r.seen = append(r.seen, coordinate)
	return r.latestMinor, nil
}

func (r *coordinateRecordingResolver) IsMajorBump(current, next string) bool {
	return strings.SplitN(current, ".", 2)[0] != strings.SplitN(next, ".", 2)[0]
}

// seedDivergentGitProject builds a consumer whose single git pack is keyed by its manifest
// name and carries a DIFFERENT recorded coordinate.
func seedDivergentGitProject(t *testing.T, coordinate string) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "backstop.yml"), "packs:\n  "+readLockedPackName+": \"1.0.0\"\n")

	packDir := filepath.Join(dir, ".backstop", "packs", filepath.FromSlash(readLockedPackName))
	writeFile(t, filepath.Join(packDir, "pack.yml"), identityManifestYAML(readLockedPackName, "1.0.0"))
	hash := mustContentHash(t, packDir)

	ref := "v1.0.0"
	entry := distribution.LockEntry{
		Name:        readLockedPackName,
		Version:     "1.0.0",
		GitRef:      &ref,
		ContentHash: hash,
		SourceType:  "git",
		InstallDate: "2026-01-01T00:00:00Z",
	}
	entry.SourceCoordinate = coordinate

	lf := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{readLockedPackName: entry}}
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	return dir
}

// assertURLNamesCoordinate requires the cloner to have reached for the COORDINATE and not
// for the lock key.
func assertURLNamesCoordinate(t *testing.T, urls []string, coordinate, lockKey string) {
	t.Helper()
	if len(urls) == 0 {
		t.Fatal("the cloner was never invoked, so this test cannot observe which repository was reached")
	}
	joined := strings.Join(urls, " ")
	if !strings.Contains(joined, coordinate) {
		t.Errorf("cloned from %v, want a URL built from the recorded coordinate %q", urls, coordinate)
	}
	if strings.Contains(joined, lockKey) {
		t.Errorf("cloned from %v, which names the LOCK KEY %q; the repository is the coordinate, not the pack name", urls, lockKey)
	}
}

// ── The four rewired call sites ─────────────────────────────────────────────────

func TestInstallCommand_Run_ClonesFromRecordedCoordinate(t *testing.T) {
	projectDir := seedDivergentGitProject(t, readCoordinate)
	// Remove the materialized tree so install must actually clone.
	if err := os.RemoveAll(filepath.Join(projectDir, ".backstop", "packs")); err != nil {
		t.Fatal(err)
	}

	source := t.TempDir()
	writeFile(t, filepath.Join(source, "pack.yml"), identityManifestYAML(readLockedPackName, "1.0.0"))
	cloner := &urlRecordingCloner{sourcefs: source}

	install := newTestInstallCommand(t, cloner)
	if _, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	assertURLNamesCoordinate(t, cloner.urls, readCoordinate, readLockedPackName)
}

func TestUpdateCommand_Run_ResolvesTagsAtRecordedCoordinate(t *testing.T) {
	projectDir := seedDivergentGitProject(t, readCoordinate)

	resolver := &coordinateRecordingResolver{latestMinor: "1.0.0"}
	update := newTestUpdateCommand(t, &urlRecordingCloner{}, &mockValidator{}, resolver)

	if _, err := update.Run(readLockedPackName, distribution.UpdateOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if len(resolver.seen) == 0 {
		t.Fatal("the resolver was never invoked")
	}
	if resolver.seen[0] != readCoordinate {
		t.Errorf("the resolver was handed %q, want the recorded coordinate %q — ls-remote must run against the repository, not the pack name",
			resolver.seen[0], readCoordinate)
	}
}

func TestUpdateCommand_Run_ClonesFromRecordedCoordinate(t *testing.T) {
	projectDir := seedDivergentGitProject(t, readCoordinate)

	source := filepath.Join("testdata", "valid-pack-v2")
	cloner := &urlRecordingCloner{sourcefs: source}
	update := newTestUpdateCommand(t, cloner, &mockValidator{}, &coordinateRecordingResolver{latestMinor: "1.1.0"})

	if _, err := update.Run(readLockedPackName, distribution.UpdateOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	assertURLNamesCoordinate(t, cloner.urls, readCoordinate, readLockedPackName)
}

func TestUpgradeCommand_Run_ClonesFromRecordedCoordinate(t *testing.T) {
	projectDir := seedDivergentGitProject(t, readCoordinate)

	source := filepath.Join("testdata", "valid-pack-v3")
	cloner := &urlRecordingCloner{sourcefs: source}
	upgrade := newTestUpgradeCommand(t, cloner, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	if _, err := upgrade.Run(readLockedPackName+"@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	assertURLNamesCoordinate(t, cloner.urls, readCoordinate, readLockedPackName)
}

// TestTagVersionResolver_ResolveLatestCompatible_UsesSuppliedCoordinate pins that the
// resolver builds its ls-remote URL from the coordinate it was PASSED (CLM-058), rather
// than deriving one from anything else.
func TestTagVersionResolver_ResolveLatestCompatible_UsesSuppliedCoordinate(t *testing.T) {
	cloner := &urlRecordingCloner{}
	resolver, err := distribution.NewTagVersionResolver(cloner)
	if err != nil {
		t.Fatalf("assembling the resolver: %v", err)
	}

	// The tags the double publishes make 1.1.0 the latest compatible with 1.0.0.
	if _, resolveErr := resolver.ResolveLatestCompatible(readCoordinate, "1.0.0"); resolveErr != nil {
		t.Fatalf("ResolveLatestCompatible: %v", resolveErr)
	}

	if len(cloner.listed) == 0 {
		t.Fatal("the resolver never listed tags, so this test cannot observe the URL it built")
	}
	joined := strings.Join(cloner.listed, " ")
	if !strings.Contains(joined, readCoordinate) {
		t.Errorf("the resolver listed tags at %v, want a URL built from the supplied coordinate %q", cloner.listed, readCoordinate)
	}
}

// ── The single-emission claim ───────────────────────────────────────────────────

// TestUpdateCommand_Run_FallbackWarningEmittedOncePerInvocation is the load-bearing one
// (CLM-059).
//
// Update needs the coordinate at TWO points — once for ls-remote, once for the clone. Two
// independent resolutions produce two identical warnings for ONE invocation, which is the
// noise that trains operators to ignore the signal entirely.
//
// IT REQUIRES EXACTLY 1, NOT >= 1. A `>= 1` assertion passes for the double-emission this
// claim exists to prevent, which would make it decorative.
func TestUpdateCommand_Run_FallbackWarningEmittedOncePerInvocation(t *testing.T) {
	// NO recorded coordinate — the shape of every entry written before this spec.
	projectDir := seedDivergentGitProject(t, "")

	cloner := &urlRecordingCloner{sourcefs: filepath.Join("testdata", "valid-pack-v2")}
	update := newTestUpdateCommand(t, cloner, &mockValidator{}, &coordinateRecordingResolver{latestMinor: "1.1.0"})

	result, err := update.Run(readLockedPackName, distribution.UpdateOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	matches := 0
	for _, w := range result.Warnings {
		if strings.Contains(w, "source_coordinate") || strings.Contains(strings.ToLower(w), "coordinate") {
			matches++
		}
	}
	if matches != 1 {
		t.Errorf("the fallback warning was emitted %d time(s), want EXACTLY 1 — update resolves the coordinate twice in one invocation and must warn once.\nWarnings: %v",
			matches, result.Warnings)
	}
}

// ── Local sources have no remote ────────────────────────────────────────────────

// seedLocalProject builds a consumer whose single pack is a LOCAL source.
func seedLocalProject(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	const name = "internal/local-rules"

	writeFile(t, filepath.Join(dir, "backstop.yml"), "packs:\n  "+name+": local\n")

	srcRel := "local-src"
	writeFile(t, filepath.Join(dir, srcRel, "pack.yml"), identityManifestYAML(name, "1.0.0"))
	hash := mustContentHash(t, filepath.Join(dir, srcRel))

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			name: {
				Name: name, ContentHash: hash, SourceType: "local",
				InstallDate: "2026-01-01T00:00:00Z", LocalPath: srcRel,
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(dir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	return dir, name
}

func TestInstallCommand_Run_LocalSourceNeedsNoCoordinate(t *testing.T) {
	projectDir, packName := seedLocalProject(t)

	cloner := &urlRecordingCloner{}
	install := newTestInstallCommand(t, cloner)

	result, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if len(cloner.urls) != 0 {
		t.Errorf("a local-source install of %s cloned from %v; it must materialize from local_path", packName, cloner.urls)
	}
	for _, w := range result.Warnings {
		if strings.Contains(strings.ToLower(w), "coordinate") {
			t.Errorf("a local-source install emitted a coordinate fallback warning %q; a local entry has no repository, so asking for its coordinate is a category error", w)
		}
	}
}

func TestUpdateCommand_Run_LocalSourceRemainsNoOpWithoutIdentityGate(t *testing.T) {
	projectDir, name := seedLocalProject(t)

	cloner := &urlRecordingCloner{}
	resolver := &coordinateRecordingResolver{latestMinor: "1.1.0"}
	update := newTestUpdateCommand(t, cloner, &mockValidator{}, resolver)

	result, err := update.Run(name, distribution.UpdateOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("a local-source update must be a clean no-op: %v", err)
	}
	if !result.NoOp {
		t.Error("a local-source update must report NoOp")
	}
	if len(cloner.urls) != 0 {
		t.Errorf("a local-source update cloned from %v", cloner.urls)
	}
	if len(resolver.seen) != 0 {
		t.Errorf("a local-source update resolved a coordinate %v; a local source has no tag to resolve", resolver.seen)
	}
}

// TestUpgradeCommand_Run_LocalSourceRefusedBeforeIdentityGate covers the NEW guard
// (CLM-027). Upgrade discards readPackVersion's isLocal result today and clones
// unconditionally.
func TestUpgradeCommand_Run_LocalSourceRefusedBeforeIdentityGate(t *testing.T) {
	projectDir, name := seedLocalProject(t)

	cloner := &urlRecordingCloner{}
	upgrade := newTestUpgradeCommand(t, cloner, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	_, err := upgrade.Run(name+"@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("upgrading a local-source pack must refuse; there is no repository to upgrade from")
	}
	if !strings.Contains(err.Error(), "relock") {
		t.Errorf("the diagnostic must name `pack relock` as the local path forward, got: %v", err)
	}
	if len(cloner.urls) != 0 {
		t.Errorf("the guard must refuse BEFORE any clone; the cloner was invoked with %v", cloner.urls)
	}
}

// TestUpgradeCommand_Run_LocalSourceResolvesNoCoordinate is the other half of the guard
// (CLM-061): without it, REQ-005's fallback warning would fire on a pack that has no
// repository at all.
func TestUpgradeCommand_Run_LocalSourceResolvesNoCoordinate(t *testing.T) {
	projectDir, name := seedLocalProject(t)

	upgrade := newTestUpgradeCommand(t, &urlRecordingCloner{}, &mockValidator{}, &mockScanner{}, &mockRemediationGenerator{})

	result, err := upgrade.Run(name+"@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("expected the local guard to refuse")
	}
	if result != nil {
		for _, w := range result.Warnings {
			if strings.Contains(strings.ToLower(w), "coordinate") {
				t.Errorf("a refused local upgrade emitted a coordinate fallback warning %q", w)
			}
		}
	}
}

// ── Install does not re-validate identity (DD-12) ───────────────────────────────

// TestInstallCommand_Run_DoesNotApplyManifestVersionEquality pins spec Review Question 13
// (CLM-025).
//
// Install is the HASH-VERIFIED RESTORE path. Applying the manifest-version equality check
// here would make install fail on a repository whose manifest drifted AFTER a successful
// add — even though the bytes it restored are exactly the bytes that were locked and the
// hash proves it.
func TestInstallCommand_Run_DoesNotApplyManifestVersionEquality(t *testing.T) {
	projectDir := seedDivergentGitProject(t, readCoordinate)

	// The remote's manifest declares a version that disagrees with the locked 1.0.0 —
	// but its BYTES are what the lock's hash was computed over.
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "pack.yml"), identityManifestYAML(readLockedPackName, "1.0.0"))
	drifted := mustContentHash(t, source)

	lf, err := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil {
		t.Fatal(err)
	}
	entry := lf.Packs[readLockedPackName]
	entry.ContentHash = drifted
	entry.Version = "9.9.9" // the LOCKED version disagrees with the manifest's 1.0.0
	lf.Packs[readLockedPackName] = entry
	if writeErr := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); writeErr != nil {
		t.Fatal(writeErr)
	}
	if rmErr := os.RemoveAll(filepath.Join(projectDir, ".backstop", "packs")); rmErr != nil {
		t.Fatal(rmErr)
	}

	install := newTestInstallCommand(t, &urlRecordingCloner{sourcefs: source})
	if _, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir}); err != nil {
		var mismatch *distribution.VersionMismatchError
		if errors.As(err, &mismatch) {
			t.Fatalf("install applied the manifest-version equality check; it is the hash-verified restore path and re-validates nothing (DD-12): %v", err)
		}
		t.Fatalf("Install: %v", err)
	}
}
