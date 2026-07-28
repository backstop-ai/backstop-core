package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// TestSeam_CoverageStepConsumesPerFileRecordsAfterEradication proves the SPEC-041
// handoff: the SPEC-040 transitional shared go-test runner is ERADICATED (no
// newSharedTestRunner / sharedTest feed in gate.go) and coverage now consumes the
// canonical per-FILE []check.CoverageRecord PRODUCED by SPEC-042's
// dispatchPackCoverage (the coverageRecordsProducer feed) — NOT a binary-resident
// `go test` runner. The coverage step still appears in the built step list and
// runs after the cutover. This REPLACES the prior transitional-seam guard
// (CLM-028 → SPEC-041 REQ-001/REQ-002).
func TestSeam_CoverageStepConsumesPerFileRecordsAfterEradication(t *testing.T) {
	src := readFileStr(t, "gate.go")
	// The transitional shared runner must be GONE.
	if strings.Contains(src, "newSharedTestRunner") || strings.Contains(src, "sharedTest") {
		t.Fatal("the baked shared go-test runner must be ERADICATED — coverage's feed is the declared toolchain coverage pass (SPEC-041 REQ-002)")
	}
	// Coverage's feed is now the per-FILE CoverageRecord producer over the
	// dispatchPackCoverage channel.
	if !strings.Contains(src, "coverageRecordsProducer(packs, projectRoot)") {
		t.Fatal("buildCoverageStep must receive the per-FILE CoverageRecord producer (dispatchPackCoverage) over the DECLARED packs — the permanent SPEC-041 coverage feed, re-keyed off the deleted bridge by SPEC-046")
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
		t.Fatalf("coverage_threshold step missing from the built step list — it was orphaned by the cutover. steps=%v", names)
	}
}

// TestSeam_BuildBreakInUnchangedFileStillRedsDiffScopedGate proves a build break
// in an UNCHANGED file still REDs a diff-scoped gate via the PERMANENT declared
// exempt bridge (SPEC-041 REQ-004/CLM-012): go-build declares
// exempt_from_scope_filter:true, so the engine dispatch stamps ProjectWide and the
// out-of-scope build violation is NOT silently scope-filtered. The go-toolchain
// build engine produces violations on files NOT in the diff scope; with ProjectWide
// they must survive scope filtering.
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

// TestSeam_NonBuildEngineViolationsNotProjectWide proves the declared
// build-exemption is narrow: ONLY exempt (go-build) engine violations carry
// ProjectWide; lint violations (exempt_from_scope_filter:false) stay
// scope-filterable (SPEC-041 CLM-014 boundary). No CheckType/GateType identity
// drives scope — only the declared per-binding property.
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
