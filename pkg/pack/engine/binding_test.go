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
	withConvert := EngineBinding{Command: "ast-grep scan", InputMode: InputModeConfigFile, InputFlag: "--config", Convert: "ast-grep/to-sarif.sh"}
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

// TestParseInputMode_PatternArgAccepted asserts ParseInputMode resolves the new
// fifth value "pattern-arg" to InputModePatternArg with no error — the seam
// BUNDLE-009 needs for pattern-as-argument engines. CLM-014.
func TestParseInputMode_PatternArgAccepted(t *testing.T) {
	got, err := ParseInputMode("pattern-arg")
	if err != nil {
		t.Fatalf("ParseInputMode(pattern-arg): %v", err)
	}
	if got != InputModePatternArg {
		t.Errorf("got %q, want pattern-arg", got)
	}
	if string(InputModePatternArg) != "pattern-arg" {
		t.Errorf("InputModePatternArg string = %q, want pattern-arg", InputModePatternArg)
	}
}

// TestParseInputMode_UnknownStillFailsLoud asserts that adding pattern-arg did
// not open a silent-default hole: an unrecognized input_mode still errors and
// names the offending value. CLM-015.
func TestParseInputMode_UnknownStillFailsLoud(t *testing.T) {
	_, err := ParseInputMode("pattern-glob")
	if err == nil {
		t.Fatal("expected error for unknown input_mode after pattern-arg added, got nil")
	}
	if !strings.Contains(err.Error(), "pattern-glob") {
		t.Errorf("error must name the offending value, got: %v", err)
	}
	// pattern-arg must appear in the accepted-values message so the enum stays
	// exhaustively documented in the fail-loud surface.
	if !strings.Contains(err.Error(), "pattern-arg") {
		t.Errorf("fail-loud message must list pattern-arg as an accepted value, got: %v", err)
	}
}

// TestRegistry_UnknownEngineFailsLoud asserts Registry.Lookup fail-louds on an
// engine name it does not know — never a silent skip (CLM-020). It builds the
// registry as a literal rather than a baked table: after ISSUE-027 the built-in
// bindings are pack DATA (the embedded base-engines pack), and this leaf package
// holds only the type + Lookup, so the test exercises Lookup's fail-loud contract
// directly.
func TestRegistry_UnknownEngineFailsLoud(t *testing.T) {
	reg := Registry{
		"semgrep": {Command: "semgrep --sarif --quiet", InputMode: InputModeRuleFlags},
	}
	_, err := reg.Lookup("clj-kondo-typo")
	if err == nil {
		t.Fatal("expected error for unknown engine, got nil")
	}
	if !strings.Contains(err.Error(), "clj-kondo-typo") {
		t.Errorf("error must name the unknown engine, got: %v", err)
	}
}

// TestRegistry_LookupKnownEngineReturnsBinding asserts Registry.Lookup resolves a
// KNOWN engine name to its binding with no error — the success path complementing
// the fail-loud unknown-engine path.
func TestRegistry_LookupKnownEngineReturnsBinding(t *testing.T) {
	want := EngineBinding{Command: "semgrep --sarif --quiet", InputMode: InputModeRuleFlags, InputFlag: "--config"}
	reg := Registry{"semgrep": want}

	got, err := reg.Lookup("semgrep")
	if err != nil {
		t.Fatalf("Lookup of a known engine must not error, got: %v", err)
	}
	if got.Command != want.Command || got.InputMode != want.InputMode {
		t.Errorf("Lookup returned %+v, want %+v", got, want)
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

// TestExemption_BindingCarriesExemptFromScopeFilterDecoupledFromScopeKind proves
// EngineBinding carries an ExemptFromScopeFilter bool DECOUPLED from ScopeKind: a
// binding can be ScopeKindProjectWide with the exemption either set or unset
// (SPEC-041 CLM-011/CLM-017). After ISSUE-027 the go-build/go-test/golangci bindings
// are pack DATA (the external go-toolchain pack asserts their concrete values in
// cmd/backstop/go_toolchain_engines_test.go); this leaf-package test proves the
// STRUCT decouples the two fields, built as literals.
func TestExemption_BindingCarriesExemptFromScopeFilterDecoupledFromScopeKind(t *testing.T) {
	exempt := EngineBinding{ScopeKind: ScopeKindProjectWide, ProjectTarget: "./...", ExemptFromScopeFilter: true}
	notExempt := EngineBinding{ScopeKind: ScopeKindProjectWide, ProjectTarget: "./...", ExemptFromScopeFilter: false}

	if !exempt.ExemptFromScopeFilter {
		t.Error("a binding declaring exempt_from_scope_filter:true must carry it")
	}
	if notExempt.ExemptFromScopeFilter {
		t.Error("a binding leaving exempt_from_scope_filter unset must be false")
	}
	// Same ScopeKind, divergent exempt values: ScopeKind does not drive the exempt
	// decision (CLM-017).
	if exempt.ScopeKind != notExempt.ScopeKind {
		t.Fatal("both bindings must share ScopeKindProjectWide for the decoupling to be observable")
	}
	if exempt.ExemptFromScopeFilter == notExempt.ExemptFromScopeFilter {
		t.Error("exempt decision must be DECOUPLED from ScopeKind: same ScopeKind, divergent exempt values (CLM-017)")
	}
}
