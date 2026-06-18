package engine

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestEngineBinding_Shape asserts the EngineBinding carries exactly the declared
// fields: command, input_mode, input_flag, scope_kind, optional convert and
// provision — and that a SARIF-native engine leaves Convert empty while a
// non-SARIF engine populates it. CLM-034.
func TestEngineBinding_Shape(t *testing.T) {
	b := EngineBinding{
		Command:   "semgrep scan",
		InputMode: InputModeRuleFlags,
		InputFlag: "--config",
		ScopeKind: ScopeKindFileArgs,
	}
	if b.Command != "semgrep scan" {
		t.Errorf("Command = %q", b.Command)
	}
	if b.InputMode != InputModeRuleFlags {
		t.Errorf("InputMode = %q", b.InputMode)
	}
	if b.InputFlag != "--config" {
		t.Errorf("InputFlag = %q", b.InputFlag)
	}
	if b.ScopeKind != ScopeKindFileArgs {
		t.Errorf("ScopeKind = %d", b.ScopeKind)
	}
	if b.Convert != "" {
		t.Errorf("SARIF-native binding must leave Convert empty, got %q", b.Convert)
	}
	if b.Provision != nil {
		t.Errorf("assumed-present binding must leave Provision nil, got %+v", b.Provision)
	}

	// A non-SARIF engine declares a convert script.
	withConvert := EngineBinding{Command: "ast-grep scan", InputMode: InputModeRuleDir, InputFlag: "--rule", Convert: "ast-grep/to-sarif.sh"}
	if withConvert.Convert == "" {
		t.Error("non-SARIF binding must populate Convert")
	}
}

// TestEngineBinding_NoFormatSelector asserts the EngineBinding type has NO Format
// field: findings output is always parsed as SARIF, so a format selector must not
// exist on the struct. CLM-021. Enforced structurally by source inspection so a
// future Format field is caught even if no test references it.
func TestEngineBinding_NoFormatSelector(t *testing.T) {
	data, err := os.ReadFile("binding.go")
	if err != nil {
		t.Fatalf("read binding.go: %v", err)
	}
	if strings.Contains(string(data), "Format") {
		t.Fatalf("binding.go must not mention Format; output is always SARIF")
	}
}

// TestInputMode_ConfigFile asserts the config-file enum value round-trips.
// CLM-044.
func TestInputMode_ConfigFile(t *testing.T) {
	got, err := ParseInputMode("config-file")
	if err != nil {
		t.Fatalf("ParseInputMode(config-file): %v", err)
	}
	if got != InputModeConfigFile {
		t.Errorf("got %q, want config-file", got)
	}
	if string(InputModeConfigFile) != "config-file" {
		t.Errorf("InputModeConfigFile string = %q", InputModeConfigFile)
	}
}

// TestInputMode_RuleFlags asserts the rule-flags enum value round-trips. CLM-045.
func TestInputMode_RuleFlags(t *testing.T) {
	got, err := ParseInputMode("rule-flags")
	if err != nil {
		t.Fatalf("ParseInputMode(rule-flags): %v", err)
	}
	if got != InputModeRuleFlags {
		t.Errorf("got %q, want rule-flags", got)
	}
}

// TestInputMode_RuleDir asserts the rule-dir enum value round-trips. CLM-046.
func TestInputMode_RuleDir(t *testing.T) {
	got, err := ParseInputMode("rule-dir")
	if err != nil {
		t.Fatalf("ParseInputMode(rule-dir): %v", err)
	}
	if got != InputModeRuleDir {
		t.Errorf("got %q, want rule-dir", got)
	}
}

// TestInputMode_None asserts the none enum value round-trips. CLM-047.
func TestInputMode_None(t *testing.T) {
	got, err := ParseInputMode("none")
	if err != nil {
		t.Fatalf("ParseInputMode(none): %v", err)
	}
	if got != InputModeNone {
		t.Errorf("got %q, want none", got)
	}
}

// TestInputMode_UnknownValueFailsLoud asserts an unrecognized input_mode value is
// a blocking config error, never a silent default. CLM-048.
func TestInputMode_UnknownValueFailsLoud(t *testing.T) {
	_, err := ParseInputMode("rule-glob")
	if err == nil {
		t.Fatal("expected error for unknown input_mode, got nil")
	}
	if !strings.Contains(err.Error(), "rule-glob") {
		t.Errorf("error must name the offending value, got: %v", err)
	}
}

// TestRegistry_SeedsBuiltins asserts the default registry seeds the built-in
// engines (semgrep, ast-grep, sandbox, config-file) with their declared shapes.
func TestRegistry_SeedsBuiltins(t *testing.T) {
	reg := DefaultRegistry()
	for _, name := range []string{"semgrep", "ast-grep", "sandbox", "config-file"} {
		b, err := reg.Lookup(name)
		if err != nil {
			t.Errorf("Lookup(%q): %v", name, err)
			continue
		}
		if b.Command == "" && name != "sandbox" && name != "config-file" {
			t.Errorf("engine %q has empty command", name)
		}
	}

	semgrep := mustLookup(t, reg, "semgrep")
	if semgrep.InputMode != InputModeRuleFlags {
		t.Errorf("semgrep input_mode = %q, want rule-flags", semgrep.InputMode)
	}
	if semgrep.Convert != "" {
		t.Errorf("semgrep is SARIF-native via --sarif; it must not declare a pack convert script, got Convert = %q", semgrep.Convert)
	}

	astgrep := mustLookup(t, reg, "ast-grep")
	if astgrep.InputMode != InputModeRuleDir {
		t.Errorf("ast-grep input_mode = %q, want rule-dir", astgrep.InputMode)
	}
	if astgrep.InputFlag != "--rule" {
		t.Errorf("ast-grep input_flag = %q, want --rule", astgrep.InputFlag)
	}
	if astgrep.Convert == "" {
		t.Error("ast-grep must declare a stdin->SARIF convert script")
	}
	if astgrep.Provision == nil {
		t.Error("ast-grep must declare a pinned provision (backstop-introduced engine)")
	}

	sandbox := mustLookup(t, reg, "sandbox")
	if sandbox.InputMode != InputModeNone {
		t.Errorf("sandbox input_mode = %q, want none", sandbox.InputMode)
	}

	configFile := mustLookup(t, reg, "config-file")
	if configFile.InputMode != InputModeConfigFile {
		t.Errorf("config-file input_mode = %q, want config-file", configFile.InputMode)
	}
}

// mustLookup resolves an engine binding or fails the test, so callers can assert
// on the binding without ignoring Lookup's error.
func mustLookup(t *testing.T, reg Registry, name string) EngineBinding {
	t.Helper()
	b, err := reg.Lookup(name)
	if err != nil {
		t.Fatalf("Lookup(%q): %v", name, err)
	}
	return b
}

// TestRegistry_UnknownEngineFailsLoud asserts Registry.Lookup fail-louds on an
// engine name it does not know — never a silent skip. CLM-020.
func TestRegistry_UnknownEngineFailsLoud(t *testing.T) {
	reg := DefaultRegistry()
	_, err := reg.Lookup("clj-kondo-typo")
	if err == nil {
		t.Fatal("expected error for unknown engine, got nil")
	}
	if !strings.Contains(err.Error(), "clj-kondo-typo") {
		t.Errorf("error must name the unknown engine, got: %v", err)
	}
}

// TestProvision_EmptyVsPinned asserts the Provision descriptor distinguishes an
// assumed-present (nil) engine from a pinned/introduced one.
func TestProvision_EmptyVsPinned(t *testing.T) {
	reg := DefaultRegistry()
	sandbox := mustLookup(t, reg, "sandbox")
	if sandbox.Provision != nil {
		t.Errorf("sandbox (logic-is-the-executable) must be assumed-present (nil provision), got %+v", sandbox.Provision)
	}
	configFile := mustLookup(t, reg, "config-file")
	if configFile.Provision != nil {
		t.Errorf("config-file (Layer-0 native linter) must be assumed-present (nil provision), got %+v", configFile.Provision)
	}
	astgrep := mustLookup(t, reg, "ast-grep")
	if astgrep.Provision == nil || astgrep.Provision.Version == "" {
		t.Errorf("ast-grep must carry a pinned provision with a version, got %+v", astgrep.Provision)
	}
}

// TestScopeKind_MirrorsButDoesNotImportCheck asserts ScopeKind is this package's
// own int type (so the leaf package need not import pkg/check). The named
// constants exist and are distinct. Sharp Edge 7.
func TestScopeKind_MirrorsButDoesNotImportCheck(t *testing.T) {
	if ScopeKindFileArgs == ScopeKindProjectWide {
		t.Error("ScopeKindFileArgs and ScopeKindProjectWide must be distinct")
	}
	if ScopeKindProjectWide == ScopeKindDependencyMapped {
		t.Error("ScopeKindProjectWide and ScopeKindDependencyMapped must be distinct")
	}
	// The three named constants must occupy three distinct int values (mirroring
	// pkg/check.ScopeKind's iota set without importing it).
	distinct := map[ScopeKind]struct{}{
		ScopeKindFileArgs:         {},
		ScopeKindProjectWide:      {},
		ScopeKindDependencyMapped: {},
	}
	if len(distinct) != 3 {
		t.Errorf("ScopeKind values collapsed: %d distinct, want 3", len(distinct))
	}
}

// TestEngine_NoForbiddenImports asserts the engine leaf package imports none of
// pkg/check, pkg/packval, or cmd/backstop — importing any would reintroduce the
// cycle this leaf placement exists to prevent. CLM-033.
func TestEngine_NoForbiddenImports(t *testing.T) {
	fset := token.NewFileSet()
	for _, name := range []string{"binding.go"} {
		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, forbidden := range []string{
				"github.com/bmanson/backstop-core/pkg/check",
				"github.com/bmanson/backstop-core/pkg/packval",
				"github.com/bmanson/backstop-core/cmd/backstop",
			} {
				if path == forbidden {
					t.Errorf("%s imports forbidden package %q (reintroduces import cycle)", name, path)
				}
			}
		}
	}
}
