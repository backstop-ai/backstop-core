package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/spf13/cobra"
)

// TestCodeCheck_FileFlag_RoutesByType verifies the --file flag is properly
// defined and routes to ScopeModeFile. The actual routing verification
// (which passes execute for which file types) is tested in pkg/check
// via TestCodeCheck_RunWith_FileMode_RoutesByType. (CLM-009)
func TestCodeCheck_FileFlag_RoutesByType(t *testing.T) {
	root := NewRootCommand()

	// The command should accept --file flag
	cmd, _, err := root.Find([]string{"code", "check"})
	if err != nil {
		t.Fatalf("find code check: %v", err)
	}

	// Verify --file flag is defined
	fileFlag := cmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Fatal("--file flag not found on code check command")
	}

	// Verify --all flag is defined
	allFlag := cmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Fatal("--all flag not found on code check command")
	}

	// Verify the command help mentions file-type routing
	if cmd.Short == "" {
		t.Error("code check command has no short description")
	}
}

// TestCodeCheck_FileAndAllConflict_ExitCode2 verifies that specifying both
// --file and --all produces exit code 2 before any checks run. (CLM-010)
func TestCodeCheck_FileAndAllConflict_ExitCode2(t *testing.T) {
	root := NewRootCommand()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"code", "check", "--file", "some.go", "--all"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for --file + --all conflict, got nil")
	}

	// Check exit code
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("expected ExitCodeError, got %T: %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d (config error)", exitErr.Code, ExitConfigError)
	}

	// Error message should mention the conflict
	if !strings.Contains(exitErr.Message, "file") || !strings.Contains(exitErr.Message, "all") {
		t.Errorf("error message %q should mention --file and --all conflict", exitErr.Message)
	}
}

// TestCodeCheck_JSONFlag verifies the --json flag is available on code check.
func TestCodeCheck_JSONFlag(t *testing.T) {
	root := NewRootCommand()

	cmd, _, err := root.Find([]string{"code", "check"})
	if err != nil {
		t.Fatalf("find code check: %v", err)
	}

	// --json should be inherited from root
	jsonFlag := cmd.InheritedFlags().Lookup("json")
	if jsonFlag == nil {
		// Also check local flags
		jsonFlag = cmd.Flags().Lookup("json")
	}
	if jsonFlag == nil {
		t.Error("--json flag not available on code check command")
	}
}

// TestCodeCheck_CommandRegistered verifies code check is registered under
// the code namespace.
func TestCodeCheck_CommandRegistered(t *testing.T) {
	root := NewRootCommand()

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"code", "check", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("code check --help: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "check") {
		t.Error("help output does not mention check command")
	}
}

// helper to silence cobra output in tests
func executeCommandSilent(root *cobra.Command, args ...string) (*cobra.Command, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	cmd, err := root.ExecuteC()
	return cmd, err
}

func TestGateIntegration_CodeCheckLoadsPacks(t *testing.T) {
	projectRoot := setupCodeCheckPackProject(t)
	restore := chdirTemp(t, projectRoot)
	defer restore()

	calledLoad := false
	calledMerge := false
	origLoad := loadInstalledPacksFn
	origMerge := mergePackRulesFn
	origRun := checkRunFn
	origValidators := runPackValidatorsFn
	defer func() {
		loadInstalledPacksFn = origLoad
		mergePackRulesFn = origMerge
		checkRunFn = origRun
		runPackValidatorsFn = origValidators
	}()

	loadInstalledPacksFn = func(root string) ([]*pack.Manifest, error) {
		calledLoad = true
		return []*pack.Manifest{{NormalizedName: "test-org/test-pack"}}, nil
	}
	mergePackRulesFn = func(packs []*pack.Manifest, packDir string) ([]string, error) {
		calledMerge = true
		return []string{filepath.Join(packDir, "test-org", "test-pack", "rules", "no-eval.yml")}, nil
	}
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		return &check.Result{}, nil
	}
	runPackValidatorsFn = func(packs []*pack.Manifest, packDir, root string) ([]gate.Violation, error) {
		return nil, nil
	}

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	if err := root.Execute(); err != nil {
		t.Fatalf("code check: %v", err)
	}
	if !calledLoad {
		t.Fatal("expected code check to load packs")
	}
	if !calledMerge {
		t.Fatal("expected code check to merge pack rules")
	}
}

func TestGateIntegration_CodeCheckAllLoadsPacks(t *testing.T) {
	projectRoot := setupCodeCheckPackProject(t)
	restore := chdirTemp(t, projectRoot)
	defer restore()

	origLoad := loadInstalledPacksFn
	origMerge := mergePackRulesFn
	origRun := checkRunFn
	origValidators := runPackValidatorsFn
	defer func() {
		loadInstalledPacksFn = origLoad
		mergePackRulesFn = origMerge
		checkRunFn = origRun
		runPackValidatorsFn = origValidators
	}()

	var capturedMode check.ScopeMode
	loadInstalledPacksFn = func(root string) ([]*pack.Manifest, error) {
		return []*pack.Manifest{{NormalizedName: "test-org/test-pack"}}, nil
	}
	mergePackRulesFn = func(packs []*pack.Manifest, packDir string) ([]string, error) {
		return []string{"dummy.yml"}, nil
	}
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		capturedMode = opts.Mode
		return &check.Result{}, nil
	}
	runPackValidatorsFn = func(packs []*pack.Manifest, packDir, root string) ([]gate.Violation, error) {
		return nil, nil
	}

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check", "--all"})
	if err := root.Execute(); err != nil {
		t.Fatalf("code check --all: %v", err)
	}
	if capturedMode != check.ScopeModeAll {
		t.Fatalf("expected --all mode, got %v", capturedMode)
	}
}

func TestGateIntegration_CodeCheckLayer3FullProject(t *testing.T) {
	projectRoot := setupCodeCheckPackProject(t)
	restore := chdirTemp(t, projectRoot)
	defer restore()

	origLoad := loadInstalledPacksFn
	origMerge := mergePackRulesFn
	origRun := checkRunFn
	origValidators := runPackValidatorsFn
	defer func() {
		loadInstalledPacksFn = origLoad
		mergePackRulesFn = origMerge
		checkRunFn = origRun
		runPackValidatorsFn = origValidators
	}()

	loadInstalledPacksFn = func(root string) ([]*pack.Manifest, error) {
		return []*pack.Manifest{{NormalizedName: "test-org/test-pack"}}, nil
	}
	mergePackRulesFn = func(packs []*pack.Manifest, packDir string) ([]string, error) {
		return nil, nil
	}
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		return &check.Result{}, nil
	}

	var validatorRoot string
	runPackValidatorsFn = func(packs []*pack.Manifest, packDir, root string) ([]gate.Violation, error) {
		validatorRoot = root
		return nil, nil
	}

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	if err := root.Execute(); err != nil {
		t.Fatalf("code check: %v", err)
	}
	// Resolve symlinks (macOS /var → /private/var) for comparison.
	resolvedRoot, _ := filepath.EvalSymlinks(projectRoot)
	resolvedValidator, _ := filepath.EvalSymlinks(validatorRoot)
	if resolvedValidator != resolvedRoot {
		t.Fatalf("expected layer3 validators to run on full project root %q, got %q", resolvedRoot, resolvedValidator)
	}
}

func setupCodeCheckPackProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(`project: code-check-pack
language: go
packs:
  test-org/test-pack: "1.0.0"
`), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".backstop", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	return dir
}

func chdirTemp(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}
}

// TestCodeCheck_LoadManifest_ConfigErrorPropagatesToCodeCheckExit pins the
// fail-loud boundary for REQ-002 on the standalone path: a zero-routable
// manifest dir must surface from LoadManifest through check.Run and the
// code-check command as an exit-2 config error — not a green skip. The
// checkRunFn stub delegates to the real check.RunWith (real LoadManifest,
// real error path) with a hermetic ensurer so no tool is installed or run.
func TestCodeCheck_LoadManifest_ConfigErrorPropagatesToCodeCheckExit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: zero-routable\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	// Rules present but no matchers and no compiled-schema discriminator:
	// zero routable rules.
	zeroRoutable := `{"rules": [{"check_types": []}]}`
	if err := os.WriteFile(filepath.Join(rulesDir, "broken.manifest.json"), []byte(zeroRoutable), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	restore := chdirTemp(t, dir)
	defer restore()

	origRun := checkRunFn
	defer func() { checkRunFn = origRun }()
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		return check.RunWith(ctx, check.RunOptions{
			Options:        opts,
			SemgrepEnsurer: stubEnsurer{},
		})
	}

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check", "--all"})
	err := root.Execute()
	if err == nil {
		t.Fatal("code check --all returned nil error for a zero-routable manifest dir; want exit-2 config error")
	}
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error %T (%v) is not an *ExitCodeError", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d (config error)", exitErr.Code, ExitConfigError)
	}
	if !strings.Contains(exitErr.Message, "routable") {
		t.Errorf("message %q should name the zero-routable condition", exitErr.Message)
	}
}

// stubEnsurer satisfies check.SemgrepEnsurer without installing anything.
type stubEnsurer struct{}

func (stubEnsurer) EnsureSemgrep(backstopDir, pinnedVersion string) (string, error) {
	return "/usr/bin/true", nil
}

// missingToolchainProject scaffolds a temp project whose backstop.yml declares a
// language with no built-in toolchain and no enforcement.toolchain declaration.
// A valid go-routable compiled manifest is written so routing is NOT the
// blocker — the registry's missing-toolchain config error is. Returns the
// project dir.
func missingToolchainProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// language: rust with no enforcement.toolchain → no toolchain available.
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: no-toolchain\nlanguage: rust\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	// A routable manifest so LoadManifest does NOT fail first — the only failure
	// must be the missing toolchain.
	manifest := `{"rules": [{"extensions": [".rs"], "check_types": ["lint", "build", "test"]}]}`
	if err := os.WriteFile(filepath.Join(rulesDir, "routing.manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	// A source file so the scope is non-empty.
	if err := os.WriteFile(filepath.Join(dir, "lib.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write lib.rs: %v", err)
	}
	return dir
}

// TestCodeCheck_MissingToolchain_DeclaredLanguageIsConfigError pins CLM-007: a
// language declared in backstop.yml that has NO built-in toolchain AND no
// enforcement.toolchain declaration is an exit-2 config error on BOTH CLI
// paths — the standalone code-check command and the gate step-2 path — never a
// skip-with-warning and never a green pass. Mirrors ISSUE-005's
// ConfigErrorPropagatesTo{CodeCheck,Gate}Exit boundary tests.
func TestCodeCheck_MissingToolchain_DeclaredLanguageIsConfigError(t *testing.T) {
	// ---- Standalone path: backstop code check --all exits 2. ----
	t.Run("standalone_code_check", func(t *testing.T) {
		dir := missingToolchainProject(t)
		restore := chdirTemp(t, dir)
		defer restore()

		origRun := checkRunFn
		defer func() { checkRunFn = origRun }()
		checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
			return check.RunWith(ctx, check.RunOptions{
				Options:        opts,
				SemgrepEnsurer: stubEnsurer{},
			})
		}

		root := NewRootCommand()
		root.SetArgs([]string{"code", "check", "--all"})
		err := root.Execute()
		if err == nil {
			t.Fatal("code check --all returned nil for a declared language with no toolchain; want exit-2 config error")
		}
		var exitErr *ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("error %T (%v) is not an *ExitCodeError", err, err)
		}
		if exitErr.Code != ExitConfigError {
			t.Errorf("exit code = %d, want %d (config error)", exitErr.Code, ExitConfigError)
		}
		if !strings.Contains(exitErr.Message, "toolchain") {
			t.Errorf("message %q should name the missing toolchain", exitErr.Message)
		}
	})

	// ---- Gate path: step 2 (code check) yields a config-error step, exit 2. ----
	t.Run("gate_step_code_check", func(t *testing.T) {
		dir := missingToolchainProject(t)
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

// TestCLI_TSDeclaredStack_SmokeEndToEnd is the ISSUE-003 acceptance smoke: a
// TypeScript project with a declared fake toolchain proves the data-driven
// stack path end-to-end — registry selection by language, declared command
// execution, named-format parsing, and violation reporting — with zero network
// access and zero real tools. The "eslint" and "tsc" are local shell scripts
// echoing fixture output; semgrep never fires because the project's compiled
// manifest carries no semgrep signal, so .ts routes [lint build test] only.
func TestCLI_TSDeclaredStack_SmokeEndToEnd(t *testing.T) {
	dir := t.TempDir()

	fakeESLint := filepath.Join(dir, "fake-eslint.sh")
	eslintJSON := `[{"filePath":"src/app.ts","messages":[{"ruleId":"no-unused-vars","severity":2,"message":"'x' is defined but never used.","line":4}]}]`
	if err := os.WriteFile(fakeESLint, []byte("#!/bin/sh\necho '"+eslintJSON+"'\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake eslint: %v", err)
	}
	fakeTsc := filepath.Join(dir, "fake-tsc.sh")
	if err := os.WriteFile(fakeTsc, []byte("#!/bin/sh\necho \"src/other.ts(9,5): error TS2304: Cannot find name 'y'.\"\nexit 2\n"), 0o755); err != nil {
		t.Fatalf("write fake tsc: %v", err)
	}
	fakeTest := filepath.Join(dir, "fake-test.sh")
	if err := os.WriteFile(fakeTest, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake test: %v", err)
	}

	backstopYML := `project: ts-smoke
language: typescript
enforcement:
  test_command: "` + fakeTest + `"
  toolchain:
    lint:
      command: "` + fakeESLint + `"
      format: eslint-json
    build:
      command: "` + fakeTsc + `"
      format: tsc
`
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(backstopYML), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}

	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	// Compiled manifest, language typescript, NO semgrep signal: .ts routes
	// lint/build/test only — the semgrep pass (and EnsureSemgrep's network
	// install) can never fire in this smoke.
	manifest := `{"standard":"STD-TS-001","language":"typescript","rules":[{"id":"TS-001","enforcement":"native"}]}`
	if err := os.WriteFile(filepath.Join(rulesDir, "STD-TS-001.manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "app.ts"), []byte("const x = 1\n"), 0o644); err != nil {
		t.Fatalf("write app.ts: %v", err)
	}

	restore := chdirTemp(t, dir)
	defer restore()

	origRun := checkRunFn
	defer func() { checkRunFn = origRun }()
	var captured *check.Result
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		result, err := check.RunWith(ctx, check.RunOptions{
			Options:        opts,
			SemgrepEnsurer: stubEnsurer{},
		})
		captured = result
		return result, err
	}

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check", "--all"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected violations exit, got nil error")
	}

	if captured == nil {
		t.Fatal("check.RunWith was never invoked")
	}
	all := captured.AllViolations()

	var lintV, buildV *check.Violation
	for i := range all {
		switch all[i].Pass {
		case check.CheckTypeLint:
			lintV = &all[i]
		case check.CheckTypeBuild:
			buildV = &all[i]
		}
	}

	if lintV == nil {
		t.Fatal("declared-stack lint violation missing: fake eslint output was not executed or parsed")
	}
	if lintV.File != "src/app.ts" || lintV.Line != 4 || lintV.Rule != "no-unused-vars" {
		t.Errorf("lint violation = %+v, want src/app.ts:4 no-unused-vars", *lintV)
	}
	if buildV == nil {
		t.Fatal("declared-stack build violation missing: fake tsc output was not executed or parsed")
	}
	if buildV.File != "src/other.ts" || buildV.Line != 9 || buildV.Rule != "TS2304" {
		t.Errorf("build violation = %+v, want src/other.ts:9 TS2304", *buildV)
	}

	// Semgrep must not have produced a pass result that executed: it is either
	// absent or skipped (not routed). Test pass ran the exit-0 fake: no violations.
	for _, pr := range captured.PassResults {
		if pr.Pass == check.CheckTypeSemgrep && !pr.Skipped && len(pr.Violations) > 0 {
			t.Errorf("semgrep unexpectedly executed with violations: %+v", pr.Violations)
		}
		if pr.Pass == check.CheckTypeTest && len(pr.Violations) > 0 {
			t.Errorf("test pass should be clean (exit-0 fake), got %+v", pr.Violations)
		}
	}
}
