package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// ISSUE-146 falsification suite. TestPackNew_ScaffoldPassesCheckAndTest already
// asserts that a freshly scaffolded pack PASSES `pack test`. On its own that
// assertion is satisfiable by a pack that CANNOT FAIL — which was precisely the
// defect: the scaffolded validator was `exit 0` and the two fixtures differed only
// in a comment, so phase 3's positive/negative classification was unfalsifiable.
//
// These two tests make that green mean something. Each mutates one scaffolded
// fixture and requires phase 3 to go RED, in opposite directions: removing the
// marker from the negative fixture must trip `validator-negative`, and planting it
// in the positive fixture must trip `validator-positive`. Together they pin that the
// pass is EARNED by fixture content rather than structural.
//
// ⛔ NO runtime.GOOS SKIP GATE. The neighbouring TestPackAuthoringLoop_EndToEnd
// skips off darwin and would be the natural thing to copy — copying it here would be
// a defect. Both platforms have a real sandbox (pkg/packval/sandbox_linux.go:
// Landlock + seccomp; pkg/packval/sandbox_nonlinux.go: sandbox-exec), the sibling
// TestPackNew_ScaffoldPassesCheckAndTest already runs on both unguarded, and these
// are this lane's ONLY proof that the green is earned. A platform gate would make
// that proof silently dark on Linux CI.

// scaffoldPackForFalsification scaffolds an engine pack through the real `pack new`
// command and returns its ABSOLUTE pack directory.
//
// ⚠ ABSOLUTE, ALWAYS. A relative packDir triggers ISSUE-147 on darwin: sandbox-exec
// refuses the profile outright, EVERY validator run reports not-passed, and both
// tests below would then "pass" for entirely the wrong reason while asserting nothing
// about fixture content. That is exactly why TestPackAuthoringLoop_EndToEnd is red.
func scaffoldPackForFalsification(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()
	code, out := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "engine", "--language", "go", "--slug", "sample-check"},
		projectDir: projectDir,
	})
	if code != 0 {
		t.Fatalf("pack new failed with exit %d: %s", code, out)
	}
	return filepath.Join(projectDir, "sample-check")
}

// runScaffoldedPackTest runs the REAL packval pipeline in test mode against packDir.
func runScaffoldedPackTest(packDir string) *packval.Result {
	return packval.NewPipeline(packDir, packval.PipelineOptions{Mode: "test"}).Run()
}

// pipelineHasCheck reports whether any pipeline error carries the given Check.
func pipelineHasCheck(res *packval.Result, check string) bool {
	for _, e := range res.Errors {
		if e.Check == check {
			return true
		}
	}
	return false
}

// scaffoldedFixturePaths returns the scaffolded positive and negative fixture paths.
func scaffoldedFixturePaths(packDir string) (positive, negative string) {
	return filepath.Join(packDir, "fixtures", "valid", "example.txt"),
		filepath.Join(packDir, "fixtures", "invalid", "example.txt")
}

// TestPackNew_ScaffoldedPackTestFailsWhenNegativeFixtureLosesTheMarker proves the
// scaffolded pack's `pack test` pass is earned in the negative direction (CLM-003):
// strip what makes the negative fixture violating and phase 3 must report
// `validator-negative`.
//
// The mutation is DERIVED, never hardcoded — the negative fixture is overwritten with
// the POSITIVE fixture's own bytes, which are by construction marker-free. A hardcoded
// copy of the marker literal would be a second source of truth that silently rots when
// the scaffolder changes.
//
// The Check is pinned, not just the status: Pipeline.Run stops at the first failing
// phase, so an unrelated phase-1/phase-2 defect would also produce a non-pass while
// phase 3 never ran at all.
func TestPackNew_ScaffoldedPackTestFailsWhenNegativeFixtureLosesTheMarker(t *testing.T) {
	packDir := scaffoldPackForFalsification(t)
	if res := runScaffoldedPackTest(packDir); res.Status != "pass" {
		t.Fatalf("freshly scaffolded pack must pass `pack test` before mutation; got %q, errors=%+v", res.Status, res.Errors)
	}

	positive, negative := scaffoldedFixturePaths(packDir)
	clean, err := os.ReadFile(positive)
	if err != nil {
		t.Fatalf("reading scaffolded positive fixture: %v", err)
	}
	if err := os.WriteFile(negative, clean, 0o644); err != nil {
		t.Fatalf("neutralising the negative fixture: %v", err)
	}

	res := runScaffoldedPackTest(packDir)
	if res.Status != "fail" {
		t.Fatalf("pack test = %q after neutralising the negative fixture, want fail — the scaffolded pack's green is not earned; errors=%+v", res.Status, res.Errors)
	}
	if !pipelineHasCheck(res, "validator-negative") {
		t.Errorf("expected a validator-negative error after neutralising the negative fixture; got %+v", res.Errors)
	}
}

// TestPackNew_ScaffoldedPackTestFailsWhenPositiveFixtureGainsTheMarker is the mirror
// (CLM-003), and it is the direction that catches a validator which fires on
// everything: plant the violating content in the clean fixture and phase 3 must report
// `validator-positive`.
//
// The mutation is DERIVED — the negative fixture's own bytes are appended to the
// positive one — so it carries whatever marker the scaffolder currently writes.
func TestPackNew_ScaffoldedPackTestFailsWhenPositiveFixtureGainsTheMarker(t *testing.T) {
	packDir := scaffoldPackForFalsification(t)
	if res := runScaffoldedPackTest(packDir); res.Status != "pass" {
		t.Fatalf("freshly scaffolded pack must pass `pack test` before mutation; got %q, errors=%+v", res.Status, res.Errors)
	}

	positive, negative := scaffoldedFixturePaths(packDir)
	violating, err := os.ReadFile(negative)
	if err != nil {
		t.Fatalf("reading scaffolded negative fixture: %v", err)
	}
	clean, err := os.ReadFile(positive)
	if err != nil {
		t.Fatalf("reading scaffolded positive fixture: %v", err)
	}
	if err := os.WriteFile(positive, append(clean, violating...), 0o644); err != nil {
		t.Fatalf("planting the marker in the positive fixture: %v", err)
	}

	res := runScaffoldedPackTest(packDir)
	if res.Status != "fail" {
		t.Fatalf("pack test = %q after planting the marker in the positive fixture, want fail — the validator does not discriminate; errors=%+v", res.Status, res.Errors)
	}
	if !pipelineHasCheck(res, "validator-positive") {
		t.Errorf("expected a validator-positive error after planting the marker in the positive fixture; got %+v", res.Errors)
	}
}
