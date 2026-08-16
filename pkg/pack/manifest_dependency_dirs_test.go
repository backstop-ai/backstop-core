package pack

import "testing"

// TestManifest_ClassificationDependencyDirsParse (ISSUE-122 CLM-002): a pack.yml
// top-level `classification:` block with a `dependency_dirs:` list parses onto
// pack.Manifest.Classification.DependencyDirs in DECLARED ORDER. These are the
// bare directory NAMES the declaring pack's ecosystem uses for vendored or
// installed dependency trees; core excludes them from artifact-corpus walks
// ONLY because a pack declares them.
func TestManifest_ClassificationDependencyDirsParse(t *testing.T) {
	src := classificationManifestPrefix + `classification:
  source:
    - "**/*.go"
  dependency_dirs:
    - "vendor"
    - "_thirdparty"
`
	m, err := ParseManifest([]byte(src))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	want := []string{"vendor", "_thirdparty"}
	if len(m.Classification.DependencyDirs) != len(want) {
		t.Fatalf("DependencyDirs len = %d, want %d (%#v)", len(m.Classification.DependencyDirs), len(want), m.Classification.DependencyDirs)
	}
	for i, w := range want {
		if m.Classification.DependencyDirs[i] != w {
			t.Errorf("DependencyDirs[%d] = %q, want %q (declared order)", i, m.Classification.DependencyDirs[i], w)
		}
	}
	// The sibling globs are unaffected by the new key.
	if len(m.Classification.Source) != 1 || m.Classification.Source[0] != "**/*.go" {
		t.Errorf("Source must be unchanged by the dependency_dirs key, got %#v", m.Classification.Source)
	}
}

// TestManifest_ClassificationAbsentYieldsNoDependencyDirs (ISSUE-122 CLM-002):
// the ADDITIVE-PARSE proof. A manifest with NO `classification:` block, and one
// whose `classification:` block declares only source/test, both parse
// successfully with an EMPTY DependencyDirs and with source/test unchanged.
// ParseManifestFile uses non-strict yaml.Unmarshal, so an existing manifest
// authored before this field existed must be entirely unaffected.
func TestManifest_ClassificationAbsentYieldsNoDependencyDirs(t *testing.T) {
	t.Run("no classification block at all", func(t *testing.T) {
		m, err := ParseManifest([]byte(classificationManifestPrefix))
		if err != nil {
			t.Fatalf("a manifest with no classification block must parse without error, got: %v", err)
		}
		if len(m.Classification.DependencyDirs) != 0 {
			t.Errorf("absent block must yield empty DependencyDirs, got %#v", m.Classification.DependencyDirs)
		}
	})

	t.Run("classification block with only source and test", func(t *testing.T) {
		src := classificationManifestPrefix + `classification:
  source:
    - "src/**/*.ts"
  test:
    - "**/*.test.ts"
`
		m, err := ParseManifest([]byte(src))
		if err != nil {
			t.Fatalf("a pre-existing classification block must parse without error, got: %v", err)
		}
		if len(m.Classification.DependencyDirs) != 0 {
			t.Errorf("a block declaring no dependency_dirs must yield empty DependencyDirs, got %#v", m.Classification.DependencyDirs)
		}
		if len(m.Classification.Source) != 1 || m.Classification.Source[0] != "src/**/*.ts" {
			t.Errorf("Source must be unchanged, got %#v", m.Classification.Source)
		}
		if len(m.Classification.Test) != 1 || m.Classification.Test[0] != "**/*.test.ts" {
			t.Errorf("Test must be unchanged, got %#v", m.Classification.Test)
		}
	})
}
