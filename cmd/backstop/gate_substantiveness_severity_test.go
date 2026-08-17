package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// gate_substantiveness_severity_test.go — ISSUE-106 hop B. Once the join FORWARDS a
// pack's declared severity (pkg/gate/substantiveness_join.go), this step is where it was
// about to be thrown away again: buildTestSubstantivenessStep reached its verdict by RAW
// COUNT, so a preserved `warning` still flipped the step to "fail" and still exited the
// gate 1. Hop A alone would have been invisible to every user.
//
// TWO ALTITUDES, ON PURPOSE, AND NEITHER REPLACES THE OTHER. The e2e is the honest proof
// — a real pack declaring `severity: warning`, real ast-grep, real convert, real install
// — but it sits behind a fail-loud ast-grep guard and therefore cannot be the only
// carrier of the claim. The seam test always runs. An engine-guarded test that silently
// does not run is the vacuous green this repo says it will not ship.

// warningSeverityHollowSeamFinding is the hollow-role finding the dispatch seam is
// stubbed to return: the shape RouteSubstantivenessFindings partitions on (the
// pack-declared substantiveness_role property, ISSUE-064) carrying the declared severity
// under test.
func warningSeverityHollowSeamFinding(testFile, severity string) gate.Violation {
	return gate.Violation{
		Rule:     substantivenessHollowNamespacedID(),
		File:     testFile,
		Line:     5,
		Message:  "test function TestSubjectHollow has no assertions (hollow) func=TestSubjectHollow",
		Severity: severity,
		Properties: map[string]string{
			"substantiveness_role": "hollow",
			"func":                 "TestSubjectHollow",
		},
	}
}

// runSubstantivenessStepWithInjectedSeverity drives the REAL buildTestSubstantivenessStep
// through the EXISTING seams (resolveSubstantivenessPacksFn + dispatchPackEnginesFn), with
// the dispatcher stubbed to return exactly ONE hollow-role finding at the given severity.
//
// ★ THE MISSING writeMandatedTestFile CALL IS DELIBERATE AND LOAD-BEARING. Every shipped
// test in gate_substantiveness_wiring_test.go makes that call; copying it here would
// silently destroy this test's entire proof, so do NOT "restore" it for symmetry. Trace
// it: writeSubstantivenessSpec mandates TestSubjectHollow against target package "gate";
// with the test file on disk ResolveMandatedTestPaths resolves mt.FilePath, pass 1 counts
// the test as join-eligible, and pass 2 raises a noTarget violation for it (nothing
// populates the referenced set — dispatch is stubbed to return only the hollow finding).
// That noTarget violation is a fixed-"error" SYNTHESIZED violation (CLM-003, by design),
// so it BLOCKS: two violations reach the verdict, the step reads "fail" even AFTER both
// hops land, and this test would fail for a reason unrelated to the fix.
//
// OMITTING the call leaves mt.FilePath == "", which pass 1 skips with `continue` BEFORE
// the eligibility check, so no noTarget violation is ever raised. The hollow violation is
// unaffected: HollowFindingsToViolations converts routed hollow findings directly and
// never consults the resolved mandated tests. The refusal cannot fire either, on two
// independent grounds — SubstantivenessEvidenceRefusal requires eligible >= 1 (here 0)
// AND hollow == 0 (here 1).
//
// THE E2E BELOW USES A DIFFERENT REMEDY FOR THE SAME SHARP EDGE, and that is correct
// rather than inconsistent: at e2e altitude the mandated test file MUST exist for the
// real engine to scan it, so the noTarget set-join is satisfied through the FIXTURE BODY
// (`gate.Build()`) instead. Here the findings are injected at the seam, so there is
// nothing for a fixture body to satisfy — the partition comes from the stub's return
// value, not from parsing. The `gate.Build()` trick has no effect at this seam.
func runSubstantivenessStepWithInjectedSeverity(t *testing.T, severity string) gate.StepResult {
	t.Helper()
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSubstantivenessSpec(t, specDir)
	injectSubstantivenessManifest(t)

	hollowPath := filepath.Join(codeDir, "subject_test.go")
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
		return []gate.Violation{warningSeverityHollowSeamFinding(hollowPath, severity)}, nil
	}

	classifier, matcher := goSubstDiscovery(t)
	step := buildTestSubstantivenessStep(specDir, codeDir, codeDir, nil, classifier, matcher)
	return step(context.Background())
}

// TestSubstantivenessStep_WarningOnlyResultDoesNotBlock (CLM-002) — a substantiveness
// result whose only finding is a pack-declared `warning` reports "warning", not "fail",
// and leaves the gate passing.
//
// The assertions are ordered so a reader can tell WHICH hop broke: assertion 1 is hop A
// (the severity arriving intact at the step), assertion 2 is hop B (the step's verdict),
// assertion 3 is the user-visible outcome — the status string is not the exit code.
func TestSubstantivenessStep_WarningOnlyResultDoesNotBlock(t *testing.T) {
	res := runSubstantivenessStepWithInjectedSeverity(t, "warning")

	// The role property is what RouteSubstantivenessFindings partitions on (ISSUE-064); a
	// finding without it is silently ignored and every assertion below would pass
	// vacuously over an empty result. So the count comes FIRST.
	if len(res.Violations) != 1 {
		t.Fatalf("expected exactly ONE substantiveness violation (the routed hollow finding), got %d: %#v",
			len(res.Violations), res.Violations)
	}
	if got := res.Violations[0].Severity; got != "warning" {
		t.Errorf("hop A: the violation reaching the step carries severity %q, want %q; the join is "+
			"still overwriting the pack's declaration", got, "warning")
	}
	if res.Status != "warning" {
		t.Errorf("hop B: a warning-only substantiveness result must report status %q, got %q; a raw "+
			"count answers \"are there findings\", not \"does anything here block\"", "warning", res.Status)
	}

	// The exit-relevant half. A status that renders correctly but still flips the gate is
	// the defect the operator actually feels.
	summary := gate.NewGateResult([]gate.StepResult{res})
	if !summary.Pass {
		t.Error("a warning-only substantiveness step flipped GateResult.Pass; the gate would exit 1 " +
			"on a finding the pack itself declared non-blocking")
	}
	if summary.StepsWarned != 1 {
		t.Errorf("expected the step to be counted in StepsWarned, got %d", summary.StepsWarned)
	}
	if summary.StepsFailed != 0 {
		t.Errorf("expected no failed steps, got %d", summary.StepsFailed)
	}

	// THE CONTROL. Without it, "does not block" is indistinguishable from "the step
	// stopped working": the SAME injected finding at severity "error" must still fail.
	t.Run("error_severity_still_blocks", func(t *testing.T) {
		blocking := runSubstantivenessStepWithInjectedSeverity(t, "error")
		if len(blocking.Violations) != 1 {
			t.Fatalf("expected exactly ONE substantiveness violation, got %d: %#v",
				len(blocking.Violations), blocking.Violations)
		}
		if blocking.Status != "fail" {
			t.Errorf("an error-severity substantiveness finding must still report %q, got %q",
				"fail", blocking.Status)
		}
		if gate.NewGateResult([]gate.StepResult{blocking}).Pass {
			t.Error("an error-severity substantiveness finding must still fail the gate")
		}
	})
}

// installWarningSeveritySubstantivenessPack installs a PATCHED COPY of
// packs/substantiveness through the SAME real newProductionAddCommand() path
// installSubstantivenessLocalPack and installZeroMatchSubstantivenessPack use. The copy's
// Q1 hollow rule gains a `severity: warning` declaration and nothing else changes.
//
// VERIFIED AGAINST REAL ast-grep 0.43.0: that one appended line makes ast-grep report
// `severity=warning` in its --json output, which the pack's own to-sarif.sh maps to SARIF
// `level: "warning"` via its explicit warning branch. The in-repo pack source is NEVER
// mutated — editing packs/substantiveness from a test would corrupt this repo's own
// dogfood install.
func (w *e2eWorkspace) installWarningSeveritySubstantivenessPack(repoRoot string) error {
	// Beside w.root, never inside it: a pack source tree inside the workspace would be
	// swept into the engine's own scan targets.
	packCopy, err := os.MkdirTemp(filepath.Dir(w.root), "warnsev-pack-")
	if err != nil {
		return fmt.Errorf("creating the warning-severity pack copy dir: %w", err)
	}
	if err := copyPackSourceTree(substantivenessSourceDir(repoRoot), packCopy); err != nil {
		return fmt.Errorf("copying the substantiveness pack source: %w", err)
	}

	rulePath := filepath.Join(packCopy, "ast-grep", "rules", "hollow-test-go.yml")
	pristine, err := os.ReadFile(rulePath)
	if err != nil {
		return fmt.Errorf("reading the hollow rule for patching: %w", err)
	}
	patched := string(pristine) + "\n# ISSUE-106 FIXTURE PATCH: the hollow rule DECLARES its finding non-blocking.\n" +
		"# ast-grep reports severity=warning, to-sarif.sh maps it to SARIF level: warning.\n" +
		"severity: warning\n"
	if err := os.WriteFile(rulePath, []byte(patched), 0o644); err != nil {
		return fmt.Errorf("writing the warning-severity hollow rule: %w", err)
	}

	add, err := newProductionAddCommand()
	if err != nil {
		return fmt.Errorf("assembling the pack add command: %w", err)
	}
	res, err := add.Run(packCopy, distribution.AddOptions{ProjectDir: w.root})
	if err != nil {
		return fmt.Errorf("installing the warning-severity substantiveness pack: %w", err)
	}
	w.installed = true
	w.installInfo = res
	return nil
}

// newSeverityE2EWorkspace scaffolds the shipped newE2EWorkspace and then OVERWRITES its
// hollow fixture body.
//
// ★ THE FIXTURE BODY IS LOAD-BEARING, FOR TWO INDEPENDENT REASONS. The shipped body is
// `doSubject()`, an UNQUALIFIED call that produces no Q2 extraction finding, so the
// mandated test's referenced-symbol set comes up empty and the step raises a noTarget
// violation ALONGSIDE the hollow one. That noTarget violation is a fixed-"error"
// SYNTHESIZED violation (CLM-003, by design) and it BLOCKS — masking this whole proof
// behind a step that reads "fail" for an unrelated reason.
//
// `gate.Build()` is package-QUALIFIED, so the pack's Q2 rule extracts symbol=gate, the
// noTarget set-join is SATISFIED, and the hollow finding is the ONLY violation. The test
// stays genuinely hollow because `Build` matches none of the hollow rule's assertion
// vocabulary (require|assert|check|verify|expect|must|Fatal|Error|Fail|Skip). A future
// reader who tidies this back to `doSubject()` will reintroduce a baffling red.
func newSeverityE2EWorkspace(t *testing.T) *e2eWorkspace {
	t.Helper()
	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding the severity e2e workspace: %v", err)
	}
	body := "package sample_test\n\nimport \"testing\"\n\n" +
		"func TestE2EHollowSubject(t *testing.T) {\n\tgate.Build()\n}\n"
	if err := os.WriteFile(ws.hollowFile, []byte(body), 0o644); err != nil {
		t.Fatalf("overwriting the severity e2e hollow fixture: %v", err)
	}
	return ws
}

// assertSoleHollowViolation fails with the FULL violation dump unless the result carries
// exactly one violation and it is the hollow one. Two violations means the fixture
// drifted (see newSeverityE2EWorkspace) — soldiering on would assert severity against
// whichever violation happened to land first.
func assertSoleHollowViolation(t *testing.T, res gate.StepResult) gate.Violation {
	t.Helper()
	if len(res.Violations) != 1 {
		t.Fatalf("expected EXACTLY ONE violation (the hollow finding); got %d — the fixture has "+
			"drifted, most likely back to an unqualified test body that re-raises noTarget: %s",
			len(res.Violations), renderViolations(res.Violations))
	}
	v := res.Violations[0]
	if v.Rule != gate.StepTestSubstantiveness || !strings.Contains(v.Message, "has no assertions (hollow)") {
		t.Fatalf("expected the single violation to be the hollow one; got %s", renderViolations(res.Violations))
	}
	return v
}

// TestE2E_SubstantivenessWarningPack_StepWarnsWithoutBlocking (CLM-004) — through the
// REAL installed-pack path, a substantiveness pack whose hollow rule declares
// `severity: warning` produces a step that DOES NOT block.
//
// The chain this exercises end to end: ast-grep → the pack's to-sarif.sh →
// check.ParsePackFindings → the cmd/backstop/pack_gate.go bridge → the role routing → the
// join → the step verdict. Every hop had to preserve the declaration.
//
// ast-grep absence is a t.Fatal via requireAstGrepE2E, NEVER a t.Skip.
func TestE2E_SubstantivenessWarningPack_StepWarnsWithoutBlocking(t *testing.T) {
	requireAstGrepE2E(t)

	ws := newSeverityE2EWorkspace(t)
	if err := ws.installWarningSeveritySubstantivenessPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the warning-severity local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()
	v := assertSoleHollowViolation(t, res)

	if v.Severity != "warning" {
		t.Errorf("the pack's declared severity did not survive the real chain: violation severity "+
			"= %q, want %q (ast-grep → to-sarif.sh → ParsePackFindings → the pack_gate bridge → "+
			"route → join)", v.Severity, "warning")
	}
	if res.Status != "warning" {
		t.Errorf("a warning-only substantiveness result must report status %q, got %q with %s",
			"warning", res.Status, renderViolations(res.Violations))
	}
	summary := gate.NewGateResult([]gate.StepResult{res})
	if !summary.Pass {
		t.Error("a pack-declared warning flipped GateResult.Pass; the gate would exit 1 on a finding " +
			"the pack itself declared non-blocking")
	}
	if summary.StepsWarned != 1 || summary.StepsFailed != 0 {
		t.Errorf("expected StepsWarned=1 StepsFailed=0, got StepsWarned=%d StepsFailed=%d",
			summary.StepsWarned, summary.StepsFailed)
	}
}

// TestE2E_SubstantivenessPristinePack_StepStillBlocks (CLM-004) — THE CONTROL, and what
// makes the warning result above meaningful. The SAME workspace and the SAME fixture
// body, installed with the UNPATCHED pack, still blocks.
//
// It proves three things at once: the harness CAN go red, the only difference is the
// pack's one-line declaration, and the measured blast radius on backstop-core is genuinely
// zero — the SHIPPED hollow rule declares no `severity:` key, so ast-grep reports `hint`,
// to-sarif.sh's else branch maps that to `error`, and every substantiveness violation this
// repo produces keeps blocking after both hops.
func TestE2E_SubstantivenessPristinePack_StepStillBlocks(t *testing.T) {
	requireAstGrepE2E(t)

	ws := newSeverityE2EWorkspace(t)
	if err := ws.installSubstantivenessLocalPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the pristine local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()
	v := assertSoleHollowViolation(t, res)

	if v.Severity != "error" {
		t.Errorf("the SHIPPED hollow rule declares no severity, so the chain must yield %q; got %q. "+
			"If this ever reports \"warning\", the measured zero blast radius on backstop-core's own "+
			"dogfood is stale", "error", v.Severity)
	}
	if res.Status != "fail" {
		t.Errorf("an error-severity hollow finding must still FAIL the step, got %q with %s",
			res.Status, renderViolations(res.Violations))
	}
	if gate.NewGateResult([]gate.StepResult{res}).Pass {
		t.Error("an error-severity hollow finding must still fail the gate")
	}
}
