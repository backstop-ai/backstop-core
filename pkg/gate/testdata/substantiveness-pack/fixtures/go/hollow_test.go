package fixtures_test

import "testing"

// TestHollowExample calls the subject under test but asserts nothing — hollow (Q1 RED).
// The subject call is gate.Build() (not gate.Run()): "Run" collides with the deleted
// analyzer's subtest-selector vocabulary, which would mask the hollow verdict and break
// strangler parity. Build matches neither the analyzer selectors nor the pack assertion
// regex, so both agree the test is hollow.
func TestHollowExample(t *testing.T) {
	gate.Build()
}
