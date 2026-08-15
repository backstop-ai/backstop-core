package gate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// writeArtifact writes one fixture artifact under dir, creating parents.
func writeArtifact(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// buildRootedCorpus lays one artifact of each walked kind directly under root and
// returns the ids it wrote, so a caller can assert on resolution by id.
func buildRootedCorpus(t *testing.T, root string) {
	t.Helper()
	writeArtifact(t, filepath.Join(root, "specs"), "SPEC-701-rooted.spec.md",
		"---\nnumber: SPEC-701\nstatus: draft\n---\n\n# Rooted Spec\n")
	writeArtifact(t, filepath.Join(root, "bundles"), "BUNDLE-701-rooted.bundle.md",
		"---\nnumber: BUNDLE-701\nbundle:\n  name: rooted\nstatus:\n  maturity: exploring\n---\n\n# Rooted Bundle\n")
	writeArtifact(t, filepath.Join(root, "issues"), "ISSUE-701-rooted.issue.md",
		"---\nissue:\n  id: ISSUE-701\n  status: open\n---\n\n# Rooted Issue\n")
	writeArtifact(t, filepath.Join(root, "directives"), "DIR-701-rooted.directive.md",
		"---\nnumber: DIR-701\ndirective:\n  status: queued\n---\n\n# Rooted Directive\n")
	writeArtifact(t, filepath.Join(root, "plans"), "PLAN-SPEC-701-rooted.plan.yml",
		"plan_id: PLAN-SPEC-701\nspec_id: SPEC-701\nstatus: draft\n")
}

func recordIDs(res *gate.ArtifactStatusResolution) map[string]bool {
	out := map[string]bool{}
	for _, rec := range res.Records {
		out[rec.ID] = true
	}
	return out
}

// TestResolveArtifactStatus_WalksResolvedArtifactRoot pins CLM-043. The SIGNATURE is
// unchanged — what changed is that the argument is now the RESOLVED artifact root
// rather than the project root. The NEGATIVE arm is what makes this falsifiable:
// handing in the PROJECT root over the same tree must resolve ZERO records, which is
// exactly the behavior a .backstop/-rooted consumer gets today and the reason this
// claim exists.
func TestResolveArtifactStatus_WalksResolvedArtifactRoot(t *testing.T) {
	project := t.TempDir()
	resolvedRoot, err := artifact.ResolveRoot(project, ".backstop")
	if err == nil {
		t.Fatalf("ResolveRoot resolved %+v before .backstop existed, want a missing-root error", resolvedRoot)
	}
	artifactRoot := filepath.Join(project, ".backstop")
	buildRootedCorpus(t, artifactRoot)

	resolvedRoot, err = artifact.ResolveRoot(project, ".backstop")
	if err != nil {
		t.Fatalf("ResolveRoot: %v", err)
	}
	if resolvedRoot.Path != artifactRoot {
		t.Fatalf("resolved root %q != %q", resolvedRoot.Path, artifactRoot)
	}

	resolved, err := gate.ResolveArtifactStatus(resolvedRoot.Path)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus(<resolved root>): %v", err)
	}
	ids := recordIDs(resolved)
	for _, want := range []string{"SPEC-701", "BUNDLE-701", "ISSUE-701", "DIR-701", "PLAN-SPEC-701"} {
		if !ids[want] {
			t.Errorf("record %s was not resolved under the artifact root; resolved ids: %v", want, ids)
		}
	}

	fromProjectRoot, err := gate.ResolveArtifactStatus(project)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus(<project root>): %v", err)
	}
	if len(fromProjectRoot.Records) != 0 {
		t.Errorf("resolving from the PROJECT root over a .backstop/-rooted tree returned %d records, want 0 — if this is non-zero the walk is not rooted where the test thinks", len(fromProjectRoot.Records))
	}
}

// TestResolveArtifactStatus_TakesTypeDirectoriesFromSharedLayout pins CLM-044. The
// expected directory names are DERIVED from artifact.LayoutFor rather than written as
// literals: a test that hardcodes "specs" passes against both the shared table and a
// fourth private copy, which is precisely what this claim is about.
func TestResolveArtifactStatus_TakesTypeDirectoriesFromSharedLayout(t *testing.T) {
	walked := map[artifact.Kind]string{
		artifact.KindSpec:      "SPEC-702",
		artifact.KindBundle:    "BUNDLE-702",
		artifact.KindIssue:     "ISSUE-702",
		artifact.KindDirective: "DIR-702",
		artifact.KindPlan:      "PLAN-SPEC-702",
	}

	bodies := map[artifact.Kind]string{
		artifact.KindSpec:      "---\nnumber: SPEC-702\nstatus: draft\n---\n\n# S\n",
		artifact.KindBundle:    "---\nnumber: BUNDLE-702\nbundle:\n  name: layout\nstatus:\n  maturity: exploring\n---\n\n# B\n",
		artifact.KindIssue:     "---\nissue:\n  id: ISSUE-702\n  status: open\n---\n\n# I\n",
		artifact.KindDirective: "---\nnumber: DIR-702\ndirective:\n  status: queued\n---\n\n# D\n",
		artifact.KindPlan:      "plan_id: PLAN-SPEC-702\nspec_id: SPEC-702\nstatus: draft\n",
	}

	root := t.TempDir()
	for kind, id := range walked {
		layout, ok := artifact.LayoutFor(kind)
		if !ok {
			t.Fatalf("artifact.LayoutFor(%q) returned ok=false", kind)
		}
		writeArtifact(t, filepath.Join(root, layout.Directory), id+"-layout"+layout.Extension, bodies[kind])
	}

	resolved, err := gate.ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	ids := recordIDs(resolved)
	for kind, id := range walked {
		if !ids[id] {
			layout, _ := artifact.LayoutFor(kind)
			t.Errorf("kind %q written into the SHARED table's directory %q was not resolved; the walker is reading a private copy", kind, layout.Directory)
		}
	}
}
