package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// TestBuildGateSteps_SpecDirectoryConsumersReadResolvedArtifactRoot pins CLM-069, the
// FIFTH REQ-007 consumer: buildGateSteps' specDir, which feeds test_verification,
// test_substantiveness, coverage and contracts.
//
// The fixture's asymmetry IS the test. dotbackstop-root keeps its specs at
// .backstop/specs/ and has NO specs/ at the project root, so a specDir still derived
// from the project root reads a directory that does not exist.
//
// THE POSITIVE SIGNAL IS NOT "PASS", AND GETTING THAT WRONG REINSTATES THE VACUITY.
// The fixture spec is pinned `status: implemented` and declares one mandated test name,
// so filterDueMandatedTests keeps it and the step reaches the capability guard, which —
// the fixture declaring no packs — returns a WARNING naming the missing discovery
// inputs. Read an EMPTY (or wrong) spec directory instead and there are zero mandated
// tests, so the step takes the len==0 early return BEFORE that guard and reports a
// clean "pass" with zero violations. So "warning, not pass" is precisely the assertion
// that separates "found the mandated name under the resolved root" from "read nothing".
func TestBuildGateSteps_SpecDirectoryConsumersReadResolvedArtifactRoot(t *testing.T) {
	dir := layoutProfileDir(t, "dotbackstop-root")
	root := layoutProfileRoot(t, dir)

	if !root.Configured || filepath.Base(root.Path) != ".backstop" {
		t.Fatalf("the fixture resolved root %q (configured=%v); this test is about a .backstop-rooted project", root.Path, root.Configured)
	}

	specReaders := map[string]bool{
		gate.StepTestVerification:    true,
		gate.StepTestSubstantiveness: true,
		gate.StepCoverageThreshold:   true,
		gate.StepContractSignature:   true,
	}

	resolved := runStepsByName(t, dir, root)

	for name := range specReaders {
		res, ok := resolved[name]
		if !ok {
			continue // a dimension the fixture's pack set does not assemble at all
		}
		if res.Status == "fail" && mentionsMandatedTestExtraction(res) {
			t.Errorf("%s reported a spec-directory read failure under the resolved root: %s", name, stepDiagnostic(res))
		}
	}

	verify, ok := resolved[gate.StepTestVerification]
	if !ok {
		t.Fatal("test_verification did not run at all; the positive assertion below cannot be made")
	}
	if verify.Status == "pass" {
		t.Errorf("test_verification reported a clean pass, which is what reading an EMPTY spec directory produces; the fixture's implemented spec declares a mandated test name, so the resolved root should have driven it to the capability-absent warning. reason=%q", verify.Reason)
	}
	if verify.Status != "warning" {
		t.Errorf("test_verification status = %q, want %q (the capability-absent warning that proves the mandated test name was seen). reason=%q", verify.Status, "warning", verify.Reason)
	}

	// THE FALSIFICATION ARM IS PART OF THIS TEST, NOT A SEPARATE ONE. Without it the
	// assertions above pass on a build where specDir still comes from the project root,
	// because a project carrying BOTH directories cannot tell the two apart.
	preFix := runStepsByName(t, dir, artifact.Root{Path: dir})
	preFixVerify, ok := preFix[gate.StepTestVerification]
	if !ok {
		t.Fatal("test_verification did not run under the pre-fix root shape; the falsification arm cannot be made")
	}
	if preFixVerify.Status != "fail" {
		t.Errorf("under the PRE-FIX root shape (project root, which has no specs/ directory) test_verification status = %q, want %q. If this arm passes, specDir is not actually reading the resolved root and the assertions above hold for the wrong reason.", preFixVerify.Status, "fail")
	}
	if !mentionsMandatedTestExtraction(preFixVerify) {
		t.Errorf("under the PRE-FIX root shape test_verification failed for some reason other than the spec-directory read: %s", stepDiagnostic(preFixVerify))
	}
}

// runStepsByName assembles the gate steps for a project root and artifact root, runs
// each one, and indexes the results by step name.
func runStepsByName(t *testing.T, projectRoot string, root artifact.Root) map[string]gate.StepResult {
	t.Helper()
	out := map[string]gate.StepResult{}
	for _, step := range buildGateSteps(projectRoot, root, &gate.GateScope{Mode: gate.GateScopeModeAll}) {
		res := step(context.Background())
		out[res.StepName] = res
	}
	return out
}

// mentionsMandatedTestExtraction reports whether a step result names the mandated-test
// extraction failure ExtractMandatedTests produces on a missing spec directory. That
// error is observable from the step result alone, which is why this test reaches into
// no unexported state.
func mentionsMandatedTestExtraction(res gate.StepResult) bool {
	if strings.Contains(res.Reason, "extract mandated tests") {
		return true
	}
	for _, v := range res.Violations {
		if strings.Contains(v.Message, "extract mandated tests") {
			return true
		}
	}
	return false
}

// stepDiagnostic renders a step's reason and violation messages for a failure report.
func stepDiagnostic(res gate.StepResult) string {
	parts := []string{"status=" + res.Status}
	if res.Reason != "" {
		parts = append(parts, "reason="+res.Reason)
	}
	for _, v := range res.Violations {
		parts = append(parts, v.Message)
	}
	return strings.Join(parts, " | ")
}
