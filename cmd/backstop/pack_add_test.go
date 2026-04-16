package main

import (
	"strings"
	"testing"
)

func TestPackAddCommand_Registration(t *testing.T) {
	root := NewRootCommand()

	cmd, _, err := root.Find([]string{"pack", "add"})
	if err != nil {
		t.Fatalf("pack add not found: %v", err)
	}

	if cmd.Use != "add [pack-ref]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "add [pack-ref]")
	}
}

func TestPackAddCommand_FlagParsing(t *testing.T) {
	root := NewRootCommand()

	cmd, _, _ := root.Find([]string{"pack", "add"})

	flag := cmd.Flags().Lookup("version")
	if flag == nil {
		t.Error("missing --version flag")
	}
}

func TestPackAddCommand_ExitCode(t *testing.T) {
	root := NewRootCommand()

	// Running without args should produce an error.
	root.SetArgs([]string{"pack", "add"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error running pack add without arguments")
	}
}

func TestPackRemoveCommand_Registration(t *testing.T) {
	root := NewRootCommand()

	cmd, _, err := root.Find([]string{"pack", "remove"})
	if err != nil {
		t.Fatalf("pack remove not found: %v", err)
	}

	if !strings.HasPrefix(cmd.Use, "remove") {
		t.Errorf("Use = %q, expected to start with 'remove'", cmd.Use)
	}
}

func TestPackRemoveCommand_ExitCode(t *testing.T) {
	root := NewRootCommand()

	root.SetArgs([]string{"pack", "remove"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error running pack remove without arguments")
	}
}

func TestPackInstallCommand_Registration(t *testing.T) {
	root := NewRootCommand()

	cmd, _, err := root.Find([]string{"pack", "install"})
	if err != nil {
		t.Fatalf("pack install not found: %v", err)
	}

	if !strings.HasPrefix(cmd.Use, "install") {
		t.Errorf("Use = %q, expected to start with 'install'", cmd.Use)
	}
}

func TestPackInstallCommand_CacheFlag(t *testing.T) {
	root := NewRootCommand()

	cmd, _, _ := root.Find([]string{"pack", "install"})

	flag := cmd.Flags().Lookup("cache")
	if flag == nil {
		t.Error("missing --cache flag")
	}
}

func TestPackUpdateCommand_Registration(t *testing.T) {
	root := NewRootCommand()

	cmd, _, err := root.Find([]string{"pack", "update"})
	if err != nil {
		t.Fatalf("pack update not found: %v", err)
	}

	if !strings.HasPrefix(cmd.Use, "update") {
		t.Errorf("Use = %q, expected to start with 'update'", cmd.Use)
	}
}

func TestPackUpdateCommand_AcknowledgeFlag(t *testing.T) {
	root := NewRootCommand()

	cmd, _, _ := root.Find([]string{"pack", "update"})

	flag := cmd.Flags().Lookup("acknowledge")
	if flag == nil {
		t.Error("missing --acknowledge flag")
	}
}
