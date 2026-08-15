package initialize

import (
	"errors"
	"strings"
	"testing"
)

// TestInit_DelegatesBaselineSeedingExactlyOnce (SPEC-069 CLM-061).
//
// When a seeding implementation IS supplied to the seam, init invokes it EXACTLY ONCE
// and reports what it returned. The call COUNT is asserted, not just the outcome: a
// step that seeded twice would produce the same report while doing the work twice.
func TestInit_DelegatesBaselineSeedingExactlyOnce(t *testing.T) {
	seeder := &fakeBaselineSeeder{path: ".backstop/baseline.json"}

	report := stepBaseline("/project", seeder)

	if seeder.calls != 1 {
		t.Fatalf("the seeder was called %d times, want exactly once", seeder.calls)
	}
	if report.Outcome != OutcomeDelivered {
		t.Fatalf("baseline step reported %v (%s), want OutcomeDelivered when a seeder is available", report.Outcome, report.Detail)
	}
	if !strings.Contains(report.Detail, ".backstop/baseline.json") {
		t.Fatalf("the report does not name what the seeder returned.\ngot: %s", report.Detail)
	}
}

// TestInit_AbsentBaselineSeederIsReportedAsAGapWithoutFailing (SPEC-069 CLM-062).
//
// With no seeder available, the step reports the gap NAMING ISSUE-056 as its owner and
// does NOT fail the run — the standing loud-is-not-blocking rule: an un-adopted
// capability is a missing benefit, not a broken promise.
//
// "No seeder available" is a seeder RETURNING ErrBaselineSeedingUnavailable, not a nil
// field: NewRunner is fail-closed and a nil seam is unconstructable. The sentinel is
// therefore driven THROUGH the seam here, exactly as production drives it.
func TestInit_AbsentBaselineSeederIsReportedAsAGapWithoutFailing(t *testing.T) {
	seeder := &unavailableSeeder{}

	report := stepBaseline("/project", seeder)

	if seeder.calls != 1 {
		t.Fatalf("the unavailable seeder was called %d times, want exactly once — the step still delegates", seeder.calls)
	}
	if report.Outcome == OutcomeBrokenPromise {
		t.Fatalf("an absent seeding capability failed the run: %s", report.Detail)
	}
	if report.Outcome != OutcomeCapabilityAbsent {
		t.Fatalf("baseline step reported %v, want OutcomeCapabilityAbsent", report.Outcome)
	}
	if !strings.Contains(report.Detail, baselineOwner) {
		t.Fatalf("the gap report does not name %s as the owner of the missing machinery.\ngot: %s", baselineOwner, report.Detail)
	}
}

// TestInit_BaselineSeederFailureIsNotTreatedAsCapabilityAbsent is the falsifier for
// the claim above. A seeder that fails for a REAL reason is a delivered-step failure,
// never a capability-absent report — an implementation that absorbed every error into
// the reassuring branch would pass CLM-062 while silently swallowing a genuine break.
func TestInit_BaselineSeederFailureIsNotTreatedAsCapabilityAbsent(t *testing.T) {
	seeder := &fakeBaselineSeeder{err: errors.New("the baseline could not be written: disk full")}

	report := stepBaseline("/project", seeder)

	if report.Outcome == OutcomeCapabilityAbsent {
		t.Fatalf("a real seeding failure was reported as capability-absent; only ErrBaselineSeedingUnavailable means the capability is missing.\ngot: %s", report.Detail)
	}
	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("a real seeding failure reported %v, want OutcomeBrokenPromise", report.Outcome)
	}
	if !strings.Contains(report.Detail, "disk full") {
		t.Fatalf("the report does not surface the underlying error.\ngot: %s", report.Detail)
	}
	if strings.Contains(report.Detail, baselineOwner) {
		t.Fatalf("a real seeding failure was blamed on %s; that owner names the ABSENT capability, not a broken one.\ngot: %s", baselineOwner, report.Detail)
	}
}
