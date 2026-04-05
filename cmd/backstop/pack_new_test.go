package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// packNewTestCase holds inputs for running the pack new command.
type packNewTestCase struct {
	args       []string
	projectDir string
}

// runPackNewTest executes the pack new command with the given args in a temp
// project dir. Returns exit code and combined stdout/stderr output.
func runPackNewTest(t *testing.T, tc packNewTestCase) (int, string) {
	t.Helper()

	projectDir := tc.projectDir
	if projectDir == "" {
		projectDir = t.TempDir()
	}

	cmd := newPackNewCommandWithRoot(projectDir)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(tc.args)

	err := cmd.Execute()
	exitCode := 0
	if err != nil {
		if ece, ok := err.(*ExitCodeError); ok {
			exitCode = ece.Code
		} else {
			exitCode = 2 // default for pack new — no exit 1
		}
	}

	return exitCode, buf.String()
}

// --- Thin adapter tests (REQ-010) ---

func TestPackNew_ThinAdapter_DelegatesScaffolding(t *testing.T) {
	tmpDir := t.TempDir()
	code, _ := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "rule", "--language", "go", "--slug", "error-handling"},
		projectDir: tmpDir,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	// Verify file was created via ScaffoldPack delegation
	matches, err := filepath.Glob(filepath.Join(tmpDir, "standards", "go", "STD-GO-*.standard.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 standard file created via ScaffoldPack, got %d", len(matches))
	}
}

func TestPackNew_ThinAdapter_DelegatesNumberResolution(t *testing.T) {
	tmpDir := t.TempDir()
	// Seed two existing standards so number resolution returns 3
	stdDir := filepath.Join(tmpDir, "standards", "go")
	if err := os.MkdirAll(stdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"STD-GO-001-first.standard.md", "STD-GO-002-second.standard.md"} {
		if err := os.WriteFile(filepath.Join(stdDir, n), []byte("---\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	code, _ := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "rule", "--language", "go", "--slug", "third"},
		projectDir: tmpDir,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	// Verify the standard file uses number 003 (from ResolvePackNumber)
	expectedFile := filepath.Join(stdDir, "STD-GO-003-third.standard.md")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("expected STD-GO-003 file from ResolvePackNumber delegation, got error: %v", err)
	}
}

// --- Exit code tests (REQ-008) ---

func TestPackNew_ExitCode_0_RulePackSuccess(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "rule", "--language", "go", "--slug", "error-handling"},
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestPackNew_ExitCode_0_CodePackSuccess(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "code", "--language", "go", "--slug", "error-handling"},
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestPackNew_ExitCode_2_InvalidType(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "bogus", "--language", "go", "--slug", "my-pack"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestPackNew_ExitCode_2_InvalidLanguage(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "rule", "--language", "Go", "--slug", "my-pack"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestPackNew_ExitCode_2_InvalidSlug(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "rule", "--language", "go", "--slug", "1bad"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestPackNew_ExitCode_2_MissingFlags(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

// --- JSON output test from command level ---

func TestPackNew_Command_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	code, output := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "rule", "--language", "go", "--slug", "error-handling", "--json"},
		projectDir: tmpDir,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &m); err != nil {
		t.Fatalf("JSON output is not valid JSON: %v\nOutput: %s", err, output)
	}
	for _, field := range []string{"type", "language", "slug", "paths", "schema_version", "pack_id"} {
		if _, ok := m[field]; !ok {
			t.Errorf("JSON output missing field %q", field)
		}
	}
}

func TestPackNew_Command_HumanOutput(t *testing.T) {
	tmpDir := t.TempDir()
	code, output := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "rule", "--language", "go", "--slug", "error-handling"},
		projectDir: tmpDir,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(output, "Created") {
		t.Error("human output should contain 'Created'")
	}
	if !strings.Contains(output, "STD-GO-001") {
		t.Error("human output should contain pack identifier")
	}
}
