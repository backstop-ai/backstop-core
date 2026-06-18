package check

import (
	"context"
	"fmt"
	"strings"

	"github.com/bmanson/backstop-core/pkg/config"
)

// ScopeKind classifies how a pass shapes its tool invocation against the scoped
// file list. It is a per-pass invocation concern (Ratified Design Constraint 2):
// lint passes the scoped files as args; build/typecheck runs project-wide and
// ignores the list; test is dependency-mapped with full-suite fallback.
type ScopeKind int

const (
	// ScopeKindFileArgs appends the scoped files to the command as arguments
	// (lint). All scoped files go into ONE invocation.
	ScopeKindFileArgs ScopeKind = iota
	// ScopeKindProjectWide runs the command once project-wide, ignoring the
	// scoped file list (build/typecheck). Its violations are never
	// scope-filtered at the gate layer.
	ScopeKindProjectWide
	// ScopeKindDependencyMapped runs the test command once, optionally using a
	// dependency-mapping command, with the full-suite command as fallback.
	ScopeKindDependencyMapped
)

// ToolchainEntry is a single pass binding in a Toolchain: the command to run,
// the named output format that parses its output, the routable extensions the
// stack declares, an optional per-toolchain test dependency-mapping command,
// and the ScopeKind shaping its invocation. It is an extensible object so a
// future stack-generic knob (e.g. ISSUE-007 exclude_paths) can slot in without
// reshaping the registry.
type ToolchainEntry struct {
	Command               string
	Format                string
	Extensions            []string
	TestDependencyCommand string
	ScopeKind             ScopeKind
}

// Toolchain is a stack: a map of pass → entry, plus the stack's aggregate
// routable extensions and an explicit test command (TS/declared stacks declare
// it; constraint 1 — no package.json detection).
type Toolchain struct {
	Entries     map[CheckType]ToolchainEntry
	Extensions  []string
	TestCommand string
}

// builtinToolchains returns the predefined built-in stacks (go, typescript).
// The go stack is sentinel-only: its executors are the bespoke goBuiltin
// implementations, so its entries carry no command (selectStack special-cases
// it). The typescript stack is fully data-driven via the generic
// commandExecutor.
func builtinToolchain(language string) (Toolchain, bool) {
	switch language {
	case "go":
		return Toolchain{
			Entries:    map[CheckType]ToolchainEntry{},
			Extensions: []string{".go"},
		}, true
	case "typescript":
		return Toolchain{
			Entries: map[CheckType]ToolchainEntry{
				CheckTypeLint: {
					Command:    "eslint --format json",
					Format:     "eslint-json",
					Extensions: []string{".ts", ".tsx"},
					ScopeKind:  ScopeKindFileArgs,
				},
				CheckTypeBuild: {
					Command:    "tsc --noEmit",
					Format:     "tsc",
					Extensions: []string{".ts", ".tsx"},
					ScopeKind:  ScopeKindProjectWide,
				},
				// Test entry's command is filled from enforcement.test_command
				// at construction time; format parses generic test output.
				CheckTypeTest: {
					Format:     "regex-lines",
					Extensions: []string{".ts", ".tsx"},
					ScopeKind:  ScopeKindDependencyMapped,
				},
			},
			Extensions: []string{".ts", ".tsx"},
		}, true
	default:
		return Toolchain{}, false
	}
}

// commandExecutor is the generic PassExecutor backing the typescript built-in
// stack and every declared stack. It is parameterized by a command string, a
// parser resolved from the named format, a ScopeKind, and the CommandRunner
// seam. The four bespoke Go executors are NOT routed through this path so the
// landed ISSUE-002 behavior has no regression risk.
type commandExecutor struct {
	pass                  CheckType
	command               string
	testDependencyCommand string
	parser                Parser
	scopeKind             ScopeKind
	runner                CommandRunner
}

// Execute splits the command, applies the ScopeKind to shape arguments, runs it
// via the runner, and parses the output with the named format. Project-wide
// passes reuse the crash-vs-findings guard: a failed run with no parseable
// violations surfaces as an error rather than a silent green.
func (e *commandExecutor) Execute(ctx context.Context, files []string) (*PassResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name, args := e.assembleCommand(files)
	if name == "" {
		return nil, &ConfigError{Message: fmt.Sprintf("%s pass has no command configured", e.pass)}
	}

	out, runErr := e.runner.Run(ctx, name, args...)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	violations, parseErr := e.parser(out, e.pass)
	if parseErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("%s (%s): %w", e.pass, name, runErr)
		}
		return nil, fmt.Errorf("parsing %s output: %w", e.pass, parseErr)
	}

	// Crash-vs-findings guard for project-wide passes: a failed run with no
	// parseable violations is a tool crash, not a finding-free pass.
	if e.scopeKind == ScopeKindProjectWide && runErr != nil && len(violations) == 0 {
		return nil, fmt.Errorf("%s failed without parseable findings: %v: %s", e.pass, runErr, firstOutputLine(out))
	}

	return &PassResult{Pass: e.pass, Violations: violations}, nil
}

// assembleCommand returns the executable name and arguments for this pass,
// honoring the ScopeKind:
//   - file-args: base command + scoped files as trailing args (ONE invocation).
//   - project-wide: base command only; scoped files ignored (ONE invocation).
//   - dependency-mapped: the test_dependency_command if set, else the base
//     command (full-suite fallback); scoped files are NOT appended.
func (e *commandExecutor) assembleCommand(files []string) (string, []string) {
	base := e.command
	if e.scopeKind == ScopeKindDependencyMapped && e.testDependencyCommand != "" {
		base = e.testDependencyCommand
	}
	parts := strings.Fields(base)
	if len(parts) == 0 {
		return "", nil
	}
	name := parts[0]
	args := append([]string(nil), parts[1:]...)
	if e.scopeKind == ScopeKindFileArgs {
		args = append(args, files...)
	}
	return name, args
}

// IsAvailable reports the generic executor as available; declared/TS toolchains
// rely on the configured command existing. Availability probing for a specific
// binary is a future refinement; today a missing binary surfaces as a run error
// via the crash-vs-findings guard rather than a silent skip.
func (e *commandExecutor) IsAvailable() (bool, string) {
	return true, ""
}

// buildExecutorsForConfig selects the toolchain by language and builds the
// executor map, panicking only on programmer error. Construction errors (a
// missing toolchain for a declared language, an unknown format, a TS stack with
// no test command, an unrecognized enforcement.toolchain pass key) are surfaced
// via buildExecutorsForConfigErr; this variant returns an empty map on such
// errors so the caller's Run path can re-derive and propagate the ConfigError
// through its normal channel.
//
// GUARD: this variant DISCARDS the ConfigError. It (and its check.go twin
// buildDefaultExecutors/buildDefaultExecutorsWithRunner) must NEVER be wired
// into a production Executors assignment — doing so would re-open the
// silent-non-enforcement hole this package fails loud on (e.g. ISSUE-008's
// typo'd toolchain key would build a partial/empty map with no exit-2). The
// real Run path uses buildExecutorsForConfigErr and propagates its error.
func buildExecutorsForConfig(opts Options, runner CommandRunner) map[CheckType]PassExecutor {
	execs, _ := buildExecutorsForConfigErr(opts, runner)
	return execs
}

// buildExecutorsForConfigErr is the error-returning registry constructor. It
// selects the stack by Options.Language (Go-default when empty), merges any
// enforcement.toolchain declarations, and builds the executor map. A declared
// language with no built-in stack and no enforcement.toolchain declaration, an
// unknown format, or a TS/declared stack missing its required test command is a
// *ConfigError (exit 2) — never a silent skip.
func buildExecutorsForConfigErr(opts Options, runner CommandRunner) (map[CheckType]PassExecutor, error) {
	if runner == nil {
		runner = &ExecCommandRunner{Dir: opts.ProjectDir}
	}

	// Single source of truth for enforcement.toolchain key validity. This runs
	// BEFORE the language defaulting and BEFORE the go/empty-language
	// early-return below, so a go-language project with a typo'd key fails loud
	// identically to a non-go one (ISSUE-008): an out-of-vocabulary pass key is a
	// *ConfigError (exit 2), never a silent skip that disables a pass.
	if err := validateToolchainKeys(opts.Config); err != nil {
		return map[CheckType]PassExecutor{}, err
	}

	language := opts.Language
	if language == "" {
		language = "go"
	}

	// Go-default stack: the bespoke executors, unchanged.
	if language == "go" {
		return goBuiltinExecutors(opts, runner), nil
	}

	toolchain, err := resolveToolchain(language, opts.Config)
	if err != nil {
		return map[CheckType]PassExecutor{}, err
	}

	execs := map[CheckType]PassExecutor{}
	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest} {
		entry, ok := toolchain.Entries[ct]
		if !ok {
			continue
		}
		parser, perr := lookupParser(entry.Format)
		if perr != nil {
			return map[CheckType]PassExecutor{}, perr
		}
		execs[ct] = &commandExecutor{
			pass:                  ct,
			command:               entry.Command,
			testDependencyCommand: entry.TestDependencyCommand,
			parser:                parser,
			scopeKind:             entry.ScopeKind,
			runner:                runner,
		}
	}

	// Semgrep stays the shared executor for all stacks.
	execs[CheckTypeSemgrep] = &semgrepExecutor{
		runner:              runner,
		ensurer:             &DefaultSemgrepEnsurer{},
		backstopDir:         opts.BackstopDir,
		pinnedVersion:       opts.PinnedSemgrepVersion,
		extraSemgrepConfigs: opts.ExtraSemgrepConfigs,
	}

	return execs, nil
}

// resolveToolchain produces the effective Toolchain for a language by starting
// from the built-in stack (if any), then overlaying enforcement.toolchain
// declarations from config. A declared-only language (no built-in) requires an
// enforcement.toolchain declaration; otherwise it is a config error. The TS
// test pass requires enforcement.test_command (constraint 1).
func resolveToolchain(language string, cfg *config.Config) (Toolchain, error) {
	toolchain, builtin := builtinToolchain(language)
	declared := declaredEntries(cfg)

	if !builtin && len(declared) == 0 {
		return Toolchain{}, &ConfigError{Message: fmt.Sprintf(
			"no toolchain for declared language %q: add a built-in stack or declare enforcement.toolchain in backstop.yml", language)}
	}

	if toolchain.Entries == nil {
		toolchain.Entries = map[CheckType]ToolchainEntry{}
	}

	// Overlay declared entries (command/format/extensions/test_dependency).
	for ct, decl := range declared {
		entry := toolchain.Entries[ct]
		entry.Command = decl.Command
		entry.Format = decl.Format
		if len(decl.Extensions) > 0 {
			entry.Extensions = decl.Extensions
		}
		entry.TestDependencyCommand = decl.TestDependencyCommand
		entry.ScopeKind = defaultScopeKind(ct, entry.ScopeKind)
		toolchain.Entries[ct] = entry
	}

	// Apply the explicit test command (TS/declared) to the test entry.
	if cfg != nil && cfg.Enforcement.TestCommand != "" {
		toolchain.TestCommand = cfg.Enforcement.TestCommand
		testEntry := toolchain.Entries[CheckTypeTest]
		if testEntry.Command == "" {
			testEntry.Command = cfg.Enforcement.TestCommand
		}
		if testEntry.Format == "" {
			testEntry.Format = "regex-lines"
		}
		testEntry.ScopeKind = ScopeKindDependencyMapped
		toolchain.Entries[CheckTypeTest] = testEntry
	}

	// TS test pass requires an explicit command (enforced in TASK-010 path).
	if err := requireTestCommand(language, toolchain); err != nil {
		return Toolchain{}, err
	}

	return toolchain, nil
}

// validateToolchainKeys is the single source of truth for enforcement.toolchain
// key validity (ISSUE-008 / REQ-001). It returns a *ConfigError for the FIRST
// key that parseCheckType does not recognize, naming the offending key and
// enumerating the allowed vocabulary (lint/build/test/semgrep) so the author
// can self-correct a typo. A `semgrep:` key is accepted as in-vocabulary even
// though it has no toolchain-overlay effect today (semgrep stays the shared
// executor). A nil cfg or empty toolchain map is valid (returns nil).
func validateToolchainKeys(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	for pass := range cfg.Enforcement.Toolchain {
		if _, ok := parseCheckType(pass); !ok {
			return &ConfigError{Message: fmt.Sprintf(
				"unknown enforcement.toolchain pass key %q in backstop.yml: allowed keys are lint, build, test, semgrep", pass)}
		}
	}
	return nil
}

// declaredEntries maps config.Enforcement.Toolchain (string-keyed by pass name)
// to CheckType-keyed ToolchainEntry values. By the time this runs the keys have
// already been validated by validateToolchainKeys (the single source of truth),
// so the unrecognized-name branch is a defensive no-op over an already-validated
// map rather than the silent-skip hole ISSUE-008 closed.
func declaredEntries(cfg *config.Config) map[CheckType]ToolchainEntry {
	entries := map[CheckType]ToolchainEntry{}
	if cfg == nil {
		return entries
	}
	for pass, decl := range cfg.Enforcement.Toolchain {
		ct, ok := parseCheckType(pass)
		if !ok {
			continue
		}
		entries[ct] = ToolchainEntry{
			Command:               decl.Command,
			Format:                decl.Format,
			Extensions:            decl.Extensions,
			TestDependencyCommand: decl.TestDependencyCommand,
			ScopeKind:             defaultScopeKind(ct, ScopeKindFileArgs),
		}
	}
	return entries
}

// defaultScopeKind returns the conventional ScopeKind for a pass when an entry
// does not pin one: lint=file-args, build=project-wide, test=dependency-mapped.
// A non-zero current value is preserved.
func defaultScopeKind(ct CheckType, current ScopeKind) ScopeKind {
	switch ct {
	case CheckTypeLint:
		return ScopeKindFileArgs
	case CheckTypeBuild:
		return ScopeKindProjectWide
	case CheckTypeTest:
		return ScopeKindDependencyMapped
	default:
		return current
	}
}

// requireTestCommand enforces that a typescript stack (and any declared stack
// whose test pass relies on an explicit command) has a non-empty test command.
// Missing it is a *ConfigError (exit 2), never a silent skip (constraint 1,
// CLM-004).
func requireTestCommand(language string, toolchain Toolchain) error {
	if language != "typescript" {
		return nil
	}
	testEntry, has := toolchain.Entries[CheckTypeTest]
	if !has || testEntry.Command == "" {
		return &ConfigError{Message: fmt.Sprintf(
			"language %q requires enforcement.test_command in backstop.yml: the test pass command must be explicitly declared (no package.json detection)", language)}
	}
	return nil
}
