package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// bunClassifier builds the SourceClassifier from the bun pack's globs ALONE (the
// merged-union over the single declared pack), so the measurable-source matrix is
// proven against the bun pack's declared classification, not a baked literal.
func bunClassifier(t *testing.T, m *pack.Manifest) gate.SourceClassifier {
	t.Helper()
	return mergeSourceClassifier([]*pack.Manifest{m})
}

// TestBunPack_ClassificationGlobsParseOntoManifest proves the bun pack's
// classification block (source **/*.ts,**/*.tsx; test **/*.test.ts,**/*.spec.ts)
// parses onto the SPEC-043 pack.Manifest.Classification field intact (CLM-008).
func TestBunPack_ClassificationGlobsParseOntoManifest(t *testing.T) {
	m := bunToolchainManifest(t)
	wantSource := []string{"**/*.ts", "**/*.tsx"}
	wantTest := []string{"**/*.test.ts", "**/*.spec.ts"}
	if !equalStringSet(m.Classification.Source, wantSource) {
		t.Errorf("classification.source must be %v, got %v", wantSource, m.Classification.Source)
	}
	if !equalStringSet(m.Classification.Test, wantTest) {
		t.Errorf("classification.test must be %v, got %v", wantTest, m.Classification.Test)
	}
}

// TestBunClassifier_TsSourceIsMeasurable proves a `.ts` source file is MEASURABLE
// source under the bun pack's globs (matches a source glob, no test glob) (CLM-009).
func TestBunClassifier_TsSourceIsMeasurable(t *testing.T) {
	c := bunClassifier(t, bunToolchainManifest(t))
	if !c.IsMeasurableSource("src/app.ts") {
		t.Error("src/app.ts must be measurable source under the bun pack's **/*.ts source glob")
	}
}

// TestBunClassifier_TsxSourceIsMeasurable proves a `.tsx` source file is
// MEASURABLE source under the bun pack's globs (CLM-010).
func TestBunClassifier_TsxSourceIsMeasurable(t *testing.T) {
	c := bunClassifier(t, bunToolchainManifest(t))
	if !c.IsMeasurableSource("src/Button.tsx") {
		t.Error("src/Button.tsx must be measurable source under the bun pack's **/*.tsx source glob")
	}
}

// TestBunClassifier_TestTsNotMeasurable proves a `.test.ts` file is NOT measurable
// — it matches both a source glob (**/*.ts) and a test glob (**/*.test.ts), and
// test-WINS-on-overlap (CLM-011).
func TestBunClassifier_TestTsNotMeasurable(t *testing.T) {
	m := bunToolchainManifest(t)
	c := bunClassifier(t, m)
	if !c.IsTestFile("src/app.test.ts") {
		t.Error("src/app.test.ts must match a declared test glob (**/*.test.ts)")
	}
	if c.IsMeasurableSource("src/app.test.ts") {
		t.Error("src/app.test.ts must NOT be measurable — it matches both source and test globs and test wins on overlap")
	}
}

// TestBunClassifier_SpecTsNotMeasurable proves a `.spec.ts` file is NOT measurable.
// It relies on test-wins-on-overlap, NOT glob exclusivity: `**/*.spec.ts` ⊂
// `**/*.ts`, so the file matches BOTH globs and is non-measurable only because the
// test glob wins (CLM-012, the sharp edge).
func TestBunClassifier_SpecTsNotMeasurable(t *testing.T) {
	m := bunToolchainManifest(t)
	c := bunClassifier(t, m)
	if !c.IsTestFile("src/app.spec.ts") {
		t.Error("src/app.spec.ts must match the declared test glob (**/*.spec.ts)")
	}
	if c.IsMeasurableSource("src/app.spec.ts") {
		t.Error("src/app.spec.ts must NOT be measurable — a .spec.ts IS a .ts, so it matches the source glob too; it is non-measurable ONLY because test wins on overlap")
	}
}

// TestBunClassifier_GoFileNotMeasurableUnderBunPack proves a `.go` file is NOT
// measurable under the bun pack ALONE — the bun pack declares no Go glob, so no
// baked Go literal leaks across packs into the classifier (CLM-013).
func TestBunClassifier_GoFileNotMeasurableUnderBunPack(t *testing.T) {
	c := bunClassifier(t, bunToolchainManifest(t))
	if c.IsMeasurableSource("pkg/x/foo.go") {
		t.Error("pkg/x/foo.go must NOT be measurable under the bun pack alone — no baked Go literal may leak across packs into the classifier")
	}
}

// TestBunClassifier_UnclassifiedMarkdownNotMeasurable proves an unclassified file
// (README.md) matching neither a source nor a test glob is NOT measurable (CLM-014).
func TestBunClassifier_UnclassifiedMarkdownNotMeasurable(t *testing.T) {
	c := bunClassifier(t, bunToolchainManifest(t))
	if c.IsMeasurableSource("README.md") {
		t.Error("README.md matches neither a source nor a test glob and must NOT be measurable")
	}
}

// equalStringSet reports whether two string slices contain the same elements in the
// same order (the classification globs are an ordered declared list).
func equalStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
