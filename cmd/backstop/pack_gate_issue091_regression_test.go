package main

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// ISSUE-091 dispatch-shape falsifiers. These assert WHAT backstop hands a
// non-project-wide findings engine under an all-scope — they are arg-shape
// proofs, not verdict proofs. The verdict half (that the shape change actually
// changes the reported finding set at a real engine) lives in
// pack_gate_issue091_e2e_test.go; neither file is redundant with the other.
//
// Every test here runs against BOTH declared input shapes — rule-flags
// (semgrep-shaped, `--config` per rule file) and config-file (ast-grep-shaped,
// one `--config` sgconfig). The defect is a property of the DISPATCH SHAPE, not
// of any pack or tool, so asserting it for one shape would prove half of it
// (CLM-006). The harness helpers are reused from pack_gate_scope_test.go rather
// than forked.

// issue091ManifestShape names one declared engine input shape together with the
// harness builder that produces a manifest of that shape. Adding a third shape
// is a one-line addition to issue091ManifestShapes.
type issue091ManifestShape struct {
	name  string
	build func(*testing.T) ([]*pack.Manifest, string)
}

func issue091ManifestShapes() []issue091ManifestShape {
	return []issue091ManifestShape{
		{name: "rule-flags", build: semgrepScopeManifest},
		{name: "config-file", build: astGrepLikeScopeManifest},
	}
}

// issue091AllScope resolves a REAL all-scope through the production scope
// resolver. The claim is about what gate.ComputeGateScope yields, so a
// hand-built GateScope literal would assert nothing about production.
func issue091AllScope(t *testing.T, projectRoot string) *gate.GateScope {
	t.Helper()
	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope(all): %v", err)
	}
	return scope
}

func issue091AssertSameTargets(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("scan targets = %v (%d), want %v (%d)", got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scan target[%d] = %q, want %q (got=%v want=%v)", i, got[i], want[i], got, want)
		}
	}
}

// TestIssue091_AllScopeDispatchesExplicitFileListNotProjectRoot (CLM-001): under
// a GateScopeModeAll scope resolved by the production resolver, the engine is
// pointed at the scope's OWN file list — never at the bare projectRoot
// directory. Pre-fix this fails with a single target: projectRoot.
//
// The fixture nests a plain .go file in a SUBDIRECTORY on purpose: a naive
// basename-only or top-level-only substitution would pass a flat fixture.
func TestIssue091_AllScopeDispatchesExplicitFileListNotProjectRoot(t *testing.T) {
	for _, shape := range issue091ManifestShapes() {
		t.Run(shape.name, func(t *testing.T) {
			manifests, packsDir := shape.build(t)
			projectRoot := t.TempDir()
			writeFileStr(t, filepath.Join(projectRoot, "root_test.go"), "package p\n")
			mkDirAll(t, filepath.Join(projectRoot, "sub"))
			writeFileStr(t, filepath.Join(projectRoot, "sub", "impl.go"), "package sub\n")
			writeFileStr(t, filepath.Join(projectRoot, "sub", "sub_test.go"), "package sub\n")

			allScope := issue091AllScope(t, projectRoot)
			if len(allScope.Files) == 0 {
				t.Fatalf("fixture precondition: all-scope resolved zero files under %s", projectRoot)
			}

			rec := &scopeTargetRunner{}
			if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, allScope, rec); err != nil {
				t.Fatalf("dispatchPackEngines: %v", err)
			}

			targets := rec.scanTargets(t)
			issue091AssertSameTargets(t, targets, allScope.Files)
			for _, tgt := range targets {
				if tgt == projectRoot {
					t.Fatalf("all scope must NOT hand over the bare projectRoot directory; targets=%v", targets)
				}
			}
		})
	}
}

// TestIssue091_AllScopeExcludesTestdataPaths (CLM-003): the all-scope file list
// passes through excludeTestdataPaths, so a path with a `testdata` directory
// SEGMENT is never a scan target while its non-testdata siblings are. This pins
// the over-report half — the ISSUE-040 treatment the diff scope has had since it
// shipped, now applied to both scopes instead of one.
//
// It fails pre-fix for a DIFFERENT reason than the test above: pre-fix there is
// exactly one target (the directory), so the testdata path is neither included
// nor excluded — the decision is delegated to the engine's own discovery. The
// assertion is on the CURRENT target list, not on absence alone, so it cannot
// pass vacuously against an empty list.
func TestIssue091_AllScopeExcludesTestdataPaths(t *testing.T) {
	for _, shape := range issue091ManifestShapes() {
		t.Run(shape.name, func(t *testing.T) {
			manifests, packsDir := shape.build(t)
			projectRoot := t.TempDir()
			writeFileStr(t, filepath.Join(projectRoot, "keep.go"), "package p\n")
			mkDirAll(t, filepath.Join(projectRoot, "sub"))
			writeFileStr(t, filepath.Join(projectRoot, "sub", "keep_test.go"), "package sub\n")
			mkDirAll(t, filepath.Join(projectRoot, "testdata"))
			writeFileStr(t, filepath.Join(projectRoot, "testdata", "planted.go"), "package planted\n")
			mkDirAll(t, filepath.Join(projectRoot, "sub", "testdata"))
			writeFileStr(t, filepath.Join(projectRoot, "sub", "testdata", "nested.go"), "package nested\n")

			allScope := issue091AllScope(t, projectRoot)

			rec := &scopeTargetRunner{}
			if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, allScope, rec); err != nil {
				t.Fatalf("dispatchPackEngines: %v", err)
			}

			targets := rec.scanTargets(t)
			// The exact expected list, so the subtest is falsifiable rather than a
			// bare absence check: both siblings present, both testdata paths gone.
			issue091AssertSameTargets(t, targets, []string{"keep.go", "sub/keep_test.go"})
			for _, tgt := range targets {
				if tgt == "testdata/planted.go" || tgt == "sub/testdata/nested.go" {
					t.Fatalf("a testdata-segment path must never be a scan target; targets=%v", targets)
				}
			}
		})
	}
}

// TestIssue091_AllScopeEmptyFileListAppendsNoTarget (CLM-004): a non-nil scope
// whose post-pruning file list is EMPTY appends ZERO scan targets — the engine
// scans nothing and yields nothing. It must NEVER fall through to the
// projectRoot directory target.
//
// This is the ISSUE-010 CLM-003 anti-fallback contract, now extended to cover
// all-scope, and it is the single most important guard in this file: a fallback
// here would silently restore the whole defect.
func TestIssue091_AllScopeEmptyFileListAppendsNoTarget(t *testing.T) {
	for _, shape := range issue091ManifestShapes() {
		t.Run(shape.name, func(t *testing.T) {
			manifests, packsDir := shape.build(t)
			projectRoot := t.TempDir() // deliberately empty

			allScope := issue091AllScope(t, projectRoot)
			if len(allScope.Files) != 0 {
				t.Fatalf("fixture precondition: empty projectRoot must resolve zero files, got %v", allScope.Files)
			}

			rec := &scopeTargetRunner{}
			violations, err := dispatchPackEngines(manifests, packsDir, projectRoot, allScope, rec)
			if err != nil {
				t.Fatalf("dispatchPackEngines: %v", err)
			}

			targets := rec.scanTargets(t)
			if len(targets) != 0 {
				t.Fatalf("an empty all-scope file list must yield zero scan targets, got %v", targets)
			}
			for _, tgt := range targets {
				if tgt == projectRoot {
					t.Fatalf("an empty all-scope must NOT fall back to projectRoot")
				}
			}
			if len(violations) != 0 {
				t.Errorf("an empty all-scope must yield zero findings, got %d: %#v", len(violations), violations)
			}
		})
	}
}

// TestIssue091_NilScopeStillTargetsProjectRoot (CLM-005): a nil scope has NO
// file list to substitute, so it retains exactly one scan target — projectRoot.
// That is the honest behavior for a caller that supplied no scope, not a
// grandfather clause.
//
// This test PASSES both pre-fix and post-fix BY DESIGN: it is the NEGATIVE
// CONTROL proving the fix is surgical. A fix that replaced the projectRoot
// target unconditionally would satisfy the three tests above and fail this one,
// and would break the nil-scope callers (baseline_identity_dispatch_e2e_test.go,
// the substantiveness E2E's runProductionSubstantivenessStep).
func TestIssue091_NilScopeStillTargetsProjectRoot(t *testing.T) {
	for _, shape := range issue091ManifestShapes() {
		t.Run(shape.name, func(t *testing.T) {
			manifests, packsDir := shape.build(t)
			projectRoot := t.TempDir()
			writeFileStr(t, filepath.Join(projectRoot, "impl.go"), "package p\n")

			rec := &scopeTargetRunner{}
			if _, err := dispatchPackEngines(manifests, packsDir, projectRoot, nil, rec); err != nil {
				t.Fatalf("dispatchPackEngines: %v", err)
			}

			targets := rec.scanTargets(t)
			issue091AssertSameTargets(t, targets, []string{projectRoot})
		})
	}
}
