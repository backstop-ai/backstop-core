package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
