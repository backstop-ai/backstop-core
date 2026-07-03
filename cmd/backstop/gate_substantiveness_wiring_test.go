package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// gate_substantiveness_wiring_test.go installs a SPY on the REAL dispatchPackEnginesFn
// seam (cmd/backstop/code_check.go) — the SAME seam code check and the pack_engines step
// (gate.go) use, not a parallel stub. It proves buildTestSubstantivenessStep reaches the
// dispatcher with the substantiveness pack set (CLM-015), that an unwired step records
// zero invocations and fails (CLM-016), and that the verdict can only originate from the
// dispatch path — no baked analyzer delegate exists (CLM-017).

// hollowBody is a hollow Go test body (calls a subject, asserts nothing).
const hollowBody = "\tdoSubject()\n"

// goSubstDiscovery builds the pack-shaped Go classifier + matcher (the go-toolchain
// pack DATA) the substantiveness step now consumes to resolve mandated Go test paths.
func goSubstDiscovery(t *testing.T) (gate.SourceClassifier, gate.TestNameMatcher) {
	t.Helper()
	classifier := gate.NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	matcher, err := gate.NewTestNameMatcher([]string{`^\s*func\s+(Test\w+)\s*\(`})
	if err != nil {
		t.Fatalf("NewTestNameMatcher(go): %v", err)
	}
	return classifier, matcher
}

// injectSubstantivenessManifest overrides the resolveSubstantivenessPacksFn seam so the
// step dispatches the substantiveness pack set (the dispatch ITSELF is spied, so the
// manifest need not be on disk for the wiring proof). Production resolves the pack from
// INSTALLED packs; this seam only feeds the spy.
func injectSubstantivenessManifest(t *testing.T) {
	t.Helper()
	orig := resolveSubstantivenessPacksFn
	t.Cleanup(func() { resolveSubstantivenessPacksFn = orig })
	resolveSubstantivenessPacksFn = func(string) ([]*pack.Manifest, error) {
		return []*pack.Manifest{{
			Name:           "backstop/substantiveness",
			NormalizedName: "backstop/substantiveness",
			Language:       "go",
			Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
				{ID: "hollow-test-go", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml"},
				{ID: "referenced-symbol-go", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml"},
			}}},
		}}, nil
	}
}

// writeSubstantivenessSpec writes a minimal spec mandating TestSubjectHollow with a
// pkg/gate implementation package (target package "gate").
func writeSubstantivenessSpec(t *testing.T, specDir string) {
	t.Helper()
	content := `---
title: "Sub Spec"
number: SUB-001
created: "2026-01-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: sub
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
      - TestSubjectHollow
---

# Sub Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "sub.spec.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing sub spec: %v", err)
	}
}

// writeMandatedTestFile writes a Go test file with the named test function and body,
// returning its path.
func writeMandatedTestFile(t *testing.T, codeDir, filename, funcName, body string) string {
	t.Helper()
	content := "package sample_test\n\nimport \"testing\"\n\nfunc " + funcName + "(t *testing.T) {\n" + body + "}\n"
	path := filepath.Join(codeDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing mandated test file: %v", err)
	}
	return path
}

// substantivenessHollowNamespacedID is the namespaced hollow rule ID the spy emits.
func substantivenessHollowNamespacedID() string {
	return pack.NamespacedRuleID("backstop/substantiveness", "hollow-test-go")
}

// TestWiring_SubstantivenessStepRoutesThroughDispatchSeam (CLM-015): the step invokes the
// dispatch seam with the substantiveness pack set, then runs the set-join — the spy
// records a non-zero invocation carrying the substantiveness pack.
func TestWiring_SubstantivenessStepRoutesThroughDispatchSeam(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSubstantivenessSpec(t, specDir)
	writeMandatedTestFile(t, codeDir, "subject_test.go", "TestSubjectHollow", hollowBody)
	injectSubstantivenessManifest(t)

	var invocations int
	var sawSubstantivenessPack bool
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func(packs []*pack.Manifest, _, _ string, _ *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
		invocations++
		for _, m := range packs {
			if m.NormalizedName == "backstop/substantiveness" {
				sawSubstantivenessPack = true
			}
		}
		return nil, nil
	}

	classifier, matcher := goSubstDiscovery(t)
	step := buildTestSubstantivenessStep(specDir, codeDir, codeDir, nil, classifier, matcher)
	_ = step(context.Background())

	if invocations == 0 {
		t.Fatal("the substantiveness step MUST reach the dispatchPackEnginesFn seam (zero invocations recorded)")
	}
	if !sawSubstantivenessPack {
		t.Fatal("the step must invoke the dispatch seam WITH the substantiveness pack set")
	}
}

// TestWiring_UnwiredSubstantivenessStep_FailsDispatchSpy (CLM-016): an UNWIRED step that
// never reaches dispatchPackEnginesFn records ZERO invocations and the assertion on
// non-zero FAILS — so a regression to an unwired/baked path is caught. The wired step,
// by contrast, DOES reach the seam, proving the assertion distinguishes wired from
// unwired.
func TestWiring_UnwiredSubstantivenessStep_FailsDispatchSpy(t *testing.T) {
	var invocations int
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
		invocations++
		return nil, nil
	}

	// An unwired step never calls the dispatch seam.
	unwired := func(_ context.Context) gate.StepResult {
		return gate.StepResult{StepName: gate.StepTestSubstantiveness, Status: "pass"}
	}
	_ = unwired(context.Background())
	if invocations != 0 {
		t.Fatalf("an unwired step must record ZERO dispatch invocations, got %d", invocations)
	}

	// The wired step DOES reach the seam — proving the assertion has teeth.
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSubstantivenessSpec(t, specDir)
	writeMandatedTestFile(t, codeDir, "subject_test.go", "TestSubjectHollow", hollowBody)
	injectSubstantivenessManifest(t)
	classifier, matcher := goSubstDiscovery(t)
	step := buildTestSubstantivenessStep(specDir, codeDir, codeDir, nil, classifier, matcher)
	_ = step(context.Background())
	if invocations == 0 {
		t.Fatal("the wired step must reach the dispatch seam (non-zero), distinguishing it from the unwired path")
	}
}

// TestWiring_NoBakedAnalyzerDelegateInvoked (CLM-017): the verdict originates from the
// dispatch seam (spy-observed) and no baked analyzer delegate is invoked — a
// hollow/noTarget verdict can only have come from the pack dispatch path. The spy returns
// a synthetic hollow finding through the seam; the resulting step verdict must reflect it,
// proving the step's verdict is sourced from the dispatch return, not a baked analyzer.
func TestWiring_NoBakedAnalyzerDelegateInvoked(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSubstantivenessSpec(t, specDir)
	testFile := writeMandatedTestFile(t, codeDir, "subject_test.go", "TestSubjectHollow", hollowBody)
	injectSubstantivenessManifest(t)

	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
		// The dispatch seam is the SOLE source of the hollow verdict: emit a hollow
		// finding for the mandated test in the pinned format.
		return []gate.Violation{{
			Rule:    substantivenessHollowNamespacedID(),
			File:    testFile,
			Message: "test function TestSubjectHollow has no assertions (hollow) func=TestSubjectHollow",
		}}, nil
	}

	classifier, matcher := goSubstDiscovery(t)
	step := buildTestSubstantivenessStep(specDir, codeDir, codeDir, nil, classifier, matcher)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Fatalf("a hollow verdict sourced from the dispatch seam must fail the step, got %q", result.Status)
	}
	foundHollow := false
	for _, v := range result.Violations {
		if v.Rule == gate.StepTestSubstantiveness && strings.Contains(v.Message, "has no assertions (hollow)") {
			foundHollow = true
		}
	}
	if !foundHollow {
		t.Fatalf("the step verdict must originate from the dispatch seam's hollow finding, got %#v", result.Violations)
	}
}
