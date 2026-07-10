package waiver

import "testing"

// hasMalformedAt reports whether Result.Malformed carries a diagnostic at the
// given file+line with Kind "malformed".
func hasMalformedAt(res Result, file string, line int) bool {
	for _, d := range res.Malformed {
		if d.File == file && d.Line == line && d.Kind == DiagnosticMalformed {
			return true
		}
	}
	return false
}

// TestWaiver_Malformed_BadStructureIsFinding proves a token with too few fields
// is reported as a gate finding (CLM-029).
func TestWaiver_Malformed_BadStructureIsFinding(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 2}}
	read := lineReaderFrom("f", map[int]string{2: "x // @waiver:r:deferred"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !hasMalformedAt(res, "f", 2) {
		t.Fatalf("bad-structure token not reported as malformed; Malformed=%+v", res.Malformed)
	}
	if suppressed(res, "f", 2, "r") {
		t.Fatal("a malformed token must not suppress")
	}
}

// TestWaiver_Malformed_UnknownReasonIsFinding proves a token with an unknown
// reason-code is reported as a gate finding (CLM-030).
func TestWaiver_Malformed_UnknownReasonIsFinding(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 2}}
	read := lineReaderFrom("f", map[int]string{2: "x // @waiver:r:bogus:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !hasMalformedAt(res, "f", 2) {
		t.Fatal("unknown reason-code token not reported as malformed")
	}
}

// TestWaiver_Malformed_MissingExpiryIsFinding proves a token with a missing
// expiry is reported as a gate finding (CLM-031).
func TestWaiver_Malformed_MissingExpiryIsFinding(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 2}}
	read := lineReaderFrom("f", map[int]string{2: "x // @waiver:r:deferred:"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !hasMalformedAt(res, "f", 2) {
		t.Fatal("missing-expiry token not reported as malformed")
	}
}

// TestWaiver_Malformed_InvalidExpiryIsFinding proves a token with an invalid
// expiry format is reported as a gate finding (CLM-032).
func TestWaiver_Malformed_InvalidExpiryIsFinding(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 2}}
	read := lineReaderFrom("f", map[int]string{2: "x // @waiver:r:deferred:01-15-2027"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if !hasMalformedAt(res, "f", 2) {
		t.Fatal("invalid-expiry token not reported as malformed")
	}
}

// TestWaiver_Malformed_WellFormedNotFlagged proves a well-formed token is NOT
// reported as malformed (CLM-033).
func TestWaiver_Malformed_WellFormedNotFlagged(t *testing.T) {
	findings := []Finding{{RuleID: "r", File: "f", Line: 2}}
	read := lineReaderFrom("f", map[int]string{2: "x // @waiver:r:deferred:2999-01-01"})
	res := Adjudicate(findings, read, nil, fixedNow)
	if hasMalformedAt(res, "f", 2) {
		t.Fatal("well-formed token was flagged malformed")
	}
	if !suppressed(res, "f", 2, "r") {
		t.Fatal("well-formed active token should suppress")
	}
}
