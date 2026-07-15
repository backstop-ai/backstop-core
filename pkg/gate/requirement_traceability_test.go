package gate

import (
	"strings"
	"testing"
)

func TestTraceStep_BlockStepNameConstant(t *testing.T) {
	if StepRequirementTraceability != "requirement_traceability" {
		t.Fatalf("StepRequirementTraceability = %q", StepRequirementTraceability)
	}
}

func TestTraceStep_AdvisoryStepNameConstant(t *testing.T) {
	if StepRequirementTraceabilityAdvisory != "requirement_traceability_advisory" {
		t.Fatalf("StepRequirementTraceabilityAdvisory = %q", StepRequirementTraceabilityAdvisory)
	}
}

func TestTraceStep_LedgerIntegrityStaysReserved(t *testing.T) {
	if StepLedgerIntegrity != "ledger_integrity" || StepRequirementTraceability == StepLedgerIntegrity || StepRequirementTraceabilityAdvisory == StepLedgerIntegrity {
		t.Fatalf("ledger_integrity must stay reserved, got ledger=%q block=%q advisory=%q", StepLedgerIntegrity, StepRequirementTraceability, StepRequirementTraceabilityAdvisory)
	}
}

func TestTraceStep_SplitPartitionsBySeverity(t *testing.T) {
	block, advisory := SplitTraceabilityResult(StepResult{StepName: StepRequirementTraceability, Status: "fail", Violations: []Violation{
		{Rule: StepRequirementTraceability, File: "a", Message: "block", Severity: "error"},
		{Rule: StepRequirementTraceabilityAdvisory, File: "b", Message: "warn", Severity: "warning"},
	}})
	if block.StepName != StepRequirementTraceability || block.Status != "fail" || len(block.Violations) != 1 {
		t.Fatalf("unexpected block split: %#v", block)
	}
	if advisory.StepName != StepRequirementTraceabilityAdvisory || advisory.Status != "warning" || len(advisory.Violations) != 1 {
		t.Fatalf("unexpected advisory split: %#v", advisory)
	}
}

func TestTrace_Delivered_ImplementedCitingSpecOK(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "pass", "pass", "")
}

func TestTrace_Delivered_DraftCitingSpecBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")), traceSpec("SPEC-001", "draft", ClassNonTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "non-implemented spec")
}

func TestTrace_Delivered_ReadyForImplCitingSpecBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")), traceSpec("SPEC-001", "ready-for-implementation", ClassNonTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "non-implemented spec")
}

func TestTrace_Delivered_ReplacedCitingSpecExcludedFromLiveCheck(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")), traceSpec("SPEC-001", "replaced", ClassRetiredTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "has no implemented-spec coverage")
}

func TestTrace_Bundle_ReplacedBundleExcluded(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "replaced", ClassRetiredTerminal, req("REQ-001", "1.0.0"))}, nil, "pass", "pass", "")
}

func TestTrace_Bundle_CanceledBundleExcluded(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "canceled", ClassRetiredTerminal, req("REQ-001", "1.0.0"))}, nil, "pass", "pass", "")
}

func TestTrace_Bundle_DeprecatedBundleExcluded(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "deprecated", ClassRetiredTerminal, req("REQ-001", "1.0.0"))}, nil, "pass", "pass", "")
}

func TestTrace_Bundle_NameWithoutNumberJoinsByName(t *testing.T) {
	bundle := traceBundle("agent-definitions", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))
	bundle.ID = ""
	assertTrace(t, []ArtifactStatusRecord{bundle, traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("agent-definitions", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "pass", "pass", "")
}

func TestTrace_Delivered_FullCoveragePasses(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"), req("REQ-002", "1.0.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md"), traceRef("feature", "REQ-002", "1.0.0", "specs/SPEC-001.spec.md")}, "pass", "pass", "")
}

func TestTrace_Delivered_UncoveredReqBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, nil, "fail", "pass", "REQ-001")
}

func TestTrace_Delivered_PerReqNotAggregate(t *testing.T) {
	block, advisory := assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"), req("REQ-002", "1.0.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "REQ-002")
	if containsViolation(block, "REQ-001") {
		t.Fatalf("covered REQ-001 should not be reported: block=%#v advisory=%#v", block, advisory)
	}
}

func TestTrace_Coverage_ImplementedSpecCovers(t *testing.T) {
	if !SupportsCoverage(KindSpec, ClassSuccessTerminal) {
		t.Fatal("implemented specs should support coverage")
	}
}

func TestTrace_Coverage_ReplacedSpecDoesNotCover(t *testing.T) {
	assertSpecStatusDoesNotCover(t, "replaced", ClassRetiredTerminal)
}

func TestTrace_Coverage_CanceledSpecDoesNotCover(t *testing.T) {
	assertSpecStatusDoesNotCover(t, "canceled", ClassRetiredTerminal)
}

func TestTrace_Coverage_DeprecatedSpecDoesNotCover(t *testing.T) {
	assertSpecStatusDoesNotCover(t, "deprecated", ClassRetiredTerminal)
}

func TestTrace_Coverage_ObsoletedSpecDoesNotCover(t *testing.T) {
	assertSpecStatusDoesNotCover(t, "obsoleted", ClassRetiredTerminal)
}

func TestTrace_Coverage_DraftSpecDoesNotCover(t *testing.T) {
	assertSpecStatusDoesNotCover(t, "draft", ClassNonTerminal)
}

func TestTrace_Coverage_ReadyForImplSpecDoesNotCover(t *testing.T) {
	assertSpecStatusDoesNotCover(t, "ready-for-implementation", ClassNonTerminal)
}

func TestTrace_Coverage_IssueRefNeverCovers(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")), traceIssue("ISSUE-001", "closed", ClassSuccessTerminal, "issues/ISSUE-001.issue.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "issues/ISSUE-001.issue.md")}, "fail", "pass", "has no implemented-spec coverage")
}

func TestTrace_Lineage_IssueRefQueryable(t *testing.T) {
	// CLM-024: an issue supports ref is RETAINED as a queryable lineage link in the
	// traceability corpus (it participates in classification and surfaces by its own
	// citing path), yet it NEVER provides coverage. Drive the classifier over real
	// records — not a hand-built literal.
	issuePath := "issues/ISSUE-001.issue.md"

	// Non-coverage: a delivered bundle REQ cited ONLY by a closed issue stays
	// uncovered — the issue lineage link does not close delivered coverage.
	coverBlock, coverAdvisory := assertTrace(t, []ArtifactStatusRecord{
		traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")),
		traceIssue("ISSUE-001", "closed", ClassSuccessTerminal, issuePath),
	}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", issuePath)}, "fail", "pass", "has no implemented-spec coverage")
	if !containsViolation(coverBlock, "REQ-001") {
		t.Fatalf("issue-origin ref must not provide coverage of the delivered REQ: block=%#v advisory=%#v", coverBlock, coverAdvisory)
	}

	// Retained + queryable: the issue ref was kept in the corpus (not dropped), so a
	// stale issue pin on an in-flight bundle surfaces a finding attributed to the
	// ISSUE's own citing path — i.e. the lineage link is queryable by that path.
	lineageBlock, lineageAdvisory := assertTrace(t, []ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceIssue("ISSUE-001", "closed", ClassSuccessTerminal, issuePath),
	}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", issuePath)}, "fail", "warning", issuePath)
	if !containsViolation(lineageBlock, issuePath) {
		t.Fatalf("retained issue lineage ref must be queryable by its citing path %q: block=%#v advisory=%#v", issuePath, lineageBlock, lineageAdvisory)
	}
}

func TestTrace_Coverage_IssueRefDoesNotAffectSpecCoverage(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"), traceIssue("ISSUE-001", "closed", ClassSuccessTerminal, "issues/ISSUE-001.issue.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "issues/ISSUE-001.issue.md"), traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "pass", "pass", "")
}

func TestTrace_State_SpecOutrunsChainBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "warning", "pins an older major/minor")
}

func TestTrace_State_PlanForUnverifiedSpecBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"), tracePlan("PLAN-SPEC-001", "completed", ClassSuccessTerminal, "SPEC-001")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "warning", "PLAN-SPEC-001")
}

func TestTrace_State_DeliveredWithoutCoverageBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, nil, "fail", "pass", "has no implemented-spec coverage")
}

func TestTrace_State_SpecWithinChainNoBlock(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.0.1")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "pass", "pass", "")
}

func TestTrace_State_VerdictIsStateNoEventInterception(t *testing.T) {
	res := ClassifyRequirementTraceability([]ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, nil)
	if res.StepName != StepRequirementTraceability || res.ConfigErr {
		t.Fatalf("traceability should return only a state StepResult verdict: %#v", res)
	}
}

func TestTrace_Posture_InFlightUncoveredAdvisoryWarn(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.0.0"))}, nil, "pass", "warning", "REQ-001")
}

func TestTrace_Posture_NonDeliveredUncoveredAdvisory(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "defined", ClassNonTerminal, req("REQ-001", "1.0.0"))}, nil, "pass", "warning", "REQ-001")
}

func TestTrace_Posture_BrokenPromiseBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, nil, "fail", "pass", "REQ-001")
}

func TestTrace_Posture_ResolutionDefectsNotInThisStep(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, []TraceRef{traceRef("missing", "REQ-404", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "has no implemented-spec coverage")
}

func TestTrace_Posture_AdvisoryNotPolicyUpgradable(t *testing.T) {
	block, advisory := SplitTraceabilityResult(StepResult{StepName: StepRequirementTraceability, Status: "warning", Violations: []Violation{{Rule: StepRequirementTraceabilityAdvisory, Severity: "warning"}}})
	if block.Status != "pass" {
		t.Fatalf("advisory split should leave block passing: %#v", block)
	}
	if advisory.StepName != StepRequirementTraceabilityAdvisory || advisory.Status != "warning" {
		t.Fatalf("advisory must stay on distinct warning surface: %#v", advisory)
	}
}

func TestTrace_Posture_NotWaivable(t *testing.T) {
	token, ok := PrefilledWaiverToken(StepRequirementTraceability, Violation{Rule: StepRequirementTraceability, File: "bundles/x.bundle.md", Message: "block", Severity: "error"}, nil, waiverTestNow)
	if ok {
		t.Fatalf("requirement traceability must not mint waiver tokens, got %q", token)
	}
}

func TestTrace_Posture_NotBaselineGrandfathered(t *testing.T) {
	res := ClassifyRequirementTraceability([]ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0"))}, nil)
	if res.Status != "fail" || res.Violations[0].Rule != StepRequirementTraceability {
		t.Fatalf("traceability block should be emitted directly, not as baseline-scoped new-code finding: %#v", res)
	}
}

func TestTrace_StalePin_SameVersionSatisfied(t *testing.T) {
	assertBump(t, "1.0.0", "1.0.0", "none", false)
}

func TestTrace_StalePin_PatchUnimplementedFree(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.0.1")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "pass", "pass", "")
}

func TestTrace_StalePin_PatchDeliveredFree(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.1")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "pass", "pass", "")
}

func TestTrace_StalePin_MinorUnimplementedRepinBlocks(t *testing.T) {
	// CLM-042: a MINOR bump on an unimplemented (in-flight, non-delivered) bundle
	// makes a downstream ref still pinned to the old version a stale-pin BLOCK — the
	// downstream must re-pin. Drives the classifier (not just the predicate): the
	// implemented spec that pinned 1.0.0 no longer satisfies the current 1.1.0.
	block, advisory := assertTrace(t, []ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "1.1.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
	}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "warning", "pins an older major/minor")
	// The block must name the old pin, the current version, and the re-pin remedy.
	if !containsViolation(block, "@1.0.0") || !containsViolation(block, "@1.1.0") || !containsViolation(block, "rework or add a new implemented spec") {
		t.Fatalf("minor stale-pin block must name the old pin, current version, and re-pin remedy: block=%#v advisory=%#v", block, advisory)
	}
}

func TestTrace_StalePin_MajorUnimplementedRepinBlocks(t *testing.T) {
	// CLM-043: a MAJOR bump on an unimplemented bundle makes a downstream old-version
	// pin a stale-pin BLOCK (must re-pin). Classifier-level, mirroring the minor case
	// with a major transition (1.0.0 -> 2.0.0).
	block, advisory := assertTrace(t, []ArtifactStatusRecord{
		traceBundle("feature", "ready", ClassNonTerminal, req("REQ-001", "2.0.0")),
		traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"),
	}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "warning", "pins an older major/minor")
	if !containsViolation(block, "@1.0.0") || !containsViolation(block, "@2.0.0") || !containsViolation(block, "rework or add a new implemented spec") {
		t.Fatalf("major stale-pin block must name the old pin, current version, and re-pin remedy: block=%#v advisory=%#v", block, advisory)
	}
}

func TestTrace_StalePin_MinorDeliveredNewSpecRequiredBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.1.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "pins an older major/minor")
}

func TestTrace_StalePin_MajorDeliveredNewSpecRequiredBlocks(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "2.0.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "pins an older major/minor")
}

func TestTrace_StalePin_DeliveredNewVersionNewSpecImplementedSatisfied(t *testing.T) {
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.1.0")), traceSpec("SPEC-001", "implemented", ClassSuccessTerminal, "specs/SPEC-001.spec.md"), traceSpec("SPEC-002", "implemented", ClassSuccessTerminal, "specs/SPEC-002.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md"), traceRef("feature", "REQ-001", "1.1.0", "specs/SPEC-002.spec.md")}, "pass", "pass", "")
}

func TestTrace_StalePin_CurrentVersionNumericMajorMinor(t *testing.T) {
	assertBump(t, "1.10.0", "1.10.9", "patch", false)
	assertBump(t, "1.9.9", "1.10.0", "minor", true)
}

func assertSpecStatusDoesNotCover(t *testing.T, status string, class StatusClass) {
	t.Helper()
	if SupportsCoverage(KindSpec, class) {
		t.Fatalf("%s spec should not support coverage", status)
	}
	assertTrace(t, []ArtifactStatusRecord{traceBundle("feature", "delivered", ClassSuccessTerminal, req("REQ-001", "1.0.0")), traceSpec("SPEC-001", status, class, "specs/SPEC-001.spec.md")}, []TraceRef{traceRef("feature", "REQ-001", "1.0.0", "specs/SPEC-001.spec.md")}, "fail", "pass", "has no implemented-spec coverage")
}

func assertBump(t *testing.T, pin string, current string, wantBump string, wantStale bool) {
	t.Helper()
	bump, stale := StalePinVerdict(pin, current)
	if bump != wantBump || stale != wantStale {
		t.Fatalf("StalePinVerdict(%q, %q) = %q/%v, want %q/%v", pin, current, bump, stale, wantBump, wantStale)
	}
}

func assertTrace(t *testing.T, records []ArtifactStatusRecord, refs []TraceRef, blockStatus string, advisoryStatus string, wantMessage string) (StepResult, StepResult) {
	t.Helper()
	block, advisory := SplitTraceabilityResult(ClassifyRequirementTraceability(records, refs))
	if block.Status != blockStatus || advisory.Status != advisoryStatus {
		t.Fatalf("unexpected statuses: block=%#v advisory=%#v", block, advisory)
	}
	if wantMessage != "" && !containsViolation(block, wantMessage) && !containsViolation(advisory, wantMessage) {
		t.Fatalf("expected violation containing %q, got block=%#v advisory=%#v", wantMessage, block, advisory)
	}
	return block, advisory
}

func containsViolation(res StepResult, want string) bool {
	for _, v := range res.Violations {
		if strings.Contains(v.Message, want) || strings.Contains(v.File, want) {
			return true
		}
	}
	return false
}

func req(id string, version string) BundleReqVersion {
	return BundleReqVersion{ReqID: id, CurrentVersion: version}
}

func traceRef(bundle string, reqID string, version string, path string) TraceRef {
	return TraceRef{BundleName: bundle, ReqID: reqID, PinVersion: version, Pinned: version != "", CitingPath: path}
}

func traceBundle(name, status string, class StatusClass, reqs ...BundleReqVersion) ArtifactStatusRecord {
	return ArtifactStatusRecord{ID: "BUNDLE-001", Kind: KindBundle, Status: status, Class: class, Path: "bundles/feature.bundle.md", BundleName: name, BundleReqs: reqs}
}

func traceSpec(id, status string, class StatusClass, path string) ArtifactStatusRecord {
	return ArtifactStatusRecord{ID: id, Kind: KindSpec, Status: status, Class: class, Path: path}
}

func traceIssue(id, status string, class StatusClass, path string) ArtifactStatusRecord {
	return ArtifactStatusRecord{ID: id, Kind: KindIssue, Status: status, Class: class, Path: path}
}

func tracePlan(id, status string, class StatusClass, specID string) ArtifactStatusRecord {
	return ArtifactStatusRecord{ID: id, Kind: KindPlan, Status: status, Class: class, Path: "plans/" + id + ".plan.yml", SpecID: specID}
}
