package contractfix

// contract-sig-mismatch.go (TASK-001): a real Go fixture whose declared symbol
// is ABSENT/MISMATCHED relative to the declared contract signature (a different
// name and shape), so the pack's ast-grep signature rule produces NO match and
// the gate verdicts a blocking VIOLATION (CLM-005). The contract declares
// `func RouteFile(path string, mode int) (string, error)` but this file only
// defines an unrelated function — the signature is not present.
func somethingElse() {}
