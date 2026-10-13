package engine

import "fmt"

// maxExitStatus is the largest exit status a POSIX process can report.
const maxExitStatus = 255

// EffectiveSuccessExitCodes returns the engine's no-findings exit-status set: the
// declared SuccessExitCodes, or exactly {0} when undeclared (ISSUE-207 D-1/D-5).
// The returned slice is a copy, so a caller cannot mutate the binding through it.
func (b EngineBinding) EffectiveSuccessExitCodes() []int {
	if b.SuccessExitCodes == nil {
		return []int{0}
	}
	return append([]int(nil), b.SuccessExitCodes...)
}

// ValidateSuccessExitCodes is the single validation authority for a declared
// success_exit_codes set, shared by both manifest decode paths (pkg/pack
// parseEngineSpec and packval phase1-structural) so they cannot drift (ISSUE-207
// D-4). Nil means undeclared and is valid. A declared set must be non-empty, hold
// distinct values in 0..255, and contain 0: a set without 0 would turn every clean
// exit into an engine failure.
func ValidateSuccessExitCodes(codes []int) error {
	if codes == nil {
		return nil
	}
	if len(codes) == 0 {
		return fmt.Errorf("success_exit_codes must not be empty: omit the key for the default [0]")
	}
	seen := make(map[int]bool, len(codes))
	for _, code := range codes {
		if code < 0 || code > maxExitStatus {
			return fmt.Errorf("success_exit_codes value %d is out of range: exit statuses are 0..%d", code, maxExitStatus)
		}
		if seen[code] {
			return fmt.Errorf("success_exit_codes has duplicate value %d", code)
		}
		seen[code] = true
	}
	if !seen[0] {
		return fmt.Errorf("success_exit_codes %v must contain 0: without it a clean exit would be an engine failure", codes)
	}
	return nil
}
