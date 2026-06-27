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
	SpecID                string
	TestCommand           string
	CoverageThreshold     int
	File                  string
	ImplementationPackage string
}

const defaultCodeScopeCoverageFloor = 90

// StepCoverageThresholdFunc returns a StepFunc that checks the spec-declared
// coverage threshold PER FILE over the canonical []check.CoverageRecord (the
// SINGLE shared type from SPEC-042 — Path/Covered/Total/Measured/Excluded/Metric,
// RAW COUNTS, file granularity). It is the convenience wrapper that delegates to
// the scoped form with a nil scope.
func StepCoverageThresholdFunc(coverage []check.CoverageRecord, specs []SpecVerification) StepFunc {
	return StepCoverageThresholdScopedFunc(coverage, specs, nil)
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
//   - An in-scope changed PATH with NO record that is NOT pack-declared-excluded is
//     a LOUD blocking error (severity error), never a silent pass (CLM-008).
//   - A pack-declared-excluded path is SKIPPED from the threshold check; but a
//     declared exclusion of an IN-SCOPE CHANGED file is LOUDLY SURFACED (path +
//     reason) on the report — never silently dropped (CLM-025).
//   - The pack-declared Metric label is SURFACED on the report and NEVER
//     interpreted, compared, or branched on (CLM-027).
func StepCoverageThresholdScopedFunc(coverage []check.CoverageRecord, specs []SpecVerification, scope *GateScope) StepFunc {
	return func(_ context.Context) StepResult {
		thresholds := coverageThresholdsForScope(specs, scope)
		threshold := coverageFloorForScope(thresholds)
		if threshold <= 0 {
			return StepResult{StepName: StepCoverageThreshold, Status: "pass", Violations: []Violation{}, Reason: "no coverage threshold declared in scope"}
		}

		byPath := indexCoverageByPath(coverage)
		paths := coveragePathsInScope(coverage, scope)
		if len(paths) == 0 {
			return StepResult{StepName: StepCoverageThreshold, Status: "pass", Violations: []Violation{}, Reason: "no in-scope files to measure for coverage"}
		}

		var violations []Violation
		for _, path := range paths {
			record, hasRecord := resolveCoverageRecord(byPath, path)
			inScopeChanged := coveragePathInDiffScope(path, scope)

			if !hasRecord {
				// An in-scope changed PATH with NO record is a LOUD blocking error —
				// never a silent pass. Only a pack-DECLARED exclusion (carried as a
				// record with Excluded==true) silences the requirement (CLM-008).
				violations = append(violations, Violation{
					Rule:     "coverage_threshold",
					Message:  fmt.Sprintf("no coverage measurement for in-scope changed file %s and it is not pack-declared excluded — refusing to pass with nothing measured", path),
					File:     path,
					Severity: "error",
				})
				continue
			}

			if record.Excluded {
				// A declared exclusion of an in-scope CHANGED file is LOUDLY SURFACED
				// (path + reason) — closing the "declare Excluded:true to suppress a
				// changed file invisibly" vacuous-green vector. An UNCHANGED-file
				// exclusion may stay quiet (CLM-025). The surfaced exclusion is a
				// NON-blocking warning (Severity warning): the threshold check is
				// skipped, but the suppression is visible.
				if inScopeChanged {
					violations = append(violations, Violation{
						Rule:     "coverage_exclusion",
						Message:  fmt.Sprintf("coverage requirement for changed file %s is suppressed by a pack-declared exclusion%s", path, coverageExclusionReason(record)),
						File:     path,
						Severity: "warning",
					})
				}
				continue
			}

			// Total==0 (no executable lines) ⇒ N/A: skip the threshold check, NEVER
			// a 0%-fail (CLM-026). Metric-blind: the gate computes the ratio from raw
			// counts and never interprets Metric.
			if record.Total == 0 {
				continue
			}

			if coverageBelowThreshold(record.Covered, record.Total, threshold) {
				violations = append(violations, Violation{
					Rule:     "coverage_threshold",
					Message:  fmt.Sprintf("file %s coverage %d/%d (%s) below threshold %d%%", path, record.Covered, record.Total, coverageMetricLabel(record), threshold),
					File:     path,
					Severity: "error",
				})
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

// indexCoverageByPath builds a path-keyed lookup over the canonical records,
// normalizing each path to the gate's project-relative slash form so it matches
// scope paths regardless of the producer's path style.
func indexCoverageByPath(coverage []check.CoverageRecord) map[string]check.CoverageRecord {
	byPath := make(map[string]check.CoverageRecord, len(coverage))
	for _, r := range coverage {
		byPath[normalizeScopePath("", r.Path)] = r
	}
	return byPath
}

// resolveCoverageRecord finds the record for a repo-relative scope path. It first
// tries an exact match, then falls back to a record whose (normalized) path ENDS
// WITH "/"+path. The fallback reconciles a producer that emits module/namespace-
// qualified paths (e.g. "github.com/org/repo/pkg/x/f.go") against the gate's
// repo-relative scope ("pkg/x/f.go"), WITHOUT the language-neutral consumer
// learning any module/tool semantics. The suffix is anchored on a path separator
// so "terminal.go" never matches "internal.go". A unique match is required: an
// ambiguous suffix (two records ending the same way) is treated as no-match so
// the loud not-measured check fires rather than silently picking one.
func resolveCoverageRecord(byPath map[string]check.CoverageRecord, path string) (check.CoverageRecord, bool) {
	if r, ok := byPath[path]; ok {
		return r, true
	}
	suffix := "/" + path
	var match check.CoverageRecord
	found := 0
	for recPath, r := range byPath {
		if strings.HasSuffix(recPath, suffix) {
			match = r
			found++
		}
	}
	if found == 1 {
		return match, true
	}
	return check.CoverageRecord{}, false
}

// coveragePathsInScope returns the sorted set of FILE paths to evaluate. In a
// diff/file scope it is exactly the in-scope changed files that look like source
// (so the loud "no record for a changed file" check fires per changed path). In
// all-mode (or nil scope) it is every record's path (the full project sweep flags
// every below-threshold file, never a whole-repo aggregate).
func coveragePathsInScope(coverage []check.CoverageRecord, scope *GateScope) []string {
	set := map[string]struct{}{}
	if scope == nil || scope.Mode == GateScopeModeAll {
		for _, r := range coverage {
			set[normalizeScopePath("", r.Path)] = struct{}{}
		}
	} else {
		for _, f := range scope.Files {
			clean := normalizeScopePath(scope.ProjectRoot, f)
			if !coverageMeasurablePath(clean) {
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

// coverageMeasurablePath reports whether a changed-file path is one coverage can
// be expected for: a source file, excluding spec docs, test files, and testdata
// fixtures (which carry no coverage requirement). This is language-agnostic by
// shape — it keys on path conventions, not a Go-package model.
func coverageMeasurablePath(path string) bool {
	if path == "" || strings.HasSuffix(path, ".spec.md") {
		return false
	}
	if strings.HasSuffix(path, "_test.go") {
		return false
	}
	if !strings.HasSuffix(path, ".go") {
		// Non-Go source files are measured only when the producer emits a record
		// for them; they are not synthesized as required paths here (the producer,
		// not the gate, decides the measured set for other languages).
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "testdata" {
			return false
		}
	}
	return true
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

// coverageFloorForScope derives the single applicable per-file threshold from the
// selected specs: the max declared threshold (so the strictest in-scope spec
// governs), falling back to the code-scope floor when the scope collapses to code
// with no spec-specific threshold.
func coverageFloorForScope(sel coverageThresholdSelection) int {
	if sel.CollapsedCodeScope {
		return sel.MaxThreshold
	}
	maxThreshold := 0
	for _, spec := range sel.Specs {
		if spec.CoverageThreshold > maxThreshold {
			maxThreshold = spec.CoverageThreshold
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

func coverageSpecRelevantToFile(spec SpecVerification, file string, includeRootCommand bool) bool {
	if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_testdata.go") {
		return false
	}
	dir := filepath.Dir(file)
	if dir == "." {
		dir = ""
	}
	if spec.ImplementationPackage != "" && packagePathMatches(dir, spec.ImplementationPackage) {
		return true
	}
	if spec.TestCommand == "" {
		return false
	}
	return includeRootCommand && strings.Contains(spec.TestCommand, "./...") || strings.Contains(spec.TestCommand, "./"+dir)
}

func packagePathMatches(changedDir string, specPackage string) bool {
	trimmed := strings.TrimPrefix(strings.Trim(specPackage, "/"), "./")
	if changedDir == "" || trimmed == "" {
		return changedDir == trimmed
	}
	return changedDir == trimmed || strings.HasPrefix(changedDir, trimmed+"/") || strings.HasPrefix(trimmed, changedDir+"/")
}
