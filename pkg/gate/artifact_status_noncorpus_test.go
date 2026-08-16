package gate

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// TestFindUngatedArtifacts_ExcludesPackDeclaredDependencyDirs (ISSUE-122
// CLM-005/CLM-007): the ungated scan honors the INJECTED exclusion set, and the
// exclusion demonstrably comes from the declaration rather than from a surviving
// core literal — the same tree with an EMPTY injection surfaces both planted
// files.
//
// It also re-pins the asymmetry SPEC-068 REQ-008 depends on: `.backstop` ITSELF
// is still WALKED under the new signature, in both directions.
func TestFindUngatedArtifacts_ExcludesPackDeclaredDependencyDirs(t *testing.T) {
	plant := func(t *testing.T) string {
		t.Helper()
		project := t.TempDir()
		plantFile(t, project, "vendor/SPEC-901-vendor.spec.md")
		plantFile(t, project, "node_modules/some-dep/ISSUE-902-node.issue.md")
		plantFile(t, project, ".backstop/SPEC-906-reachable.spec.md")
		return project
	}

	declared := []string{"vendor/SPEC-901-vendor.spec.md", "node_modules/some-dep/ISSUE-902-node.issue.md"}

	t.Run("declared dependency dirs are excluded", func(t *testing.T) {
		project := plant(t)
		found, err := FindUngatedArtifacts(project, unconfiguredRootAt(t, project), artifact.NewNonCorpusDirs([]string{"vendor", "node_modules"}))
		if err != nil {
			t.Fatalf("FindUngatedArtifacts: %v", err)
		}
		got := ungatedRelPaths(t, project, found)
		for _, rel := range declared {
			for _, g := range got {
				if g == rel {
					t.Errorf("the scan descended into a PACK-DECLARED dependency tree and surfaced %s", rel)
				}
			}
		}
		assertDotBackstopWalked(t, got)
	})

	t.Run("without the declaration the same trees ARE walked", func(t *testing.T) {
		project := plant(t)
		found, err := FindUngatedArtifacts(project, unconfiguredRootAt(t, project), artifact.NonCorpusDirs{})
		if err != nil {
			t.Fatalf("FindUngatedArtifacts: %v", err)
		}
		got := ungatedRelPaths(t, project, found)
		for _, rel := range declared {
			present := false
			for _, g := range got {
				if g == rel {
					present = true
				}
			}
			if !present {
				t.Errorf("with NO declaration the scan still skipped %s; the exclusion must come from the pack declaration, not from a surviving core literal. findings: %v", rel, got)
			}
		}
		assertDotBackstopWalked(t, got)
	})
}

// assertDotBackstopWalked pins the half of the exclusion rule this change does
// not touch: `.backstop` itself is walked (only `.backstop/packs` beneath it is
// excluded), because excluding it wholesale removes REQ-008's own motivating
// unconfigured case.
func assertDotBackstopWalked(t *testing.T, got []string) {
	t.Helper()
	for _, g := range got {
		if g == ".backstop/SPEC-906-reachable.spec.md" {
			return
		}
	}
	t.Errorf("the scan did NOT reach an artifact-shaped file directly under .backstop/; that asymmetry is unchanged by the injected set. findings: %v", got)
}
