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
	// lapsedCiter[bundleName][reqID] records that a retired-terminal SPEC citer once
	// supported this bundle+REQ (DD-3). It is what lets the emission loop tell a REQ whose
	// coverage LAPSED (an existing spec need only be re-pointed) apart from one that was
	// never covered (a new spec must be authored) — the two are otherwise indistinguishable
	// once the retired citer is skipped. Only SPEC citers count: issues never provide
	// coverage (DD-9), so a retired ISSUE citer is not lapse.
	lapsedCiter := map[string]map[string]bool{}
	staleSpecs := map[string]staleSpecOrigin{}
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
		if !hasBundle || !hasCiter || bundle.Class == ClassRetiredTerminal {
			continue
		}
		if citer.Class == ClassRetiredTerminal {
			// The retired citer is skipped from live checks, but a retired SPEC citer is
			// the DD-3 lapsed-coverage signal — record it before dropping the ref so the
			// emission loop can distinguish coverage_lapsed from uncovered. Issues never
			// cover (DD-9), so a retired ISSUE citer is not lapse.
			if citer.Kind == KindSpec {
				if lapsedCiter[ref.BundleName] == nil {
					lapsedCiter[ref.BundleName] = map[string]bool{}
				}
				lapsedCiter[ref.BundleName][ref.ReqID] = true
			}
			continue
		}

		currentVersion := bundleReqVersion(bundle, ref.ReqID)
		bump, stale := StalePinVerdict(ref.PinVersion, currentVersion)
		if stale {
			staleRefs = append(staleRefs, staleTraceRef{ref: ref, currentVersion: currentVersion, citerID: citer.ID, citerKind: citer.Kind, bump: bump})
		}

		if bundle.Class == ClassSuccessTerminal && citer.Kind == KindSpec && citer.Class != ClassSuccessTerminal {
			violations = append(violations, Violation{
				Rule:     StepRequirementTraceability,
				File:     ref.CitingPath,
				Message:  fmt.Sprintf("delivered bundle %s is cited by non-implemented spec %s (%s); delivered coverage requires every citing spec to be implemented", ref.BundleName, citer.ID, citer.Status),
				Severity: "error",
				Trace: &Trace{
					Bundle:         ref.BundleName,
					BundleMaturity: bundle.Status,
					ReqID:          ref.ReqID,
					ReqVersion:     currentVersion,
					GapKind:        "citing_spec_not_implemented",
					Remedy:         "implement_spec",
					CitingArtifact: citer.ID,
				},
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
		bundle := bundles[staleRef.ref.BundleName]
		if staleRef.citerKind == KindSpec {
			staleSpecs[staleRef.citerID] = staleSpecOrigin{bundleName: staleRef.ref.BundleName, reqID: staleRef.ref.ReqID, reqVersion: staleRef.currentVersion}
		}
		// DD-12 lifecycle-keyed remedy: an in-flight bundle re-pins the current version;
		// a covered (success-terminal) bundle's specs are immutable, so a new implemented
		// spec is required.
		remedy := "re_pin"
		if bundle.Class == ClassSuccessTerminal {
			remedy = "new_spec"
		}
		violations = append(violations, Violation{
			Rule:     StepRequirementTraceability,
			File:     staleRef.ref.CitingPath,
			Message:  fmt.Sprintf("supports ref %s:%s@%s pins an older major/minor than current %s:%s@%s; rework or add a new implemented spec for the current requirement", staleRef.ref.BundleName, staleRef.ref.ReqID, staleRef.ref.PinVersion, staleRef.ref.BundleName, staleRef.ref.ReqID, staleRef.currentVersion),
			Severity: "error",
			Trace: &Trace{
				Bundle:         staleRef.ref.BundleName,
				BundleMaturity: bundle.Status,
				ReqID:          staleRef.ref.ReqID,
				ReqVersion:     staleRef.currentVersion,
				GapKind:        "stale_pin",
				Remedy:         remedy,
				PinnedVersion:  staleRef.ref.PinVersion,
				Bump:           staleRef.bump,
				CitingArtifact: staleRef.citerID,
			},
		})
	}

	for _, rec := range records {
		if rec.Kind != KindPlan || rec.Class == ClassRetiredTerminal {
			continue
		}
		spec, ok := specs[rec.SpecID]
		origin, stale := staleSpecs[spec.ID]
		if ok && stale {
			violations = append(violations, Violation{
				Rule:     StepRequirementTraceability,
				File:     rec.Path,
				Message:  fmt.Sprintf("plan %s targets spec %s whose bundle requirement chain does not verify", rec.ID, spec.ID),
				Severity: "error",
				Trace: &Trace{
					Bundle:         origin.bundleName,
					BundleMaturity: bundles[origin.bundleName].Status,
					ReqID:          origin.reqID,
					ReqVersion:     origin.reqVersion,
					GapKind:        "chain_outrun",
					Remedy:         "resolve_upstream",
					Via:            spec.ID,
				},
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
			gapKind := "uncovered"
			remedy := "author_spec"
			if lapsedCiter[bundle.BundleName][req.ReqID] {
				gapKind = "coverage_lapsed"
				remedy = "restate_supports"
			}
			violations = append(violations, Violation{
				Rule:     rule,
				File:     bundle.Path,
				Message:  fmt.Sprintf("bundle %s requirement %s has no implemented-spec coverage", bundle.BundleName, req.ReqID),
				Severity: severity,
				Trace: &Trace{
					Bundle:         bundle.BundleName,
					BundleMaturity: bundle.Status,
					ReqID:          req.ReqID,
					ReqVersion:     req.CurrentVersion,
					GapKind:        gapKind,
					Remedy:         remedy,
				},
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
	bump           string
}

// staleSpecOrigin carries the originating bundle+REQ of a stale spec ref forward to the
// chain_outrun emission site, where only the plan and the targeted spec are otherwise in
// scope, so a chain_outrun trace can report the bundle/req/version its upstream chain fails
// on rather than leaving them empty.
type staleSpecOrigin struct {
	bundleName string
	reqID      string
	reqVersion string
}

// Trace is the ISSUE-059 structured payload attached to every requirement_traceability
// violation (both the block and advisory steps). It carries the same facts the prose
// Message states, in machine-readable form, so a corpus-parsing consumer never has to
// scrape the message string. Unconditional fields (Bundle, BundleMaturity, ReqID,
// ReqVersion, GapKind, Remedy) are populated on every violation; the remainder are
// gap-kind-specific and omitempty. GapKind names the kind of coverage gap and Remedy the
// action that closes it, one-to-one with the BUNDLE-014 design decisions:
//   - uncovered / author_spec: a bundle REQ that never had implemented-spec coverage.
//   - coverage_lapsed / restate_supports: a bundle REQ whose only support is a
//     retired-terminal spec — coverage lapsed, so re-point an existing spec's supports ref
//     rather than authoring a new one.
//   - citing_spec_not_implemented / implement_spec: a covered bundle cited by a spec that
//     is not yet implemented.
//   - stale_pin / re_pin | new_spec: a supports ref pins an older major/minor than the
//     REQ's current version; PinnedVersion and Bump describe the drift and Remedy is
//     lifecycle-keyed.
//   - chain_outrun / resolve_upstream: a downstream artifact whose upstream chain does not
//     verify; Via names the upstream link to fix first.
//
// GapKind and Remedy are plain strings, not a closed Go enum, precisely so this is
// forward-compatible under gate/v1: a consumer that encounters an unrecognized value MUST
// degrade to rendering the prose Message and MUST NOT error — an unknown value is retained
// verbatim on round-trip, never rejected. A vocabulary-breaking change requires a gate
// schema_version bump, not a silent reinterpretation. Vocabulary is deliberate: these
// fields describe REQ-level coverage state ("covered"/"coverage"), never bundle maturity.
type Trace struct {
	Bundle         string `json:"bundle"`
	BundleMaturity string `json:"bundle_maturity"`
	ReqID          string `json:"req_id"`
	ReqVersion     string `json:"req_version"`
	GapKind        string `json:"gap_kind"`
	Remedy         string `json:"remedy"`
	PinnedVersion  string `json:"pinned_version,omitempty"`
	Bump           string `json:"bump,omitempty"`
	CitingArtifact string `json:"citing_artifact,omitempty"`
	Via            string `json:"via,omitempty"`
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
