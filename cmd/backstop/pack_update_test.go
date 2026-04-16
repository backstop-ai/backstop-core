package main

import (
	"testing"
)

func TestPackUpdateCommand_NoPackName(t *testing.T) {
	root := NewRootCommand()

	root.SetArgs([]string{"pack", "update"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error running pack update without pack name")
	}
}
