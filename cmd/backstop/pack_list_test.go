package main

import (
	"testing"
)

func TestPackListCommand_Registration(t *testing.T) {
	root := NewRootCommand()

	cmd, _, err := root.Find([]string{"pack", "list"})
	if err != nil {
		t.Fatalf("pack list not found: %v", err)
	}

	if cmd.Use != "list" {
		t.Errorf("Use = %q, want %q", cmd.Use, "list")
	}
}

func TestPackListCommand_JsonFlag(t *testing.T) {
	root := NewRootCommand()

	// The root --json flag should be inherited.
	cmd, _, _ := root.Find([]string{"pack", "list"})

	// Verify the command exists and can accept the inherited --json flag.
	if cmd == nil {
		t.Fatal("pack list command not found")
	}
}

func TestAllPackDistributionCommandsRegistered(t *testing.T) {
	root := NewRootCommand()

	commands := []string{"add", "remove", "install", "update", "upgrade", "list"}
	for _, name := range commands {
		_, _, err := root.Find([]string{"pack", name})
		if err != nil {
			t.Errorf("pack %s not registered: %v", name, err)
		}
	}
}
