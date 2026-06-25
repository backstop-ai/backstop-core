package main

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// TestSeam_CoverageStepNotOrphanedByStep2Deletion proves deleting the
// pkg/check.Run Step-2 consumption does NOT orphan the still-baked coverage step
// (CLM-028, Sharp Edge 1): the shared go-test runner SURVIVES as the transitional
// coverage feed, buildCoverageStep still receives its whole-module feed, and the
// coverage step still appears in the built step list and runs after the cutover.
func TestSeam_CoverageStepNotOrphanedByStep2Deletion(t *testing.T) {
	// Source guard: newSharedTestRunner must SURVIVE (it is the transitional
	// coverage feed) and must still feed buildCoverageStep.
	src := readFileStr(t, "gate.go")
	if !strings.Contains(src, "newSharedTestRunner(projectRoot)") {
		t.Fatal("the shared test runner must survive the Step-2 deletion as the transitional coverage feed (CLM-028)")
	}
	if !strings.Contains(src, "buildCoverageStep(specDir, projectRoot, activeScope, sharedTest)") {
		t.Fatal("buildCoverageStep must still receive the shared go-test runner feed after the cutover (CLM-028)")
	}

	// Behavioral: the coverage step is still present in the built step list.
	root := goToolchainProjectRoot(t)
	names := gateStepNames(t, root, emptyDiffScope())
	found := false
	for _, n := range names {
		if n == gate.StepCoverageThreshold {
			found = true
		}
	}
	if !found {
		t.Fatalf("coverage_threshold step missing from the built step list — it was orphaned by the Step-2 deletion. steps=%v", names)
	}
}

// TestSeam_BuildBreakInUnchangedFileStillRedsDiffScopedGate proves a build break
// in an UNCHANGED file still REDs a diff-scoped gate across the cutover window
// (CLM-029, Sharp Edge 2): build-pass ProjectWide is preserved transitionally on
// the engine dispatch path, so engine-path build violations are NOT silently
// scope-filtered. The go-toolchain build engine produces violations on files NOT
// in the diff scope; with ProjectWide they must survive scope filtering.
func TestSeam_BuildBreakInUnchangedFileStillRedsDiffScopedGate(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	stubSandboxedRunStdout(t, nil)
	// The build fixture reports errors in pkg/widget/widget.go and
	// pkg/gadget/gadget.go — none of which are in the diff scope below.
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (build): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected build violations from the convert")
	}
	// Every build-pass engine violation must carry ProjectWide=true — the
	// structural exemption pkg/gate/scope.go's filterViolations keys on to keep an
	// unchanged-file build break out of the diff-scope filter (Ratified Design
	// Constraint 3). The breaking files (pkg/widget, pkg/gadget) are deliberately
	// NOT in any diff scope, so without ProjectWide they would be silently filtered.
	for _, v := range violations {
		if !v.ProjectWide {
			t.Fatalf("build-pass engine violation %q (%s) must carry ProjectWide=true so an unchanged-file build break is not scope-filtered (CLM-029)", v.Message, v.File)
		}
	}
}

// TestSeam_NonBuildEngineViolationsNotProjectWide proves the transitional
// build-exemption is narrow: ONLY build-pass engine violations carry ProjectWide;
// lint/findings violations stay scope-filterable (CLM-029 boundary). Mirrors the
// legacy `cv.Pass == check.CheckTypeBuild`-only exemption.
func TestSeam_NonBuildEngineViolationsNotProjectWide(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "golangci")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"golangci-lint": readFixture(t, "golangci-v2.sarif")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (lint): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected lint violations")
	}
	for _, v := range violations {
		if v.ProjectWide {
			t.Errorf("lint-pass engine violation %q must NOT carry ProjectWide — only build-pass is exempt (CLM-029)", v.Message)
		}
	}
}

var _ = []*pack.Manifest(nil) // pack import is used via onlyRules/dispatchPackEngines
