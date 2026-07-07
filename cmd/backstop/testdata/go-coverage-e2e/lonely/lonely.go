package lonely

// Marker carries no executable statements and its package has no test, so go test
// emits NO profile block for the package — a genuinely UNMEASURED file (distinct from
// a zero-statement file in a MEASURED package, which is N/A). It must STILL fire
// coverage_unmeasured, proving the fix did not blind the genuine-gap check.
type Marker struct {
	Tag string
}
