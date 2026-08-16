package gate

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// TestUngatedArtifactFindings_SurviveDiffScopedFiltering pins CLM-064 and Sharp Edge 3.
//
// filterViolations keeps a violation only when it is ProjectWide or its File is in the
// diff scope, and an ungated finding names a file that is BY DEFINITION not in the diff
// — nobody just edited the stray artifact, that is why it is stray. Without the
// ProjectWide marking a bare `backstop gate` silently drops exactly the findings REQ-008
// exists to surface.
//
// The findings are driven through the PRODUCTION conversion rather than hand-built as
// Violation literals. A literal with ProjectWide:true would only prove filterViolations
// works, which is tested elsewhere; what needs proving is that the conversion SETS the
// field on every finding it produces.
func TestUngatedArtifactFindings_SurviveDiffScopedFiltering(t *testing.T) {
	project := t.TempDir()
	plantFile(t, project, "docs/SPEC-001-stray.spec.md")
	plantFile(t, project, ".backstop/bundles/BUNDLE-001-stray.bundle.md")

	root := unconfiguredRootAt(t, project)
	found, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 ungated findings, got %d (%v); the filtering assertion below would not be exercised", len(found), ungatedRelPaths(t, project, found))
	}

	violations := UngatedFindingsToViolations(found)
	if len(violations) != len(found) {
		t.Fatalf("the conversion produced %d violations for %d findings", len(violations), len(found))
	}

	// A diff scope containing NONE of the named files — the real situation, since a
	// stray artifact is precisely a file nobody just touched.
	scope := &GateScope{Mode: GateScopeModeDiff, Files: []string{"cmd/backstop/gate.go"}}
	for _, v := range violations {
		if scope.Contains(v.File) {
			t.Fatalf("the diff scope contains %s, so this test would pass without the ProjectWide marking", v.File)
		}
	}

	kept := filterViolations(scope, violations)
	if len(kept) != len(violations) {
		t.Errorf("diff-scoped filtering dropped %d of %d ungated findings; without ProjectWide a bare `backstop gate` silently discards exactly what REQ-008 exists to surface", len(violations)-len(kept), len(violations))
	}

	// The control: an otherwise identical violation that is NOT ProjectWide IS dropped,
	// so the assertion above is attributable to the marking and not to a scope that
	// keeps everything.
	notProjectWide := Violation{
		Rule:    violations[0].Rule,
		File:    violations[0].File,
		Message: violations[0].Message,
	}
	if got := filterViolations(scope, []Violation{notProjectWide}); len(got) != 0 {
		t.Errorf("a non-ProjectWide violation naming an out-of-scope file survived filtering (%d kept); the scope is not actually filtering and the assertion above proves nothing", len(got))
	}
}

// TestUngatedFindingsToViolations_MarksEveryFindingProjectWide is the direct statement
// of the property the test above depends on: the conversion sets ProjectWide on EVERY
// finding, in one place, so the marking cannot be forgotten at a call site.
func TestUngatedFindingsToViolations_MarksEveryFindingProjectWide(t *testing.T) {
	found := []UngatedArtifact{
		{Path: filepath.Join("a", "SPEC-001-x.spec.md"), Kind: "spec", ExpectedDir: "specs", Root: "."},
		{Path: filepath.Join("b", "BUNDLE-001-y.bundle.md"), Kind: "bundle", ExpectedDir: "bundles", Root: "."},
	}

	violations := UngatedFindingsToViolations(found)
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(violations))
	}
	for i, v := range violations {
		if !v.ProjectWide {
			t.Errorf("violation %d (%s) is not marked ProjectWide", i, v.File)
		}
		if v.File != found[i].Path {
			t.Errorf("violation %d names file %q, want %q", i, v.File, found[i].Path)
		}
		if v.Message == "" {
			t.Errorf("violation %d carries no message", i)
		}
	}
}
