package main

import (
	"testing"
)

func TestPackUpgradeCommand_Registration(t *testing.T) {
	root := NewRootCommand()

	cmd, _, err := root.Find([]string{"pack", "upgrade"})
	if err != nil {
		t.Fatalf("pack upgrade not found: %v", err)
	}

	if cmd.Use != "upgrade [pack-ref@version]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "upgrade [pack-ref@version]")
	}
}

func TestPackUpgradeCommand_NoArgs(t *testing.T) {
	root := NewRootCommand()

	root.SetArgs([]string{"pack", "upgrade"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error running pack upgrade without arguments")
	}
}
