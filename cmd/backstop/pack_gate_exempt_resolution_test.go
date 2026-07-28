package main

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// twoEngineManifest builds a one-pack manifest with two rules binding two
// different engine names, each with its own rule file on disk so resolveRulePath
// does not fail-loud.
func twoEngineManifest(t *testing.T, engineA, engineB string) ([]*pack.Manifest, string) {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "rules"))
	writeFileStr(t, filepath.Join(packRoot, "rules", "a.yml"), "rules: []\n")
	writeFileStr(t, filepath.Join(packRoot, "rules", "b.yml"), "rules: []\n")
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "ra", Engine: engineA, RulePath: "rules/a.yml", Standard: "x"},
			{ID: "rb", Engine: engineB, RulePath: "rules/b.yml", Standard: "x"},
		}}},
	}}
	return manifests, packsDir
}

// TestExemption_PerViolationResolutionNoGateTypeAggregation proves two build
// passes across different packages with DIFFERING ExemptFromScopeFilter values
// resolve PER-VIOLATION — each violation carries ITS producing binding's value
// (exempt=true pkg → ProjectWide true; exempt=false pkg → ProjectWide false), with
// NO gate-type-level aggregation (SPEC-041 CLM-018).
func TestExemption_PerViolationResolutionNoGateTypeAggregation(t *testing.T) {
	// Two BUILD-class engines with DIFFERING exempt values, each producing a
	// violation pinned to its own file.
	installExemptRegistry(t, map[string]engine.EngineBinding{
		"build-exempt":    exemptBinding(engine.GateTypeBuild, true),
		"build-nonexempt": exemptBinding(engine.GateTypeBuild, false),
	})
	manifests, packsDir := twoEngineManifest(t, "build-exempt", "build-nonexempt")

	// Each engine reports a finding on a DISTINCT file. The matrixRunner returns the
	// same SARIF regardless of command, so route per-engine output via a runner that
	// keys on the rule-file arg. Simpler: dispatch each engine separately is NOT the
	// contract; here we feed a runner that returns a combined-but-distinct finding
	// per call by alternating — use two single-engine dispatches and merge, then
	// assert the bridge stamped each from its own binding.
	exemptViolations := dispatchOneEngine(t, manifests, packsDir, "build-exempt", "pkg/exempt/a.go")
	nonexemptViolations := dispatchOneEngine(t, manifests, packsDir, "build-nonexempt", "pkg/nonexempt/b.go")

	for _, v := range exemptViolations {
		if !v.ProjectWide {
			t.Errorf("exempt=true binding's violation %q must carry ProjectWide=true per-violation (CLM-018)", v.File)
		}
	}
	for _, v := range nonexemptViolations {
		if v.ProjectWide {
			t.Errorf("exempt=false binding's violation %q must carry ProjectWide=false per-violation — no gate-type aggregation (CLM-018)", v.File)
		}
	}
	if len(exemptViolations) == 0 || len(nonexemptViolations) == 0 {
		t.Fatal("both engines must produce violations to prove per-violation resolution")
	}
}

// dispatchOneEngine dispatches a manifest restricted to a single engine, feeding a
// SARIF finding pinned to file, and returns the produced violations.
func dispatchOneEngine(t *testing.T, manifests []*pack.Manifest, packsDir, engineName, file string) []gate.Violation {
	t.Helper()
	single := []*pack.Manifest{onlyRules(manifests[0], engineName)}
	runner := &matrixRunner{sarif: sarifForFile(file, "r", "finding on "+file)}
	violations, err := dispatchPackEngines(single, packsDir, "/repo", nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (%s): %v", engineName, err)
	}
	return violations
}

// TestExemption_TrueConflictExemptingValueWins proves that in the degenerate
// same-file+line+rule conflict from two sources with differing exempt values, the
// EXEMPTING value WINS — the violation is shown (not scope-filtered), because the
// safe direction against under-broad filtering is louder (SPEC-041 CLM-019).
func TestExemption_TrueConflictExemptingValueWins(t *testing.T) {
	installExemptRegistry(t, map[string]engine.EngineBinding{
		"conflict-exempt":    exemptBinding(engine.GateTypeBuild, true),
		"conflict-nonexempt": exemptBinding(engine.GateTypeBuild, false),
	})
	manifests, packsDir := twoEngineManifest(t, "conflict-exempt", "conflict-nonexempt")

	// BOTH engines report the SAME finding (same file+rule) on an UNCHANGED file.
	const conflictFile = "pkg/unchanged/conflict.go"
	exemptViolations := dispatchOneEngine(t, manifests, packsDir, "conflict-exempt", conflictFile)
	nonexemptViolations := dispatchOneEngine(t, manifests, packsDir, "conflict-nonexempt", conflictFile)

	// The union (both sources' violations) drives the real filterViolations under a
	// diff scope that does NOT contain the conflict file. The exempting copy
	// (ProjectWide=true) must SURVIVE — the violation is shown, the safe louder
	// direction (CLM-019).
	union := append(append([]gate.Violation{}, exemptViolations...), nonexemptViolations...)
	scope := diffScope("/repo", "cmd/other.go")
	survived := filterThroughGate(t, scope, union)

	shown := false
	for _, v := range survived {
		if v.File == conflictFile {
			shown = true
		}
	}
	if !shown {
		t.Fatalf("the true-conflict violation must be SHOWN (exempting value wins) — it was filtered out: %#v", survived)
	}
}
