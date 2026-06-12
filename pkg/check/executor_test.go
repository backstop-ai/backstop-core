package check

import (
	"context"
	"strings"
	"testing"
)

// findViolation returns the first violation in vs whose File and Line match,
// or nil. Used so assertions do not depend on ordering.
func findViolation(vs []Violation, file string, line int) *Violation {
	for i := range vs {
		if vs[i].File == file && vs[i].Line == line {
			return &vs[i]
		}
	}
	return nil
}

// TestCodeCheck_LintExecutor_ParsesGolangciJSON verifies that lintExecutor
// unmarshals golangci-lint `--out-format json` output and maps each Issue to a
// check.Violation with the correct File (.Pos.Filename), Line (.Pos.Line),
// Message (.Text), and Severity, tagged with CheckTypeLint. (CLM-001)
func TestCodeCheck_LintExecutor_ParsesGolangciJSON(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{"golangci-lint": []byte(fixtureGolangciLintFindings)},
		// golangci-lint exits non-zero when it finds issues; the executor must
		// still parse the JSON rather than treating the exit as a hard error.
		err: &exitError{code: 1},
	}
	e := &lintExecutor{runner: runner}

	res, err := e.Execute(context.Background(), []string{"pkg/server/handler.go", "cmd/app/main.go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Pass != CheckTypeLint {
		t.Errorf("Pass = %v, want %v", res.Pass, CheckTypeLint)
	}
	if len(res.Violations) != 3 {
		t.Fatalf("got %d violations, want 3: %+v", len(res.Violations), res.Violations)
	}

	v := findViolation(res.Violations, "pkg/server/handler.go", 37)
	if v == nil {
		t.Fatal("missing violation for pkg/server/handler.go:37")
	}
	if v.Message != "Error return value of `w.Write` is not checked" {
		t.Errorf("Message = %q, want errcheck text", v.Message)
	}
	if v.Severity != "error" {
		t.Errorf("Severity = %q, want error", v.Severity)
	}
	if v.Pass != CheckTypeLint {
		t.Errorf("violation Pass = %v, want lint", v.Pass)
	}

	if w := findViolation(res.Violations, "pkg/server/handler.go", 52); w == nil {
		t.Error("missing violation for handler.go:52")
	} else if w.Severity != "warning" {
		t.Errorf("Severity = %q, want warning", w.Severity)
	}

	if s := findViolation(res.Violations, "cmd/app/main.go", 9); s == nil {
		t.Error("missing violation for cmd/app/main.go:9")
	}
}

// TestCodeCheck_LintExecutor_CleanOutputNoViolations verifies that lintExecutor
// returns a passing PassResult with zero violations when golangci-lint reports
// no issues. (CLM-002)
func TestCodeCheck_LintExecutor_CleanOutputNoViolations(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{"golangci-lint": []byte(fixtureGolangciLintClean)},
		// clean run exits zero, no error.
	}
	e := &lintExecutor{runner: runner}

	res, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Pass != CheckTypeLint {
		t.Errorf("Pass = %v, want %v", res.Pass, CheckTypeLint)
	}
	if len(res.Violations) != 0 {
		t.Errorf("got %d violations, want 0: %+v", len(res.Violations), res.Violations)
	}

	// The runner must actually have been invoked — a clean result must come
	// from a real (faked) tool run, not from a short-circuit.
	if len(runner.calls) == 0 {
		t.Error("expected golangci-lint to be invoked")
	}
	if got := runner.lastCall().name; got != "golangci-lint" {
		t.Errorf("invoked %q, want golangci-lint", got)
	}
}

// TestCodeCheck_BuildExecutor_ParsesCompileErrors verifies that buildExecutor
// runs go build and parses `file:line:col: message` stderr lines into
// Violations with the correct File/Line/Message, ignoring the "# package"
// header and non-positional note lines. (CLM-003)
func TestCodeCheck_BuildExecutor_ParsesCompileErrors(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{"go": []byte(fixtureGoBuildErrors)},
		err:     &exitError{code: 2}, // go build exits non-zero on compile errors
	}
	e := &buildExecutor{runner: runner}

	res, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Pass != CheckTypeBuild {
		t.Errorf("Pass = %v, want %v", res.Pass, CheckTypeBuild)
	}
	if len(res.Violations) != 3 {
		t.Fatalf("got %d violations, want 3: %+v", len(res.Violations), res.Violations)
	}

	v := findViolation(res.Violations, "pkg/server/handler.go", 42)
	if v == nil {
		t.Fatal("missing violation for handler.go:42")
	}
	if v.Message != "undefined: doThing" {
		t.Errorf("Message = %q, want undefined: doThing", v.Message)
	}
	if v.Pass != CheckTypeBuild {
		t.Errorf("Pass = %v, want build", v.Pass)
	}
	if v.Severity != "error" {
		t.Errorf("Severity = %q, want error", v.Severity)
	}

	if findViolation(res.Violations, "pkg/server/handler.go", 58) == nil {
		t.Error("missing violation for handler.go:58")
	}
	if findViolation(res.Violations, "cmd/app/main.go", 7) == nil {
		t.Error("missing violation for main.go:7")
	}

	// the go build invocation should have been made
	if got := runner.lastCall().name; got != "go" {
		t.Errorf("invoked %q, want go", got)
	}
}

// TestCodeCheck_BuildExecutor_ToolCrashSurfacesError verifies that a go build
// run that fails without producing any parseable compiler errors (toolchain
// crash, internal error, unparseable output) surfaces an executor error —
// which the engine renders as a violation — rather than a clean pass. A
// silent pass here would reintroduce the vacuous enforcement this issue
// exists to kill.
func TestCodeCheck_BuildExecutor_ToolCrashSurfacesError(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{"go": []byte("go: internal error: panic during compile\n")},
		err:     &exitError{code: 2},
	}
	e := &buildExecutor{runner: runner}

	res, err := e.Execute(context.Background(), nil)
	if err == nil {
		t.Fatalf("Execute returned a clean result (%+v) for a crashed tool run with no parseable findings; want error", res)
	}
	if !strings.Contains(err.Error(), "go build") {
		t.Errorf("error %q should identify the go build run", err)
	}
}

// TestCodeCheck_TestExecutor_ToolCrashSurfacesError verifies the same
// crash-vs-findings distinction for go test: a non-zero exit with no
// parseable test failures must surface an error, not a clean pass.
func TestCodeCheck_TestExecutor_ToolCrashSurfacesError(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{"go": []byte("flag provided but not defined: -bogus\n")},
		err:     &exitError{code: 2},
	}
	e := &testExecutor{runner: runner}

	res, err := e.Execute(context.Background(), nil)
	if err == nil {
		t.Fatalf("Execute returned a clean result (%+v) for a crashed tool run with no parseable failures; want error", res)
	}
	if !strings.Contains(err.Error(), "go test") {
		t.Errorf("error %q should identify the go test run", err)
	}
}

// TestCodeCheck_TestExecutor_ParsesTestFailures verifies that testExecutor runs
// go test and parses `--- FAIL: TestName` plus the file:line from the failure
// body into Violations tagged with CheckTypeTest. (CLM-004)
func TestCodeCheck_TestExecutor_ParsesTestFailures(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{"go": []byte(fixtureGoTestFailures)},
		err:     &exitError{code: 1}, // go test exits non-zero on failure
	}
	e := &testExecutor{runner: runner}

	res, err := e.Execute(context.Background(), []string{"math_test.go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Pass != CheckTypeTest {
		t.Errorf("Pass = %v, want %v", res.Pass, CheckTypeTest)
	}
	if len(res.Violations) != 2 {
		t.Fatalf("got %d violations, want 2: %+v", len(res.Violations), res.Violations)
	}

	v := findViolation(res.Violations, "math_test.go", 14)
	if v == nil {
		t.Fatal("missing violation for math_test.go:14 (TestAdd)")
	}
	if v.Pass != CheckTypeTest {
		t.Errorf("Pass = %v, want test", v.Pass)
	}
	if v.Severity != "error" {
		t.Errorf("Severity = %q, want error", v.Severity)
	}
	if !strings.Contains(v.Message, "TestAdd") {
		t.Errorf("Message = %q, want it to name the failing test TestAdd", v.Message)
	}

	if w := findViolation(res.Violations, "math_test.go", 22); w == nil {
		t.Error("missing violation for math_test.go:22 (TestSub)")
	} else if !strings.Contains(w.Message, "TestSub") {
		t.Errorf("Message = %q, want it to name TestSub", w.Message)
	}
}

// TestCodeCheck_TestExecutor_FileModeScopesToPackage verifies that in file mode
// the executor runs go test against the scoped file's PACKAGE directory rather
// than ./..., honoring REQ-003 file-mode scoping. (CLM-004)
func TestCodeCheck_TestExecutor_FileModeScopesToPackage(t *testing.T) {
	runner := &fakeRunner{outputs: map[string][]byte{"go": []byte("ok  \tpkg/server\t0.01s\n")}}
	e := &testExecutor{runner: runner, fileMode: true}

	_, err := e.Execute(context.Background(), []string{"pkg/server/client_test.go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	call := runner.lastCall()
	if call.name != "go" {
		t.Fatalf("invoked %q, want go", call.name)
	}
	if len(call.args) < 2 || call.args[0] != "test" {
		t.Fatalf("args = %v, want a go test invocation", call.args)
	}

	// The package selector must target the file's package directory, not the
	// whole module.
	selector := call.args[len(call.args)-1]
	if selector == "./..." {
		t.Errorf("file mode ran go test ./... — must scope to the file's package")
	}
	if selector != "./pkg/server" {
		t.Errorf("package selector = %q, want ./pkg/server", selector)
	}
}

// TestCodeCheck_TestExecutor_DefaultModeRunsAllPackages verifies that without
// file mode the executor targets the whole module (./...).
func TestCodeCheck_TestExecutor_DefaultModeRunsAllPackages(t *testing.T) {
	runner := &fakeRunner{outputs: map[string][]byte{"go": []byte("ok\n")}}
	e := &testExecutor{runner: runner, fileMode: false}

	if _, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	call := runner.lastCall()
	if got := call.args[len(call.args)-1]; got != "./..." {
		t.Errorf("default-mode selector = %q, want ./...", got)
	}
}

// TestCodeCheck_LintExecutor_MalformedJSONErrors verifies that unparseable
// golangci-lint output surfaces an error (the run error when the tool exited
// non-zero, otherwise the parse error) rather than a silent clean pass.
func TestCodeCheck_LintExecutor_MalformedJSONErrors(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{"golangci-lint": []byte("not json")},
		err:     &exitError{code: 3},
	}
	e := &lintExecutor{runner: runner}
	if _, err := e.Execute(context.Background(), []string{"a.go"}); err == nil {
		t.Error("expected an error for malformed output with a non-zero exit")
	}

	runner2 := &fakeRunner{outputs: map[string][]byte{"golangci-lint": []byte("{not json")}}
	e2 := &lintExecutor{runner: runner2}
	if _, err := e2.Execute(context.Background(), []string{"a.go"}); err == nil {
		t.Error("expected a parse error for malformed output")
	}
}

// TestCodeCheck_Executors_EntryContextCancelled verifies that each executor
// returns ctx.Err() immediately when the context is already cancelled at entry,
// without invoking the runner.
func TestCodeCheck_Executors_EntryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &fakeRunner{outputs: map[string][]byte{}}

	execs := []PassExecutor{
		&lintExecutor{runner: runner},
		&buildExecutor{runner: runner},
		&testExecutor{runner: runner},
		&semgrepExecutor{runner: runner, ensurer: &mockSemgrepEnsurer{}},
	}
	for _, ex := range execs {
		if _, err := ex.Execute(ctx, []string{"a.go"}); err == nil {
			t.Errorf("%T.Execute should return ctx error when cancelled at entry", ex)
		}
	}
	if len(runner.calls) != 0 {
		t.Errorf("no runner call expected on entry-cancellation, got %d", len(runner.calls))
	}
}

// TestCodeCheck_SemgrepExecutor_EnsureFailureErrors verifies that an
// EnsureSemgrep failure surfaces as an Execute error rather than a silent pass.
func TestCodeCheck_SemgrepExecutor_EnsureFailureErrors(t *testing.T) {
	runner := &fakeRunner{outputs: map[string][]byte{}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) {
		return "", &DegradedError{Message: "install failed"}
	}}
	e := &semgrepExecutor{runner: runner, ensurer: ensurer}
	if _, err := e.Execute(context.Background(), []string{"a.go"}); err == nil {
		t.Error("expected an error when EnsureSemgrep fails")
	}
	if len(runner.calls) != 0 {
		t.Error("runner should not be invoked when the binary cannot be resolved")
	}
}

// TestCodeCheck_GoPackageSelector covers the package-selector edge cases.
func TestCodeCheck_GoPackageSelector(t *testing.T) {
	cases := map[string]string{
		"main.go":                 ".",
		"pkg/server/handler.go":   "./pkg/server",
		"./pkg/server/handler.go": "./pkg/server",
		"/abs/pkg/server/file.go": "/abs/pkg/server",
	}
	for in, want := range cases {
		if got := goPackageSelector(in); got != want {
			t.Errorf("goPackageSelector(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCodeCheck_SemgrepSeverity covers severity normalization, including the
// default (unknown) branch.
func TestCodeCheck_SemgrepSeverity(t *testing.T) {
	cases := map[string]string{
		"ERROR":   "error",
		"WARNING": "warning",
		"INFO":    "info", // default branch lowercases
		"":        "",
	}
	for in, want := range cases {
		if got := semgrepSeverity(in); got != want {
			t.Errorf("semgrepSeverity(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCodeCheck_SemgrepExecutor_MalformedJSONErrors verifies a parse error is
// surfaced for unparseable semgrep output.
func TestCodeCheck_SemgrepExecutor_MalformedJSONErrors(t *testing.T) {
	const binPath = "semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte("{not json")}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}
	e := &semgrepExecutor{runner: runner, ensurer: ensurer}
	if _, err := e.Execute(context.Background(), []string{"a.go"}); err == nil {
		t.Error("expected a parse error for malformed semgrep output")
	}
}

// exitError is a fake non-nil error simulating a tool's non-zero exit. Real
// tools (golangci-lint, go build/test) exit non-zero when they find problems;
// the executors must parse their output instead of bubbling the exit up.
type exitError struct{ code int }

func (e *exitError) Error() string { return "exit status non-zero" }
