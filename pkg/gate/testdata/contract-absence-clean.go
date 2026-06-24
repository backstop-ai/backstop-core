package contractfix

// contract-absence-clean.go (TASK-001): a real Go fixture from which the
// forbidden symbol is genuinely ABSENT, so the pack's grep absence probe yields
// an EMPTY result and the gate verdicts PASS (CLM-009). The forbidden token does
// NOT appear anywhere in this file (not even in this comment).
func cleanReplacement() string { return "ok" }
