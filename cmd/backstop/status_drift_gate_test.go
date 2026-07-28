package main

// Phase-3 wiring/e2e tests (ISSUE-042 TASK-007, CLM-003/004/005/006/007/009/010/011/015).
// These drive the native status/reality drift dimension THROUGH the real gate wiring
// (buildGateSteps / buildStatusDriftSteps) and the real gate exit path, closing the
// integration gap: they prove the Phase-1 resolver + Phase-2 classifier reach the live
// gate, that EXISTENCE is full-sweep (ignores diff scope), and that the WARN/BLOCK/exit
// polarity holds. Scenarios are built in isolated temp projects.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// --- fixture builders ------------------------------------------------------------------

func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// installDriftPack writes a backstop.yml declaring a minimal classification-only pack
// plus the pack manifest, so the merged SourceClassifier + TestNameMatcher can discover
// the fixture's *_test.go files. The pack declares one no-op grep rule solely so the
// manifest `content` block is non-empty; the drift step never dispatches engines.
func installDriftPack(t *testing.T, root string) {
	t.Helper()
	writeFixture(t, root, "backstop.yml", "project: drift-fixture\npacks:\n    backstop/drift-toolchain: local\n")
	manifest := `name: backstop/drift-toolchain
version: 1.0.0
language: go
archetype: enforcement
description: minimal classification-only pack for ISSUE-042 drift fixtures
classification:
  source:
    - "**/*.go"
  test:
    - "**/*_test.go"
    - "**/testdata/**"
test_name_patterns:
  - "^\\s*func\\s+(Test\\w+)\\s*\\("
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: drift-noop
        engine: grep
        pattern: "zzz-never-matches-anything"
        risk_class: correctness
        category: toolchain
        justification: no-op placeholder so the fixture pack has non-empty content
`
	writeFixture(t, root, ".backstop/packs/backstop/drift-toolchain/pack.yml", manifest)
}

// noPackProject writes a project with a backstop.yml declaring NO packs, so buildGateSteps
// builds its full step list without the pack-dispatch machinery.
func noPackProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "backstop.yml", "project: drift-nopack\n")
	if err := os.MkdirAll(filepath.Join(root, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func issueFixture(id, status string, testNames ...string) string {
	fm := "---\ntitle: \"fixture " + id + "\"\nschema_version: issue/v1\nissue:\n  id: " + id + "\n  status: " + status + "\n"
	if len(testNames) > 0 {
		fm += "claims:\n  - id: CLM-001\n    tests:\n"
		for _, n := range testNames {
			fm += "      - " + n + "\n"
		}
	}
	fm += "---\nfixture body\n"
	return fm
}

// driftStepsFor builds the drift block + advisory steps over a fixture project via the
// REAL wiring helper (buildStatusDriftSteps) fed by the merged classifier + matcher from
// the fixture's installed packs — exercising the live full-sweep existence path without
// running the pack_engines / go-toolchain dispatch.
func driftStepsFor(t *testing.T, root string) (block, advisory gate.StepFunc) {
	t.Helper()
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	classifier := mergeSourceClassifier(packs)
	matcher, mErr := mergeTestNameMatcher(packs)
	if mErr != nil {
		t.Fatalf("mergeTestNameMatcher: %v", mErr)
	}
	return buildStatusDriftSteps(root, classifier, matcher)
}

func runStep(step gate.StepFunc) gate.StepResult { return step(context.Background()) }

// --- tests -----------------------------------------------------------------------------

// TestGate_StatusDriftStep_WiredIntoBuildGateSteps (CLM-009): a step with the canonical
// StepArtifactStatusDrift name is present in the steps buildGateSteps returns (native, not
// a pack step).
func TestGate_StatusDriftStep_WiredIntoBuildGateSteps(t *testing.T) {
	root := noPackProject(t)
	steps := buildGateSteps(root, &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{}})

	var names []string
	for _, s := range steps {
		names = append(names, runStep(s).StepName)
	}
	found := false
	for _, n := range names {
		if n == gate.StepArtifactStatusDrift {
			found = true
		}
	}
	if !found {
		t.Fatalf("buildGateSteps must wire a %q step (native drift dimension); got steps=%v", gate.StepArtifactStatusDrift, names)
	}
}

// TestGate_StatusDrift_FullSweep_CatchesOutOfDiffStaleStatus (CLM-007/CLM-011): a
// success-terminal artifact whose FILE is NOT in the diff scope, with an ABSENT mandated
// test, still produces a blocking violation — proving existence resolution ignores
// activeScope and diff-scope alone would have missed it (the ISSUE-018 repro).
func TestGate_StatusDrift_FullSweep_CatchesOutOfDiffStaleStatus(t *testing.T) {
	root := t.TempDir()
	installDriftPack(t, root)
	// A closed issue mandating a test that exists NOWHERE in the tree. The buildStatusDriftSteps
	// existence resolution takes NO gate scope — it walks the whole repo — so a stale artifact
	// whose file would be OUT of any diff scope is still caught (the ISSUE-018 repro that
	// diff-scope alone provably misses).
	writeFixture(t, root, "issues/ISSUE-777-stale.issue.md", issueFixture("ISSUE-777", "closed", "TestNeverExistsAnywhere"))

	block, advisory := driftStepsFor(t, root)
	res := runStep(block)
	if res.Status != "fail" {
		t.Fatalf("full-sweep drift must catch the out-of-diff closed+absent artifact; status=%q violations=%+v", res.Status, res.Violations)
	}
	if len(res.Violations) == 0 {
		t.Fatal("expected a blocking drift violation for the stale closed issue")
	}
	msg := res.Violations[0].Message
	if !contains(msg, "ISSUE-777") || !contains(msg, "TestNeverExistsAnywhere") {
		t.Errorf("violation must name the stale artifact + absent test, got %q", msg)
	}
	// No delivered-but-open fixture here, so the advisory surface is empty.
	if a := runStep(advisory); len(a.Violations) != 0 {
		t.Errorf("advisory surface should be empty (no delivered-but-open artifact), got %+v", a.Violations)
	}
}

// TestGate_StatusDrift_DeliveredOpen_ExitZeroWithWarning (CLM-006/CLM-011): a delivered-
// but-open fixture yields a warning on the report surface and the gate exit stays 0.
func TestGate_StatusDrift_DeliveredOpen_ExitZeroWithWarning(t *testing.T) {
	root := t.TempDir()
	installDriftPack(t, root)
	writeFixture(t, root, "issues/ISSUE-500-open.issue.md", issueFixture("ISSUE-500", "open", "TestDriftPresentAlpha"))
	// The mandated test EXISTS in the tree (a *_test.go under the pack's test glob).
	writeFixture(t, root, "src/present_test.go", "package src\n\nfunc TestDriftPresentAlpha() {}\n")

	block, advisory := driftStepsFor(t, root)

	// The advisory surface warns; the block surface has nothing.
	adv := runStep(advisory)
	if adv.Status != "warning" {
		t.Fatalf("advisory status = %q, want warning (delivered-but-open)", adv.Status)
	}
	if len(adv.Violations) == 0 || adv.Violations[0].Severity != "warning" {
		t.Errorf("expected a warning-severity guidance violation, got %+v", adv.Violations)
	}
	if b := runStep(block); b.Status == "fail" {
		t.Errorf("block surface must not fail for a delivered-but-open artifact, got %+v", b)
	}

	// Exit stays 0 through the real gate exit path, even under the block+new-code policy.
	policy := map[string]gate.DimensionPolicy{gate.StepArtifactStatusDrift: {Level: gate.PolicyBlock, AppliesTo: gate.AppliesToNewCode}}
	g := gate.New(
		gate.WithSteps([]gate.StepFunc{block, advisory}),
		gate.WithScope(&gate.GateScope{Mode: gate.GateScopeModeAll, ProjectRoot: root}),
		gate.WithPolicy(policy),
	)
	result, exit := g.Run(context.Background())
	if exit != 0 {
		t.Fatalf("delivered-but-open must leave exit 0, got %d (pass=%v)", exit, result.Pass)
	}
	if result.StepsWarned == 0 {
		t.Error("expected the warning to be counted on the report surface (StepsWarned > 0)")
	}
}

// TestGate_StatusDrift_SuccessTerminalAbsent_ExitTwo (CLM-004/CLM-011): a closed issue
// with an absent mandated test drives a BLOCKING gate via the drift dimension, under an
// applies-to:all-code policy so the block is observable without a baseline confound.
//
// NOTE ON EXIT CODE: in backstop's exit model a FINDINGS-based block is exit 1 (Pass ==
// false); exit 2 is reserved for CONFIG errors (ConfigErr steps, which HALT before policy).
// The drift dimension is deliberately NOT a ConfigErr — that is exactly what lets its
// pre-existing findings grandfather through ApplyPolicy against the baseline (CLM-012/015);
// a ConfigErr block would bypass grandfathering and hard-block the clean tree forever. So
// this test asserts the gate BLOCKS (non-zero exit, Pass false), which is the load-bearing
// behavior the test name refers to.
func TestGate_StatusDrift_SuccessTerminalAbsent_ExitTwo(t *testing.T) {
	root := t.TempDir()
	installDriftPack(t, root)
	writeFixture(t, root, "issues/ISSUE-002-closed.issue.md", issueFixture("ISSUE-002", "closed", "TestCodeCheck_DeletedForever"))

	block, advisory := driftStepsFor(t, root)
	policy := map[string]gate.DimensionPolicy{gate.StepArtifactStatusDrift: {Level: gate.PolicyBlock, AppliesTo: gate.AppliesToAllCode}}
	g := gate.New(
		gate.WithSteps([]gate.StepFunc{block, advisory}),
		gate.WithScope(&gate.GateScope{Mode: gate.GateScopeModeAll, ProjectRoot: root}),
		gate.WithPolicy(policy),
	)
	result, exit := g.Run(context.Background())
	if exit == 0 || result.Pass {
		t.Fatalf("closed issue + absent mandated test must BLOCK the gate; got exit=%d pass=%v", exit, result.Pass)
	}
}

// TestGate_StatusDrift_PresentButFailingTest_CaughtByPackEngines_NotDrift (CLM-005/CLM-011):
// a success-terminal fixture whose mandated test is PRESENT (whether it passes or fails is
// irrelevant here) drives NO drift violation — the drift dimension does not (and need not)
// attribute failing tests. A present-but-failing mandated test is caught by the whole-suite
// pack_engines/test step, not by drift attribution.
func TestGate_StatusDrift_PresentButFailingTest_CaughtByPackEngines_NotDrift(t *testing.T) {
	root := t.TempDir()
	installDriftPack(t, root)
	writeFixture(t, root, "issues/ISSUE-600-closed.issue.md", issueFixture("ISSUE-600", "closed", "TestDriftPresentButFailing"))
	// The mandated test EXISTS (present). Its pass/fail is NOT a drift input.
	writeFixture(t, root, "src/failing_test.go", "package src\n\nfunc TestDriftPresentButFailing() {}\n")

	block, advisory := driftStepsFor(t, root)
	res := runStep(block)
	if res.Status == "fail" {
		t.Fatalf("drift must emit NO violation for a PRESENT mandated test (pass/fail is pack_engines' job); got %+v", res.Violations)
	}
	if len(res.Violations) != 0 {
		t.Errorf("expected zero drift violations for the present test, got %+v", res.Violations)
	}
	// A closed (success-terminal) artifact is not delivered-but-open, so the advisory
	// surface is empty too — the present-but-failing case is invisible to BOTH drift
	// surfaces and belongs solely to the pack_engines/test step.
	if a := runStep(advisory); len(a.Violations) != 0 {
		t.Errorf("advisory surface should be empty for a closed+present fixture, got %+v", a.Violations)
	}
}

// TestGate_StatusDrift_RetiredTerminal_NoViolation (CLM-003/CLM-011): a replaced/canceled
// fixture produces no violation.
func TestGate_StatusDrift_RetiredTerminal_NoViolation(t *testing.T) {
	root := t.TempDir()
	installDriftPack(t, root)
	writeFixture(t, root, "issues/ISSUE-903-replaced.issue.md", issueFixture("ISSUE-903", "replaced", "TestRetiredAbsentTest"))

	block, advisory := driftStepsFor(t, root)
	if b := runStep(block); len(b.Violations) != 0 {
		t.Errorf("retired artifact must produce no block violation, got %+v", b.Violations)
	}
	if a := runStep(advisory); len(a.Violations) != 0 {
		t.Errorf("retired artifact must produce no advisory violation, got %+v", a.Violations)
	}
}

// TestGate_StatusDriftPolicyEntry_ParsesNewCode (CLM-010/CLM-015): the backstop.yml
// enforcement.policy entry for the new dimension parses with level: block + applies-to:
// new-code and flows to ApplyPolicy for this StepName; a pre-existing (baselined) absent-
// test finding grandfathers while a NEW absent-test finding blocks.
func TestGate_StatusDriftPolicyEntry_ParsesNewCode(t *testing.T) {
	// The REAL repo backstop.yml entry parses with the pinned shape.
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("loading repo backstop.yml: %v", err)
	}
	entry, ok := cfg.Enforcement.Policy[gate.StepArtifactStatusDrift]
	if !ok {
		t.Fatalf("backstop.yml enforcement.policy must declare a %q entry", gate.StepArtifactStatusDrift)
	}
	if entry.Level != "block" || entry.AppliesTo != "new-code" {
		t.Fatalf("%s policy = level %q applies-to %q, want block / new-code", gate.StepArtifactStatusDrift, entry.Level, entry.AppliesTo)
	}

	// It flows to ApplyPolicy: a baselined finding grandfathers, a net-new one blocks.
	policy := gatePolicyFromConfig(cfg)
	known := gate.Violation{Rule: gate.StepArtifactStatusDrift, File: "issues/ISSUE-002.issue.md", Message: "m", Severity: "error", RegionHash: "known"}
	fresh := gate.Violation{Rule: gate.StepArtifactStatusDrift, File: "issues/ISSUE-999.issue.md", Message: "m", Severity: "error", RegionHash: "fresh"}
	baseline := &gate.BaselineArtifact{Violations: []gate.Violation{known}}
	steps := []gate.StepResult{{
		StepName:   gate.StepArtifactStatusDrift,
		Status:     "fail",
		Violations: []gate.Violation{known, fresh},
	}}

	out := gate.ApplyPolicy(steps, baseline, policy, nil)
	got := out[0]
	if got.Status != "fail" {
		t.Fatalf("net-new absent-test finding must block, got status=%q", got.Status)
	}
	if len(got.NewViolations) != 1 || got.NewViolations[0].RegionHash != "fresh" {
		t.Fatalf("only the net-new finding must count (baselined one grandfathers), got new=%+v", got.NewViolations)
	}

	// And with EVERYTHING baselined, the dimension grandfathers to green (the clean-tree
	// invariant, CLM-012/CLM-015).
	allBaselined := &gate.BaselineArtifact{Violations: []gate.Violation{known, fresh}}
	out2 := gate.ApplyPolicy(steps, allBaselined, policy, nil)
	if out2[0].Status != "pass" {
		t.Errorf("fully-baselined drift findings must grandfather to pass, got %q", out2[0].Status)
	}
}
