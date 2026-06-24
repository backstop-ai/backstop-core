package contractfix

// contract-sig-present.go (TASK-001): a real Go fixture whose declared symbol
// signature is PRESENT, so the pack's ast-grep signature rule MATCHES and the
// gate verdicts SATISFIED (CLM-004). Used by the Go pack rule tests and the
// strangler-equivalence pass on real ast-grep.

// RouteFile resolves a path to a routing decision. Its signature is the one the
// contract fixtures declare as PRESENT.
func RouteFile(path string, mode int) (string, error) {
	if path == "" {
		return "", nil
	}
	return path, nil
}
