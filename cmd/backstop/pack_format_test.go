package main

import (
	"strings"
	"testing"
)

// validPackForFormat writes a minimal valid pack.yml into a temp dir, chdirs into
// it, and returns. Cleanup restores the cwd.
func validPackForFormat(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	writeFileForTest(t, dir, "pack.yml", `
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
content:
  ruleset:
    rules:
      - id: r1
        engine: semgrep
        risk_class: style
        claims:
          - id: c1
            fixtures:
              positive: [fixtures/p.txt]
              negative: [fixtures/n.txt]
`)
	writeFileForTest(t, dir, "fixtures/p.txt", "positive")
	writeFileForTest(t, dir, "fixtures/n.txt", "negative")
	t.Cleanup(chdirForTest(t, dir))
}

// TestPackFormat_CheckDefaultIsText proves `pack check` with no --format and no
// --json emits TEXT, not JSON (ISSUE-032 Defect C / CLM-007). Against the dead branch
// — both if/else arms assigned "json" — the default was JSON and this fails.
func TestPackFormat_CheckDefaultIsText(t *testing.T) {
	validPackForFormat(t)
	root := NewRootCommand()
	out, err := executeCommand(root, "pack", "check")
	if err != nil {
		t.Fatalf("pack check errored: %v (out=%s)", err, out)
	}
	assertText(t, "pack check default", out)
}

// TestPackFormat_TestDefaultIsText proves the same for `pack test`.
func TestPackFormat_TestDefaultIsText(t *testing.T) {
	validPackForFormat(t)
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "test")
	assertText(t, "pack test default", out)
}

// TestPackFormat_CheckFormatTextIsText guards that --format text stays TEXT.
func TestPackFormat_CheckFormatTextIsText(t *testing.T) {
	validPackForFormat(t)
	root := NewRootCommand()
	out, err := executeCommand(root, "pack", "check", "--format", "text")
	if err != nil {
		t.Fatalf("pack check --format text errored: %v", err)
	}
	assertText(t, "pack check --format text", out)
}

// TestPackFormat_TestFormatTextIsText guards --format text for pack test.
func TestPackFormat_TestFormatTextIsText(t *testing.T) {
	validPackForFormat(t)
	root := NewRootCommand()
	out, _ := executeCommand(root, "pack", "test", "--format", "text")
	assertText(t, "pack test --format text", out)
}

// assertText asserts the output is the text renderer's shape (a "status:" line, no
// leading JSON brace).
func assertText(t *testing.T, label, out string) {
	t.Helper()
	trimmed := strings.TrimSpace(out)
	if strings.HasPrefix(trimmed, "{") {
		t.Fatalf("%s: expected TEXT output, got JSON: %s", label, out)
	}
	if !strings.Contains(out, "status:") {
		t.Fatalf("%s: expected a text 'status:' line, got: %s", label, out)
	}
}
