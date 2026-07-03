package pack

import "testing"

// classificationManifestPrefix is the minimal valid manifest preamble the
// classification-parse tests append a `classification:` block to. It uses the
// `code` archetype with SDK content so the manifest validates without declaring
// any execution engines (the classification block is the only thing under test).
const classificationManifestPrefix = `name: backstop/fixture-toolchain
version: 1.0.0
language: go
archetype: code
description: Fixture pack exercising the classification contract.
content:
  sdk:
    module: example/fixture
    version: 1.0.0
    provides:
      - classification
`

// TestManifest_ParsesClassificationGlobs (CLM-001): a pack.yml top-level
// `classification:` block with `source:` and `test:` glob lists parses onto
// pack.Manifest.Classification (Source/Test string slices) in DECLARED ORDER.
func TestManifest_ParsesClassificationGlobs(t *testing.T) {
	src := classificationManifestPrefix + `classification:
  source:
    - "src/**/*.ts"
    - "lib/**/*.ts"
  test:
    - "**/*.test.ts"
    - "**/*.spec.ts"
`
	m, err := ParseManifest([]byte(src))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	wantSource := []string{"src/**/*.ts", "lib/**/*.ts"}
	wantTest := []string{"**/*.test.ts", "**/*.spec.ts"}
	if len(m.Classification.Source) != len(wantSource) {
		t.Fatalf("Source len = %d, want %d (%#v)", len(m.Classification.Source), len(wantSource), m.Classification.Source)
	}
	for i, want := range wantSource {
		if m.Classification.Source[i] != want {
			t.Errorf("Source[%d] = %q, want %q (declared order)", i, m.Classification.Source[i], want)
		}
	}
	if len(m.Classification.Test) != len(wantTest) {
		t.Fatalf("Test len = %d, want %d (%#v)", len(m.Classification.Test), len(wantTest), m.Classification.Test)
	}
	for i, want := range wantTest {
		if m.Classification.Test[i] != want {
			t.Errorf("Test[%d] = %q, want %q (declared order)", i, m.Classification.Test[i], want)
		}
	}
}

// TestManifest_AbsentClassificationIsEmptyNotError (CLM-002): a manifest with NO
// `classification:` block yields a zero-value Classification (empty Source and
// Test) and NO parse error — the block is OPTIONAL.
func TestManifest_AbsentClassificationIsEmptyNotError(t *testing.T) {
	m, err := ParseManifest([]byte(classificationManifestPrefix))
	if err != nil {
		t.Fatalf("a manifest with no classification block must parse without error, got: %v", err)
	}
	if len(m.Classification.Source) != 0 {
		t.Errorf("absent block must yield empty Source, got %#v", m.Classification.Source)
	}
	if len(m.Classification.Test) != 0 {
		t.Errorf("absent block must yield empty Test, got %#v", m.Classification.Test)
	}
}

// TestManifest_GoToolchainClassificationReferenceShapeRoundTrips (CLM-003): the
// go-toolchain reference shape round-trips — source ["**/*.go"], test
// ["**/*_test.go", "**/testdata/**"] parse intact, the fixture/testdata
// convention folded into the test list (no separate baked testdata dimension).
func TestManifest_GoToolchainClassificationReferenceShapeRoundTrips(t *testing.T) {
	src := classificationManifestPrefix + `classification:
  source:
    - "**/*.go"
  test:
    - "**/*_test.go"
    - "**/testdata/**"
`
	m, err := ParseManifest([]byte(src))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(m.Classification.Source) != 1 || m.Classification.Source[0] != "**/*.go" {
		t.Errorf("reference Source must be [\"**/*.go\"], got %#v", m.Classification.Source)
	}
	if len(m.Classification.Test) != 2 ||
		m.Classification.Test[0] != "**/*_test.go" ||
		m.Classification.Test[1] != "**/testdata/**" {
		t.Errorf("reference Test must be [\"**/*_test.go\", \"**/testdata/**\"] (testdata folded in), got %#v", m.Classification.Test)
	}
}
