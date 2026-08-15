package initialize

import (
	"strings"
	"testing"
)

// TestInit_GateFindingsAreGroupedByDimensionWithCounts (SPEC-069 CLM-064).
//
// Findings are grouped by gate DIMENSION with a COUNT per dimension, so the report
// presents STRUCTURE rather than a wall of findings. The dimension names are
// backstop's own universal vocabulary, so the grouping introduces no tool or language
// noun into init's output.
func TestInit_GateFindingsAreGroupedByDimensionWithCounts(t *testing.T) {
	_, _, gates, _, _ := defaultFakes()
	gates.counts = []DimensionCount{
		{Dimension: "pack_engines", Count: 12},
		{Dimension: "coverage_threshold", Count: 3},
		{Dimension: "contract_signature", Count: 1},
	}

	report, observations := stepObserve("/project", gates)

	if gates.calls != 1 {
		t.Fatalf("the gate ran %d times, want exactly once", gates.calls)
	}
	if len(observations) != len(gates.counts) {
		t.Fatalf("the step returned %d dimension counts, want %d", len(observations), len(gates.counts))
	}
	for i, want := range gates.counts {
		if observations[i] != want {
			t.Fatalf("dimension %d was %+v, want %+v", i, observations[i], want)
		}
		if !strings.Contains(report.Detail, want.Dimension) {
			t.Fatalf("the report does not name the dimension %q.\ngot: %s", want.Dimension, report.Detail)
		}
	}
	// The COUNT is what makes it structure rather than a list of names.
	for _, want := range []string{"12", "3", "1"} {
		if !strings.Contains(report.Detail, want) {
			t.Fatalf("the report does not carry the per-dimension count %q.\ngot: %s", want, report.Detail)
		}
	}
}

// TestInit_ObservationSummaryCarriesNoVerdictLanguage (SPEC-069 CLM-065).
//
// The summary is phrased as WHAT WAS NOTICED. Verdict language would tell a consumer
// their brand new project is already failing, about findings it inherited and that init
// did not cause — which is why REQ-014 makes pre-existing findings observation rather
// than failure in the first place.
func TestInit_ObservationSummaryCarriesNoVerdictLanguage(t *testing.T) {
	_, _, gates, _, _ := defaultFakes()
	gates.counts = []DimensionCount{
		{Dimension: "pack_engines", Count: 41},
		{Dimension: "test_substantiveness", Count: 7},
	}

	report, _ := stepObserve("/project", gates)

	if report.Outcome == OutcomeBrokenPromise {
		t.Fatalf("pre-existing findings failed the observe step: %s", report.Detail)
	}

	detail := strings.ToLower(report.Detail)
	for _, verdict := range []string{"violation", "failed", "failure", "error", "fail:", "passed", "verdict", "blocked"} {
		if strings.Contains(detail, verdict) {
			t.Fatalf("the observation summary uses the verdict word %q; findings are what was NOTICED, not a judgement on a project init has only just started governing.\ngot: %s",
				verdict, report.Detail)
		}
	}
	if !strings.Contains(detail, "noticed") {
		t.Fatalf("the summary is not phrased as what was noticed.\ngot: %s", report.Detail)
	}
}

// TestInit_ObserveReportsACleanGateWithoutClaimingAVerdict is additive: a gate that
// found nothing must also be reported without verdict language, or the neutral phrasing
// is only skin-deep on the findings path.
func TestInit_ObserveReportsACleanGateWithoutClaimingAVerdict(t *testing.T) {
	_, _, gates, _, _ := defaultFakes()
	gates.counts = []DimensionCount{{Dimension: "pack_engines", Count: 0}}

	report, _ := stepObserve("/project", gates)

	if report.Outcome != OutcomeDelivered {
		t.Fatalf("a clean gate reported %v, want OutcomeDelivered", report.Outcome)
	}
	detail := strings.ToLower(report.Detail)
	for _, verdict := range []string{"passed", "pass:", "green", "clean bill"} {
		if strings.Contains(detail, verdict) {
			t.Fatalf("a clean gate was reported with the verdict word %q.\ngot: %s", verdict, report.Detail)
		}
	}
}
