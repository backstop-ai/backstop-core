package gate

import (
	"os"
	"strings"
	"testing"
)

// TestClassifier_SourceOnlyIsMeasurable (CLM-004): a path matching a declared
// SOURCE glob and NO test glob is MEASURABLE.
func TestClassifier_SourceOnlyIsMeasurable(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	if !c.IsMeasurableSource("pkg/x/foo.go") {
		t.Errorf("pkg/x/foo.go matches source **/*.go and no test glob — must be measurable")
	}
}

// TestClassifier_SourceAndTestOverlapNotMeasurable (CLM-005): a path matching
// BOTH a source and a test glob is NOT measurable — TEST-WINS-ON-OVERLAP.
func TestClassifier_SourceAndTestOverlapNotMeasurable(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go"})
	if c.IsMeasurableSource("pkg/x/foo_test.go") {
		t.Errorf("pkg/x/foo_test.go matches source **/*.go AND test **/*_test.go — test must win, not measurable")
	}
}

// TestClassifier_TestOnlyNotMeasurable (CLM-006): a path matching ONLY a test
// glob is NOT measurable.
func TestClassifier_TestOnlyNotMeasurable(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*.spec.ts"})
	if c.IsMeasurableSource("app/foo.spec.ts") {
		t.Errorf("app/foo.spec.ts matches only a test glob (no source glob) — not measurable")
	}
}

// TestClassifier_UnclassifiedNotMeasurable (CLM-007): a path matching NO declared
// glob is NOT measurable.
func TestClassifier_UnclassifiedNotMeasurable(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go"})
	if c.IsMeasurableSource("README.md") {
		t.Errorf("README.md matches no declared glob — not measurable")
	}
	if c.IsMeasurableSource("docs/guide.txt") {
		t.Errorf("docs/guide.txt matches no declared glob — not measurable")
	}
}

// TestClassifier_UnionAcrossMultipleToolchainPacks (CLM-008): a classifier built
// from a go pack (source **/*.go) AND a bun pack (source **/*.ts) measures BOTH a
// .go and a .ts source file (polyglot union).
func TestClassifier_UnionAcrossMultipleToolchainPacks(t *testing.T) {
	c := NewSourceClassifier(
		[]string{"**/*.go", "**/*.ts"},
		[]string{"**/*_test.go", "**/*.test.ts"},
	)
	if !c.IsMeasurableSource("pkg/x/foo.go") {
		t.Errorf("union must measure the Go source file pkg/x/foo.go")
	}
	if !c.IsMeasurableSource("app/foo.ts") {
		t.Errorf("union must measure the TS source file app/foo.ts")
	}
}

// TestClassifier_NoBakedGoLiteral_GoNotMeasurableWithoutGoGlobs (CLM-009): with
// ONLY non-Go globs declared (bun **/*.ts), a `.go` file is NOT measurable —
// proving no baked Go literal.
func TestClassifier_NoBakedGoLiteral_GoNotMeasurableWithoutGoGlobs(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.ts"}, []string{"**/*.test.ts"})
	if c.IsMeasurableSource("pkg/x/foo.go") {
		t.Errorf("with only **/*.ts declared, a .go file must NOT be measurable — a baked Go literal survived")
	}
}

// TestClassifier_GlobSemanticsDoublestarAndSegmentAware (CLM-010): `**/*.ts`
// matches a nested app/a/b.ts; a non-doublestar `*.ts` does not match across
// separators; matching is on the project-relative slash path.
func TestClassifier_GlobSemanticsDoublestarAndSegmentAware(t *testing.T) {
	doublestarC := NewSourceClassifier([]string{"**/*.ts"}, nil)
	if !doublestarC.IsMeasurableSource("app/a/b.ts") {
		t.Errorf("**/*.ts must match the nested path app/a/b.ts (doublestar crosses separators)")
	}

	singleStarC := NewSourceClassifier([]string{"*.ts"}, nil)
	if singleStarC.IsMeasurableSource("app/a/b.ts") {
		t.Errorf("a non-doublestar *.ts must NOT match across separators")
	}
	if !singleStarC.IsMeasurableSource("b.ts") {
		t.Errorf("*.ts must match a single-segment path b.ts")
	}
}

// TestClassifier_NoBakedExtensionLiteralsInClassificationGo (CLM-011): a source
// guard scoped to classification.go asserts it contains NO baked extension/fixture
// string literal — a reintroduced baked extension fails. Scoped to the
// classification implementation ONLY (step_coverage.go's relevance helpers
// legitimately keep their literals until SPEC-045).
func TestClassifier_NoBakedExtensionLiteralsInClassificationGo(t *testing.T) {
	data, err := os.ReadFile("classification.go")
	if err != nil {
		t.Fatalf("reading classification.go: %v", err)
	}
	src := string(data)
	// The classifier must key ONLY on the pack-declared globs handed to it — it
	// must hold no baked language/fixture extension of its own. These are the
	// exact baked tokens the de-Go work eradicated from the consumer.
	bakedTokens := []string{".g" + "o", "_test." + "go", "test" + "data"}
	for _, tok := range bakedTokens {
		if strings.Contains(src, tok) {
			t.Errorf("classification.go must contain NO baked %q literal — the classifier keys only on declared globs (CLM-011)", tok)
		}
	}
}

// TestClassifier_RootFileMeasurableUnderDoublestarSourceGlob (CLM-023): under
// source **/*.go a repo-ROOT file (embed.go) IS measurable — `**` matches zero
// directories (the zero-leading-segment property gobwas/glob lacks).
func TestClassifier_RootFileMeasurableUnderDoublestarSourceGlob(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	if !c.IsMeasurableSource("embed.go") {
		t.Errorf("repo-ROOT embed.go must be measurable under **/*.go (zero-leading-segment) — gobwas would drop it, re-opening the vacuous-green hole")
	}
}

// TestClassifier_RootTestFileMatchesTestGlobNotMeasurable (CLM-023): a repo-ROOT
// foo_test.go matches test **/*_test.go and is NOT measurable — proving the
// matcher does not silently drop root files in EITHER set.
func TestClassifier_RootTestFileMatchesTestGlobNotMeasurable(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	if c.IsMeasurableSource("foo_test.go") {
		t.Errorf("repo-ROOT foo_test.go matches test **/*_test.go (zero-leading-segment) — must NOT be measurable")
	}
}

// TestClassifier_HasSourceGlobsReportsDeclaration (REQ-004 support): HasSourceGlobs
// reports whether any source globs are declared, the signal the coverage step
// uses to surface the capability-absent state instead of a silent pass.
func TestClassifier_HasSourceGlobsReportsDeclaration(t *testing.T) {
	if NewSourceClassifier(nil, []string{"**/*_test.go"}).HasSourceGlobs() {
		t.Errorf("a classifier with no source globs must report HasSourceGlobs()==false")
	}
	if !NewSourceClassifier([]string{"**/*.go"}, nil).HasSourceGlobs() {
		t.Errorf("a classifier with source globs must report HasSourceGlobs()==true")
	}
}

// TestClassifier_IsTestFileReadsStoredTestSet (SPEC-045 seam): the classifier
// stores BOTH glob sets; IsTestFile reads exactly the stored test set, so a path
// matching a declared test glob is a test file and an unmatched path is not.
func TestClassifier_IsTestFileReadsStoredTestSet(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	if !c.IsTestFile("pkg/x/foo_test.go") {
		t.Errorf("pkg/x/foo_test.go matches a declared test glob — IsTestFile must be true")
	}
	if !c.IsTestFile("pkg/x/testdata/golden.txt") {
		t.Errorf("a testdata fixture matches **/testdata/** — IsTestFile must be true")
	}
	if c.IsTestFile("pkg/x/foo.go") {
		t.Errorf("pkg/x/foo.go matches no test glob — IsTestFile must be false")
	}
}

// TestClassifier_HasTestGlobsReportsDeclaration (SPEC-045 seam): HasTestGlobs
// reports whether any test globs are stored on the classifier.
func TestClassifier_HasTestGlobsReportsDeclaration(t *testing.T) {
	if NewSourceClassifier([]string{"**/*.go"}, nil).HasTestGlobs() {
		t.Errorf("a classifier with no test globs must report HasTestGlobs()==false")
	}
	if !NewSourceClassifier(nil, []string{"**/*_test.go"}).HasTestGlobs() {
		t.Errorf("a classifier with test globs must report HasTestGlobs()==true")
	}
}

// ISSUE-093 CLM-008 — ClaimsPath is the "does this pack own this file at all"
// predicate the file-mode dispatch decision reads. These three tests pin the
// UNION semantics, the anti-shortcut guard against IsMeasurableSource, and the
// UNKNOWABLE-vs-CLAIMS-NOTHING distinction a caller needs to tell a pack that
// declared nothing apart from a pack that declared globs matching nothing.

// TestSourceClassifier_ClaimsPathUnionsSourceAndTestGlobs proves ClaimsPath is
// TRUE for a path matching EITHER a declared source glob OR a declared test glob,
// and FALSE for the issue's own reproductions (a workflow file, a README, a spec
// artifact) — expressed at the predicate level, below any dispatch machinery.
// The globs are the installed go-toolchain pack's REAL declaration, not an
// invented one.
func TestSourceClassifier_ClaimsPathUnionsSourceAndTestGlobs(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})

	claimed := []string{
		"pkg/foo/bar.go",
		"pkg/foo/bar_test.go",
		// Root-level: doublestar's zero-leading-segment property matters here —
		// gobwas-style globbing would wrongly miss it and silently un-claim every
		// root source file.
		"main.go",
		"pkg/gate/testdata/x.go",
	}
	for _, p := range claimed {
		if !c.ClaimsPath(p) {
			t.Errorf("ClaimsPath(%q) = false, want true — the path matches a declared source or test glob", p)
		}
	}

	// The issue's three reproductions: none is claimed by a pack declaring only
	// these globs, so a package-scoped engine from that pack has no business
	// running for them.
	unclaimed := []string{
		".github/workflows/ci.yml",
		"README.md",
		"specs/SPEC-018-gate-diff-scope.spec.md",
	}
	for _, p := range unclaimed {
		if c.ClaimsPath(p) {
			t.Errorf("ClaimsPath(%q) = true, want false — the path matches no declared glob", p)
		}
	}
}

// TestSourceClassifier_ClaimsPathIsNotMeasurableSource is THE ANTI-SHORTCUT
// GUARD. IsMeasurableSource is `source AND NOT test` and deliberately EXCLUDES a
// test file; ClaimsPath is the UNION and deliberately INCLUDES it. Collapsing the
// two would make `gate --file pkg/foo/foo_test.go` claim nothing and skip that
// package's test engine — a vacuous green strictly worse than the crash ISSUE-093
// removes.
func TestSourceClassifier_ClaimsPathIsNotMeasurableSource(t *testing.T) {
	c := NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})

	const testFile = "pkg/foo/bar_test.go"
	if !c.ClaimsPath(testFile) {
		t.Errorf("ClaimsPath(%q) = false, want true — a test file IS owned by the pack that declares it", testFile)
	}
	if c.IsMeasurableSource(testFile) {
		t.Errorf("IsMeasurableSource(%q) = true, want false — the two predicates must not be the same question", testFile)
	}
	if c.ClaimsPath(testFile) == c.IsMeasurableSource(testFile) {
		t.Errorf("ClaimsPath and IsMeasurableSource must DIVERGE on a test file; both returned %v", c.ClaimsPath(testFile))
	}
}

// TestSourceClassifier_ClaimsPathFalseWithNoDeclaredGlobs pins the distinction
// CLM-006 is unanswerable without: a classifier that declares NOTHING claims
// nothing (ClaimsPath false everywhere) AND reports that it declares nothing, so
// a caller can tell UNKNOWABLE ("the pack never said what it owns") apart from
// CLAIMS-NOTHING ("the pack said, and none of these files matched").
func TestSourceClassifier_ClaimsPathFalseWithNoDeclaredGlobs(t *testing.T) {
	empty := NewSourceClassifier(nil, nil)
	for _, p := range []string{"pkg/foo/bar.go", "main.go", "README.md", ".github/workflows/ci.yml", ""} {
		if empty.ClaimsPath(p) {
			t.Errorf("ClaimsPath(%q) = true on a classifier declaring no globs, want false", p)
		}
	}
	if empty.DeclaresAnyGlobs() {
		t.Errorf("DeclaresAnyGlobs() = true on a classifier built from (nil, nil), want false")
	}

	// The distinction is only useful if it is OBSERVABLE: a classifier that DID
	// declare globs reports so, even when nothing in a given scope matches them.
	declared := NewSourceClassifier([]string{"**/*.go"}, nil)
	if !declared.DeclaresAnyGlobs() {
		t.Errorf("DeclaresAnyGlobs() = false on a classifier with source globs, want true")
	}
	if declared.ClaimsPath("README.md") {
		t.Errorf("declared-but-unmatched must still be ClaimsPath==false for README.md")
	}
	if !NewSourceClassifier(nil, []string{"**/*_test.go"}).DeclaresAnyGlobs() {
		t.Errorf("DeclaresAnyGlobs() = false on a classifier with only TEST globs, want true — either set counts as a declaration")
	}
}
