package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// bunConvertScript returns the path to the bun pack's lcov coverage convert.
func bunConvertScript(t *testing.T) string {
	t.Helper()
	return filepath.Join(bunPackRoot(t), "scripts", "coverage-to-records.sh")
}

// readBunLcovFixture reads an lcov .info convert-input fixture from the
// bun-coverage testdata dir (the Phase-1 DATA the real convert runs over).
func readBunLcovFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "bun-coverage", name))
	if err != nil {
		t.Fatalf("read lcov fixture %s: %v", name, err)
	}
	return b
}

// runBunConvert runs the REAL bun lcov convert over the fixture and parses its
// stdout through the EXISTING check.ParsePackCoverage (the SPEC-044 contract,
// consumed not redefined). Returns the canonical records.
func runBunConvert(t *testing.T, fixture string) []check.CoverageRecord {
	t.Helper()
	out, err := runConvertScriptDirect(bunConvertScript(t), readBunLcovFixture(t, fixture))
	if err != nil {
		t.Fatalf("running coverage-to-records.sh over %s: %v\noutput: %s", fixture, err, out)
	}
	records, err := check.ParsePackCoverage(out)
	if err != nil {
		t.Fatalf("parsing convert output as coverage-records: %v\noutput: %s", err, out)
	}
	return records
}

// recordFor returns the record for (path, metric) or fails.
func recordFor(t *testing.T, records []check.CoverageRecord, path, metric string) check.CoverageRecord {
	t.Helper()
	for _, r := range records {
		if r.Path == path && r.Metric == metric {
			return r
		}
	}
	t.Fatalf("no record for (path=%q, metric=%q); records=%#v", path, metric, records)
	return check.CoverageRecord{}
}

// TestBunCoverageConvert_EmitsLineAndBranchPerFileSharingPath proves that for each
// measured source file the convert emits EXACTLY TWO records sharing the same Path
// — one metric "line", one metric "branch" — never a single collapsed record
// (CLM-015).
func TestBunCoverageConvert_EmitsLineAndBranchPerFileSharingPath(t *testing.T) {
	records := runBunConvert(t, "lcov-line-branch.info")
	byMetric := map[string]int{}
	for _, r := range records {
		if r.Path != "src/app.ts" {
			t.Errorf("unexpected record path %q (the fixture measures one file, src/app.ts)", r.Path)
		}
		byMetric[r.Metric]++
	}
	if len(records) != 2 {
		t.Fatalf("a measured file must yield EXACTLY two records (line + branch), got %d: %#v", len(records), records)
	}
	if byMetric["line"] != 1 || byMetric["branch"] != 1 {
		t.Errorf("the two records must be one metric \"line\" and one metric \"branch\" (never collapsed), got %v", byMetric)
	}
}

// TestBunCoverageConvert_LineCountsFromLcovLfLh proves the line record carries raw
// counts covered=LH, total=LF — not a pre-computed percentage (CLM-016). The
// fixture declares LF:10 LH:8.
func TestBunCoverageConvert_LineCountsFromLcovLfLh(t *testing.T) {
	line := recordFor(t, runBunConvert(t, "lcov-line-branch.info"), "src/app.ts", "line")
	if line.Covered != 8 || line.Total != 10 {
		t.Errorf("line record must carry raw covered=LH(8), total=LF(10), got covered=%d total=%d", line.Covered, line.Total)
	}
	if !line.Measured {
		t.Error("line record must be measured")
	}
}

// TestBunCoverageConvert_BranchCountsFromLcovBrfBrh proves the branch record
// carries raw counts covered=BRH, total=BRF — not a pre-computed percentage
// (CLM-017). The fixture declares BRF:4 BRH:3.
func TestBunCoverageConvert_BranchCountsFromLcovBrfBrh(t *testing.T) {
	branch := recordFor(t, runBunConvert(t, "lcov-line-branch.info"), "src/app.ts", "branch")
	if branch.Covered != 3 || branch.Total != 4 {
		t.Errorf("branch record must carry raw covered=BRH(3), total=BRF(4), got covered=%d total=%d", branch.Covered, branch.Total)
	}
}

// TestBunCoverageConvert_NoBranchableCodeEmitsTotalZeroBranchRecord proves a BRF:0
// file emits a branch record with total:0 (the N/A cell, never coerced to 0%)
// carrying metric:"branch" + measured:true, while its line record is still
// measured (CLM-018). The empty-metric fail-loud sharp edge: the N/A cell must
// still carry the metric label.
func TestBunCoverageConvert_NoBranchableCodeEmitsTotalZeroBranchRecord(t *testing.T) {
	records := runBunConvert(t, "lcov-no-branch-na.info")
	branch := recordFor(t, records, "src/pure.ts", "branch")
	if branch.Total != 0 {
		t.Errorf("a BRF:0 file's branch record must carry total:0 (N/A, never coerced), got total=%d", branch.Total)
	}
	if branch.Metric != "branch" {
		t.Errorf("the N/A branch cell must still carry metric \"branch\" (empty metric is fail-loud upstream), got %q", branch.Metric)
	}
	if !branch.Measured {
		t.Error("total:0 is N/A, NOT unmeasured — the branch record must still be measured:true")
	}
	// The line record is still fully measured (LF:6 LH:6).
	line := recordFor(t, records, "src/pure.ts", "line")
	if !line.Measured || line.Covered != 6 || line.Total != 6 {
		t.Errorf("the line record must still be measured (covered=6 total=6) even when branches are N/A, got %#v", line)
	}
}

// TestBunCoverageConvert_RecordsParseThroughCanonicalParsePackCoverageNoNewType
// proves the emitted records parse through the EXISTING check.ParsePackCoverage
// into canonical check.CoverageRecords with NO new type/field/schema fork —
// DisallowUnknownFields accepts exactly {path,covered,total,measured,excluded,
// metric} and would reject any stray field (CLM-019).
func TestBunCoverageConvert_RecordsParseThroughCanonicalParsePackCoverageNoNewType(t *testing.T) {
	out, err := runConvertScriptDirect(bunConvertScript(t), readBunLcovFixture(t, "lcov-line-branch.info"))
	if err != nil {
		t.Fatalf("convert: %v\n%s", err, out)
	}
	// The canonical strict parser (DisallowUnknownFields) accepts the bun output —
	// proof the convert uses ONLY the canonical keys, no new field.
	records, err := check.ParsePackCoverage(out)
	if err != nil {
		t.Fatalf("the bun convert output must parse through the EXISTING strict parser with no new field: %v\noutput: %s", err, out)
	}
	if len(records) == 0 {
		t.Fatal("expected canonical records from the bun convert")
	}
	// A stray field WOULD be rejected — guard that the parser is genuinely strict, so
	// this test is a real no-schema-fork proof and not a vacuous accept.
	if _, err := check.ParsePackCoverage([]byte(`[{"path":"x","covered":1,"total":1,"measured":true,"excluded":false,"metric":"line","branches":2}]`)); err == nil {
		t.Error("ParsePackCoverage must reject a stray field (DisallowUnknownFields); the no-new-field guarantee depends on it")
	}
}

// TestBunCoverage_LineAndBranchIndexUnderPathMetricNoCollision proves the
// line+branch records index under the SPEC-044 (path, metric) key for one Path
// with NO collision — both metrics survive, neither overwrites the other
// (CLM-020).
func TestBunCoverage_LineAndBranchIndexUnderPathMetricNoCollision(t *testing.T) {
	records := runBunConvert(t, "lcov-line-branch.info")
	type key struct{ path, metric string }
	index := map[key]check.CoverageRecord{}
	for _, r := range records {
		k := key{r.Path, r.Metric}
		if _, dup := index[k]; dup {
			t.Fatalf("collision: two records for (path=%q, metric=%q) — the (path, metric) key must be unique per producer", r.Path, r.Metric)
		}
		index[k] = r
	}
	if _, ok := index[key{"src/app.ts", "line"}]; !ok {
		t.Error("the line metric must survive under (src/app.ts, line)")
	}
	if _, ok := index[key{"src/app.ts", "branch"}]; !ok {
		t.Error("the branch metric must survive under (src/app.ts, branch) — no last-write-wins collapse")
	}
}

// TestBunCoverageConvert_RawCountsNotPrecomputedPercentNoAggregation is the
// DENYLIST: feeding line 95/100 + branch 60/100 yields two raw-count records,
// never one aggregated/averaged value or a pre-computed percent (CLM-021).
func TestBunCoverageConvert_RawCountsNotPrecomputedPercentNoAggregation(t *testing.T) {
	records := runBunConvert(t, "lcov-raw-counts.info")
	if len(records) != 2 {
		t.Fatalf("raw-counts fixture must yield two records (no aggregation into one), got %d: %#v", len(records), records)
	}
	line := recordFor(t, records, "src/big.ts", "line")
	branch := recordFor(t, records, "src/big.ts", "branch")
	// The two metrics keep their DISTINCT raw ratios — proof nothing was averaged
	// (95/100 and 60/100 would collapse to a single ~77/100 if aggregated).
	if line.Covered != 95 || line.Total != 100 {
		t.Errorf("line must stay raw 95/100, got %d/%d", line.Covered, line.Total)
	}
	if branch.Covered != 60 || branch.Total != 100 {
		t.Errorf("branch must stay raw 60/100, got %d/%d", branch.Covered, branch.Total)
	}
	if line.Covered == branch.Covered {
		t.Error("line and branch raw counts collapsed to one value — the convert must NOT aggregate the two metrics")
	}
}
