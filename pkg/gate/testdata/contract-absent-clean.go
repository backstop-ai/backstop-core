package testdata

// This fixture does NOT declare LintExecutor, BespokeExecutor, GoBuiltinExecutors,
// or Probe. A {Absent:true} entry naming any of those against THIS file must
// PASS — the deletion genuinely held. It contains one unrelated decl so the file
// parses as real source and is a non-empty package member.

// SurvivingSymbol is an unrelated declaration kept so this file parses as real
// source; it is not the target of any absence assertion.
type SurvivingSymbol struct {
	ID int
}
