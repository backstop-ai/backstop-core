package contractse2e

// contract_signature_mismatch.go (SPEC-038 TASK-038): a real Go fixture whose declared
// contract signature (func RouteFile(...)) is ABSENT — only an unrelated function exists
// — so the installed contracts pack's ast-grep signature rule produces NO match and the
// production gate yields a real blocking contract VIOLATION (CLM-046).
func somethingElse() {}
