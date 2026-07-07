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
	case InputModeConfigFile, InputModeRuleFlags, InputModeNone, InputModePatternArg:
		return InputMode(s), nil
	default:
		return "", fmt.Errorf("unknown input_mode %q: must be one of config-file, rule-flags, none, pattern-arg", s)
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
	// Producer is an OPTIONAL pack-relative script (symmetric with Convert) that,
	// when set, the dispatch runs UN-SANDBOXED to produce this engine's payload
	// INSTEAD of the plain Command (ISSUE-045 option (ii)). It exists for a payload
	// that requires full toolchain/project access to produce — e.g. a coverage
	// producer that folds the module path + package file list into the profile the
	// SANDBOXED convert then parses. The dispatch resolves it under packRoot
	// (filepath.Join(packRoot, Producer) + os.Stat, the SAME pattern Convert uses)
	// and runs it via the runner (cwd = projectRoot) so its `go test`/`go list` see
	// the project. It is pack DATA — the binary stays tool/language-blind: it knows
	// only "run this pack-relative producer to get the output", never what the output
	// means. Empty => the plain Command produces the payload (every other engine).
	Producer string `yaml:"producer"`
	// StdoutArtifact, when non-empty, names a FILE (relative to the run's working
	// dir) that the engine writes its real output to INSTEAD of stdout. The
	// dispatch then feeds THAT file's contents — not the command's stdout — into
	// the Convert. It exists for tools whose stdout is summary/noise while the
	// payload lands in a file (e.g. a coverage pass that writes a profile to a
	// file and prints only a test summary to stdout). The filename is pack DATA —
	// the binary stays tool/language-blind: it knows only "this engine's output is
	// in a declared file", never what the file means.
	StdoutArtifact string `yaml:"stdout_artifact"`
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
	// ExemptFromScopeFilter, when true, marks an engine whose violations are EXEMPT
	// from diff-scope filtering — they are stamped gate.Violation.ProjectWide on the
	// engine path (cmd/backstop/pack_gate.go) so an out-of-scope (unchanged-file)
	// violation still REDs a diff-scoped gate (SPEC-041 REQ-004/CLM-011). It is the
	// DECLARED replacement for the deleted baked `cv.Pass == check.CheckTypeBuild`
	// identity check AND the SPEC-040 transitional `GateType == GateTypeBuild` seam:
	// no CheckType enum identity and no GateType identity drives scope — the property
	// is explicit and per-binding. It is DECOUPLED from ScopeKind (which stays
	// arg-shaping-only): the go-build engine declares it true; golangci and go-test
	// declare it false/unset (CLM-017). Resolution is per-violation (REQ-007).
	ExemptFromScopeFilter bool `yaml:"exempt_from_scope_filter"`
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

// The baked built-in engine table that once lived here is DELETED (ISSUE-027): the
// four generic engines (semgrep, ast-grep, sandbox, config-file) are now DATA in the
// embedded base-engines pack (pkg/baseengines loads packs/base-engines/pack.yml), and
// the three Go toolchain engines (go-build, go-test, golangci) are DATA in the
// external go-toolchain pack. The leaf engine package holds NO runtime engine table —
// only the types, ParseInputMode, and Registry.Lookup. Registering an engine is a
// pack DATA edit, never a binary edit.
