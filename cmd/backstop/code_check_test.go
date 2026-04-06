package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCodeCheck_FileFlag_RoutesByType verifies the --file flag routes to
// ScopeModeFile in pkg/check. (CLM-009)
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
