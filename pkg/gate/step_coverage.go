package gate

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmanson/backstop-core/pkg/check"
)

// SpecVerification holds the verification block fields from a spec.
type SpecVerification struct {
	SpecID            string
	TestCommand       string
	CoverageThreshold int
	// MetricThresholds is the OPTIONAL per-metric declared threshold surface (SQ-2,
	// REQ-003): metric label → integer threshold. A metric absent from the map uses
	// CoverageThreshold as its default; a nil/empty map means scalar-only — the
	// backward-compatible shape (REQ-004). It NEVER ranks or interprets a metric — it
	// is consulted only as a lookup keyed by the pack-declared metric label.
	MetricThresholds      map[string]int
	File                  string
	ImplementationPackage string
}

const defaultCodeScopeCoverageFloor = 90

// StepCoverageThresholdFunc returns a StepFunc that checks the spec-declared
// coverage threshold PER FILE over the canonical []check.CoverageRecord (the
// SINGLE shared type from SPEC-042 — Path/Covered/Total/Measured/Excluded/Metric,
// RAW COUNTS, file granularity). It is the convenience wrapper that delegates to
// the scoped form with a nil scope, now carrying the SourceClassifier (SPEC-043).
func StepCoverageThresholdFunc(coverage []check.CoverageRecord, specs []SpecVerification, classifier SourceClassifier) StepFunc {
	return StepCoverageThresholdScopedFunc(coverage, specs, nil, classifier)
}

// StepCoverageThresholdScopedFunc CONSUMES the canonical per-FILE
// []check.CoverageRecord (raw counts; the single shared type from SPEC-042) and
// computes the verdict itself as Covered/Total >= threshold PER CHANGED/NEW FILE,
// metric-blind. It holds NO `go test` invocation, NO Go-coverage output parsing,
// and NO go.mod / Go-package knowledge — the baked Go coverage analyzer is
// eradicated (REQ-001/CLM-001/CLM-002). Verdict rules (all per FILE, never
// rescued by aggregation):
//
//   - Total==0 (no executable lines) ⇒ N/A: the threshold check is skipped, NEVER
//     a 0%-fail (CLM-026).
//   - A measured file whose Covered/Total ratio is below the applicable threshold
//     produces a blocking coverage_threshold violation, regardless of its
//     directory siblings (CLM-007/CLM-009/CLM-010).
//   - An in-scope changed measurable-source PATH with NO record for the path AT
//     ALL (any metric) that is NOT pack-declared-excluded is a LOUD blocking error
//     under the DISTINCT coverage_unmeasured rule — distinct from below-threshold,
//     and fired even when no positive numeric threshold is in scope, because the
//     `if threshold <= 0 { return pass }` early return is DISMANTLED so the
//     no-record scan runs BEFORE/INDEPENDENT of threshold resolution (REQ-003).
//   - A pack-declared-excluded path is SKIPPED from the threshold check; but a
//     declared exclusion of an IN-SCOPE CHANGED file is LOUDLY SURFACED (path +
//     reason) on the report — never silently dropped (CLM-025).
//   - The pack-declared Metric label is SURFACED on the report and NEVER
//     interpreted, compared, or branched on (CLM-027).
//
// The MEASURABLE-SOURCE set is derived from the SourceClassifier (the merged
// union of pack-declared source/test globs), NOT a baked extension literal
// (REQ-002): coverageMeasurablePath is DELETED. When the classifier declares no
// source globs at all and in-scope changed files exist, the step surfaces a
// DISTINCT non-blocking "classification capability absent" warning instead of a
// silent pass (REQ-004). The no-record predicate is "any record for the path" so
// it composes with SPEC-044's (path, metric) index; per-metric verdicts are
// SPEC-044's.
func StepCoverageThresholdScopedFunc(coverage []check.CoverageRecord, specs []SpecVerification, scope *GateScope, classifier SourceClassifier) StepFunc {
	return func(_ context.Context) StepResult {
		// REQ-004: no declared source globs + in-scope changed files => a DISTINCT
		// visible, non-blocking capability-absent state — never a silent pass. This
		// is checked FIRST, before threshold/path resolution, because with no source
		// globs the measurable set is empty and the step would otherwise pass quietly.
		if !classifier.HasSourceGlobs() && scopeHasChangedFiles(scope) {
			return coverageClassificationCapabilityAbsent()
		}

		// The threshold SELECTION is resolved once (which specs govern the in-scope
		// changed files); the per-metric floor is then derived PER metric inside the
		// loop via coverageThresholdForMetric. The SPEC-043 `if threshold <= 0 { return
		// pass }` early return stays DISMANTLED: the path-level no-record scan below runs
		// independent of any numeric floor, so a declared source glob is honored as the
		// measurement promise even when no positive floor is declared.
		thresholds := coverageThresholdsForScope(specs, scope)

		// REQ-001: the index is keyed by (path, metric) so line AND branch coexist with
		// no last-write-wins overwrite; dupKeys lists (path, metric) pairs the producer
		// double-emitted (a defect surfaced loudly below).
		byPathMetric, dupKeys := indexCoverageByPathMetric(coverage)
		paths := coveragePathsInScope(coverage, scope, classifier)
		if len(paths) == 0 {
			return StepResult{StepName: StepCoverageThreshold, Status: "pass", Violations: []Violation{}, Reason: "no in-scope files to measure for coverage"}
		}

		// The metrics EXPLICITLY declared in any in-scope spec's
		// coverage_metric_thresholds. The metric-granular coverage_metric_missing guard
		// (REQ-005) fires only for THESE — a scalar-only spec declares none, so it never
		// fires (CLM-017/CLM-019).
		declaredMetrics := declaredCoverageMetrics(thresholds)

		var violations []Violation

		// A duplicate (path, metric) is a producer defect (two measurements of one file
		// under one metric). Surface it LOUDLY rather than silently keeping one survivor —
		// the not-last-wins signal from indexCoverageByPathMetric (REQ-001/CLM-003).
		for _, key := range dupKeys {
			dupPath, dupMetric := splitCoverageDupKey(key)
			violations = append(violations, Violation{
				Rule:     "coverage_metric_collision",
				Message:  fmt.Sprintf("duplicate coverage measurement for file %s under metric %q — a producer emitted two records for the same (path, metric); refusing to silently keep one", dupPath, dupMetric),
				File:     dupPath,
				Severity: "error",
			})
		}

		for _, path := range paths {
			metricRecords, hasRecords := resolveCoverageRecordsForPath(byPathMetric, path)
			inScopeChanged := coveragePathInDiffScope(path, scope)

			if !hasRecords {
				// SPEC-043's PATH-LEVEL guard: an in-scope changed measurable-source path
				// with NO record AT ALL (any metric) that is not pack-declared excluded is
				// a LOUD blocking error under the DISTINCT coverage_unmeasured rule — never
				// conflated with below-threshold. It is ordered BEFORE the per-metric loop
				// so a zero-record path yields THIS guard ALONE and never the metric-
				// granular coverage_metric_missing guard (Sharp Edge 7 / CLM-020).
				violations = append(violations, Violation{
					Rule:     "coverage_unmeasured",
					Message:  fmt.Sprintf("no coverage measurement for in-scope changed measurable-source file %s (any metric) and it is not pack-declared excluded — refusing to pass with nothing measured", path),
					File:     path,
					Severity: "error",
				})
				continue
			}

			// METRIC-GRANULAR missing guard (REQ-005), DISTINCT from the path-level
			// zero-record guard above: the path HAS records but is missing a metric
			// EXPLICITLY declared in scope. Reached ONLY for has-records paths (the
			// zero-record path took the continue above), and fired ONLY for in-scope
			// CHANGED paths — an unchanged/all-mode path stays quiet (CLM-021), parallel
			// to the exclusion-loudness scoping. The two guards never double-report or
			// shadow: zero records ⇒ coverage_unmeasured ALONE; has-records-missing-metric
			// ⇒ coverage_metric_missing ALONE (CLM-018/CLM-020, Sharp Edge 7).
			if inScopeChanged {
				for _, metric := range declaredMetrics {
					if _, measured := metricRecords[metric]; !measured {
						violations = append(violations, Violation{
							Rule:     "coverage_metric_missing",
							Message:  fmt.Sprintf("in-scope changed file %s has coverage records but is missing the explicitly-declared %q metric — refusing to pass with a declared metric silently unmeasured", path, metric),
							File:     path,
							Severity: "error",
						})
					}
				}
			}

			// REQ-002: iterate EVERY metric record for the path and threshold each
			// INDEPENDENTLY — no aggregation, no sibling-metric rescue. A single file may
			// thus emit several verdicts (e.g. line passes, branch fails). Sorted for a
			// deterministic report order.
			for _, metric := range sortedCoverageMetrics(metricRecords) {
				record := metricRecords[metric]

				if record.Excluded {
					// A declared exclusion of an in-scope CHANGED file is LOUDLY SURFACED at
					// (path, metric) granularity — a NON-blocking warning; the threshold
					// check is skipped but the suppression is visible (CLM-025). An
					// UNCHANGED-file exclusion may stay quiet.
					if inScopeChanged {
						violations = append(violations, Violation{
							Rule:     "coverage_exclusion",
							Message:  fmt.Sprintf("coverage requirement for changed file %s (metric %s) is suppressed by a pack-declared exclusion%s", path, coverageMetricLabel(record), coverageExclusionReason(record)),
							File:     path,
							Severity: "warning",
						})
					}
					continue
				}

				// Total==0 ⇒ N/A for THIS metric: skipped, NEVER a 0%-fail (CLM-009),
				// while OTHER metrics on the same file are still thresholded.
				if record.Total == 0 {
					continue
				}

				// Resolve the metric's governing threshold (per-metric override or scalar
				// default, strictest in scope). A non-positive threshold means none is
				// declared for this metric in scope ⇒ SKIP it (CLM-015).
				threshold := coverageThresholdForMetric(thresholds, metric)
				if threshold <= 0 {
					continue
				}

				// Verdict computed metric-BLIND from RAW COUNTS (CLM-010): the Metric label
				// is consulted only as the threshold-lookup key and a report label, never
				// ranked or interpreted.
				if coverageBelowThreshold(record.Covered, record.Total, threshold) {
					violations = append(violations, Violation{
						Rule:     "coverage_threshold",
						Message:  fmt.Sprintf("file %s coverage %d/%d (%s) below threshold %d%%", path, record.Covered, record.Total, coverageMetricLabel(record), threshold),
						File:     path,
						Severity: "error",
					})
				}
			}
		}

		status := "pass"
		for _, v := range violations {
			if v.Severity == "error" {
				status = "fail"
			}
		}
		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{StepName: StepCoverageThreshold, Status: status, Violations: violations}
	}
}

// coverageBelowThreshold computes the per-file verdict from RAW COUNTS as
// Covered/Total >= threshold% WITHOUT floating-point: covered*100 >= total*threshold.
// This keeps the gate metric-blind (it never consumes a pre-computed percent) and
// avoids rounding drift at the boundary (CLM-026).
func coverageBelowThreshold(covered, total, threshold int) bool {
	return covered*100 < total*threshold
}

// coverageMetricLabel surfaces the pack-declared Metric label for the report,
// NEVER interpreting it (CLM-027). An empty Metric renders as "unlabeled".
func coverageMetricLabel(r check.CoverageRecord) string {
	if strings.TrimSpace(r.Metric) == "" {
		return "unlabeled"
	}
	return r.Metric
}

// coverageExclusionReason renders the declared exclusion detail for the loud
// surfacing of a changed-file exclusion (CLM-025). The canonical record carries
// no free-form reason field, so the surfaced detail names the declaring source
// (the Metric label when present) so the suppression is attributable.
func coverageExclusionReason(r check.CoverageRecord) string {
	if metric := strings.TrimSpace(r.Metric); metric != "" {
		return " (declared metric: " + metric + ")"
	}
	return " (pack-declared exclusion)"
}

// coverageDupPairSep joins a (path, metric) pair into the duplicate-key strings
// indexCoverageByPathMetric returns. A tab never appears in a file path or a
// pack-declared metric label, so the verdict loop can split it back apart to
// surface the path and metric of a duplicate-measurement producer defect.
const coverageDupPairSep = "\t"

// indexCoverageByPathMetric builds a (path, metric)-keyed index over the canonical
// records (REQ-001): the outer key is the normalizeScopePath project-relative path
// (so it matches scope paths regardless of the producer's path style, exactly as the
// old path-keyed index did), the inner key is the Metric label, and the value is the
// canonical record. A file carrying line AND branch retains BOTH — neither overwrites
// the other. The second return value lists the (path, metric) keys seen MORE THAN
// ONCE (joined by coverageDupPairSep) so the step can surface them as loud coverage
// errors rather than silently collapsing them last-wins — a duplicate (path, metric)
// is a duplicate-measurement producer defect. REPLACES indexCoverageByPath.
func indexCoverageByPathMetric(coverage []check.CoverageRecord) (map[string]map[string]check.CoverageRecord, []string) {
	byPathMetric := make(map[string]map[string]check.CoverageRecord, len(coverage))
	var dupKeys []string
	for _, r := range coverage {
		path := normalizeScopePath("", r.Path)
		inner, ok := byPathMetric[path]
		if !ok {
			inner = make(map[string]check.CoverageRecord)
			byPathMetric[path] = inner
		}
		if _, exists := inner[r.Metric]; exists {
			// Loud, not last-wins: keep the first record and report the duplicate so
			// the step blocks rather than silently dropping half the measurements.
			dupKeys = append(dupKeys, path+coverageDupPairSep+r.Metric)
			continue
		}
		inner[r.Metric] = r
	}
	return byPathMetric, dupKeys
}

// resolveCoverageRecordsForPath returns ALL metric records for a repo-relative scope
// path (REQ-001). It first tries an exact match, then falls back to the UNIQUE
// record-path whose normalized path ENDS WITH "/"+path — the same separator-anchored,
// unique-match-required reconciliation resolveCoverageRecord used for module/namespace-
// qualified producer paths, now returning the whole per-metric map. An ambiguous suffix
// (two qualified paths ending the same way) is treated as no-match so the loud guards
// fire rather than silently picking one. REPLACES resolveCoverageRecord.
func resolveCoverageRecordsForPath(byPathMetric map[string]map[string]check.CoverageRecord, path string) (map[string]check.CoverageRecord, bool) {
	if m, ok := byPathMetric[path]; ok {
		return m, true
	}
	suffix := "/" + path
	var match map[string]check.CoverageRecord
	found := 0
	for recPath, m := range byPathMetric {
		if strings.HasSuffix(recPath, suffix) {
			match = m
			found++
		}
	}
	if found == 1 {
		return match, true
	}
	return nil, false
}

// splitCoverageDupKey reverses the coverageDupPairSep join, recovering the (path,
// metric) pair from a duplicate-key string so the step can surface the offending
// file and metric on the report.
func splitCoverageDupKey(key string) (path, metric string) {
	if i := strings.Index(key, coverageDupPairSep); i >= 0 {
		return key[:i], key[i+len(coverageDupPairSep):]
	}
	return key, ""
}

// declaredCoverageMetrics returns the sorted, de-duplicated set of metric labels
// EXPLICITLY declared across the selected specs' MetricThresholds maps (REQ-005). The
// metric-granular coverage_metric_missing guard fires only for these — a scalar-only
// spec contributes none, so the guard never fires for it (CLM-017/CLM-019).
func declaredCoverageMetrics(sel coverageThresholdSelection) []string {
	set := map[string]struct{}{}
	for _, spec := range sel.Specs {
		for metric := range spec.MetricThresholds {
			set[metric] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// sortedCoverageMetrics returns the metric labels of a per-metric record map in
// sorted order, so the per-(path, metric) verdict loop emits violations in a stable,
// reproducible order regardless of Go's randomized map iteration.
func sortedCoverageMetrics(metricRecords map[string]check.CoverageRecord) []string {
	out := make([]string, 0, len(metricRecords))
	for m := range metricRecords {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// coveragePathsInScope returns the sorted set of FILE paths to evaluate. In a
// diff/file scope it is exactly the in-scope changed files that look like source
// (so the loud "no record for a changed file" check fires per changed path). In
// all-mode (or nil scope) it is every record's path (the full project sweep flags
// every below-threshold file, never a whole-repo aggregate).
func coveragePathsInScope(coverage []check.CoverageRecord, scope *GateScope, classifier SourceClassifier) []string {
	set := map[string]struct{}{}
	if scope == nil || scope.Mode == GateScopeModeAll {
		for _, r := range coverage {
			set[normalizeScopePath("", r.Path)] = struct{}{}
		}
	} else {
		for _, f := range scope.Files {
			clean := normalizeScopePath(scope.ProjectRoot, f)
			// The MEASURABLE-SOURCE decision comes from the pack-declared globs
			// (SourceClassifier), never a baked extension literal (REQ-002): a path
			// is in-scope-to-measure iff it matches a declared source glob and no
			// declared test glob (test-wins-on-overlap).
			if !classifier.IsMeasurableSource(clean) {
				continue
			}
			set[clean] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// scopeHasChangedFiles reports whether the scope is a diff/file scope carrying at
// least one in-scope changed file. It is the precondition for the REQ-004
// classification-capability-absent state (no source globs declared yet changed
// files exist), distinguishing it from an all-mode sweep.
func scopeHasChangedFiles(scope *GateScope) bool {
	return scope != nil && scope.Mode != GateScopeModeAll && len(scope.Files) > 0
}

// coverageClassificationCapabilityAbsent builds the DISTINCT, VISIBLE, NON-blocking
// "classification capability absent" StepResult for REQ-004: when no declared
// toolchain pack carries a classification.source list the step cannot classify
// which changed files are measurable source, so it surfaces a warning rather than
// an unqualified pass. It reuses the EXISTING capability-absent convention (the
// warning-status `<dim>_capability_absent` shape PolarityStepResult emits) as a
// coverage-dimension advisory: Severity warning, ConfigErr false, exit 0.
func coverageClassificationCapabilityAbsent() StepResult {
	msg := fmt.Sprintf(
		"coverage classification capability absent: no pack-declared %s globs are in effect, so the gate cannot determine which in-scope changed files are measurable source — install/declare a toolchain pack carrying a classification.source list. This advisory is non-blocking (exit 0).",
		DimensionCoverage,
	)
	return StepResult{
		StepName:   StepCoverageThreshold,
		Status:     "warning",
		ConfigErr:  false,
		Violations: []Violation{{Rule: string(DimensionCoverage) + "_capability_absent", Message: msg, Severity: "warning"}},
	}
}

// coveragePathInDiffScope reports whether path is an in-scope CHANGED file under a
// diff/file scope (used to decide whether a declared exclusion must be loudly
// surfaced). In all-mode every measured path is treated as not-changed for the
// purpose of exclusion loudness (an all-mode sweep may stay quiet on exclusions).
func coveragePathInDiffScope(path string, scope *GateScope) bool {
	if scope == nil || scope.Mode == GateScopeModeAll {
		return false
	}
	return scope.Contains(path)
}

// coverageThresholdForMetric resolves the governing threshold for a SINGLE metric
// from the selected specs (REQ-003, SQ-2). It GENERALIZES coverageFloorForScope from
// a single floor to a per-metric floor: for each selected spec the applicable
// threshold for the metric is MetricThresholds[metric] if explicitly declared, else
// CoverageThreshold (the scalar default); the governing value is the MAX across the
// selected specs (the strictest in-scope spec governs, now per metric). It returns 0
// when no threshold is declared for the metric in scope — the caller then SKIPS the
// metric ("no threshold declared in scope ⇒ pass", at metric granularity). An
// override declared for metric M never alters metric N: only MetricThresholds[metric]
// is consulted, and the metric label is used solely as a map key (never ranked or
// interpreted).
func coverageThresholdForMetric(sel coverageThresholdSelection, metric string) int {
	if sel.CollapsedCodeScope {
		// In a collapsed code scope there are no file-specific specs to carry
		// per-metric overrides; the single derived floor is the scalar default for
		// every metric (backward-compatible with the per-file floor).
		return sel.MaxThreshold
	}
	maxThreshold := 0
	for _, spec := range sel.Specs {
		applicable := spec.CoverageThreshold
		if override, ok := spec.MetricThresholds[metric]; ok {
			applicable = override
		}
		if applicable > maxThreshold {
			maxThreshold = applicable
		}
	}
	return maxThreshold
}

type coverageThresholdSelection struct {
	Specs              []SpecVerification
	CollapsedCodeScope bool
	MaxThreshold       int
}

// coverageThresholdsForScope is RETAINED, language-agnostic threshold-derivation
// logic (re-keyed from the deleted per-package model to per-FILE consumption): it
// selects the specs whose declared threshold governs the in-scope changed files,
// or collapses to the code-scope floor when no spec is file-specific.
func coverageThresholdsForScope(specs []SpecVerification, scope *GateScope) coverageThresholdSelection {
	if scope == nil || scope.Mode == GateScopeModeAll {
		return coverageThresholdSelection{Specs: specs}
	}
	selected := []SpecVerification{}
	for _, spec := range specs {
		if spec.File != "" && scope.Contains(spec.File) {
			selected = append(selected, spec)
		}
	}
	if len(selected) > 0 {
		return coverageThresholdSelection{Specs: selected}
	}
	maxThreshold := 0
	hasSpecific := false
	for _, spec := range specs {
		if !coverageSpecRelevantToCodeScope(spec, scope, false) {
			continue
		}
		hasSpecific = true
		if spec.CoverageThreshold > maxThreshold {
			maxThreshold = spec.CoverageThreshold
		}
	}
	if !hasSpecific {
		for _, spec := range specs {
			if !coverageSpecRelevantToCodeScope(spec, scope, true) {
				continue
			}
			if spec.CoverageThreshold > maxThreshold {
				maxThreshold = spec.CoverageThreshold
			}
		}
	}
	if maxThreshold == 0 {
		maxThreshold = defaultCodeScopeCoverageFloor
	}
	return coverageThresholdSelection{CollapsedCodeScope: true, MaxThreshold: maxThreshold}
}

func coverageSpecRelevantToCodeScope(spec SpecVerification, scope *GateScope, includeRootCommand bool) bool {
	if scope == nil || scope.Empty() {
		return false
	}
	for _, file := range scope.Files {
		if coverageSpecRelevantToFile(spec, normalizeScopePath("", file), includeRootCommand) {
			return true
		}
	}
	return false
}

// coverageSpecRelevantToFile decides whether a spec's coverage threshold governs a
// changed file by LANGUAGE-NEUTRAL directory matching (SPEC-045 REQ-004). The baked
// Go source-extension suffix gate and the recursive-Go-package test-command-substring
// convention are GONE: relevance is decided by packagePathMatches(dir,
// spec.ImplementationPackage), plus a root fallback applied ONLY when
// includeRootCommand AND the spec declares no specific implementation package — so a
// package-scoped spec is matched by its package and is NEVER over-broadened
// project-wide (CLM-028). A spec carrying no implementation package falls back to
// project-wide relevance under includeRootCommand (CLM-027). Which changed files
// actually need a coverage record is decided upstream by the SourceClassifier's
// measurable-source guard, so dropping the `.go` gate does not create vacuous
// coverage requirements for non-source changes.
func coverageSpecRelevantToFile(spec SpecVerification, file string, includeRootCommand bool) bool {
	dir := filepath.Dir(file)
	if dir == "." {
		dir = ""
	}
	if spec.ImplementationPackage != "" {
		return packagePathMatches(dir, spec.ImplementationPackage)
	}
	// No specific implementation package: the spec is project-wide, relevant to any
	// changed file under the root fallback (no Go-package glob literal involved).
	return includeRootCommand
}

func packagePathMatches(changedDir string, specPackage string) bool {
	trimmed := strings.TrimPrefix(strings.Trim(specPackage, "/"), "./")
	if changedDir == "" || trimmed == "" {
		return changedDir == trimmed
	}
	return changedDir == trimmed || strings.HasPrefix(changedDir, trimmed+"/") || strings.HasPrefix(trimmed, changedDir+"/")
}
