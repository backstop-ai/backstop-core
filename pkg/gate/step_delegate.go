package gate

import (
	"context"
	"errors"
)

// ConfigError wraps an error to distinguish config errors (exit 2) from
// step failures (exit 1).
type ConfigError struct {
	Err error
}

func (e *ConfigError) Error() string { return e.Err.Error() }
func (e *ConfigError) Unwrap() error { return e.Err }

// ArtifactValidator abstracts the artifact validation logic for testability.
type ArtifactValidator interface {
	ValidateAll(ctx context.Context) ([]Violation, error)
}

// CodeChecker abstracts the code check logic for testability.
type CodeChecker interface {
	CheckAll(ctx context.Context) ([]Violation, error)
}

// ScopedCodeChecker can run code checks using the already-computed gate scope.
type ScopedCodeChecker interface {
	CheckScoped(ctx context.Context, scope *GateScope) ([]Violation, error)
}

// StepArtifactValidationFunc returns a StepFunc that delegates to an
// ArtifactValidator. Config errors from the validator are signaled via
// the ConfigErr flag on StepResult.
func StepArtifactValidationFunc(validator ArtifactValidator) StepFunc {
	return StepArtifactValidationScopedFunc(validator, nil)
}

// StepArtifactValidationScopedFunc filters delegated artifact findings to scope.
func StepArtifactValidationScopedFunc(validator ArtifactValidator, scope *GateScope) StepFunc {
	return func(ctx context.Context) StepResult {
		violations, err := validator.ValidateAll(ctx)
		if err != nil {
			result := StepResult{
				StepName:   StepArtifactValidation,
				Status:     "fail",
				Violations: []Violation{{Rule: "artifact_validation", Message: err.Error(), Severity: "error"}},
			}
			var cfgErr *ConfigError
			if errors.As(err, &cfgErr) {
				result.ConfigErr = true
			}
			return result
		}

		violations = filterViolations(scope, violations)
		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{
			StepName:   StepArtifactValidation,
			Status:     status,
			Violations: violations,
		}
	}
}

// StepCodeCheckFunc returns a StepFunc that delegates to a CodeChecker.
// Config errors from the checker are signaled via the ConfigErr flag.
func StepCodeCheckFunc(checker CodeChecker) StepFunc {
	return StepCodeCheckScopedFunc(checker, nil)
}

// StepCodeCheckScopedFunc filters delegated code findings to scope.
func StepCodeCheckScopedFunc(checker CodeChecker, scope *GateScope) StepFunc {
	return func(ctx context.Context) StepResult {
		if scope.Empty() {
			return StepResult{StepName: StepCodeCheck, Status: "pass", Violations: []Violation{}}
		}
		var violations []Violation
		var err error
		if scoped, ok := checker.(ScopedCodeChecker); ok {
			violations, err = scoped.CheckScoped(ctx, scope)
		} else {
			violations, err = checker.CheckAll(ctx)
		}
		if err != nil {
			result := StepResult{
				StepName:   StepCodeCheck,
				Status:     "fail",
				Violations: []Violation{{Rule: "code_check", Message: err.Error(), Severity: "error"}},
			}
			var cfgErr *ConfigError
			if errors.As(err, &cfgErr) {
				result.ConfigErr = true
			}
			return result
		}

		violations = filterViolations(scope, violations)
		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{
			StepName:   StepCodeCheck,
			Status:     status,
			Violations: violations,
		}
	}
}
