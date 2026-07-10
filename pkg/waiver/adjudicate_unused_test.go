package waiver

import "testing"

// unusedContains reports whether the Unused list carries a waiver for rule.
func unusedContains(res Result, rule string) bool {
	for _, w := range res.Unused {
		if w.RuleID == rule {
			return true
		}
	}
	return false
}

// TestWaiver_Unused_DanglingSurfacedAsWarning proves a waiver with no matching
// live finding at its location is surfaced as an unused/dangling warning
// (CLM-022). The token names a rule that does not fire at its location, so it
// covers nothing.
func TestWaiver_Unused_DanglingSurfacedAsWarning(t *testing.T) {
	findings := []Finding{{RuleID: "real-rule", File: "f", Line: 10}}
	read := lineReaderFrom("f", map[int]string{
		9:  "// @waiver:ghost-rule:deferred:2999-01-01",
		10: "code()",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !unusedContains(res, "ghost-rule") {
		t.Fatalf("dangling waiver not surfaced as unused; Unused=%+v", res.Unused)
	}
	if suppressed(res, "f", 10, "real-rule") {
		t.Fatal("a non-matching token must not suppress the finding")
	}
}

// TestWaiver_Unused_UsedWaiverNotFlagged proves a waiver with a matching live
// finding at its location is NOT flagged unused (CLM-023).
func TestWaiver_Unused_UsedWaiverNotFlagged(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 10}}
	read := lineReaderFrom("f", map[int]string{
		9:  "// @waiver:r:deferred:2999-01-01",
		10: "code()",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if unusedContains(res, "r") {
		t.Fatal("a used waiver was wrongly flagged unused")
	}
	if !suppressed(res, "f", 10, "r") {
		t.Fatal("the matching waiver should suppress")
	}
}

// TestWaiver_RuleId_RenamedRuleSurfacesUnused proves that after a pack renames a
// rule, the stale rule-id matches no live finding and the waiver surfaces as
// unused/dangling rather than silently waiving a DIFFERENT rule (CLM-040).
func TestWaiver_RuleId_RenamedRuleSurfacesUnused(t *testing.T) {
	// The finding now fires under the NEW rule-id; the token still names the OLD.
	findings := []Finding{{RuleID: "new-rule-name", File: "f", Line: 10}}
	read := lineReaderFrom("f", map[int]string{
		9:  "// @waiver:old-rule-name:deferred:2999-01-01",
		10: "code()",
	})
	res := Adjudicate(findings, read, nil, fixedNow)
	if suppressed(res, "f", 10, "new-rule-name") {
		t.Fatal("a stale rule-id must not silently waive the renamed rule's finding")
	}
	if !unusedContains(res, "old-rule-name") {
		t.Fatal("the stale rule-id waiver should surface as unused/dangling")
	}
}
