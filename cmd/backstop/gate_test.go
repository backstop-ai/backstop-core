package main

import (
	"strings"
	"testing"

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
