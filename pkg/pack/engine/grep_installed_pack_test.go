package engine_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// grep_installed_pack_test.go (PLAN-ISSUE-166, CLM-006).
//
// WHY THIS FILE EXISTS AT ALL. The pack that runs in backstop-core's REAL gate is
// the INSTALLED `backstop-ai/go-contracts`, not the in-repo `packs/contracts`
// source — they have already diverged on both name and version. Core's spec corpus
// declares real absence contracts (`absent: true` on a `contracts.provides` entry),
// and an absence scope may be a single FILE, which is exactly the shape GNU grep
// strips the filename from. So until the RELEASED pack carries both halves of the
// fix, core's own contracts gate silently confirms absence on Linux regardless of
// what the file contains. Fixing only the in-repo copies would leave the production
// consumer broken.
//
// It is in `engine_test` for the same reason grep_dispatch_convention_test.go is:
// it resolves the pack through the PRODUCTION readers (`pack.ParseManifestFile`,
// `distribution.ReadLockfile`), and `pkg/pack` imports `pkg/pack/engine`, so an
// in-package test importing them is an import cycle.

// installedContractsPackName is the pack that actually runs — the INSTALLED, LOCKED
// pack the gate reads, NOT the tracked <repoRoot>/packs/contracts fixture, which is
// under no lock entry and has already diverged on name and version. The constant
// follows the established convention in cmd/backstop/pack_claim_index_test.go.
const installedContractsPackName = "backstop-ai/go-contracts"

// preFixContractsPackVersion is the version that carried the defect. The fix ships
// as a MINOR bump above it: it is a behavior change in a shipped rule path, not a
// patch.
const preFixContractsPackVersion = "1.2.0"

// semverAtLeastGreater reports whether version a is strictly greater than b, over
// dotted numeric components. Both are pack versions, which are strict X.Y.Z.
func semverGreater(t *testing.T, a, b string) bool {
	t.Helper()
	parse := func(v string) [3]int {
		var out [3]int
		parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
		for i := 0; i < 3 && i < len(parts); i++ {
			n, err := strconv.Atoi(parts[i])
			if err != nil {
				t.Fatalf("version %q is not strict X.Y.Z: %v", v, err)
			}
			out[i] = n
		}
		return out
	}
	av, bv := parse(a), parse(b)
	for i := 0; i < 3; i++ {
		if av[i] != bv[i] {
			return av[i] > bv[i]
		}
	}
	return false
}

// TestInstalledGoContractsPack_CarriesFilenameHeaderFix (CLM-006) asserts the
// RELEASED pack carries both halves of the fix and that core actually consumes it.
//
// Three legs, deliberately split by what each can prove where:
//   - THE LOCK leg reads TRACKED data, so it holds even in a fresh checkout where
//     the pack fleet is not installed. It is what keeps this test from being
//     vacuous when the content legs cannot run.
//   - THE DECLARATION and CONVERT legs read the INSTALLED tree, so they follow the
//     established guard: a load ERROR fails, a genuinely-absent fleet skips with a
//     `pack install` directive.
func TestInstalledGoContractsPack_CarriesFilenameHeaderFix(t *testing.T) {
	root := conventionRepoRoot(t)

	// ── LEG 1: THE LOCK (tracked data — never skipped) ──────────────────────────
	lockPath := filepath.Join(root, "backstop.lock")
	lockfile, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		// A tracked lockfile that will not read is a broken repo, never a skip.
		t.Fatalf("ReadLockfile(%s): %v", lockPath, err)
	}
	entry, ok := lockfile.Packs[installedContractsPackName]
	if !ok {
		t.Fatalf("backstop.lock has no entry for %s; core's contracts gate rides that pack",
			installedContractsPackName)
	}
	if !semverGreater(t, entry.Version, preFixContractsPackVersion) {
		t.Errorf("backstop.lock pins %s at %s, which is not greater than the pre-fix %s — "+
			"the released pack still carries the silent-drop convert and the flag-less grep "+
			"declaration, so core's own contracts gate is still vacuous on Linux",
			installedContractsPackName, entry.Version, preFixContractsPackVersion)
	}
	gitRef := "<none>"
	if entry.GitRef != nil {
		gitRef = *entry.GitRef
	}
	t.Logf("backstop.lock: %s version=%s git_ref=%s source_type=%s",
		installedContractsPackName, entry.Version, gitRef, entry.SourceType)

	// ── The installed tree, behind the established guard ────────────────────────
	packRoot := filepath.Join(root, ".backstop", "packs", installedContractsPackName)
	manifestPath := filepath.Join(packRoot, "pack.yml")
	manifest, err := pack.ParseManifestFile(manifestPath)
	if err != nil {
		// Distinguish "not installed" from "installed and broken". Only the former
		// skips; a manifest that exists and will not parse is a real failure.
		if _, statErr := os.Stat(manifestPath); statErr != nil {
			t.Skipf("%s is not installed — run `./bin/backstop pack install` (the pack fleet is not installed)",
				installedContractsPackName)
		}
		t.Fatalf("ParseManifestFile(%s): %v", manifestPath, err)
	}

	// ── LEG 2: THE DECLARATION carries both flags ───────────────────────────────
	spec, ok := manifest.Engines["grep"]
	if !ok {
		t.Fatalf("installed %s declares no grep engine; the absence probe rides it",
			installedContractsPackName)
	}
	t.Logf("installed %s v%s grep command: %q", manifest.Name, manifest.Version, spec.Command)
	for _, flag := range []string{"-H", "-I"} {
		found := false
		for _, f := range strings.Fields(spec.Command) {
			if f == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("installed %s grep command %q is missing %s — this is the pack REAL "+
				"CONSUMERS install, and without both flags its convert is fed a shape it "+
				"cannot parse", installedContractsPackName, spec.Command, flag)
		}
	}
	if spec.Convert == "" {
		t.Fatalf("installed %s grep binding declares no convert; the flags exist to protect it",
			installedContractsPackName)
	}

	// ── LEG 3: THE CONVERT refuses the verbatim GNU bytes ───────────────────────
	// Same constant and same runner the in-repo copies are driven by — a second
	// spelling of the fixture bytes is exactly the drift this lane exists to remove.
	convertPath := filepath.Join(packRoot, filepath.FromSlash(spec.Convert))
	stdout, stderr, code := engine.RunGrepConvert(t, convertPath, engine.GNUTwoFieldGrepOutput)
	if code == 0 {
		t.Errorf("installed %s convert exited 0 on the verbatim 2-field GNU bytes — the "+
			"released pack still drops real matches silently.\nstdout: %s", installedContractsPackName, stdout)
	}
	if !strings.Contains(stderr, "REFUSING") {
		t.Errorf("installed %s convert must refuse LOUDLY; stderr was: %s",
			installedContractsPackName, stderr)
	}
	if !strings.Contains(stderr, "-H -I") {
		t.Errorf("installed %s convert's diagnostic must name the -H -I remedy — its "+
			"audience is a pack author in another repo whose only clue is this message; "+
			"stderr was: %s", installedContractsPackName, stderr)
	}
}
