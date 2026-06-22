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
	InputModeConfigFile InputMode = "config-file" // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// InputModeRuleFlags repeats input_flag once per rule file (e.g. semgrep's
	// --config X --config Y).
	InputModeRuleFlags InputMode = "rule-flags" // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// InputModeRuleDir collects rule files into one directory passed once via
	// input_flag (e.g. ast-grep --rule DIR).
	InputModeRuleDir InputMode = "rule-dir" // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// InputModeNone injects no rules or config; the executable is the logic
	// (the sandbox engine).
	InputModeNone InputMode = "none" // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// InputModePatternArg passes each rule's inline pattern as a command argument
	// (via InputFlag) instead of resolving a rule file on disk — the BUNDLE-009
	// seam for pattern-as-argument engines (REQ-004/CLM-014). gatherEngineInputs
	// emits [InputFlag, rule.Pattern] per rule and never os.Stats a rule path
	// (CLM-016/CLM-018).
	InputModePatternArg InputMode = "pattern-arg" // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
)

// ParseInputMode resolves a raw string to an InputMode, fail-louding on an
// unrecognized value (REQ-020 / CLM-048). It never defaults silently.
func ParseInputMode(s string) (InputMode, error) {
	switch InputMode(s) {
	case InputModeConfigFile, InputModeRuleFlags, InputModeRuleDir, InputModeNone, InputModePatternArg:
		return InputMode(s), nil
	default:
		return "", fmt.Errorf("unknown input_mode %q: must be one of config-file, rule-flags, rule-dir, none, pattern-arg", s)
	}
}

// ScopeKind classifies how an engine attaches scoped files to its invocation. It
// is this package's OWN int type that mirrors but does not import
// pkg/check.ScopeKind — reusing the pkg/check type would reintroduce the import
// cycle this leaf package exists to prevent (REQ-013 / Sharp Edge 7).
type ScopeKind int

const (
	// ScopeKindFileArgs appends the scoped files to the command as arguments.
	ScopeKindFileArgs ScopeKind = iota // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// ScopeKindProjectWide runs the command once project-wide, ignoring the
	// scoped file list.
	ScopeKindProjectWide
	// ScopeKindDependencyMapped runs once, optionally using a dependency map.
	ScopeKindDependencyMapped
)

// EngineCategory classifies an engine as Go-native toolchain MECHANISM or
// coding-standards OPINION for the pack-separation enforcer (ISSUE-015 /
// SPEC-034 REQ-007). It is the single source of truth for the mechanism/opinion
// boundary: the enforcer reads this field off the engine's binding instead of
// reconstructing the classification from a hardcoded engine set. The zero value
// EngineCategoryUnset means the engine is NEITHER mechanism nor opinion (e.g.
// sandbox, config-file), so a rule binding it trips no boundary check — identical
// to the pre-ISSUE-015 switches returning false for both.
type EngineCategory int

const (
	// EngineCategoryUnset is the zero value: the engine is neither toolchain
	// mechanism nor coding-standards opinion. A rule binding such an engine is
	// invisible to the pack-separation boundary check.
	EngineCategoryUnset EngineCategory = iota // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// EngineCategoryMechanism marks a Go native-toolchain engine that runs the
	// toolchain (build/test/lint) and normalizes its output (go-build, go-test,
	// golangci). A pack carrying these rules is a mechanism pack.
	EngineCategoryMechanism
	// EngineCategoryOpinion marks a swappable coding-standards engine fed by rule
	// files (semgrep, ast-grep). A pack carrying these rules is an opinion pack.
	EngineCategoryOpinion
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
	Command string `yaml:"command"`
	// InputMode declares how rule/config inputs are gathered (REQ-020).
	InputMode InputMode `yaml:"input_mode"`
	// InputFlag is the flag used to inject inputs per InputMode (e.g. "--config",
	// "--rule"). Empty for InputModeNone.
	InputFlag string `yaml:"input_flag"`
	// ScopeKind governs how the engine attaches scoped files (Sharp Edge 7).
	ScopeKind ScopeKind `yaml:"scope_kind"`
	// Convert is an optional pack-relative stdin->SARIF converter script. Empty
	// for a SARIF-native engine; non-empty for a tool whose native output is not
	// SARIF (e.g. ast-grep). REQ-007/REQ-008.
	Convert string `yaml:"convert"`
	// Provision is an optional pinned install descriptor. Nil => assumed-present
	// Layer-0 engine (REQ-019).
	Provision *Provision `yaml:"provision"`
	// CrashGuard, when true, marks a findings engine whose tool run exiting
	// non-zero with NO parseable findings is a CRASH (a broken tool/infra error
	// naming the engine), not a finding-free green (SPEC-034 REQ-003/CLM-010).
	// The native go build/test passes set this: a compiler/test-binary crash must
	// never read as a silent pass. semgrep/ast-grep/golangci leave it false —
	// they legitimately exit non-zero WHEN they report findings, so the SARIF on
	// stdout, not the exit code, is their contract.
	CrashGuard bool `yaml:"crash_guard"`
	// Category classifies the engine as toolchain MECHANISM or coding-standards
	// OPINION for the pack-separation enforcer (ISSUE-015 / SPEC-034 REQ-007). The
	// zero value EngineCategoryUnset means neither, so a rule binding the engine is
	// invisible to the boundary check. This is the single source of truth for the
	// classification: the enforcer reads it here rather than mirroring an engine set.
	Category EngineCategory `yaml:"category"`
	// ProjectTarget, when non-empty, is the project-wide scan target a toolchain
	// pass shapes for ITSELF (e.g. "./..." for `go build ./...`) INSTEAD of having
	// the project root appended as a scan argument the way rule-fed findings
	// engines do (SPEC-034 REQ-010/CLM-034, N1). It is the scope-kind-aware
	// arg-shaping notion: a ScopeKindProjectWide engine with a ProjectTarget runs
	// `<command> <ProjectTarget>` and never gets projectRoot bolted on.
	ProjectTarget string `yaml:"project_target"`
	// GateType is the backstop-owned, tool-NEUTRAL gate-type this engine fills
	// (lint/build/test/findings/coverage/substantiveness/contracts) — REQ-005/
	// CLM-019/CLM-020. The zero value is GateTypeLint; a pack declares the value
	// explicitly via gate_type. Defined in this leaf package so the binding carries
	// no pkg/check import.
	GateType GateType `yaml:"gate_type"`
	// StrictSarif, when true, marks an engine whose stdout must be strictly valid
	// SARIF (the declared output-contract flag that REPLACES the isNativeSarifLint
	// Engine "golangci-lint" command-prefix sniff — REQ-006a/CLM-023). The strict-
	// SARIF shape guard keys off this declared flag, never a tool-name prefix.
	StrictSarif bool `yaml:"strict_sarif"`
	// PackageScoped, when true, marks an engine that runs per Go package rather
	// than over a flat file list (the declared file-mode capability that REPLACES
	// the isNativeGoTestEngine "go test" command-prefix sniff — REQ-006b/CLM-024).
	PackageScoped bool `yaml:"package_scoped"`
	// FieldContract is the engine's DECLARED field-contract: the Rule field names
	// it requires and forbids (REQ-003/CLM-036). A pack-declared engine supplies
	// this inline in the engines: block; the validator reads it FROM the binding,
	// not from a name-keyed map. The zero value (empty lists) imposes no contract.
	FieldContract FieldContract `yaml:"field_contract"`
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
// OQ-1 disposition (SPEC-035 REQ-007, resolved to OPTION i — incremental
// overridable fallback): this table is the FALLBACK the stage-1 registry merge
// (resolveEngineRegistry) overrides and extends. A pack-declared binding of the
// SAME engine name WINS over the same-named built-in here, while the built-ins
// stay available to dispatch (CLM-027/CLM-004). It is NOT a frozen baked table:
// the merge makes pack-declared engines first-class and lets a pack vendor its
// own same-named engine. Full eradication of this fallback into a default pack is
// entangled with BUNDLE-011's toolchain-pack collapse and is tracked separately;
// option (i) ships SPEC-035 complete with the built-ins overridable, not deleted.
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
			Category:  EngineCategoryOpinion,
		},
		"ast-grep": {
			Command:   "ast-grep scan",
			InputMode: InputModeRuleDir,
			InputFlag: "--rule",
			ScopeKind: ScopeKindFileArgs,
			Convert:   "ast-grep/to-sarif.sh",
			Provision: &Provision{Tool: "ast-grep", Version: "0.43.0"},
			Category:  EngineCategoryOpinion,
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
			Category:      EngineCategoryMechanism,
			// Declares strict native v2 SARIF on stdout: the strict-SARIF shape
			// guard keys off THIS flag, not a "golangci-lint" name sniff (REQ-006a/
			// CLM-023). A v1/too-old binary emitting non-SARIF JSON fails loud.
			StrictSarif: true,
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
			Category:      EngineCategoryMechanism,
		},
		"go-test": {
			Command:       "go test",
			InputMode:     InputModeNone,
			ScopeKind:     ScopeKindProjectWide,
			ProjectTarget: "./...",
			Convert:       "scripts/test-to-sarif.sh",
			CrashGuard:    true,
			Category:      EngineCategoryMechanism,
			// Runs per Go package: the file-mode package scoping (`code check
			// --file`) keys off THIS flag, not a "go test" name sniff (REQ-006b/
			// CLM-024). go-build is project-wide too but does NOT set it.
			PackageScoped: true,
		},
	}
}
