package waiver

import (
	"testing"
	"time"
)

// fixedNow is a stable "now" well before the far-future expiry dates the core
// tests use, so every parsed token is ACTIVE for these matcher tests.
var fixedNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// lineReaderFrom builds an in-memory LineReader over a single file's lines. It
// is the ONLY source access adjudication is given — no filesystem, no parser.
func lineReaderFrom(file string, lines map[int]string) LineReader {
	return func(f string, line int) (string, bool) {
		if f != file {
			return "", false
		}
		s, ok := lines[line]
		return s, ok
	}
}

// suppressed reports whether a finding matching file+line+rule is in the set.
func suppressed(res Result, file string, line int, rule string) bool {
	for _, f := range res.Suppressed {
		if f.File == file && f.Line == line && f.RuleID == rule {
			return true
		}
	}
	return false
}

// TestWaiver_Adjudicate_SuppressesAtReportedLocation proves an engine-emitted
// finding is suppressed when a matching token sits at its reported location
// (CLM-009).
func TestWaiver_Adjudicate_SuppressesAtReportedLocation(t *testing.T) {
	findings := []Finding{{RuleID: "no-panic", File: "app.go", Line: 5}}
	read := lineReaderFrom("app.go", map[int]string{
		5: "\tpanic(\"boom\") // @waiver:no-panic:accepted-risk:2999-01-01",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "app.go", 5, "no-panic") {
		t.Fatalf("finding at reported location not suppressed; Suppressed=%+v", res.Suppressed)
	}
}

// TestWaiver_Adjudicate_CommentSyntaxAgnostic proves suppression works
// regardless of the surrounding comment prefix, reading only raw line bytes
// (CLM-010).
func TestWaiver_Adjudicate_CommentSyntaxAgnostic(t *testing.T) {
	prefixes := map[string]string{
		"go-slashes": "code() // @waiver:r1:deferred:2999-01-01",
		"hash":       "code() # @waiver:r1:deferred:2999-01-01",
		"semicolon":  "code() ; @waiver:r1:deferred:2999-01-01",
		"html":       "code() <!-- @waiver:r1:deferred:2999-01-01 -->",
		"none":       "@waiver:r1:deferred:2999-01-01",
	}
	for name, lineText := range prefixes {
		t.Run(name, func(t *testing.T) {
			findings := []Finding{{RuleID: "r1", File: "f", Line: 1}}
			read := lineReaderFrom("f", map[int]string{1: lineText})
			res := Adjudicate(findings, read, nil, fixedNow)
			if !suppressed(res, "f", 1, "r1") {
				t.Fatalf("prefix %q not suppressed: line=%q", name, lineText)
			}
		})
	}
}

// TestWaiver_Adjudicate_NoLanguageParserInvoked is the ZERO-BAKED absence guard
// (CLM-011): Adjudicate receives no language identifier and invokes no
// comment/source parser. Proof: a finding in a file whose extension names no
// language and that does NOT exist on disk is still suppressed purely from the
// bytes the LineReader supplies — Adjudicate never opens the file, never
// dispatches on extension, and finds the token by raw byte scan of an arbitrary
// comment syntax.
func TestWaiver_Adjudicate_NoLanguageParserInvoked(t *testing.T) {
	readCalls := 0
	read := func(f string, line int) (string, bool) {
		readCalls++
		if f == "phantom.zzz" && line == 5 {
			// A comment syntax no language owns; only a raw byte scan finds it.
			return "?!? @waiver:mystery-rule:false-positive:2999-01-01 ?!?", true
		}
		return "", false
	}
	findings := []Finding{{RuleID: "mystery-rule", File: "phantom.zzz", Line: 5}}
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "phantom.zzz", 5, "mystery-rule") {
		t.Fatal("adjudication failed on an unknown extension / nonexistent file; it must byte-scan the reader bytes, not parse a language")
	}
	if readCalls == 0 {
		t.Fatal("adjudication never used the LineReader; it must obtain source bytes ONLY through the reader")
	}
}

// TestWaiver_Identity_SuppressesColocatedFinding proves a waiver suppresses
// exactly the finding co-located with its token (CLM-012).
func TestWaiver_Identity_SuppressesColocatedFinding(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 3}}
	read := lineReaderFrom("f", map[int]string{3: "x // @waiver:r:deferred:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 3, "r") {
		t.Fatal("co-located finding not suppressed")
	}
}

// TestWaiver_Identity_NoFileBlanket proves a same-rule finding at a different
// location in the same file is NOT suppressed (no file-blanket, CLM-013).
func TestWaiver_Identity_NoFileBlanket(t *testing.T) {
	findings := []Finding{
		{RuleID: "r", File: "f", Line: 3},
		{RuleID: "r", File: "f", Line: 20},
	}
	read := lineReaderFrom("f", map[int]string{3: "x // @waiver:r:deferred:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 3, "r") {
		t.Fatal("co-located finding should be suppressed")
	}
	if suppressed(res, "f", 20, "r") {
		t.Fatal("distant same-rule finding was suppressed: a waiver must not blanket the file")
	}
}

// TestWaiver_Identity_NoRuleBlanket proves that with two same-rule findings,
// waiving one leaves the other firing (no rule-blanket, CLM-014).
func TestWaiver_Identity_NoRuleBlanket(t *testing.T) {
	findings := []Finding{
		{RuleID: "r", File: "f", Line: 3},
		{RuleID: "r", File: "g", Line: 3},
	}
	read := lineReaderFrom("f", map[int]string{3: "x // @waiver:r:deferred:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 3, "r") {
		t.Fatal("waived finding should be suppressed")
	}
	if suppressed(res, "g", 3, "r") {
		t.Fatal("other same-rule finding was suppressed: a waiver must not blanket the rule")
	}
}

// TestWaiver_Match_SameLineTrailingSuppresses proves a token trailing on the
// finding's own start line associates and suppresses (CLM-034).
func TestWaiver_Match_SameLineTrailingSuppresses(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 7}}
	read := lineReaderFrom("f", map[int]string{7: "risky() // @waiver:r:deferred:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 7, "r") {
		t.Fatal("same-line trailing token did not suppress")
	}
}

// TestWaiver_Match_LineAboveSuppresses proves a token on the line immediately
// above the finding associates and suppresses (CLM-035).
func TestWaiver_Match_LineAboveSuppresses(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 7}}
	read := lineReaderFrom("f", map[int]string{
		6: "// @waiver:r:deferred:2999-01-01",
		7: "risky()",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 7, "r") {
		t.Fatal("line-above token did not suppress")
	}
}

// TestWaiver_Match_TwoLinesAboveDoesNotAssociate proves a token two or more
// lines above the finding does NOT associate (CLM-036).
func TestWaiver_Match_TwoLinesAboveDoesNotAssociate(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 7}}
	read := lineReaderFrom("f", map[int]string{
		5: "// @waiver:r:deferred:2999-01-01",
		7: "risky()",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if suppressed(res, "f", 7, "r") {
		t.Fatal("a token two lines above must NOT associate with the finding")
	}
}

// TestWaiver_Match_MultiLineFindingAssociates proves that for a multi-line
// finding, a token above the finding's start line associates with the whole
// region (CLM-037).
func TestWaiver_Match_MultiLineFindingAssociates(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 10, EndLine: 14}}
	read := lineReaderFrom("f", map[int]string{
		9:  "// @waiver:r:accepted-risk:2999-01-01",
		10: "func big() {",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 10, "r") {
		t.Fatal("token above a multi-line finding's start line did not associate")
	}
}

// TestWaiver_Match_MultiLineTrailingStartLineAssociates proves that for a
// multi-line finding, a token TRAILING on its start line associates and
// suppresses (CLM-066).
func TestWaiver_Match_MultiLineTrailingStartLineAssociates(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 10, EndLine: 14}}
	read := lineReaderFrom("f", map[int]string{
		10: "func big() { // @waiver:r:accepted-risk:2999-01-01",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 10, "r") {
		t.Fatal("trailing token on a multi-line finding's start line did not suppress")
	}
}

// TestWaiver_Match_MismatchedRuleIdNoSuppress proves a token whose rule-id
// differs from the finding's rule-id does NOT suppress (false-match avoidance,
// CLM-038).
func TestWaiver_Match_MismatchedRuleIdNoSuppress(t *testing.T) {
	findings := []Finding{{RuleID: "actual-rule", File: "f", Line: 4}}
	read := lineReaderFrom("f", map[int]string{4: "x // @waiver:other-rule:deferred:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if suppressed(res, "f", 4, "actual-rule") {
		t.Fatal("a token naming a DIFFERENT rule suppressed the finding")
	}
}

// TestWaiver_RuleId_MatchingRuleSuppresses proves a waiver whose rule-id
// matches the live finding's rule-id suppresses it (CLM-039).
func TestWaiver_RuleId_MatchingRuleSuppresses(t *testing.T) {
	findings := []Finding{{RuleID: "exact-rule", File: "f", Line: 4}}
	read := lineReaderFrom("f", map[int]string{4: "x // @waiver:exact-rule:deferred:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !suppressed(res, "f", 4, "exact-rule") {
		t.Fatal("exact rule-id match did not suppress")
	}
}
