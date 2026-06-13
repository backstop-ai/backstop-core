package check

import (
	"context"
	"testing"
)

// TestCodeCheck_TSExecutors_ParseESLintJSON pins CLM-003: the TS lint executor,
// driven through a fakeRunner returning eslint JSON, yields violations with the
// correct File(filePath)/Line/Message/Severity (2→error, 1→warning) and
// Rule(ruleId).
func TestCodeCheck_TSExecutors_ParseESLintJSON(t *testing.T) {
	runner := &fakeRunner{outputs: map[string][]byte{
		"eslint": []byte(eslintSampleJSON),
	}}
	cfg := loadConfigFromYAML(t, tsBackstopYML)
	execs := buildExecutorsForConfig(Options{Language: "typescript", Config: cfg}, runner)

	lint, ok := execs[CheckTypeLint].(*commandExecutor)
	if !ok {
		t.Fatalf("lint executor = %T, want *commandExecutor", execs[CheckTypeLint])
	}
	pr, err := lint.Execute(context.Background(), []string{"src/app.ts"})
	if err != nil {
		t.Fatalf("TS lint Execute: %v", err)
	}
	if len(pr.Violations) != 2 {
		t.Fatalf("got %d violations, want 2 (empty-messages file contributes none)", len(pr.Violations))
	}
	v0 := pr.Violations[0]
	if v0.File != "/repo/src/app.ts" || v0.Line != 12 {
		t.Errorf("v[0] file/line = %q/%d, want /repo/src/app.ts/12", v0.File, v0.Line)
	}
	if v0.Message != "'foo' is not defined." {
		t.Errorf("v[0].Message = %q", v0.Message)
	}
	if v0.Rule != "no-undef" {
		t.Errorf("v[0].Rule = %q, want no-undef", v0.Rule)
	}
	if v0.Severity != "error" {
		t.Errorf("v[0].Severity = %q, want error (severity 2)", v0.Severity)
	}
	if pr.Violations[1].Severity != "warning" {
		t.Errorf("v[1].Severity = %q, want warning (severity 1)", pr.Violations[1].Severity)
	}

	// The eslint command was actually invoked with the scoped file appended.
	last := runner.lastCall()
	if last.name != "eslint" {
		t.Errorf("lint runner command = %q, want eslint", last.name)
	}
	sawFile := false
	for _, a := range last.args {
		if a == "src/app.ts" {
			sawFile = true
		}
	}
	if !sawFile {
		t.Errorf("lint args %v missing scoped file src/app.ts", last.args)
	}
}

// TestCodeCheck_TSExecutors_ParseTscOutput pins CLM-003: the TS build executor,
// driven through a fakeRunner returning tsc output, parses
// `file(line,col): error TSxxxx: message` lines into build violations with the
// right File/Line/Message and Rule=TSxxxx.
func TestCodeCheck_TSExecutors_ParseTscOutput(t *testing.T) {
	runner := &fakeRunner{outputs: map[string][]byte{
		"tsc": []byte(tscSampleTxt),
	}}
	cfg := loadConfigFromYAML(t, tsBackstopYML)
	execs := buildExecutorsForConfig(Options{Language: "typescript", Config: cfg}, runner)

	build, ok := execs[CheckTypeBuild].(*commandExecutor)
	if !ok {
		t.Fatalf("build executor = %T, want *commandExecutor", execs[CheckTypeBuild])
	}
	pr, err := build.Execute(context.Background(), []string{"src/app.ts"})
	if err != nil {
		t.Fatalf("TS build Execute: %v", err)
	}
	// 3 diagnostic lines (2 error + 1 warning); the "Found 2 errors" summary
	// line must not match.
	if len(pr.Violations) != 3 {
		t.Fatalf("got %d violations, want 3 (summary line must be skipped)", len(pr.Violations))
	}
	v0 := pr.Violations[0]
	if v0.File != "src/app.ts" || v0.Line != 12 {
		t.Errorf("v[0] file/line = %q/%d, want src/app.ts/12", v0.File, v0.Line)
	}
	if v0.Rule != "TS2304" {
		t.Errorf("v[0].Rule = %q, want TS2304", v0.Rule)
	}
	if v0.Message != "Cannot find name 'foo'." {
		t.Errorf("v[0].Message = %q", v0.Message)
	}
	if v0.Pass != CheckTypeBuild {
		t.Errorf("v[0].Pass = %v, want build", v0.Pass)
	}

	// Build is project-wide: the scoped file must NOT be appended to the args.
	last := runner.lastCall()
	if last.name != "tsc" {
		t.Errorf("build runner command = %q, want tsc", last.name)
	}
	for _, a := range last.args {
		if a == "src/app.ts" {
			t.Errorf("build args %v wrongly include the scoped file; build is project-wide", last.args)
		}
	}
}

// TestCodeCheck_Routing_TSFilesRouteAllPasses pins CLM-006: with a
// typescript-language compiled manifest, RouteFile on a `.ts` AND a `.tsx` path
// returns {lint, build, test, semgrep} — matching the treatment `.go` receives.
func TestCodeCheck_Routing_TSFilesRouteAllPasses(t *testing.T) {
	dir := t.TempDir()
	writeRawManifest(t, dir, "STD-TS-001.manifest.json", `{
  "standard": "STD-TS-001",
  "language": "typescript",
  "semgrep_config": "STD-TS-001.semgrep.yml",
  "rules": [
    {"id": "TS-001", "name": "no-any", "enforcement": "semgrep"}
  ]
}`)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.isDefaults {
		t.Fatal("manifest fell back to defaults; want typescript routing derived")
	}

	for _, path := range []string{"src/app.ts", "src/widget.tsx"} {
		checks := m.RouteFile(path)
		want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep}
		if len(checks) != len(want) {
			t.Fatalf("%s routed %d checks, want 4: %v", path, len(checks), checks)
		}
		for _, ct := range want {
			if !containsCheckType(checks, ct) {
				t.Errorf("%s routing %v missing %v", path, checks, ct)
			}
		}
	}
}

// TestCodeCheck_Routing_DeclaredStackExtensionsRoute pins CLM-006: a declared
// stack whose toolchain declares extensions routes files of those extensions to
// the declared passes equivalently. A compiled manifest for an arbitrary
// language with declared extensions routes its extension to lint/build/test
// (+semgrep when signalled).
func TestCodeCheck_Routing_DeclaredStackExtensionsRoute(t *testing.T) {
	dir := t.TempDir()
	// A compiled manifest declaring routable extensions for a non-builtin
	// language. The manifest carries the declared extensions so routing can
	// derive them without a built-in languageExtensions entry.
	writeRawManifest(t, dir, "STD-RS-001.manifest.json", `{
  "standard": "STD-RS-001",
  "language": "rust",
  "extensions": [".rs"],
  "semgrep_config": "STD-RS-001.semgrep.yml",
  "rules": [
    {"id": "RS-001", "name": "no-unwrap", "enforcement": "semgrep"}
  ]
}`)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.isDefaults {
		t.Fatal("manifest fell back to defaults; want declared-extension routing")
	}

	checks := m.RouteFile("src/lib.rs")
	if !containsCheckType(checks, CheckTypeLint) ||
		!containsCheckType(checks, CheckTypeBuild) ||
		!containsCheckType(checks, CheckTypeTest) {
		t.Errorf("declared .rs routing = %v, want at least lint/build/test", checks)
	}
	if !containsCheckType(checks, CheckTypeSemgrep) {
		t.Errorf("declared .rs routing = %v, want semgrep (signal present)", checks)
	}

	// A file of an undeclared extension must NOT route to the declared passes.
	other := m.RouteFile("docs/readme.md")
	if containsCheckType(other, CheckTypeLint) || containsCheckType(other, CheckTypeBuild) {
		t.Errorf("undeclared extension routed to lint/build: %v", other)
	}
}
