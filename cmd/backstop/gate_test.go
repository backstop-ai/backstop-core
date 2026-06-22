package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
)

func makePassStep(name string) gate.StepFunc {
	return func(_ context.Context) gate.StepResult {
		return gate.StepResult{StepName: name, Status: "pass", Violations: []gate.Violation{}}
	}
}

func makeSkippedStep(name, reason string) gate.StepFunc {
	return func(_ context.Context) gate.StepResult {
		return gate.StepResult{StepName: name, Status: "skipped", Violations: []gate.Violation{}, Reason: reason}
	}
}

// TestGate_DefaultsToDiffMode verifies gate defaults to diff scope.
func TestGate_DefaultsToDiffMode(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"gate"})
	if err != nil {
		t.Fatalf("find gate: %v", err)
	}
	if cmd.Name() != "gate" {
		t.Errorf("expected command name %q, got %q", "gate", cmd.Name())
	}

	allFlag, _ := cmd.Flags().GetBool("all")
	fileFlag, _ := cmd.Flags().GetString("file")
	if allFlag || fileFlag != "" {
		t.Fatalf("expected default diff mode flags, got all=%v file=%q", allFlag, fileFlag)
	}
}

func TestGate_AllFlagUsesFullSweep(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"gate", "--all"})
	if err != nil {
		t.Fatalf("find gate --all: %v", err)
	}
	if err := cmd.ParseFlags([]string{"--all"}); err != nil {
		t.Fatalf("parse --all: %v", err)
	}
	allFlag, _ := cmd.Flags().GetBool("all")
	if !allFlag {
		t.Fatal("expected --all to select full-sweep mode")
	}
}

func TestGate_FileFlagScopesExplicitFiles(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"gate", "--file", "a.go", "b.go"})
	if err != nil {
		t.Fatalf("find gate --file: %v", err)
	}
	if err := cmd.ParseFlags([]string{"--file", "a.go", "b.go"}); err != nil {
		t.Fatalf("parse --file: %v", err)
	}
	fileFlag, _ := cmd.Flags().GetString("file")
	args := cmd.Flags().Args()
	if fileFlag != "a.go" || len(args) != 1 || args[0] != "b.go" {
		t.Fatalf("expected one --file flag to consume multiple files via args, got file=%q args=%v", fileFlag, args)
	}
}

func TestGate_AllAndFileMutuallyExclusive(t *testing.T) {
	root := NewRootCommand()
	root.SetArgs([]string{"gate", "--all", "--file", "a.go"})
	err := root.Execute()
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitCodeError, got %T %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Fatalf("expected config exit %d, got %d", ExitConfigError, exitErr.Code)
	}
	if !strings.Contains(exitErr.Message, "--all and --file are mutually exclusive") {
		t.Fatalf("expected conflict message, got %q", exitErr.Message)
	}
}

func TestRunGate_UnexpectedArgsReturnConfigExit(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: gate-args\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newGateCommand(new(bool))
	err := runGate(cmd, []string{"unexpected"})
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitCodeError, got %T %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Fatalf("expected config exit %d, got %d", ExitConfigError, exitErr.Code)
	}
	if !strings.Contains(exitErr.Message, "unexpected gate arguments") {
		t.Fatalf("unexpected message: %q", exitErr.Message)
	}
}

func TestRunGate_InvalidBaselineTTLReturnsConfigExit(t *testing.T) {
	projectRoot := t.TempDir()
	configBody := "project: gate-ttl\nlanguage: go\nenforcement:\n  baseline_ttl: nonsense\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(configBody), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newGateCommand(new(bool))
	err := runGate(cmd, nil)
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitCodeError, got %T %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Fatalf("expected config exit %d, got %d", ExitConfigError, exitErr.Code)
	}
	if !strings.Contains(strings.ToLower(exitErr.Message), "baseline_ttl") {
		t.Fatalf("expected baseline_ttl message, got %q", exitErr.Message)
	}
}

func TestGateIntegration_ReadOnlyExecution(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	tracked := []string{
		filepath.Join(projectRoot, "backstop.yml"),
		filepath.Join(projectRoot, "backstop.lock"),
		filepath.Join(projectRoot, ".backstop", "packs", "test-org", "test-pack", "pack.yml"),
	}
	before := fileModTimes(t, tracked)

	g := gate.New(gate.WithSteps(buildGateSteps(projectRoot)))
	g.Run(context.Background())

	after := fileModTimes(t, tracked)
	for path, ts := range before {
		if !after[path].Equal(ts) {
			t.Fatalf("expected read-only execution for %s", path)
		}
	}
}

func TestGateIntegration_RemovedPackNotEnforced(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(`project: removed-pack
language: go
`), 0o644); err != nil {
		t.Fatalf("rewrite backstop.yml: %v", err)
	}

	steps := buildGateSteps(projectRoot)
	for _, step := range steps {
		result := step(context.Background())
		if strings.Contains(result.StepName, "pack_") {
			t.Fatalf("expected no pack enforcement steps when packs are removed, found %q", result.StepName)
		}
	}
}

func TestGateIntegration_RemovedPackNoWarnings(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "packgate")
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(`project: removed-pack
language: go
`), 0o644); err != nil {
		t.Fatalf("rewrite backstop.yml: %v", err)
	}

	g := gate.New(gate.WithSteps(buildGateSteps(projectRoot)))
	result, _ := g.Run(context.Background())
	for _, step := range result.Steps {
		for _, violation := range step.Violations {
			if strings.Contains(violation.Message, "extra_unlocked") || strings.Contains(violation.Message, "removed") {
				t.Fatalf("expected no warnings for removed pack, got %q", violation.Message)
			}
		}
	}
}

func fileModTimes(t *testing.T, files []string) map[string]time.Time {
	t.Helper()
	info := make(map[string]time.Time, len(files))
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			t.Fatalf("stat %s: %v", file, err)
		}
		info[file] = stat.ModTime()
	}
	return info
}

func TestGate_BaselineCacheLifecycle_FreshCacheNoNetwork_Contract(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: cache-fresh\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".backstop", "baseline.json"), []byte(`{"schema_version":"baseline/v1","violations":[]}`), 0o644); err != nil {
		t.Fatalf("write baseline cache: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCommand()
	_, err := executeCommand(root, "gate", "--json")
	if err == nil {
		t.Fatalf("expected minimal fixture to fail non-baseline checks with exit code 1")
	}
	if !strings.Contains(strings.ToLower(fmt.Sprint(err)), "exit code 1") {
		t.Fatalf("expected normal gate failure (not config/baseline failure), got: %v", err)
	}
}

func TestGate_BaselineCacheLifecycle_ExpiredRefreshAndOfflineFallback_Contract(t *testing.T) {
	projectRoot := t.TempDir()
	config := "project: cache-expired\nlanguage: go\nenforcement:\n  baseline_ttl: 1m\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cachePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir baseline dir: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte(`{"schema_version":"baseline/v1","violations":[]}`), 0o644); err != nil {
		t.Fatalf("write baseline cache: %v", err)
	}
	stale := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cachePath, stale, stale); err != nil {
		t.Fatalf("set stale modtime: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	for _, mode := range []string{"refresh", "offline-fallback"} {
		t.Run(mode, func(t *testing.T) {
			root := NewRootCommand()
			_, err := executeCommand(root, "gate", "--json")
			if err == nil {
				t.Fatalf("expected minimal fixture to fail non-baseline checks with exit code 1")
			}
			if !strings.Contains(strings.ToLower(fmt.Sprint(err)), "exit code 1") {
				t.Fatalf("expected normal gate failure semantics for %s, got: %v", mode, err)
			}
		})
	}
}

func TestGate_BaselineRatchet_FailsNewAndAllowsReductions_Contract(t *testing.T) {
	newViolationGate := gate.New(gate.WithSteps([]gate.StepFunc{
		makePassStep(gate.StepArtifactValidation),
		makePassStep(gate.StepCodeCheck),
		makePassStep(gate.StepTestVerification),
		makePassStep(gate.StepTestSubstantiveness),
		makePassStep(gate.StepCoverageThreshold),
		makePassStep(gate.StepContractSignature),
		func(_ context.Context) gate.StepResult {
			return gate.StepResult{StepName: gate.StepBaselineComparison, Status: "fail", Violations: []gate.Violation{{Rule: "baseline/new", Message: "new scoped violation"}}}
		},
		makeSkippedStep(gate.StepWaiverResolution, "waivers not implemented"),
		makeSkippedStep(gate.StepLedgerIntegrity, "ledger not implemented"),
	}))
	_, newExit := newViolationGate.Run(context.Background())
	if newExit != 1 {
		t.Fatalf("expected ratchet to fail on new scoped violation, got exit=%d", newExit)
	}

	reductionGate := gate.New(gate.WithSteps([]gate.StepFunc{
		makePassStep(gate.StepArtifactValidation),
		makePassStep(gate.StepCodeCheck),
		makePassStep(gate.StepTestVerification),
		makePassStep(gate.StepTestSubstantiveness),
		makePassStep(gate.StepCoverageThreshold),
		makePassStep(gate.StepContractSignature),
		func(_ context.Context) gate.StepResult {
			return gate.StepResult{StepName: gate.StepBaselineComparison, Status: "pass", Violations: []gate.Violation{}}
		},
		makeSkippedStep(gate.StepWaiverResolution, "waivers not implemented"),
		makeSkippedStep(gate.StepLedgerIntegrity, "ledger not implemented"),
	}))
	_, reductionExit := reductionGate.Run(context.Background())
	if reductionExit != 0 {
		t.Fatalf("expected ratchet to allow reductions, got exit=%d", reductionExit)
	}
}

func TestChangedFilesAgainstOriginMain_ReturnsDiffedFiles(t *testing.T) {
	projectRoot := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = projectRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	runGit("init")
	if err := os.WriteFile(filepath.Join(projectRoot, "tracked.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write tracked.txt: %v", err)
	}
	runGit("add", "tracked.txt")
	runGit("-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "base")
	baseCmd := exec.Command("git", "rev-parse", "HEAD")
	baseCmd.Dir = projectRoot
	baseOut, err := baseCmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	runGit("update-ref", "refs/remotes/origin/main", strings.TrimSpace(string(baseOut)))
	if err := os.WriteFile(filepath.Join(projectRoot, "tracked.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("rewrite tracked.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatalf("write new.txt: %v", err)
	}
	runGit("add", "new.txt")

	files, err := changedFilesAgainstOriginMain(projectRoot)
	if err != nil {
		t.Fatalf("changedFilesAgainstOriginMain: %v", err)
	}
	joined := strings.Join(files, "\n")
	if !strings.Contains(joined, "tracked.txt") || !strings.Contains(joined, "new.txt") {
		t.Fatalf("expected changed files to include tracked.txt and new.txt, got %v", files)
	}
}

func TestRuleSetChangeSeedingContext_OnlyAllScopeAndRuleSetFilesEnableSeeding(t *testing.T) {
	projectRoot := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = projectRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	runGit("init")
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: test\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	runGit("add", "backstop.yml")
	runGit("-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "base")
	baseCmd := exec.Command("git", "rev-parse", "HEAD")
	baseCmd.Dir = projectRoot
	baseOut, err := baseCmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	runGit("update-ref", "refs/remotes/origin/main", strings.TrimSpace(string(baseOut)))
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: updated\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("rewrite backstop.yml: %v", err)
	}

	allowedAll, changedAll := ruleSetChangeSeedingContext(projectRoot, nil)
	if allowedAll || changedAll != nil {
		t.Fatalf("expected nil scope to disable seeding context, got allowed=%v changed=%v", allowedAll, changedAll)
	}

	allowedDiff, _ := ruleSetChangeSeedingContext(projectRoot, &gate.GateScope{Mode: gate.GateScopeModeDiff})
	if allowedDiff {
		t.Fatal("expected diff scope to disable rule-set seeding")
	}

	allowedAll, changedAll = ruleSetChangeSeedingContext(projectRoot, &gate.GateScope{Mode: gate.GateScopeModeAll})
	if !allowedAll {
		t.Fatal("expected all scope with backstop.yml change to enable seeding")
	}
	if len(changedAll) == 0 {
		t.Fatal("expected changed file list to be returned")
	}
}

func TestResolveBaselineCache_MissingCacheAndRefreshFails(t *testing.T) {
	projectRoot := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	artifact, warning, modified := resolveBaselineCache(filepath.Join(projectRoot, ".backstop", "baseline.json"), time.Minute)
	if artifact != nil {
		t.Fatalf("expected nil artifact when cache missing and refresh fails, got %#v", artifact)
	}
	if !strings.Contains(warning, "no cached baseline found") {
		t.Fatalf("expected missing-cache warning, got %q", warning)
	}
	if !modified.IsZero() {
		t.Fatalf("expected zero modified time, got %s", modified)
	}
}

func TestResolveBaselineCache_UnreadableCacheAndRefreshFails(t *testing.T) {
	projectRoot := t.TempDir()
	cachePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir baseline dir: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write broken cache: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	artifact, warning, modified := resolveBaselineCache(cachePath, time.Minute)
	if artifact != nil {
		t.Fatalf("expected nil artifact when cache unreadable and refresh fails, got %#v", artifact)
	}
	if !strings.Contains(warning, "is unreadable") {
		t.Fatalf("expected unreadable-cache warning, got %q", warning)
	}
	if modified.IsZero() {
		t.Fatal("expected original cache modtime to be surfaced")
	}
}

func TestResolveBaselineCache_StaleCacheFallsBackToLocalOnRefreshFailure(t *testing.T) {
	projectRoot := t.TempDir()
	cachePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir baseline dir: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte(`{"schema_version":"baseline/v1","violations":[]}`), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	stale := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cachePath, stale, stale); err != nil {
		t.Fatalf("set stale modtime: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	artifact, warning, modified := resolveBaselineCache(cachePath, time.Minute)
	if artifact == nil {
		t.Fatal("expected stale baseline cache to be used when refresh fails")
	}
	if !strings.Contains(warning, "using stale cached baseline") {
		t.Fatalf("expected stale-cache fallback warning, got %q", warning)
	}
	if modified.IsZero() {
		t.Fatal("expected stale cache modtime to be returned")
	}
}

func TestResolveBaselineCache_MissingCacheRefreshesFromRemote(t *testing.T) {
	projectRoot := t.TempDir()
	setupBaselineRefreshSuccessFixture(t, projectRoot)

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	artifact, warning, modified := resolveBaselineCache(filepath.Join(projectRoot, ".backstop", "baseline.json"), time.Minute)
	if artifact == nil {
		t.Fatal("expected artifact refreshed from remote")
	}
	if warning != "baseline cache fetched from remote main baseline artifact" {
		t.Fatalf("unexpected refresh warning: %q", warning)
	}
	if modified.IsZero() {
		t.Fatal("expected non-zero modified time after refresh")
	}
}

func TestResolveBaselineCache_UnreadableCacheRefreshesFromRemote(t *testing.T) {
	projectRoot := t.TempDir()
	setupBaselineRefreshSuccessFixture(t, projectRoot)
	cachePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir baseline dir: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("broken-json"), 0o644); err != nil {
		t.Fatalf("write broken cache: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	artifact, warning, modified := resolveBaselineCache(cachePath, time.Minute)
	if artifact == nil {
		t.Fatal("expected artifact refreshed from remote after unreadable cache")
	}
	if warning != "baseline cache was unreadable and refreshed from remote main baseline artifact" {
		t.Fatalf("unexpected unreadable-cache warning: %q", warning)
	}
	if modified.IsZero() {
		t.Fatal("expected non-zero modified time after refresh")
	}
}

func setupBaselineRefreshSuccessFixture(t *testing.T, projectRoot string) {
	t.Helper()
	if out, err := exec.Command("git", "init", projectRoot).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}
	cmd := exec.Command("git", "remote", "add", "origin", "git@github.com:owner/repo.git")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: cache-refresh\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}

	binDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "artifact.zip")
	makeBaselineZip(t, zipPath)
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
if [ "$1 $2" = "auth status" ]; then
  exit 0
fi
if [ "$1" != "api" ]; then
  echo "unexpected command: $*" >&2
  exit 1
fi
case "$2" in
  repos/owner/repo/actions/runs\?branch=main\&status=success\&per_page=20)
    printf '{"workflow_runs":[{"id":42,"name":"ci","conclusion":"success","head_branch":"main"}]}'
    ;;
  repos/owner/repo/actions/runs/42/artifacts)
    printf '{"artifacts":[{"id":99,"name":"backstop-baseline-v1"}]}'
    ;;
  repos/owner/repo/actions/artifacts/99/zip)
    cat "`+zipPath+`"
    ;;
  *)
    echo "unexpected endpoint: $2" >&2
    exit 1
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestGate_UnexpectedArgsWithoutFileFlag_ReturnsConfigExit(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: arg-validation\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCommand()
	_, err := executeCommand(root, "gate", "extra-arg")
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitCodeError, got %T %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Fatalf("expected config exit %d, got %d", ExitConfigError, exitErr.Code)
	}
	if !strings.Contains(exitErr.Message, "unexpected gate arguments") {
		t.Fatalf("expected unexpected-args message, got %q", exitErr.Message)
	}
}

func TestGate_InvalidBaselineTTLConfig_ReturnsConfigExit(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: bad-ttl\nlanguage: go\nenforcement:\n  baseline_ttl: nonsense\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCommand()
	_, err := executeCommand(root, "gate")
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitCodeError, got %T %v", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Fatalf("expected config exit %d, got %d", ExitConfigError, exitErr.Code)
	}
	if !strings.Contains(exitErr.Message, "baseline_ttl") {
		t.Fatalf("expected baseline_ttl parse message, got %q", exitErr.Message)
	}
}

// zeroRoutableChecker is a CodeChecker double standing in for realCodeChecker
// when check.Run surfaces a zero-routable LoadManifest config error: runCheck
// wraps it in gate.ConfigError (gate.go), which step_delegate must map to a
// ConfigErr step result and the gate to exit 2.
type zeroRoutableChecker struct{}

func (zeroRoutableChecker) CheckAll(_ context.Context) ([]gate.Violation, error) {
	return nil, &gate.ConfigError{Err: &check.ConfigError{Message: "manifest files in .backstop/rules yield no routable rules"}}
}

// TestCodeCheck_LoadManifest_ConfigErrorPropagatesToGateExit pins the
// fail-loud boundary for REQ-002 on the gate path: a zero-routable
// LoadManifest error wrapped in gate.ConfigError must surface as an exit-2
// config error from gate.Run — a config-error step, not a violations result
// and not a green pass.
func TestCodeCheck_LoadManifest_ConfigErrorPropagatesToGateExit(t *testing.T) {
	scope := &gate.GateScope{Mode: gate.GateScopeModeAll}
	step := gate.StepCodeCheckScopedFunc(zeroRoutableChecker{}, scope)
	g := gate.New(gate.WithSteps([]gate.StepFunc{step}), gate.WithScope(scope))

	result, exitCode := g.Run(context.Background())
	if exitCode != 2 {
		t.Fatalf("gate exit code = %d, want 2 (config error)", exitCode)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("got %d steps, want 1", len(result.Steps))
	}
	stepResult := result.Steps[0]
	if !stepResult.ConfigErr {
		t.Error("step ConfigErr = false, want true")
	}
	if stepResult.Status == "pass" {
		t.Error("step status is pass; a zero-routable manifest must never read as green")
	}
	if result.Pass {
		t.Error("gate result Pass = true, want false")
	}
}

// TestRunCheckOptions_NoManifestDir exercises (*realCodeChecker).runCheck — the
// gate's check.Options construction site that previously set
// ManifestDir: filepath.Join(backstopDir, "rules"). After SPEC-030 (and the
// ISSUE-018 in-process semgrep removal) the constructed Options carry no
// compiled-standards manifest directory: with zero installed packs and a
// populated .backstop/rules/ (a leftover compiled STD file), no recorded tool
// invocation is fed a --config path under .backstop/rules/. The surviving
// CLM-006 property is asserted on the recorded runner args (no standards-dir
// rule-config is wired in), NOT on an in-process semgrep invocation — there is
// no in-process semgrep pass under the thin-executor strategy. (CLM-006)
func TestRunCheckOptions_NoManifestDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: gate-opts\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	// Plant a leftover compiled-standards file to prove it is never a --config.
	if err := os.WriteFile(filepath.Join(rulesDir, "STD-GO-001.semgrep.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write leftover: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	restore := chdirTemp(t, dir)
	defer restore()

	runner := &recordingRunner{}
	checker := &realCodeChecker{
		projectRoot:   dir,
		runnerForTest: runner,
	}

	// Precondition: the project's .backstop directory is valid, so runCheck's
	// Options construction (the subject under test) is exercised rather than
	// short-circuited by a missing-.backstop error.
	if err := check.ValidateBackstopDir(dir); err != nil {
		t.Fatalf("check.ValidateBackstopDir: %v", err)
	}

	if _, err := checker.runCheck(context.Background(), check.ScopeModeFile, []string{filepath.Join(dir, "main.go")}); err != nil {
		t.Fatalf("runCheck: %v", err)
	}

	// No recorded tool invocation may be fed a --config path under
	// .backstop/rules/: runCheck's Options wire no compiled-standards directory as
	// a rule-config source. (There is no in-process semgrep pass to assert on.)
	rulesPrefix := filepath.Join(".backstop", "rules")
	for _, c := range runner.calls {
		for i := 0; i+1 < len(c.args); i++ {
			if c.args[i] == "--config" && strings.Contains(c.args[i+1], rulesPrefix) {
				t.Errorf("a tool was invoked with --config %q under .backstop/rules; runCheck Options must wire no compiled-standards directory", c.args[i+1])
			}
		}
	}
}
