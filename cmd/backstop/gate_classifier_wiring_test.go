package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// classifierE2EProjectRoot resolves the classifier-e2e testdata project the
// REQ-005 wiring tests drive the FULL gate assembly over.
func classifierE2EProjectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("testdata", "classifier-e2e"))
	if err != nil {
		t.Fatalf("resolving classifier-e2e project root: %v", err)
	}
	return root
}

// coverageStepResultOverProject drives the FULL gate assembly (buildGateSteps)
// over the given project + changed files and returns the coverage step's result.
// It deliberately runs the assembled steps — NOT a hand-merged classifier handed
// to buildCoverageStep/StepCoverageThresholdScopedFunc — so the live
// mergeSourceClassifier(packs) call site in buildGateSteps is the exercised path
// (the SPEC-035/037 integration-gap closure).
func coverageStepResultOverProject(t *testing.T, projectRoot string, files ...string) gate.StepResult {
	t.Helper()
	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, files)
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	steps := buildGateSteps(projectRoot, scope)
	for _, step := range steps {
		res := step(context.Background())
		if res.StepName == gate.StepCoverageThreshold {
			return res
		}
	}
	t.Fatalf("coverage step (%s) not present in the assembled gate steps", gate.StepCoverageThreshold)
	return gate.StepResult{}
}

func violationPresentForFile(violations []gate.Violation, file string) bool {
	for _, v := range violations {
		if v.File == file {
			return true
		}
	}
	return false
}

// TestGate_BuildGateStepsConsumesMergedClassifierEndToEnd (CLM-020): with a
// declared toolchain pack whose source globs cover `.ts`, a changed `.ts` file
// with no coverage record REDs the REAL assembled gate — proving the classifier
// is built-and-consumed on the production path via the live
// mergeSourceClassifier(packs) call site in buildGateSteps.
func TestGate_BuildGateStepsConsumesMergedClassifierEndToEnd(t *testing.T) {
	root := classifierE2EProjectRoot(t)
	res := coverageStepResultOverProject(t, root, "app/foo.ts")

	if res.Status != "fail" {
		t.Fatalf("a changed .ts source file with no record must RED the real assembled gate (vacuous-green closed), got %s: %#v", res.Status, res.Violations)
	}
	if !violationPresentForFile(res.Violations, "app/foo.ts") {
		t.Fatalf("expected a loud coverage violation for app/foo.ts from the live merged classifier, got %#v", res.Violations)
	}
}

// TestGate_ClassifierMergesAcrossDeclaredToolchainPacks (CLM-021): with TWO
// toolchain packs declared (go + bun), the coverage step measures BOTH a changed
// `.go` and a changed `.ts` file from the merged glob set (UNION across declared
// packs).
func TestGate_ClassifierMergesAcrossDeclaredToolchainPacks(t *testing.T) {
	root := classifierE2EProjectRoot(t)
	res := coverageStepResultOverProject(t, root, "app/foo.ts", "pkg/x/foo.go")

	if res.Status != "fail" {
		t.Fatalf("both changed source files (no records) must RED via the merged union, got %s: %#v", res.Status, res.Violations)
	}
	if !violationPresentForFile(res.Violations, "app/foo.ts") {
		t.Errorf("the bun (.ts) glob member must be measured by the merged classifier, got %#v", res.Violations)
	}
	if !violationPresentForFile(res.Violations, "pkg/x/foo.go") {
		t.Errorf("the go (.go) glob member must be measured by the merged classifier, got %#v", res.Violations)
	}
}

// TestGate_MergeSourceClassifierSourcesFromDeclaredPacksNotBridge (CLM-022):
// mergeSourceClassifier builds from the FULL declared-manifest set
// (loadInstalledPacks over backstop.yml packs:, passed wholesale with NO
// toolchain-only pre-filter) and takes NO `bridged` argument — given declared
// manifests where only a toolchain pack carries source globs it produces a
// classifier whose source globs measure the in-scope changed files, proving it
// survives SPEC-046's bridge deletion.
func TestGate_MergeSourceClassifierSourcesFromDeclaredPacksNotBridge(t *testing.T) {
	root := classifierE2EProjectRoot(t)
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	if len(packs) == 0 {
		t.Fatal("test precondition: the declared packs must resolve from backstop.yml")
	}

	classifier := mergeSourceClassifier(packs)

	if !classifier.IsMeasurableSource("app/foo.ts") {
		t.Errorf("the merged classifier (from declared packs) must measure the .ts source file app/foo.ts")
	}
	if !classifier.IsMeasurableSource("pkg/x/foo.go") {
		t.Errorf("the merged classifier (from declared packs) must measure the .go source file pkg/x/foo.go")
	}
	if !classifier.HasSourceGlobs() {
		t.Errorf("the merged classifier must report declared source globs from the declared toolchain packs")
	}
}
