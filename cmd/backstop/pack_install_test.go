package main

import (
	"testing"
)

func TestPackInstallCommand_MissingLockfile(t *testing.T) {
	root := NewRootCommand()

	// Run in temp dir with no backstop.lock.
	root.SetArgs([]string{"pack", "install"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error running pack install without lockfile")
	}
}
