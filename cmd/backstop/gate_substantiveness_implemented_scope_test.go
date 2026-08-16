package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// gate_substantiveness_implemented_scope_test.go proves the test_substantiveness
// consumer (buildTestSubstantivenessStep) runs the mandated-test-keyed noTarget join
// ONLY for `implemented` specs' mandated tests (ISSUE-054 CLM-004/CLM-007). It drives
// the REAL consumer through the SAME dispatch seam the wiring tests spy, so the
// assertion keys off the production path, not a re-implemented dispatcher.

// writeScopedSubstantivenessSpec writes a spec at the given status mandating
// TestSubjectNoTarget with a pkg/gate implementation package (target package "gate").
func writeScopedSubstantivenessSpec(t *testing.T, specDir, status string) {
	t.Helper()
	content := fmt.Sprintf(`---
title: "Scoped Sub Spec"
number: SUBSCOPE-001
created: "2026-01-01"
status: %s
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: scoped sub
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
      - TestSubjectNoTarget
---

# Scoped Sub Spec
`, status)
	if err := os.WriteFile(filepath.Join(specDir, "subscope.spec.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing scoped sub spec: %v", err)
	}
}

// TestSubstantiveness_EnforcesOnlyImplementedSpecMandatedTests (CLM-004, CLM-007):
// the SAME spec (mandating a test that does not reference its target package and is
// not co-located with it) yields a test_substantiveness noTarget "does not call
// package" finding when the spec is `implemented`, but NONE when it is `draft` /
// `ready-for-implementation` (the draft-spec mandated test is not joined). The
// dispatch seam is spied to return a single referenced-symbol finding naming an
// unrelated symbol, so the referenced set omits the target and the noTarget
// decision-table fires purely on the mandated-test join — exactly the surface the
// consumer filter must bite.
func TestSubstantiveness_EnforcesOnlyImplementedSpecMandatedTests(t *testing.T) {
	cases := []struct {
		status          string
		wantEnforcement bool
	}{
		{"implemented", true},
		{"draft", false},
		{"ready-for-implementation", false},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			specDir := t.TempDir()
			codeDir := t.TempDir()
			writeScopedSubstantivenessSpec(t, specDir, tc.status)
			// A found mandated test file, NOT co-located with target package "gate".
			writeMandatedTestFile(t, codeDir, "subject_test.go", "TestSubjectNoTarget", "\tdoSubject()\n")
			injectSubstantivenessManifest(t)

			// Spy the REAL dispatch seam: return ONE referenced-symbol finding that joins
			// this mandated test but names an UNRELATED symbol, so the test's referenced
			// set is {"helper"} — target package "gate" is absent and the noTarget path is
			// driven exactly as before.
			//
			// MUST NOT be simplified back to `return nil, nil` (ISSUE-113). An empty
			// dispatch stream means the pack produced no evidence of ANY kind, which is
			// now the starved-join state the substantiveness step REFUSES on with a
			// config-error instead of emitting unfounded per-test noTarget violations. A
			// nil stream would make this fixture exercise that refusal rather than its own
			// subject. The subject here is ISSUE-054's implemented-only scope filter, so
			// the pack must present as HEALTHY (it ran, it classified this file) and merely
			// report a symbol that is not the target.
			orig := dispatchPackEnginesFn
			t.Cleanup(func() { dispatchPackEnginesFn = orig })
			dispatchPackEnginesFn = func(_ []*pack.Manifest, _, _ string, _ *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
				return []gate.Violation{{
					Rule:    "backstop/substantiveness/referenced-symbol-go",
					File:    "subject_test.go",
					Message: "referenced-symbol func=TestSubjectNoTarget symbol=helper",
					Properties: map[string]string{
						"substantiveness_role": "referenced-symbol",
						"func":                 "TestSubjectNoTarget",
						"symbol":               "helper",
					},
				}}, nil
			}

			classifier, matcher := goSubstDiscovery(t)
			step := buildTestSubstantivenessStep(specDir, codeDir, codeDir, nil, classifier, matcher)
			result := step(context.Background())

			var noTargetRaised bool
			for _, v := range result.Violations {
				if v.Rule == gate.StepTestSubstantiveness &&
					v.Message == "test function TestSubjectNoTarget does not call package gate" {
					noTargetRaised = true
				}
			}
			if noTargetRaised != tc.wantEnforcement {
				t.Fatalf("status %q: expected noTarget enforcement=%v, got %v (status=%q violations=%#v)",
					tc.status, tc.wantEnforcement, noTargetRaised, result.Status, result.Violations)
			}
		})
	}
}
