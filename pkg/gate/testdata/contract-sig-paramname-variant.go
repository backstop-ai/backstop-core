package contractfix

// contract-sig-paramname-variant.go (TASK-001): a real Go fixture whose function
// has the SAME param TYPES and return as the declared contract but DIFFERENT
// parameter NAMES and whitespace, so the ast-grep STRUCTURAL pattern still
// MATCHES (CLM-007) where the deleted string-equality compare would have failed.
// Declared contract: `func RouteFile(path string, mode int) (string, error)`.
func RouteFile(p string, m int) (string, error) {
	return p, nil
}
