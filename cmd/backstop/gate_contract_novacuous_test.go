package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// gate_contract_novacuous_test.go (SPEC-038 TASK-027, REQ-009): the deletion did NOT
// trade enforcement for silence. Each enforcement case produces a REAL blocking violation
// through the rewired pack path (CLM-031/032), and backstop's OWN previously-red
// contract_signature case resolves to GREEN under the new pack path (CLM-033 — the
// dual-substrate dogfood payoff, the brittle signaturesMatch/formatFuncSignature
// round-trip dissolved). These run through the WIRED gate, not the unit verdict in
// isolation.

func requireNoVacEngines(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"ast-grep", "grep"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("%s required (no t.Skip): %v", tool, err)
		}
	}
}

// novacWorkspace builds a temp workspace with the contracts pack installed, COPIES the
// named dogfood fixture into the workspace (contracts reference files in the project being
// gated, so the file must live under projectRoot for the real dispatch to scan it), and a
// spec declaring the given contracts (as raw frontmatter contract entries, with the file
// path = the in-workspace copy). It then runs the production contract step.
func novacWorkspace(t *testing.T, fixtureName, contractsYAMLTemplate string) gate.StepResult {
	t.Helper()
	root := repoRoot(t)
	tmp := t.TempDir()
	specDir := filepath.Join(tmp, "specs")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte("project: nv\nlanguage: go\npacks: {}\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}

	// Copy the dogfood fixture INTO the workspace so the real dispatch can scan it.
	srcFixture := filepath.Join(root, "cmd", "backstop", "testdata", "dogfood_enforcement", fixtureName)
	data, err := os.ReadFile(srcFixture)
	if err != nil {
		t.Fatalf("reading dogfood fixture %s: %v", fixtureName, err)
	}
	wsFile := filepath.Join(tmp, fixtureName)
	if err := os.WriteFile(wsFile, data, 0o644); err != nil {
		t.Fatalf("copying fixture into workspace: %v", err)
	}

	// The template carries a {{FILE}} placeholder for the in-workspace path.
	contractsYAML := strings.ReplaceAll(contractsYAMLTemplate, "{{FILE}}", wsFile)
	spec := "---\n" +
		"title: \"NoVac Spec\"\nnumber: NV-001\ncreated: \"2026-01-01\"\nstatus: draft\n" +
		"schema_version: spec/v1\nspec_version: 1.0.0\n\n" +
		"implementation:\n  summary: nv\n  package: pkg/gate\n\n" +
		"verification:\n  level: integration\n  test_command: go test ./pkg/gate/\n  coverage_threshold: 80\n\n" +
		"requirements:\n  - id: REQ-001\n    text: req\n    supports: x:REQ-001\n\n" +
		"claims:\n  - id: CLM-001\n    requirement: REQ-001\n    text: claim\n    tests:\n      - TestNV\n\n" +
		"contracts:\n" + contractsYAML +
		"---\n\n# NoVac Spec\n"
	if err := os.WriteFile(filepath.Join(specDir, "nv.spec.md"), []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if _, err := distribution.InstallContractsLocalPack(root, tmp); err != nil {
		t.Fatalf("install contracts pack: %v", err)
	}
	step := buildContractStep(specDir, tmp, nil)
	return step(context.Background())
}

func nvHasViolation(res gate.StepResult, needle string) bool {
	for _, v := range res.Violations {
		if strings.Contains(v.Message, needle) {
			return true
		}
	}
	return false
}

// TestNoVacuousGreen_MissingSignatureBlocks (CLM-031): a missing/mismatched signature
// through the pack path yields a real blocking contract violation (step fail / exit 2).
func TestNoVacuousGreen_MissingSignatureBlocks(t *testing.T) {
	requireNoVacEngines(t)
	res := novacWorkspace(t, "contract_signature_mismatch.go", "  - file: {{FILE}}\n"+
		"    provides:\n      - name: RouteFile\n        kind: function\n"+
		"        signature: \"func RouteFile(path string, mode int) (string, error)\"\n")
	if res.Status != "fail" {
		t.Fatalf("missing/mismatched signature must BLOCK (step fail); got status %q, %#v", res.Status, res.Violations)
	}
	if !nvHasViolation(res, "RouteFile") {
		t.Fatalf("must surface a real contract violation for the missing RouteFile signature; got %#v", res.Violations)
	}
}

// TestNoVacuousGreen_PresentForbiddenSymbolBlocks (CLM-032): a present forbidden symbol
// through the pack grep absence path yields a real blocking absence violation.
func TestNoVacuousGreen_PresentForbiddenSymbolBlocks(t *testing.T) {
	requireNoVacEngines(t)
	res := novacWorkspace(t, "contract_absence_present.go", "  - file: {{FILE}}\n"+
		"    provides:\n      - name: legacyProbeSymbol\n        kind: function\n"+
		"        absent: true\n        scope: {{FILE}}\n")
	if res.Status != "fail" {
		t.Fatalf("present forbidden symbol must BLOCK (step fail); got status %q, %#v", res.Status, res.Violations)
	}
	if !nvHasViolation(res, "legacyProbeSymbol") {
		t.Fatalf("must surface a real absence violation for the present legacyProbeSymbol; got %#v", res.Violations)
	}
}

// TestDogfood_BackstopOwnContractSignatureTurnsGreen (CLM-033): running the new pack
// contract path against backstop's OWN previously-red contract_signature case (a present
// matching signature that the deleted brittle string-equality analyzer round-tripped to
// RED) resolves it to GREEN — the dogfood payoff.
func TestDogfood_BackstopOwnContractSignatureTurnsGreen(t *testing.T) {
	requireNoVacEngines(t)
	res := novacWorkspace(t, "contract_signature_present.go", "  - file: {{FILE}}\n"+
		"    provides:\n      - name: RouteFile\n        kind: function\n"+
		"        signature: \"func RouteFile(path string, mode int) (string, error)\"\n")
	if res.Status != "pass" {
		t.Fatalf("backstop's own present-signature contract must turn GREEN under the pack path "+
			"(the brittle string-equality round-trip is dissolved); got status %q, violations %#v (CLM-033)", res.Status, res.Violations)
	}
}
