package covge

// Untested has real statements but no test exercises it, so the profile carries it
// with total>0/covered=0 — a genuine below-threshold gap, NOT a zero-statement N/A.
func Untested(n int) int {
	if n > 0 {
		return n
	}
	return -n
}
