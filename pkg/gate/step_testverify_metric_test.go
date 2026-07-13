package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMetricSpecFixture writes a spec file whose verification block is supplied
// verbatim, so the per-metric extraction tests can exercise coverage_threshold and
// coverage_metric_thresholds in combinations the shared writeSpecFixture helper does
// not emit (scalar+map, map-only, scalar-only).
func writeMetricSpecFixture(t *testing.T, dir, filename, verificationBlock string) {
	t.Helper()
	content := `---
title: "Metric Spec"
number: TEST-METRIC
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Metric spec
  package: pkg/gate

` + verificationBlock + `

requirements:
  - id: REQ-001
    text: Test requirement
    supports: cli:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Test claim
    tests:
      - TestSomething
---

# Metric Spec
`
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("writing metric spec fixture: %v", err)
	}
}

// specVerificationFor returns the extracted SpecVerification from a single-spec dir.
func specVerificationFor(t *testing.T, dir string) (SpecVerification, bool) {
	t.Helper()
	specs, err := ExtractSpecVerifications(dir)
	if err != nil {
		t.Fatalf("ExtractSpecVerifications: %v", err)
	}
	if len(specs) == 0 {
		return SpecVerification{}, false
	}
	if len(specs) != 1 {
		t.Fatalf("expected exactly one extracted spec, got %d", len(specs))
	}
	return specs[0], true
}

// TestExtractSpecVerifications_PopulatesMetricThresholds (CLM-011): a spec whose
// verification block declares coverage_threshold 90 and coverage_metric_thresholds
// {branch: 70} extracts to a SpecVerification with CoverageThreshold 90 and
// MetricThresholds {"branch": 70}.
func TestExtractSpecVerifications_PopulatesMetricThresholds(t *testing.T) {
	dir := t.TempDir()
	writeMetricSpecFixture(t, dir, "scalar-and-metric.spec.md", `verification:
  level: unit
  test_command: go test ./pkg/gate/... -race
  coverage_threshold: 90
  coverage_metric_thresholds:
    branch: 70`)

	spec, ok := specVerificationFor(t, dir)
	if !ok {
		t.Fatalf("expected the spec to be extracted")
	}
	if spec.CoverageThreshold != 90 {
		t.Errorf("expected scalar CoverageThreshold 90, got %d", spec.CoverageThreshold)
	}
	if spec.MetricThresholds == nil {
		t.Fatalf("expected MetricThresholds to be populated, got nil")
	}
	if got := spec.MetricThresholds["branch"]; got != 70 {
		t.Errorf("expected MetricThresholds[branch] == 70, got %d", got)
	}
}

// TestExtractSpecVerifications_PerMetricOnlyNoScalarStillExtracted (CLM-015): a spec
// declaring coverage_metric_thresholds {branch: 70} WITHOUT a scalar
// coverage_threshold is STILL extracted (the loosened gate) — proving the
// per-metric-only declaration path is supported.
func TestExtractSpecVerifications_PerMetricOnlyNoScalarStillExtracted(t *testing.T) {
	dir := t.TempDir()
	writeMetricSpecFixture(t, dir, "metric-only.spec.md", `verification:
  level: unit
  test_command: go test ./pkg/gate/... -race
  coverage_metric_thresholds:
    branch: 70`)

	spec, ok := specVerificationFor(t, dir)
	if !ok {
		t.Fatalf("a spec declaring only per-metric thresholds (no scalar) must STILL be extracted (loosened gate)")
	}
	if spec.CoverageThreshold != 0 {
		t.Errorf("expected no scalar threshold (0) for a metric-only spec, got %d", spec.CoverageThreshold)
	}
	if got := spec.MetricThresholds["branch"]; got != 70 {
		t.Errorf("expected MetricThresholds[branch] == 70, got %d", got)
	}
}

// TestExtractSpecVerifications_NilMetricThresholdsPreservesScalarOnly (CLM-012): a
// spec with only the scalar coverage_threshold (no coverage_metric_thresholds)
// extracts with a nil/empty MetricThresholds — the backward-compatible scalar-only
// shape.
func TestExtractSpecVerifications_NilMetricThresholdsPreservesScalarOnly(t *testing.T) {
	dir := t.TempDir()
	writeMetricSpecFixture(t, dir, "scalar-only.spec.md", `verification:
  level: unit
  test_command: go test ./pkg/gate/... -race
  coverage_threshold: 80`)

	spec, ok := specVerificationFor(t, dir)
	if !ok {
		t.Fatalf("a scalar-only spec must be extracted")
	}
	if spec.CoverageThreshold != 80 {
		t.Errorf("expected scalar CoverageThreshold 80, got %d", spec.CoverageThreshold)
	}
	if len(spec.MetricThresholds) != 0 {
		t.Errorf("a scalar-only spec must extract with nil/empty MetricThresholds, got %#v", spec.MetricThresholds)
	}
}
