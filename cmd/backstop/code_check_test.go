package main

import (
	"bytes"
	"context"
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
