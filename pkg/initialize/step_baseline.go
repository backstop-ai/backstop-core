package initialize

import (
	"errors"
	"fmt"
)

// stepBaselineName is the report name for step 8.
const stepBaselineName = "baseline"

// baselineOwner names the artifact that owns the machinery this step delegates to, so
// the gap report points at something a reader can actually go and find.
const baselineOwner = "ISSUE-056"

// stepBaseline is a DELEGATION SEAM and nothing else (SPEC-069 REQ-012).
//
// The gitignored local baseline at `.backstop/baseline.json` is owned by ISSUE-056;
// this command designs and builds NONE of that machinery. NOTHING HERE WRITES
// `.backstop/baseline.json` OR COMPUTES A FINGERPRINT — the step calls the seam exactly
// once and reports what it returned.
//
// "NO SEEDER AVAILABLE" IS A VALUE, NOT A NIL FIELD. NewRunner is fail-closed, so a nil
// seam is unconstructable by design; a seeder with no machinery behind it returns
// ErrBaselineSeedingUnavailable instead, and this step maps that sentinel to
// capability-ABSENT and does NOT fail the run. An un-adopted capability is a missing
// benefit, not a broken promise.
//
// EVERY OTHER ERROR IS A STEP THAT FAILED TO DELIVER. The match is errors.Is, never a
// string compare: a seeder that fails for a real reason must not be silently absorbed
// into the capability-absent branch, which would turn a genuine failure into a
// reassuring line of report text.
func stepBaseline(projectRoot string, seeder BaselineSeeder) StepReport {
	path, err := seeder.Seed(projectRoot)

	if errors.Is(err, ErrBaselineSeedingUnavailable) {
		return StepReport{
			Step:    stepBaselineName,
			Outcome: OutcomeCapabilityAbsent,
			Detail: fmt.Sprintf("no local baseline was seeded: the seeding machinery does not exist yet and is owned by %s. "+
				"Nothing is broken and nothing is owed — the gate runs without it, you simply do not get a local ratchet until that lands", baselineOwner),
		}
	}
	if err != nil {
		return StepReport{
			Step:    stepBaselineName,
			Outcome: OutcomeBrokenPromise,
			Detail:  fmt.Sprintf("baseline seeding failed: %v", err),
		}
	}

	return StepReport{
		Step:    stepBaselineName,
		Outcome: OutcomeDelivered,
		Detail:  fmt.Sprintf("seeded the local baseline at %s", path),
	}
}
