package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// TestGate_UngatedBundlesUnderDotBackstopSurfacedInOutput pins CLM-065: THE
// BACKSTOP-RUNTIME CASE, END TO END.
//
// The fixture declares NO artifact_root and keeps a bundle at
// .backstop/bundles/BUNDLE-001-sample.bundle.md. That file is inside the default root
// (the project root contains itself) so a containment predicate reports nothing; CLI
// discovery skips .backstop when it is not the root so the file is never validated; and
// the non-recursive status walk reads <root>/bundles/ so it is never gated either. It is
// invisible from every direction, which is what REQ-008 exists to fix.
//
// IT DRIVES THE REAL GATE COMMAND, DIFF-SCOPED, AND ASSERTS ON RENDERED OUTPUT. Calling
// the validator directly would prove the finding is PRODUCED; it would not prove the
// finding SURVIVES filterViolations and reaches what an operator actually reads, and
// those are the two things that can independently break. A bare `backstop gate` is also
// the command an operator types — an --all run keeps the finding whether or not the
// ProjectWide marking is there, so it could not fail for the reason this claim is about.
func TestGate_UngatedBundlesUnderDotBackstopSurfacedInOutput(t *testing.T) {
	dir := layoutProfileDir(t, "unconfigured-dotbackstop-bundles")

	planted := filepath.Join(dir, ".backstop", "bundles", "BUNDLE-001-sample.bundle.md")
	if _, err := os.Stat(planted); err != nil {
		t.Fatalf("the fixture's ungated bundle is missing, so this test would pass for the wrong reason: %v", err)
	}

	root := rootAtDir(t, dir)
	if root.Configured {
		t.Fatal("the fixture resolved a CONFIGURED root; the motivating case is a project that configures nothing")
	}

	// The premise: the file really is invisible to discovery, which is what makes
	// surfacing it necessary rather than redundant.
	discovered, err := DiscoverArtifacts(root, []string{"bundle"})
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}
	for _, d := range discovered {
		if strings.Contains(filepath.ToSlash(d.Path), ".backstop/bundles/") {
			t.Fatalf("discovery reached %s; if it does, this file is not the invisible case CLM-065 is about", d.Path)
		}
	}

	// Drive the REAL command with NO --all, so the run is diff-scoped.
	out, _ := gateOutputInDir(t, dir)

	// The run really was diff-scoped — otherwise the ProjectWide marking is untested,
	// because a full sweep keeps every finding regardless. FormatHuman prints this
	// scope line ONLY for a non-ModeAll scope, so its presence is the evidence.
	if !strings.Contains(out, "changed files (use --all for full sweep)") {
		t.Fatalf("the gate did not report a diff-scoped run, so the scope filter was never exercised:\n%s", out)
	}

	// THE ASSERTION: the finding reaches RENDERED output, naming the file and saying
	// what is wrong with it.
	if !strings.Contains(out, ".backstop/bundles/BUNDLE-001-sample.bundle.md") {
		t.Errorf("the gate's rendered output does not name the ungated bundle:\n%s", out)
	}
	if !strings.Contains(out, "UNGATED") {
		t.Errorf("the gate's rendered output does not describe the file as ungated:\n%s", out)
	}
	// Actionable, not merely accusatory: it names where the walk DOES read.
	if !strings.Contains(out, filepath.Join(root.Path, "bundles")) {
		t.Errorf("the rendered finding does not name the directory the status walk reads:\n%s", out)
	}
	// UNGATED IS NOT UNDISCOVERED — the report must not overclaim.
	if strings.Contains(strings.ToLower(out), "undiscovered") {
		t.Errorf("the rendered finding describes the file as undiscovered:\n%s", out)
	}

	// It lands on the artifact_validation dimension, not somewhere incidental.
	if !strings.Contains(out, gate.StepArtifactValidation) {
		t.Errorf("the rendered output does not attribute the finding to %s:\n%s", gate.StepArtifactValidation, out)
	}
}

// TestGate_UngatedFindingWouldBeDroppedWithoutProjectWide is the falsification arm for
// the test above, and it is what makes that test's diff-scoped framing load-bearing
// rather than decorative.
//
// It takes the REAL findings the production scan produces, converts them through the
// REAL conversion, and then re-runs the production scope filter over a copy with the
// ProjectWide marking stripped — showing the finding is kept ONLY because of it. Without
// this, the end-to-end test above would still pass if filterViolations happened to keep
// everything for some unrelated reason.
func TestGate_UngatedFindingWouldBeDroppedWithoutProjectWide(t *testing.T) {
	dir := layoutProfileDir(t, "unconfigured-dotbackstop-bundles")
	root := rootAtDir(t, dir)

	found, err := gate.FindUngatedArtifacts(dir, root)
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("the scan surfaced nothing over the fixture, so this test would be vacuous")
	}

	marked := gate.UngatedFindingsToViolations(found)

	// A diff scope naming a file that is NOT the ungated bundle — the real situation,
	// since a stray artifact is by definition a file nobody just edited.
	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"cmd/backstop/gate.go"}}
	for _, v := range marked {
		if scope.Contains(v.File) {
			t.Fatalf("the diff scope contains %s, so the comparison below proves nothing", v.File)
		}
	}

	if kept := scope.FilterViolations(marked); len(kept) != len(marked) {
		t.Errorf("diff-scoped filtering dropped %d of %d ProjectWide ungated findings", len(marked)-len(kept), len(marked))
	}

	// Strip the marking and the SAME findings vanish.
	stripped := make([]gate.Violation, 0, len(marked))
	for _, v := range marked {
		v.ProjectWide = false
		stripped = append(stripped, v)
	}
	if kept := scope.FilterViolations(stripped); len(kept) != 0 {
		t.Errorf("%d ungated findings survived diff-scoped filtering WITHOUT the ProjectWide marking; the marking is then not what keeps them and the claim is untested", len(kept))
	}
}
