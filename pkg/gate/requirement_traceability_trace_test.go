package gate

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// traceBundlePath builds a bundle record with a caller-chosen name and path so a
// multi-bundle corpus can produce every gap kind side by side (the shared-path
// traceBundle helper is fine for single-bundle cases).
func traceBundlePath(name, path, status string, class StatusClass, reqs ...BundleReqVersion) ArtifactStatusRecord {
	return ArtifactStatusRecord{ID: "BUNDLE-" + name, Kind: KindBundle, Status: status, Class: class, Path: path, BundleName: name, BundleReqs: reqs}
}

func classifyViolations(records []ArtifactStatusRecord, refs []TraceRef) []Violation {
	return ClassifyRequirementTraceability(records, refs).Violations
}

func findTraceByReq(vs []Violation, reqID string) (Violation, bool) {
	for _, v := range vs {
		if v.Trace != nil && v.Trace.ReqID == reqID {
			return v, true
		}
	}
	return Violation{}, false
}

func findTraceByGapKind(vs []Violation, gapKind string) (Violation, bool) {
	for _, v := range vs {
		if v.Trace != nil && v.Trace.GapKind == gapKind {
			return v, true
		}
	}
	return Violation{}, false
}

// allFiveGapKindCorpus builds a records+refs corpus that drives all five gap kinds
// through ClassifyRequirementTraceability in a single call.
func allFiveGapKindCorpus() ([]ArtifactStatusRecord, []TraceRef) {
	records := []ArtifactStatusRecord{
		// coverage_lapsed: delivered bundle whose only citer for REQ-001 is a replaced spec.
		traceBundlePath("lapse", "bundles/lapse.bundle.md", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")),
		traceSpec("SPEC-LAPSE", "replaced", ClassRetiredTerminal, "specs/SPEC-LAPSE.spec.md"),
		// uncovered: delivered bundle REQ never cited.
		traceBundlePath("uncov", "bundles/uncov.bundle.md", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")),
		// citing_spec_not_implemented: delivered bundle cited by a draft spec.
		traceBundlePath("citing", "bundles/citing.bundle.md", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")),
		traceSpec("SPEC-CITING", "draft", ClassNonTerminal, "specs/SPEC-CITING.spec.md"),
		// stale_pin + chain_outrun: in-flight bundle at 1.1.0 with an implemented spec pinned to 1.0.0,
		// and a plan targeting that spec.
		traceBundlePath("stale", "bundles/stale.bundle.md", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-STALE", "implemented", ClassSuccessTerminal, "specs/SPEC-STALE.spec.md"),
		tracePlan("PLAN-STALE", "completed", ClassSuccessTerminal, "SPEC-STALE"),
	}
	refs := []TraceRef{
		traceRef("lapse", "REQ-001", "1.0.0", "specs/SPEC-LAPSE.spec.md"),
		traceRef("citing", "REQ-001", "1.0.0", "specs/SPEC-CITING.spec.md"),
		traceRef("stale", "REQ-001", "1.0.0", "specs/SPEC-STALE.spec.md"),
	}
	return records, refs
}

func TestTrace_Fields_EveryViolationCarriesTraceObject(t *testing.T) {
	records, refs := allFiveGapKindCorpus()
	vs := classifyViolations(records, refs)
	if len(vs) == 0 {
		t.Fatal("expected violations across all five gap kinds, got none")
	}
	seen := map[string]bool{}
	for _, v := range vs {
		if v.Trace == nil {
			t.Fatalf("violation %q has nil trace: %#v", v.Message, v)
		}
		seen[v.Trace.GapKind] = true
	}
	for _, want := range []string{"uncovered", "coverage_lapsed", "citing_spec_not_implemented", "stale_pin", "chain_outrun"} {
		if !seen[want] {
			t.Fatalf("gap kind %q missing from corpus; saw %v", want, seen)
		}
	}
}

func TestTrace_Fields_AdvisoryStepAlsoCarriesTraceObject(t *testing.T) {
	// A non-delivered bundle REQ with no coverage is a WARN on the advisory step.
	block, advisory := SplitTraceabilityResult(ClassifyRequirementTraceability(
		[]ArtifactStatusRecord{traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.0.0"))}, nil))
	if len(advisory.Violations) == 0 {
		t.Fatalf("expected an advisory violation, got block=%#v advisory=%#v", block, advisory)
	}
	for _, v := range advisory.Violations {
		if v.Trace == nil {
			t.Fatalf("advisory violation carries nil trace: %#v", v)
		}
	}
}

func TestTrace_Fields_UncoveredCarriesBundleReqAndVersion(t *testing.T) {
	vs := classifyViolations([]ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.2.3"))}, nil)
	v, ok := findTraceByReq(vs, "REQ-001")
	if !ok {
		t.Fatalf("no violation for REQ-001: %#v", vs)
	}
	if v.Trace.Bundle != "feature" || v.Trace.ReqID != "REQ-001" || v.Trace.ReqVersion != "1.2.3" {
		t.Fatalf("trace bundle/req/version mismatch: %#v", v.Trace)
	}
}

func TestTrace_Fields_BundleMaturityMatchesRecordAtClassification(t *testing.T) {
	vs := classifyViolations([]ArtifactStatusRecord{traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.0.0"))}, nil)
	v, ok := findTraceByReq(vs, "REQ-001")
	if !ok {
		t.Fatalf("no violation for REQ-001: %#v", vs)
	}
	if v.Trace.BundleMaturity != "ready" {
		t.Fatalf("bundle_maturity = %q, want %q", v.Trace.BundleMaturity, "ready")
	}
}

func TestTrace_GapKind_UncoveredMapsToAuthorSpec(t *testing.T) {
	vs := classifyViolations([]ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, nil)
	v, ok := findTraceByReq(vs, "REQ-001")
	if !ok {
		t.Fatalf("no violation for REQ-001: %#v", vs)
	}
	if v.Trace.GapKind != "uncovered" || v.Trace.Remedy != "author_spec" {
		t.Fatalf("gap_kind/remedy = %q/%q, want uncovered/author_spec", v.Trace.GapKind, v.Trace.Remedy)
	}
}

func TestTrace_GapKind_CoverageLapsedDistinctFromUncovered(t *testing.T) {
	records := []ArtifactStatusRecord{
		traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"), req("REQ-002", "1.0.0"), req("REQ-003", "1.0.0")),
		traceSpec("SPEC-001", "replaced", ClassRetiredTerminal, "specs/SPEC-001.spec.md"),
		traceIssue("ISSUE-001", "obsoleted", ClassRetiredTerminal, "issues/ISSUE-001.issue.md"),
	}
	refs := []TraceRef{
		// REQ-001: only citer is a retired-terminal SPEC → coverage_lapsed.
		traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md"),
		// REQ-003: only citer is a retired-terminal ISSUE → uncovered, NOT lapsed.
		traceRef("feature", "REQ-003", "1.0.0", "issues/ISSUE-001.issue.md"),
		// REQ-002: never cited → uncovered.
	}
	vs := classifyViolations(records, refs)

	lapsed, ok := findTraceByReq(vs, "REQ-001")
	if !ok || lapsed.Trace.GapKind != "coverage_lapsed" || lapsed.Trace.Remedy != "restate_supports" {
		t.Fatalf("REQ-001 must be coverage_lapsed/restate_supports, got %#v", lapsed.Trace)
	}
	neverCited, ok := findTraceByReq(vs, "REQ-002")
	if !ok || neverCited.Trace.GapKind != "uncovered" || neverCited.Trace.Remedy != "author_spec" {
		t.Fatalf("REQ-002 must be uncovered/author_spec, got %#v", neverCited.Trace)
	}
	retiredIssue, ok := findTraceByReq(vs, "REQ-003")
	if !ok || retiredIssue.Trace.GapKind != "uncovered" || retiredIssue.Trace.Remedy != "author_spec" {
		t.Fatalf("REQ-003 (retired ISSUE citer) must be uncovered/author_spec, not lapsed, got %#v", retiredIssue.Trace)
	}
}

func TestTrace_GapKind_CitingSpecNotImplementedMapsToImplementSpec(t *testing.T) {
	records := []ArtifactStatusRecord{
		traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")),
		traceSpec("SPEC-001", "draft", ClassNonTerminal, "specs/SPEC-001.spec.md"),
	}
	refs := []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}
	vs := classifyViolations(records, refs)
	v, ok := findTraceByGapKind(vs, "citing_spec_not_implemented")
	if !ok {
		t.Fatalf("no citing_spec_not_implemented violation: %#v", vs)
	}
	if v.Trace.Remedy != "implement_spec" {
		t.Fatalf("remedy = %q, want implement_spec", v.Trace.Remedy)
	}
}

func TestTrace_GapKind_ChainOutrunMapsToResolveUpstream(t *testing.T) {
	records := []ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
		tracePlan("PLAN-SPEC-001", "completed", ClassSuccessTerminal, "SPEC-001"),
	}
	refs := []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}
	vs := classifyViolations(records, refs)
	v, ok := findTraceByGapKind(vs, "chain_outrun")
	if !ok {
		t.Fatalf("no chain_outrun violation: %#v", vs)
	}
	if v.Trace.Remedy != "resolve_upstream" {
		t.Fatalf("remedy = %q, want resolve_upstream", v.Trace.Remedy)
	}
}

func TestTrace_StalePin_RemedyRePinWhenBundleUnimplemented(t *testing.T) {
	records := []ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
	}
	refs := []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}
	vs := classifyViolations(records, refs)
	v, ok := findTraceByGapKind(vs, "stale_pin")
	if !ok {
		t.Fatalf("no stale_pin violation: %#v", vs)
	}
	if v.Trace.Remedy != "re_pin" {
		t.Fatalf("remedy = %q, want re_pin", v.Trace.Remedy)
	}
}

func TestTrace_StalePin_RemedyNewSpecWhenBundleDelivered(t *testing.T) {
	records := []ArtifactStatusRecord{
		traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
	}
	refs := []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}
	vs := classifyViolations(records, refs)
	v, ok := findTraceByGapKind(vs, "stale_pin")
	if !ok {
		t.Fatalf("no stale_pin violation: %#v", vs)
	}
	if v.Trace.Remedy != "new_spec" {
		t.Fatalf("remedy = %q, want new_spec", v.Trace.Remedy)
	}
}

func TestTrace_StalePin_PinnedVersionAndBumpPopulated(t *testing.T) {
	records := []ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
	}
	refs := []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}
	vs := classifyViolations(records, refs)
	v, ok := findTraceByGapKind(vs, "stale_pin")
	if !ok {
		t.Fatalf("no stale_pin violation: %#v", vs)
	}
	if v.Trace.PinnedVersion != "1.0.0" || v.Trace.Bump != "minor" {
		t.Fatalf("pinned_version/bump = %q/%q, want 1.0.0/minor", v.Trace.PinnedVersion, v.Trace.Bump)
	}
}

func TestTrace_CitingArtifact_PresentOnStalePinAndCitingSpecNotImplemented(t *testing.T) {
	stalePinVs := classifyViolations([]ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
	}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")})
	stalePin, ok := findTraceByGapKind(stalePinVs, "stale_pin")
	if !ok || stalePin.Trace.CitingArtifact != "SPEC-001" {
		t.Fatalf("stale_pin citing_artifact must be SPEC-001, got %#v", stalePin.Trace)
	}

	citingVs := classifyViolations([]ArtifactStatusRecord{
		traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")),
		traceSpec("SPEC-002", "draft", ClassNonTerminal, "specs/SPEC-002.spec.md"),
	}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-002.spec.md")})
	citing, ok := findTraceByGapKind(citingVs, "citing_spec_not_implemented")
	if !ok || citing.Trace.CitingArtifact != "SPEC-002" {
		t.Fatalf("citing_spec_not_implemented citing_artifact must be SPEC-002, got %#v", citing.Trace)
	}
}

func TestTrace_CitingArtifact_AbsentOnUncoveredAndChainOutrun(t *testing.T) {
	uncoveredVs := classifyViolations([]ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, nil)
	uncovered, ok := findTraceByGapKind(uncoveredVs, "uncovered")
	if !ok {
		t.Fatalf("no uncovered violation: %#v", uncoveredVs)
	}
	if uncovered.Trace.CitingArtifact != "" {
		t.Fatalf("uncovered must omit citing_artifact, got %q", uncovered.Trace.CitingArtifact)
	}

	chainVs := classifyViolations([]ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
		tracePlan("PLAN-SPEC-001", "completed", ClassSuccessTerminal, "SPEC-001"),
	}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")})
	chain, ok := findTraceByGapKind(chainVs, "chain_outrun")
	if !ok {
		t.Fatalf("no chain_outrun violation: %#v", chainVs)
	}
	if chain.Trace.CitingArtifact != "" {
		t.Fatalf("chain_outrun must omit citing_artifact, got %q", chain.Trace.CitingArtifact)
	}
}

func TestTrace_ChainOutrun_ViaPointsAtNonVerifyingSpec(t *testing.T) {
	records := []ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
		tracePlan("PLAN-SPEC-001", "completed", ClassSuccessTerminal, "SPEC-001"),
	}
	refs := []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}
	vs := classifyViolations(records, refs)
	v, ok := findTraceByGapKind(vs, "chain_outrun")
	if !ok {
		t.Fatalf("no chain_outrun violation: %#v", vs)
	}
	if v.Trace.Via != "SPEC-001" {
		t.Fatalf("via = %q, want SPEC-001", v.Trace.Via)
	}
	if v.Trace.Via == "PLAN-SPEC-001" {
		t.Fatalf("via must point at the upstream spec, not the violating plan's own ID")
	}
	// Also carries the originating bundle/req/version and maturity, not left empty.
	if v.Trace.Bundle != "feature" || v.Trace.ReqID != "REQ-001" || v.Trace.ReqVersion != "1.1.0" || v.Trace.BundleMaturity != "ready" {
		t.Fatalf("chain_outrun must carry originating bundle/req/version/maturity: %#v", v.Trace)
	}
}

func TestTrace_ForwardCompat_UnknownGapKindDegradesToMessage(t *testing.T) {
	v := Violation{
		Rule:     StepRequirementTraceability,
		File:     "bundles/x.bundle.md",
		Message:  "bundle x requirement REQ-001 has no implemented-spec coverage",
		Severity: "error",
		Trace:    &Trace{Bundle: "x", ReqID: "REQ-001", GapKind: "some_future_gap_kind", Remedy: "some_future_remedy"},
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Violation
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal must not error on unknown gap_kind/remedy: %v", err)
	}
	if back.Message != v.Message {
		t.Fatalf("prose message not preserved: got %q", back.Message)
	}
	if back.Trace == nil || back.Trace.GapKind != "some_future_gap_kind" || back.Trace.Remedy != "some_future_remedy" {
		t.Fatalf("unknown enum values must be retained verbatim as strings: %#v", back.Trace)
	}
}

func TestTrace_Identity_UnchangedByTraceFieldAddition(t *testing.T) {
	base := Violation{Rule: StepRequirementTraceability, File: "bundles/x.bundle.md", Message: "m", Severity: "error"}
	withNil := EnrichViolationIdentity(base)

	withTrace := base
	withTrace.Trace = &Trace{Bundle: "x", BundleMaturity: "delivered", ReqID: "REQ-001", ReqVersion: "1.0.0", GapKind: "uncovered", Remedy: "author_spec"}
	withTrace = EnrichViolationIdentity(withTrace)

	if withNil.Identity != withTrace.Identity {
		t.Fatalf("Identity changed by trace: %q vs %q", withNil.Identity, withTrace.Identity)
	}
	if withNil.IdentityHash != withTrace.IdentityHash {
		t.Fatalf("IdentityHash changed by trace: %q vs %q", withNil.IdentityHash, withTrace.IdentityHash)
	}
	if withNil.RegionHash != withTrace.RegionHash {
		t.Fatalf("RegionHash changed by trace: %q vs %q", withNil.RegionHash, withTrace.RegionHash)
	}
}

func TestTrace_Baseline_PreExistingBaselineStillCompares(t *testing.T) {
	// A baseline whose violation carries NO trace.
	baseViolation := Violation{Rule: StepRequirementTraceability, File: "bundles/x.bundle.md", Message: "m", Severity: "error"}
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{baseViolation}}

	// The current run's identical violation now carries a trace object.
	current := []Violation{{
		Rule:     StepRequirementTraceability,
		File:     "bundles/x.bundle.md",
		Message:  "m",
		Severity: "error",
		Trace:    &Trace{Bundle: "x", ReqID: "REQ-001", GapKind: "uncovered", Remedy: "author_spec"},
	}}

	cmp := CompareBaseline(current, baseline, BaselineCompareOptions{})
	if len(cmp.NewViolations) != 0 {
		t.Fatalf("trace addition must not break grandfathering, got new=%#v", cmp.NewViolations)
	}
	if len(cmp.FixedViolations) != 0 {
		t.Fatalf("identity match must not report the baseline finding fixed, got fixed=%#v", cmp.FixedViolations)
	}
}

func TestTrace_Vocabulary_NoDeliveredUsedForCoverageState(t *testing.T) {
	// The emitted gap_kind/remedy vocabulary never uses "delivered" for REQ-level state.
	records := []ArtifactStatusRecord{
		traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"), req("REQ-002", "1.0.0")),
		traceSpec("SPEC-001", "replaced", ClassRetiredTerminal, "specs/SPEC-001.spec.md"),
	}
	refs := []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}
	for _, v := range classifyViolations(records, refs) {
		if v.Trace == nil {
			continue
		}
		if strings.Contains(v.Trace.GapKind, "delivered") || strings.Contains(v.Trace.Remedy, "delivered") {
			t.Fatalf("trace vocabulary must not use 'delivered' for coverage state: %#v", v.Trace)
		}
	}

	// The Trace type's own doc comment uses covered/coverage vocabulary, never "delivered".
	src, err := os.ReadFile("requirement_traceability.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	block := traceTypeDocRegion(string(src))
	if block == "" {
		t.Fatal("could not locate the Trace type definition + doc comment in source")
	}
	if strings.Contains(strings.ToLower(block), "delivered") {
		t.Fatalf("Trace type doc comment must not use 'delivered' for REQ coverage state:\n%s", block)
	}
}

// traceTypeDocRegion extracts the contiguous doc-comment block immediately above
// `type Trace struct` plus the struct body, so the vocabulary check is scoped to the
// NEW type and never trips on the pre-existing bundle-maturity message elsewhere.
func traceTypeDocRegion(src string) string {
	lines := strings.Split(src, "\n")
	structIdx := -1
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "type Trace struct") {
			structIdx = i
			break
		}
	}
	if structIdx < 0 {
		return ""
	}
	start := structIdx
	for start-1 >= 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "//") {
		start--
	}
	end := structIdx
	for end < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[end]), "}") {
		end++
	}
	if end < len(lines) {
		end++
	}
	return strings.Join(lines[start:end], "\n")
}
