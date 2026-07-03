package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// minimalCoverageConsumer is a TEST-LOCAL stand-in for SPEC-041's
// StepCoverageThreshold (fenced to Seed 3). It exercises the producer→consumer
// boundary: it consumes the canonical []check.CoverageRecord the dispatch produces,
// computes the verdict from RAW COUNTS (Covered/Total — staying metric-BLIND), skips
// a Total==0 record as N/A, and SURFACES the pack-declared Metric label on its
// report lines. It interprets Metric for NOTHING — it only surfaces it. The real
// threshold verdict logic is SPEC-041's; this proves the records flow and the
// conventions hold at the boundary, not the gate step itself.
type minimalCoverageConsumer struct {
	thresholdPct int
}

// coverageReportLine is one surfaced consumer line: the metric label is carried
// onto the report so a statement-% is never silently compared against a branch-%.
type coverageReportLine struct {
	Path    string
	Metric  string
	NA      bool
	Shortfall bool
}

func (c minimalCoverageConsumer) verdict(records []check.CoverageRecord) []coverageReportLine {
	lines := make([]coverageReportLine, 0, len(records))
	for _, r := range records {
		line := coverageReportLine{Path: r.Path, Metric: r.Metric}
		if r.Total == 0 {
			// Total==0 ⇒ N/A; a naive 0/0 → 0% would have failed. Skipped, not red.
			line.NA = true
			lines = append(lines, line)
			continue
		}
		// Metric-blind: the verdict is computed ONLY from Covered/Total, never from
		// the metric label.
		line.Shortfall = r.Covered*100 < c.thresholdPct*r.Total
		lines = append(lines, line)
	}
	return lines
}

// dispatchRecordsForConvert runs dispatchPackCoverage over a coverage manifest whose
// real on-disk convert emits the supplied coverage-records JSON.
func dispatchRecordsForConvert(t *testing.T, convert string) []check.CoverageRecord {
	t.Helper()
	stubSandboxedRunStdout(t, nil)
	packsDir := coverageRoutingPacksDir(t, convert)
	manifest := gateTypeRoutingManifest("cov-engine", engine.GateTypeCoverage)
	runner := &fixtureRunner{byCmd: map[string][]byte{"neutral-tool": []byte("raw")}}
	records, err := dispatchPackCoverage([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackCoverage: %v", err)
	}
	return records
}

// TestCoverage_MetricLabelCarriedAndSurfacedOnReport proves the pack-declared Metric
// label (e.g. "statement") is carried through the channel unchanged and SURFACED on
// the report surface for measured records, so a statement-% is never silently
// compared against a branch-% under one number (CLM-015).
func TestCoverage_MetricLabelCarriedAndSurfacedOnReport(t *testing.T) {
	records := dispatchRecordsForConvert(t, coverageRecordsJSON())
	consumer := minimalCoverageConsumer{thresholdPct: 80}
	lines := consumer.verdict(records)

	if len(lines) == 0 {
		t.Fatal("expected surfaced report lines")
	}
	for _, ln := range lines {
		if ln.Metric == "" {
			t.Errorf("the pack-declared Metric must be SURFACED on the report for %q, got empty", ln.Path)
		}
		if ln.Metric != "statement" {
			t.Errorf("the Metric label must be carried through UNCHANGED, got %q", ln.Metric)
		}
	}
}

// TestCoverage_GateIsMetricBlind proves the consumer stays metric-BLIND — two records
// with DIFFERENT Metric labels but identical Covered/Total produce identical verdicts;
// the gate never interprets/compares/branches on Metric, only surfaces it (CLM-016).
func TestCoverage_GateIsMetricBlind(t *testing.T) {
	// Same raw counts (30/100 — below an 80% threshold), different metric labels.
	stmt := `[{"path":"a.go","covered":30,"total":100,"measured":true,"excluded":false,"metric":"statement"}]`
	branch := `[{"path":"b.ts","covered":30,"total":100,"measured":true,"excluded":false,"metric":"branch"}]`

	consumer := minimalCoverageConsumer{thresholdPct: 80}
	stmtLines := consumer.verdict(dispatchRecordsForConvert(t, stmt))
	branchLines := consumer.verdict(dispatchRecordsForConvert(t, branch))

	if len(stmtLines) != 1 || len(branchLines) != 1 {
		t.Fatalf("expected one line each, got %d and %d", len(stmtLines), len(branchLines))
	}
	// Identical counts ⇒ identical verdicts, regardless of the (different) metric.
	if stmtLines[0].Shortfall != branchLines[0].Shortfall || !stmtLines[0].Shortfall {
		t.Errorf("the gate must be metric-BLIND: identical counts must yield identical verdicts, got %+v vs %+v", stmtLines[0], branchLines[0])
	}
	if stmtLines[0].Metric == branchLines[0].Metric {
		t.Errorf("test precondition: the two records must carry DIFFERENT metric labels")
	}
}

// TestCoverage_TotalZeroIsNAnotFail proves a Total==0 record is treated as N/A by the
// verdict (skipped), never a 0%-fail — feeding it yields no coverage shortfall, where
// a naive 0/0 → 0% would have failed (CLM-014).
func TestCoverage_TotalZeroIsNAnotFail(t *testing.T) {
	na := `[{"path":"iface.go","covered":0,"total":0,"measured":true,"excluded":false,"metric":"statement"}]`
	consumer := minimalCoverageConsumer{thresholdPct: 80}
	lines := consumer.verdict(dispatchRecordsForConvert(t, na))

	if len(lines) != 1 {
		t.Fatalf("expected one line, got %d", len(lines))
	}
	if !lines[0].NA {
		t.Errorf("a Total==0 record must be N/A, got %+v", lines[0])
	}
	if lines[0].Shortfall {
		t.Errorf("a Total==0 record must NEVER be a 0%%-fail shortfall, got %+v", lines[0])
	}
}

// TestCoverageRecord_GateComputesPercentFromRawCounts proves the consumer computes
// the threshold verdict from Covered/Total (staying metric-blind) — feeding raw
// counts, the pass/fail verdict matches Covered/Total >= threshold, with the producer
// never emitting a percentage (CLM-011).
func TestCoverageRecord_GateComputesPercentFromRawCounts(t *testing.T) {
	// passing 92/100 (>=80), failing 30/100 (<80).
	records := dispatchRecordsForConvert(t, coverageRecordsJSON())
	consumer := minimalCoverageConsumer{thresholdPct: 80}
	lines := consumer.verdict(records)

	got := map[string]bool{}
	for _, ln := range lines {
		got[ln.Path] = ln.Shortfall
	}
	if got["pkg/svc/handler.go"] {
		t.Errorf("92/100 >= 80%% must pass, got shortfall")
	}
	if !got["pkg/svc/shortfall.go"] {
		t.Errorf("30/100 < 80%% must fail, got pass")
	}
	// The producer emitted RAW COUNTS — assert no record carries a synthesized percent
	// (the record type has no percent field; the verdict came from counts).
	for _, r := range records {
		if r.Total > 0 && r.Covered > r.Total {
			t.Errorf("raw counts must be coherent (Covered<=Total), got %#v — a percent leaked in", r)
		}
	}
}
