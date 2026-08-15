package main

import (
	"path/filepath"
	"sort"
	"testing"
)

// TestE2E_UnconfiguredRepoRootLayoutDiscoversUnchangedCorpus pins CLM-051. The
// repo-root profile is backstop-core's own OQ-1 framework exception and the
// behavior-preservation floor for the whole change: a project that declares no
// artifact_root must discover exactly what it discovered before SPEC-068.
func TestE2E_UnconfiguredRepoRootLayoutDiscoversUnchangedCorpus(t *testing.T) {
	dir := layoutProfileDir(t, "repo-root")
	root := layoutProfileRoot(t, dir)

	if root.Configured {
		t.Fatalf("the repo-root fixture resolved a CONFIGURED root at %s; it declares no artifact_root and the framework exception is exactly that it does not have to", root.Path)
	}
	if root.Path != filepath.Clean(dir) {
		t.Errorf("an unconfigured root resolved to %s, want the project root %s", root.Path, filepath.Clean(dir))
	}

	arts, err := DiscoverArtifacts(root, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	got := discoveredRelSet(t, root.Path, arts)
	want := map[string]string{
		"specs/SPEC-001-sample.spec.md":       "spec",
		"bundles/BUNDLE-001-sample.bundle.md": "bundle",
		"plans/PLAN-SPEC-001-sample.plan.yml": "plan",
	}
	if len(got) != len(want) {
		t.Errorf("repo-root discovery found %d artifacts, want %d: %v", len(got), len(want), got)
	}
	for rel, kind := range want {
		if got[rel] != kind {
			t.Errorf("repo-root discovery: %q = %q, want %q (full set: %v)", rel, got[rel], kind, got)
		}
	}
}

// TestE2E_DotBackstopRootedProjectDiscoversEquivalentCorpus pins CLM-052, the outcome
// the init seed's REQ-004 depends on: a .backstop/-rooted project reaches the SAME
// corpus a byte-identical repo-root project does.
//
// This is an EQUALITY over (type, path-relative-to-root) after sorting, NOT a count.
// A count assertion passes when two different files are found, which would let a layout
// bug that discovers the wrong three artifacts read as success.
func TestE2E_DotBackstopRootedProjectDiscoversEquivalentCorpus(t *testing.T) {
	repoDir := layoutProfileDir(t, "repo-root")
	dotDir := layoutProfileDir(t, "dotbackstop-root")

	repoRoot := layoutProfileRoot(t, repoDir)
	dotRoot := layoutProfileRoot(t, dotDir)

	// The two profiles must genuinely differ in WHERE they root, or the equality below
	// is comparing a project against itself.
	if repoRoot.Configured || !dotRoot.Configured {
		t.Fatalf("the two profiles did not resolve to different root kinds: repo-root configured=%v, dotbackstop-root configured=%v", repoRoot.Configured, dotRoot.Configured)
	}
	if filepath.Base(dotRoot.Path) != ".backstop" {
		t.Fatalf("the dotbackstop-root profile resolved to %s, which is not a .backstop root", dotRoot.Path)
	}

	repoArts, err := DiscoverArtifacts(repoRoot, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts(repo-root): %v", err)
	}
	dotArts, err := DiscoverArtifacts(dotRoot, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts(dotbackstop-root): %v", err)
	}

	repoCorpus := sortedCorpus(t, repoRoot.Path, repoArts)
	dotCorpus := sortedCorpus(t, dotRoot.Path, dotArts)

	if len(repoCorpus) == 0 {
		t.Fatal("the repo-root profile discovered nothing, so the equality below would hold vacuously")
	}
	if len(repoCorpus) != len(dotCorpus) {
		t.Fatalf("corpus sizes differ: repo-root %d %v, dotbackstop-root %d %v", len(repoCorpus), repoCorpus, len(dotCorpus), dotCorpus)
	}
	for i := range repoCorpus {
		if repoCorpus[i] != dotCorpus[i] {
			t.Errorf("corpus entry %d differs: repo-root %q, dotbackstop-root %q", i, repoCorpus[i], dotCorpus[i])
		}
	}
}

// sortedCorpus renders a discovered set as a sorted, comparable list of
// "<type> <path-relative-to-root>" entries.
func sortedCorpus(t *testing.T, root string, arts []DiscoveredArtifact) []string {
	t.Helper()
	out := make([]string, 0, len(arts))
	for rel, kind := range discoveredRelSet(t, root, arts) {
		out = append(out, kind+" "+rel)
	}
	sort.Strings(out)
	return out
}
