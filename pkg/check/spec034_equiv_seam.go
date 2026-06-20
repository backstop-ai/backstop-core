package check

// SPEC-034 phase-2 equivalence-gate test seams. These exported wrappers expose
// the STILL-PRESENT bespoke Go toolchain parsers so the cmd/backstop equivalence
// gate (cmd/backstop/equivalence_test.go) can compare the bespoke normalization
// against the new go-toolchain engine path on the SAME captured tool output, in
// ONE working tree, BEFORE any deletion (REQ-009). They mirror the established
// ParseSemgrepJSONForTest precedent.
//
// They are deliberately isolated in this dedicated file (not check.go) so the
// equivalence work touches no other pkg/check source, and they RETIRE in the
// phase-3 deletion alongside parseGoBuildErrors / parseGoTestFailures /
// parseGolangciJSON (REQ-003/REQ-005/REQ-011) — this whole file is deleted with
// the bespoke parsers it wraps.

// ParseGoBuildErrorsForTest exposes parseGoBuildErrors for the build equivalence
// gate (CLM-008/CLM-030).
func ParseGoBuildErrorsForTest(out []byte) []Violation {
	return parseGoBuildErrors(out)
}

// ParseGoTestFailuresForTest exposes parseGoTestFailures for the test
// equivalence gate (CLM-009/CLM-031).
func ParseGoTestFailuresForTest(out []byte) []Violation {
	return parseGoTestFailures(out)
}

// ParseGolangciJSONForTest exposes parseGolangciJSON (golangci-lint v1 JSON) for
// the lint equivalence gate, compared against the engine path's v2 SARIF ->
// parseSarif on the equivalent findings (CLM-017/CLM-032).
func ParseGolangciJSONForTest(out []byte) ([]Violation, error) {
	return parseGolangciJSON(out)
}
