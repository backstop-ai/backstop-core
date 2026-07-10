package waiver

import (
	"testing"
	"time"
)

const day = 24 * time.Hour

// activeContains reports whether the Active list carries a waiver for rule.
func activeContains(res Result, rule string) bool {
	for _, w := range res.Active {
		if w.RuleID == rule {
			return true
		}
	}
	return false
}

// expiredContains reports whether the Expired list carries a waiver for rule.
func expiredContains(res Result, rule string) bool {
	for _, w := range res.Expired {
		if w.RuleID == rule {
			return true
		}
	}
	return false
}

// expiringContains reports whether the Expiring list carries a waiver for rule.
func expiringContains(res Result, rule string) bool {
	for _, w := range res.Expiring {
		if w.RuleID == rule {
			return true
		}
	}
	return false
}

// TestWaiver_DefaultDuration_FalsePositiveLongLived proves false-positive
// resolves to a long-lived default duration (CLM-015).
func TestWaiver_DefaultDuration_FalsePositiveLongLived(t *testing.T) {
	d, ok := DefaultDuration(ReasonFalsePositive)
	if !ok {
		t.Fatal("false-positive did not resolve a default duration")
	}
	if d < 300*day {
		t.Errorf("false-positive duration %v is not long-lived (want >= 300 days)", d)
	}
}

// TestWaiver_DefaultDuration_AcceptedRiskShortLived proves accepted-risk
// resolves to a short-lived default duration (CLM-016).
func TestWaiver_DefaultDuration_AcceptedRiskShortLived(t *testing.T) {
	d, ok := DefaultDuration(ReasonAcceptedRisk)
	if !ok {
		t.Fatal("accepted-risk did not resolve a default duration")
	}
	if d <= 0 || d > 120*day {
		t.Errorf("accepted-risk duration %v is not short-lived (want 0 < d <= 120 days)", d)
	}
}

// TestWaiver_DefaultDuration_DeferredShortLived proves deferred resolves to a
// short-lived default duration (CLM-017).
func TestWaiver_DefaultDuration_DeferredShortLived(t *testing.T) {
	d, ok := DefaultDuration(ReasonDeferred)
	if !ok {
		t.Fatal("deferred did not resolve a default duration")
	}
	if d <= 0 || d > 120*day {
		t.Errorf("deferred duration %v is not short-lived", d)
	}
}

// TestWaiver_DefaultDuration_ThirdPartyShortLived proves third-party resolves to
// a short-lived default duration (CLM-018).
func TestWaiver_DefaultDuration_ThirdPartyShortLived(t *testing.T) {
	d, ok := DefaultDuration(ReasonThirdParty)
	if !ok {
		t.Fatal("third-party did not resolve a default duration")
	}
	if d <= 0 || d > 120*day {
		t.Errorf("third-party duration %v is not short-lived", d)
	}
	// An unknown reason-code resolves nothing.
	if _, ok := DefaultDuration(ReasonCode("bogus")); ok {
		t.Error("an unknown reason-code must NOT resolve a default duration")
	}
}

// TestWaiver_Expiry_ActiveWaiverSuppresses proves an active (unexpired) waiver
// suppresses its finding (CLM-019).
func TestWaiver_Expiry_ActiveWaiverSuppresses(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	findings := []Finding{{RuleID: "r", File: "f", Line: 5}}
	read := lineReaderFrom("f", map[int]string{5: "x // @waiver:r:deferred:2027-01-01"})
	res := Adjudicate(findings, read, nil, now)
	if !suppressed(res, "f", 5, "r") {
		t.Fatal("active waiver did not suppress")
	}
	if !activeContains(res, "r") {
		t.Fatal("active waiver not recorded in Result.Active")
	}
}

// TestWaiver_Expiry_ExpiredWaiverRefires proves an expired waiver does NOT
// suppress — the finding re-fires under normal enforcement at the instant of
// expiry (CLM-020, Sharp Edge 5).
func TestWaiver_Expiry_ExpiredWaiverRefires(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	findings := []Finding{{RuleID: "r", File: "f", Line: 5}}
	read := lineReaderFrom("f", map[int]string{5: "x // @waiver:r:deferred:2026-05-01"})
	res := Adjudicate(findings, read, nil, now)
	if suppressed(res, "f", 5, "r") {
		t.Fatal("an expired waiver must NOT suppress; the finding must re-fire")
	}
	if !expiredContains(res, "r") {
		t.Fatal("expired waiver not recorded in Result.Expired")
	}
}

// TestWaiver_Expiry_PreExpiryWarningEmitted proves a loud pre-expiry warning is
// emitted within the grace window before a waiver expires (CLM-021). The waiver
// is still active (still suppresses) but is surfaced as Expiring.
func TestWaiver_Expiry_PreExpiryWarningEmitted(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	findings := []Finding{{RuleID: "r", File: "f", Line: 5}}
	read := lineReaderFrom("f", map[int]string{5: "x // @waiver:r:accepted-risk:2026-06-15"})
	res := Adjudicate(findings, read, nil, now)
	if !suppressed(res, "f", 5, "r") {
		t.Fatal("a waiver in its grace window is still active and must suppress")
	}
	if !expiringContains(res, "r") {
		t.Fatal("a waiver within the grace window must be surfaced as Expiring (pre-expiry warning)")
	}
}
