package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// typedRefKinds binds each typed-ref PREFIX to the artifact Kind whose layout it
// resolves through. The prefixes are the resolved-by GRAMMAR's accepted vocabulary —
// a validation concern — while the directory and extension are LAYOUT, which is the
// split this test exists to pin.
func typedRefKinds() map[string]artifact.Kind {
	return map[string]artifact.Kind{
		"BUNDLE": artifact.KindBundle,
		"SPEC":   artifact.KindSpec,
		"ISSUE":  artifact.KindIssue,
		"PLAN":   artifact.KindPlan,
		"DIR":    artifact.KindDirective,
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// TestResolvedBy_TakesTypeDirectoriesFromSharedLayout pins CLM-047: the directory and
// the extension come from artifact.LayoutFor for ALL FIVE typed-ref prefixes, and
// resolved_by carries no private copy of either. Both the fixture layout AND the
// expectation are DERIVED from the shared table, so a surviving private copy that
// drifts goes red here.
//
// DIR→directives is included explicitly: it is the prefix that defeats any naive
// lowercase coercion, and the one most likely to be silently dropped.
func TestResolvedBy_TakesTypeDirectoriesFromSharedLayout(t *testing.T) {
	root := t.TempDir()

	// The citing issue lives in its own type directory, since resolution is anchored
	// on the CITING artifact's source path.
	issueLayout, ok := artifact.LayoutFor(artifact.KindIssue)
	if !ok {
		t.Fatal("artifact.LayoutFor(issue) returned ok=false")
	}
	citingPath := filepath.Join(root, issueLayout.Directory, "ISSUE-800-citer"+issueLayout.Extension)
	mustWrite(t, citingPath)

	for prefix, kind := range typedRefKinds() {
		layout, layoutOK := artifact.LayoutFor(kind)
		if !layoutOK {
			t.Fatalf("artifact.LayoutFor(%q) returned ok=false", kind)
		}
		ref := prefix + "-801"
		mustWrite(t, filepath.Join(root, layout.Directory, ref+"-target"+layout.Extension))
	}

	citing := &artifact.ParsedArtifact{Filename: filepath.Base(citingPath), SourcePath: citingPath}

	for prefix := range typedRefKinds() {
		ref := prefix + "-801"
		if violations := validateResolvedBy(citing, ref); len(violations) != 0 {
			t.Errorf("resolved-by %q did not resolve against the SHARED layout table: %v", ref, violations[0].Message)
		}
	}

	// A ref whose target does not exist still fails loudly — otherwise the assertions
	// above would pass for an implementation that resolves everything.
	if violations := validateResolvedBy(citing, "SPEC-899"); len(violations) == 0 {
		t.Error("resolved-by SPEC-899 resolved even though no such artifact exists")
	}
}

// TestResolvedBy_ResolvesUnderNonRootLayout pins CLM-048 and is the REGRESSION GUARD
// for Sharp Edge 4: typedRefArtifactExists anchors on the CITING artifact's own
// SourcePath, so it ALREADY works under a .backstop/ layout. The bundle's phrasing
// calls this file a root hardcoding; it is not. Feeding it a configured ROOT would
// break this test.
func TestResolvedBy_ResolvesUnderNonRootLayout(t *testing.T) {
	project := t.TempDir()
	nested := filepath.Join(project, ".backstop")

	issueLayout, _ := artifact.LayoutFor(artifact.KindIssue)
	specLayout, _ := artifact.LayoutFor(artifact.KindSpec)

	citingPath := filepath.Join(nested, issueLayout.Directory, "ISSUE-802-nested"+issueLayout.Extension)
	mustWrite(t, citingPath)
	mustWrite(t, filepath.Join(nested, specLayout.Directory, "SPEC-802-nested"+specLayout.Extension))

	citing := &artifact.ParsedArtifact{Filename: filepath.Base(citingPath), SourcePath: citingPath}

	if violations := validateResolvedBy(citing, "SPEC-802"); len(violations) != 0 {
		t.Errorf("a typed ref failed to resolve under a .backstop/ layout: %v", violations[0].Message)
	}

	// The sibling anchoring is what makes that work: a spec placed at the PROJECT root
	// rather than beside the citing issue must NOT resolve.
	otherProject := t.TempDir()
	strayCitingPath := filepath.Join(otherProject, ".backstop", issueLayout.Directory, "ISSUE-803-nested"+issueLayout.Extension)
	mustWrite(t, strayCitingPath)
	mustWrite(t, filepath.Join(otherProject, specLayout.Directory, "SPEC-803-stray"+specLayout.Extension))

	strayCiting := &artifact.ParsedArtifact{Filename: filepath.Base(strayCitingPath), SourcePath: strayCitingPath}
	if violations := validateResolvedBy(strayCiting, "SPEC-803"); len(violations) == 0 {
		t.Error("a spec outside the citing artifact's sibling directory resolved; the anchoring is not source-path relative")
	}
}
