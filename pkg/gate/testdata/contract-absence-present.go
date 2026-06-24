package contractfix

// contract-absence-present.go (TASK-001): a real Go fixture in which a FORBIDDEN
// symbol is PRESENT, so the pack's grep absence probe MATCHES and the gate
// inverts it to a blocking absence VIOLATION (CLM-008). The forbidden token
// "legacyProbeSymbol" appears here (even in a comment-adjacent identifier), the
// conservative text-presence direction grep enforces.
func legacyProbeSymbol() string { return "should have been deleted" }
