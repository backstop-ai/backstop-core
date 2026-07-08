package gate

// ISSUE-042 Phase 2 (CLM-003/004/005/006/008): the status/reality drift classifier. It
// consumes the Phase-1 resolver records plus ONE supplied signal — the PRESENT test-name
// set (EXISTENCE) — and emits the pinned WARN/BLOCK/excluded polarity. It takes NO
// pass/fail set: the gate exposes only a WHOLE-SUITE pass/fail (pack_engines), never a
// per-test verdict, so a present-but-FAILING mandated test is out of this dimension's
// scope (subsumed by the existing pack_engines/test step). EXISTENCE is the sole lever.
//
// Polarity (mirrors traceability_polarity.go's ClassDeclaredIntentUnmet broken-promise
// treatment; reuse its vocabulary, do NOT invent new classes):
//   - retired-terminal              -> excluded, no violation (CLM-003).
//   - success-terminal + ABSENT     -> error-severity broken-promise violation (CLM-004).
//   - non-terminal + ALL PRESENT    -> warning-severity guidance violation (CLM-006);
//                                      NEVER a fail/ConfigErr — the delivered-heuristic-
//                                      not-proof asymmetry is structural, never blocks.

import "fmt"

// ClassifyStatusDrift computes the drift verdict for a set of resolved artifact records
// against the present test-name set (EXISTENCE only). It returns ONE combined StepResult
// under StepArtifactStatusDrift carrying both the error-severity BLOCK violations
// (success-terminal + absent) and the warning-severity WARN violations (delivered-but-
// open). Status is "fail" if any block, else "warning" if any warn, else "pass". The
// gate wiring splits this into a policied block surface and a non-policied advisory
// surface via SplitDriftResult so the WARN direction can never be upgraded to a block.
//
// The signature accepts ONLY records + the present set — there is deliberately no
// pass/fail parameter (CLM-005/008): a failing-but-present mandated test is pack_engines'
// job, not this dimension's.
func ClassifyStatusDrift(records []ArtifactStatusRecord, present map[string]bool) StepResult {
	var violations []Violation
	for _, rec := range records {
		switch rec.Class {
		case ClassRetiredTerminal:
			// Excluded — no delivery obligation. Preserves ISSUE-031's retired-exclusion.
			continue

		case ClassSuccessTerminal:
			// Broken-promise direction: every mandated test MUST exist. An absent one is
			// a delivered-claim the codebase does not back — block.
			for _, mt := range rec.MandatedTests {
				if present[mt.FuncName] {
					continue
				}
				violations = append(violations, Violation{
					Rule:     StepArtifactStatusDrift,
					File:     rec.Path,
					Message:  driftBrokenPromiseMessage(rec, mt),
					Severity: "error",
				})
			}

		case ClassNonTerminal:
			// Delivered-but-open direction: if the artifact declares mandated tests and
			// ALL of them are present, it LOOKS delivered — warn to advance/close its
			// status. This is a HEURISTIC (present tests suggest coverage, not the whole
			// intent), so it is WARN-only and NEVER blocks.
			if looksDelivered(rec, present) {
				violations = append(violations, Violation{
					Rule:     StepArtifactStatusDriftAdvisory,
					File:     rec.Path,
					Message:  driftDeliveredOpenMessage(rec),
					Severity: "warning",
				})
			}
		}
	}

	return driftStepResult(StepArtifactStatusDrift, violations)
}

// looksDelivered reports whether a non-terminal artifact declares at least one mandated
// test AND every mandated test is present — the delivered-looking heuristic.
func looksDelivered(rec ArtifactStatusRecord, present map[string]bool) bool {
	if len(rec.MandatedTests) == 0 {
		return false
	}
	for _, mt := range rec.MandatedTests {
		if !present[mt.FuncName] {
			return false
		}
	}
	return true
}

// driftStepResult assembles the combined StepResult from a set of drift violations,
// setting Status "fail" if any error-severity violation exists, else "warning" if any
// warning-severity violation exists, else "pass". ConfigErr is ALWAYS false — a drift
// finding is a real finding, grandfatherable against the baseline (CLM-012/015), NOT a
// config error (which would halt the gate and bypass policy grandfathering).
func driftStepResult(stepName string, violations []Violation) StepResult {
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
	return StepResult{
		StepName:   stepName,
		Status:     status,
		ConfigErr:  false,
		Violations: violations,
	}
}

// SplitDriftResult partitions a combined drift StepResult (from ClassifyStatusDrift) into
// two surfaces the gate wiring emits as distinct steps:
//
//   - block: StepArtifactStatusDrift carrying ONLY the error-severity (success-terminal +
//     absent) violations. This is the POLICIED surface (level: block, applies-to:
//     new-code) — its pre-existing findings grandfather against the baseline while
//     net-new ones block.
//   - advisory: StepArtifactStatusDriftAdvisory carrying ONLY the warning-severity
//     (delivered-but-open) violations. It carries NO policy entry, so its intrinsic
//     "warning" status is structurally non-blocking — no policy can upgrade the WARN
//     direction to a block (CLM-006/010).
//
// Splitting by SEVERITY (not re-running the classifier) keeps the two surfaces derived
// from a single existence resolution.
func SplitDriftResult(combined StepResult) (block StepResult, advisory StepResult) {
	var blockViolations, advisoryViolations []Violation
	for _, v := range combined.Violations {
		if v.Severity == "error" {
			blockViolations = append(blockViolations, v)
		} else {
			advisoryViolations = append(advisoryViolations, v)
		}
	}
	return driftStepResult(StepArtifactStatusDrift, blockViolations),
		driftStepResult(StepArtifactStatusDriftAdvisory, advisoryViolations)
}

// driftBrokenPromiseMessage renders the fail-loud-and-useful BLOCK message, reusing the
// ClassDeclaredIntentUnmet broken-promise vocabulary ("broken promise", named subject,
// remediation) rather than inventing new phrasing.
func driftBrokenPromiseMessage(rec ArtifactStatusRecord, mt MandatedTest) string {
	claim := ""
	if mt.ClaimID != "" {
		claim = fmt.Sprintf(", claim %s", mt.ClaimID)
	}
	return fmt.Sprintf(
		"artifact %s is %s (%s) but its mandated test %s is ABSENT%s — a broken promise (claimed done, isn't). "+
			"Restore/repoint the mandated test, or retire the artifact (replaced/canceled/deprecated) if the promise no longer holds.",
		rec.ID, rec.Status, rec.Class, mt.FuncName, claim,
	)
}

// driftDeliveredOpenMessage renders the non-blocking WARN guidance for a delivered-but-
// open artifact.
func driftDeliveredOpenMessage(rec ArtifactStatusRecord) string {
	return fmt.Sprintf(
		"artifact %s is %s (%s) but all its mandated tests are PRESENT — it looks delivered. "+
			"Advance its status toward closure if the work is in fact done. This advisory is non-blocking (exit 0).",
		rec.ID, rec.Status, rec.Class,
	)
}
