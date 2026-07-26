package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// gate_contract_e2e.go (SPEC-038 REQ-014/Phase 7c) is the cmd/backstop E2E harness the
// REAL over-installed-pack tests drive. It installs the packs/contracts/ SOURCE as a
// LOCAL pack via the REAL distribution.Add path into a temp workspace, then runs the
// PRODUCTION contract gate path (the Phase-6-rewired buildContractStep → produce-
// ContractEngineResults → gate.PackContractResult resolving the INSTALLED pack scripts
// → real ast-grep [signature] + real grep [absence] → real convert-under-sandbox → SARIF
// → match-verdict + absence polarity + file-scanned guard) over real fixtures, returning
// the produced violations. It resolves the pack from the INSTALLED (local) declaration —
// NEVER from a stub, NEVER from production-pointed-at-testdata — and adds NO //go:embed
// and NO baked path. It reuses the Phase-6 wiring + Phase-3 verdict + Phase-7a install
// helper; it does NOT re-implement dispatch.

// contractE2EWorkspace is a temp workspace with a backstop.yml + specs/ + real Go
// fixtures (a missing/mismatched signature, a present forbidden symbol), into which the
// contracts pack is installed as a local pack.
type contractE2EWorkspace struct {
	root      string
	specDir   string
	installed bool
}

// newContractE2EWorkspace scaffolds a temp workspace: a minimal backstop.yml, the real Go
// fixtures, and a spec declaring BOTH a signature contract (over the mismatch fixture, so
// the present-signature is ABSENT → ast-grep no-match → violation) and an absence contract
// (over the present-forbidden-symbol fixture → grep match → absence violation). It does
// NOT install the pack — the caller decides (so the negative twin can run uninstalled).
func newContractE2EWorkspace(tmp string) (*contractE2EWorkspace, error) {
	specDir := filepath.Join(tmp, "specs")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating e2e spec dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte("project: e2e\nlanguage: go\npacks: {}\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e backstop.yml: %w", err)
	}

	// Real Go fixtures in the workspace.
	mismatch := "package sample\n\nfunc somethingElse() {}\n"
	if err := os.WriteFile(filepath.Join(tmp, "sig_mismatch.go"), []byte(mismatch), 0o644); err != nil {
		return nil, fmt.Errorf("writing mismatch fixture: %w", err)
	}
	present := "package sample\n\nfunc legacyProbeSymbol() string { return \"present\" }\n"
	if err := os.WriteFile(filepath.Join(tmp, "absence_present.go"), []byte(present), 0o644); err != nil {
		return nil, fmt.Errorf("writing absence-present fixture: %w", err)
	}

	// A spec declaring the two contracts. The signature contract declares
	// `func RouteFile(...)` which is ABSENT from sig_mismatch.go → ast-grep no-match →
	// VIOLATION. The absence contract forbids `legacyProbeSymbol` in absence_present.go
	// → grep match → absence VIOLATION.
	spec := `---
title: "Contract E2E Spec"
number: E2E-CON-001
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: e2e
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
      - TestContractE2E

contracts:
  - file: sig_mismatch.go
    provides:
      - name: RouteFile
        kind: function
        signature: "func RouteFile(path string, mode int) (string, error)"
  - file: absence_present.go
    provides:
      - name: legacyProbeSymbol
        kind: function
        absent: true
        scope: absence_present.go
---

# Contract E2E Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "e2e.spec.md"), []byte(spec), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e spec: %w", err)
	}
	return &contractE2EWorkspace{root: tmp, specDir: specDir}, nil
}

// installContractsLocalPack installs the packs/contracts/ SOURCE as a LOCAL pack into the
// workspace via the REAL distribution.Add path (declared `backstop/contracts: local` +
// a `local` lockfile entry). The installed pack's scripts land at
// <root>/.backstop/packs/backstop/contracts/, which the production contract path resolves.
func (w *contractE2EWorkspace) installContractsLocalPack(repoRoot string) error {
	add, err := newProductionAddCommand()
	if err != nil {
		return fmt.Errorf("assembling the production add command: %w", err)
	}
	if _, err := distribution.InstallContractsLocalPack(add, repoRoot, w.root); err != nil {
		return err
	}
	w.installed = true
	return nil
}

// runProductionContractStep runs the PRODUCTION contract gate step over the workspace
// through the REAL dispatch path: buildContractStep extracts the contract entries and
// routes them through produceContractEngineResults → dispatchContractEntry →
// resolveDispatchPackEngines, which
// resolves the INSTALLED pack scripts under <root>/.backstop/packs/backstop/contracts/
// (or, when uninstalled, finds no resolvable pack and produces no results). It returns
// the step result so the test can assert real violations (or their absence).
//
// NOTE: this deliberately installs NO seam stub (contractEngineResultsFn stays nil) —
// the pack is resolved from the real installed declaration and dispatched through the
// real engines, so the proof is unstubbable.
func (w *contractE2EWorkspace) runProductionContractStep() gate.StepResult {
	step := buildContractStep(w.specDir, w.root, nil)
	return step(context.Background())
}
