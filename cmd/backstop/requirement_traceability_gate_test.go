package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
)

func TestGateCLI_RequirementTraceabilityStepsWired(t *testing.T) {
	root := traceWorkspace(t)
	names := map[string]bool{}
	for _, step := range buildGateSteps(root, &gate.GateScope{Mode: gate.GateScopeModeDiff}) {
		res := step(context.Background())
		names[res.StepName] = true
	}
	if !names[gate.StepRequirementTraceability] || !names[gate.StepRequirementTraceabilityAdvisory] {
		t.Fatalf("traceability steps not wired: %#v", names)
	}
}

func TestGateCLI_RequirementTraceability_DeliveredUncoveredBlocksOverFixtureCorpus(t *testing.T) {
	root := traceWorkspace(t)
	writeTraceBundle(t, root, "delivered", "1.0.0")
	block, advisory := computeRequirementTraceabilitySurfaces(root)
	if block.StepName != gate.StepRequirementTraceability || block.Status != "fail" || !traceResultContains(block, "REQ-001") {
		t.Fatalf("delivered uncovered bundle should block over fixture corpus: %#v", block)
	}
	if advisory.StepName != gate.StepRequirementTraceabilityAdvisory || advisory.Status != "pass" {
		t.Fatalf("unexpected advisory result: %#v", advisory)
	}
}

func TestGateCLI_RequirementTraceability_PathNormalizationJoins(t *testing.T) {
	root := traceWorkspace(t)

	// End-to-end sanity over the real fixture corpus: a delivered bundle + implemented
	// spec join and cover.
	writeTraceBundle(t, root, "delivered", "1.0.0")
	writeTraceSpec(t, root, "SPEC-001", "implemented", "trace-fixture:REQ-001@1.0.0")
	block, advisory := computeRequirementTraceabilitySurfaces(root)
	if block.Status != "pass" || advisory.Status != "pass" {
		t.Fatalf("normalized spec path should join support ref to citing record: block=%#v advisory=%#v", block, advisory)
	}

	// CLM-050: a deliberate absolute/relative path SKEW — the discovered spec RECORD
	// carries an ABSOLUTE path while its supports REF cites the same file by a
	// RELATIVE path. This is the mismatch the CLI wiring reconciles by
	// NormalizePath-ing both against projectRoot before classification.
	absSpecPath := filepath.Join(root, "specs", "SPEC-001-trace-fixture.spec.md")
	relSpecPath := "specs/SPEC-001-trace-fixture.spec.md"
	if !filepath.IsAbs(absSpecPath) || absSpecPath == relSpecPath {
		t.Fatalf("test setup broken: expected a real abs/rel skew, got abs=%q rel=%q", absSpecPath, relSpecPath)
	}

	skewRecords := []gate.ArtifactStatusRecord{
		{ID: "BUNDLE-999", Kind: gate.KindBundle, Status: "delivered", Class: gate.ClassSuccessTerminal, Path: filepath.Join(root, "bundles", "BUNDLE-999-trace-fixture.bundle.md"), BundleName: "trace-fixture", BundleReqs: []gate.BundleReqVersion{{ReqID: "REQ-001", CurrentVersion: "1.0.0"}}},
		{ID: "SPEC-001", Kind: gate.KindSpec, Status: "implemented", Class: gate.ClassSuccessTerminal, Path: absSpecPath},
	}
	skewRefs := []gate.TraceRef{{BundleName: "trace-fixture", ReqID: "REQ-001", PinVersion: "1.0.0", Pinned: true, CitingPath: relSpecPath}}

	// WITHOUT projectRoot-anchored normalization the abs record path and the rel ref
	// path never join → the delivered REQ is silently dropped to uncovered → block FAILS.
	rawBlock, rawAdvisory := gate.SplitTraceabilityResult(gate.ClassifyRequirementTraceability(skewRecords, skewRefs))
	if rawBlock.Status != "fail" || !traceResultContains(rawBlock, "REQ-001") {
		t.Fatalf("an unnormalized abs/rel skew must drop coverage (block fail on REQ-001): block=%#v advisory=%#v", rawBlock, rawAdvisory)
	}

	// WITH the CLI wiring's NormalizePath(projectRoot, …) on both sides the skew
	// collapses to a single key → coverage joins → block PASSES.
	for i := range skewRecords {
		skewRecords[i].Path = gate.NormalizePath(root, skewRecords[i].Path)
	}
	for i := range skewRefs {
		skewRefs[i].CitingPath = gate.NormalizePath(root, skewRefs[i].CitingPath)
	}
	joinedBlock, joinedAdvisory := gate.SplitTraceabilityResult(gate.ClassifyRequirementTraceability(skewRecords, skewRefs))
	if joinedBlock.Status != "pass" || joinedAdvisory.Status != "pass" {
		t.Fatalf("normalized abs/rel paths must join and cover REQ-001: block=%#v advisory=%#v", joinedBlock, joinedAdvisory)
	}
}

func TestGateCLI_RequirementTraceabilityVerdictInOutput(t *testing.T) {
	root := traceWorkspace(t)
	writeTraceBundle(t, root, "delivered", "1.0.0")
	block, advisory := computeRequirementTraceabilitySurfaces(root)
	out := gate.FormatHuman(gate.GateResult{Steps: []gate.StepResult{block, advisory}}, true)
	if !strings.Contains(out, gate.StepRequirementTraceability) || !strings.Contains(out, gate.StepRequirementTraceabilityAdvisory) {
		t.Fatalf("gate output must include traceability step names, got:\n%s", out)
	}
}

func traceWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"bundles", "specs", "issues", "plans"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte("project: trace-fixture\npacks: {}\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	return root
}

func writeTraceBundle(t *testing.T, root string, maturity string, version string) {
	t.Helper()
	writeTraceFile(t, filepath.Join(root, "bundles", "BUNDLE-999-trace-fixture.bundle.md"), `---
number: BUNDLE-999
bundle:
  name: trace-fixture
status:
  maturity: `+maturity+`
requirements:
  - id: REQ-001
    version: `+version+`
---

# Trace fixture
`)
}

func writeTraceSpec(t *testing.T, root string, id string, status string, supports string) {
	t.Helper()
	writeTraceFile(t, filepath.Join(root, "specs", id+"-trace-fixture.spec.md"), `---
title: Trace Fixture
number: `+id+`
status: `+status+`
requirements:
  - id: REQ-001
    text: covers trace fixture
    supports: `+supports+`
claims: []
---

# Trace fixture
`)
}

func writeTraceFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func traceResultContains(res gate.StepResult, want string) bool {
	for _, v := range res.Violations {
		if strings.Contains(v.Message, want) || strings.Contains(v.File, want) {
			return true
		}
	}
	return false
}
