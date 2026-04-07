package gate

import "context"

// StepBaselineComparisonFunc returns a StepFunc for the baseline comparison
// step. Currently deferred — always reports "skipped" with reason.
func StepBaselineComparisonFunc() StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{
			StepName:   StepBaselineComparison,
			Status:     "skipped",
			Violations: []Violation{},
			Reason:     "baseline not implemented",
		}
	}
}

// StepWaiverResolutionFunc returns a StepFunc for the waiver resolution
// step. Currently deferred — always reports "skipped" with reason.
func StepWaiverResolutionFunc() StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{
			StepName:   StepWaiverResolution,
			Status:     "skipped",
			Violations: []Violation{},
			Reason:     "waivers not implemented",
		}
	}
}

// StepLedgerIntegrityFunc returns a StepFunc for the ledger integrity
// step. Currently deferred — always reports "skipped" with reason.
func StepLedgerIntegrityFunc() StepFunc {
	return func(_ context.Context) StepResult {
		return StepResult{
			StepName:   StepLedgerIntegrity,
			Status:     "skipped",
			Violations: []Violation{},
			Reason:     "ledger not implemented",
		}
	}
}
