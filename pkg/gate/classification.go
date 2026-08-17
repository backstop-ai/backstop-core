package gate

import "github.com/bmatcuk/doublestar/v4"

// SourceClassifier is the language-neutral file classifier the coverage consumer
// reads instead of a baked extension literal (SPEC-043 REQ-002). It holds the
// MERGED UNION of the pack-declared source AND test glob sets (assembled by the
// caller across all declared toolchain packs) and answers whether a path is a
// MEASURABLE SOURCE file: it matches at least one declared source glob and no
// declared test glob (test-wins-on-overlap).
//
// It stores BOTH glob sets — the source set drives IsMeasurableSource/
// HasSourceGlobs, and the retained test set is load-bearing for Seed 2, which
// reads it via IsTestFile/HasTestGlobs. The classifier carries zero language
// knowledge: it is data (the declared globs) plus match logic only (DD-1).
type SourceClassifier struct {
	source []string
	test   []string
}

// NewSourceClassifier constructs a classifier from the MERGED source and test
// glob lists (the union across all declared toolchain packs is assembled by the
// caller and passed here). It RETAINS both sets on the struct — the test set is
// not dropped after computing the measurable predicate, so Seed 2 can read it via
// IsTestFile/HasTestGlobs (SPEC-043 contract). Matching uses
// github.com/bmatcuk/doublestar (TRUE doublestar, zero-leading-segment: `**`
// matches zero OR more directories, so a root-level file is matched as well as a
// nested one — the property gobwas/glob lacks, which would silently drop root
// source files and re-open the vacuous-green hole).
func NewSourceClassifier(source, test []string) SourceClassifier {
	return SourceClassifier{
		source: append([]string(nil), source...),
		test:   append([]string(nil), test...),
	}
}

// IsMeasurableSource reports whether path is a measurable SOURCE file: it matches
// at least one declared SOURCE glob AND no declared TEST glob (test-wins-on-
// overlap, since a test glob is normally a more-specific subset of a source glob).
// With only non-matching globs declared, an unrelated path returns false — no
// baked extension survives (CLM-009). Matching is on the project-relative slash
// path, normalized to the same form the scope and record index use.
func (c SourceClassifier) IsMeasurableSource(path string) bool {
	norm := NormalizePath("", path)
	return matchesAnyGlob(c.source, norm) && !matchesAnyGlob(c.test, norm)
}

// IsTestFile reports whether path matches at least one declared TEST glob. It
// reads exactly the test set retained by NewSourceClassifier (the Seed 2 seam).
func (c SourceClassifier) IsTestFile(path string) bool {
	norm := NormalizePath("", path)
	return matchesAnyGlob(c.test, norm)
}

// ClaimsPath reports whether the pack that declared these globs OWNS this path at
// all: it matches at least one declared SOURCE glob OR at least one declared TEST
// glob. It is the predicate the file-mode dispatch decision reads to answer "does
// the pack dispatching this engine claim any file in this scope" (ISSUE-093
// CLM-008) without core ever asking a language-specific question.
//
// THE UNION IS THE POINT, AND IT IS DELIBERATELY DIFFERENT FROM
// IsMeasurableSource. That predicate is `source AND NOT test` because its question
// is "should this file's coverage be measured", and a test file's own coverage is
// not the subject. The question HERE is "does this pack own this file", and a test
// file is owned — emphatically so, since it is the file whose package a
// package-scoped test engine would run. Collapsing the two would make a file-mode
// scope over a TEST file claim nothing, skip the test engine, and report a clean
// pass for a package whose tests never ran: a vacuous green strictly worse than
// the crash ISSUE-093 removes.
//
// Path normalization and malformed-glob handling are shared with the other
// predicates (a bad declared glob is a non-match, never a match-everything), so
// all three agree on path form.
func (c SourceClassifier) ClaimsPath(path string) bool {
	norm := NormalizePath("", path)
	return matchesAnyGlob(c.source, norm) || matchesAnyGlob(c.test, norm)
}

// DeclaresAnyGlobs reports whether the classifier carries ANY declaration at all,
// in either set. It is what separates UNKNOWABLE from CLAIMS-NOTHING: a false
// ClaimsPath on a classifier that DECLARES globs means the pack said what it owns
// and this path is not it; the same false on a classifier that declares NOTHING
// means the pack never said, and treating that as "owns nothing" would turn a
// missing declaration into a silent skip (ISSUE-093 CLM-006).
func (c SourceClassifier) DeclaresAnyGlobs() bool { return c.HasSourceGlobs() || c.HasTestGlobs() }

// HasSourceGlobs reports whether any source globs are declared. The coverage step
// reads it to surface the DISTINCT "classification capability absent" state when
// no toolchain pack declares source globs, instead of a silent pass (REQ-004).
func (c SourceClassifier) HasSourceGlobs() bool { return len(c.source) > 0 }

// HasTestGlobs reports whether any test globs are stored on the classifier (the
// Seed 2 seam, symmetric with HasSourceGlobs).
func (c SourceClassifier) HasTestGlobs() bool { return len(c.test) > 0 }

// matchesAnyGlob reports whether the project-relative slash path matches any glob
// in the list under doublestar semantics. A malformed pattern (the only error
// doublestar.Match returns) is treated as a non-match — a bad declared glob must
// not silently match everything.
func matchesAnyGlob(globs []string, path string) bool {
	for _, g := range globs {
		if ok, err := doublestar.Match(g, path); err == nil && ok {
			return true
		}
	}
	return false
}
