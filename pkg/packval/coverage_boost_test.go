package packval

import (
	"os/exec"
	"strings"
	"testing"
)

// --- resultFromCmd coverage ---

func TestPackVal_ResultFromCmd_NilError(t *testing.T) {
	r := resultFromCmd([]byte("all good"), nil)
	if !r.Passed {
		t.Error("expected Passed=true for nil error")
	}
	if r.ExitCode != 0 {
		t.Errorf("expected ExitCode=0, got %d", r.ExitCode)
	}
	if r.Output != "all good" {
		t.Errorf("expected output 'all good', got %q", r.Output)
	}
}

func TestPackVal_ResultFromCmd_ExitError(t *testing.T) {
	// Create a real exec.ExitError by running a command that fails
	cmd := exec.Command("sh", "-c", "exit 42")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error from exit 42")
	}
	r := resultFromCmd([]byte("failed output"), err)
	if r.Passed {
		t.Error("expected Passed=false for exit error")
	}
	if r.ExitCode != 42 {
		t.Errorf("expected ExitCode=42, got %d", r.ExitCode)
	}
}

func TestPackVal_ResultFromCmd_NonExitError(t *testing.T) {
	// A non-exec.ExitError (e.g., command not found wrapping)
	err := &exec.Error{Name: "nonexistent-binary", Err: exec.ErrNotFound}
	r := resultFromCmd([]byte(""), err)
	if r.Passed {
		t.Error("expected Passed=false for non-exit error")
	}
	if r.ExitCode != 1 {
		t.Errorf("expected ExitCode=1 for non-exit error, got %d", r.ExitCode)
	}
}

// --- FormatResult coverage ---

func TestPackVal_FormatResult_UnknownFormat(t *testing.T) {
	r := &Result{Status: "pass"}
	_, err := FormatResult(r, "xml")
	if err == nil {
		t.Error("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("expected 'unknown format' in error, got %q", err.Error())
	}
}

func TestPackVal_FormatResult_TextWithErrors(t *testing.T) {
	r := &Result{
		Status: "fail",
		Phases: []PhaseResult{
			{Phase: "structural", Status: "fail"},
			{Phase: "coherence", Status: "skipped", Reason: "structural failed"},
		},
		Errors: []ValidationError{
			{Phase: "structural", Check: "required-fields", Message: "missing name"},
		},
		Warnings: []ValidationWarning{
			{Phase: "coherence", Check: "orphan-fixtures", Message: "2 orphan files"},
		},
	}
	out, err := FormatResult(r, "text")
	if err != nil {
		t.Fatalf("FormatResult text: %v", err)
	}
	if !strings.Contains(out, "status: fail") {
		t.Error("text output missing status")
	}
	if !strings.Contains(out, "structural: fail") {
		t.Error("text output missing phase status")
	}
	if !strings.Contains(out, "reason: structural failed") {
		t.Error("text output missing skip reason")
	}
	if !strings.Contains(out, "ERROR [structural/required-fields]") {
		t.Error("text output missing error")
	}
	if !strings.Contains(out, "WARN [coherence/orphan-fixtures]") {
		t.Error("text output missing warning")
	}
}

func TestPackVal_FormatResult_EmptyStringDefaultsToJSON(t *testing.T) {
	r := &Result{Status: "pass", Phases: []PhaseResult{}}
	out, err := FormatResult(r, "")
	if err != nil {
		t.Fatalf("FormatResult empty: %v", err)
	}
	if !strings.Contains(out, `"status"`) {
		t.Error("empty format should default to JSON")
	}
}

func TestPackVal_FormatResult_WhitespaceDefaultsToJSON(t *testing.T) {
	r := &Result{Status: "pass", Phases: []PhaseResult{}}
	out, err := FormatResult(r, "  ")
	if err != nil {
		t.Fatalf("FormatResult whitespace: %v", err)
	}
	if !strings.Contains(out, `"status"`) {
		t.Error("whitespace format should default to JSON")
	}
}

// --- AllRules nil manifest coverage ---

func TestPackVal_AllRules_NilManifest(t *testing.T) {
	rules := AllRules(nil)
	if rules != nil {
		t.Error("expected nil rules for nil manifest")
	}
}

func TestPackVal_AllRules_NilRuleset(t *testing.T) {
	m := &PackManifest{}
	rules := AllRules(m)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for nil ruleset, got %d", len(rules))
	}
}

func TestPackVal_AllRules_WithToolConfig(t *testing.T) {
	m := &PackManifest{
		Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1"}}}},
		ToolConfig: []ToolConfigEntry{{ID: "tc1"}},
	}
	rules := AllRules(m)
	if len(rules) != 2 {
		t.Errorf("expected 2 rules (1 rule + 1 tool_config), got %d", len(rules))
	}
}

func TestPackVal_DefaultExecutor_RunToolConfig_GolangciLint(t *testing.T) {
	d := &DefaultExecutor{}
	// golangci-lint likely not installed but exercises the "golangci-lint" branch
	result, err := d.RunToolConfig(t.TempDir(), "golangci-lint", "config.yml", "fixture.go")
	if err != nil {
		t.Fatalf("RunToolConfig returned error: %v", err)
	}
	// Command will fail (no real config/fixture) but we exercised the code path
	_ = result
}

// --- MockExecutor nil fn branches ---

func TestPackVal_MockExecutor_NilFunctions(t *testing.T) {
	m := &MockExecutor{}

	// All nil fns should return default pass result
	r1, err := m.RunSemgrep("", "", "")
	if err != nil || !r1.Passed {
		t.Error("nil SemgrepFn should return pass")
	}

	r2, err := m.RunToolConfig("", "", "", "")
	if err != nil || !r2.Passed {
		t.Error("nil ToolConfigFn should return pass")
	}

	r3, err := m.RunValidator("", "", nil)
	if err != nil || !r3.Passed {
		t.Error("nil ValidatorFn should return pass")
	}

	r4, err := m.RunScaffoldTest("", "", "")
	if err != nil || !r4.Passed {
		t.Error("nil ScaffoldTestFn should return pass")
	}
}

// --- DefaultExecutor coverage (exercise error paths without real tools) ---

func TestPackVal_DefaultExecutor_RunToolConfig_UnsupportedTool(t *testing.T) {
	d := &DefaultExecutor{}
	_, err := d.RunToolConfig("/tmp", "unknown-tool", "config.yml", "fixture.go")
	if err == nil {
		t.Error("expected error for unsupported tool")
	}
	if !strings.Contains(err.Error(), "unsupported tool") {
		t.Errorf("expected 'unsupported tool' error, got %q", err.Error())
	}
}

func TestPackVal_DefaultExecutor_RunSemgrep_MissingBinary(t *testing.T) {
	// This will fail because semgrep likely isn't installed in test env
	// but it exercises the code path and resultFromCmd
	d := &DefaultExecutor{}
	result, err := d.RunSemgrep(t.TempDir(), "nonexistent.yml", "nonexistent.go")
	// err should be nil (resultFromCmd absorbs exec errors)
	if err != nil {
		t.Fatalf("RunSemgrep returned error: %v", err)
	}
	// The command will fail (semgrep not found or bad args) but that's OK
	// We just care that resultFromCmd handled it
	if result.Passed && result.ExitCode == 0 {
		// semgrep somehow ran — that's fine too
		return
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit for missing semgrep/bad args")
	}
}

func TestPackVal_DefaultExecutor_RunScaffoldTest_BadCommand(t *testing.T) {
	d := &DefaultExecutor{}
	dir := t.TempDir()
	result, err := d.RunScaffoldTest(dir, ".", "exit 1")
	if err != nil {
		t.Fatalf("RunScaffoldTest returned error: %v", err)
	}
	if result.Passed {
		t.Error("expected Passed=false for failing command")
	}
}

func TestPackVal_DefaultExecutor_RunValidator_BadScript(t *testing.T) {
	d := &DefaultExecutor{}
	dir := t.TempDir()
	result, err := d.RunValidator(dir, "/nonexistent/script.sh", []string{"fixture.go"})
	// RunValidator wraps SandboxedRun which may fail
	if err != nil {
		t.Fatalf("RunValidator returned error: %v", err)
	}
	if result.Passed {
		t.Error("expected Passed=false for nonexistent validator")
	}
}
