package gate

// Phase-2 drift-classifier polarity tests (ISSUE-042 TASK-004, CLM-003/004/005/006/008).
// These drive ClassifyStatusDrift over HAND-BUILT resolver records + a supplied PRESENT
// test-name set — the ONLY signal the classifier consumes (EXISTENCE). No live gate here:
// the polarity is proven in isolation. The classifier takes NO pass/fail set — a
// present-but-failing test is out of this dimension's scope (subsumed by pack_engines).

import (
	"strings"
	"testing"
)

// presentSet builds the present-test-name set the classifier consumes.
func presentSet(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

// mandated builds a MandatedTest slice from bare function names.
func mandated(names ...string) []MandatedTest {
	out := make([]MandatedTest, 0, len(names))
	for _, n := range names {
		out = append(out, MandatedTest{FuncName: n})
	}
	return out
}

func hasFail(vs []Violation) bool {
	for _, v := range vs {
		if v.Severity == "error" {
			return true
		}
	}
	return false
}

// TestStatusDrift_SuccessTerminalAbsentTest_Blocks (CLM-004): a success-terminal artifact
// with a mandated test NOT in the present set yields a fail StepResult reading as a broken
// promise ("claimed done, isn't"), the ClassDeclaredIntentUnmet treatment (blocks).
func TestStatusDrift_SuccessTerminalAbsentTest_Blocks(t *testing.T) {
	records := []ArtifactStatusRecord{{
		ID:            "ISSUE-002",
		Kind:          KindIssue,
		Status:        "closed",
		Class:         ClassSuccessTerminal,
		MandatedTests: []MandatedTest{{FuncName: "TestCodeCheck_GoneForever", ClaimID: "CLM-001"}},
	}}
	res := ClassifyStatusDrift(records, presentSet())

	if res.Status != "fail" {
		t.Fatalf("status = %q, want fail", res.Status)
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected exactly one drift violation, got %d", len(res.Violations))
	}
	v := res.Violations[0]
	if v.Severity != "error" {
		t.Errorf("violation severity = %q, want error", v.Severity)
	}
	// Reads as a broken promise, reusing the ClassDeclaredIntentUnmet vocabulary, and
	// names both the artifact and the absent mandated test.
	msg := v.Message
	if !strings.Contains(msg, "broken promise") {
		t.Errorf("violation message %q must read as a broken promise", msg)
	}
	if !strings.Contains(msg, "claimed done") {
		t.Errorf("violation message %q must read as 'claimed done, isn't'", msg)
	}
	if !strings.Contains(msg, "ISSUE-002") {
		t.Errorf("violation message %q must name the artifact (ISSUE-002)", msg)
	}
	if !strings.Contains(msg, "TestCodeCheck_GoneForever") {
		t.Errorf("violation message %q must name the absent mandated test", msg)
	}
	// A non-empty ClaimID is surfaced in the message (the claim clause branch).
	if !strings.Contains(msg, "claim CLM-001") {
		t.Errorf("violation message %q must name the mandating claim", msg)
	}
}

// TestStatusDrift_SuccessTerminalPresentTest_NoDriftViolation (CLM-005): a success-terminal
// artifact whose mandated test IS present yields NO drift violation — proving the drift
// dimension does NOT attribute pass/fail (a present-but-failing test is caught by
// pack_engines, not here). The classifier consumes only the present set.
func TestStatusDrift_SuccessTerminalPresentTest_NoDriftViolation(t *testing.T) {
	records := []ArtifactStatusRecord{{
		ID:            "ISSUE-100",
		Kind:          KindIssue,
		Status:        "closed",
		Class:         ClassSuccessTerminal,
		MandatedTests: mandated("TestPresentAndPassingOrFailing"),
	}}
	// The test is PRESENT. Whether it passes or fails is NOT an input here — existence
	// is the only signal — so a present mandated test produces no drift violation.
	res := ClassifyStatusDrift(records, presentSet("TestPresentAndPassingOrFailing"))

	if res.Status == "fail" {
		t.Errorf("status = fail; a present mandated test must NOT drive a drift block (pass/fail is pack_engines' job)")
	}
	if len(res.Violations) != 0 {
		t.Errorf("expected no drift violation for a present mandated test, got %d: %+v", len(res.Violations), res.Violations)
	}
}

// TestStatusDrift_DeliveredButOpen_WarnsNonBlocking (CLM-006): a non-terminal artifact
// with ALL mandated tests PRESENT yields a WARNING StepResult — Status "warning",
// ConfigErr false, a guidance violation tagged Severity "warning". It does NOT fail.
func TestStatusDrift_DeliveredButOpen_WarnsNonBlocking(t *testing.T) {
	records := []ArtifactStatusRecord{{
		ID:            "ISSUE-047",
		Kind:          KindIssue,
		Status:        "open",
		Class:         ClassNonTerminal,
		MandatedTests: mandated("TestLooksDeliveredAlpha", "TestLooksDeliveredBeta"),
	}}
	res := ClassifyStatusDrift(records, presentSet("TestLooksDeliveredAlpha", "TestLooksDeliveredBeta"))

	if res.Status != "warning" {
		t.Fatalf("status = %q, want warning", res.Status)
	}
	if res.ConfigErr {
		t.Error("ConfigErr must be false for the WARN direction (never a config error / block)")
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected one guidance violation, got %d", len(res.Violations))
	}
	if res.Violations[0].Severity != "warning" {
		t.Errorf("guidance violation severity = %q, want warning", res.Violations[0].Severity)
	}
	if hasFail(res.Violations) {
		t.Error("WARN direction must emit NO error-severity violation")
	}
}

// TestStatusDrift_WarnNeverBlocks_HeuristicAsymmetry (CLM-006): across MANY delivered-but-
// open artifacts, the WARN direction never produces a failing/ConfigErr result — the
// heuristic-not-proof asymmetry is structural, never upgraded to a block.
func TestStatusDrift_WarnNeverBlocks_HeuristicAsymmetry(t *testing.T) {
	var records []ArtifactStatusRecord
	var present []string
	for _, id := range []string{"ISSUE-201", "ISSUE-202", "ISSUE-203", "ISSUE-204"} {
		tn := "Test_" + id + "_Delivered"
		records = append(records, ArtifactStatusRecord{
			ID:            id,
			Kind:          KindIssue,
			Status:        "in-progress",
			Class:         ClassNonTerminal,
			MandatedTests: mandated(tn),
		})
		present = append(present, tn)
	}
	res := ClassifyStatusDrift(records, presentSet(present...))

	if res.Status == "fail" {
		t.Error("many delivered-but-open artifacts must never produce a fail status")
	}
	if res.ConfigErr {
		t.Error("the WARN direction must never set ConfigErr")
	}
	if hasFail(res.Violations) {
		t.Error("the WARN direction must never emit an error-severity violation, regardless of count")
	}
	if res.Status != "warning" {
		t.Errorf("status = %q, want warning (all delivered-but-open)", res.Status)
	}
}

// TestStatusDrift_RetiredTerminal_Excluded (CLM-003): a replaced/canceled/deprecated
// artifact produces NO violation regardless of test presence — retired-exclusion preserved.
func TestStatusDrift_RetiredTerminal_Excluded(t *testing.T) {
	records := []ArtifactStatusRecord{
		{ID: "ISSUE-903", Kind: KindIssue, Status: "replaced", Class: ClassRetiredTerminal, MandatedTests: mandated("TestGoneAbsent")},
		{ID: "SPEC-902", Kind: KindSpec, Status: "deprecated", Class: ClassRetiredTerminal, MandatedTests: mandated("TestAlsoGone")},
	}
	// Even with an EMPTY present set (both mandated tests absent), a retired artifact
	// yields no violation.
	res := ClassifyStatusDrift(records, presentSet())

	if len(res.Violations) != 0 {
		t.Errorf("retired-terminal artifacts must produce no violation, got %d: %+v", len(res.Violations), res.Violations)
	}
	if res.Status == "fail" || res.Status == "warning" {
		t.Errorf("status = %q, want pass — retired artifacts are excluded", res.Status)
	}
}

// TestStatusDrift_ExistenceIsTheOnlySignal (CLM-008): the classifier computes its verdict
// from the resolver records + the present-test set ALONE — there is no test-run/pass-fail
// input, no fictional failing-test set. A success-terminal absent test BLOCKS with only
// the present set supplied.
func TestStatusDrift_ExistenceIsTheOnlySignal(t *testing.T) {
	records := []ArtifactStatusRecord{{
		ID:            "SPEC-001",
		Kind:          KindSpec,
		Status:        "implemented",
		Class:         ClassSuccessTerminal,
		MandatedTests: mandated("TestStandardsCompiler_Gone"),
	}}
	// Only the present set is supplied — no pass/fail map exists in the signature.
	res := ClassifyStatusDrift(records, presentSet("SomeUnrelatedTest"))

	if res.Status != "fail" {
		t.Fatalf("status = %q, want fail (success-terminal + absent, existence-only)", res.Status)
	}
	if !hasFail(res.Violations) {
		t.Error("expected an error-severity broken-promise violation from existence alone")
	}

	// Supplying the test as present flips the verdict — existence is the sole lever.
	res2 := ClassifyStatusDrift(records, presentSet("TestStandardsCompiler_Gone"))
	if res2.Status == "fail" {
		t.Error("with the mandated test present, existence-only must not block")
	}
}

// TestSplitDriftResult_PartitionsBySeverity covers SplitDriftResult: a combined result
// with mixed severities splits into an error-only block surface (fail) and a warning-only
// advisory surface (warning); an empty combined yields two clean passes.
func TestSplitDriftResult_PartitionsBySeverity(t *testing.T) {
	combined := StepResult{
		StepName: StepArtifactStatusDrift,
		Status:   "fail",
		Violations: []Violation{
			{Rule: StepArtifactStatusDrift, Severity: "error", Message: "block"},
			{Rule: StepArtifactStatusDriftAdvisory, Severity: "warning", Message: "warn"},
		},
	}
	block, advisory := SplitDriftResult(combined)
	if block.StepName != StepArtifactStatusDrift || block.Status != "fail" || len(block.Violations) != 1 || block.Violations[0].Severity != "error" {
		t.Errorf("block surface must carry the error-severity violation as a fail, got %+v", block)
	}
	if advisory.StepName != StepArtifactStatusDriftAdvisory || advisory.Status != "warning" || len(advisory.Violations) != 1 || advisory.Violations[0].Severity != "warning" {
		t.Errorf("advisory surface must carry the warning-severity violation as a warning, got %+v", advisory)
	}

	eb, ea := SplitDriftResult(StepResult{StepName: StepArtifactStatusDrift, Violations: []Violation{}})
	if eb.Status != "pass" || len(eb.Violations) != 0 || ea.Status != "pass" || len(ea.Violations) != 0 {
		t.Errorf("empty split must yield two clean passes, got block=%+v advisory=%+v", eb, ea)
	}
}

// TestStatusDrift_NonTerminalPartialOrNoTests_NoWarn covers the looksDelivered branches:
// a non-terminal artifact with NO mandated tests, and one with a PARTLY-absent test, both
// yield no WARN (only ALL-present delivered-looking artifacts warn).
func TestStatusDrift_NonTerminalPartialOrNoTests_NoWarn(t *testing.T) {
	records := []ArtifactStatusRecord{
		{ID: "ISSUE-300", Kind: KindIssue, Status: "open", Class: ClassNonTerminal},
		{ID: "ISSUE-301", Kind: KindIssue, Status: "open", Class: ClassNonTerminal, MandatedTests: mandated("TestPresent301", "TestAbsent301")},
	}
	res := ClassifyStatusDrift(records, presentSet("TestPresent301")) // TestAbsent301 missing
	if len(res.Violations) != 0 {
		t.Errorf("non-terminal with no tests OR a partly-absent test must not warn, got %+v", res.Violations)
	}
	if res.Status != "pass" {
		t.Errorf("status = %q, want pass", res.Status)
	}
}

// TestStatusDrift_BrokenPromiseMessage_NoClaimID covers the driftBrokenPromiseMessage
// branch where a mandated test carries no ClaimID: the message omits the claim clause but
// still names the test and reads as a broken promise.
func TestStatusDrift_BrokenPromiseMessage_NoClaimID(t *testing.T) {
	records := []ArtifactStatusRecord{{
		ID:            "SPEC-050",
		Kind:          KindSpec,
		Status:        "implemented",
		Class:         ClassSuccessTerminal,
		MandatedTests: []MandatedTest{{FuncName: "TestNoClaim"}},
	}}
	res := ClassifyStatusDrift(records, presentSet())
	if len(res.Violations) != 1 {
		t.Fatalf("want exactly one violation, got %d", len(res.Violations))
	}
	msg := res.Violations[0].Message
	if strings.Contains(msg, ", claim ") {
		t.Errorf("no ClaimID -> message must omit the claim clause, got %q", msg)
	}
	if !strings.Contains(msg, "broken promise") || !strings.Contains(msg, "TestNoClaim") {
		t.Errorf("message must still name the test and read as a broken promise, got %q", msg)
	}
}
