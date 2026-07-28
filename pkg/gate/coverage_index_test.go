package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// readTestdata reads a raw fixture file from pkg/gate/testdata so the bun-shape
// tests can feed the EXISTING check.ParsePackCoverage the on-the-wire bytes a
// producer emits (not a pre-unmarshaled record set).
func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return data
}

// dupKeyPresent reports whether the index's duplicate-key return value reports a
// (path, metric) pair — the loud, not-last-wins signal. The dup keys are formatted
// so both the path and the metric label are recoverable.
func dupKeyPresent(dupKeys []string, path, metric string) bool {
	for _, k := range dupKeys {
		if strings.Contains(k, path) && strings.Contains(k, metric) {
			return true
		}
	}
	return false
}

// TestIndexByPathMetric_LineAndBranchCoexistNoCollision (CLM-001): a file carrying
// two records under one Path with distinct metrics (line and branch) indexes so
// BOTH are retained — the path's inner map has exactly two metric entries, neither
// overwriting the other.
func TestIndexByPathMetric_LineAndBranchCoexistNoCollision(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
		{Path: "src/foo.ts", Covered: 60, Total: 100, Measured: true, Metric: "branch"},
	}
	byPathMetric, dupKeys := indexCoverageByPathMetric(records)

	if len(dupKeys) != 0 {
		t.Fatalf("distinct metrics on one path are NOT duplicates, got dup keys %#v", dupKeys)
	}
	inner, ok := byPathMetric["src/foo.ts"]
	if !ok {
		t.Fatalf("expected an index entry for src/foo.ts, got %#v", byPathMetric)
	}
	if len(inner) != 2 {
		t.Fatalf("expected exactly two metric entries (line, branch), got %d: %#v", len(inner), inner)
	}
	line, hasLine := inner["line"]
	branch, hasBranch := inner["branch"]
	if !hasLine || !hasBranch {
		t.Fatalf("both line and branch must be retained, got %#v", inner)
	}
	if line.Covered != 95 || branch.Covered != 60 {
		t.Errorf("neither record may overwrite the other: line=%d/%d branch=%d/%d",
			line.Covered, line.Total, branch.Covered, branch.Total)
	}
}

// TestIndexByPathMetric_SingleMetricIndexesToOneEntry (CLM-002): a single-metric
// file (one statement record per path) indexes to exactly one (path, metric)
// entry — the backward-compatible shape, proving the (path, metric) index is a
// strict generalization of the old one-record-per-path index.
func TestIndexByPathMetric_SingleMetricIndexesToOneEntry(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "pkg/a/file.go", Covered: 80, Total: 100, Measured: true, Metric: "statement"},
	}
	byPathMetric, dupKeys := indexCoverageByPathMetric(records)

	if len(dupKeys) != 0 {
		t.Fatalf("a single-metric file has no duplicates, got %#v", dupKeys)
	}
	inner, ok := byPathMetric["pkg/a/file.go"]
	if !ok {
		t.Fatalf("expected an index entry for pkg/a/file.go, got %#v", byPathMetric)
	}
	if len(inner) != 1 {
		t.Fatalf("a single-metric file must index to exactly one entry, got %d: %#v", len(inner), inner)
	}
	if _, ok := inner["statement"]; !ok {
		t.Errorf("expected the lone statement metric entry, got %#v", inner)
	}
}

// TestIndexByPathMetric_DuplicatePathMetricFailsLoud (CLM-003): two records sharing
// the SAME (path, metric) are reported in the index's duplicate-key return value (a
// duplicate-measurement producer defect) rather than silently collapsed to one
// survivor — the loud-not-last-wins signal.
func TestIndexByPathMetric_DuplicatePathMetricFailsLoud(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "src/foo.ts", Covered: 95, Total: 100, Measured: true, Metric: "line"},
		{Path: "src/foo.ts", Covered: 40, Total: 100, Measured: true, Metric: "line"},
	}
	_, dupKeys := indexCoverageByPathMetric(records)

	if len(dupKeys) == 0 {
		t.Fatalf("a duplicate (path, metric) must be reported loudly, not silently collapsed; got no dup keys")
	}
	if !dupKeyPresent(dupKeys, "src/foo.ts", "line") {
		t.Errorf("the duplicate (path, metric) must name the path and metric, got %#v", dupKeys)
	}
}

// TestIndexByPathMetric_QualifiedPathSuffixResolvesPerMetric (CLM-004): a record set
// emitted under a module/namespace-qualified Path with metrics line and branch
// resolves via resolveCoverageRecordsForPath for the repo-relative scope path
// returning BOTH metric records; an AMBIGUOUS suffix (two qualified paths ending the
// same way) is no-match so the loud guards fire.
func TestIndexByPathMetric_QualifiedPathSuffixResolvesPerMetric(t *testing.T) {
	records := []check.CoverageRecord{
		{Path: "github.com/org/repo/pkg/x/f.go", Covered: 90, Total: 100, Measured: true, Metric: "line"},
		{Path: "github.com/org/repo/pkg/x/f.go", Covered: 70, Total: 100, Measured: true, Metric: "branch"},
	}
	byPathMetric, _ := indexCoverageByPathMetric(records)

	metrics, ok := resolveCoverageRecordsForPath(byPathMetric, "pkg/x/f.go")
	if !ok {
		t.Fatalf("a module-qualified path must resolve for its repo-relative scope path, got no match")
	}
	if len(metrics) != 2 {
		t.Fatalf("the qualified-suffix fallback must return BOTH metric records, got %d: %#v", len(metrics), metrics)
	}
	if _, ok := metrics["line"]; !ok {
		t.Errorf("expected the line metric in the resolved map, got %#v", metrics)
	}
	if _, ok := metrics["branch"]; !ok {
		t.Errorf("expected the branch metric in the resolved map, got %#v", metrics)
	}

	// Ambiguous suffix: two distinct qualified paths both ending in "/x/f.go" is
	// no-match, so the loud guards fire rather than silently picking one.
	ambiguous := []check.CoverageRecord{
		{Path: "github.com/org/repo/a/x/f.go", Covered: 90, Total: 100, Measured: true, Metric: "line"},
		{Path: "github.com/org/repo/b/x/f.go", Covered: 70, Total: 100, Measured: true, Metric: "line"},
	}
	ambIndex, _ := indexCoverageByPathMetric(ambiguous)
	if _, ok := resolveCoverageRecordsForPath(ambIndex, "x/f.go"); ok {
		t.Errorf("an ambiguous suffix must be treated as no-match, but it resolved")
	}
}

// TestBunShape_LineAndBranchParseAndIndexThroughCanonicalType (CLM-022): the bun
// fixture parses through the EXISTING check.ParsePackCoverage into canonical
// check.CoverageRecords and indexes under (path, metric) with no collision — no new
// type, no new field, no schema fork.
func TestBunShape_LineAndBranchParseAndIndexThroughCanonicalType(t *testing.T) {
	data := readTestdata(t, "coverage-bun-line-branch.json")
	records, err := check.ParsePackCoverage(data)
	if err != nil {
		t.Fatalf("the bun fixture must parse through the canonical parser, got %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("expected 4 canonical records (line+branch over two files), got %d", len(records))
	}

	byPathMetric, dupKeys := indexCoverageByPathMetric(records)
	if len(dupKeys) != 0 {
		t.Fatalf("a bun-shaped set has no duplicate (path, metric), got %#v", dupKeys)
	}
	foo, ok := byPathMetric["src/foo.ts"]
	if !ok || len(foo) != 2 {
		t.Fatalf("src/foo.ts must index line+branch with no collision, got %#v", foo)
	}
	if foo["line"].Covered != 95 || foo["branch"].Covered != 60 {
		t.Errorf("the bun line/branch records must survive intact, got %#v", foo)
	}
}
