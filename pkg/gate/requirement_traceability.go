package gate

import (
	"fmt"
	"strings"
)

// TraceRef is a gate-side supports edge from a spec/issue requirement to a bundle REQ.
type TraceRef struct {
	BundleName string
	ReqID      string
	PinVersion string
	Pinned     bool
	CitingPath string
}

// SupportsCoverage reports whether a citing artifact can close bundle coverage.
// Only implemented specs cover bundle REQs; issues are lineage-only, and retired specs
// such as `replaced` never flow coverage forward.
func SupportsCoverage(kind ArtifactKind, class StatusClass) bool {
	return kind == KindSpec && class == ClassSuccessTerminal
}

// StalePinVerdict reports how a ref pin relates to the bundle REQ's current version.
// Patch movement within the same major.minor is compatible; major/minor movement means
// the ref satisfied an older meaning and must be reworked or replaced. The bump follows
// the contract vocabulary none/patch/minor/major (identical pins report "none"); an
// unparseable version reports "unknown". Callers key off `stale`; `bump` is descriptive.
func StalePinVerdict(pin string, current string) (bump string, stale bool) {
	if pin == current {
		return "none", false
	}
	pmj, pmn, okPinned := semverMajorMinor(pin)
	cmj, cmn, okCurrent := semverMajorMinor(current)
	if !okPinned || !okCurrent {
		return "unknown", false
	}
	if pmj != cmj {
		return "major", true
	}
	if pmn != cmn {
		return "minor", true
	}
	return "patch", false
}

// ClassifyRequirementTraceability computes the BUNDLE-014 corpus-state verdict. It assumes
// validate already handled ref-resolution defects; unresolved refs are ignored here rather
// than duplicated under a second step.
func ClassifyRequirementTraceability(records []ArtifactStatusRecord, refs []TraceRef) StepResult {
	var violations []Violation
	bundles := map[string]ArtifactStatusRecord{}
	citers := map[string]ArtifactStatusRecord{}
	specs := map[string]ArtifactStatusRecord{}
	covered := map[string]map[string]bool{}
	staleSpecs := map[string]bool{}
	var staleRefs []staleTraceRef

	for _, rec := range records {
		switch rec.Kind {
		case KindBundle:
			if rec.BundleName != "" {
				bundles[rec.BundleName] = rec
				covered[rec.BundleName] = map[string]bool{}
			}
		case KindSpec:
			citers[NormalizePath("", rec.Path)] = rec
			specs[rec.ID] = rec
		case KindIssue:
			citers[NormalizePath("", rec.Path)] = rec
		}
	}

	for _, ref := range refs {
		bundle, hasBundle := bundles[ref.BundleName]
		citer, hasCiter := citers[NormalizePath("", ref.CitingPath)]
		if !hasBundle || !hasCiter || bundle.Class == ClassRetiredTerminal || citer.Class == ClassRetiredTerminal {
			continue
		}

		currentVersion := bundleReqVersion(bundle, ref.ReqID)
		_, stale := StalePinVerdict(ref.PinVersion, currentVersion)
		if stale {
			staleRefs = append(staleRefs, staleTraceRef{ref: ref, currentVersion: currentVersion, citerID: citer.ID, citerKind: citer.Kind})
		}

		if bundle.Class == ClassSuccessTerminal && citer.Kind == KindSpec && citer.Class != ClassSuccessTerminal {
			violations = append(violations, Violation{
				Rule:     StepRequirementTraceability,
				File:     ref.CitingPath,
				Message:  fmt.Sprintf("delivered bundle %s is cited by non-implemented spec %s (%s); delivered coverage requires every citing spec to be implemented", ref.BundleName, citer.ID, citer.Status),
				Severity: "error",
			})
		}

		if SupportsCoverage(citer.Kind, citer.Class) && !stale {
			covered[ref.BundleName][ref.ReqID] = true
		}
	}

	for _, staleRef := range staleRefs {
		if covered[staleRef.ref.BundleName][staleRef.ref.ReqID] {
			continue
		}
		if staleRef.citerKind == KindSpec {
			staleSpecs[staleRef.citerID] = true
		}
		violations = append(violations, Violation{
			Rule:     StepRequirementTraceability,
			File:     staleRef.ref.CitingPath,
			Message:  fmt.Sprintf("supports ref %s:%s@%s pins an older major/minor than current %s:%s@%s; rework or add a new implemented spec for the current requirement", staleRef.ref.BundleName, staleRef.ref.ReqID, staleRef.ref.PinVersion, staleRef.ref.BundleName, staleRef.ref.ReqID, staleRef.currentVersion),
			Severity: "error",
		})
	}

	for _, rec := range records {
		if rec.Kind != KindPlan || rec.Class == ClassRetiredTerminal {
			continue
		}
		spec, ok := specs[rec.SpecID]
		if ok && staleSpecs[spec.ID] {
			violations = append(violations, Violation{
				Rule:     StepRequirementTraceability,
				File:     rec.Path,
				Message:  fmt.Sprintf("plan %s targets spec %s whose bundle requirement chain does not verify", rec.ID, spec.ID),
				Severity: "error",
			})
		}
	}

	for _, bundle := range bundles {
		if bundle.Class == ClassRetiredTerminal {
			continue
		}
		for _, req := range bundle.BundleReqs {
			if covered[bundle.BundleName][req.ReqID] {
				continue
			}
			severity := "warning"
			rule := StepRequirementTraceabilityAdvisory
			if bundle.Class == ClassSuccessTerminal {
				severity = "error"
				rule = StepRequirementTraceability
			}
			violations = append(violations, Violation{
				Rule:     rule,
				File:     bundle.Path,
				Message:  fmt.Sprintf("bundle %s requirement %s has no implemented-spec coverage", bundle.BundleName, req.ReqID),
				Severity: severity,
			})
		}
	}

	return traceabilityStepResult(StepRequirementTraceability, violations)
}

type staleTraceRef struct {
	ref            TraceRef
	currentVersion string
	citerID        string
	citerKind      ArtifactKind
}

// SplitTraceabilityResult partitions the combined classifier result into the policied
// block step and the structurally non-policied advisory step.
func SplitTraceabilityResult(combined StepResult) (block StepResult, advisory StepResult) {
	var blockViolations, advisoryViolations []Violation
	for _, v := range combined.Violations {
		if v.Severity == "error" {
			blockViolations = append(blockViolations, v)
		} else {
			advisoryViolations = append(advisoryViolations, v)
		}
	}
	return traceabilityStepResult(StepRequirementTraceability, blockViolations),
		traceabilityStepResult(StepRequirementTraceabilityAdvisory, advisoryViolations)
}

func traceabilityStepResult(stepName string, violations []Violation) StepResult {
	if violations == nil {
		violations = []Violation{}
	}
	status := "pass"
	warned := false
	for _, v := range violations {
		if v.Severity == "error" {
			status = "fail"
			break
		}
		if v.Severity == "warning" {
			warned = true
		}
	}
	if status == "pass" && warned {
		status = "warning"
	}
	return StepResult{StepName: stepName, Status: status, Violations: violations}
}

func bundleReqVersion(bundle ArtifactStatusRecord, reqID string) string {
	for _, req := range bundle.BundleReqs {
		if req.ReqID == reqID {
			return req.CurrentVersion
		}
	}
	return ""
}

func semverMajorMinor(version string) (int, int, bool) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, false
	}
	major, okMajor := parseNonNegativeInt(parts[0])
	minor, okMinor := parseNonNegativeInt(parts[1])
	if !okMajor || !okMinor {
		return 0, 0, false
	}
	if _, ok := parseNonNegativeInt(parts[2]); !ok {
		return 0, 0, false
	}
	return major, minor, true
}

func parseNonNegativeInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}
