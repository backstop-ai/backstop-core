package gate

import "testing"

// TestThreshold_PerMetricOverrideAppliedScalarDefaultForUnlisted (CLM-011): a spec
// with coverage_threshold 90 and coverage_metric_thresholds {branch: 70} resolves
// branch ⇒ 70 and the unlisted line ⇒ 90 (the scalar default).
func TestThreshold_PerMetricOverrideAppliedScalarDefaultForUnlisted(t *testing.T) {
	sel := coverageThresholdSelection{
		Specs: []SpecVerification{
			{SpecID: "SPEC-A", CoverageThreshold: 90, MetricThresholds: map[string]int{"branch": 70}},
		},
	}
	if got := coverageThresholdForMetric(sel, "branch"); got != 70 {
		t.Errorf("branch must use its per-metric override 70, got %d", got)
	}
	if got := coverageThresholdForMetric(sel, "line"); got != 90 {
		t.Errorf("unlisted line must fall back to the scalar default 90, got %d", got)
	}
}

// TestThreshold_UnlistedMetricUsesScalarDefault (CLM-012): a metric with no override
// falls back to the scalar coverage_threshold, identical to the metric-blind scalar
// path today.
func TestThreshold_UnlistedMetricUsesScalarDefault(t *testing.T) {
	sel := coverageThresholdSelection{
		Specs: []SpecVerification{
			{SpecID: "SPEC-A", CoverageThreshold: 85},
		},
	}
	if got := coverageThresholdForMetric(sel, "statement"); got != 85 {
		t.Errorf("a metric with no override must use the scalar default 85, got %d", got)
	}
	if got := coverageThresholdForMetric(sel, "branch"); got != 85 {
		t.Errorf("any unlisted metric resolves to the scalar default 85, got %d", got)
	}
}

// TestThreshold_StrictestSpecGovernsPerMetric (CLM-013): two in-scope specs
// declaring branch overrides 70 and 80 ⇒ the governing branch threshold is the MAX
// (80); the per-metric selection takes the max of each spec's applicable
// override-or-default PER METRIC.
func TestThreshold_StrictestSpecGovernsPerMetric(t *testing.T) {
	sel := coverageThresholdSelection{
		Specs: []SpecVerification{
			{SpecID: "SPEC-A", CoverageThreshold: 90, MetricThresholds: map[string]int{"branch": 70}},
			{SpecID: "SPEC-B", CoverageThreshold: 90, MetricThresholds: map[string]int{"branch": 80}},
		},
	}
	if got := coverageThresholdForMetric(sel, "branch"); got != 80 {
		t.Errorf("the strictest in-scope branch override (80) must govern, got %d", got)
	}
	if got := coverageThresholdForMetric(sel, "line"); got != 90 {
		t.Errorf("line (no override) must resolve to the MAX scalar default 90, got %d", got)
	}
}

// TestThreshold_OverrideForOneMetricDoesNotAffectAnother (CLM-014): declaring branch
// 70 leaves the line threshold at the scalar default — overrides are isolated per
// metric.
func TestThreshold_OverrideForOneMetricDoesNotAffectAnother(t *testing.T) {
	sel := coverageThresholdSelection{
		Specs: []SpecVerification{
			{SpecID: "SPEC-A", CoverageThreshold: 95, MetricThresholds: map[string]int{"branch": 70}},
		},
	}
	branch := coverageThresholdForMetric(sel, "branch")
	line := coverageThresholdForMetric(sel, "line")
	if branch != 70 {
		t.Errorf("branch must be its override 70, got %d", branch)
	}
	if line != 95 {
		t.Errorf("a branch override must NOT alter line — line stays at the scalar default 95, got %d", line)
	}
}

// TestThreshold_MetricWithNoDeclaredThresholdSkipped (CLM-015): a metric whose
// resolved threshold is <= 0 (no scalar and no override in scope) resolves to 0 so
// the caller SKIPS it rather than failing — "no threshold declared in scope ⇒ pass"
// at metric granularity.
func TestThreshold_MetricWithNoDeclaredThresholdSkipped(t *testing.T) {
	sel := coverageThresholdSelection{
		Specs: []SpecVerification{
			{SpecID: "SPEC-A", CoverageThreshold: 0},
		},
	}
	if got := coverageThresholdForMetric(sel, "branch"); got != 0 {
		t.Errorf("a metric with no scalar and no override must resolve to 0 (skipped), got %d", got)
	}
}
