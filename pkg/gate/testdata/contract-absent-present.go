package testdata

import "context"

// This fixture DECLARES the identifiers that an absence assertion will (wrongly)
// claim are gone. A {Absent:true} entry naming any of these against THIS file
// must FAIL — the symbol reappeared (regression caught).

// LintExecutor is a function the absence tests assert should be gone; its
// presence here makes an absence assertion against this file fail.
func LintExecutor(ctx context.Context) error {
	_ = ctx
	return nil
}

// BespokeExecutor is a type that an absence assertion (wrongly) claims is gone.
type BespokeExecutor struct {
	Name string
}

// GoBuiltinExecutors is a variable that an absence assertion (wrongly) claims is gone.
var GoBuiltinExecutors []string

// Probe is a method on BespokeExecutor used to exercise method-kind absence.
func (b *BespokeExecutor) Probe(ctx context.Context) error {
	_ = ctx
	return nil
}
