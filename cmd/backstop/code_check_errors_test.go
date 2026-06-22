package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// stubCodeCheckFns swaps the injectable code-check function vars for the
// duration of a test and restores them afterward. After SPEC-031 the two
// symmetric pack feeders (mergePackRules + runPackValidators) collapse into the
// single dispatchPackEngines seam, so the stub takes one dispatch override that
// stands in for both the findings feeder and the sandbox validator feeder.
func stubCodeCheckFns(t *testing.T,
	load func(string) ([]*pack.Manifest, error),
	run func(context.Context, check.Options) (*check.Result, error),
	dispatch func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error),
) {
	t.Helper()
	origLoad := loadInstalledPacksFn
	origRun := checkRunFn
	origDispatch := dispatchPackEnginesFn
	t.Cleanup(func() {
		loadInstalledPacksFn = origLoad
		checkRunFn = origRun
		dispatchPackEnginesFn = origDispatch
	})
	loadInstalledPacksFn = load
	checkRunFn = run
	dispatchPackEnginesFn = dispatch
}

// asExitCodeError extracts an *ExitCodeError from err, failing the test if it is
// not one.
func asExitCodeError(t *testing.T, err error) *ExitCodeError {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var ece *ExitCodeError
	if !errors.As(err, &ece) {
		t.Fatalf("expected *ExitCodeError, got %T: %v", err, err)
	}
	return ece
}

// TestCodeCheck_PackLoadingError_ExitConfig verifies that a failure loading
// installed packs surfaces as an ExitConfigError naming the pack-loading stage.
func TestCodeCheck_PackLoadingError_ExitConfig(t *testing.T) {
	dir := setupCodeCheckPackProject(t)
	defer chdirTemp(t, dir)()

	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) { return nil, errors.New("boom loading packs") },
		func(context.Context, check.Options) (*check.Result, error) { return &check.Result{}, nil },
		func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
			return nil, nil
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	ece := asExitCodeError(t, root.Execute())
	if ece.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d", ece.Code, ExitConfigError)
	}
	if !strings.Contains(ece.Message, "pack loading") {
		t.Errorf("message = %q, want it to mention pack loading", ece.Message)
	}
}

// TestCodeCheck_DispatchError_ExitConfig verifies that a failure in the
// group-by-engine dispatch (reached only when packs are present) surfaces as an
// ExitConfigError naming the pack-engines stage. Re-keyed from the retired
// TestCodeCheck_MergeRulesError_ExitConfig: the separate merge feeder is gone,
// so a broken-pack/unknown-engine error now propagates from dispatchPackEngines.
func TestCodeCheck_DispatchError_ExitConfig(t *testing.T) {
	dir := setupCodeCheckPackProject(t)
	defer chdirTemp(t, dir)()

	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) {
			return []*pack.Manifest{{NormalizedName: "org/pack"}}, nil
		},
		func(context.Context, check.Options) (*check.Result, error) { return &check.Result{}, nil },
		func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
			return nil, errors.New("dispatch failed")
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	ece := asExitCodeError(t, root.Execute())
	if ece.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d", ece.Code, ExitConfigError)
	}
	if !strings.Contains(ece.Message, "pack engines") {
		t.Errorf("message = %q, want it to mention pack engines", ece.Message)
	}
}

// TestCodeCheck_CheckRunConfigError_PropagatesConfigError verifies that a
// *check.ConfigError returned by the check engine is translated to an
// ExitConfigError carrying the engine's message.
func TestCodeCheck_CheckRunConfigError_PropagatesConfigError(t *testing.T) {
	dir := setupCodeCheckPackProject(t)
	defer chdirTemp(t, dir)()

	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) { return nil, nil },
		func(context.Context, check.Options) (*check.Result, error) {
			return nil, &check.ConfigError{Message: "manifest has zero routable rules"}
		},
		func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
			return nil, nil
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	ece := asExitCodeError(t, root.Execute())
	if ece.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d", ece.Code, ExitConfigError)
	}
	if !strings.Contains(ece.Message, "manifest has zero routable rules") {
		t.Errorf("message = %q, want the engine config-error message", ece.Message)
	}
}

// TestCodeCheck_CheckRunGenericError_ExitConfig verifies that a non-ConfigError
// returned by the check engine still surfaces as an ExitConfigError carrying
// the underlying message.
func TestCodeCheck_CheckRunGenericError_ExitConfig(t *testing.T) {
	dir := setupCodeCheckPackProject(t)
	defer chdirTemp(t, dir)()

	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) { return nil, nil },
		func(context.Context, check.Options) (*check.Result, error) {
			return nil, errors.New("toolchain exploded")
		},
		func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
			return nil, nil
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	ece := asExitCodeError(t, root.Execute())
	if ece.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d", ece.Code, ExitConfigError)
	}
	if !strings.Contains(ece.Message, "toolchain exploded") {
		t.Errorf("message = %q, want the underlying error", ece.Message)
	}
}

// TestCodeCheck_PackEngineError_ExitConfig verifies that a pack-engine dispatch
// failure (only reached when packs are present) surfaces as an ExitConfigError
// naming the pack-engines stage. Re-keyed from
// TestCodeCheck_PackValidatorError_ExitConfig: the sandbox validator feeder is
// folded into dispatchPackEngines.
func TestCodeCheck_PackEngineError_ExitConfig(t *testing.T) {
	dir := setupCodeCheckPackProject(t)
	defer chdirTemp(t, dir)()

	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) {
			return []*pack.Manifest{{NormalizedName: "org/pack"}}, nil
		},
		func(context.Context, check.Options) (*check.Result, error) { return &check.Result{}, nil },
		func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
			return nil, errors.New("validator crashed")
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	ece := asExitCodeError(t, root.Execute())
	if ece.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d", ece.Code, ExitConfigError)
	}
	if !strings.Contains(ece.Message, "pack engines") {
		t.Errorf("message = %q, want it to mention pack engines", ece.Message)
	}
}

// TestCodeCheck_PackEngineViolations_BecomeExitViolations verifies that
// pack-engine dispatch violations are appended to the result and drive a
// non-zero exit code, proving the violations actually flow through to the final
// status. Re-keyed from TestCodeCheck_PackValidatorViolations_BecomeExitViolations.
func TestCodeCheck_PackEngineViolations_BecomeExitViolations(t *testing.T) {
	dir := setupCodeCheckPackProject(t)
	defer chdirTemp(t, dir)()

	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) {
			return []*pack.Manifest{{NormalizedName: "org/pack"}}, nil
		},
		func(context.Context, check.Options) (*check.Result, error) { return &check.Result{}, nil },
		func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
			return []gate.Violation{
				{Rule: "no-eval", File: "danger.go", Message: "eval is forbidden", Severity: "error"},
			}, nil
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	err := root.Execute()
	ece := asExitCodeError(t, err)
	if ece.Code == 0 {
		t.Fatalf("expected a non-zero exit code from pack-validator violations, got %d", ece.Code)
	}
	if !strings.Contains(ece.Message, "violation") {
		t.Errorf("message = %q, want it to mention violation(s)", ece.Message)
	}
}

// TestCodeCheck_FileMode_SetsTimeout verifies that --file mode selects
// ScopeModeFile and applies the 2-second hook-dispatch timeout. (The config
// semgrep_version pin was removed with the in-process semgrep pass, ISSUE-018.)
func TestCodeCheck_FileMode_SetsTimeout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(
		"project: p\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}
	defer chdirTemp(t, dir)()

	var captured check.Options
	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) { return nil, nil },
		func(_ context.Context, opts check.Options) (*check.Result, error) {
			captured = opts
			return &check.Result{}, nil
		},
		func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
			return nil, nil
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check", "--file", "main.go"})
	if err := root.Execute(); err != nil {
		t.Fatalf("code check --file: %v", err)
	}
	if captured.Mode != check.ScopeModeFile {
		t.Errorf("mode = %v, want ScopeModeFile", captured.Mode)
	}
	if captured.FilePath != "main.go" {
		t.Errorf("FilePath = %q, want main.go", captured.FilePath)
	}
	if captured.Timeout != 2*time.Second {
		t.Errorf("Timeout = %v, want 2s", captured.Timeout)
	}
}

// TestCodeCheck_MissingBackstopDir_ExitConfig verifies that the absence of a
// .backstop directory surfaces as an ExitConfigError before any checks run.
func TestCodeCheck_MissingBackstopDir_ExitConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(
		"project: p\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Deliberately no .backstop directory.
	defer chdirTemp(t, dir)()

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	ece := asExitCodeError(t, root.Execute())
	if ece.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d", ece.Code, ExitConfigError)
	}
	if !strings.Contains(ece.Message, ".backstop") {
		t.Errorf("message = %q, want it to mention the missing .backstop dir", ece.Message)
	}
}
