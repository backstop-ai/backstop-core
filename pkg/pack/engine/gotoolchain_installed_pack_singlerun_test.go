package engine_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// gotoolchain_installed_pack_singlerun_test.go (ISSUE-172).
//
// WHY THIS FILE EXISTS AT ALL — AND WHY THE FIXTURE PIN IS NOT ENOUGH. The
// go-toolchain assertions in cmd/backstop pin the convention against the FIXTURE at
// cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain, which is
// a deliberately diverged older snapshot (name backstop/go-toolchain, version 1.1.0).
// The pack that actually runs in backstop-core's REAL gate is the INSTALLED
// backstop-ai/go-toolchain under .backstop/packs/ — a different tree, at a different
// name and a different version. Without this pin the claim dies with this plan's
// terminal: the fixture could carry the convention forever while the pack core
// actually dispatches quietly reverted, and every gate run would go back to paying
// for two whole-module `go test` runs with nothing red.
//
// It is in `engine_test` for the same reason grep_installed_pack_test.go is: it
// resolves the pack through the PRODUCTION readers (pack.ParseManifestFile,
// distribution.ReadLockfile), and `pkg/pack` imports `pkg/pack/engine`, so an
// in-package test importing them is an import cycle.
//
// It reuses conventionRepoRoot and semverGreater from this package's existing test
// files. It deliberately does NOT reuse preFixContractsPackVersion: that constant is
// 1.2.0 and belongs to go-contracts, so reusing it would pass against a still-defective
// go-toolchain — a check reading green for the wrong reason.

// installedGoToolchainPackName is the pack that actually runs — the INSTALLED, LOCKED
// pack the gate reads, NOT the cmd/backstop testdata fixture, which has already
// diverged on both name and version.
const installedGoToolchainPackName = "backstop-ai/go-toolchain"

// preFixGoToolchainPackVersion is the version that carried the BACKWARDS FRESHNESS
// COMPARISON (ISSUE-179) — 1.7.0. That is a different defect from the one 1.6.0
// carried: 1.6.0 ran the whole-module suite TWICE (ISSUE-172), and 1.7.0 shipped the
// reuse mechanism that was supposed to remove the second run but compared the two
// mtimes in the wrong direction, so the reuse never fired on a nanosecond-resolution
// `-ot` and the second run came back on Linux with nothing red.
//
// The bar MOVED from 1.6.0 to 1.7.0 deliberately. The lock already pinned 1.7.0, so a
// bar at 1.6.0 was ALREADY satisfied and could never red on this defect — a check that
// cannot distinguish the good state from the bad one.
//
// The fix ships as a MINOR bump above it: it changes what a shipped engine executes,
// which is a behaviour change in a released rule path, not a patch.
//
// It stays a BAR, not an equality pin — a later unrelated bump keeps this green; only a
// revert below the fix reds it.
const preFixGoToolchainPackVersion = "1.7.0"

// goToolchainSharedProfile is the profile path the two engines must agree on. go-test
// WRITES it (via -coverprofile) and go-coverage READS it (as its stdout_artifact); if
// they ever name different files the reuse goes silently dark and the suite runs twice
// again.
const goToolchainSharedProfile = "cover.out"

// goToolchainFreshStampPath is the freshness stamp test-produce.sh writes after a
// whole-module run and coverage-produce.sh consumes. Asserting on the STAMP PATH and
// the `-ot` comparison pins the MECHANISM; asserting on a comment string would pin
// prose, which drifts.
const goToolchainFreshStampPath = ".backstop/go-coverage-fresh"

// The two freshness comparisons, spelled EXACTLY as the producer spells them —
// asymmetrically quoted, bare `cover.out` and quoted `"$stamp"`. The quoting is part of
// the literal being matched, so a cosmetic re-quote in the script breaks these pins;
// that is the intended sensitivity, since the operand ORDER is the mechanism and there
// is no way to read it other than literally.
const (
	// goToolchainCorrectFreshnessTest is the direction a successful run actually
	// produces: the profile is written first, so the stamp must be the newer file.
	goToolchainCorrectFreshnessTest = `"$stamp" -ot cover.out`
	// goToolchainBackwardsFreshnessTest is the ISSUE-179 defect: it demands the profile
	// be no-older-than the stamp, which inverts the real write-then-touch order.
	goToolchainBackwardsFreshnessTest = `cover.out -ot "$stamp"`
)

// TestInstalledGoToolchainPack_CarriesSingleRunConvention asserts the RELEASED pack
// carries the single-run convention and that core actually consumes it.
//
// Three legs, split by what each can prove where:
//   - THE LOCK leg reads TRACKED data, so it holds even in a fresh checkout where the
//     pack fleet is not installed. It is what keeps this test from being vacuous when
//     the content legs cannot run.
//   - THE MANIFEST and PRODUCER legs read the INSTALLED tree, so they follow the
//     established guard: a load ERROR fails, a genuinely-absent fleet skips with a
//     `pack install` directive.
func TestInstalledGoToolchainPack_CarriesSingleRunConvention(t *testing.T) {
	root := conventionRepoRoot(t)

	// ── LEG 1: THE LOCK (tracked data — never skipped) ──────────────────────────
	lockPath := filepath.Join(root, "backstop.lock")
	lockfile, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		// A tracked lockfile that will not read is a broken repo, never a skip.
		t.Fatalf("ReadLockfile(%s): %v", lockPath, err)
	}
	entry, ok := lockfile.Packs[installedGoToolchainPackName]
	if !ok {
		t.Fatalf("backstop.lock has no entry for %s; core's build/test/lint/coverage gate rides that pack",
			installedGoToolchainPackName)
	}
	// A BAR, NOT A PIN — deliberately. A later unrelated bump keeps this green; only a
	// revert below the fix reds it.
	if !semverGreater(t, entry.Version, preFixGoToolchainPackVersion) {
		t.Errorf("backstop.lock pins %s at %s, which is not greater than the pre-fix %s — "+
			"the released pack still runs the whole-module suite TWICE (once for go-test, once for "+
			"go-coverage), which is the entire cost ISSUE-172 removes",
			installedGoToolchainPackName, entry.Version, preFixGoToolchainPackVersion)
	}
	gitRef := "<none>"
	if entry.GitRef != nil {
		gitRef = *entry.GitRef
	}
	t.Logf("backstop.lock: %s version=%s git_ref=%s source_type=%s",
		installedGoToolchainPackName, entry.Version, gitRef, entry.SourceType)

	// ── The installed tree, behind the established guard ────────────────────────
	packRoot := filepath.Join(root, ".backstop", "packs", installedGoToolchainPackName)
	manifestPath := filepath.Join(packRoot, "pack.yml")
	manifest, err := pack.ParseManifestFile(manifestPath)
	if err != nil {
		// Distinguish "not installed" from "installed and broken". Only the former
		// skips; a manifest that exists and will not parse is a real failure.
		if _, statErr := os.Stat(manifestPath); statErr != nil {
			t.Skipf("%s is not installed — run `./bin/backstop pack install` (the pack fleet is not installed)",
				installedGoToolchainPackName)
		}
		t.Fatalf("ParseManifestFile(%s): %v", manifestPath, err)
	}

	// ── LEG 2: THE INSTALLED MANIFEST — one run produces the profile ────────────
	testSpec, ok := manifest.Engines["go-test"]
	if !ok {
		t.Fatalf("installed %s declares no go-test engine", installedGoToolchainPackName)
	}
	t.Logf("installed %s v%s go-test command: %q", manifest.Name, manifest.Version, testSpec.Command)
	if !strings.Contains(testSpec.Command, "-coverprofile="+goToolchainSharedProfile) {
		t.Errorf("installed %s go-test command %q carries no -coverprofile=%s — this is the pack REAL "+
			"CONSUMERS install, and without the flag its run produces no profile for the coverage "+
			"producer to reuse, so the suite runs twice",
			installedGoToolchainPackName, testSpec.Command, goToolchainSharedProfile)
	}

	coverageSpec, ok := manifest.Engines["go-coverage"]
	if !ok {
		t.Fatalf("installed %s declares no go-coverage engine", installedGoToolchainPackName)
	}
	if coverageSpec.StdoutArtifact != goToolchainSharedProfile {
		t.Errorf("installed %s go-coverage stdout_artifact = %q, want %q — the two engines must name the "+
			"SAME profile or go-test writes a file nothing reads and the reuse is silently dark",
			installedGoToolchainPackName, coverageSpec.StdoutArtifact, goToolchainSharedProfile)
	}

	// ── LEG 3: THE INSTALLED PRODUCER — the reuse branch is really there ────────
	if coverageSpec.Producer == "" {
		t.Fatalf("installed %s go-coverage declares no producer; the reuse check lives in it",
			installedGoToolchainPackName)
	}
	producerPath := filepath.Join(packRoot, filepath.FromSlash(coverageSpec.Producer))
	raw, err := os.ReadFile(producerPath)
	if err != nil {
		t.Fatalf("reading the installed coverage producer %s: %v", producerPath, err)
	}
	producer := string(raw)
	// Assert on the MECHANISM — the stamp path and the freshness comparison — never on
	// a comment string, which is prose and will drift.
	if !strings.Contains(producer, goToolchainFreshStampPath) {
		t.Errorf("installed %s coverage producer never mentions the stamp path %s, so it cannot know a "+
			"whole-module profile was already produced in this gate invocation and will re-run the suite",
			installedGoToolchainPackName, goToolchainFreshStampPath)
	}
	// ★ A PRESENCE CHECK ON `-ot` WAS THE ASSERTION THAT LET ISSUE-179 SHIP. The
	// backwards form and the correct form both contain `-ot`, so the old check was
	// satisfied equally by a comparison that worked and one that could never fire — the
	// exact "cannot distinguish the good state from the bad one" defect class this file
	// exists to prevent. What must be pinned is the DIRECTION, and the direction lives
	// in the OPERAND ORDER, not in a comment (comments are prose and drift).
	if !strings.Contains(producer, goToolchainCorrectFreshnessTest) {
		t.Errorf("installed %s coverage producer does not carry the freshness comparison %q. "+
			"test-produce.sh writes cover.out and THEN touches the stamp, so on a genuine same-invocation "+
			"success the STAMP is the newer of the two; the check must therefore ask whether the stamp is at "+
			"least as new as the profile. Without it a STALE profile could be reused, which is worse than "+
			"re-running: it would report a coverage number that describes different code (ISSUE-179)",
			installedGoToolchainPackName, goToolchainCorrectFreshnessTest)
	}
	// ★ THIS READS THE WHOLE FILE, COMMENTS INCLUDED, AND THAT IS DELIBERATE. If this
	// reds on a producer whose LIVE condition is correct, the offender is almost
	// certainly a COMMENT quoting the old expression while explaining the fix. The
	// remedy is to reword that comment in PROSE — "the previous check asked whether the
	// profile was no older than the stamp" — and NEVER to weaken this check.
	if strings.Contains(producer, goToolchainBackwardsFreshnessTest) {
		t.Errorf("installed %s coverage producer still carries the BACKWARDS freshness comparison %q. "+
			"test-produce.sh writes the profile and then touches the stamp, so demanding the profile be "+
			"no-older-than the stamp asks for a state a successful run NEVER produces: the reuse ties by "+
			"coincidence where `-ot` truncates to whole seconds and never fires at all where it reads "+
			"nanoseconds, which is why CI run 32275399064 measured no improvement whatsoever (ISSUE-179). "+
			"If the live condition here is already correct, this is a COMMENT quoting the old expression — "+
			"reword the comment in prose rather than weakening this assertion",
			installedGoToolchainPackName, goToolchainBackwardsFreshnessTest)
	}
}
