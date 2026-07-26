package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// pack_remote_e2e_test.go drives the REAL pack lifecycle over a REAL git clone through
// the BUILT binary (SPEC-055 REQ-010). These are the claims that decide whether this
// spec delivered anything: every other proof in the suite runs against a double, and a
// double cannot tell you whether production assembles its dependencies at all.
//
// HERMETIC BY CONSTRUCTION, NOT BY CONVENTION. Every test installs a
// `url.<file://…>.insteadOf` redirect for the exact URL distribution builds, and proves
// with assertPackURLRedirected that the redirect reaches a CHILD process before it
// asserts on any lifecycle behavior. Nothing here reaches the network; a redirect that
// silently missed would be a green suite talking to GitHub.
//
// Streams are read through runBackstopStreams, never the merged helper: "the diagnostic
// is on stderr" is not assertable against a merged buffer.

// remoteE2EOrg is the organization half of every hermetic pack coordinate here. It
// cannot exist upstream, so a redirect that missed fails rather than fetching something.
const remoteE2EOrg = "hermetic"

const (
	validPackFixture   = "valid-pack"
	invalidPackFixture = "invalid-pack"
)

// TestE2E_PackAdd_RemoteTaggedRepositoryInstallsAndLocks (CLM-062) — THE HEADLINE.
//
// `pack add hermetic/valid-pack@1.0.0` against a hermetic tagged repository performs a
// real git clone through the production path and leaves the pack installed on disk AND
// recorded in backstop.lock. Before this spec the same invocation nil-dereferenced
// (ISSUE-073) because cmd/backstop passed an AddOptions with no cloner in it.
//
// Content and lock entry are both asserted. Exit 0 alone would pass for a command that
// resolved the ref and installed nothing.
func TestE2E_PackAdd_RemoteTaggedRepositoryInstallsAndLocks(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, project := remoteE2ESetup(t, validPackFixture, "v1.0.0")

	stdout, stderr, code := runBackstopStreams(t, bin, project, "pack", "add", packName+"@1.0.0")
	if code != 0 {
		t.Fatalf("pack add of a hermetic tagged repository must succeed, got exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Added "+packName+"@1.0.0") {
		t.Errorf("expected the success line naming the pack and version, got stdout: %q", stdout)
	}

	installed := filepath.Join(project, ".backstop", "packs", packName)
	for _, relative := range []string{"pack.yml", filepath.Join("rules", "forbidden-marker.yml")} {
		if _, err := os.Stat(filepath.Join(installed, relative)); err != nil {
			t.Errorf("the clone's %s was not installed: %v", relative, err)
		}
	}
	// The clone's repository metadata must never reach the consumer's project: it is
	// what would make the recorded hash unreproducible on the next machine.
	if _, err := os.Stat(filepath.Join(installed, ".git")); !os.IsNotExist(err) {
		t.Errorf("the installed pack still carries a .git directory (stat error: %v); the clone strip did not run", err)
	}

	entry := remoteE2ELockEntry(t, project, packName)
	if entry.Version != "1.0.0" {
		t.Errorf("lock entry version = %q, want 1.0.0", entry.Version)
	}
	if entry.SourceType != "git" {
		t.Errorf("lock entry source_type = %q, want git — a remote add that recorded a local source proves nothing was cloned", entry.SourceType)
	}
	if entry.GitRef == nil || *entry.GitRef != "v1.0.0" {
		t.Errorf("lock entry git_ref = %v, want v1.0.0", entry.GitRef)
	}
	if entry.ContentHash == "" {
		t.Error("lock entry carries no content hash; there would be nothing for a later install to verify against")
	}
}

// TestE2E_PackAdd_MissingTagDiagnosticNotPanic (CLM-063) — the closure of ISSUE-073.
//
// Adding a tag the repository does not publish exits non-zero with a diagnostic an
// operator can read. The absence of a stack trace is the point: the whole reported
// symptom was a raw SIGSEGV with no message, and a panic that merely produced a
// non-zero exit would still satisfy an exit-code-only assertion.
func TestE2E_PackAdd_MissingTagDiagnosticNotPanic(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, project := remoteE2ESetup(t, validPackFixture, "v1.0.0")

	stdout, stderr, code := runBackstopStreams(t, bin, project, "pack", "add", packName+"@2.0.0")
	if code == 0 {
		t.Fatalf("pack add of an absent tag must exit non-zero\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	assertNoStackTrace(t, "stderr", stderr)
	assertNoStackTrace(t, "stdout", stdout)

	if !strings.Contains(stderr, packName) {
		t.Errorf("the diagnostic must name the pack that could not be cloned, got stderr: %q", stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "clon") {
		t.Errorf("the diagnostic must say the clone failed, got stderr: %q", stderr)
	}
	if _, err := os.Stat(filepath.Join(project, "backstop.lock")); !os.IsNotExist(err) {
		t.Errorf("a failed add must lock nothing (stat error: %v)", err)
	}
}

// TestE2E_PackAdd_ValidationFailureIsLoud (CLM-055) — proof the validator is WIRED,
// not merely present.
//
// The fixture fails `pack check` in phase1-structural and nothing else: everything in
// it parses, so the only thing that can reject it is validation. Against the pre-spec
// code — where a nil validator skipped both phases — an invalid pack installed cleanly,
// so a test that only asserted a VALID pack installs would have passed against the
// broken wiring. Asserting that NOTHING was installed is the other half: validation
// that ran but did not abort the pipeline would leave content behind.
func TestE2E_PackAdd_ValidationFailureIsLoud(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, project := remoteE2ESetup(t, invalidPackFixture, "v1.0.0")

	stdout, stderr, code := runBackstopStreams(t, bin, project, "pack", "add", packName+"@1.0.0")
	if code == 0 {
		t.Fatalf("adding a pack that fails validation must exit non-zero\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "pack check") {
		t.Errorf("the diagnostic must name the validation step that rejected the pack, got stderr: %q", stderr)
	}
	if !strings.Contains(stderr, "validation") {
		t.Errorf("the diagnostic must name the validation failure, got stderr: %q", stderr)
	}

	if _, err := os.Stat(filepath.Join(project, ".backstop", "packs", packName)); !os.IsNotExist(err) {
		t.Errorf("a pack that failed validation must not be installed (stat error: %v)", err)
	}
	if _, err := os.Stat(filepath.Join(project, "backstop.lock")); !os.IsNotExist(err) {
		t.Errorf("a pack that failed validation must not be locked (stat error: %v)", err)
	}
}

// TestE2E_PackInstall_CacheRestore (CLM-064) — the airgapped path.
//
// `pack install --cache` restores a locked pack from a local directory and never
// clones. This is the case the no-PATH-probe rule in NewExecGitCloner exists for:
// assembly must succeed where git is unusable, because this path does not need it.
func TestE2E_PackInstall_CacheRestore(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, project := remoteE2ESetup(t, validPackFixture, "v1.0.0")

	if _, stderr, code := runBackstopStreams(t, bin, project, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("seeding the lock with a real add failed: exit %d\nstderr: %s", code, stderr)
	}

	// The cache holds the pack content under its name, which is where install looks.
	cache := t.TempDir()
	copyTree(t, filepath.Join(project, ".backstop", "packs", packName), filepath.Join(cache, packName))

	restored := remoteE2EFreshConsumer(t, project)
	stdout, stderr, code := runBackstopStreams(t, bin, restored, "pack", "install", "--cache", cache)
	if code != 0 {
		t.Fatalf("pack install --cache must restore the locked pack, got exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, packName) {
		t.Errorf("the installed summary must name the restored pack, got stdout: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(restored, ".backstop", "packs", packName, "pack.yml")); err != nil {
		t.Errorf("the cached pack was not materialized: %v", err)
	}
}

// TestE2E_PackAddThenInstall_RoundTripHashesMatch (CLM-065) — THE ROUND TRIP.
//
// A pack added from a tag, and then installed from the committed lock on a SEPARATE
// project that re-clones that same tag, produces the SAME content hash. That is the
// property a lock file is for: what one machine recorded, another machine reproduces.
//
// TWO CLONES ARE LOAD-BEARING. The install is driven with NO --cache and against a
// project that does not share the add's materialized tree, so it genuinely re-clones. A
// cache-restore or a reuse of the add's directory would compare a hash to itself and
// prove nothing about reproducibility.
//
// The reason this can hold at all is that Clone strips the root .git before returning:
// two clones of one tag are otherwise identical except for reflog timestamps and object
// layout, which is exactly what the strip removes.
//
// Its anti-vacuity twin is TestE2E_PackInstall_GitSourceHashMismatchIsLoud. Equality on
// its own is satisfiable by deleting hash verification entirely; NEITHER TEST MAY BE
// DROPPED AS REDUNDANT WITH THE OTHER.
func TestE2E_PackAddThenInstall_RoundTripHashesMatch(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, adder := remoteE2ESetup(t, validPackFixture, "v1.0.0")

	if _, stderr, code := runBackstopStreams(t, bin, adder, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("pack add failed: exit %d\nstderr: %s", code, stderr)
	}
	addedHash := remoteE2ELockEntry(t, adder, packName).ContentHash

	// The lock and manifest travel to a fresh project, exactly as committing them and
	// checking the repository out elsewhere would deliver them.
	installer := remoteE2EFreshConsumer(t, adder)

	stdout, stderr, code := runBackstopStreams(t, bin, installer, "pack", "install")
	if code != 0 {
		t.Fatalf("pack install from the committed lock must succeed on a fresh clone of the same tag, got exit %d\nstdout: %s\nstderr: %s\n"+
			"if this reds, the clone's .git strip is missing or mis-ordered rather than the hash being wrong", code, stdout, stderr)
	}

	installedHash, err := distribution.ComputeContentHash(filepath.Join(installer, ".backstop", "packs", packName))
	if err != nil {
		t.Fatalf("hashing the installed pack: %v", err)
	}
	if installedHash != addedHash {
		t.Errorf("round-trip content hash mismatch: add recorded %s, a fresh install produced %s; a remote pack's hash is not reproducible across clones",
			addedHash, installedHash)
	}
}

// TestE2E_PackInstall_GitSourceHashMismatchIsLoud (CLM-103) — the anti-vacuity twin of
// the round trip.
//
// A git-sourced lock entry whose recorded hash does not match what the clone produced
// fails loudly, naming BOTH hashes so an operator can see which side moved. Without
// this, the round-trip equality claim would stay green if hash verification were
// deleted outright — which is the one change that would make every pack silently
// installable regardless of its content.
func TestE2E_PackInstall_GitSourceHashMismatchIsLoud(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, adder := remoteE2ESetup(t, validPackFixture, "v1.0.0")

	if _, stderr, code := runBackstopStreams(t, bin, adder, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("pack add failed: exit %d\nstderr: %s", code, stderr)
	}
	recordedHash := remoteE2ELockEntry(t, adder, packName).ContentHash

	installer := remoteE2EFreshConsumer(t, adder)
	corruptHash := strings.Repeat("0", len(recordedHash))
	remoteE2ECorruptLockHash(t, installer, packName, corruptHash)

	stdout, stderr, code := runBackstopStreams(t, bin, installer, "pack", "install")
	if code == 0 {
		t.Fatalf("pack install against a lock whose hash does not match the clone must fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, corruptHash) {
		t.Errorf("the diagnostic must name the LOCKED hash %s, got stderr: %q", corruptHash, stderr)
	}
	if !strings.Contains(stderr, recordedHash) {
		t.Errorf("the diagnostic must name the COMPUTED hash %s, got stderr: %q", recordedHash, stderr)
	}
	if _, err := os.Stat(filepath.Join(installer, ".backstop", "packs", packName)); !os.IsNotExist(err) {
		t.Errorf("a hash-mismatched pack must not be left installed (stat error: %v)", err)
	}
}

// TestE2E_PackUpdate_ResolvesNewerCompatibleTag (CLM-066) — update resolves a real tag
// listing.
//
// The repository publishes v1.0.0 and v1.1.0 with genuinely different content, so a
// resolver that returned the current version, or one that resolved but installed the
// old tree, both fail here.
func TestE2E_PackUpdate_ResolvesNewerCompatibleTag(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, project := remoteE2ESetup(t, validPackFixture, "v1.0.0", "v1.1.0")

	if _, stderr, code := runBackstopStreams(t, bin, project, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("seeding the project with a real add failed: exit %d\nstderr: %s", code, stderr)
	}

	stdout, stderr, code := runBackstopStreams(t, bin, project, "pack", "update", packName)
	if code != 0 {
		t.Fatalf("pack update must resolve and apply the newer compatible tag, got exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "1.0.0 -> 1.1.0") {
		t.Errorf("expected the update to report 1.0.0 -> 1.1.0, got stdout: %q", stdout)
	}

	entry := remoteE2ELockEntry(t, project, packName)
	if entry.Version != "1.1.0" {
		t.Errorf("lock entry version = %q after update, want 1.1.0", entry.Version)
	}
	// The newer tag's content is materially different (its manifest names the new
	// version), so an update that resolved but reinstalled the old tree is caught.
	manifest, err := os.ReadFile(filepath.Join(project, ".backstop", "packs", packName, "pack.yml"))
	if err != nil {
		t.Fatalf("reading the updated pack manifest: %v", err)
	}
	if !strings.Contains(string(manifest), "version: 1.1.0") {
		t.Errorf("the installed pack still carries the old tag's content: %q", string(manifest))
	}
}

// TestE2E_PackUpgrade_UnavailableCapabilityIsLoud (CLM-060) — pack upgrade gets WORSE
// before it gets better, and that is correct.
//
// The violation scan and remediation generation are BUNDLE-006 REQ-014/REQ-018 and are
// not built. Before this spec they were nil and silently skipped, so a major upgrade
// reported success with zero baselined violations it had never scanned for — a vacuous
// green on the single operation where the consumer most needs to know what broke. Now
// the missing capability is reported and the upgrade fails.
func TestE2E_PackUpgrade_UnavailableCapabilityIsLoud(t *testing.T) {
	bin := buildBackstopBinary(t)
	packName, project := remoteE2ESetup(t, validPackFixture, "v1.0.0", "v2.0.0")

	if _, stderr, code := runBackstopStreams(t, bin, project, "pack", "add", packName+"@1.0.0"); code != 0 {
		t.Fatalf("seeding the project with a real add failed: exit %d\nstderr: %s", code, stderr)
	}

	stdout, stderr, code := runBackstopStreams(t, bin, project, "pack", "upgrade", packName+"@2.0.0")
	if code == 0 {
		t.Fatalf("pack upgrade must report the unavailable scan capability rather than succeeding\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if strings.Contains(stdout, "Upgraded") {
		t.Errorf("pack upgrade reported success despite an unavailable capability, stdout: %q", stdout)
	}
	if !strings.Contains(strings.ToLower(stderr), "scan") {
		t.Errorf("the diagnostic must name the unavailable capability, got stderr: %q", stderr)
	}
	if !strings.Contains(stderr, "REQ-014") || !strings.Contains(stderr, bundleReference) {
		t.Errorf("the diagnostic must cite %s REQ-014 as the requirement tracking the gap, or it reads as a defect rather than as scheduled work; got stderr: %q",
			bundleReference, stderr)
	}
}

// remoteE2ESetup builds a hermetic repository from the named fixture at the given tags,
// redirects the production URL at it, PROVES the redirect reaches a child process, and
// returns the pack's coordinate together with a fresh consumer project.
//
// The redirect proof is not ceremony: a missed rewrite does not fail, it silently sends
// the child to github.com, and the whole suite becomes network-dependent with nothing
// turning red.
func remoteE2ESetup(t *testing.T, fixture string, tags ...string) (packName, projectDir string) {
	t.Helper()

	source, err := filepath.Abs(filepath.Join("testdata", "hermetic-remote", fixture))
	if err != nil {
		t.Fatalf("resolving the %s fixture: %v", fixture, err)
	}

	remote := newHermeticRemote(t, source, tags...)
	redirectPackURL(t, remoteE2EOrg, fixture, remote.Path)
	assertPackURLRedirected(t, remoteE2EOrg, fixture, remote)

	return remoteE2EOrg + "/" + fixture, newConsumerProject(t)
}

// remoteE2EFreshConsumer returns a new project directory carrying ONLY the manifest and
// lock from source — no .backstop/, no materialized pack.
//
// That absence is what makes an install genuinely re-clone. A fresh project that
// inherited the added tree would let install verify a hash against the very bytes the
// add produced.
func remoteE2EFreshConsumer(t *testing.T, source string) string {
	t.Helper()

	fresh := t.TempDir()
	for _, name := range []string{"backstop.yml", "backstop.lock"} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatalf("reading %s to carry into a fresh project: %v", name, err)
		}
		if writeErr := os.WriteFile(filepath.Join(fresh, name), data, 0o644); writeErr != nil {
			t.Fatalf("writing %s into the fresh project: %v", name, writeErr)
		}
	}
	if _, err := os.Stat(filepath.Join(fresh, ".backstop")); !os.IsNotExist(err) {
		t.Fatalf("the fresh project must carry no materialized packs (stat error: %v)", err)
	}
	return fresh
}

// remoteE2ELockEntry reads the lock entry a lifecycle command recorded for packName.
func remoteE2ELockEntry(t *testing.T, projectDir, packName string) distribution.LockEntry {
	t.Helper()

	lockfile, err := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil {
		t.Fatalf("reading backstop.lock: %v", err)
	}
	entry, ok := lockfile.Packs[packName]
	if !ok {
		t.Fatalf("backstop.lock records no entry for %s; it holds %d entries", packName, len(lockfile.Packs))
	}
	return entry
}

// remoteE2ECorruptLockHash rewrites packName's recorded content hash, standing in for a
// lock that disagrees with what the remote now publishes.
func remoteE2ECorruptLockHash(t *testing.T, projectDir, packName, hash string) {
	t.Helper()

	lockPath := filepath.Join(projectDir, "backstop.lock")
	lockfile, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("reading backstop.lock: %v", err)
	}
	entry, ok := lockfile.Packs[packName]
	if !ok {
		t.Fatalf("backstop.lock records no entry for %s", packName)
	}
	entry.ContentHash = hash
	lockfile.Packs[packName] = entry
	if writeErr := distribution.WriteLockfile(lockPath, lockfile); writeErr != nil {
		t.Fatalf("writing the corrupted backstop.lock: %v", writeErr)
	}
}

// assertNoStackTrace requires that stream carries no Go panic output. A panic that
// happened to exit non-zero would satisfy an exit-code assertion while still being the
// unreadable crash this claim exists to close.
func assertNoStackTrace(t *testing.T, stream, content string) {
	t.Helper()

	for _, marker := range []string{"panic:", "goroutine ", "runtime error:"} {
		if strings.Contains(content, marker) {
			t.Errorf("%s carries a stack trace (%q); a missing tag must produce a diagnostic, not a crash:\n%s", stream, marker, content)
		}
	}
}
