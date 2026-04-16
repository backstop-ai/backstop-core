package main

import (
	"testing"
)

func TestPackRemoveCommand_NotInstalledError(t *testing.T) {
	root := NewRootCommand()

	root.SetArgs([]string{"pack", "remove"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error running pack remove without pack name")
	}
}
