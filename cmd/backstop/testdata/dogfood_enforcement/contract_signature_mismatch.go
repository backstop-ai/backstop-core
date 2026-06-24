package dogfoodenf

// contract_signature_mismatch.go (SPEC-038 TASK-028): an enforcement case — a
// missing/mismatched signature. Through the pack path this yields a real blocking
// contract violation (CLM-031): deletion did not trade enforcement for silence.
func unrelatedFn() {}
