package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// gate_substantiveness_e2e.go is the cmd/backstop E2E harness the provisioning + REAL
// over-installed-pack tests drive (SPEC-037 REQ-009/REQ-010). It installs the
// packs/substantiveness/ SOURCE as a LOCAL pack via the REAL distribution.Add path into
// a temp workspace, then runs the PRODUCTION substantiveness gate path (the re-wired
// buildTestSubstantivenessStep → real resolveDispatchPackEngines → dispatchPackEngines →
// real ast-grep → real convert-under-sandbox → route + set-join) over a hollow backstop
// *_test.go fixture, returning the produced violations for the tests to assert. It
// resolves the pack from the INSTALLED (local) declaration — NEVER from testdata — and
// adds NO //go:embed and NO baked path. It reuses the Phase-5 wiring + Phase-2 helpers;
// it does NOT re-implement dispatch.

// substantivenessSourceDir returns the in-repo installable pack SOURCE at
// packs/substantiveness/ (location B), relative to the repo root.
func substantivenessSourceDir(repoRoot string) string {
	return filepath.Join(repoRoot, "packs", "substantiveness")
}

// e2eWorkspace is a temp workspace with a backstop.yml + specs/ + a hollow mandated
// test file, into which the substantiveness pack is installed as a local pack.
type e2eWorkspace struct {
	root        string
	specDir     string
	hollowFile  string
	installed   bool
	installInfo *distribution.AddResult
}

// newE2EWorkspace scaffolds a temp workspace: a minimal backstop.yml, a spec mandating
// a hollow test, and the hollow *_test.go source. It does NOT install the pack — the
// caller decides (so the negative twin can run the SAME workspace uninstalled).
func newE2EWorkspace(tmp string) (*e2eWorkspace, error) {
	specDir := filepath.Join(tmp, "specs")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating e2e spec dir: %w", err)
	}
	// Minimal backstop.yml (no packs yet). distribution.Add appends the
	// substantiveness pack to this when the workspace is installed. SPEC-046: no
	// `language:` key — a project is described by its declared packs.
	ymlContent := "project: e2e\npacks: {}\n"
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte(ymlContent), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e backstop.yml: %w", err)
	}
	// A spec mandating a hollow test in pkg/gate (target package "gate").
	spec := `---
title: "E2E Sub Spec"
number: E2E-001
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
      - TestE2EHollowSubject
---

# E2E Sub Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "e2e.spec.md"), []byte(spec), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e spec: %w", err)
	}
	// A genuinely HOLLOW backstop *_test.go source: calls a subject, asserts nothing.
	hollowFile := filepath.Join(tmp, "subject_test.go")
	hollow := "package sample_test\n\nimport \"testing\"\n\n" +
		"func TestE2EHollowSubject(t *testing.T) {\n\tdoSubject()\n}\n"
	if err := os.WriteFile(hollowFile, []byte(hollow), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e hollow fixture: %w", err)
	}
	return &e2eWorkspace{root: tmp, specDir: specDir, hollowFile: hollowFile}, nil
}

// installSubstantivenessLocalPack installs the packs/substantiveness/ SOURCE as a LOCAL
// pack via the REAL pack add path (declared `backstop/substantiveness: local` in
// backstop.yml + a `local` lockfile entry) — the install path itself is REAL, not mocked.
//
// The command comes from the PRODUCTION assembly (SPEC-055 REQ-013), so pack check and
// pack test run over the source exactly as they do for a consumer. Assembling here is
// legitimate because cmd/backstop IS the assembly layer; the same construction inside
// pkg/pack/distribution would be the internal defaulting that makes a test double
// indistinguishable from production wiring.
//
// Its receiver and signature are pinned: the four call sites in
// gate_substantiveness_e2e_test.go depend on this exact shape.
func (w *e2eWorkspace) installSubstantivenessLocalPack(repoRoot string) error {
	add, err := newProductionAddCommand()
	if err != nil {
		return fmt.Errorf("assembling the pack add command: %w", err)
	}

	res, err := add.Run(substantivenessSourceDir(repoRoot), distribution.AddOptions{
		ProjectDir: w.root,
	})
	if err != nil {
		return fmt.Errorf("installing substantiveness local pack: %w", err)
	}
	w.installed = true
	w.installInfo = res
	return nil
}

// runProductionSubstantivenessStep runs the PRODUCTION substantiveness gate step over
// the workspace through the REAL dispatch path: buildTestSubstantivenessStep resolves
// the pack from the INSTALLED (local) declaration via loadInstalledPacks, dispatches
// through the real resolveDispatchPackEngines (NOT the seam stub) → real ast-grep → real
// convert-under-sandbox → route + set-join. It returns the step result so the test can
// assert a real test_substantiveness violation (or its absence when uninstalled).
//
// NOTE: this deliberately does NOT install a dispatch seam stub and does NOT override
// resolveSubstantivenessPacksFn — the pack is resolved from the real installed
// declaration and dispatched through the real engine, so the proof is unstubbable.
func (w *e2eWorkspace) runProductionSubstantivenessStep() gate.StepResult {
	// The workspace is a Go project; build the pack-shaped Go classifier + matcher the
	// production substantiveness step now consumes to resolve mandated test file paths
	// (mirrors the go-toolchain pack DATA — this Go self-toolchain harness is not on
	// the language-neutral gate spine).
	classifier := gate.NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	matcher, err := gate.NewTestNameMatcher([]string{`^\s*func\s+(Test\w+)\s*\(`})
	if err != nil {
		return gate.StepResult{StepName: gate.StepTestSubstantiveness, Status: "fail", Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "compiling go test-name pattern: " + err.Error(), Severity: "error"}}}
	}
	step := buildTestSubstantivenessStep(w.specDir, w.root, w.root, nil, classifier, matcher)
	return step(context.Background())
}
