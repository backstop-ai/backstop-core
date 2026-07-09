package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackVal_PackCheckCommand_Exists(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"pack", "check"})
	if err != nil || cmd == nil {
		t.Fatalf("pack check missing: %v", err)
	}
}

// TestPackVal_PackCheckCommand_DefaultText proves the human default is TEXT after the
// Defect C dead-branch fix (ISSUE-032 CLM-007): the default without --json is the text
// renderer, not JSON. (Run in a dir with no pack.yml, the pipeline still renders a
// phase1 failure — as text, no leading brace.)
func TestPackVal_PackCheckCommand_DefaultText(t *testing.T) {
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "check") // nosemgrep: go.core.no-ignored-errors — asserts on stdout; the command error is deliberately not the assertion here
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("expected text default, got JSON: %s", out)
	}
}

func TestPackVal_PackCheckCommand_TextFormat(t *testing.T) {
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "check", "--format", "text") // nosemgrep: go.core.no-ignored-errors — asserts on stdout; the command error is deliberately not the assertion here
	if !strings.Contains(out, "status:") {
		t.Fatal("expected text output")
	}
}

func TestPackVal_PackCheckCommand_InvalidPack(t *testing.T) {
	root := NewRootCommand()
	_, err := executeCommand(root, "pack", "check", "--format", "json")
	if err == nil {
		t.Fatal("expected error in repo root without pack.yml")
	}
}

func TestPackVal_PackCheckCommand_ValidPack(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "rules/r1.yml", "rules: []")
	writeFileForTest(t, dir, "fixtures/p.go", "package p")
	writeFileForTest(t, dir, "fixtures/n.go", "package p")
	writeFileForTest(t, dir, "pack.yml", `
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
content:
  ruleset:
    rules:
      - id: R1
        file: rules/r1.yml
        risk_class: style
        layer: 1
        claims:
          - id: C1
            fixtures:
              positive: [fixtures/p.go]
              negative: [fixtures/n.go]
`)
	cwd := chdirForTest(t, dir)
	defer cwd()
	root := NewRootCommand()
	// The human default is now text (Defect C); request JSON explicitly to assert the
	// structured "status": "pass".
	out, err := executeCommand(root, "pack", "check", "--format", "json")
	if err != nil || !strings.Contains(out, `"status": "pass"`) {
		t.Fatalf("want pass output err=%v out=%s", err, out)
	}
}

// TestPackVal_PackCheckCommand_PathArg proves `pack check <pack-dir>` validates the pack at
// the GIVEN path from a DIFFERENT cwd — the real post-`pack new` flow (ISSUE-049). The
// ISSUE-032 e2e missed this by cd-ing into the pack dir (cmd.Dir=packDir), so the arg was
// never exercised. Here we stay in the parent dir and address the pack by path.
func TestPackVal_PackCheckCommand_PathArg(t *testing.T) {
	parent := t.TempDir()
	pack := filepath.Join(parent, "mypack")
	writeFileForTest(t, pack, "rules/r1.yml", "rules: []")
	writeFileForTest(t, pack, "fixtures/p.go", "package p")
	writeFileForTest(t, pack, "fixtures/n.go", "package p")
	writeFileForTest(t, pack, "pack.yml", `
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
content:
  ruleset:
    rules:
      - id: R1
        file: rules/r1.yml
        risk_class: style
        layer: 1
        claims:
          - id: C1
            fixtures:
              positive: [fixtures/p.go]
              negative: [fixtures/n.go]
`)
	cwd := chdirForTest(t, parent)
	defer cwd()

	// Valid pack addressed by path from the parent cwd → pass.
	out, err := executeCommand(NewRootCommand(), "pack", "check", "./mypack", "--format", "json")
	if err != nil || !strings.Contains(out, `"status": "pass"`) {
		t.Fatalf("pack check ./mypack from parent: want pass err=%v out=%s", err, out)
	}

	// A bad path fails and the report names THE PATH (not the bare cwd), so it is actionable.
	out, err = executeCommand(NewRootCommand(), "pack", "check", "./nonexistent", "--format", "json")
	if err == nil {
		t.Fatal("pack check ./nonexistent: expected error for missing pack")
	}
	if !strings.Contains(out, "nonexistent/pack.yml") {
		t.Errorf("report should name the given path, got out=%s err=%v", out, err)
	}
}

func TestPackVal_PackTestCommand_Exists(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"pack", "test"})
	if err != nil || cmd == nil {
		t.Fatalf("pack test missing: %v", err)
	}
}

func TestPackVal_PackTestCommand_RunsAllPhases(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "pack.yml", `
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
content:
  ruleset:
    rules:
      - id: R1
        risk_class: style
`)
	cwd := chdirForTest(t, dir)
	defer cwd()
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "test") // nosemgrep: go.core.no-ignored-errors — asserts on stdout; the command error is deliberately not the assertion here
	if !strings.Contains(out, "phase6-risk-class") {
		t.Fatal("expected all phases")
	}
}

// TestPackVal_PackTestCommand_DefaultText proves the human default is TEXT after the
// Defect C fix (ISSUE-032 CLM-007).
func TestPackVal_PackTestCommand_DefaultText(t *testing.T) {
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "test") // nosemgrep: go.core.no-ignored-errors — asserts on stdout; the command error is deliberately not the assertion here
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("expected text default, got JSON: %s", out)
	}
}

func TestPackVal_PackTestCommand_TextFormat(t *testing.T) {
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "test", "--format", "text") // nosemgrep: go.core.no-ignored-errors — asserts on stdout; the command error is deliberately not the assertion here
	if !strings.Contains(out, "status:") {
		t.Fatal("expected text")
	}
}

// TestPackVal_PackCheckCommand_GlobalJSONFlag verifies that the persistent
// --json flag (parsed before the subcommand) drives the pack check format to
// JSON via the shared jsonFlag pointer.
func TestPackVal_PackCheckCommand_GlobalJSONFlag(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "pack.yml", `
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
content:
  ruleset:
    rules:
      - id: R1
        risk_class: style
`)
	defer chdirForTest(t, dir)()

	root := NewRootCommand()
	out, _ := executeCommand(root, "--json", "pack", "check") // nosemgrep: go.core.no-ignored-errors — asserts on stdout; the command error is deliberately not the assertion here
	if !strings.Contains(out, "{") {
		t.Fatalf("expected JSON output under global --json, got: %s", out)
	}
}

// TestPackVal_PackTestCommand_GlobalJSONFlag verifies the same for pack test.
func TestPackVal_PackTestCommand_GlobalJSONFlag(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "pack.yml", `
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
content:
  ruleset:
    rules:
      - id: R1
        risk_class: style
`)
	defer chdirForTest(t, dir)()

	root := NewRootCommand()
	out, _ := executeCommand(root, "--json", "pack", "test") // nosemgrep: go.core.no-ignored-errors — asserts on stdout; the command error is deliberately not the assertion here
	if !strings.Contains(out, "{") {
		t.Fatalf("expected JSON output under global --json, got: %s", out)
	}
}

func writeFileForTest(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(strings.TrimSpace(content)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func chdirForTest(t *testing.T, dir string) func() {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() {
		_ = os.Chdir(old)
	}
}
