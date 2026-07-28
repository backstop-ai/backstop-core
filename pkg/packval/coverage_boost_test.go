package packval

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
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

// TestPackVal_DefaultExecutor_RunEngine_CleanRunNoFindings exercises the generic
// engine dispatch when the (un-provisioned) command runs cleanly but emits no
// findings: RunEngine reports Passed=false (did not fire) with no error — the
// negative-fixture-clean path.
func TestPackVal_DefaultExecutor_RunEngine_CleanRunNoFindings(t *testing.T) {
	d := &DefaultExecutor{}
	binding := engine.EngineBinding{Command: "true", InputMode: engine.InputModeNone}
	result, err := d.RunEngine(t.TempDir(), binding, nil)
	if err != nil {
		t.Fatalf("a clean run with no findings must not error: %v", err)
	}
	if result.Passed {
		t.Error("no findings must read as did-not-fire (Passed=false)")
	}
}

// TestPackVal_DefaultExecutor_RunEngine_BadSarif surfaces a parse failure when the
// engine emits non-SARIF bytes rather than reading it as a silent finding-free pass.
func TestPackVal_DefaultExecutor_RunEngine_BadSarif(t *testing.T) {
	d := &DefaultExecutor{}
	binding := engine.EngineBinding{Command: "printf not-sarif", InputMode: engine.InputModeNone}
	result, err := d.RunEngine(t.TempDir(), binding, nil)
	if err == nil {
		t.Fatal("expected a parse error when the engine emits non-SARIF output")
	}
	if result.Passed {
		t.Error("unparseable output must not read as a fired result")
	}
}

// --- MockExecutor nil fn branches ---

func TestPackVal_MockExecutor_NilFunctions(t *testing.T) {
	m := &MockExecutor{}

	// All nil fns should return default pass result
	r1, err := m.RunEngine("", engine.EngineBinding{}, nil)
	if err != nil || !r1.Passed {
		t.Error("nil EngineFn should return pass")
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

func TestPackVal_DefaultExecutor_RunEngine_MissingBinary(t *testing.T) {
	// A binding whose command names a non-existent binary: cmd.Run fails, stdout is
	// empty, so there is no SARIF to parse — RunEngine reports the failure loud.
	d := &DefaultExecutor{}
	binding := engine.EngineBinding{Command: "definitely-not-a-real-binary-xyz --go", InputMode: engine.InputModeNone}
	result, err := d.RunEngine(t.TempDir(), binding, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing binary with no SARIF output")
	}
	if result.Passed {
		t.Error("missing binary must not read as a fired result")
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
