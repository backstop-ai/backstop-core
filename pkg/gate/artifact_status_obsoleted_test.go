package gate

// ISSUE-048 facet-3 tests (CLM-008): `obsoleted` joins the retired-terminal class the
// drift dimension EXCLUDES. These exercise the REAL ClassifyArtifactStatus + the resolver
// over the status_drift_vocab fixture root. Package `gate` (internal) to reuse the
// presentSet/hasFail helpers in status_drift_test.go.

import (
	"testing"
)

// vocabDriftRoot is the drift-dimension fixture project root (obsoleted + plan test_names).
func vocabDriftRoot() string { return "testdata/status_drift_vocab/root" }

// findRecord returns the resolved record for an artifact id, or fails the test.
func findRecord(t *testing.T, records []ArtifactStatusRecord, id string) ArtifactStatusRecord {
	t.Helper()
	for _, r := range records {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no resolved record for %q (have %d records)", id, len(records))
	return ArtifactStatusRecord{}
}

// errorViolationsForFile returns the error-severity drift violations attributed to a
// given file path.
func errorViolationsForFile(vs []Violation, path string) []Violation {
	var out []Violation
	for _, v := range vs {
		if v.Severity == "error" && v.File == path {
			out = append(out, v)
		}
	}
	return out
}

// TestClassifyArtifactStatus_ObsoletedIsRetiredTerminal (CLM-008): `obsoleted` maps to
// ClassRetiredTerminal for every artifact type, so the drift dimension excludes it.
func TestClassifyArtifactStatus_ObsoletedIsRetiredTerminal(t *testing.T) {
	for _, kind := range []ArtifactKind{KindIssue, KindSpec, KindPlan, KindDirective} {
		if got := ClassifyArtifactStatus(kind, "obsoleted"); got != ClassRetiredTerminal {
			t.Errorf("ClassifyArtifactStatus(%q, obsoleted) = %v, want retired-terminal", kind, got)
		}
	}
	// Non-over-broadening guard: the success terminals stay success-terminal.
	if ClassifyArtifactStatus(KindIssue, "closed") != ClassSuccessTerminal {
		t.Error("issue `closed` must remain success-terminal (obsoleted must not over-broaden retired)")
	}
	if ClassifyArtifactStatus(KindPlan, "completed") != ClassSuccessTerminal {
		t.Error("plan `completed` must remain success-terminal")
	}
}

// TestStatusDrift_ExcludesObsoletedArtifact (CLM-008): an obsoleted artifact declaring a
// mandated test that is ABSENT produces NO broken-promise violation — it is excluded as
// retired, self-documenting via obsoleted-by.
func TestStatusDrift_ExcludesObsoletedArtifact(t *testing.T) {
	res, err := ResolveArtifactStatus(vocabDriftRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	obs := findRecord(t, res.Records, "ISSUE-840")
	if obs.Class != ClassRetiredTerminal {
		t.Fatalf("ISSUE-840 (obsoleted) class = %v, want retired-terminal", obs.Class)
	}
	// Sanity: the fixture DOES declare a mandated test that is absent — so if it were
	// NOT excluded as retired the exclusion path would be untested.
	if len(obs.MandatedTests) == 0 {
		t.Fatal("ISSUE-840 fixture must declare a mandated test to exercise the exclusion path")
	}

	drift := ClassifyStatusDrift(res.Records, presentSet())
	if got := errorViolationsForFile(drift.Violations, obs.Path); len(got) != 0 {
		t.Errorf("obsoleted ISSUE-840 must produce NO broken-promise violation, got %d: %+v", len(got), got)
	}
}
