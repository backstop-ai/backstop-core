package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/spf13/pflag"
)

// TestGate_NoScopeFlags_FullProject verifies gate command runs without any
// scope flags and executes against the full project.
func TestGate_NoScopeFlags_FullProject(t *testing.T) {
	root := NewRootCommand()
	// gate should not require any scope flags — just --json is inherited from root
	cmd, _, err := root.Find([]string{"gate"})
	if err != nil {
		t.Fatalf("find gate: %v", err)
	}
	if cmd.Name() != "gate" {
		t.Errorf("expected command name %q, got %q", "gate", cmd.Name())
	}

	// Verify gate has no local flags (only inherits --json from root).
	// We check by counting flags on the command's own flag set.
	localFlagCount := 0
	cmd.LocalFlags().VisitAll(func(_ *pflag.Flag) { localFlagCount++ })
	if localFlagCount != 0 {
		t.Errorf("expected 0 local flags on gate command, got %d", localFlagCount)
	}
}

// TestGate_RejectsScopeFlags verifies gate does not accept --diff, --file,
// --spec, --plan, or other scoping flags.
func TestGate_RejectsScopeFlags(t *testing.T) {
	scopeFlags := []string{"--diff", "--file", "--spec", "--plan", "--all"}

	for _, flag := range scopeFlags {
		t.Run(flag, func(t *testing.T) {
			root := NewRootCommand()
			_, err := executeCommand(root, "gate", flag)
			if err == nil {
				t.Errorf("expected error for gate %s, but got nil", flag)
				return
			}
			if !strings.Contains(err.Error(), "unknown flag") {
				t.Errorf("expected 'unknown flag' error for gate %s, got: %v", flag, err)
			}
		})
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
