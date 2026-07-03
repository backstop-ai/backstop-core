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
	if astgrep.InputMode != InputModeConfigFile {
		t.Errorf("ast-grep input_mode = %q, want config-file", astgrep.InputMode)
	}
	if astgrep.InputFlag != "--config" {
		t.Errorf("ast-grep input_flag = %q, want --config", astgrep.InputFlag)
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

// TestDefaultRegistry_AstGrepUsesConfigFileMode pins the corrected ast-grep
// dispatch shape (ISSUE-028 / CLM-001): the ast-grep built-in resolves its
// multi-rule input through the EXISTING config-file mode (a single pack-shipped
// sgconfig.yml via --config), NOT the retired rule-dir/"--rule" shape that
// emitted `--rule <DIR>` and ran zero rules. The flip changes ONLY input
// mode/flag: the engine name stays "ast-grep" and it retains its Convert script
// (ast-grep/to-sarif.sh), pinned Provision, and EngineCategoryOpinion.
func TestDefaultRegistry_AstGrepUsesConfigFileMode(t *testing.T) {
	reg := DefaultRegistry()
	astgrep := mustLookup(t, reg, "ast-grep")

	if astgrep.InputMode != InputModeConfigFile {
		t.Errorf("ast-grep input_mode = %q, want config-file (the multi-rule sgconfig.yml mechanism)", astgrep.InputMode)
	}
	if astgrep.InputFlag != "--config" {
		t.Errorf("ast-grep input_flag = %q, want --config", astgrep.InputFlag)
	}
	// The flip keeps the engine's identity (still the "ast-grep scan" invocation,
	// now with --json so the real binary emits the JSON the convert script reads).
	if astgrep.Command != "ast-grep scan --json" {
		t.Errorf("ast-grep command must stay the ast-grep scan invocation, got %q", astgrep.Command)
	}
	if astgrep.Convert != "ast-grep/to-sarif.sh" {
		t.Errorf("ast-grep must keep its stdin->SARIF convert script, got %q", astgrep.Convert)
	}
	if astgrep.Provision == nil || astgrep.Provision.Tool != "ast-grep" {
		t.Errorf("ast-grep must keep its pinned provision, got %+v", astgrep.Provision)
	}
	if astgrep.Category != EngineCategoryOpinion {
		t.Errorf("ast-grep must stay an OPINION engine, got category %d", astgrep.Category)
	}
	// The engine is still registered under the name "ast-grep" (not renamed to
	// "config-file", which forbids rule_path — CLM-009).
	if _, ok := reg["ast-grep"]; !ok {
		t.Error("ast-grep must remain registered under the name \"ast-grep\"")
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

// TestExemption_BindingDeclaresExemptFromScopeFilterDecoupledFromScopeKind proves
// EngineBinding carries an ExemptFromScopeFilter bool DECOUPLED from ScopeKind:
// the DefaultRegistry go-build engine declares it true, golangci and go-test
// declare it false/unset — asserted against the declared binding records
// (SPEC-041 CLM-011).
func TestExemption_BindingDeclaresExemptFromScopeFilterDecoupledFromScopeKind(t *testing.T) {
	reg := DefaultRegistry()

	build, err := reg.Lookup("go-build")
	if err != nil {
		t.Fatalf("go-build binding must exist: %v", err)
	}
	if !build.ExemptFromScopeFilter {
		t.Error("go-build must declare exempt_from_scope_filter:true — it is the build-pass exemption (CLM-011)")
	}

	for _, name := range []string{"golangci", "go-test"} {
		b, err := reg.Lookup(name)
		if err != nil {
			t.Fatalf("%s binding must exist: %v", name, err)
		}
		if b.ExemptFromScopeFilter {
			t.Errorf("%s must NOT declare exempt_from_scope_filter — only go-build is exempt (CLM-011)", name)
		}
	}
}

// TestExemption_ScopeKindDecoupledFromExemptDecision proves ScopeKind and
// ExemptFromScopeFilter are independent: golangci/go-build/go-test all remain
// ScopeKindProjectWide (each still appends its ./... ProjectTarget) while ONLY
// go-build is exempt_from_scope_filter — ScopeKind is NOT consulted for the
// exempt/ProjectWide decision (SPEC-041 CLM-017).
func TestExemption_ScopeKindDecoupledFromExemptDecision(t *testing.T) {
	reg := DefaultRegistry()
	for _, name := range []string{"golangci", "go-build", "go-test"} {
		b, err := reg.Lookup(name)
		if err != nil {
			t.Fatalf("%s binding must exist: %v", name, err)
		}
		if b.ScopeKind != ScopeKindProjectWide {
			t.Errorf("%s must stay ScopeKindProjectWide (arg-shaping), got %v (CLM-017)", name, b.ScopeKind)
		}
		if b.ProjectTarget != "./..." {
			t.Errorf("%s must keep its ./... ProjectTarget (arg-shaping), got %q (CLM-017)", name, b.ProjectTarget)
		}
	}
	// Decoupling: all three share ScopeKindProjectWide, yet ONLY go-build is exempt.
	// If ScopeKind drove the exempt decision, golangci/go-test would be exempt too.
	golangci, _ := reg.Lookup("golangci")
	build, _ := reg.Lookup("go-build")
	gotest, _ := reg.Lookup("go-test")
	if !(build.ExemptFromScopeFilter && !golangci.ExemptFromScopeFilter && !gotest.ExemptFromScopeFilter) {
		t.Error("exempt decision must be DECOUPLED from ScopeKind: same ScopeKindProjectWide, divergent exempt values (CLM-017)")
	}
}
