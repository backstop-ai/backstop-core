package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
)

// SPEC-039 REQ-010 PRESERVATION tests. These guard that deleting the
// manifest-path zero-routable ConfigError EMISSION does NOT remove the
// ConfigError TYPE or any .manifest.json-INDEPENDENT trigger, and that the
// .standard.md scaffolder is out of scope. The MissingToolchain assertion holds
// BEFORE and AFTER the deletion: it drives the exit-2 via the registry path
// (resolveToolchain/validateToolchainKeys), never via a .manifest.json fixture.

// missingToolchainProjectNoManifest scaffolds a temp project whose backstop.yml
// declares a language (rust) with no built-in toolchain and no
// enforcement.toolchain declaration, and a single source file so scope is
// non-empty. Deliberately writes NO .manifest.json — the missing-toolchain
// ConfigError must come from the registry path alone.
func missingToolchainProjectNoManifest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: no-toolchain\nlanguage: rust\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	// .backstop/ must exist (ValidateBackstopDir) but carries NO .manifest.json —
	// the missing-toolchain ConfigError must come from the registry path alone.
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write lib.rs: %v", err)
	}
	return dir
}

// TestCodeCheck_MissingToolchain_StillConfigErrorAfterManifestBranchRemoval pins
// CLM-020: a declared language with no toolchain still emits a *ConfigError and
// exits 2 on BOTH the standalone code-check and gate paths, with NO
// .manifest.json present — proving the trigger is the registry path
// (resolveToolchain), independent of the deleted manifest reader.
func TestCodeCheck_MissingToolchain_StillConfigErrorAfterManifestBranchRemoval(t *testing.T) {
	// ---- Standalone path: backstop code check --all exits 2. ----
	t.Run("standalone_code_check", func(t *testing.T) {
		dir := missingToolchainProjectNoManifest(t)
		restore := chdirTemp(t, dir)
		defer restore()

		origRun := checkRunFn
		defer func() { checkRunFn = origRun }()
		checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
			return check.RunWith(ctx, check.RunOptions{Options: opts})
		}

		root := NewRootCommand()
		root.SetArgs([]string{"code", "check", "--all"})
		err := root.Execute()
		if err == nil {
			t.Fatal("code check --all returned nil for a declared language with no toolchain (no .manifest.json); want exit-2 config error")
		}
		var exitErr *ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("error %T (%v) is not an *ExitCodeError", err, err)
		}
		if exitErr.Code != ExitConfigError {
			t.Errorf("exit code = %d, want %d (config error)", exitErr.Code, ExitConfigError)
		}
		if !strings.Contains(exitErr.Message, "toolchain") {
			t.Errorf("message %q should name the missing toolchain (registry-path trigger)", exitErr.Message)
		}
	})

	// ---- Gate path: step 2 (code check) yields a config-error step, exit 2. ----
	t.Run("gate_step_code_check", func(t *testing.T) {
		dir := missingToolchainProjectNoManifest(t)
		restore := chdirTemp(t, dir)
		defer restore()

		checker := &realCodeChecker{projectRoot: dir}
		scope := &gate.GateScope{Mode: gate.GateScopeModeAll}
		step := gate.StepCodeCheckScopedFunc(checker, scope)
		g := gate.New(gate.WithSteps([]gate.StepFunc{step}), gate.WithScope(scope))

		result, exitCode := g.Run(context.Background())
		if exitCode != 2 {
			t.Fatalf("gate exit code = %d, want 2 (config error)", exitCode)
		}
		if len(result.Steps) != 1 {
			t.Fatalf("got %d steps, want 1", len(result.Steps))
		}
		if !result.Steps[0].ConfigErr {
			t.Error("step ConfigErr = false, want true (missing toolchain is a config error)")
		}
		if result.Pass {
			t.Error("gate Pass = true; a missing toolchain must never read as green")
		}
	})
}

// TestStandardScaffolder_Untouched pins CLM-018: the .standard.md scaffolder
// (pkg/pack/scaffold.go) is out of scope for SPEC-039 — it still exists and this
// change does not reference or remove it (scope fence for ISSUE-030).
func TestStandardScaffolder_Untouched(t *testing.T) {
	// The scaffolder file lives at <repo>/pkg/pack/scaffold.go. From the
	// cmd/backstop package dir (CWD during tests), that is ../../pkg/pack.
	p := filepath.Join("..", "..", "pkg", "pack", "scaffold.go")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("pkg/pack/scaffold.go must remain (ISSUE-030 scope fence); stat error: %v", err)
	}
}
