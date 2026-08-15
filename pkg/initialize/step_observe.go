package initialize

import (
	"fmt"
	"strings"
)

// stepObserveName is the report name for step 10.
const stepObserveName = "observe"

// stepObserve runs the gate ONCE and presents the result as OBSERVATION (SPEC-069
// REQ-013, REQ-014).
//
// FINDINGS ARE OBSERVATION, NOT FAILURE. Pre-existing findings are inherited from a
// project init just started governing; they are NEVER an init failure, and the report
// is phrased as WHAT WAS NOTICED with no verdict language in the summary. That rule
// governs the CLASSIFICATION of findings only — it does not make init exit 0 when a
// STEP failed to deliver what it promised, which is a separate question the runner's
// aggregation answers.
//
// FINDINGS ARE GROUPED BY GATE DIMENSION WITH A COUNT PER DIMENSION, so the report
// presents STRUCTURE rather than a wall of findings. Dimension names are backstop's own
// universal vocabulary, so the grouping introduces no tool or language noun.
//
// INIT CHANGES NO FILE UNDER pkg/gate, and it neither rewrites, suppresses nor
// substitutes for the remoteless `baseline_comparison` message. That message's
// self-consistency is pure gate machinery owned elsewhere, and papering over it here
// would hide a real gap behind an init-flavored summary.
func stepObserve(projectRoot string, gates GateRunner) (StepReport, []DimensionCount) {
	counts, err := gates.Run(projectRoot)
	if err != nil {
		return StepReport{
			Step:    stepObserveName,
			Outcome: OutcomeBrokenPromise,
			Detail:  fmt.Sprintf("the gate could not be run: %v", err),
		}, nil
	}

	if len(counts) == 0 {
		return StepReport{
			Step:    stepObserveName,
			Outcome: OutcomeDelivered,
			Detail:  "ran the gate once; it noticed nothing in any dimension",
		}, nil
	}

	lines := make([]string, 0, len(counts))
	total := 0
	for _, count := range counts {
		total += count.Count
		lines = append(lines, fmt.Sprintf("%s: %d", count.Dimension, count.Count))
	}

	if total == 0 {
		return StepReport{
			Step:    stepObserveName,
			Outcome: OutcomeDelivered,
			Detail:  fmt.Sprintf("ran the gate once across %d dimensions; it noticed nothing in any of them", len(counts)),
		}, counts
	}

	// PHRASED AS WHAT WAS NOTICED. No "failed", no "violation", no "error", no
	// "passed" — the summary is an observation about a project init has only just
	// started governing, and verdict language here would tell a consumer their brand
	// new project is already in trouble.
	return StepReport{
		Step:    stepObserveName,
		Outcome: OutcomeDelivered,
		Detail: fmt.Sprintf("ran the gate once and noticed %d finding(s), grouped by dimension — %s. These were already in your project; init does not treat them as something it broke",
			total, strings.Join(lines, ", ")),
	}, counts
}
