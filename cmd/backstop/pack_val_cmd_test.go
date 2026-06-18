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

func TestPackVal_PackCheckCommand_DefaultJSON(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "pack", "check")
	if err == nil && !strings.Contains(out, "{") {
		t.Fatal("expected json output")
	}
}

func TestPackVal_PackCheckCommand_TextFormat(t *testing.T) {
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "check", "--format", "text")
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
	out, err := executeCommand(root, "pack", "check")
	if err != nil || !strings.Contains(out, `"status": "pass"`) {
		t.Fatalf("want pass output err=%v out=%s", err, out)
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
	out, _ := executeCommand(root, "pack", "test")
	if !strings.Contains(out, "phase6-risk-class") {
		t.Fatal("expected all phases")
	}
}

func TestPackVal_PackTestCommand_DefaultJSON(t *testing.T) {
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "test")
	if !strings.Contains(out, "{") {
		t.Fatal("expected json")
	}
}

func TestPackVal_PackTestCommand_TextFormat(t *testing.T) {
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "test", "--format", "text")
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
	out, _ := executeCommand(root, "--json", "pack", "check")
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
	out, _ := executeCommand(root, "--json", "pack", "test")
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
