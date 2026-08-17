package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// ciGlobScopingHarnessFile is the file under guard. Named once so both tests here agree.
const ciGlobScopingHarnessFile = "ci_recipes_harness_test.go"

// ciInlineSlashCheckExpr matches an inline "does this string contain a slash" test.
// Deliberately a regexp rather than a literal: it survives a rename of the loop variable
// and a gofmt pass, so it guards the CONTRACT rather than one line's spelling.
const ciInlineSlashCheckExpr = `strings\.Contains\([^,]+,\s*"/"\)`

func ciInlineSlashCheckPattern() *regexp.Regexp {
	return regexp.MustCompile(ciInlineSlashCheckExpr)
}

// TestCIGlobScoping_UsesSharedInertnessPredicate (CLM-008) is the SOURCE guard, and the
// only mechanism that can catch a skipped refactor.
//
// A behavioural agreement test cannot substitute for it. `strings.Contains(p, "/")` and
// packval.PathPatternInertUnderFileDispatch are identical BY CONSTRUCTION, so an
// agreement assertion passes whether or not the refactor ever happened — which is exactly
// the failure mode this test exists to close. This is the narrow case where the usual
// "prefer behaviour over source text" instinct is wrong: there is no behavioural
// difference to observe.
//
// THE ASSERTION IS SCOPED TO TWO FUNCTIONS, AND MUST BE. A whole-file zero-match is
// UNSATISFIABLE: the regexp also matches `strings.Contains(token, "/")` inside
// ciOrgRepoLiterals, which parses an org/repo release COORDINATE. A slash there means
// "this token has two segments", not "this path pattern is inert under file dispatch";
// routing it through a path-scope predicate would be a semantic error that happens to
// compile. It stays exactly as it is.
func TestCIGlobScoping_UsesSharedInertnessPredicate(t *testing.T) {
	raw, err := os.ReadFile(ciGlobScopingHarnessFile)
	if err != nil {
		t.Fatalf("reading %s: %v", ciGlobScopingHarnessFile, err)
	}
	src := string(raw)

	const predicate = "packval.PathPatternInertUnderFileDispatch"
	if !strings.Contains(src, predicate) {
		t.Fatalf("%s does not call %s, so the inertness contract is still stated a second time here and the two can drift", ciGlobScopingHarnessFile, predicate)
	}

	regions := map[string][2]int{}
	for _, fn := range []string{"ciGlobScopingProblems", "ciMultiSegmentIncludeProblems"} {
		start, end, ok := ciFuncRegion(src, fn)
		if !ok {
			t.Fatalf("%s declares no func %s; the seam TASK-009 extracts is missing", ciGlobScopingHarnessFile, fn)
		}
		regions[fn] = [2]int{start, end}
		if hits := ciInlineSlashCheckPattern().FindAllString(src[start:end], -1); len(hits) != 0 {
			t.Errorf("func %s still contains %d inline slash check(s) %v; it must decide inertness through %s so the contract is stated exactly once", fn, len(hits), hits, predicate)
		}
	}

	// Stronger than a bare region scan, and it additionally fails loudly if a third
	// occurrence is ever introduced: exactly ONE inline slash check may survive
	// file-wide, and its offset must fall OUTSIDE both guarded regions.
	all := ciInlineSlashCheckPattern().FindAllStringIndex(src, -1)
	if len(all) != 1 {
		t.Fatalf("expected exactly 1 surviving inline slash check in %s (the org/repo coordinate parser in ciOrgRepoLiterals), got %d at %v", ciGlobScopingHarnessFile, len(all), all)
	}
	for fn, r := range regions {
		if all[0][0] >= r[0] && all[0][0] < r[1] {
			t.Fatalf("the surviving inline slash check at offset %d falls INSIDE func %s; the path-scope contract must not be restated there", all[0][0], fn)
		}
	}
}

// TestCIGlobScoping_MultiSegmentProblemsAgreeWithPredicate (CLM-008) is the BEHAVIOURAL
// half, and the ONLY thing in the tree that executes the slash-detection branch.
//
// The four real callers cannot verify this refactor: ci_recipes_{github_actions,
// gitlab_ci,bitbucket_pipelines,jenkins}_test.go all pass slash-FREE include sets and all
// assert only len(problems) != 0, so the branch never runs under their data and "those
// tests still pass" is vacuous evidence. ciGlobScopingProblems itself cannot be driven
// over synthetic patterns either — it derives its set from ciIncludeSetFor, which parses
// the REAL installed pack off disk, and it SHORT-CIRCUITS before the slash loop when that
// set does not equal the caller's want list. Hence the extracted seam.
func TestCIGlobScoping_MultiSegmentProblemsAgreeWithPredicate(t *testing.T) {
	// The same spellings the packval predicate table uses, measured against real semgrep
	// under real explicit-file dispatch.
	cases := []struct {
		pattern string
		inert   bool
	}{
		{"cmd/backstop/pack_gate*.go", true},
		{"**/cmd/backstop/pack_gate*.go", true},
		{"/cmd/backstop/pack_gate*.go", true},
		{"backstop/pack_gate*.go", true},
		{"*/*/pack_gate*.go", true},
		{"pack_gate*.go", false},
		{"*_test.go", false},
		{"*testdata*", false},
		{"*.go", false},
	}

	var inertPatterns []string
	for _, tc := range cases {
		// Drive one pattern at a time: problem strings embed the pattern text, and
		// several of these spellings are substrings of one another, so a combined-call
		// substring search could not tell them apart.
		got := ciMultiSegmentIncludeProblems([]string{tc.pattern})
		want := 0
		if tc.inert {
			want = 1
			inertPatterns = append(inertPatterns, tc.pattern)
		}
		if len(got) != want {
			t.Errorf("ciMultiSegmentIncludeProblems([%q]) returned %d problems, want %d: %v", tc.pattern, len(got), want, got)
			continue
		}
		if tc.inert && !strings.Contains(got[0], tc.pattern) {
			t.Errorf("the problem raised for %q does not name the pattern: %q", tc.pattern, got[0])
		}
		// Entry for entry against the single authority.
		if inert := packval.PathPatternInertUnderFileDispatch(tc.pattern); inert != tc.inert {
			t.Errorf("packval.PathPatternInertUnderFileDispatch(%q) = %v, want %v — the harness and the shared predicate disagree", tc.pattern, inert, tc.inert)
		}
		if (len(got) != 0) != packval.PathPatternInertUnderFileDispatch(tc.pattern) {
			t.Errorf("harness verdict for %q (%d problems) disagrees with the shared predicate", tc.pattern, len(got))
		}
	}

	// One combined call: the helper preserves input order, so problem i names inert
	// pattern i. This catches an implementation that reports the right COUNT off the
	// wrong patterns.
	all := ciMultiSegmentIncludeProblems(patternsOf(cases))
	if len(all) != len(inertPatterns) {
		t.Fatalf("over the full mixed set, ciMultiSegmentIncludeProblems returned %d problems, want %d: %v", len(all), len(inertPatterns), all)
	}
	for i, pattern := range inertPatterns {
		if !strings.Contains(all[i], pattern) {
			t.Errorf("problem %d does not name inert pattern %q: %q", i, pattern, all[i])
		}
	}

	// An empty set must be silent, so the helper never manufactures a problem from
	// nothing.
	if got := ciMultiSegmentIncludeProblems(nil); len(got) != 0 {
		t.Errorf("ciMultiSegmentIncludeProblems(nil) returned %d problems, want 0: %v", len(got), got)
	}
}

func patternsOf(cases []struct {
	pattern string
	inert   bool
}) []string {
	out := make([]string, 0, len(cases))
	for _, c := range cases {
		out = append(out, c.pattern)
	}
	return out
}

// ciFuncRegion returns the byte range of a top-level func declaration's source, from its
// `func NAME(` line to the next top-level `\nfunc `. A Go parser is not needed for this:
// top-level funcs in this file are the only declarations starting at column zero with
// `func `.
func ciFuncRegion(src, name string) (int, int, bool) {
	start := strings.Index(src, "\nfunc "+name+"(")
	if start < 0 {
		if strings.HasPrefix(src, "func "+name+"(") {
			start = 0
		} else {
			return 0, 0, false
		}
	} else {
		start++
	}
	next := strings.Index(src[start+1:], "\nfunc ")
	if next < 0 {
		return start, len(src), true
	}
	return start, start + 1 + next, true
}
