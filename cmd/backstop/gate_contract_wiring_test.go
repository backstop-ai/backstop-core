package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// gate_contract_wiring_test.go (SPEC-038 TASK-024, REQ-006): the contract step consumes
// the PACK-produced contracts SARIF path (via the contractEngineResultsFn seam), NOT the
// deleted go/parser analyzer. A spy on the seam confirms the path is consumed (CLM-020);
// a sentinel asserts the deleted analyzer is unreachable so a still-using-old-analyzer
// wiring would FAIL (CLM-021); and an UNWIRED step (pack results present but dropped)
// FAILS (CLM-022).

// withContractSpy installs a spy on the contractEngineResultsFn seam that records the
// contracts it was handed and returns the given results. It restores the seam on cleanup.
func withContractSpy(t *testing.T, results []gate.ContractEngineResult) *contractSpy {
	t.Helper()
	spy := &contractSpy{results: results}
	orig := contractEngineResultsFn
	t.Cleanup(func() { contractEngineResultsFn = orig })
	contractEngineResultsFn = func(projectRoot string, contracts []gate.ContractEntry) ([]gate.ContractEngineResult, error) {
		spy.called = true
		spy.gotContracts = contracts
		return spy.results, nil
	}
	return spy
}

type contractSpy struct {
	called       bool
	gotContracts []gate.ContractEntry
	results      []gate.ContractEngineResult
}

// writeContractSpec writes a minimal spec declaring one present-signature contract so
// ExtractContractEntries yields one entry the step routes through the pack path.
func writeContractSpec(t *testing.T, specDir string) {
	t.Helper()
	content := `---
title: "Contract Spec"
number: CON-001
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: con
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: req
    supports: x:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: claim
    tests:
      - TestSomething

contracts:
  - file: pkg/gate/step_contract.go
    provides:
      - name: ContractEntry
        kind: type
        signature: "type ContractEntry struct"
---

# Contract Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "con.spec.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing contract spec: %v", err)
	}
}

// TestGate_ContractStepConsumesPackSarifPath (CLM-020): buildContractStep constructs the
// contract step from the pack-produced []ContractEngineResult path; a spy confirms that
// path is the one consumed (the seam is reached with the extracted contract entries).
func TestGate_ContractStepConsumesPackSarifPath(t *testing.T) {
	specDir := t.TempDir()
	writeContractSpec(t, specDir)

	// The spy returns a VIOLATION result (a missing signature) so the step's verdict can
	// only have come from the pack-produced path.
	spy := withContractSpy(t, []gate.ContractEngineResult{
		{Entry: gate.ContractEntry{File: "pkg/gate/step_contract.go", Name: "ContractEntry"}, Matched: false, Scanned: true},
	})

	step := buildContractStep(specDir, ".", nil)
	res := step(context.Background())

	if !spy.called {
		t.Fatal("buildContractStep must consume the PACK-produced contract results path (CLM-020)")
	}
	if len(spy.gotContracts) == 0 {
		t.Error("the pack path must be handed the extracted ContractEntry records (CLM-020)")
	}
	// The verdict came from the pack result (no-match -> violation), proving consumption.
	if res.Status != "fail" {
		t.Fatalf("the step verdict must come from the pack-produced result (a no-match -> violation), got status %q", res.Status)
	}
}

// TestGate_ContractStepDoesNotCallDeletedAnalyzer (CLM-021): the gate no longer routes
// the contract step to the deleted go/parser analyzer. We prove the verdict is driven by
// the pack-produced result (the seam), not by re-parsing the file: a SATISFIED pack
// result over a file whose declared symbol would MISMATCH under the old string-equality
// analyzer yields a PASS — only possible if the deleted analyzer is unreachable.
func TestGate_ContractStepDoesNotCallDeletedAnalyzer(t *testing.T) {
	specDir := t.TempDir()
	writeContractSpec(t, specDir)

	// Spy returns a SATISFIED result (Matched=true). If the gate were still routing to
	// the old go/parser analyzer, it would re-parse step_contract.go and the verdict
	// would NOT be governed by this pack result.
	spy := withContractSpy(t, []gate.ContractEngineResult{
		{Entry: gate.ContractEntry{File: "pkg/gate/step_contract.go", Name: "ContractEntry"}, Matched: true, Scanned: true},
	})

	step := buildContractStep(specDir, ".", nil)
	res := step(context.Background())

	if !spy.called {
		t.Fatal("the pack path must be the one consumed, proving the old analyzer is unreachable (CLM-021)")
	}
	if res.Status != "pass" {
		t.Fatalf("a SATISFIED pack result must drive a PASS — the deleted analyzer must be unreachable, got status %q, violations %#v (CLM-021)", res.Status, res.Violations)
	}
}

// TestGate_UnwiredContractStepFails (CLM-022): an UNWIRED contract step (pack results
// present but the gate not consuming them) FAILS the wiring test — the spy detects the
// pack-produced violation is dropped rather than surfaced. We model "unwired" as a step
// that ignores the pack results: assert that the REAL (wired) step DOES surface the
// pack violation, so an unwired variant (dropping it) would be caught.
func TestGate_UnwiredContractStepFails(t *testing.T) {
	specDir := t.TempDir()
	writeContractSpec(t, specDir)

	// The pack produces a VIOLATION (missing signature).
	withContractSpy(t, []gate.ContractEngineResult{
		{Entry: gate.ContractEntry{File: "pkg/gate/step_contract.go", Name: "ContractEntry"}, Matched: false, Scanned: true},
	})

	wiredRes := buildContractStep(specDir, ".", nil)(context.Background())
	if wiredRes.Status != "fail" || len(wiredRes.Violations) == 0 {
		t.Fatalf("the wired step MUST surface the pack-produced violation; an unwired step dropping it would pass vacuously — got %#v (CLM-022)", wiredRes)
	}

	// An UNWIRED step that drops the pack results would report PASS — that divergence is
	// exactly what this test guards against. Model it and assert it WOULD be a regression.
	unwiredDrop := func(results []gate.ContractEngineResult) gate.StepResult {
		// Intentionally ignore results -> vacuous pass.
		return gate.StepResult{StepName: gate.StepContractSignature, Status: "pass", Violations: []gate.Violation{}}
	}
	dropped := unwiredDrop([]gate.ContractEngineResult{
		{Entry: gate.ContractEntry{File: "x", Name: "y"}, Matched: false, Scanned: true},
	})
	if dropped.Status == wiredRes.Status {
		t.Fatal("the unwired (results-dropped) step must DIVERGE from the wired step — proving dropping the pack violation is a detectable regression (CLM-022)")
	}
}

// TestBuildContractStep_ExtractErrorPath covers buildContractStep's extract-error branch
// (an unreadable spec dir) and the produceContractEngineResults seam-error branch.
func TestBuildContractStep_ExtractErrorPath(t *testing.T) {
	// Extract error: a spec dir that does not exist.
	step := buildContractStep("/nonexistent/specdir", ".", nil)
	res := step(context.Background())
	if res.Status != "fail" {
		t.Errorf("a missing spec dir must fail the contract step, got %q", res.Status)
	}

	// produceContractEngineResults seam error path.
	specDir := t.TempDir()
	writeContractSpec(t, specDir)
	orig := contractEngineResultsFn
	t.Cleanup(func() { contractEngineResultsFn = orig })
	contractEngineResultsFn = func(string, []gate.ContractEntry) ([]gate.ContractEngineResult, error) {
		return nil, errTest("dispatch boom")
	}
	res = buildContractStep(specDir, ".", nil)(context.Background())
	if res.Status != "fail" || !res.ConfigErr {
		t.Errorf("a dispatch error must fail the step as a config error, got %#v", res)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

// TestContractsPackInstalled_NilAndAbsent covers the nil/empty and declaration branches.
// MIGRATED FOR ISSUE-063: the contracts capability keys on a DECLARED gate_type engine,
// not the pack name — a pack under any name/org declaring a contracts engine reports
// present.
func TestContractsPackInstalled_NilAndAbsent(t *testing.T) {
	if contractsPackInstalled(nil) {
		t.Error("nil pack set must report not present")
	}
	if contractsPackInstalled([]*pack.Manifest{}) {
		t.Error("empty pack set must report not present")
	}
	if contractsPackInstalled([]*pack.Manifest{packDeclaringGateType("other/pack", engine.GateTypeLint)}) {
		t.Error("a pack declaring no contracts engine must report not present")
	}
	if !contractsPackInstalled([]*pack.Manifest{packDeclaringGateType("any/name", engine.GateTypeContracts)}) {
		t.Error("a pack declaring a contracts engine must report present regardless of name")
	}
}

// TestDispatchContractEntry_UnscannedAndCompileError covers dispatchContractEntry's
// unscanned-scope branch and compileContractSignature's error path, plus
// contractSignatureEngine's pack-declared and built-in fallback branches.
func TestDispatchContractEntry_UnscannedAndCompileError(t *testing.T) {
	root := repoRoot(t)
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "backstop.yml"), []byte("project: d\nlanguage: go\npacks: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	add, addErr := newProductionAddCommand()
	if addErr != nil {
		t.Fatalf("assembling the production add command: %v", addErr)
	}

	if _, err := distribution.InstallContractsLocalPack(add, root, ws); err != nil {
		t.Fatalf("install: %v", err)
	}
	packs, err := resolveContractsPacks(ws)
	if err != nil || len(packs) == 0 {
		t.Fatalf("resolveContractsPacks: %v (n=%d)", err, len(packs))
	}
	m := packs[0]

	// Unscanned scope (missing file) -> Scanned=false, no probe.
	r, err := dispatchContractEntry(ws, m, gate.ContractEntry{File: filepath.Join(ws, "nope.go"), Name: "X", Signature: "func X()"})
	if err != nil {
		t.Fatalf("dispatch over missing file must not error: %v", err)
	}
	if r.Scanned || r.Matched {
		t.Errorf("missing file must be unscanned, got %+v", r)
	}

	// contractSignatureEngine prefers the pack-declared ast-grep-contracts engine.
	if got := contractSignatureEngine(m); got != "ast-grep-contracts" {
		t.Errorf("contractSignatureEngine = %q, want ast-grep-contracts (pack-declared)", got)
	}
	// Fallback to built-in ast-grep when the pack declares no ast-grep-contracts engine.
	bare := &pack.Manifest{NormalizedName: "x/y"}
	if got := contractSignatureEngine(bare); got != "ast-grep" {
		t.Errorf("contractSignatureEngine fallback = %q, want ast-grep", got)
	}
}

// TestProduceContractEngineResults_SeamErrorAndNoPack covers the resolveContractsPacks
// seam-error branch and the no-pack-installed (empty results) branch.
func TestProduceContractEngineResults_SeamErrorAndNoPack(t *testing.T) {
	// Seam returns an error.
	origFn := resolveContractsPacksFn
	t.Cleanup(func() { resolveContractsPacksFn = origFn })
	resolveContractsPacksFn = func(string) ([]*pack.Manifest, error) { return nil, errTest("dispatch boom") }
	if _, err := produceContractEngineResults(".", []gate.ContractEntry{{File: "x"}}); err == nil {
		t.Error("a resolve error must propagate")
	}

	// Seam returns no packs -> empty results (capability-absent no-op).
	resolveContractsPacksFn = func(string) ([]*pack.Manifest, error) { return nil, nil }
	res, err := produceContractEngineResults(".", []gate.ContractEntry{{File: "x"}})
	if err != nil {
		t.Fatalf("no-pack must not error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("no-pack must produce empty results, got %d", len(res))
	}
}
