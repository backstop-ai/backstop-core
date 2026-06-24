package dogfoodenf

// contract_signature_present.go (SPEC-038 TASK-028): backstop's OWN previously-red
// contract_signature case — a present matching signature. Under the deleted brittle
// string-equality analyzer this round-tripped to RED ([]byte vs []uint8, named results);
// under the new ast-grep structural pack path it MATCHES → SATISFIED → GREEN (CLM-033,
// the dogfood payoff).
func RouteFile(path string, mode int) (string, error) { return path, nil }
