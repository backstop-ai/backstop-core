package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/packval"
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

// --- Thin adapter tests ---

func TestPackNew_ThinAdapter_DelegatesScaffolding(t *testing.T) {
	tmpDir := t.TempDir()
	code, _ := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "engine", "--language", "go", "--slug", "error-handling"},
		projectDir: tmpDir,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	// Verify pack.yml was created via ScaffoldPack delegation.
	matches, err := filepath.Glob(filepath.Join(tmpDir, "error-handling", "pack.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 pack.yml created via ScaffoldPack, got %d", len(matches))
	}
}

// TestPackNew_ScaffoldPassesCheckAndTest is the Defect A integration proof: a freshly
// scaffolded engine pack passes BOTH `pack check` (phases 1,2,4,5,6) AND `pack test`
// (adds phase3) through the REAL packval pipeline — i.e. the scaffolder's engines:
// block parses under the Phase-2 fixes and the sample rule validates (CLM-001).
func TestPackNew_ScaffoldPassesCheckAndTest(t *testing.T) {
	for _, typ := range []string{"engine", "mechanism", "toolchain"} {
		tmpDir := t.TempDir()
		code, out := runPackNewTest(t, packNewTestCase{
			args:       []string{"--type", typ, "--language", "go", "--slug", "sample-check"},
			projectDir: tmpDir,
		})
		if code != 0 {
			t.Fatalf("type %s: expected exit 0, got %d (%s)", typ, code, out)
		}
		packDir := filepath.Join(tmpDir, "sample-check")
		for _, mode := range []string{"check", "test"} {
			res := packval.NewPipeline(packDir, packval.PipelineOptions{Mode: mode}).Run()
			if res.Status != "pass" {
				t.Errorf("type %s: pack %s on scaffolded pack = %q, want pass; errors=%+v", typ, mode, res.Status, res.Errors)
			}
		}
	}
}

// --- Exit code tests ---

func TestPackNew_ExitCode_0_EnginePackSuccess(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "engine", "--language", "go", "--slug", "error-handling"},
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestPackNew_ExitCode_0_ToolchainPackSuccess(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "toolchain", "--language", "go", "--slug", "error-handling"},
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestPackNew_ExitCode_2_InvalidType(t *testing.T) {
	// The retired "rule" type is now rejected (CLM-002).
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "rule", "--language", "go", "--slug", "my-pack"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2 for retired type, got %d", code)
	}
}

// TestPackNew_InvalidType_ErrorNamesLiveTypes checks the error text lists the live
// pack types (engine/mechanism/toolchain), invoking the command directly to read the
// returned ExitCodeError message (SilenceErrors keeps it off the output buffer).
func TestPackNew_InvalidType_ErrorNamesLiveTypes(t *testing.T) {
	cmd := newPackNewCommandWithRoot(t.TempDir())
	cmd.SetArgs([]string{"--type", "rule", "--language", "go", "--slug", "my-pack"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error for retired type 'rule'")
	}
	if !strings.Contains(err.Error(), "engine, mechanism, or toolchain") {
		t.Errorf("invalid-type error should list the live types, got %q", err.Error())
	}
}

func TestPackNew_ExitCode_2_InvalidLanguage(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "engine", "--language", "Go", "--slug", "my-pack"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestPackNew_ExitCode_2_InvalidSlug(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "engine", "--language", "go", "--slug", "1bad"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestPackNew_ExitCode_2_MissingFlags(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{args: []string{}})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestPackNew_ExitCode_2_MissingLanguage(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "engine", "--slug", "error-handling"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2 for missing --language, got %d", code)
	}
}

func TestPackNew_ExitCode_2_MissingSlug(t *testing.T) {
	code, _ := runPackNewTest(t, packNewTestCase{
		args: []string{"--type", "engine", "--language", "go"},
	})
	if code != 2 {
		t.Fatalf("expected exit 2 for missing --slug, got %d", code)
	}
}

// --- Output tests ---

func TestPackNew_Command_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	code, output := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "engine", "--language", "go", "--slug", "error-handling", "--json"},
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
	if m["pack_id"] != "local/error-handling" {
		t.Errorf("pack_id = %v, want local/error-handling", m["pack_id"])
	}
}

func TestPackNew_Command_HumanOutput(t *testing.T) {
	tmpDir := t.TempDir()
	code, output := runPackNewTest(t, packNewTestCase{
		args:       []string{"--type", "engine", "--language", "go", "--slug", "error-handling"},
		projectDir: tmpDir,
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(output, "Created") {
		t.Error("human output should contain 'Created'")
	}
	if !strings.Contains(output, "local/error-handling") {
		t.Error("human output should contain the pack identifier")
	}
}

func TestPackNew_PublicConstructor(t *testing.T) {
	cmd := NewPackNewCommand()
	if cmd == nil {
		t.Fatal("NewPackNewCommand returned nil")
	}
	if cmd.Name() != "new" {
		t.Errorf("expected command name 'new', got %q", cmd.Name())
	}
}
