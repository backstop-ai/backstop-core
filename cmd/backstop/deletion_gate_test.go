package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nonTestGoSources returns every non-test .go file path in a package directory.
func nonTestGoSources(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	return out
}

// grepNonTestSource reports whether needle appears in any non-test .go source in
// dir — the source-scan invariant the deletion-gate guard and the deletion
// assertions share.
func grepNonTestSource(t *testing.T, dir, needle string) bool {
	t.Helper()
	for _, p := range nonTestGoSources(t, dir) {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		if strings.Contains(string(b), needle) {
			return true
		}
	}
	return false
}

// TestGoldenEquivalence_DeletionGatedOnProvenEquivalence is the deletion-gate
// guard (CLM-012, Sharp Edge 5), phrased as a ONE-TREE invariant: the
// golden-equivalence test exists in this tree AND the legacy Step-2 symbols
// (realCodeChecker as a wired gate step, builtinToolchain) are ABSENT from
// non-test source — so bespoke-absent implies equivalence-present. While the
// bespoke symbols still exist (pre-Phase-6) this guard is RED; it goes green only
// after the Phase-6 deletions land, so a deletion that outran the proof fails the
// gate rather than shipping green.
func TestGoldenEquivalence_DeletionGatedOnProvenEquivalence(t *testing.T) {
	// The golden-equivalence proof must EXIST in this tree (the safety net the
	// deletion is gated on).
	goldenTest := filepath.Join(repoRoot(t), "cmd", "backstop", "golden_equivalence_test.go")
	src, err := os.ReadFile(goldenTest)
	if err != nil {
		t.Fatalf("the golden-equivalence test must exist before any deletion: %v", err)
	}
	for _, mandated := range []string{
		"TestGoldenEquivalence_LegacyViolationSetCaptured",
		"TestGoldenEquivalence_PackEnginePathReproducesGoldenSet",
		"TestGoldenEquivalence_RealInstalledPackThroughUnstubbedDispatch",
	} {
		if !strings.Contains(string(src), mandated) {
			t.Fatalf("golden-equivalence proof is missing mandated test %s — the deletion is not gated on a complete proof", mandated)
		}
	}

	// The legacy Step-2 symbols must be ABSENT from non-test source (the deletion
	// has landed). RED until Phase 6 deletes them.
	cmdDir := filepath.Join(repoRoot(t), "cmd", "backstop")
	checkDir := filepath.Join(repoRoot(t), "pkg", "check")
	if grepNonTestSource(t, cmdDir, "realCodeChecker") {
		t.Error("realCodeChecker still present in cmd/backstop non-test source — a deletion that outran the golden proof, or the deletion has not yet landed (RED until Phase 6)")
	}
	if grepNonTestSource(t, checkDir, "builtinToolchain") {
		t.Error("builtinToolchain still present in pkg/check non-test source — the baked go/ts stacks must be deleted (RED until Phase 6)")
	}
}
