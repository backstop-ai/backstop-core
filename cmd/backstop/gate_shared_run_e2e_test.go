package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// gate_shared_run_e2e_test.go proves ISSUE-068 P4: ONE sharedRunCache, constructed
// per gate in buildGateSteps, is threaded into BOTH the pack_engines (writer) and
// coverage (reader) steps, so a run_group-shared suite runs ONCE across the whole
// gate — the cross-step integration the P3 unit tests cannot prove. It drives the
// REAL assembled gate (buildGateSteps) over a fixture project whose test + coverage
// engines are PROJECT-WIDE passes sharing one run_group and one command; the command
// is a real counting script so command executions are observed end-to-end (the gate's
// internal ExecCommandRunner is not injectable, so the count is external).

// rungroupGateFixture returns the committed rungroup-gate fixture project root.
func rungroupGateFixture(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "rungroup-gate")
}

// rungroupGateResult captures the observations one assembled-gate run yields.
type rungroupGateResult struct {
	counterAfterEngines  int
	counterAfterCoverage int
	totalRuns            int
	enginesRes           gate.StepResult
	coverageRes          gate.StepResult
	sawEngines           bool
	sawCoverage          bool
}

// writeExecutable writes an executable script at path.
func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

// countFile returns the byte length of the counter file (0 when absent). The suite
// script appends one byte per execution, so the length is the execution count.
func countFile(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read counter: %v", err)
	}
	return len(b)
}

// rewriteRungroupGatePack rewrites the copied fixture's installed pack.yml to a
// variant: withRunGroup toggles the shared run_group key on both engines;
// withCoverage toggles the presence of the coverage engine + its rule. The baseline
// committed fixture is (withRunGroup=true, withCoverage=true); variants exercise the
// no-key two-run baseline and the coverage-absent WARN path.
func rewriteRungroupGatePack(t *testing.T, projectRoot string, withRunGroup, withCoverage bool) {
	t.Helper()
	rg := ""
	if withRunGroup {
		rg = "\n    run_group: suite"
	}
	var b strings.Builder
	b.WriteString(`name: backstop/rungroup-gate
version: 1.0.0
language: typescript
archetype: enforcement
description: ISSUE-068 cross-step shared run-key fixture variant
engines:
  suite-test:
    command: rungroup-suite
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: test
    convert: scripts/test-convert.sh` + rg + "\n")
	if withCoverage {
		b.WriteString(`  suite-coverage:
    command: rungroup-suite
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: coverage
    convert: scripts/coverage-convert.sh` + rg + "\n")
	}
	b.WriteString(`content:
  ruleset:
    version: 1.0.0
    rules:
      - id: suite-test-rule
        engine: suite-test
        risk_class: correctness
`)
	if withCoverage {
		b.WriteString(`      - id: suite-coverage-rule
        engine: suite-coverage
        risk_class: correctness
`)
	}
	packYML := filepath.Join(projectRoot, ".backstop", "packs", "backstop", "rungroup-gate", "pack.yml")
	if err := os.WriteFile(packYML, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("rewrite variant pack.yml: %v", err)
	}
}

// runAssembledRungroupGate copies the fixture to a temp dir (optionally rewriting the
// pack.yml variant), installs a real counting `rungroup-suite` command on PATH, stubs
// the sandboxed convert to the direct script runner, then drives the REAL assembled
// gate (buildGateSteps) step-by-step — snapshotting the command count after the
// pack_engines and coverage steps.
func runAssembledRungroupGate(t *testing.T, rewrite bool, withRunGroup, withCoverage bool) rungroupGateResult {
	t.Helper()
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — the cross-step e2e must run the REAL cache-aware dispatch")
	}

	projectRoot := t.TempDir()
	copyTree(t, rungroupGateFixture(t), projectRoot)
	if rewrite {
		rewriteRungroupGatePack(t, projectRoot, withRunGroup, withCoverage)
	}

	binDir := t.TempDir()
	counterPath := filepath.Join(t.TempDir(), "counter")
	writeExecutable(t, filepath.Join(binDir, "rungroup-suite"),
		"#!/bin/sh\nprintf 'x' >> \"$RUNGROUP_SUITE_COUNTER\"\nprintf 'SUITE-PAYLOAD\\n'\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RUNGROUP_SUITE_COUNTER", counterPath)

	// The convert runs via the direct script runner (portable stand-in for the real
	// sandbox); the command still executes for real via the gate's ExecCommandRunner.
	orig := sandboxedRunStdout
	sandboxedRunStdout = func(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
		return runConvertScriptDirect(cmd, stdin)
	}
	t.Cleanup(func() { sandboxedRunStdout = orig })

	steps := buildGateSteps(projectRoot, emptyDiffScope())
	res := rungroupGateResult{}
	for _, step := range steps {
		sr := step(context.Background())
		switch sr.StepName {
		case "pack_engines":
			res.enginesRes = sr
			res.sawEngines = true
			res.counterAfterEngines = countFile(t, counterPath)
		case gate.StepCoverageThreshold:
			res.coverageRes = sr
			res.sawCoverage = true
			res.counterAfterCoverage = countFile(t, counterPath)
		}
	}
	res.totalRuns = countFile(t, counterPath)
	if !res.sawEngines {
		t.Fatal("pack_engines step not present in the assembled gate")
	}
	if !res.sawCoverage {
		t.Fatal("coverage_threshold step not present in the assembled gate")
	}
	return res
}

// TestGate_SharedRunKey_SuiteRunsOncePerGate proves the shared suite command executes
// EXACTLY ONCE across the WHOLE assembled gate when the test + coverage engines
// declare the same run_group (ISSUE-068 CLM-004/CLM-007) — the one shared cache
// instance threaded through both steps de-dupes the run.
func TestGate_SharedRunKey_SuiteRunsOncePerGate(t *testing.T) {
	res := runAssembledRungroupGate(t, false, true, true)
	if res.totalRuns != 1 {
		t.Fatalf("shared run_group must run the suite ONCE across the whole gate, got %d executions", res.totalRuns)
	}
	// Neither step regressed to a fail (pure de-dup, clean pass).
	if res.enginesRes.Status == "fail" {
		t.Errorf("pack_engines must pass, got %#v", res.enginesRes.Violations)
	}
	if res.coverageRes.Status == "fail" {
		t.Errorf("coverage must pass, got %#v", res.coverageRes.Violations)
	}
}

// TestGate_SharedRunKey_TestStepWritesCacheCoverageStepReuses proves the WRITER/READER
// ordering explicitly (ISSUE-068 CLM-004/CLM-007): the pack_engines/test step (first
// in gate order) EXECUTES the command (cache writer), and the coverage step REUSES the
// cached payload — zero additional command invocations during coverage. This proves
// the SEAM-routed test step reaches the SAME cache the direct coverage call reads.
func TestGate_SharedRunKey_TestStepWritesCacheCoverageStepReuses(t *testing.T) {
	res := runAssembledRungroupGate(t, false, true, true)
	if res.counterAfterEngines != 1 {
		t.Fatalf("the pack_engines step (writer) must EXECUTE the suite once, got %d after that step", res.counterAfterEngines)
	}
	if res.counterAfterCoverage != res.counterAfterEngines {
		t.Fatalf("the coverage step (reader) must REUSE the cached payload with ZERO extra runs; count went %d -> %d across the coverage step", res.counterAfterEngines, res.counterAfterCoverage)
	}
	// The coverage step still produced its verdict from the reused payload (it ran
	// its own convert, not a second suite execution).
	if res.coverageRes.Status == "fail" {
		t.Errorf("coverage must still verdict from the reused payload, got fail: %#v", res.coverageRes.Violations)
	}
}

// TestGate_SharedRunKey_VerdictsUnchangedVsTwoRun proves verdict INVARIANCE (ISSUE-068
// CLM-007): the gate pass/fail AND per-step results are identical between the
// shared-key single-run gate and the no-key two-run baseline over the same fixture —
// pure de-dup, zero verdict drift. Only the execution COUNT differs (1 vs 2).
func TestGate_SharedRunKey_VerdictsUnchangedVsTwoRun(t *testing.T) {
	shared := runAssembledRungroupGate(t, false, true, true)   // committed: run_group set
	twoRun := runAssembledRungroupGate(t, true, false, true)   // variant: run_group removed

	if shared.totalRuns != 1 {
		t.Fatalf("shared-key gate must run once, got %d", shared.totalRuns)
	}
	if twoRun.totalRuns != 2 {
		t.Fatalf("no-key gate must run twice (unchanged two-run baseline), got %d", twoRun.totalRuns)
	}

	// Verdicts must match byte-for-byte across the two — de-dup changes NOTHING but
	// the redundant run.
	if shared.enginesRes.Status != twoRun.enginesRes.Status {
		t.Errorf("pack_engines status drifted: shared=%q twoRun=%q", shared.enginesRes.Status, twoRun.enginesRes.Status)
	}
	if !reflect.DeepEqual(shared.enginesRes.Violations, twoRun.enginesRes.Violations) {
		t.Errorf("pack_engines violations drifted between shared-key and two-run:\n shared=%#v\n twoRun=%#v", shared.enginesRes.Violations, twoRun.enginesRes.Violations)
	}
	if shared.coverageRes.Status != twoRun.coverageRes.Status {
		t.Errorf("coverage status drifted: shared=%q twoRun=%q", shared.coverageRes.Status, twoRun.coverageRes.Status)
	}
	if !reflect.DeepEqual(shared.coverageRes.Violations, twoRun.coverageRes.Violations) {
		t.Errorf("coverage violations drifted between shared-key and two-run:\n shared=%#v\n twoRun=%#v", shared.coverageRes.Violations, twoRun.coverageRes.Violations)
	}
}

// TestGate_SharedRunKey_CoverageAbsentStillWarns proves A3 (ISSUE-068 CLM-009): a pack
// that ships a test engine but NO coverage engine still emits the graceful
// coverage_capability_absent WARN — the run-group mechanism does not disturb the
// coverage-absent path.
func TestGate_SharedRunKey_CoverageAbsentStillWarns(t *testing.T) {
	res := runAssembledRungroupGate(t, true, false, false) // test engine only, no coverage engine
	if res.coverageRes.Status != "warning" {
		t.Fatalf("coverage-absent must be a graceful WARN, got status %q: %#v", res.coverageRes.Status, res.coverageRes.Violations)
	}
	foundAbsent := false
	for _, v := range res.coverageRes.Violations {
		if strings.Contains(v.Rule, "coverage_capability_absent") {
			foundAbsent = true
		}
	}
	if !foundAbsent {
		t.Errorf("coverage-absent WARN must carry the coverage_capability_absent advisory, got %#v", res.coverageRes.Violations)
	}
}

// TestGate_SharedRunKey_DispatchSeamStillFires proves the WithCache bridge preserves
// the dispatchPackEnginesFn spy point (ISSUE-068): when the seam is set,
// dispatchPackEnginesWithCache delegates to the spy (so every hermetic seam spy still
// compiles AND fires), leaving the concrete cache-aware path for production only.
func TestGate_SharedRunKey_DispatchSeamStillFires(t *testing.T) {
	fired := false
	orig := dispatchPackEnginesFn
	dispatchPackEnginesFn = func(_ []*pack.Manifest, _, _ string, _ *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
		fired = true
		return nil, nil
	}
	t.Cleanup(func() { dispatchPackEnginesFn = orig })

	if _, err := dispatchPackEnginesWithCache(newSharedRunCache(), nil, "", "", nil, nil); err != nil {
		t.Fatalf("WithCache bridge over the spy: %v", err)
	}
	if !fired {
		t.Fatal("the dispatchPackEnginesFn spy must still fire through the WithCache bridge")
	}
}
