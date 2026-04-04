package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// TestCLI_ExitCode_0_OnPass verifies ExitWithResult with passing result
// and no config error returns 0. (CLM-017)
func TestCLI_ExitCode_0_OnPass(t *testing.T) {
	result := validate.ValidationResult{}
	code := ExitWithResult(result, nil)
	if code != ExitPass {
		t.Errorf("ExitWithResult(pass, nil) = %d, want %d", code, ExitPass)
	}
}

// TestCLI_ExitCode_1_OnViolations verifies ExitWithResult with violations
// and no config error returns 1. (CLM-018)
func TestCLI_ExitCode_1_OnViolations(t *testing.T) {
	result := validate.ValidationResult{
		Violations: []validate.Violation{{Rule: "R001", Message: "violation1"}},
	}
	code := ExitWithResult(result, nil)
	if code != ExitViolations {
		t.Errorf("ExitWithResult(violations, nil) = %d, want %d", code, ExitViolations)
	}
}

// TestCLI_ExitCode_2_OnConfigError verifies ExitWithResult with config
// error returns 2. (CLM-019)
func TestCLI_ExitCode_2_OnConfigError(t *testing.T) {
	result := validate.ValidationResult{}
	code := ExitWithResult(result, errForTest("config error"))
	if code != ExitConfigError {
		t.Errorf("ExitWithResult(pass, err) = %d, want %d", code, ExitConfigError)
	}
}

// TestCLI_ExitCode_2_PrecedesViolations verifies ExitWithResult with both
// violations and config error returns 2 (precedence). (CLM-020)
func TestCLI_ExitCode_2_PrecedesViolations(t *testing.T) {
	result := validate.ValidationResult{
		Violations: []validate.Violation{{Rule: "R001", Message: "violation1"}},
	}
	code := ExitWithResult(result, errForTest("config error"))
	if code != ExitConfigError {
		t.Errorf("ExitWithResult(violations, err) = %d, want %d (config error takes precedence)", code, ExitConfigError)
	}
}

// TestCLI_ExitCode_OnlyValidCodes verifies ExitWithResult only ever returns
// 0, 1, or 2 across a range of inputs. (CLM-021)
func TestCLI_ExitCode_OnlyValidCodes(t *testing.T) {
	cases := []struct {
		name   string
		result validate.ValidationResult
		err    error
	}{
		{"pass no error", validate.ValidationResult{}, nil},
		{"violations no error", validate.ValidationResult{Violations: []validate.Violation{{Rule: "R001", Message: "v1"}}}, nil},
		{"pass with error", validate.ValidationResult{}, errForTest("err")},
		{"violations with error", validate.ValidationResult{Violations: []validate.Violation{{Rule: "R001", Message: "v1"}}}, errForTest("err")},
		{"empty result no error", validate.ValidationResult{}, nil},
		{"empty result with error", validate.ValidationResult{}, errForTest("err")},
	}

	validCodes := map[int]bool{ExitPass: true, ExitViolations: true, ExitConfigError: true}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := ExitWithResult(tc.result, tc.err)
			if !validCodes[code] {
				t.Errorf("ExitWithResult returned %d, want one of 0, 1, 2", code)
			}
		})
	}
}

// errForTest is a simple error type for testing.
type errForTest string

func (e errForTest) Error() string { return string(e) }
