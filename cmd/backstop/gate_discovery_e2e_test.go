package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// gate_discovery_e2e_test.go proves SPEC-045 REQ-006: the merged SourceClassifier +
// merged TestNameMatcher are threaded into the LIVE gate test-verification and
// substantiveness steps and consumed END-TO-END over REAL declared toolchain packs
// (the spec045-discovery testdata's go-toolchain + bun-toolchain packs), NOT only by
// unit tests over hand-constructed inputs ([[feedback_integration_gap]]).

// spec045DiscoveryRoot resolves the spec045-discovery testdata project the REQ-006
// e2e tests drive the FULL gate assembly over.
func spec045DiscoveryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("testdata", "spec045-discovery"))
	if err != nil {
		t.Fatalf("resolving spec045-discovery project root: %v", err)
	}
	return root
}

// testVerifyStepResultOverProject drives the FULL gate assembly (buildGateSteps)
// over the given project + scope files and returns the test-verification step's
// result — deliberately the assembled live steps, so the live mergeSourceClassifier
// + mergeTestNameMatcher call sites in buildGateSteps are the exercised path.
func testVerifyStepResultOverProject(t *testing.T, projectRoot string, files ...string) gate.StepResult {
	t.Helper()
	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, files)
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	steps := buildGateSteps(projectRoot, scope)
	for _, step := range steps {
		res := step(context.Background())
		if res.StepName == gate.StepTestVerification {
			return res
		}
	}
	t.Fatalf("test-verification step (%s) not present in the assembled gate steps", gate.StepTestVerification)
	return gate.StepResult{}
}

func violationMentions(violations []gate.Violation, needle string) bool {
	for _, v := range violations {
		if strings.Contains(v.Message, needle) {
			return true
		}
	}
	return false
}

// TestGate_TestVerificationConsumesMergedDiscoveryEndToEnd (CLM-034): with a declared
// toolchain pack whose test globs + name patterns cover `.test.ts`, a `.test.ts` file
// whose test name matches a mandated claim is DISCOVERED and verified by the REAL
// assembled gate — so it is NOT reported as "not found". Proves the live gate threads
// the merged classifier + matcher into the test-verification step.
func TestGate_TestVerificationConsumesMergedDiscoveryEndToEnd(t *testing.T) {
	root := spec045DiscoveryRoot(t)
	res := testVerifyStepResultOverProject(t, root, "specs/discovery.spec.md")

	if violationMentions(res.Violations, "renders the widget") {
		t.Fatalf("the `.test.ts` mandated test must be DISCOVERED via the merged bun globs+patterns (not reported not-found), got %s: %#v", res.Status, res.Violations)
	}
	if res.Status == "fail" {
		t.Fatalf("the live test-verification step must not fail when the mandated TS test is discovered, got %#v", res.Violations)
	}
}

// TestGate_DiscoveryMergesAcrossDeclaredToolchainPacks (CLM-036): with TWO toolchain
// packs declared (go + bun), the live test-verification step discovers BOTH a
// `_test.go` and a `.test.ts` mandated test from the merged glob + pattern set (the
// UNION across declared packs), so neither is reported "not found".
func TestGate_DiscoveryMergesAcrossDeclaredToolchainPacks(t *testing.T) {
	root := spec045DiscoveryRoot(t)
	res := testVerifyStepResultOverProject(t, root, "specs/discovery.spec.md")

	if violationMentions(res.Violations, "TestDiscoveredGoTest") {
		t.Errorf("the go `_test.go` mandated test must be discovered from the merged classifier (union), got %#v", res.Violations)
	}
	if violationMentions(res.Violations, "renders the widget") {
		t.Errorf("the bun `.test.ts` mandated test must be discovered from the merged classifier (union), got %#v", res.Violations)
	}
	if res.Status == "fail" {
		t.Fatalf("both mandated tests must be discovered via the merged union, got fail: %#v", res.Violations)
	}
}

// --- CLM-035: substantiveness consumes testFileColocatedWithTarget end-to-end ---

// writeColocationSpec writes a spec mandating the TS test name with a pkg/<leaf>
// implementation package so TargetPackageName yields <leaf> (TargetPackageName
// returns "" for non-pkg/ paths). The co-located TS test lives in app/<leaf>.
func writeColocationSpec(t *testing.T, specDir string) {
	t.Helper()
	content := `---
title: "Coloc Spec"
number: COLOC-001
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: coloc
  package: pkg/widget

verification:
  level: integration
  test_command: bun test
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
      - renders the widget
---

# Coloc Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "coloc.spec.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing coloc spec: %v", err)
	}
}

// TestGate_SubstantivenessUsesColocationEndToEnd (CLM-035): the live substantiveness
// step consumes testFileColocatedWithTarget — a TS test file co-located with its
// target (directory leaf == targetPkg) short-circuits the same-unit join in the real
// step WITHOUT any Go `package` clause involved. Were the deleted Go package-clause
// reader still wired, it would open the `.test.ts` file, find no `package widget`
// clause, judge it NOT same-unit, and (with a non-target referenced set) RAISE a
// "does not call package widget" noTarget violation. testFileColocatedWithTarget
// makes it same-unit by directory leaf, so no violation is raised.
func TestGate_SubstantivenessUsesColocationEndToEnd(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeColocationSpec(t, specDir)

	// A co-located TS test: directory leaf "widget" == targetPkg "widget"; NO Go
	// package clause exists in the file.
	tsDir := filepath.Join(codeDir, "app", "widget")
	if err := os.MkdirAll(tsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tsFile := filepath.Join(tsDir, "widget.test.ts")
	if err := os.WriteFile(tsFile, []byte("test('renders the widget', () => {\n  expect(1).toBe(1)\n})\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pack-shaped bun classifier + matcher (the merged discovery the live gate builds).
	classifier := gate.NewSourceClassifier([]string{"**/*.ts"}, []string{"**/*.test.ts", "**/*.spec.ts"})
	matcher, err := gate.NewTestNameMatcher([]string{"(?:\\bit|\\btest|\\bdescribe)\\s*\\(\\s*['\"`]([^'\"`]+)"})
	if err != nil {
		t.Fatalf("NewTestNameMatcher: %v", err)
	}

	// Inject the substantiveness pack so the step dispatches; spy the REAL dispatch
	// seam so the step runs end-to-end (live wiring), returning NO findings (empty
	// referenced set) — so only the co-location short-circuit can prevent a violation.
	injectSubstantivenessManifest(t)
	var invocations int
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
		invocations++
		return nil, nil
	}

	step := buildTestSubstantivenessStep(specDir, codeDir, codeDir, nil, classifier, matcher)
	res := step(context.Background())

	if invocations == 0 {
		t.Fatal("the live substantiveness step must reach the dispatch seam (proves end-to-end wiring)")
	}
	if violationMentions(res.Violations, "does not call package widget") {
		t.Fatalf("a co-located TS test must short-circuit same-unit via testFileColocatedWithTarget (no package clause), got noTarget violation: %#v", res.Violations)
	}
	if res.Status != "pass" {
		t.Fatalf("the co-located TS test must not raise a substantiveness violation, got %s: %#v", res.Status, res.Violations)
	}
}
