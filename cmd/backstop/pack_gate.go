package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
	"github.com/bmanson/backstop-core/pkg/packval"
)

// sandboxedRun / sandboxedRunStdout / engineRegistry are test seams: nil in
// production (the resolveXxx helpers below fall back to the concrete
// packval.SandboxedRun, packval.SandboxedRunStdout, and engine.DefaultRegistry),
// and overridden by tests to substitute a stub sandbox or inject a custom engine
// binding. They are declared WITHOUT initializers so they hold no package-level
// mutable default; the real implementation is resolved lazily at the call site.
var (
	sandboxedRun       func(cmd string, args []string, packDir string) ([]byte, error)
	sandboxedRunStdout func(cmd string, args []string, packDir string, stdin []byte) ([]byte, error)
	engineRegistry     engine.Registry
)

// resolveSandboxedRun returns the injected sandbox seam or the concrete
// packval.SandboxedRun.
func resolveSandboxedRun() func(cmd string, args []string, packDir string) ([]byte, error) {
	if sandboxedRun != nil {
		return sandboxedRun
	}
	return packval.SandboxedRun
}

// resolveSandboxedRunStdout returns the injected clean-stdout sandbox seam or the
// concrete packval.SandboxedRunStdout (REQ-007/REQ-009/CLM-065).
func resolveSandboxedRunStdout() func(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
	if sandboxedRunStdout != nil {
		return sandboxedRunStdout
	}
	return packval.SandboxedRunStdout
}

// resolveEngineRegistry returns the injected engine registry seam or the
// concrete engine.DefaultRegistry, so tests can register a new engine binding
// and prove it dispatches without an executor edit (CLM-003).
func resolveEngineRegistry() engine.Registry {
	if engineRegistry != nil {
		return engineRegistry
	}
	return engine.DefaultRegistry()
}

func loadInstalledPacks(projectRoot string) ([]*pack.Manifest, error) {
	cfg, err := config.LoadConfigFromPath(filepath.Join(projectRoot, "backstop.yml"))
	if err != nil {
		return nil, fmt.Errorf("loading backstop.yml: %w", err)
	}

	packNames := declaredPackNames(cfg)
	if len(packNames) == 0 {
		return []*pack.Manifest{}, nil
	}

	packsDir := filepath.Join(projectRoot, ".backstop", "packs")
	manifests := make([]*pack.Manifest, 0, len(packNames))
	for _, packName := range packNames {
		packPath := filepath.Join(packsDir, filepath.FromSlash(packName))
		info, statErr := os.Stat(packPath)
		if statErr != nil || !info.IsDir() {
			return nil, fmt.Errorf("declared pack %s is missing from %s", packName, packsDir)
		}

		manifestPath := filepath.Join(packPath, "pack.yml")
		manifest, parseErr := pack.ParseManifestFile(manifestPath)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", manifestPath, parseErr)
		}
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func verifyPackLock(projectRoot string, packs []string) error {
	if len(packs) == 0 {
		return nil
	}

	lockPath := filepath.Join(projectRoot, "backstop.lock")
	var lockfile *distribution.Lockfile
	lockfile, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading lockfile: %w", err)
		}
		lockfile = nil
	}

	verifyResult, err := distribution.VerifyLock(lockfile, filepath.Join(projectRoot, ".backstop", "packs"), packs)
	if err != nil {
		return fmt.Errorf("verifying lockfile: %w", err)
	}
	if verifyResult.Pass {
		return nil
	}

	parts := make([]string, 0, len(verifyResult.Failures))
	for _, failure := range verifyResult.Failures {
		if failure.Pack == "" {
			parts = append(parts, fmt.Sprintf("%s: %s", failure.Kind, failure.Message))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s): %s", failure.Kind, failure.Pack, failure.Message))
	}
	return fmt.Errorf("pack lock verification failed: %s", strings.Join(parts, "; "))
}

func declaredPackNames(cfg *config.Config) []string {
	if cfg == nil || len(cfg.Packs) == 0 {
		return nil
	}
	names := make([]string, 0, len(cfg.Packs))
	for ref := range cfg.Packs {
		names = append(names, ref)
	}
	sort.Strings(names)
	return names
}

// dispatchPackEngines is the group-by-engine gate-time dispatch (SPEC-031
// REQ-001/REQ-011/REQ-014). It replaces BOTH the layer-2 mergePackRules ->
// ExtraSemgrepConfigs -> semgrepExecutor findings feeder AND the layer-3
// runPackValidators sandbox feeder, re-keyed from rule.Layer to the rule's
// declared engine. It groups every installed-pack rule by its declared engine,
// looks up each EngineBinding, and runs each engine once:
//   - findings engines (semgrep, ast-grep, config-file): gather inputs per
//     input_mode, run the command via the clean-stdout runner, pipe through the
//     sandboxed convert step when the binding declares one, and parse the
//     normalized output exclusively via parseSarif (resolved through
//     check.ParsePackFindings / lookupParser — dispatch owns no engine
//     enumeration). Violations are namespaced pack-name/rule-id.
//   - the sandbox engine (input_mode none): takes the exit-code terminal branch
//     relocated from runPackValidators (rule.Layer==3), preserving the
//     single-file/multi-file fan-out and SandboxedRun (CombinedOutput) capture;
//     it never enters convert or parseSarif.
//
// An unknown engine, an unknown input_mode, or a missing declared input path is
// a fail-loud config error naming the pack — never a silent skip or inferred
// fallback.
// The scope parameter carries the gate's changed-file diff scope (untracked
// inclusive) so rule-fed findings engines scan only the in-scope changed files
// instead of the whole repository (ISSUE-010). A nil scope or a
// GateScopeModeAll scope is the explicit whole-repo escape hatch; `backstop code
// check` and `gate --all` pass that sentinel. The project-wide toolchain branch
// (go build/test ./..., golangci-lint run ./...) is unaffected by scope — it
// stays project-wide so unchanged-file breakage still fails the gate.
func dispatchPackEngines(packs []*pack.Manifest, packDir, projectRoot string, scope *gate.GateScope, runner check.CommandRunner) ([]gate.Violation, error) {
	violations := []gate.Violation{}
	for _, manifest := range packs {
		packRoot := filepath.Join(packDir, filepath.FromSlash(manifest.NormalizedName))

		// Group this pack's rules by declared engine, preserving order.
		grouped := map[string][]pack.Rule{}
		order := []string{}
		for _, rule := range manifest.Content.Ruleset.Rules {
			if _, seen := grouped[rule.Engine]; !seen {
				order = append(order, rule.Engine)
			}
			grouped[rule.Engine] = append(grouped[rule.Engine], rule)
		}

		for _, engineName := range order {
			binding, lookupErr := resolveEngineRegistry().Lookup(engineName)
			if lookupErr != nil {
				return nil, fmt.Errorf("pack %s: %w", manifest.NormalizedName, lookupErr)
			}
			rules := grouped[engineName]

			// The exit-code terminal branch (sandbox) is the validator-driven
			// engine: input_mode none AND no command (the executable is the pack
			// validator). The native go-build/go-test engines also declare
			// input_mode none but DO carry a command + convert script, so they ride
			// the findings path below — keyed off Command, not input_mode alone
			// (SPEC-034 REQ-001/REQ-004).
			if binding.InputMode == engine.InputModeNone && binding.Command == "" {
				vs, err := runSandboxEngine(manifest, packRoot, projectRoot, rules)
				if err != nil {
					return nil, fmt.Errorf("dispatching sandbox engine %q for pack %s: %w", engineName, manifest.NormalizedName, err)
				}
				violations = append(violations, vs...)
				continue
			}

			// Findings engine: gather inputs, run, convert, parseSarif.
			vs, err := runFindingsEngine(manifest, packRoot, projectRoot, scope, binding, rules, runner)
			if err != nil {
				return nil, fmt.Errorf("dispatching findings engine %q for pack %s: %w", engineName, manifest.NormalizedName, err)
			}
			violations = append(violations, vs...)
		}
	}
	return violations, nil
}

// gatherEngineInputs shapes the engine invocation's input arguments from the
// declared input_mode (REQ-020/REQ-021). Inputs resolve relative to the
// per-engine pack directory; a missing declared rule path is a fail-loud
// broken-pack error naming the pack and the path (REQ-021/CLM-050). A
// config-file engine with no pack config injects nothing (Sharp Edge 4: "no
// config offered" is legitimate, distinct from "declared path absent").
func gatherEngineInputs(manifest *pack.Manifest, packRoot string, binding engine.EngineBinding, rules []pack.Rule) ([]string, error) {
	switch binding.InputMode {
	case engine.InputModeConfigFile:
		// One optional pack config: the first rule that declares a rule_path
		// supplies it; absence is fine (the tool runs its own rules).
		for _, rule := range rules {
			if rule.RulePath == "" {
				continue
			}
			path, err := resolveRulePath(manifest, packRoot, rule)
			if err != nil {
				return nil, fmt.Errorf("gathering config-file input for rule %s: %w", rule.ID, err)
			}
			return []string{binding.InputFlag, path}, nil
		}
		return nil, nil
	case engine.InputModeRuleFlags:
		seen := map[string]struct{}{}
		paths := []string{}
		for _, rule := range rules {
			path, err := resolveRulePath(manifest, packRoot, rule)
			if err != nil {
				return nil, fmt.Errorf("gathering rule-flag input for rule %s: %w", rule.ID, err)
			}
			if _, dup := seen[path]; dup {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
		sort.Strings(paths)
		args := make([]string, 0, len(paths)*2)
		for _, p := range paths {
			args = append(args, binding.InputFlag, p)
		}
		return args, nil
	case engine.InputModeRuleDir:
		// Collect each rule's directory; ast-grep scans a directory passed once.
		seen := map[string]struct{}{}
		dirs := []string{}
		for _, rule := range rules {
			path, err := resolveRulePath(manifest, packRoot, rule)
			if err != nil {
				return nil, fmt.Errorf("gathering rule-dir input for rule %s: %w", rule.ID, err)
			}
			dir := filepath.Dir(path)
			if _, dup := seen[dir]; dup {
				continue
			}
			seen[dir] = struct{}{}
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		args := make([]string, 0, len(dirs)*2)
		for _, d := range dirs {
			args = append(args, binding.InputFlag, d)
		}
		return args, nil
	case engine.InputModeNone:
		return nil, nil
	default:
		return nil, fmt.Errorf("pack %s: unknown input_mode %q", manifest.NormalizedName, binding.InputMode)
	}
}

// resolveRulePath resolves a rule's declared input path relative to the
// per-engine pack directory and fail-louds if it is absent on disk
// (REQ-021/CLM-049/CLM-050).
func resolveRulePath(manifest *pack.Manifest, packRoot string, rule pack.Rule) (string, error) {
	if rule.RulePath == "" {
		return "", fmt.Errorf("broken pack %s: rule %s declares no rule_path", manifest.NormalizedName, rule.ID)
	}
	rulePath := filepath.Join(packRoot, filepath.FromSlash(rule.RulePath))
	abs, err := filepath.Abs(rulePath)
	if err != nil {
		return "", fmt.Errorf("resolving rule path for %s: %w", manifest.NormalizedName, err)
	}
	info, statErr := os.Stat(abs)
	if statErr != nil || info.IsDir() {
		return "", fmt.Errorf("broken pack %s: missing rule file %s", manifest.NormalizedName, abs)
	}
	return abs, nil
}

// runFindingsEngine runs one findings engine: gather inputs, run the command via
// the clean-stdout runner, pipe through the sandboxed convert when declared, and
// parse the normalized SARIF (REQ-006/REQ-007/REQ-009). Violations are
// namespaced to the pack.
func runFindingsEngine(manifest *pack.Manifest, packRoot, projectRoot string, scope *gate.GateScope, binding engine.EngineBinding, rules []pack.Rule, runner check.CommandRunner) ([]gate.Violation, error) {
	inputs, err := gatherEngineInputs(manifest, packRoot, binding, rules)
	if err != nil {
		return nil, fmt.Errorf("gathering engine inputs for pack %s: %w", manifest.NormalizedName, err)
	}

	cmdName, cmdArgs := splitCommand(binding.Command)
	cmdArgs = append(cmdArgs, inputs...)
	// Scope-kind-aware arg-shaping (SPEC-034 REQ-010/CLM-034, N1; ISSUE-010).
	// Rule-fed findings engines (semgrep --config X <targets>, ast-grep scan
	// --rule DIR <targets>) and config-file engines with no self-declared target
	// scan the files they are pointed at. A project-wide toolchain pass (go build
	// ./..., go test ./..., golangci-lint run ./...) shapes its OWN target via
	// ProjectTarget and must NOT have a scan target bolted on — appending <root>
	// to `go build ./...` is wrong (CLM-005, Ratified Design Constraint 3).
	if binding.ScopeKind == engine.ScopeKindProjectWide && binding.ProjectTarget != "" {
		cmdArgs = append(cmdArgs, binding.ProjectTarget)
	} else {
		// Diff-scope the rule-fed engine to the gate's changed files (ISSUE-010).
		// A nil scope or GateScopeModeAll is the explicit whole-repo escape hatch
		// (gate --all, code check) — scan projectRoot exactly as before (CLM-004).
		// Otherwise point the engine at ONLY the in-scope changed files (untracked
		// included, project-relative as they arrive from the gate scope) so it
		// never produces out-of-scope findings (CLM-001/CLM-002/CLM-007). When the
		// resulting target list is empty, append NOTHING — the engine scans
		// nothing and yields zero findings; it must NOT silently fall back to the
		// whole repo (CLM-003).
		if scope == nil || scope.Mode == gate.GateScopeModeAll {
			cmdArgs = append(cmdArgs, projectRoot)
		} else {
			cmdArgs = append(cmdArgs, scope.Files...)
		}
	}

	stdout, runErr := runner.RunStdout(context.Background(), cmdName, cmdArgs...)
	// A rule-fed/config-file findings engine exits non-zero when it reports
	// findings; the SARIF on stdout is the contract, so runErr is not fatal on
	// its own. A CrashGuard engine (native build/test) is different: a non-zero
	// exit that yields NO parseable findings is a tool/infra crash, not a
	// finding-free pass, and must fail loud rather than read as a silent green
	// (SPEC-034 REQ-003/CLM-010). runErr was discarded before this bridge.

	sarifBytes := stdout
	if binding.Convert != "" {
		convertPath := filepath.Join(packRoot, filepath.FromSlash(binding.Convert))
		if info, statErr := os.Stat(convertPath); statErr != nil || info.IsDir() {
			return nil, fmt.Errorf("broken pack %s: missing convert script %s", manifest.NormalizedName, convertPath)
		}
		converted, convErr := resolveSandboxedRunStdout()(convertPath, nil, packRoot, stdout)
		if convErr != nil {
			return nil, fmt.Errorf("pack %s: convert step (%s) failed: %w", manifest.NormalizedName, binding.Convert, convErr)
		}
		sarifBytes = converted
	}

	checkViolations, parseErr := check.ParsePackFindings(sarifBytes)
	if parseErr != nil {
		return nil, fmt.Errorf("pack %s engine %s: convert/parse to SARIF failed: %w", manifest.NormalizedName, binding.Command, parseErr)
	}

	// Crash-vs-findings guard (SPEC-034 REQ-003/CLM-010). For a CrashGuard engine
	// a non-zero run with zero parseable findings is a compiler/test-binary crash
	// or unparseable output, not a clean pass — surface it (naming the pack and
	// engine) instead of returning a silent green.
	if binding.CrashGuard && runErr != nil && len(checkViolations) == 0 {
		return nil, fmt.Errorf("pack %s engine %q crashed: non-zero exit with no parseable findings: %w", manifest.NormalizedName, binding.Command, runErr)
	}

	out := make([]gate.Violation, 0, len(checkViolations))
	for _, v := range checkViolations {
		out = append(out, gate.Violation{
			Rule:       pack.NamespacedRuleID(manifest.NormalizedName, v.Rule),
			File:       v.File,
			Message:    v.Message,
			Severity:   nonEmpty(v.Severity, "error"),
			SourcePack: manifest.NormalizedName,
		})
	}
	return out, nil
}

// runSandboxEngine is the exit-code terminal branch relocated from
// runPackValidators (REQ-014/CLM-066/CLM-067), re-keyed from rule.Layer==3 to
// engine==sandbox. It preserves the single-file/multi-file target fan-out and
// the SandboxedRun (CombinedOutput) capture as the violation message body, and
// never enters convert or parseSarif.
func runSandboxEngine(manifest *pack.Manifest, packRoot, projectRoot string, rules []pack.Rule) ([]gate.Violation, error) {
	violations := []gate.Violation{}
	for _, rule := range rules {
		validatorPath := filepath.Join(packRoot, filepath.FromSlash(rule.Validator))
		info, statErr := os.Stat(validatorPath)
		if statErr != nil || info.IsDir() {
			return nil, fmt.Errorf("broken pack %s: missing validator %s", manifest.NormalizedName, validatorPath)
		}

		targets := []string{projectRoot}
		if rule.InputScope == "single-file" {
			targets = []string{}
			walkErr := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				targets = append(targets, path)
				return nil
			})
			if walkErr != nil {
				return nil, fmt.Errorf("walking project files for %s: %w", manifest.NormalizedName, walkErr)
			}
		}

		for _, target := range targets {
			output, err := resolveSandboxedRun()(validatorPath, []string{target}, packRoot)
			if err == nil {
				continue
			}

			ruleID := rule.NamespacedID
			if ruleID == "" {
				ruleID = pack.NamespacedRuleID(manifest.NormalizedName, rule.ID)
			}
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			violations = append(violations, gate.Violation{
				Rule:       ruleID,
				Message:    message,
				SourcePack: manifest.NormalizedName,
				Severity:   "error",
			})
		}
	}
	return violations, nil
}

// splitCommand splits a binding's Command string ("semgrep", "ast-grep scan")
// into the executable name and its leading subcommand args.
func splitCommand(command string) (string, []string) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
