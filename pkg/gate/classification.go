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
