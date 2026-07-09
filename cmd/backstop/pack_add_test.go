package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLocalPackSource writes a minimal versionless local pack source (pack.yml + a
// representative rule file) under parent/rel and returns its absolute dir.
func writeLocalPackSource(t *testing.T, parent, rel, name string) string {
	t.Helper()
	writeFileForTest(t, parent, filepath.Join(rel, "pack.yml"),
		"name: "+name+"\narchetype: rule-pack\ncontent:\n  ruleset:\n    rules:\n      - id: R1\n")
	writeFileForTest(t, parent, filepath.Join(rel, "rules", "r1.yml"), "rules:\n  - id: R1\n")
	return filepath.Join(parent, rel)
}

// TestPackAddCLI_LocalPackNoBareAt asserts CLM-004: adding a versionless LOCAL pack via
// the real `pack add` prints a success line with the pack name but NO bare trailing `@`.
func TestPackAddCLI_LocalPackNoBareAt(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, projectDir, "backstop.yml", "packs: {}")
	packName := "internal/local-rules"
	writeLocalPackSource(t, parent, "local-src", packName)

	restore := chdirForTest(t, projectDir)
	defer restore()

	out, err := executeCommand(NewRootCommand(), "pack", "add", "../local-src")
	if err != nil {
		t.Fatalf("pack add: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, "Added") || !strings.Contains(out, packName) {
		t.Errorf("expected a non-empty Added line naming %q, got: %q", packName, out)
	}
	if strings.Contains(out, packName+"@") {
		t.Errorf("versionless local pack must not render a bare `@` after the name, got: %q", out)
	}
}

// TestPackAddCLI_GitPackKeepsVersion asserts CLM-004: a pack WITH a version still renders
// `<name>@<version>` — the versionless stripping must not be over-eager. Exercises the
// pure display helper directly (a real git clone is out of scope for a unit test).
func TestPackAddCLI_GitPackKeepsVersion(t *testing.T) {
	line := formatAddedLine("acme/some-pack", "1.2.3", "sha256:abc")
	if !strings.Contains(line, "acme/some-pack@1.2.3") {
		t.Errorf("versioned pack must render name@version, got: %q", line)
	}
}

// TestPackAddCLI_AlreadyCurrentHonestMessage asserts CLM-005: when a pack is genuinely
// installed and current, a second `pack add` prints a clear already-installed message and
// exits 0 — never silent, never an error.
func TestPackAddCLI_AlreadyCurrentHonestMessage(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, projectDir, "backstop.yml", "packs: {}")
	writeLocalPackSource(t, parent, "local-src", "internal/local-rules")

	restore := chdirForTest(t, projectDir)
	defer restore()

	if out, err := executeCommand(NewRootCommand(), "pack", "add", "../local-src"); err != nil {
		t.Fatalf("first pack add: %v (out: %s)", err, out)
	}

	out, err := executeCommand(NewRootCommand(), "pack", "add", "../local-src")
	if err != nil {
		t.Fatalf("second pack add (already current) should exit 0, got error: %v (out: %s)", err, out)
	}
	if !strings.Contains(strings.ToLower(out), "already installed") {
		t.Errorf("expected an honest already-installed message, got: %q", out)
	}
}

// TestPackAddCLI_DeclaredButAbsentPrintsAdded asserts CLM-005: a pack DECLARED in
// backstop.yml but not materialized on disk installs and prints a real, non-empty Added
// line (not a silent no-op, not an error).
func TestPackAddCLI_DeclaredButAbsentPrintsAdded(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	packName := "internal/local-rules"
	writeFileForTest(t, projectDir, "backstop.yml", "packs:\n  "+packName+": local")
	writeLocalPackSource(t, parent, "local-src", packName)

	restore := chdirForTest(t, projectDir)
	defer restore()

	out, err := executeCommand(NewRootCommand(), "pack", "add", "../local-src")
	if err != nil {
		t.Fatalf("declared-but-absent pack add should install, got error: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, "Added") {
		t.Errorf("expected a real Added line, got: %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", packName, "pack.yml")); statErr != nil {
		t.Errorf("pack not materialized on disk: %v", statErr)
	}
}

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
