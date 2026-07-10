package waiver

import "testing"

// loc is a small helper producing a Location for grammar tests.
func grammarLoc() Location { return Location{File: "app.go", Line: 10} }

// TestWaiver_ParseToken_WellFormedParses proves a well-formed token
// @waiver:<rule>:<reason>:<expiry> parses into a populated Waiver (CLM-001).
func TestWaiver_ParseToken_WellFormedParses(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:false-positive:2027-01-15", grammarLoc())
	if err != nil {
		t.Fatalf("well-formed token returned error: %v", err)
	}
	if w.RuleID != "pkg-rule" {
		t.Errorf("RuleID = %q, want pkg-rule", w.RuleID)
	}
	if w.Reason != ReasonFalsePositive {
		t.Errorf("Reason = %q, want %q", w.Reason, ReasonFalsePositive)
	}
	if got := w.Expiry.Format("2006-01-02"); got != "2027-01-15" {
		t.Errorf("Expiry = %q, want 2027-01-15", got)
	}
	if w.File != "app.go" || w.Line != 10 {
		t.Errorf("location not carried: File=%q Line=%d", w.File, w.Line)
	}
}

// TestWaiver_ParseToken_OptionalNoteCaptured proves an optional trailing
// free-text note is captured (CLM-002).
func TestWaiver_ParseToken_OptionalNoteCaptured(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:accepted-risk:2027-01-15 known slow path, revisit later", grammarLoc())
	if err != nil {
		t.Fatalf("token with note returned error: %v", err)
	}
	if w.Note != "known slow path, revisit later" {
		t.Errorf("Note = %q, want the trailing free text", w.Note)
	}
	if w.IssueRef != "" {
		t.Errorf("IssueRef = %q, want empty (no issue ref present)", w.IssueRef)
	}
}

// TestWaiver_ParseToken_OptionalIssueRefCaptured proves an optional issue
// reference is captured (CLM-003).
func TestWaiver_ParseToken_OptionalIssueRefCaptured(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:deferred:2027-01-15 ISSUE-123", grammarLoc())
	if err != nil {
		t.Fatalf("token with issue-ref returned error: %v", err)
	}
	if w.IssueRef != "ISSUE-123" {
		t.Errorf("IssueRef = %q, want ISSUE-123", w.IssueRef)
	}
}

// TestWaiver_ParseToken_ReasonFalsePositiveValid proves reason-code
// false-positive is accepted by the grammar (CLM-004).
func TestWaiver_ParseToken_ReasonFalsePositiveValid(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:false-positive:2027-01-15", grammarLoc())
	if err != nil {
		t.Fatalf("false-positive rejected: %v", err)
	}
	if w.Reason != ReasonFalsePositive {
		t.Errorf("Reason = %q, want false-positive", w.Reason)
	}
}

// TestWaiver_ParseToken_ReasonAcceptedRiskValid proves reason-code
// accepted-risk is accepted (CLM-005).
func TestWaiver_ParseToken_ReasonAcceptedRiskValid(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:accepted-risk:2027-01-15", grammarLoc())
	if err != nil {
		t.Fatalf("accepted-risk rejected: %v", err)
	}
	if w.Reason != ReasonAcceptedRisk {
		t.Errorf("Reason = %q, want accepted-risk", w.Reason)
	}
}

// TestWaiver_ParseToken_ReasonDeferredValid proves reason-code deferred is
// accepted (CLM-006).
func TestWaiver_ParseToken_ReasonDeferredValid(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:deferred:2027-01-15", grammarLoc())
	if err != nil {
		t.Fatalf("deferred rejected: %v", err)
	}
	if w.Reason != ReasonDeferred {
		t.Errorf("Reason = %q, want deferred", w.Reason)
	}
}

// TestWaiver_ParseToken_ReasonThirdPartyValid proves reason-code third-party
// is accepted (CLM-007).
func TestWaiver_ParseToken_ReasonThirdPartyValid(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:third-party:2027-01-15", grammarLoc())
	if err != nil {
		t.Fatalf("third-party rejected: %v", err)
	}
	if w.Reason != ReasonThirdParty {
		t.Errorf("Reason = %q, want third-party", w.Reason)
	}
}

// TestWaiver_ParseToken_UnknownReasonRejected proves an unknown reason-code
// outside the closed enum is rejected with a non-nil parse error (CLM-008).
func TestWaiver_ParseToken_UnknownReasonRejected(t *testing.T) {
	if _, err := ParseToken("@waiver:pkg-rule:not-a-reason:2027-01-15", grammarLoc()); err == nil {
		t.Fatal("unknown reason-code was accepted; want a non-nil parse error")
	}
}

// TestWaiver_ParseToken_HashIssueRefCaptured covers a #-number issue reference.
func TestWaiver_ParseToken_HashIssueRefCaptured(t *testing.T) {
	w, err := ParseToken("@waiver:pkg-rule:deferred:2027-01-15 fix soon #4213", grammarLoc())
	if err != nil {
		t.Fatalf("token with #-issue-ref returned error: %v", err)
	}
	if w.IssueRef != "#4213" {
		t.Errorf("IssueRef = %q, want #4213", w.IssueRef)
	}
	if w.Note != "fix soon" {
		t.Errorf("Note = %q, want 'fix soon'", w.Note)
	}
}

// TestWaiver_ParseToken_NoMarkerErrors covers the no-@waiver-marker error branch.
func TestWaiver_ParseToken_NoMarkerErrors(t *testing.T) {
	if _, err := ParseToken("just some code with no marker", grammarLoc()); err == nil {
		t.Fatal("a string with no @waiver marker must error")
	}
}

// TestWaiver_ParseToken_MissingRuleIDErrors covers the empty rule-id branch.
func TestWaiver_ParseToken_MissingRuleIDErrors(t *testing.T) {
	if _, err := ParseToken("@waiver::deferred:2027-01-15", grammarLoc()); err == nil {
		t.Fatal("a token with an empty rule-id must error")
	}
}

// TestWaiver_IssueRefLooksLike_NonMatches covers non-issue-ref trailing words so
// they are captured as Note, not IssueRef (issueRefLooksLike negative branches).
func TestWaiver_IssueRefLooksLike_NonMatches(t *testing.T) {
	// A lowercase word, a bare '#', and a 'KEY-' with no number are all NOT refs.
	w, err := ParseToken("@waiver:r:deferred:2027-01-15 lowercase # ABC-", grammarLoc())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.IssueRef != "" {
		t.Errorf("IssueRef = %q, want empty (no valid issue ref present)", w.IssueRef)
	}
	if w.Note == "" {
		t.Error("Note should capture the non-issue-ref trailing words")
	}
}
