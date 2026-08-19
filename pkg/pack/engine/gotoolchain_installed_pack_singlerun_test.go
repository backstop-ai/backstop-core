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

// preFixGoToolchainPackVersion is the version that carried the double run. The fix
// ships as a MINOR bump above it: it changes what a shipped engine executes, which is
// a behaviour change in a released rule path, not a patch.
const preFixGoToolchainPackVersion = "1.6.0"

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
	if !strings.Contains(producer, "-ot") {
		t.Errorf("installed %s coverage producer carries no `-ot` freshness comparison — without it a "+
			"STALE profile could be reused, which is worse than re-running: it would report a coverage "+
			"number that describes different code",
			installedGoToolchainPackName)
	}
}
