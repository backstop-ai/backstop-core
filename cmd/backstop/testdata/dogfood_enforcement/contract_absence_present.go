package dogfoodenf

// contract_absence_present.go (SPEC-038 TASK-028): an enforcement case — a present
// forbidden symbol. Through the pack grep absence path this yields a real blocking
// absence violation (CLM-032).
func legacyProbeSymbol() string { return "present" }
