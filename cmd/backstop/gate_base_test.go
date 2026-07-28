package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// CLI-level coverage for `gate --base <rev>`.
//
// The flag exists for CI, where the working tree is PRISTINE and bare diff mode
// therefore resolves an EMPTY scope that passes every dimension. These tests pin the
// three things that make the flag trustworthy from the outside: the conflicts are
// refused as CONFIG errors (exit 2, not exit 1 — exit 1 would claim violations were
// found, which is a lie about what happened), an unresolvable base never degrades
// into a silent scope, and the resolved scope is REPORTED so a zero-file CI run is
// legible rather than an unexplained green.

// runGateCommand executes `gate` with args in dir, returning stdout, stderr and the
// exit code. It mirrors runValidateCommand (artifact_validate_helpers_test.go) —
// there is one way this repo drives a command in a test.
func runGateCommand(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	root := NewRootCommand()
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs(append([]string{"gate"}, args...))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatalf("chdir to %s: %v", dir, chdirErr)
	}
	defer func() {
		if chdirErr := os.Chdir(origDir); chdirErr != nil {
			t.Fatalf("chdir back: %v", chdirErr)
		}
	}()

	execErr := root.Execute()

	exitCode = ExitPass
	if execErr != nil {
		// Route the error through reportError, which is what main() does and the
		// SOLE printer of command diagnostics (the root command sets SilenceErrors,
		// so cobra prints nothing itself). Deriving the exit code by type-asserting
		// here instead would assert against a message the user never sees.
		exitCode = reportError(stderrBuf, execErr)
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// gitInProject runs a git command in dir, failing the test on error.
func gitInProject(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newBasedGateProject builds a git-backed project carrying a backstop.yml, one base
// commit, and a second commit adding changed.go. It returns the project root, the
// base commit SHA, and the name of the file the second commit added.
//
// The tree is left FULLY COMMITTED on purpose: that is the CI condition, and it is
// what makes the base flag's scope non-empty where bare diff mode's would be empty.
func newBasedGateProject(t *testing.T) (root string, baseSHA string, changedFile string) {
	t.Helper()

	root = t.TempDir()
	writeProjectFile(t, root, "backstop.yml", "project: base-scope-test\n")

	gitInProject(t, root, "init")
	gitInProject(t, root, "config", "user.email", "test@example.com")
	gitInProject(t, root, "config", "user.name", "Test User")
	gitInProject(t, root, "checkout", "-b", "main")

	writeProjectFile(t, root, "base.go", "package main\n")
	gitInProject(t, root, "add", ".")
	gitInProject(t, root, "commit", "-m", "commit A")
	baseSHA = gitInProject(t, root, "rev-parse", "HEAD")

	changedFile = "changed.go"
	writeProjectFile(t, root, changedFile, "package main\n\nfunc changed() {}\n")
	gitInProject(t, root, "add", ".")
	gitInProject(t, root, "commit", "-m", "commit B")

	return root, baseSHA, changedFile
}

func writeProjectFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// TestGateCommand_BaseFlagScopesToChangedFiles asserts `--base <sha>` scopes the run
// to the files changed since that commit, on a tree with nothing uncommitted.
// (CLM-010)
func TestGateCommand_BaseFlagScopesToChangedFiles(t *testing.T) {
	root, baseSHA, changedFile := newBasedGateProject(t)

	stdout, _, _ := runGateCommand(t, root, "--base", baseSHA)

	if !strings.Contains(stdout, changedFile) && !strings.Contains(stdout, "1 changed file") {
		t.Errorf("gate --base must scope to the changed file %q; stdout:\n%s", changedFile, stdout)
	}
	// base.go existed AT the base commit, so it is not part of the diff since it.
	if strings.Contains(stdout, "base.go") {
		t.Errorf("gate --base must not scope in a file that was already present at the base; stdout:\n%s", stdout)
	}
}

// TestGateCommand_BaseWithAllIsConfigError asserts --base and --all are refused
// together, mirroring the existing --all/--file mutual exclusion. (CLM-008)
func TestGateCommand_BaseWithAllIsConfigError(t *testing.T) {
	root, baseSHA, _ := newBasedGateProject(t)

	stdout, stderr, exitCode := runGateCommand(t, root, "--base", baseSHA, "--all")

	if exitCode != ExitConfigError {
		t.Errorf("gate --base with --all must exit %d (config error), got %d", ExitConfigError, exitCode)
	}
	combined := stdout + stderr
	for _, flag := range []string{"--base", "--all"} {
		if !strings.Contains(combined, flag) {
			t.Errorf("the refusal must name the conflicting flag %q; output:\n%s", flag, combined)
		}
	}
}

// TestGateCommand_BaseWithFileIsConfigError is the same refusal for --file. It is a
// separate test rather than a table row because the two flags reach the check by
// different routes and a shared row would hide one of them regressing. (CLM-008)
func TestGateCommand_BaseWithFileIsConfigError(t *testing.T) {
	root, baseSHA, changedFile := newBasedGateProject(t)

	stdout, stderr, exitCode := runGateCommand(t, root, "--base", baseSHA, "--file", changedFile)

	if exitCode != ExitConfigError {
		t.Errorf("gate --base with --file must exit %d (config error), got %d", ExitConfigError, exitCode)
	}
	combined := stdout + stderr
	for _, flag := range []string{"--base", "--file"} {
		if !strings.Contains(combined, flag) {
			t.Errorf("the refusal must name the conflicting flag %q; output:\n%s", flag, combined)
		}
	}
}

// TestGateCommand_UnresolvableBaseExitsTwo asserts an unresolvable base is a CONFIG
// error naming the ref.
//
// The exit CODE is the assertion that matters. Exit 1 means "violations were found",
// which would be a lie about what happened and would send someone hunting for a
// finding that does not exist. (CLM-008)
func TestGateCommand_UnresolvableBaseExitsTwo(t *testing.T) {
	root, _, _ := newBasedGateProject(t)

	const missingRef = "no-such-ref-anywhere"
	stdout, stderr, exitCode := runGateCommand(t, root, "--base", missingRef)

	if exitCode != ExitConfigError {
		t.Errorf("an unresolvable --base must exit %d (config error), not %d — exit %d would claim violations were found",
			ExitConfigError, exitCode, ExitViolations)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, missingRef) {
		t.Errorf("the error must NAME the unresolvable ref %q; output:\n%s", missingRef, combined)
	}
}

// TestGateCommand_ReportsScopeModeBaseAndFileCount asserts the HUMAN output names
// the scope mode, the requested base, the resolved merge-base SHA and the in-scope
// file count.
//
// This is what makes an empty CI scope visible instead of silent — the difference
// between a green someone can audit and an unexplained one. (CLM-010)
func TestGateCommand_ReportsScopeModeBaseAndFileCount(t *testing.T) {
	root, baseSHA, _ := newBasedGateProject(t)
	// With base==A and HEAD==B, the merge-base IS A, so the resolved SHA is baseSHA.
	mergeBase := baseSHA

	stdout, _, _ := runGateCommand(t, root, "--base", baseSHA)

	for label, want := range map[string]string{
		"scope mode":         string(gateScopeModeDiffLiteral),
		"requested base":     baseSHA,
		"resolved mergebase": mergeBase,
		"in-scope count":     strconv.Itoa(1),
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("human output must report the %s (%q); stdout:\n%s", label, want, stdout)
		}
	}
}

// TestGateCommand_JSONReportsResolvedBaseAndFileCount asserts the same four facts in
// --json output. CI logs are read through the JSON, so this is where "green over 12
// files since <sha>" is distinguished from an unexplained green over zero. (CLM-010)
func TestGateCommand_JSONReportsResolvedBaseAndFileCount(t *testing.T) {
	root, baseSHA, changedFile := newBasedGateProject(t)

	stdout, _, _ := runGateCommand(t, root, "--json", "--base", baseSHA)

	var payload struct {
		Scope *struct {
			Mode          string   `json:"mode"`
			Files         []string `json:"files"`
			RequestedBase string   `json:"requested_base"`
			MergeBase     string   `json:"merge_base"`
		} `json:"scope"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("gate --json must emit a parseable document: %v\nstdout:\n%s", err, stdout)
	}
	if payload.Scope == nil {
		t.Fatalf("gate --json must carry a scope object; stdout:\n%s", stdout)
	}

	if payload.Scope.Mode != string(gateScopeModeDiffLiteral) {
		t.Errorf("json scope.mode = %q, want %q", payload.Scope.Mode, gateScopeModeDiffLiteral)
	}
	if payload.Scope.RequestedBase != baseSHA {
		t.Errorf("json scope.requested_base = %q, want %q", payload.Scope.RequestedBase, baseSHA)
	}
	if payload.Scope.MergeBase != baseSHA {
		t.Errorf("json scope.merge_base = %q, want the fork point %q", payload.Scope.MergeBase, baseSHA)
	}
	// The file COUNT is the fact a CI reader acts on: zero here would be the vacuous
	// green this flag exists to prevent.
	if len(payload.Scope.Files) != 1 {
		t.Errorf("json scope.files = %#v, want exactly the one changed file", payload.Scope.Files)
	}
	if len(payload.Scope.Files) == 1 && payload.Scope.Files[0] != changedFile {
		t.Errorf("json scope.files[0] = %q, want %q", payload.Scope.Files[0], changedFile)
	}
}

// gateScopeModeDiffLiteral is the serialized diff-scope mode. Declared here so the
// assertions above read against a named value rather than a bare string repeated
// across two tests.
const gateScopeModeDiffLiteral = "diff"
