package check

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// readCoverageFixture reads a coverage-records JSON fixture from testdata.
func readCoverageFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read coverage fixture %s: %v", name, err)
	}
	return b
}

// TestCoverageRecord_RawCountsNoPrecomputedPercent asserts CoverageRecord is the
// canonical {Path, Covered, Total, Measured, Excluded, Metric} with RAW integer
// counts (Covered, Total) and NO pre-computed percent field — a structural
// assertion over the field set (CLM-010). The gate computes Covered/Total >=
// threshold and stays metric-blind, so the producer record must bake no percentage.
func TestCoverageRecord_RawCountsNoPrecomputedPercent(t *testing.T) {
	rt := reflect.TypeOf(CoverageRecord{})

	wantInt := map[string]bool{"Covered": true, "Total": true}
	for name := range wantInt {
		f, ok := rt.FieldByName(name)
		if !ok {
			t.Fatalf("CoverageRecord must carry a raw-count field %q", name)
		}
		if f.Type.Kind() != reflect.Int {
			t.Errorf("CoverageRecord.%s must be a raw int count, got %s", name, f.Type.Kind())
		}
	}

	// No float / percent field may exist on the producer record — a pre-computed
	// percentage is exactly what REQ-003 forbids.
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Type.Kind() == reflect.Float32 || f.Type.Kind() == reflect.Float64 {
			t.Errorf("CoverageRecord must carry NO float/percent field, found %q (%s)", f.Name, f.Type.Kind())
		}
		if f.Name == "Pct" || f.Name == "Percent" {
			t.Errorf("CoverageRecord must NOT carry a pre-computed percent field, found %q", f.Name)
		}
	}

	// The canonical field set must be present.
	for _, name := range []string{"Path", "Covered", "Total", "Measured", "Excluded", "Metric"} {
		if _, ok := rt.FieldByName(name); !ok {
			t.Errorf("CoverageRecord must declare field %q", name)
		}
	}
}

// TestCoverageRecord_PerFilePathKeyedNoPackageNoun asserts the record is per-FILE
// and path-keyed — Path is a file path — and there is NO Package field / no
// package-granular modeling, because a "package" noun would re-bake Go-native
// language knowledge into a language-agnostic record (CLM-012).
func TestCoverageRecord_PerFilePathKeyedNoPackageNoun(t *testing.T) {
	rt := reflect.TypeOf(CoverageRecord{})

	pathField, ok := rt.FieldByName("Path")
	if !ok || pathField.Type.Kind() != reflect.String {
		t.Fatalf("CoverageRecord must carry a string Path (file path) field")
	}
	if _, ok := rt.FieldByName("Package"); ok {
		t.Errorf("CoverageRecord must carry NO Package field — package is a Go-native noun")
	}

	// A parsed record's Path is a FILE path (carries a file extension), proving file
	// granularity rather than a package selector.
	recs, err := ParsePackCoverage(readCoverageFixture(t, "coverage-records-measured-passing.json"))
	if err != nil {
		t.Fatalf("ParsePackCoverage: %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("expected per-file records")
	}
	if filepath.Ext(recs[0].Path) == "" {
		t.Errorf("Path must be a file path (file granularity), got %q", recs[0].Path)
	}
}

// TestParseCoverage_TotalZeroPreservedAsNA asserts a no-executable-lines record
// (Total 0) parses with Total==0 preserved faithfully — ParsePackCoverage does NOT
// synthesize a 0% value, so the consumer can treat it as N/A rather than a 0%-fail
// (CLM-013). The producer must not coerce 0/0 into a percentage.
func TestParseCoverage_TotalZeroPreservedAsNA(t *testing.T) {
	recs, err := ParsePackCoverage(readCoverageFixture(t, "coverage-records-total-zero-na.json"))
	if err != nil {
		t.Fatalf("ParsePackCoverage (total-zero): %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	r := recs[0]
	if r.Total != 0 {
		t.Errorf("Total==0 must be preserved faithfully, got Total=%d", r.Total)
	}
	if r.Covered != 0 {
		t.Errorf("a no-executable-lines record carries Covered=0, got %d", r.Covered)
	}
	if !r.Measured {
		t.Errorf("a measured no-executable-lines file is still Measured (the engine saw it), got Measured=false")
	}
}

// TestParseCoverage_EmptyMetricOnMeasuredRecordFailsLoud asserts a MEASURED record
// with an empty Metric is a fail-loud parse error, never silently accepted as blank
// — an unlabeled measurement is the silent-comparison hazard in seed form (CLM-017).
func TestParseCoverage_EmptyMetricOnMeasuredRecordFailsLoud(t *testing.T) {
	_, err := ParsePackCoverage(readCoverageFixture(t, "coverage-records-empty-metric.json"))
	if err == nil {
		t.Fatal("a MEASURED record with an empty Metric must fail loud, got nil error")
	}
}

// TestParseCoverage_EmptyInputYieldsNoRecords asserts empty/whitespace coverage
// output parses to no records and no error — an engine that measured nothing yields
// an empty census, not a parse failure.
func TestParseCoverage_EmptyInputYieldsNoRecords(t *testing.T) {
	for _, in := range [][]byte{nil, {}, []byte("   \n\t ")} {
		recs, err := ParsePackCoverage(in)
		if err != nil {
			t.Errorf("empty coverage output must not error, got %v", err)
		}
		if len(recs) != 0 {
			t.Errorf("empty coverage output must yield no records, got %#v", recs)
		}
	}
}

// TestParseCoverage_SarifObjectRejected asserts a SARIF/JSON-object document (a
// leading '{') is rejected fail-loud — coverage is a DISTINCT typed output, never
// tunneled through a SARIF object's properties (CLM-007, the parser-level guard).
func TestParseCoverage_SarifObjectRejected(t *testing.T) {
	sarif := readCoverageFixture(t, "coverage-tunneled-through-sarif-properties.json")
	if _, err := ParsePackCoverage(sarif); err == nil {
		t.Fatal("a SARIF object carrying coverage in properties must be REJECTED by the coverage-records parser")
	}
}

// TestParseCoverage_MalformedArrayFailsLoud asserts a malformed coverage-records
// array (a JSON array with an unknown field / broken shape) is a fail-loud parse
// error, never silently read as zero records.
func TestParseCoverage_MalformedArrayFailsLoud(t *testing.T) {
	// An unknown field trips DisallowUnknownFields — a mis-shaped record fails loud
	// rather than dropping data silently.
	if _, err := ParsePackCoverage([]byte(`[{"path":"a.go","pct":42}]`)); err == nil {
		t.Fatal("a coverage-records array with an unknown field must fail loud (no silent drop)")
	}
}

// TestParseCoverage_LanguageAgnosticRecordShape asserts the language-agnostic
// fixture (Metric "line"/"branch", as a typescript-toolchain engine would emit)
// parses through the SAME ParsePackCoverage into the SAME CoverageRecord shape — no
// Go-specific assumption baked into the parser or record (CLM-023).
func TestParseCoverage_LanguageAgnosticRecordShape(t *testing.T) {
	recs, err := ParsePackCoverage(readCoverageFixture(t, "coverage-records-language-agnostic.json"))
	if err != nil {
		t.Fatalf("ParsePackCoverage (language-agnostic): %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 language-agnostic records, got %d", len(recs))
	}
	gotMetrics := map[string]bool{}
	for _, r := range recs {
		gotMetrics[r.Metric] = true
		if r.Total == 0 {
			t.Errorf("language-agnostic fixture records carry non-zero totals, got %#v", r)
		}
	}
	// The pack-declared metric labels are carried through unchanged — no Go-statement
	// assumption is imposed on a non-Go engine's records.
	if !gotMetrics["line"] || !gotMetrics["branch"] {
		t.Errorf("language-agnostic metrics (line/branch) must be carried unchanged, got %v", gotMetrics)
	}
}
