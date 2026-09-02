package main

import (
	"os/exec"
	"testing"
)

func TestGeneratedPagesMatchCommitted(t *testing.T) {
	root := repoRoot()
	cmd := exec.Command("go", "run", "./scripts/entityref", "-check")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("entityref -check: %v\n%s", err, out)
	}
}
