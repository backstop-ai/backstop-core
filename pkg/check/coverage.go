package check

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// CoverageRecord is the canonical producer-side coverage record: one FILE's
// coverage as normalized by a coverage engine's convert (SPEC-042 REQ-003). It is
// the SECOND normalized output type's carrier — DISTINCT from a SARIF finding — and
// the SINGLE shared type that crosses the producer (dispatchPackCoverage) and the
// consumer (SPEC-041's coverage step); SPEC-041's drafted {Path, Pct, Measured,
// Excluded} is RECONCILED to this (Pct -> Covered/Total + Metric), so there is no
// second divergent shape and no lossy translation (REQ-006).
//
// The record carries RAW COUNTS (Covered/Total), NEVER a pre-computed percent: the
// GATE computes Covered/Total >= threshold so it stays metric-BLIND and the pack
// bakes no percentage (REQ-003). Granularity is per-FILE — Path is the
// toolchain-declared file path and there is NO "package" noun (package is a
// Go-native concept that would re-bake language knowledge). A Total==0 record
// (no executable lines: pure declarations/interfaces) is N/A, never a 0%-fail
// (REQ-004). Metric is a PACK-DECLARED label (statement/line/branch/…) surfaced on
// the report but NEVER interpreted by the gate (REQ-005).
type CoverageRecord struct {
	// Path is the toolchain-declared FILE path (file granularity; no package noun).
	Path string `json:"path"`
	// Covered is the raw count of covered units (the gate computes the percentage).
	Covered int `json:"covered"`
	// Total is the raw count of measurable units. Total==0 ⇒ N/A (no executable
	// lines), preserved faithfully and never coerced to a 0% value (REQ-004).
	Total int `json:"total"`
	// Measured records whether the engine measured this file. The
	// measured-and-passed vs not-measured distinction is the one SARIF-as-findings
	// structurally cannot carry — the load-bearing reason coverage is not SARIF
	// (REQ-002).
	Measured bool `json:"measured"`
	// Excluded marks a pack-DECLARED exclusion (generated/vendored/no-executable).
	Excluded bool `json:"excluded"`
	// Metric is the pack-declared measurement label (statement/line/branch/…). It is
	// surfaced on the report and NEVER interpreted by the gate; an empty Metric on a
	// measured record is a fail-loud error (REQ-005).
	Metric string `json:"metric"`
}

// ParsePackCoverage parses a coverage engine's normalized coverage-records JSON
// (the SECOND output type, DISTINCT from SARIF findings) into []CoverageRecord — the
// coverage analogue of ParsePackFindings (SPEC-042 REQ-001/REQ-004/REQ-005).
//
// It is NOT SARIF and MUST NOT accept a SARIF document: the coverage-records wire
// shape is a JSON ARRAY of records, so a SARIF object (`{...}`, e.g. coverage
// tunneled through result.properties) is rejected fail-loud (CLM-007). It preserves
// Total==0 faithfully — no synthesized 0% (REQ-004/CLM-013) — and fail-louds on a
// MEASURED record with an empty Metric (REQ-005/CLM-017), an unlabeled measurement
// being a silent-comparison hazard.
func ParsePackCoverage(out []byte) ([]CoverageRecord, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	// Coverage-records is a JSON ARRAY. A leading '{' is a JSON object — a SARIF
	// document (coverage tunneled through SARIF properties) or any non-records
	// payload — and is rejected, never silently read as zero records (CLM-007).
	if trimmed[0] != '[' {
		return nil, fmt.Errorf("coverage-records must be a JSON array of records, not a SARIF/object document: coverage is a DISTINCT typed output, never tunneled through SARIF")
	}

	var records []CoverageRecord
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&records); err != nil {
		return nil, fmt.Errorf("parsing coverage-records JSON: %w", err)
	}

	for i := range records {
		r := records[i]
		// A MEASURED record with an empty Metric is a fail-loud produce/parse error —
		// an unlabeled measurement would let a statement-% be silently compared
		// against a branch-% under one number (REQ-005/CLM-017).
		if r.Measured && r.Metric == "" {
			return nil, fmt.Errorf("coverage record for %q is measured but carries an empty metric: an unlabeled measurement is a silent-comparison hazard", r.Path)
		}
	}
	return records, nil
}
