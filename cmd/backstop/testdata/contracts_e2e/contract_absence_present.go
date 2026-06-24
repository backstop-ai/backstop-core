package contractse2e

// contract_absence_present.go (SPEC-038 TASK-038): a real Go fixture in which a FORBIDDEN
// symbol is PRESENT, so the installed contracts pack's grep absence probe matches and the
// production gate yields a real blocking absence VIOLATION (CLM-047).
func legacyProbeSymbol() string { return "present" }
