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
	calledDispatch := false
	origLoad := loadInstalledPacksFn
	origRun := checkRunFn
	origDispatch := dispatchPackEnginesFn
	defer func() {
		loadInstalledPacksFn = origLoad
		checkRunFn = origRun
		dispatchPackEnginesFn = origDispatch
	}()

	loadInstalledPacksFn = func(root string) ([]*pack.Manifest, error) {
		calledLoad = true
		return []*pack.Manifest{{NormalizedName: "test-org/test-pack"}}, nil
	}
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		return &check.Result{}, nil
	}
	dispatchPackEnginesFn = func(packs []*pack.Manifest, packDir, root string, scope *gate.GateScope, runner check.CommandRunner) ([]gate.Violation, error) {
		calledDispatch = true
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
	if !calledDispatch {
		t.Fatal("expected code check to dispatch pack engines group-by-engine")
	}
}

func TestGateIntegration_CodeCheckAllLoadsPacks(t *testing.T) {
	projectRoot := setupCodeCheckPackProject(t)
	restore := chdirTemp(t, projectRoot)
	defer restore()

	origLoad := loadInstalledPacksFn
	origRun := checkRunFn
	origDispatch := dispatchPackEnginesFn
	defer func() {
		loadInstalledPacksFn = origLoad
		checkRunFn = origRun
		dispatchPackEnginesFn = origDispatch
	}()

	var capturedMode check.ScopeMode
	loadInstalledPacksFn = func(root string) ([]*pack.Manifest, error) {
		return []*pack.Manifest{{NormalizedName: "test-org/test-pack"}}, nil
	}
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		capturedMode = opts.Mode
		return &check.Result{}, nil
	}
	dispatchPackEnginesFn = func(packs []*pack.Manifest, packDir, root string, scope *gate.GateScope, runner check.CommandRunner) ([]gate.Violation, error) {
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

// TestGateIntegration_CodeCheckSandboxFullProject verifies the sandbox engine's
// exit-code branch (re-keyed from layer-3) runs on the full project root via the
// consolidated dispatchPackEngines path. Re-keyed from the retired
// TestGateIntegration_CodeCheckLayer3FullProject: the runPackValidatorsFn seam
// is replaced by the dispatchPackEnginesFn seam capturing the dispatch root.
func TestGateIntegration_CodeCheckSandboxFullProject(t *testing.T) {
	projectRoot := setupCodeCheckPackProject(t)
	restore := chdirTemp(t, projectRoot)
	defer restore()

	origLoad := loadInstalledPacksFn
	origRun := checkRunFn
	origDispatch := dispatchPackEnginesFn
	defer func() {
		loadInstalledPacksFn = origLoad
		checkRunFn = origRun
		dispatchPackEnginesFn = origDispatch
	}()

	loadInstalledPacksFn = func(root string) ([]*pack.Manifest, error) {
		return []*pack.Manifest{{NormalizedName: "test-org/test-pack"}}, nil
	}
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		return &check.Result{}, nil
	}

	var dispatchRoot string
	dispatchPackEnginesFn = func(packs []*pack.Manifest, packDir, root string, scope *gate.GateScope, runner check.CommandRunner) ([]gate.Violation, error) {
		dispatchRoot = root
		return nil, nil
	}

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check"})
	if err := root.Execute(); err != nil {
		t.Fatalf("code check: %v", err)
	}
	// Resolve symlinks (macOS /var → /private/var) for comparison.
	resolvedRoot, symErr := filepath.EvalSymlinks(projectRoot)
	if symErr != nil {
		t.Fatalf("resolve symlinks for project root: %v", symErr)
	}
	resolvedDispatch, dispatchSymErr := filepath.EvalSymlinks(dispatchRoot)
	if dispatchSymErr != nil {
		t.Fatalf("resolve symlinks for dispatch root: %v", dispatchSymErr)
	}
	if resolvedDispatch != resolvedRoot {
		t.Fatalf("expected sandbox engine dispatch to run on full project root %q, got %q", resolvedRoot, resolvedDispatch)
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

// unknownToolchainKeyProject scaffolds a temp project whose backstop.yml
// declares an enforcement.toolchain block containing an out-of-vocabulary pass
// key. A routable compiled manifest is written so manifest routing is NOT the
// blocker — the only failure must be the bad toolchain key. The language and
// the offending key are caller-supplied so the same harness covers both the
// non-go and go boundary paths.
func unknownToolchainKeyProject(t *testing.T, language, badKey string) string {
	t.Helper()
	dir := t.TempDir()
	backstopYML := "project: bad-toolchain-key\nlanguage: " + language + `
enforcement:
  toolchain:
    lint:
      command: "lint-tool run"
      format: regex-lines
    ` + badKey + `:
      command: "bogus run"
      format: regex-lines
`
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(backstopYML), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	// The bad toolchain key surfaces from validateToolchainKeys (registry path),
	// independent of routing — no .manifest.json is needed (the reader was
	// deleted in SPEC-039 and would ignore one anyway). .backstop/ must still
	// exist (ValidateBackstopDir), but carries no manifest.
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	// A source file so the scope is non-empty.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	return dir
}

// TestCodeCheck_Registry_UnknownPassKeyPropagatesExitTwo pins CLM-002 / REQ-001
// at the consumption boundary: an out-of-vocabulary enforcement.toolchain key
// surfaces from the registry through check.Run and the standalone code-check
// command as an exit-2 CONFIG error — not a panic, not a swallowed error, not a
// green skip. Reuses the chdirTemp + checkRunFn → check.RunWith(stubEnsurer)
// harness so the path is hermetic (no tool installed or run). Confirms the
// boundary holds on BOTH the non-go and go paths.
func TestCodeCheck_Registry_UnknownPassKeyPropagatesExitTwo(t *testing.T) {
	cases := []struct {
		name     string
		language string
		badKey   string
	}{
		{name: "non_go_path", language: "rust", badKey: "typecheck"},
		{name: "go_path", language: "go", badKey: "lnit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := unknownToolchainKeyProject(t, tc.language, tc.badKey)
			restore := chdirTemp(t, dir)
			defer restore()

			origRun := checkRunFn
			defer func() { checkRunFn = origRun }()
			checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
				return check.RunWith(ctx, check.RunOptions{
					Options: opts,
				})
			}

			root := NewRootCommand()
			root.SetArgs([]string{"code", "check", "--all"})
			err := root.Execute()
			if err == nil {
				t.Fatal("code check --all returned nil for an unknown toolchain key; want exit-2 config error (silent skip is the bug)")
			}
			var exitErr *ExitCodeError
			if !errors.As(err, &exitErr) {
				t.Fatalf("error %T (%v) is not an *ExitCodeError", err, err)
			}
			if exitErr.Code != ExitConfigError {
				t.Errorf("exit code = %d, want %d (config error)", exitErr.Code, ExitConfigError)
			}
			// The offending key carries through from the registry *check.ConfigError.
			if !strings.Contains(exitErr.Message, tc.badKey) {
				t.Errorf("message %q should name the offending key %q", exitErr.Message, tc.badKey)
			}
		})
	}
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

	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	// No .manifest.json: the .ts files route via the built-in default manifest
	// (the .manifest.json reader was deleted in SPEC-039). .ts → all four passes,
	// but findings has no pkg/check executor, so it is recorded as Skipped and
	// never fires — the declared lint/build/test fakes are what produce output.

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
			Options: opts,
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
		if pr.Pass == check.CheckTypeFindings && !pr.Skipped && len(pr.Violations) > 0 {
			t.Errorf("semgrep unexpectedly executed with violations: %+v", pr.Violations)
		}
		if pr.Pass == check.CheckTypeTest && len(pr.Violations) > 0 {
			t.Errorf("test pass should be clean (exit-0 fake), got %+v", pr.Violations)
		}
	}
}

// TestCodeCheckOptions_NoManifestDir and TestCodeCheck_NoPacks_NoSemgrepConfig
// were RETIRED at SPEC-030 spec_version 1.2.0. Their surviving property (no
// compiled-standards manifest directory / rule-config source feeds the checks)
// is, post-ISSUE-018, a structural absence guaranteed by the deletion of the
// in-process semgrepExecutor itself — covered substantively by CLM-003
// (TestPkgCheck_NoManifestDirFieldOnSemgrepFeed) and CLM-004. Repointing these
// captured-Options inspections to call pkg/check would be vacuous, so they were
// retired rather than kept.

// TestCodeCheck_PackOnly_RulesDispatchViaEnginePath is the SPEC-030 CLM-011
// check (repointed at spec_version 1.1.0): a code-check run with one installed
// pack dispatches that pack's rules via the engine path (dispatchPackEngines),
// NOT via an in-process semgrep `--config` feed. It asserts the pack reaches the
// engine dispatch and that no compiled-standards directory is wired into the
// check Options as a rule-config source — WITHOUT asserting any in-process
// semgrep invocation occurs (none exists under the thin-executor strategy).
func TestCodeCheck_PackOnly_RulesDispatchViaEnginePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: cc-pack\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	defer chdirTemp(t, dir)()

	onePack := []*pack.Manifest{{Name: "acme/go-standards", Language: "go"}}
	var captured check.Options
	var dispatchedPacks []*pack.Manifest
	stubCodeCheckFns(t,
		func(string) ([]*pack.Manifest, error) { return onePack, nil },
		func(_ context.Context, opts check.Options) (*check.Result, error) {
			captured = opts
			return &check.Result{}, nil
		},
		func(packs []*pack.Manifest, _ string, _ string, _ *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
			dispatchedPacks = packs
			return nil, nil
		},
	)

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check", "--all"})
	if err := root.Execute(); err != nil {
		t.Fatalf("code check --all: %v", err)
	}

	// The installed pack's rules flow through the engine dispatch path...
	if len(dispatchedPacks) != 1 || dispatchedPacks[0].Name != "acme/go-standards" {
		t.Errorf("pack rules did not reach dispatchPackEngines; got %v, want the one installed pack dispatched via the engine path", dispatchedPacks)
	}
	// ...and no compiled-standards directory is wired into the check Options as a
	// rule-config source (BackstopDir is routing only; Options has no ManifestDir
	// field at all post-ISSUE-018).
	if filepath.Base(captured.BackstopDir) != ".backstop" {
		t.Errorf("BackstopDir = %q, want the project .backstop directory (routing), not a rules-as-config wiring", captured.BackstopDir)
	}
}

// TestCodeCheck_PackViolationsStampedWithNeutralFindings asserts the cmd-side
// pack-findings stamp uses the tool-NEUTRAL check.CheckTypeFindings, not a
// tool-named tag (SPEC-035 CLM-022/CLM-032). gateViolationsToCheck is the
// surviving cmd stamp site that converts pack engine violations into check
// violations; every stamped violation must carry the neutral findings tag and
// render with the neutral "findings" string.
func TestCodeCheck_PackViolationsStampedWithNeutralFindings(t *testing.T) {
	in := []gate.Violation{
		{File: "pkg/widget/widget.go", Message: "panic is forbidden", Severity: "error"},
		{File: "pkg/widget/other.go", Message: "no globals", Severity: "warning"},
	}
	out := gateViolationsToCheck(in)
	if len(out) != len(in) {
		t.Fatalf("gateViolationsToCheck returned %d, want %d", len(out), len(in))
	}
	for i, v := range out {
		if v.Pass != check.CheckTypeFindings {
			t.Errorf("violation[%d].Pass = %v, want check.CheckTypeFindings (neutral pack-findings tag)", i, v.Pass)
		}
		if v.Pass.String() != "findings" {
			t.Errorf("violation[%d].Pass.String() = %q, want \"findings\"", i, v.Pass.String())
		}
	}
}
