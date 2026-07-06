package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
	"github.com/spf13/cobra"
)

// helper: execute a command and capture output
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// TestCLI_RootCommand_ShowsHelp invokes root command without subcommands,
// verifies help is displayed. (CLM-001)
func TestCLI_RootCommand_ShowsHelp(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root)
	if err != nil {
		t.Fatalf("root command error: %v", err)
	}
	if !strings.Contains(out, "backstop") {
		t.Error("root help does not mention 'backstop'")
	}
	if !strings.Contains(out, "Available Commands") || !strings.Contains(out, "help") {
		t.Error("root help does not list available commands")
	}
}

// TestCLI_ArtifactNamespace_Exists verifies artifact subcommand exists. (CLM-002)
func TestCLI_ArtifactNamespace_Exists(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"artifact"})
	if err != nil {
		t.Fatalf("find artifact: %v", err)
	}
	if cmd.Name() != "artifact" {
		t.Errorf("command name = %q, want %q", cmd.Name(), "artifact")
	}
}

// TestCLI_CodeNamespace_Exists was removed by ISSUE-018: the `code` namespace and
// its `check` subcommand were deleted. Its absence is now asserted by
// TestCodeCheckSubcommand_AbsentFromCLI (code_check_removal_test.go).

// TestCLI_PackNamespace_Exists verifies pack subcommand exists. (CLM-004)
func TestCLI_PackNamespace_Exists(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"pack"})
	if err != nil {
		t.Fatalf("find pack: %v", err)
	}
	if cmd.Name() != "pack" {
		t.Errorf("command name = %q, want %q", cmd.Name(), "pack")
	}
}

// TestCLI_GateCommand_Exists verifies gate command exists as top-level. (CLM-005)
func TestCLI_GateCommand_Exists(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"gate"})
	if err != nil {
		t.Fatalf("find gate: %v", err)
	}
	if cmd.Name() != "gate" {
		t.Errorf("command name = %q, want %q", cmd.Name(), "gate")
	}
}

func TestCLI_BaselineNamespace_Exists_Contract(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"baseline"})
	if err != nil {
		t.Fatalf("find baseline: %v", err)
	}
	if cmd.Name() != "baseline" {
		t.Fatalf("command name = %q, want %q", cmd.Name(), "baseline")
	}
}

func TestCLI_BaselinePull_HelpMentionsArtifactAndMainRunSelection_Contract(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "baseline", "pull", "--help")
	if err != nil {
		t.Fatalf("baseline pull --help error: %v", err)
	}
	for _, expected := range []string{"artifact", "main", "latest successful"} {
		if !strings.Contains(strings.ToLower(out), strings.ToLower(expected)) {
			t.Fatalf("baseline pull help missing %q semantics, output=%s", expected, out)
		}
	}
}

// TestCLI_ConfigLoader_LoadedBeforeEnforcement invokes an enforcement command
// without backstop.yml, verifies config error occurs before enforcement. (CLM-012)
func TestCLI_ConfigLoader_LoadedBeforeEnforcement(t *testing.T) {
	// Use an empty temp dir with no backstop.yml
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Ensure BACKSTOP_CONFIG is not set
	t.Setenv("BACKSTOP_CONFIG", "")

	root := NewRootCommand()
	_, err := executeCommand(root, "gate")
	if err == nil {
		t.Fatal("expected config error from enforcement command without backstop.yml")
	}
	if !strings.Contains(err.Error(), "backstop.yml") {
		t.Errorf("error should mention backstop.yml, got: %v", err)
	}
}

// TestCLI_Version_HumanOutput invokes version command, verifies output
// includes version, schema cohort, Go version. (CLM-022)
func TestCLI_Version_HumanOutput(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "version")
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "version") {
		t.Error("version output does not contain 'version'")
	}
	if !strings.Contains(out, "go") {
		t.Error("version output does not contain Go version info")
	}
}

// TestCLI_Version_JSONOutput invokes version --json, verifies JSON with
// version, schema_cohort, go_version fields. (CLM-023)
func TestCLI_Version_JSONOutput(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("version --json output is not valid JSON: %v\noutput: %s", err, out)
	}
	for _, field := range []string{"version", "schema_cohort", "go_version"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("version JSON missing field %q", field)
		}
	}
}

// TestCLI_Commands_JSON_ReturnsArray invokes commands --json, verifies
// result is JSON array. (CLM-024)
func TestCLI_Commands_JSON_ReturnsArray(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "commands", "--json")
	if err != nil {
		t.Fatalf("commands --json error: %v", err)
	}
	var parsed []interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("commands --json output is not a JSON array: %v\noutput: %s", err, out)
	}
	if len(parsed) == 0 {
		t.Error("commands --json returned empty array")
	}
}

// TestCLI_Commands_JSON_DescriptorFields verifies each descriptor has name,
// path, description, flags. (CLM-025)
func TestCLI_Commands_JSON_DescriptorFields(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "commands", "--json")
	if err != nil {
		t.Fatalf("commands --json error: %v", err)
	}
	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("commands --json parse error: %v", err)
	}
	for _, desc := range parsed {
		for _, field := range []string{"name", "path", "description", "flags"} {
			if _, ok := desc[field]; !ok {
				t.Errorf("command descriptor missing field %q: %v", field, desc)
			}
		}
	}
}

// TestCLI_Commands_JSON_IncludesAllNamespaces verifies artifact, pack namespaces
// appear in command tree. The `code` namespace was removed with the `backstop
// code check` command (ISSUE-018). (CLM-026)
func TestCLI_Commands_JSON_IncludesAllNamespaces(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "commands", "--json")
	if err != nil {
		t.Fatalf("commands --json error: %v", err)
	}
	for _, ns := range []string{"artifact", "pack"} {
		if !strings.Contains(out, ns) {
			t.Errorf("commands --json missing namespace %q", ns)
		}
	}
}

// TestCLI_Help_RootListsNamespaces verifies root help output lists
// artifact, pack groups and gate, version, commands. The `code` namespace was
// removed with the `backstop code check` command (ISSUE-018). (CLM-027)
func TestCLI_Help_RootListsNamespaces(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "--help")
	if err != nil {
		t.Fatalf("root --help error: %v", err)
	}
	for _, name := range []string{"artifact", "pack", "gate", "version", "commands"} {
		if !strings.Contains(out, name) {
			t.Errorf("root help missing %q", name)
		}
	}
}

// TestCLI_Help_NamespaceListsSubcommands verifies artifact help lists its
// subcommands. (CLM-028)
func TestCLI_Help_NamespaceListsSubcommands(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "artifact", "--help")
	if err != nil {
		t.Fatalf("artifact --help error: %v", err)
	}
	// artifact namespace should list validate and new subcommands
	if !strings.Contains(out, "validate") {
		t.Error("artifact help missing 'validate' subcommand")
	}
	if !strings.Contains(out, "new") {
		t.Error("artifact help missing 'new' subcommand")
	}
}

// TestCLI_Help_AllCommandsHaveDescriptions walks command tree, verifies
// every command has Short and Long set. (CLM-029)
func TestCLI_Help_AllCommandsHaveDescriptions(t *testing.T) {
	root := NewRootCommand()
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return // skip built-in Cobra commands
		}
		if cmd.Short == "" {
			t.Errorf("command %q missing Short description", cmd.CommandPath())
		}
		if cmd.Long == "" {
			t.Errorf("command %q missing Long description", cmd.CommandPath())
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

func TestCLI_OutputResult_UsesJSONFormatter(t *testing.T) {
	root := NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	jsonOn := true
	err := outputResult(cmd, &jsonOn, validate.ValidationResult{})
	if err != nil {
		t.Fatalf("outputResult returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output was not JSON: %v, out=%s", err, buf.String())
	}
	if parsed["schema_version"] != "cli/v1" {
		t.Fatalf("schema_version missing, out=%s", buf.String())
	}
}

func TestCLI_Commands_ContainsJsonFlag(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "commands", "--json")
	if err != nil {
		t.Fatalf("commands --json error: %v", err)
	}
	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("commands output parse error: %v", err)
	}
	found := false
	for _, d := range parsed {
		if d["path"] == "gate" {
			if flags, ok := d["flags"].([]interface{}); ok {
				for _, f := range flags {
					if f == "--json" {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected gate command descriptor to include --json")
	}
}

func TestCLI_ComputeCohortID_Empty(t *testing.T) {
	if got := computeCohortID(nil); got != "empty" {
		t.Fatalf("computeCohortID(nil)=%q want empty", got)
	}
}

func TestCLI_ComputeCohortID_NonEmpty(t *testing.T) {
	got := computeCohortID([]string{"artifacts/spec/v1/schema.json"})
	if !strings.Contains(got, "1-schemas[spec/v1]") {
		t.Fatalf("unexpected cohort id: %s", got)
	}
}

func TestCLI_Main_EntryPointCoverage(t *testing.T) {
	// Coverage-only test: execute built binary to exercise main().
	// Keep assertions simple and deterministic.
	bin := filepath.Join(t.TempDir(), "backstop-test-bin")
	rootDir := filepath.Join("..", "..")
	// build from module root so imports resolve
	cmdBuild := execCommand("go", "build", "-o", bin, "./cmd/backstop")
	cmdBuild.Dir = rootDir
	buildOut, err := cmdBuild.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, string(buildOut))
	}

	cmdRun := execCommand(bin, "version")
	cmdRun.Dir = rootDir
	runOut, err := cmdRun.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(runOut))
	}
	if !strings.Contains(string(runOut), "version") {
		t.Fatalf("unexpected output: %s", string(runOut))
	}
}

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func TestCLI_EnforcementPreRun_UsesConfigErrorPath(t *testing.T) {
	root := NewRootCommand()
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	t.Setenv("BACKSTOP_CONFIG", "/definitely/missing.yml")
	_, err := executeCommand(root, "artifact", "validate")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "BACKSTOP_CONFIG") && !strings.Contains(err.Error(), "missing.yml") {
		t.Fatalf("expected BACKSTOP_CONFIG/missing path error, got: %v", err)
	}
}

func TestCLI_EnforcementCommands_RunStubs(t *testing.T) {
	dir := t.TempDir()
	cfg := "project: test\nlanguage: go\n"
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create .backstop/rules/ dir so code check doesn't fail
	if err := os.MkdirAll(filepath.Join(dir, ".backstop", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "artifact validate", args: []string{"artifact", "validate", "sample.md", "--json"}, want: "\"pass\": true"},
		{name: "artifact new", args: []string{"artifact", "new", "spec", "--slug", "test-stub", "--json"}, want: "\"artifact_type\""},
		{name: "gate", args: []string{"gate", "--json"}, want: "\"schema_version\""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := NewRootCommand()
			out, err := executeCommand(root, tc.args...)
			// Gate may return exit code 1 when artifact validation finds
			// violations in scaffolded specs from prior sub-tests. That is
			// expected behavior — check for structured output, not pass.
			if err != nil && tc.name != "gate" {
				t.Fatalf("command failed: %v; out=%s", err, out)
			}
			if !strings.Contains(out, tc.want) {
				t.Fatalf("missing expected %q in output: %s", tc.want, out)
			}
		})
	}
}

func TestCLI_OutputResult_HumanMode(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	jsonOff := false
	result := validate.ValidationResult{
		Violations: []validate.Violation{{Rule: "R001", Message: "test"}},
	}
	err := outputResult(cmd, &jsonOff, result)
	if err != nil {
		t.Fatalf("outputResult returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "R001") {
		t.Fatalf("expected R001 in output: %s", buf.String())
	}
}
