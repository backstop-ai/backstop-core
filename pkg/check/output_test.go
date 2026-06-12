package check

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// TestCodeCheck_Output_JSONWithSchemaVersion verifies JSON output includes
// schema_version field. (CLM-022)
func TestCodeCheck_Output_JSONWithSchemaVersion(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{{Pass: CheckTypeLint, File: "a.go", Message: "lint-err", Severity: "error"}}},
		},
	}

	out, err := FormatResult(result, OutputModeJSON)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(out), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}

	sv, ok := parsed["schema_version"]
	if !ok {
		t.Error("JSON output missing schema_version field")
	}
	if sv == "" {
		t.Error("schema_version is empty")
	}
}

// TestCodeCheck_Output_JSONIncludesRule verifies that a semgrep Violation
// carrying a pack-namespaced Rule survives the JSON output bridge: the
// JSONViolation carries the rule in a `rule` field so the namespaced ID is no
// longer dropped from `backstop code check` output. (CLM-006)
func TestCodeCheck_Output_JSONIncludesRule(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeSemgrep, Violations: []Violation{
				{
					Pass:     CheckTypeSemgrep,
					File:     "pkg/server/handler.go",
					Line:     31,
					Message:  "panic() is disallowed",
					Severity: "error",
					Rule:     "slotly/go-standards/no-panic",
				},
			}},
		},
	}

	out, err := FormatResult(result, OutputModeJSON)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	var parsed JSONOutput
	if jsonErr := json.Unmarshal([]byte(out), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if len(parsed.Violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(parsed.Violations))
	}
	if got := parsed.Violations[0].Rule; got != "slotly/go-standards/no-panic" {
		t.Errorf("JSONViolation.Rule = %q, want the pack-namespaced ID", got)
	}

	// The raw JSON must use the `rule` key.
	if !strings.Contains(out, `"rule": "slotly/go-standards/no-panic"`) {
		t.Errorf("JSON output missing `rule` field for the namespaced ID:\n%s", out)
	}
}

// TestCodeCheck_Output_IdenticalViolationData verifies JSON and human output
// contain identical violation data. (CLM-023)
func TestCodeCheck_Output_IdenticalViolationData(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{
				{Pass: CheckTypeLint, File: "a.go", Line: 10, Message: "unused variable", Severity: "warning"},
			}},
			{Pass: CheckTypeBuild},
		},
	}

	jsonOut, err := FormatResult(result, OutputModeJSON)
	if err != nil {
		t.Fatalf("JSON format: %v", err)
	}
	humanOut, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("Human format: %v", err)
	}

	// Parse JSON to extract violation data
	var parsed JSONOutput
	if jsonErr := json.Unmarshal([]byte(jsonOut), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}

	if len(parsed.Violations) != 1 {
		t.Errorf("JSON has %d violations, want 1", len(parsed.Violations))
	}

	// Human output should mention the same violation
	if !strings.Contains(humanOut, "unused variable") {
		t.Error("human output does not contain violation message")
	}
	if !strings.Contains(humanOut, "a.go") {
		t.Error("human output does not contain violation file")
	}
}

// TestCodeCheck_Output_NoColorRespected verifies NO_COLOR disables ANSI codes
// in human output. (CLM-024)
func TestCodeCheck_Output_NoColorRespected(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{
				{Pass: CheckTypeLint, File: "a.go", Message: "err", Severity: "error"},
			}},
		},
	}

	// Set NO_COLOR
	t.Setenv("NO_COLOR", "1")

	out, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	if strings.Contains(out, "\033[") {
		t.Error("human output contains ANSI escape codes despite NO_COLOR being set")
	}
}

// TestCodeCheck_ExitCode_0OnPass verifies exit code 0 when all checks pass. (CLM-025)
func TestCodeCheck_ExitCode_0OnPass(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint},
			{Pass: CheckTypeBuild},
		},
	}
	code := DetermineExitCode(result, nil, false)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

// TestCodeCheck_ExitCode_1OnViolations verifies exit code 1 with violations. (CLM-026)
func TestCodeCheck_ExitCode_1OnViolations(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{{Pass: CheckTypeLint, Message: "err"}}},
		},
	}
	code := DetermineExitCode(result, nil, false)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

// TestCodeCheck_ExitCode_2OnConfigError verifies exit code 2 on config error. (CLM-027)
func TestCodeCheck_ExitCode_2OnConfigError(t *testing.T) {
	result := &Result{}
	code := DetermineExitCode(result, &ConfigError{Message: "bad config"}, false)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

// TestCodeCheck_ExitCode_2PrecedesOver1 verifies exit code 2 takes precedence
// over exit code 1. (CLM-028)
func TestCodeCheck_ExitCode_2PrecedesOver1(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{{Pass: CheckTypeLint, Message: "err"}}},
		},
	}
	code := DetermineExitCode(result, &ConfigError{Message: "bad config"}, false)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 (config error takes precedence)", code)
	}
}

// TestCodeCheck_ExitCode_2OnFlagConflict verifies exit code 2 when --file + --all. (CLM-029)
func TestCodeCheck_ExitCode_2OnFlagConflict(t *testing.T) {
	result := &Result{}
	code := DetermineExitCode(result, nil, true)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 (flag conflict)", code)
	}
}

// TestCodeCheck_ExitCode_2OnMissingBackstopDir verifies exit code 2 when
// .backstop/ directory is missing. (CLM-030)
func TestCodeCheck_ExitCode_2OnMissingBackstopDir(t *testing.T) {
	dir := t.TempDir() // no .backstop/ directory

	err := ValidateBackstopDir(dir)
	if err == nil {
		t.Fatal("expected error for missing .backstop dir, got nil")
	}

	var cfgErr *ConfigError
	if !asConfigError(err, &cfgErr) {
		t.Errorf("expected ConfigError, got %T: %v", err, err)
	}
}

// TestCodeCheck_Config_LoadedBeforeChecks verifies config is loaded before
// any pass executes. When .backstop/ is missing (config invalid), no passes
// should execute. (CLM-034)
func TestCodeCheck_Config_LoadedBeforeChecks(t *testing.T) {
	// Verify that ValidateBackstopDir returns ConfigError when .backstop/ is
	// missing, which prevents pass execution in the cmd layer.
	dir := t.TempDir() // no .backstop/ directory

	err := ValidateBackstopDir(dir)
	if err == nil {
		t.Fatal("expected ConfigError for missing .backstop dir")
	}

	var cfgErr *ConfigError
	if !asConfigError(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}

	// Verify this produces exit code 2 (config error blocks execution)
	exitCode := DetermineExitCode(nil, err, false)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for config error, got %d", exitCode)
	}

	// Now verify that with a valid .backstop/ dir, passes DO execute
	validDir := t.TempDir()
	if mkErr := os.MkdirAll(filepath.Join(validDir, ".backstop", "rules"), 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}

	validErr := ValidateBackstopDir(validDir)
	if validErr != nil {
		t.Fatalf("expected no error for valid .backstop dir, got: %v", validErr)
	}

	// Add a Go file so scope resolution finds something
	if err := os.WriteFile(filepath.Join(validDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	invoked := false
	executors := map[CheckType]PassExecutor{
		CheckTypeLint: &mockPassExecutor{fn: func(_ context.Context, _ []string) (*PassResult, error) {
			invoked = true
			return &PassResult{Pass: CheckTypeLint}, nil
		}},
	}

	opts := RunOptions{
		Options: Options{
			Mode:       ScopeModeAll,
			ProjectDir: validDir,
		},
		Git:            &mockGitExecutor{isGitRepo: false},
		Executors:      executors,
		SemgrepEnsurer: &mockSemgrepEnsurer{},
	}

	_, runErr := RunWith(context.Background(), opts)
	if runErr != nil {
		t.Fatalf("RunWith: %v", runErr)
	}

	if !invoked {
		t.Error("expected pass executor to be invoked when config is valid")
	}
}

// TestCodeCheck_Config_MissingYmlExitCode2 verifies missing backstop.yml
// produces exit code 2 through the real ValidateBackstopDir path. (CLM-035)
func TestCodeCheck_Config_MissingYmlExitCode2(t *testing.T) {
	dir := t.TempDir() // no .backstop/ directory, no backstop.yml

	// ValidateBackstopDir should return ConfigError
	err := ValidateBackstopDir(dir)
	if err == nil {
		t.Fatal("expected error for missing .backstop dir")
	}

	var cfgErr *ConfigError
	if !asConfigError(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}

	// DetermineExitCode with this ConfigError should yield 2
	code := DetermineExitCode(&Result{}, err, false)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

// TestCodeCheck_Output_HumanPassingNoViolations verifies human output for
// passing result with no violations.
func TestCodeCheck_Output_HumanPassingNoViolations(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint},
			{Pass: CheckTypeBuild},
		},
	}

	out, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("human output missing pass message: %s", out)
	}
}

// TestCodeCheck_Output_HumanWithWarnings verifies human output includes warnings.
func TestCodeCheck_Output_HumanWithWarnings(t *testing.T) {
	result := &Result{
		Warnings: []string{"golangci-lint not found"},
	}

	out, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if !strings.Contains(out, "golangci-lint not found") {
		t.Errorf("human output missing warning: %s", out)
	}
}

// TestCodeCheck_Output_HumanWithColor verifies human output uses ANSI
// codes when NO_COLOR is not set.
func TestCodeCheck_Output_HumanWithColor(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{
				{Pass: CheckTypeLint, File: "a.go", Message: "err", Severity: "error"},
			}},
		},
	}

	// Unset NO_COLOR entirely — color should be enabled
	os.Unsetenv("NO_COLOR")

	out, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	// With NO_COLOR unset, output should have ANSI codes
	if !strings.Contains(out, "\033[") {
		t.Error("expected ANSI codes in colored output")
	}
}

// TestCodeCheck_Output_NoColorEmptyStringDisables verifies that NO_COLOR
// set to empty string still disables color per the spec.
func TestCodeCheck_Output_NoColorEmptyStringDisables(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{
				{Pass: CheckTypeLint, File: "a.go", Message: "err", Severity: "error"},
			}},
		},
	}

	// NO_COLOR="" (empty) should still disable color per spec
	t.Setenv("NO_COLOR", "")

	out, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if strings.Contains(out, "\033[") {
		t.Error("NO_COLOR='' should disable color, but ANSI codes found")
	}
}

// TestCodeCheck_Output_HumanSkippedPasses verifies skipped passes appear in output.
func TestCodeCheck_Output_HumanSkippedPasses(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Skipped: true, SkipReason: "tool not found"},
		},
	}

	out, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("human output missing skip info: %s", out)
	}
}

// TestCodeCheck_Output_InvalidMode verifies error for unknown output mode.
func TestCodeCheck_Output_InvalidMode(t *testing.T) {
	result := &Result{}
	_, err := FormatResult(result, OutputMode(99))
	if err == nil {
		t.Error("expected error for invalid output mode")
	}
}

// TestCodeCheck_Output_JSONNilWarnings verifies JSON output serializes nil
// warnings as empty array, not null.
func TestCodeCheck_Output_JSONNilWarnings(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint},
		},
		Warnings: nil,
	}

	out, err := FormatResult(result, OutputModeJSON)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}

	var parsed JSONOutput
	if jsonErr := json.Unmarshal([]byte(out), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if parsed.Warnings == nil {
		t.Error("warnings should be empty array, not null")
	}
	if len(parsed.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(parsed.Warnings))
	}
}

// TestCodeCheck_Output_HumanViolationNoFile verifies human output handles
// violations with empty file field.
func TestCodeCheck_Output_HumanViolationNoFile(t *testing.T) {
	result := &Result{
		PassResults: []PassResult{
			{Pass: CheckTypeLint, Violations: []Violation{
				{Pass: CheckTypeLint, Message: "general lint error", Severity: "warning"},
			}},
		},
	}

	t.Setenv("NO_COLOR", "1")
	out, err := FormatResult(result, OutputModeHuman)
	if err != nil {
		t.Fatalf("FormatResult: %v", err)
	}
	if !strings.Contains(out, "(no file)") {
		t.Errorf("expected (no file) placeholder, got: %s", out)
	}
	if !strings.Contains(out, "general lint error") {
		t.Errorf("expected violation message in output, got: %s", out)
	}
}

// TestCodeCheck_Config_EnvVarOverride verifies BACKSTOP_CONFIG env var
// overrides walk-up discovery. (CLM-036)
func TestCodeCheck_Config_EnvVarOverride(t *testing.T) {
	// Create a backstop.yml in a non-standard location
	customDir := t.TempDir()
	customConfig := filepath.Join(customDir, "backstop.yml")
	if err := os.WriteFile(customConfig, []byte("project: custom\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a different backstop.yml in a "normal" location that walk-up would find
	normalDir := t.TempDir()
	normalConfig := filepath.Join(normalDir, "backstop.yml")
	if err := os.WriteFile(normalConfig, []byte("project: normal\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set BACKSTOP_CONFIG to the custom path
	t.Setenv("BACKSTOP_CONFIG", customConfig)

	// Call the real config discovery function — it should use the env var
	// override, not walk-up from normalDir
	discoveredPath, err := config.DiscoverConfigPathFrom(normalDir)
	if err != nil {
		t.Fatalf("DiscoverConfigPathFrom: %v", err)
	}

	// The discovered path should be the custom one, not the normal one
	if discoveredPath != customConfig {
		t.Errorf("expected config path %q (from BACKSTOP_CONFIG), got %q", customConfig, discoveredPath)
	}
}
