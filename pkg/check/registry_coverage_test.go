package check

import (
	"context"
	"errors"
	"testing"
)

// TestCodeCheck_Parsers_GoFormatWrappers exercises the golangci-json/go-build/
// go-test named formats through the registry, confirming they wrap the existing
// parse* funcs and stamp the target pass (the retarget/retargetViolations path).
func TestCodeCheck_Parsers_GoFormatWrappers(t *testing.T) {
	golangci := `{"Issues":[{"FromLinter":"govet","Text":"shadow","Severity":"error","Pos":{"Filename":"a.go","Line":3}}]}`
	p, err := lookupParser("golangci-json")
	if err != nil {
		t.Fatalf("lookupParser(golangci-json): %v", err)
	}
	vs, err := p([]byte(golangci), CheckTypeLint)
	if err != nil {
		t.Fatalf("golangci parse: %v", err)
	}
	if len(vs) != 1 || vs[0].Pass != CheckTypeLint || vs[0].File != "a.go" {
		t.Fatalf("golangci wrapper = %+v, want one lint violation on a.go", vs)
	}

	// go-build wrapper.
	pb, _ := lookupParser("go-build")
	bvs, err := pb([]byte("pkg/x.go:10:2: undefined: foo"), CheckTypeBuild)
	if err != nil {
		t.Fatalf("go-build parse: %v", err)
	}
	if len(bvs) != 1 || bvs[0].Pass != CheckTypeBuild {
		t.Fatalf("go-build wrapper = %+v, want one build violation", bvs)
	}

	// go-test wrapper.
	pt, _ := lookupParser("go-test")
	tvs, err := pt([]byte("--- FAIL: TestFoo (0.00s)\n    foo_test.go:5: boom"), CheckTypeTest)
	if err != nil {
		t.Fatalf("go-test parse: %v", err)
	}
	if len(tvs) != 1 || tvs[0].Pass != CheckTypeTest {
		t.Fatalf("go-test wrapper = %+v, want one test violation", tvs)
	}
}

// TestCodeCheck_Parsers_EmptyAndMalformedInputs covers the empty-input and
// malformed-JSON branches of the new parsers (fail-loud on bad JSON, nil on
// empty).
func TestCodeCheck_Parsers_EmptyAndMalformedInputs(t *testing.T) {
	for _, format := range []string{"eslint-json", "sarif"} {
		p, _ := lookupParser(format)
		vs, err := p([]byte("   "), CheckTypeLint)
		if err != nil || len(vs) != 0 {
			t.Errorf("%s empty input = (%v, %v), want (nil, nil)", format, vs, err)
		}
		if _, err := p([]byte("{not json"), CheckTypeLint); err == nil {
			t.Errorf("%s malformed input returned nil error; want a parse error", format)
		}
	}

	// tsc/regex-lines never error; empty input yields no violations.
	for _, format := range []string{"tsc", "regex-lines"} {
		p, _ := lookupParser(format)
		vs, err := p(nil, CheckTypeBuild)
		if err != nil || len(vs) != 0 {
			t.Errorf("%s nil input = (%v, %v), want (nil, nil)", format, vs, err)
		}
	}
}

// TestCodeCheck_CommandExecutor_ProjectWideCrashGuard covers the project-wide
// crash-vs-findings guard: a non-zero run with no parseable findings surfaces as
// an error, not a silent green.
func TestCodeCheck_CommandExecutor_ProjectWideCrashGuard(t *testing.T) {
	parser, _ := lookupParser("tsc")
	runner := &fakeRunner{
		outputs: map[string][]byte{"tsc": []byte("internal compiler panic")},
		err:     errors.New("exit status 2"),
	}
	exec := &commandExecutor{
		pass:      CheckTypeBuild,
		command:   "tsc --noEmit",
		parser:    parser,
		scopeKind: ScopeKindProjectWide,
		runner:    runner,
	}
	_, err := exec.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("project-wide pass with run error and no findings returned nil error; want a crash-guard error")
	}

	// A run error WITH parseable findings is NOT a crash — findings returned.
	runner2 := &fakeRunner{
		outputs: map[string][]byte{"tsc": []byte("a.ts(1,1): error TS1: boom")},
		err:     errors.New("exit status 2"),
	}
	exec.runner = runner2
	pr, err := exec.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("project-wide pass with findings errored: %v", err)
	}
	if len(pr.Violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(pr.Violations))
	}
}

// TestCodeCheck_CommandExecutor_DependencyMappedAndEmptyCommand covers the
// dependency-mapped arg assembly (test_dependency_command preferred over the
// full-suite command) and the empty-command config error.
func TestCodeCheck_CommandExecutor_DependencyMappedAndEmptyCommand(t *testing.T) {
	parser, _ := lookupParser("regex-lines")
	runner := &fakeRunner{outputs: map[string][]byte{"vitest": nil}}
	exec := &commandExecutor{
		pass:                  CheckTypeTest,
		command:               "vitest run",
		testDependencyCommand: "vitest related",
		parser:                parser,
		scopeKind:             ScopeKindDependencyMapped,
		runner:                runner,
	}
	if _, err := exec.Execute(context.Background(), []string{"a.ts"}); err != nil {
		t.Fatalf("dependency-mapped Execute: %v", err)
	}
	last := runner.lastCall()
	if last.name != "vitest" || len(last.args) == 0 || last.args[0] != "related" {
		t.Errorf("dependency-mapped command = %q %v, want 'vitest related'", last.name, last.args)
	}
	// Scoped files are NOT appended for dependency-mapped passes.
	for _, a := range last.args {
		if a == "a.ts" {
			t.Errorf("dependency-mapped args %v wrongly include the scoped file", last.args)
		}
	}

	// Empty command → config error.
	empty := &commandExecutor{pass: CheckTypeLint, parser: parser, scopeKind: ScopeKindFileArgs, runner: runner}
	_, err := empty.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("empty-command executor returned nil error; want a config error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("empty-command error %T is not a *ConfigError", err)
	}
}

// TestCodeCheck_CommandExecutor_IsAvailableAndContextCancel covers IsAvailable
// (always true for the generic executor) and the context-cancellation
// short-circuit.
func TestCodeCheck_CommandExecutor_IsAvailableAndContextCancel(t *testing.T) {
	parser, _ := lookupParser("regex-lines")
	exec := &commandExecutor{pass: CheckTypeLint, command: "tool", parser: parser, scopeKind: ScopeKindFileArgs, runner: &fakeRunner{}}
	if ok, _ := exec.IsAvailable(); !ok {
		t.Error("commandExecutor.IsAvailable() = false, want true")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := exec.Execute(ctx, nil); err == nil {
		t.Error("Execute with cancelled context returned nil error; want context error")
	}
}

// TestCodeCheck_Registry_UnknownFormatIsConfigError covers the unknown-format
// branch of buildExecutorsForConfigErr: a declared toolchain whose format is not
// known is a config error.
func TestCodeCheck_Registry_UnknownFormatIsConfigError(t *testing.T) {
	yaml := `project: x
language: rust
enforcement:
  toolchain:
    lint:
      command: "tool"
      format: not-a-real-format
`
	cfg := loadConfigFromYAML(t, yaml)
	_, err := buildExecutorsForConfigErr(Options{Language: "rust", Config: cfg}, &fakeRunner{})
	if err == nil {
		t.Fatal("unknown format returned nil error; want a config error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("unknown-format error %T is not a *ConfigError", err)
	}
}

// TestCodeCheck_Registry_UnknownPassNameSkipped covers declaredEntries skipping
// an unrecognized pass name without crashing.
func TestCodeCheck_Registry_UnknownPassNameSkipped(t *testing.T) {
	yaml := `project: x
language: rust
enforcement:
  toolchain:
    boguspass:
      command: "tool"
      format: regex-lines
    lint:
      command: "lint-tool"
      format: regex-lines
      extensions: [".rs"]
`
	cfg := loadConfigFromYAML(t, yaml)
	execs, err := buildExecutorsForConfigErr(Options{Language: "rust", Config: cfg}, &fakeRunner{})
	if err != nil {
		t.Fatalf("buildExecutorsForConfigErr: %v", err)
	}
	if _, ok := execs[CheckTypeLint]; !ok {
		t.Error("lint executor missing; the recognized pass must still bind despite the bogus pass name")
	}
}
