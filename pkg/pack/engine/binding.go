// Package engine holds the shared EngineBinding table that the gate uses to
// dispatch pack-rule enforcement onto a declared execution engine. It is a leaf
// package: it imports none of pkg/check, pkg/packval, or cmd/backstop, so all
// three can import it without an import cycle (REQ-013). The engine model is
// open and declared — adding an engine is an EngineBinding record (plus, at
// most, a registered convert script), never a surgical edit to the gate's
// executor (REQ-001/REQ-005).
package engine

import "fmt"

// InputMode is the structured enum describing how an engine receives its inputs
// (REQ-020). It has exactly four values; an unrecognized value is a blocking
// config error via ParseInputMode.
type InputMode string

const (
	// InputModeConfigFile passes a single optional pack-supplied config file; the
	// tool runs its own built-in rules (e.g. golangci-lint/eslint/tsc).
	InputModeConfigFile InputMode = "config-file"
	// InputModeRuleFlags repeats input_flag once per rule file (e.g. semgrep's
	// --config X --config Y).
	InputModeRuleFlags InputMode = "rule-flags"
	// InputModeRuleDir collects rule files into one directory passed once via
	// input_flag (e.g. ast-grep --rule DIR).
	InputModeRuleDir InputMode = "rule-dir"
	// InputModeNone injects no rules or config; the executable is the logic
	// (the sandbox engine).
	InputModeNone InputMode = "none"
)

// ParseInputMode resolves a raw string to an InputMode, fail-louding on an
// unrecognized value (REQ-020 / CLM-048). It never defaults silently.
func ParseInputMode(s string) (InputMode, error) {
	switch InputMode(s) {
	case InputModeConfigFile, InputModeRuleFlags, InputModeRuleDir, InputModeNone:
		return InputMode(s), nil
	default:
		return "", fmt.Errorf("unknown input_mode %q: must be one of config-file, rule-flags, rule-dir, none", s)
	}
}

// ScopeKind classifies how an engine attaches scoped files to its invocation. It
// is this package's OWN int type that mirrors but does not import
// pkg/check.ScopeKind — reusing the pkg/check type would reintroduce the import
// cycle this leaf package exists to prevent (REQ-013 / Sharp Edge 7).
type ScopeKind int

const (
	// ScopeKindFileArgs appends the scoped files to the command as arguments.
	ScopeKindFileArgs ScopeKind = iota
	// ScopeKindProjectWide runs the command once project-wide, ignoring the
	// scoped file list.
	ScopeKindProjectWide
	// ScopeKindDependencyMapped runs once, optionally using a dependency map.
	ScopeKindDependencyMapped
)

// Provision is a pinned install descriptor for a backstop-introduced engine
// (semgrep, ast-grep). A nil *Provision means the engine is an assumed-present
// Layer-0 native tool: a missing binary fails loud and backstop never installs
// it (REQ-019). A non-nil Provision is auto-provisioned and verified through the
// backstop.lock / VerifyLock path — data-driven, with no per-engine Go.
type Provision struct {
	// Tool is the provisioned binary/package name (e.g. "semgrep", "ast-grep").
	Tool string
	// Version is the pinned version the lock path verifies against.
	Version string
}

// EngineBinding is the open, declared description of one execution engine
// (REQ-001/REQ-006/REQ-013). There is intentionally NO format selector: findings
// output is always parsed as SARIF (REQ-006/CLM-021). A SARIF-native engine
// leaves Convert empty; a non-SARIF engine declares a pack-relative Convert
// script that transforms its stdout to SARIF.
type EngineBinding struct {
	// Command is the engine invocation prefix (e.g. "semgrep scan").
	Command string
	// InputMode declares how rule/config inputs are gathered (REQ-020).
	InputMode InputMode
	// InputFlag is the flag used to inject inputs per InputMode (e.g. "--config",
	// "--rule"). Empty for InputModeNone.
	InputFlag string
	// ScopeKind governs how the engine attaches scoped files (Sharp Edge 7).
	ScopeKind ScopeKind
	// Convert is an optional pack-relative stdin->SARIF converter script. Empty
	// for a SARIF-native engine; non-empty for a tool whose native output is not
	// SARIF (e.g. ast-grep). REQ-007/REQ-008.
	Convert string
	// Provision is an optional pinned install descriptor. Nil => assumed-present
	// Layer-0 engine (REQ-019).
	Provision *Provision
	// CrashGuard, when true, marks a findings engine whose tool run exiting
	// non-zero with NO parseable findings is a CRASH (a broken tool/infra error
	// naming the engine), not a finding-free green (SPEC-034 REQ-003/CLM-010).
	// The native go build/test passes set this: a compiler/test-binary crash must
	// never read as a silent pass. semgrep/ast-grep/golangci leave it false —
	// they legitimately exit non-zero WHEN they report findings, so the SARIF on
	// stdout, not the exit code, is their contract.
	CrashGuard bool
	// ProjectTarget, when non-empty, is the project-wide scan target a toolchain
	// pass shapes for ITSELF (e.g. "./..." for `go build ./...`) INSTEAD of having
	// the project root appended as a scan argument the way rule-fed findings
	// engines do (SPEC-034 REQ-010/CLM-034, N1). It is the scope-kind-aware
	// arg-shaping notion: a ScopeKindProjectWide engine with a ProjectTarget runs
	// `<command> <ProjectTarget>` and never gets projectRoot bolted on.
	ProjectTarget string
}

// Registry maps an engine name to its EngineBinding. The gate looks up a rule's
// declared engine here and dispatches through the binding; an unknown engine is
// a fail-loud config error, never a silent skip (REQ-005 / CLM-020). Packs may
// contribute additional bindings beyond the built-ins.
type Registry map[string]EngineBinding

// Lookup resolves an engine name to its binding, fail-louding on an unknown
// engine (REQ-005 / CLM-020).
func (r Registry) Lookup(name string) (EngineBinding, error) {
	b, ok := r[name]
	if !ok {
		return EngineBinding{}, fmt.Errorf("unknown engine %q: no EngineBinding registered", name)
	}
	return b, nil
}

// DefaultRegistry returns the built-in engine bindings: semgrep, ast-grep,
// sandbox, and the config-file native-linter shape. Registering a new engine is
// a declaration here (an EngineBinding record), not an edit to the gate's
// executor switch (REQ-001/REQ-003/CLM-003).
//
// Output contract: every findings engine's normalized output is SARIF
// (REQ-006). semgrep emits SARIF natively via its --sarif flag, so it carries no
// Convert script. ast-grep emits its own JSON, so it ships a pack-relative
// stdin->SARIF converter (REQ-008).
//
// Provisioning (REQ-019): semgrep and ast-grep are backstop-introduced engines
// with pinned Provision records (auto-provisioned via the lock path). sandbox
// and config-file are assumed-present (nil Provision): the sandbox executable is
// shipped by the pack itself and a config-file native linter is a Layer-0 tool
// that must already be present.
func DefaultRegistry() Registry {
	return Registry{
		"semgrep": {
			// --sarif makes semgrep emit SARIF JSON on stdout (the engine
			// output contract); --quiet suppresses the human-readable banner
			// so stdout is pure SARIF. Without --sarif, semgrep prints its text
			// report and the SARIF parse fails on the box-drawing bytes.
			Command:   "semgrep --sarif --quiet",
			InputMode: InputModeRuleFlags,
			InputFlag: "--config",
			ScopeKind: ScopeKindFileArgs,
			Provision: &Provision{Tool: "semgrep", Version: "1.96.0"},
		},
		"ast-grep": {
			Command:   "ast-grep scan",
			InputMode: InputModeRuleDir,
			InputFlag: "--rule",
			ScopeKind: ScopeKindFileArgs,
			Convert:   "ast-grep/to-sarif.sh",
			Provision: &Provision{Tool: "ast-grep", Version: "0.43.0"},
		},
		"sandbox": {
			Command:   "",
			InputMode: InputModeNone,
			InputFlag: "",
			ScopeKind: ScopeKindFileArgs,
		},
		"config-file": {
			Command:   "",
			InputMode: InputModeConfigFile,
			InputFlag: "--config",
			ScopeKind: ScopeKindProjectWide,
		},
		// Native Go toolchain engines (SPEC-034). These are the Layer-0
		// assume-present passes the native code-check toolchain was bridged onto:
		// adding them is an EngineBinding record (the engine model is open and
		// declared), not a gate executor edit. They carry nil Provision (the
		// project owns its own `go`/`golangci-lint`; a missing binary fails loud,
		// backstop never installs it — REQ-008).
		//
		// golangci is a config-file engine on golangci-lint v2 native SARIF: it
		// declares no Convert (v2 emits SARIF on stdout) and its config (the
		// pack-owned .golangci.yml) is the optional config-file input. It runs the
		// `run` subcommand with the SARIF output flags and no version probe
		// (REQ-005). ScopeKindProjectWide + a ProjectTarget of "./..." so the lint
		// run is project-wide and the project root is never appended.
		"golangci": {
			Command:       "golangci-lint run --output.sarif.path stdout --show-stats=false",
			InputMode:     InputModeConfigFile,
			InputFlag:     "--config",
			ScopeKind:     ScopeKindProjectWide,
			ProjectTarget: "./...",
		},
		// go-build / go-test are findings engines: their stdout is normalized to
		// SARIF by the pack-relative convert script (the retired
		// parseGoBuildErrors / parseGoTestFailures logic relocated OUTSIDE the core
		// binary — DD-2). CrashGuard distinguishes a compiler/test crash from
		// findings; ProjectTarget "./..." is the project-wide scope (REQ-003/004/010).
		"go-build": {
			Command:       "go build",
			InputMode:     InputModeNone,
			ScopeKind:     ScopeKindProjectWide,
			ProjectTarget: "./...",
			Convert:       "scripts/build-to-sarif.sh",
			CrashGuard:    true,
		},
		"go-test": {
			Command:       "go test",
			InputMode:     InputModeNone,
			ScopeKind:     ScopeKindProjectWide,
			ProjectTarget: "./...",
			Convert:       "scripts/test-to-sarif.sh",
			CrashGuard:    true,
		},
	}
}
