package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// contract_equivalence_test.go (SPEC-038 TASK-018, REQ-008): strangler-equivalence.
// The pack path (real ast-grep + real grep -> ContractEngineResult ->
// VerifyContractVerdict) reproduces the LIVE go/parser analyzer's verdicts on real Go
// fixtures across present / mismatch / absent-present / absent-clean / missing. This
// is the GATE that licenses the Phase-6 deletion: replacing the engine with a stub
// would diverge from the analyzer and FAIL these tests.

func eqRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("module root not found")
	return ""
}

func requireEngines(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"ast-grep", "grep"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("real %s is required for strangler-equivalence (no t.Skip): %v", tool, err)
		}
	}
}

func eqTd(root, name string) string {
	return filepath.Join(root, "pkg", "gate", "testdata", name)
}

// assertEquivalent runs both paths for one entry and asserts the verdict bit matches.
func assertEquivalent(t *testing.T, root string, entry ContractEntry, wantRaised bool) {
	t.Helper()
	packRaised, err := PackVerdict(root, entry)
	if err != nil {
		t.Fatalf("pack verdict: %v", err)
	}
	analyzerRaised := AnalyzerVerdict(entry)
	if packRaised != analyzerRaised {
		t.Fatalf("pack verdict (%v) != analyzer verdict (%v) — equivalence broken", packRaised, analyzerRaised)
	}
	if packRaised != wantRaised {
		t.Fatalf("verdict = %v, want %v (both paths agreed but the expected verdict is wrong)", packRaised, wantRaised)
	}
}

// TestEquivalence_GoSignaturePresentMatchesLegacy: present matching signature ->
// pack SATISFIED == legacy match (no violation) (CLM-027).
func TestEquivalence_GoSignaturePresentMatchesLegacy(t *testing.T) {
	requireEngines(t)
	root := eqRepoRoot(t)
	entry := ContractEntry{
		File:      eqTd(root, "contract-sig-present.go"),
		Name:      "RouteFile",
		Kind:      "function",
		Signature: "func RouteFile(path string, mode int) (string, error)",
	}
	assertEquivalent(t, root, entry, false)
}

// TestEquivalence_GoSignatureMismatchMatchesLegacy: mismatched/missing signature ->
// pack VIOLATION == legacy mismatch/not-found (CLM-028).
func TestEquivalence_GoSignatureMismatchMatchesLegacy(t *testing.T) {
	requireEngines(t)
	root := eqRepoRoot(t)
	entry := ContractEntry{
		File:      eqTd(root, "contract-sig-mismatch.go"),
		Name:      "RouteFile",
		Kind:      "function",
		Signature: "func RouteFile(path string, mode int) (string, error)",
	}
	assertEquivalent(t, root, entry, true)
}

// TestEquivalence_GoAbsencePresentAndAbsentMatchLegacy: absence present->violation and
// absent->pass equal the legacy probeSymbol verdicts (CLM-029).
func TestEquivalence_GoAbsencePresentAndAbsentMatchLegacy(t *testing.T) {
	requireEngines(t)
	root := eqRepoRoot(t)

	present := ContractEntry{
		File:   eqTd(root, "contract-absence-present.go"),
		Name:   "legacyProbeSymbol",
		Kind:   "function",
		Absent: true,
		Scope:  eqTd(root, "contract-absence-present.go"),
	}
	assertEquivalent(t, root, present, true)

	clean := ContractEntry{
		File:   eqTd(root, "contract-absence-clean.go"),
		Name:   "legacyProbeSymbol",
		Kind:   "function",
		Absent: true,
		Scope:  eqTd(root, "contract-absence-clean.go"),
	}
	assertEquivalent(t, root, clean, false)
}

// TestEquivalence_GoAbsenceMissingFileMatchesLegacyLoudError: a missing/non-target
// Go file -> pack path + file-scanned guard reproduces the analyzer's loud config
// error (a raised violation, not a silent pass), matching ISSUE-013 (CLM-030).
func TestEquivalence_GoAbsenceMissingFileMatchesLegacyLoudError(t *testing.T) {
	requireEngines(t)
	root := eqRepoRoot(t)
	missing := ContractEntry{
		File:   eqTd(root, "does-not-exist.go"),
		Name:   "legacyProbeSymbol",
		Kind:   "function",
		Absent: true,
		Scope:  eqTd(root, "does-not-exist.go"),
	}
	assertEquivalent(t, root, missing, true)
}
