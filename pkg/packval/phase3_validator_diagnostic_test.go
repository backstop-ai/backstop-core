package packval

import (
	"errors"
	"strings"
	"testing"
)

// ISSUE-216 CLM-003: a failed validator run must be diagnosable. Before this, the
// three validator seams in RunFixtures reported a bare "layer3 positive failed" /
// "layer3 negative run failed" / "multi-file validator failed", so a sandbox denial
// (exit 126, "Operation not permitted") read exactly like a pack whose validator gave
// the wrong verdict. Each message now carries the exit status and the trimmed output.
// Branch conditions (validator polarity) are unchanged; see the VALIDATOR SEAM comment
// in phase3.go and TestPackVal_P3_ValidatorPolarityUnchangedByFindingsFix.

const validatorDenialOutput = "/bin/sh: validators/check.sh: Operation not permitted"

func validatorDiagnosticManifest(inputScope string) *PackManifest {
	return &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", Validator: "validators/check.sh", InputScope: inputScope,
		Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
			Positive: []FixtureRef{{Path: "fixtures/pos.txt"}},
			Negative: []FixtureRef{{Path: "fixtures/neg.txt"}},
		}}},
	}}}}}
}

func validatorErrorsWithCheck(errs []ValidationError, check string) []ValidationError {
	var out []ValidationError
	for _, e := range errs {
		if e.Check == check {
			out = append(out, e)
		}
	}
	return out
}

func TestPackVal_P3_ValidatorFailureMessageCarriesExitAndOutput(t *testing.T) {
	t.Run("positive fixture denial carries exit 126 and the denial output", func(t *testing.T) {
		mock := &MockExecutor{
			ValidatorFn: func(_, _ string, paths []string) (ExecutionResult, error) {
				if len(paths) == 1 && paths[0] == "fixtures/pos.txt" {
					return ExecutionResult{Passed: false, ExitCode: 126, Output: validatorDenialOutput}, nil
				}
				return ExecutionResult{Passed: false, ExitCode: 1}, nil
			},
		}
		r := RunFixtures(validatorDiagnosticManifest("single-file"), t.TempDir(), mock)
		got := validatorErrorsWithCheck(r.Errors, "validator-positive")
		if len(got) != 1 {
			t.Fatalf("want exactly one validator-positive error; got %+v", r.Errors)
		}
		msg := got[0].Message
		if !strings.Contains(msg, "layer3 positive failed") {
			t.Errorf("message lost its seam identity %q: %q", "layer3 positive failed", msg)
		}
		if !strings.Contains(msg, "126") {
			t.Errorf("message must carry the validator exit status 126: %q", msg)
		}
		if !strings.Contains(msg, "Operation not permitted") {
			t.Errorf("message must carry the validator output: %q", msg)
		}
		if r.Status != "fail" {
			t.Errorf("a failing positive validator run must still fail phase 3; got %q", r.Status)
		}
	})

	t.Run("negative fixture run failure carries the exit code and the output", func(t *testing.T) {
		mock := &MockExecutor{
			ValidatorFn: func(_, _ string, paths []string) (ExecutionResult, error) {
				if len(paths) == 1 && paths[0] == "fixtures/neg.txt" {
					return ExecutionResult{Passed: false, ExitCode: 71, Output: "sandbox_apply: Operation not permitted"},
						errors.New("sandboxed run failed: exit status 71")
				}
				return ExecutionResult{Passed: true, ExitCode: 0}, nil
			},
		}
		r := RunFixtures(validatorDiagnosticManifest("single-file"), t.TempDir(), mock)
		got := validatorErrorsWithCheck(r.Errors, "validator-negative")
		if len(got) != 1 {
			t.Fatalf("want exactly one validator-negative error; got %+v", r.Errors)
		}
		msg := got[0].Message
		if !strings.Contains(msg, "layer3 negative run failed") {
			t.Errorf("message lost its seam identity %q: %q", "layer3 negative run failed", msg)
		}
		if !strings.Contains(msg, "71") {
			t.Errorf("message must carry the validator exit status 71: %q", msg)
		}
		if !strings.Contains(msg, "sandbox_apply: Operation not permitted") {
			t.Errorf("message must carry the validator output: %q", msg)
		}
		if len(validatorErrorsWithCheck(r.Errors, "validator-positive")) != 0 {
			t.Errorf("a positive that exits zero must not fail; got %+v", r.Errors)
		}
	})

	t.Run("output longer than 1024 bytes is cut to its last 1024 bytes", func(t *testing.T) {
		long := "HEADMARK" + strings.Repeat("x", 2000) + "TAILMARK"
		mock := &MockExecutor{
			ValidatorFn: func(_, _ string, paths []string) (ExecutionResult, error) {
				if len(paths) == 1 && paths[0] == "fixtures/pos.txt" {
					return ExecutionResult{Passed: false, ExitCode: 126, Output: long}, nil
				}
				return ExecutionResult{Passed: false, ExitCode: 1}, nil
			},
		}
		r := RunFixtures(validatorDiagnosticManifest("single-file"), t.TempDir(), mock)
		got := validatorErrorsWithCheck(r.Errors, "validator-positive")
		if len(got) != 1 {
			t.Fatalf("want exactly one validator-positive error; got %+v", r.Errors)
		}
		msg := got[0].Message
		tail := long[len(long)-1024:]
		if !strings.Contains(msg, tail) {
			t.Errorf("message must carry the last 1024 bytes of the output")
		}
		if strings.Contains(msg, long[len(long)-1025:]) {
			t.Errorf("message carries more than the last 1024 bytes of the output")
		}
		if strings.Contains(msg, "HEADMARK") {
			t.Errorf("message carries the head of an over-long output: %q", msg)
		}
	})

	t.Run("multi-file seam denial carries exit 126 and the denial output", func(t *testing.T) {
		mock := &MockExecutor{
			ValidatorFn: func(_, _ string, paths []string) (ExecutionResult, error) {
				if len(paths) == 2 {
					return ExecutionResult{Passed: false, ExitCode: 126, Output: validatorDenialOutput}, nil
				}
				// Single-file runs at correct polarity so only the multi-file seam fails.
				if paths[0] == "fixtures/neg.txt" {
					return ExecutionResult{Passed: false, ExitCode: 1}, nil
				}
				return ExecutionResult{Passed: true, ExitCode: 0}, nil
			},
		}
		r := RunFixtures(validatorDiagnosticManifest("multi-file"), t.TempDir(), mock)
		got := validatorErrorsWithCheck(r.Errors, "validator-multi-file")
		if len(got) != 1 {
			t.Fatalf("want exactly one validator-multi-file error; got %+v", r.Errors)
		}
		msg := got[0].Message
		if !strings.Contains(msg, "multi-file validator failed") {
			t.Errorf("message lost its seam identity %q: %q", "multi-file validator failed", msg)
		}
		if !strings.Contains(msg, "126") {
			t.Errorf("message must carry the validator exit status 126: %q", msg)
		}
		if !strings.Contains(msg, "Operation not permitted") {
			t.Errorf("message must carry the validator output: %q", msg)
		}
		if len(r.Errors) != 1 {
			t.Errorf("only the multi-file seam should fail; got %+v", r.Errors)
		}
	})
}
