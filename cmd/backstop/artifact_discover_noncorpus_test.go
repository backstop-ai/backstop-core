package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// plantDiscoverFile writes an artifact-shaped file at <root>/<rel>, creating
// parents. Discovery classifies by FILENAME, so the content is irrelevant here
// and the assertions stay about location.
func plantDiscoverFile(t *testing.T, root, rel string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", full, err)
	}
}

// discoveredRelPaths returns the discovered artifact paths relative to root, so
// assertions read in the fixture's own vocabulary.
func discoveredRelPaths(t *testing.T, root string, arts []DiscoveredArtifact) []string {
	t.Helper()
	out := make([]string, 0, len(arts))
	for _, a := range arts {
		rel, err := filepath.Rel(root, a.Path)
		if err != nil {
			t.Fatalf("relativizing %s against %s: %v", a.Path, root, err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

// dependencyTreeFixture plants a corpus spec plus one artifact-shaped file
// inside each of two dependency trees.
func dependencyTreeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	plantDiscoverFile(t, root, "specs/SPEC-001-corpus.spec.md")
	plantDiscoverFile(t, root, "vendor/X.spec.md")
	plantDiscoverFile(t, root, "node_modules/Y.issue.md")
	return root
}

// TestDiscoverArtifacts_ExcludesPackDeclaredDependencyDirs (ISSUE-122
// CLM-005/CLM-007): discovery honors the INJECTED exclusion set — a tree whose
// base name a pack declared under classification.dependency_dirs is not walked.
func TestDiscoverArtifacts_ExcludesPackDeclaredDependencyDirs(t *testing.T) {
	root := dependencyTreeFixture(t)

	arts, err := DiscoverArtifacts(rootAtDir(t, root), nil, artifact.NewNonCorpusDirs([]string{"vendor", "node_modules"}))
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	got := discoveredRelPaths(t, root, arts)
	for _, unwanted := range []string{"vendor/X.spec.md", "node_modules/Y.issue.md"} {
		for _, g := range got {
			if g == unwanted {
				t.Errorf("discovery descended into a PACK-DECLARED dependency tree and returned %s", unwanted)
			}
		}
	}
	if len(got) != 1 || got[0] != "specs/SPEC-001-corpus.spec.md" {
		t.Errorf("the legitimate corpus must still be discovered; got %v", got)
	}
}

// TestDiscoverArtifacts_WithoutPackDeclarationWalksDependencyTrees (ISSUE-122
// CLM-007): THE PROOF THE EXCLUSION COMES FROM THE DECLARATION. The SAME tree
// with an EMPTY injection discovers both planted files. If someone re-adds a
// `vendor`/`node_modules` default anywhere in core "for safety", this test goes
// red — which is the point: that default would move the bake rather than remove
// it.
func TestDiscoverArtifacts_WithoutPackDeclarationWalksDependencyTrees(t *testing.T) {
	root := dependencyTreeFixture(t)

	arts, err := DiscoverArtifacts(rootAtDir(t, root), nil, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	got := discoveredRelPaths(t, root, arts)
	for _, wanted := range []string{"vendor/X.spec.md", "node_modules/Y.issue.md"} {
		present := false
		for _, g := range got {
			if g == wanted {
				present = true
			}
		}
		if !present {
			t.Errorf("with NO declaration discovery still skipped %s; core must carry no ecosystem noun of its own. discovered: %v", wanted, got)
		}
	}
}

// TestDiscoverArtifacts_BackstopRulesUnchangedUnderInjection (ISSUE-122
// CLM-005): behavior preservation for the part nobody asked to change. The
// root-relative `.backstop` rules are LOCAL to this walk and are untouched by
// the injected set — `.backstop/packs` always skips, and `.backstop` skips only
// when it is not the root.
func TestDiscoverArtifacts_BackstopRulesUnchangedUnderInjection(t *testing.T) {
	injections := map[string]artifact.NonCorpusDirs{
		"with a pack declaration": artifact.NewNonCorpusDirs([]string{"vendor", "node_modules"}),
		"with the zero value":     {},
	}

	for name, set := range injections {
		t.Run(name, func(t *testing.T) {
			t.Run(".backstop skips when it is not the root, and packs beneath it always skip", func(t *testing.T) {
				root := t.TempDir()
				plantDiscoverFile(t, root, "specs/SPEC-001-corpus.spec.md")
				plantDiscoverFile(t, root, ".backstop/SPEC-002-hidden.spec.md")
				plantDiscoverFile(t, root, ".backstop/packs/org/sample/specs/SPEC-003-pack.spec.md")

				arts, err := DiscoverArtifacts(rootAtDir(t, root), nil, set)
				if err != nil {
					t.Fatalf("DiscoverArtifacts: %v", err)
				}
				got := discoveredRelPaths(t, root, arts)
				if len(got) != 1 || got[0] != "specs/SPEC-001-corpus.spec.md" {
					t.Errorf("`.backstop` must be skipped wholesale when it is not the root; discovered %v", got)
				}
			})

			t.Run(".backstop is reachable when it IS the root, but its packs tree is not", func(t *testing.T) {
				project := t.TempDir()
				backstopRoot := filepath.Join(project, ".backstop")
				plantDiscoverFile(t, project, ".backstop/specs/SPEC-001-corpus.spec.md")
				plantDiscoverFile(t, project, ".backstop/packs/org/sample/specs/SPEC-002-pack.spec.md")

				arts, err := DiscoverArtifacts(rootAtDir(t, backstopRoot), nil, set)
				if err != nil {
					t.Fatalf("DiscoverArtifacts: %v", err)
				}
				got := discoveredRelPaths(t, backstopRoot, arts)
				if len(got) != 1 || got[0] != "specs/SPEC-001-corpus.spec.md" {
					t.Errorf("a project rooted at `.backstop` must reach its own corpus while its installed packs stay excluded; discovered %v", got)
				}
			})
		})
	}
}
