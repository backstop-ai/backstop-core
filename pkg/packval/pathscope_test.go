package packval

import (
	"path/filepath"
	"strings"
	"testing"
)

// pathScopeDispatchCheck is the Check string the include/exclude inertness advisory
// carries. Declared once so every test in this file and its mask sibling agree on the
// identifier under test.
const pathScopeDispatchCheck = "path-scope-dispatch"

// pathScopeFixtureDir is the falsification corpus for both path-scope advisories. Every
// test here parses through the real ParseManifest and drives the real RunCoherence — a
// hand-built PackManifest literal would carry no on-disk rule file, and since this check
// reads the rule source off disk for its `paths:` block, every assertion would go
// vacuously green.
func pathScopeFixtureDir(name string) string {
	return filepath.Join("testdata", "path-scope", name)
}

func parsePathScopeFixture(t *testing.T, name string) (*PackManifest, string) {
	t.Helper()
	dir := pathScopeFixtureDir(name)
	pack, err := ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("ParseManifest(%s): %v", name, err)
	}
	return pack, dir
}

// pathScopeWarnings returns the warnings RunCoherence emitted for the named fixture
// under the given Check, plus the whole phase result for status assertions.
func pathScopeWarnings(t *testing.T, name, check string) ([]ValidationWarning, *PhaseResult) {
	t.Helper()
	pack, dir := parsePathScopeFixture(t, name)
	res := RunCoherence(pack, dir)
	var hits []ValidationWarning
	for _, w := range res.Warnings {
		if w.Check == check {
			hits = append(hits, w)
		}
	}
	return hits, res
}

// pathScopeCorpus names every fixture pack both advisories are exercised against. The
// non-blocking tests sweep all of it, so a malformed pack anywhere surfaces as a
// phase-2 ValidationError rather than as a silently missing assertion.
func pathScopeCorpus() []string {
	return []string{
		"inert-include",
		"inert-exclude",
		"slash-free-only",
		"semgrep-suggested-spellings",
		"non-ruleflags",
		"fixture-masked",
		"multiseg-no-mask",
		"two-rule-cross-contamination",
		"shared-rule-file",
		"multi-semgrep-rule-file",
		"partial-fixture-coverage",
	}
}

// TestPathScope_PredicateDecidesOnSeparatorAlone (CLM-001) tables the predicate over the
// spellings MEASURED against real semgrep 1.156.0 with real explicit-file dispatch. The
// `**/`-prefixed and `/`-anchored rows are the load-bearing ones: semgrep itself prints
// a deprecation warning recommending exactly those two rewrites, and BOTH were measured
// to stay dark. An implementer who trusts the tool's own remedy ships a fix that changes
// nothing, so the predicate must call them INERT.
func TestPathScope_PredicateDecidesOnSeparatorAlone(t *testing.T) {
	cases := []struct {
		pattern string
		inert   bool
		why     string
	}{
		{"cmd/backstop/pack_gate*.go", true, "bare directory-prefixed: measured 0 findings under explicit-file dispatch"},
		{"**/cmd/backstop/pack_gate*.go", true, "semgrep's OWN suggested unanchored rewrite: measured 0, still dark"},
		{"/cmd/backstop/pack_gate*.go", true, "semgrep's OWN suggested anchored rewrite: measured 0, still dark"},
		{"backstop/pack_gate*.go", true, "tail-only two-segment spelling: measured 0"},
		{"*/*/pack_gate*.go", true, "wildcard segments still contain separators: measured 0"},
		{"pack_gate*.go", false, "slash-free basename: measured 2 findings, honored under BOTH dispatch shapes"},
		{"*_test.go", false, "slash-free"},
		{"*testdata*", false, "slash-free"},
		{"*.go", false, "slash-free"},
	}
	for _, tc := range cases {
		t.Run(tc.pattern, func(t *testing.T) {
			if got := PathPatternInertUnderFileDispatch(tc.pattern); got != tc.inert {
				t.Fatalf("PathPatternInertUnderFileDispatch(%q) = %v, want %v (%s)", tc.pattern, got, tc.inert, tc.why)
			}
		})
	}
}

// TestPathScope_AdvisoryNamesEveryInertPattern (CLM-002) pins that every inert pattern
// is reported, and reported with enough identity to act on: the SEMGREP rule id it was
// declared under, the key it came from, the pattern text, and the rule-source file.
// ValidationWarning has no Rule field, so the identity lands in Message and Files.
func TestPathScope_AdvisoryNamesEveryInertPattern(t *testing.T) {
	cases := []struct {
		pack         string
		wantCount    int
		semgrepID    string
		manifestID   string
		ruleSource   string
		wantKey      string
		wantPatterns []string
	}{
		{
			pack:       "inert-include",
			wantCount:  1,
			semgrepID:  "sg-inert-include",
			manifestID: "inert-include-rule",
			ruleSource: "rules/inert-include.yml",
			wantKey:    "include",
			wantPatterns: []string{
				"cmd/app/handler*.go",
			},
		},
		{
			pack:       "inert-exclude",
			wantCount:  1,
			semgrepID:  "sg-inert-exclude",
			manifestID: "inert-exclude-rule",
			ruleSource: "rules/inert-exclude.yml",
			wantKey:    "exclude",
			wantPatterns: []string{
				"cmd/app/handler.go",
			},
		},
		{
			pack:       "semgrep-suggested-spellings",
			wantCount:  3,
			semgrepID:  "sg-suggested",
			manifestID: "suggested-spellings-rule",
			ruleSource: "rules/suggested-spellings.yml",
			wantKey:    "include",
			wantPatterns: []string{
				"**/cmd/app/handler*.go",
				"/cmd/app/handler*.go",
				"*/*/handler*.go",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.pack, func(t *testing.T) {
			hits, _ := pathScopeWarnings(t, tc.pack, pathScopeDispatchCheck)
			if len(hits) != tc.wantCount {
				t.Fatalf("%s: got %d %s warnings, want %d: %+v", tc.pack, len(hits), pathScopeDispatchCheck, tc.wantCount, hits)
			}
			for _, want := range tc.wantPatterns {
				found := false
				for _, w := range hits {
					if !strings.Contains(w.Message, want) {
						continue
					}
					found = true
					if !strings.Contains(w.Message, tc.semgrepID) {
						t.Errorf("%s: warning for pattern %q does not name its semgrep rule id %q: %q", tc.pack, want, tc.semgrepID, w.Message)
					}
					// SE-4: attributing to the pack MANIFEST rule id points the author
					// at the wrong rule in a multi-rule file. The corpus gives the two
					// ids deliberately different spellings so this can fail.
					if strings.Contains(w.Message, tc.manifestID) {
						t.Errorf("%s: warning names the MANIFEST rule id %q instead of the semgrep rule id: %q", tc.pack, tc.manifestID, w.Message)
					}
					if !strings.Contains(w.Message, tc.wantKey) {
						t.Errorf("%s: warning for pattern %q does not name the %q key: %q", tc.pack, want, tc.wantKey, w.Message)
					}
					if !containsString(w.Files, tc.ruleSource) {
						t.Errorf("%s: warning for pattern %q does not carry rule source %q in Files: %v", tc.pack, want, tc.ruleSource, w.Files)
					}
				}
				if !found {
					t.Errorf("%s: no %s warning names pattern %q; got %+v", tc.pack, pathScopeDispatchCheck, want, hits)
				}
			}
		})
	}
}

// TestPathScope_AdvisoryNamesFailureDirection (CLM-003) pins that the advisory says which
// WAY the failure runs. An inert include is a silent no-op (the rule scans nothing); an
// inert exclude fails OPEN (the rule fires on files the pack meant to exempt). Half the
// harm stays invisible if one generic string covers both.
func TestPathScope_AdvisoryNamesFailureDirection(t *testing.T) {
	includeHits, _ := pathScopeWarnings(t, "inert-include", pathScopeDispatchCheck)
	excludeHits, _ := pathScopeWarnings(t, "inert-exclude", pathScopeDispatchCheck)
	if len(includeHits) != 1 || len(excludeHits) != 1 {
		t.Fatalf("expected one warning each; got include=%d exclude=%d", len(includeHits), len(excludeHits))
	}
	inc := includeHits[0].Message
	exc := excludeHits[0].Message

	if !strings.Contains(strings.ToLower(inc), "scans nothing") {
		t.Errorf("include-side message does not say the rule scans nothing: %q", inc)
	}
	if !strings.Contains(strings.ToLower(exc), "fails open") {
		t.Errorf("exclude-side message does not say the exclusion fails open: %q", exc)
	}
	if !strings.Contains(strings.ToLower(exc), "exempt") {
		t.Errorf("exclude-side message does not say the rule fires on files the pack meant to exempt: %q", exc)
	}
	if inc == exc {
		t.Fatalf("include-side and exclude-side messages are identical, so the direction is not reported: %q", inc)
	}
}

// TestPathScope_SlashFreePatternsSilent (CLM-004) is what makes the whole check
// falsifiable. A slash-free pattern is the ONE spelling honored under both dispatch
// shapes, so flagging it would leave the advisory with no actionable remedy. An
// implementation that reports every pattern passes every other test and fails only here.
func TestPathScope_SlashFreePatternsSilent(t *testing.T) {
	hits, _ := pathScopeWarnings(t, "slash-free-only", pathScopeDispatchCheck)
	if len(hits) != 0 {
		t.Fatalf("slash-free-only declares only slash-free include and exclude patterns, which are honored under BOTH dispatch shapes, but got %d %s warnings: %+v", len(hits), pathScopeDispatchCheck, hits)
	}
}

// TestPathScope_SharedRuleFileFindingsDeduplicated (CLM-009) is the check that keeps this
// advisory usable on real packs. backstop-self declares SEVEN manifest rules all naming
// one 26-pattern file; a per-manifest-rule walk emits 7x26=182 warnings where the correct
// answer is 26. Here three manifest rules resolve to one two-pattern file, so the correct
// answer is exactly TWO and the duplicating answer is SIX.
func TestPathScope_SharedRuleFileFindingsDeduplicated(t *testing.T) {
	hits, _ := pathScopeWarnings(t, "shared-rule-file", pathScopeDispatchCheck)
	// Exact, never a floor: a per-manifest-rule walk emits six and would satisfy
	// "at least two".
	if len(hits) != 2 {
		t.Fatalf("shared-rule-file has 3 manifest rules resolving to 1 rule file with 2 distinct inert patterns; want exactly 2 %s warnings, got %d: %+v", pathScopeDispatchCheck, len(hits), hits)
	}
	var sawOne, sawTwo bool
	for _, w := range hits {
		if strings.Contains(w.Message, "sg-shared-one") && strings.Contains(w.Message, "cmd/app/one*.go") {
			sawOne = true
		}
		if strings.Contains(w.Message, "sg-shared-two") && strings.Contains(w.Message, "pkg/svc/two.go") {
			sawTwo = true
		}
	}
	if !sawOne {
		t.Errorf("no warning names sg-shared-one and its include pattern: %+v", hits)
	}
	if !sawTwo {
		t.Errorf("no warning names sg-shared-two and its exclude pattern: %+v", hits)
	}
	// A key-blind dedup could collapse the include and the exclude into one entry and
	// then re-emit it, landing on a count of two for the wrong reason.
	if hits[0].Message == hits[1].Message {
		t.Fatalf("both warnings carry the same Message, so the two distinct patterns were collapsed: %q", hits[0].Message)
	}
}

// TestPathScope_AdvisoryNeverBlocks (CLM-005) pins the advisory as non-blocking across
// the WHOLE corpus: it never populates Errors, never moves the phase off `pass`, and
// never changes the finalized Result status — so `pack check`, `pack test`, `pack add`
// and `pack install` keep their exit codes for the packs this repo already has installed.
//
// This is also the corpus's own guard. The mask and attribution packs declare claims, so
// a missing or empty fixture file there surfaces HERE as a fixtures-* / fixture-exists /
// fixture-empty ValidationError. A failure naming a Check other than path-scope-* means
// the fixture pack is malformed — fix the pack, never this assertion.
func TestPathScope_AdvisoryNeverBlocks(t *testing.T) {
	for _, name := range pathScopeCorpus() {
		t.Run(name, func(t *testing.T) {
			pack, dir := parsePathScopeFixture(t, name)
			res := RunCoherence(pack, dir)
			if len(res.Errors) != 0 {
				t.Fatalf("%s: phase 2 reported %d errors, want 0: %+v", name, len(res.Errors), res.Errors)
			}
			if res.Status != "pass" {
				t.Fatalf("%s: PhaseResult.Status = %q, want \"pass\"", name, res.Status)
			}
			full := &Result{Phases: []PhaseResult{*res}, Errors: res.Errors, Warnings: res.Warnings}
			full.FinalizeStatus()
			if full.Status != "pass" {
				t.Fatalf("%s: finalized Result.Status = %q, want \"pass\"", name, full.Status)
			}
		})
	}
}

// TestPathScope_NonRuleFlagsEngineNotScanned (CLM-007) pins the read behind the resolved
// binding's declared InputMode, never a tool-name or engine-name equality test — the same
// gate phase3's semgrep-rule-id check uses, and for the same reason. Under config-file the
// declared source is a PROJECT CONFIG naming rule directories, so a rules[].paths scan of
// it is categorically inapplicable rather than merely empty. The fixture's config file
// carries an inert-looking include precisely so an ungated implementation reports it.
func TestPathScope_NonRuleFlagsEngineNotScanned(t *testing.T) {
	hits, res := pathScopeWarnings(t, "non-ruleflags", pathScopeDispatchCheck)
	if len(hits) != 0 {
		t.Fatalf("non-ruleflags declares input_mode config-file, so its declared source is a project config this check must never interpret; got %d %s warnings: %+v", len(hits), pathScopeDispatchCheck, hits)
	}
	if res.Status != "pass" {
		t.Fatalf("non-ruleflags: PhaseResult.Status = %q, want \"pass\"", res.Status)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
