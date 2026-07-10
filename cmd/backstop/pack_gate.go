package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmanson/backstop-core/pkg/baseengines"
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
// packval.SandboxedRun, packval.SandboxedRunStdout, and the embedded base-engines
// pack via baseengines.Registry), and overridden by tests to substitute a stub
// sandbox or inject a custom engine binding. They are declared WITHOUT initializers
// so they hold no package-level mutable default; the real implementation is resolved
// lazily at the call site.
var (
	sandboxedRun       func(cmd string, args []string, packDir string) ([]byte, error)
	sandboxedRunStdout func(cmd string, args []string, packDir string, stdin []byte) ([]byte, error)
	engineRegistry     engine.Registry
	// trustedToolAllowlist is a test seam for the backstop-owned trusted-tool
	// allowlist (SPEC-035 REQ-002): nil in production (resolveTrustedToolAllowlist
	// falls back to engine.TrustedToolAllowlist), overridden by tests to drive the
	// allowlist matrix from a fixture WITHOUT stubbing the allowlist OPEN on the
	// dispatch path (Sharp Edge 3 forbids a stub-open allowlist). Declared WITHOUT
	// an initializer so it holds no package-level mutable default; the real
	// allowlist is resolved lazily at the call site (same shape as the other seams).
	trustedToolAllowlist func() map[string]string
)

// resolveTrustedToolAllowlist returns the injected allowlist seam or the concrete
// engine.TrustedToolAllowlist — the backstop-owned trust floor every pack-declared
// command's tool must satisfy before backstop runs it (SPEC-035 REQ-002).
func resolveTrustedToolAllowlist() map[string]string {
	if trustedToolAllowlist != nil {
		return trustedToolAllowlist()
	}
	return engine.TrustedToolAllowlist()
}

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

// resolveEngineRegistry returns the registry a rule's engine resolves through:
// the built-in registry (the injected engineRegistry seam or, in production, the
// embedded base-engines pack via baseengines.Registry) MERGED with the manifest's
// pack-declared engines: block, with a pack-declared binding OVERRIDING a same-named
// built-in (SPEC-035 REQ-001/CLM-002/CLM-004). The merge is what makes a
// pack-declared command REACHABLE; it ships ATOMICALLY with the dispatch trust gate
// (runFindingsEngine) so no pack-declared command is ever runnable without the gate
// (Sharp Edge 1). A nil manifest yields just the built-in registry (the callers that
// only inspect built-in bindings pass nil). The result is a fresh map so a merge
// never mutates the seam or the shared base table. There is NO baked
// engine.DefaultRegistry fallback (ISSUE-027): the built-ins are pack DATA sourced
// from the embedded base pack.
func resolveEngineRegistry(manifest *pack.Manifest) engine.Registry {
	base := engineRegistry
	if base == nil {
		base = baseengines.Registry()
	}
	merged := make(engine.Registry, len(base))
	for name, binding := range base {
		merged[name] = binding
	}
	if manifest != nil {
		for name, spec := range manifest.Engines {
			merged[name] = spec.Binding
		}
	}
	return merged
}

// gateTypeHasDedicatedStep reports whether a gate_type is dispatched by its OWN
// gate step (contract_signature for contracts, test_substantiveness for
// substantiveness, coverage_threshold for coverage). Engines of these types must
// NOT also run through the generic pack_engines/code_check findings dispatch: that
// path scans context-free over the whole project, so a traceability rule (a
// pattern-arg ast-grep/grep probe, or a hollow-test rule) fired without its
// per-dimension driver matches everything and emits garbage findings.
func gateTypeHasDedicatedStep(gt engine.GateType) bool {
	switch gt {
	case engine.GateTypeSubstantiveness, engine.GateTypeContracts, engine.GateTypeCoverage:
		return true
	default:
		return false
	}
}

// excludeDedicatedStepRules returns the manifests with every rule whose resolved
// engine declares a dedicated-step gate_type removed, so the generic
// pack_engines/findings dispatch runs ONLY the generic stages
// (lint/build/test/findings). The dedicated gate steps dispatch the excluded
// engines per-dimension themselves. Routing is by DECLARED gate_type — no pack name
// is hardcoded, so any pack (incl. a third party's) that fills a dedicated dimension
// is partitioned correctly. A manifest with no excluded rules is returned unchanged.
func excludeDedicatedStepRules(packs []*pack.Manifest) []*pack.Manifest {
	out := make([]*pack.Manifest, 0, len(packs))
	for _, m := range packs {
		reg := resolveEngineRegistry(m)
		kept := make([]pack.Rule, 0, len(m.Content.Ruleset.Rules))
		dropped := false
		for _, rule := range m.Content.Ruleset.Rules {
			if b, err := reg.Lookup(rule.Engine); err == nil && gateTypeHasDedicatedStep(b.GateType) {
				dropped = true
				continue
			}
			kept = append(kept, rule)
		}
		if !dropped {
			out = append(out, m)
			continue
		}
		clone := *m
		clone.Content.Ruleset.Rules = kept
		out = append(out, &clone)
	}
	return out
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
// REQ-001/REQ-011/REQ-014). It replaces BOTH the layer-2 mergePackRules
// in-process findings feeder AND the layer-3 runPackValidators sandbox feeder,
// re-keyed from rule.Layer to the rule's declared engine. It groups every
// installed-pack rule by its declared engine,
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
// dispatchPackEnginesFn is a test seam: nil in production (resolveDispatchPackEngines
// falls back to the concrete dispatchPackEngines), overridden by tests to inject
// a hermetic stub. It is declared WITHOUT an initializer so it holds no
// package-level mutable default. It is the gate's dispatch seam (used by the
// pack_engines / substantiveness / contracts steps in gate.go).
var dispatchPackEnginesFn func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error)

// resolveDispatchPackEngines returns the injected dispatch seam or the concrete
// dispatchPackEngines.
func resolveDispatchPackEngines() func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
	if dispatchPackEnginesFn != nil {
		return dispatchPackEnginesFn
	}
	return dispatchPackEngines
}

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

		registry := resolveEngineRegistry(manifest)
		for _, engineName := range order {
			binding, lookupErr := registry.Lookup(engineName)
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

// dispatchPackCoverage is the coverage-records dispatch channel (SPEC-042
// REQ-001/REQ-007) — the SECOND normalized output type, DISTINCT from the SARIF
// findings channel dispatchPackEngines/runFindingsEngine drive. For each
// installed-pack engine whose binding declares gate_type coverage
// (engine.GateTypeCoverage), it runs the engine command via the runner, pipes the
// engine's stdout through the pack's declared convert via resolveSandboxedRunStdout
// (the SAME sandboxed-convert seam runFindingsEngine reuses), and parses the
// normalized output via check.ParsePackCoverage — TERMINATING in the coverage-records
// parser, NOT ParsePackFindings (CLM-001/CLM-002).
//
// Routing is PURELY on the binding's DECLARED GateType — no command-string sniff,
// no pack-name check, no engine-name check — so a lint/build/test/findings engine is
// never parsed as coverage-records (CLM-003..CLM-006) and a coverage engine is never
// parsed as SARIF (CLM-002). The records are a DISTINCT typed output
// ([]check.CoverageRecord), never coverage tunneled through SARIF properties: the
// parser rejects a SARIF/object document (CLM-007). The path bakes NO Go-coverage
// profile knowledge of any kind — that lives in the pack DATA (binding command +
// convert script), keeping the dispatch tool/language-blind (CLM-022).
//
// The scope parameter is accepted for signature parity with dispatchPackEngines
// (and so a future scoped coverage pass can shape its own target); coverage engines
// are project-wide toolchain passes (ScopeKindProjectWide + ProjectTarget), so the
// engine shapes its OWN target and the project root is never appended.
func dispatchPackCoverage(packs []*pack.Manifest, packDir, projectRoot string, scope *gate.GateScope, runner check.CommandRunner) ([]check.CoverageRecord, error) {
	_ = scope
	records := []check.CoverageRecord{}
	for _, manifest := range packs {
		packRoot := filepath.Join(packDir, filepath.FromSlash(manifest.NormalizedName))

		grouped := map[string][]pack.Rule{}
		order := []string{}
		for _, rule := range manifest.Content.Ruleset.Rules {
			if _, seen := grouped[rule.Engine]; !seen {
				order = append(order, rule.Engine)
			}
			grouped[rule.Engine] = append(grouped[rule.Engine], rule)
		}

		registry := resolveEngineRegistry(manifest)
		for _, engineName := range order {
			binding, lookupErr := registry.Lookup(engineName)
			if lookupErr != nil {
				return nil, fmt.Errorf("pack %s: %w", manifest.NormalizedName, lookupErr)
			}
			// Route SOLELY on the declared GateType. A non-coverage engine is skipped
			// here — it belongs to the SARIF findings channel (dispatchPackEngines),
			// never the records channel.
			if binding.GateType != engine.GateTypeCoverage {
				continue
			}
			rules := grouped[engineName]
			recs, err := runCoverageEngine(manifest, packRoot, projectRoot, binding, rules, runner)
			if err != nil {
				return nil, fmt.Errorf("dispatching coverage engine %q for pack %s: %w", engineName, manifest.NormalizedName, err)
			}
			records = append(records, recs...)
		}
	}
	return records, nil
}

// configErrorPassthrough returns a dispatch trust-gate error UNCHANGED, preserving
// its concrete *check.ConfigError type so the exit-code-2 type assertion downstream
// (code_check.go) still fires. It exists so the trust gate's already-contextualized
// ConfigError is passed through verbatim rather than %w-wrapped (which would change
// the concrete type and silently demote the exit code). The error is fully formed by
// checkEngineToolAllowed (it names the tool + pack), so no added context is warranted.
func configErrorPassthrough(err error) error {
	return err
}

// runCoverageEngine runs one coverage engine: the trust gate, gather inputs, run the
// command via the clean-stdout runner, pipe through the pack's declared convert via
// resolveSandboxedRunStdout, and parse the normalized coverage-records JSON via
// check.ParsePackCoverage (SPEC-042 REQ-001/REQ-007). It is the coverage analogue of
// runFindingsEngine's run->convert step, terminating in the records parser instead of
// ParsePackFindings. A coverage engine declaring no convert is a broken-pack error:
// the engine's native profile is not coverage-records, so a convert is required to
// normalize it.
func runCoverageEngine(manifest *pack.Manifest, packRoot, projectRoot string, binding engine.EngineBinding, rules []pack.Rule, runner check.CommandRunner) ([]check.CoverageRecord, error) {
	inputs, err := gatherEngineInputs(manifest, packRoot, binding, rules)
	if err != nil {
		return nil, fmt.Errorf("gathering coverage engine inputs for pack %s: %w", manifest.NormalizedName, err)
	}

	// The SAME dispatch trust gate the findings path runs (SPEC-035 REQ-002): an
	// un-allowlisted/unpinned provisioned tool's command is never handed to the runner.
	// The gate returns a *check.ConfigError (exit 2) that MUST pass through with its
	// concrete type intact (code_check.go type-asserts it), so it is returned UNWRAPPED
	// — never %w-wrapped, which would defeat the exit-code type assertion.
	if gateErr := checkEngineToolAllowed(manifest, binding); gateErr != nil {
		return nil, configErrorPassthrough(gateErr)
	}

	// PRODUCER vs plain-command split (ISSUE-045 option (ii)). The producer runs
	// UN-SANDBOXED (via the runner, full toolchain — it resolves the module + gofile
	// list and folds them into the payload); the convert below runs SANDBOXED (parse
	// only). This split is what keeps BOTH the executor language-blind AND the convert
	// toolchain-free: the producer path is pack DATA (binding.Producer), never a
	// language/tool literal in the dispatch.
	var stdout []byte
	if binding.Producer != "" {
		// A pack-declared producer resolved under packRoot (the SAME
		// filepath.Join(packRoot, …)+os.Stat pattern Convert uses) and run
		// UN-SANDBOXED via the runner (cwd = projectRoot, so its `go test`/`go list`
		// see the project). It shapes its OWN invocation — no inputs/ProjectTarget
		// bolted on. A declared-but-missing producer is a fail-loud broken-pack error
		// naming pack + path, mirroring the convert-missing error.
		producerPath := filepath.Join(packRoot, filepath.FromSlash(binding.Producer))
		if info, statErr := os.Stat(producerPath); statErr != nil || info.IsDir() {
			return nil, fmt.Errorf("broken pack %s: missing coverage producer script %s", manifest.NormalizedName, producerPath)
		}
		out, runErr := runner.RunStdout(context.Background(), producerPath)
		// A coverage producer may exit non-zero when tests fail yet still emit a usable
		// profile, so runErr is not fatal on its own — the convert+parser contract is
		// what matters. A convert failure below fails loud.
		_ = runErr
		stdout = out
	} else {
		cmdName, cmdArgs := splitCommand(binding.Command)
		cmdArgs = append(cmdArgs, inputs...)
		// A coverage engine is a project-wide toolchain pass: it shapes its OWN target
		// (ProjectTarget) and the project root is never bolted on, exactly as the
		// project-wide branch of runFindingsEngine does.
		if binding.ScopeKind == engine.ScopeKindProjectWide && binding.ProjectTarget != "" {
			cmdArgs = append(cmdArgs, binding.ProjectTarget)
		}
		out, runErr := runner.RunStdout(context.Background(), cmdName, cmdArgs...)
		_ = runErr
		stdout = out
	}

	// Select the bytes to feed the convert. By default the engine's payload IS its
	// stdout. When the binding declares a stdout_artifact, the engine instead writes
	// its real output to that FILE (relative to the run's working dir = projectRoot)
	// and prints only summary/noise to stdout — so read the FILE and feed THAT. The
	// filename is pack DATA; this stays tool/language-blind (no coverage/profile
	// literal here). A declared-but-missing artifact is a fail-loud broken run, not a
	// silent fall-back to the noise stdout (which would re-introduce the bug).
	payload := stdout
	if binding.StdoutArtifact != "" {
		artifactPath := filepath.Join(projectRoot, filepath.FromSlash(binding.StdoutArtifact))
		body, readErr := os.ReadFile(artifactPath)
		if readErr != nil {
			return nil, fmt.Errorf("pack %s coverage engine %q: declared stdout_artifact %q not produced (read %s): %w", manifest.NormalizedName, binding.Command, binding.StdoutArtifact, artifactPath, readErr)
		}
		payload = body
	}

	if binding.Convert == "" {
		return nil, fmt.Errorf("broken pack %s: coverage engine %q declares no convert — its native profile is not coverage-records and must be normalized", manifest.NormalizedName, binding.Command)
	}
	convertPath := filepath.Join(packRoot, filepath.FromSlash(binding.Convert))
	if info, statErr := os.Stat(convertPath); statErr != nil || info.IsDir() {
		return nil, fmt.Errorf("broken pack %s: missing coverage convert script %s", manifest.NormalizedName, convertPath)
	}
	normalized, convErr := resolveSandboxedRunStdout()(convertPath, nil, packRoot, payload)
	if convErr != nil {
		return nil, fmt.Errorf("pack %s: coverage convert step (%s) failed: %w", manifest.NormalizedName, binding.Convert, convErr)
	}

	records, parseErr := check.ParsePackCoverage(normalized)
	if parseErr != nil {
		return nil, fmt.Errorf("pack %s coverage engine %s: convert/parse to coverage-records failed: %w", manifest.NormalizedName, binding.Command, parseErr)
	}
	return records, nil
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
	case engine.InputModePatternArg:
		// Pattern-arg engines pass each rule's inline pattern as a command
		// argument via InputFlag (the BUNDLE-009 seam, REQ-004/CLM-016). The
		// pattern is the literal value — NOT a rule-file path — so this case does
		// NO filesystem rule-path resolution and never os.Stats a path
		// (CLM-018): a rule carrying a rule_path is ignored here. An EMPTY pattern
		// is a broken-pack config error naming pack + rule, parallel to
		// resolveRulePath's missing-path fail-loud (CLM-017) — never a silently
		// emitted empty arg.
		args := make([]string, 0, len(rules)*2)
		for _, rule := range rules {
			if rule.Pattern == "" {
				return nil, fmt.Errorf("broken pack %s: pattern-arg rule %s declares no pattern", manifest.NormalizedName, rule.ID)
			}
			args = append(args, binding.InputFlag, rule.Pattern)
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

	// TRUST GATE (SPEC-035 REQ-002, Sharp Edge 1) — the ATOMIC half that makes a
	// pack-declared command SAFE. This sits BEFORE splitCommand(binding.Command) ->
	// runner.RunStdout so an un-allowlisted/unpinned tool's command is NEVER handed
	// to the runner (CLM-005..008): the merge above (resolveEngineRegistry) makes a
	// pack-declared command reachable, and this gate ensures it cannot run un-trusted.
	// The lockedVersion fed into CheckToolAllowed is the binding's Provision.Version
	// — the version the backstop.lock / VerifyLock / Provision path pins — NOT a
	// second literal, so the pin rides the lock and cannot drift (CLM-029).
	if gateErr := checkEngineToolAllowed(manifest, binding); gateErr != nil {
		return nil, gateErr
	}

	cmdName, cmdArgs := splitCommand(binding.Command)
	cmdArgs = append(cmdArgs, inputs...)
	// Scope-kind-aware arg-shaping (SPEC-034 REQ-010/CLM-034, N1; ISSUE-010).
	// Rule-fed findings engines (semgrep --config X <targets>, ast-grep scan
	// --config sgconfig.yml <targets>) and config-file engines with no self-declared target
	// scan the files they are pointed at. A project-wide toolchain pass (go build
	// ./..., go test ./..., golangci-lint run ./...) shapes its OWN target via
	// ProjectTarget and must NOT have a scan target bolted on — appending <root>
	// to `go build ./...` is wrong (CLM-005, Ratified Design Constraint 3).
	if binding.ScopeKind == engine.ScopeKindProjectWide {
		// A project-wide toolchain pass shapes its OWN target and must NOT have a
		// scan target bolted on. When it declares a ProjectTarget (go build ./...,
		// golangci-lint run ./...), append that. When it declares NONE, the engine
		// self-targets — tsc --noEmit reads tsconfig.json, `bun test` discovers its
		// own tests — and appending <projectRoot> is WRONG: `tsc --noEmit <dir>`
		// treats the path as a file and IGNORES tsconfig, silently typechecking
		// nothing (vacuous green). So append nothing for an empty ProjectTarget
		// (SPEC-048 REQ-001/CLM-001, DEFECT-1 fix).
		if binding.ProjectTarget != "" {
			// File-mode go-test PACKAGE scoping (SPEC-034 REQ-010/CLM-034, N1): the
			// `code check --file` hook scopes `go test` to the changed file's package,
			// not ./..., to stay within its tight budget. fileModeTestTarget returns the
			// package selector ONLY for the native go-test engine under a file-mode
			// scope (pack_gate_filemode.go); every other project-wide pass keeps its
			// ProjectTarget so unchanged-file breakage still fails a full run.
			if target, ok := fileModeTestTarget(binding, scope); ok {
				cmdArgs = append(cmdArgs, target)
			} else {
				cmdArgs = append(cmdArgs, binding.ProjectTarget)
			}
		}
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
			// Drop any changed path under a `testdata` directory before it becomes
			// an engine scan target (ISSUE-040). Standard tooling convention treats a
			// `testdata` directory as inert data — deliberately-hollow negative
			// fixtures and planted rule-violations live there and are NOT real
			// findings. This narrows the diff-scoped target list; it does NOT widen
			// it. If the result is empty (a testdata-only diff), NOTHING is appended
			// and the engine scans nothing — it must NEVER fall through to the
			// projectRoot whole-repo branch above (the ISSUE-010 CLM-003
			// anti-fallback contract, preserved).
			cmdArgs = append(cmdArgs, excludeTestdataPaths(scope.Files)...)
		}
	}

	stdout, runErr := runner.RunStdout(context.Background(), cmdName, cmdArgs...)
	// A rule-fed/config-file findings engine exits non-zero when it reports
	// findings; the SARIF on stdout is the contract, so runErr is not fatal on
	// its own. A CrashGuard engine (native build/test) is different: a non-zero
	// exit that yields NO parseable findings is a tool/infra crash, not a
	// finding-free pass, and must fail loud rather than read as a silent green
	// (SPEC-034 REQ-003/CLM-010). runErr was discarded before this bridge.

	// Select the bytes the convert/shape-guard see (mirrors runCoverageEngine). By
	// default a findings engine's payload IS its stdout. When the binding declares a
	// stdout_artifact, the engine writes its real machine-readable output to that FILE
	// (relative to the run's working dir = projectRoot) and prints only summary/noise
	// to stdout — e.g. `bun test` writes JUnit XML to a --reporter-outfile while its
	// stdout is a human summary, and reading stdout would find no <testcase> and
	// silently green a failing test suite. Read the FILE and feed THAT. The filename
	// is pack DATA; this stays tool/language-blind. A declared-but-missing artifact is
	// a fail-loud broken run, not a silent fall-back to the noise stdout (SPEC-048
	// REQ-002/CLM-005..008, DEFECT-2 fix).
	payload := stdout
	if binding.StdoutArtifact != "" {
		artifactPath := filepath.Join(projectRoot, filepath.FromSlash(binding.StdoutArtifact))
		body, readErr := os.ReadFile(artifactPath)
		if readErr != nil {
			return nil, fmt.Errorf("pack %s findings engine %q: declared stdout_artifact %q not produced (%s): %w", manifest.NormalizedName, binding.Command, binding.StdoutArtifact, artifactPath, readErr)
		}
		payload = body
	}

	// Strict-SARIF guard for a config-file engine that assumes a NATIVE SARIF tool
	// (golangci-lint v2): a v1/too-old binary emits non-SARIF JSON that the lenient
	// parser would silently read as zero findings — vacuous green. Fail loud,
	// engine-attributed, instead (SPEC-034 REQ-005/CLM-019, Sharp Edge 5). The
	// guard lives in pack_gate_golint.go and is a no-op unless binding.StrictSarif.
	if binding.Convert == "" {
		if shapeErr := requireLintSarifShape(manifest, binding, payload); shapeErr != nil {
			return nil, fmt.Errorf("validating SARIF shape for pack %s: %w", manifest.NormalizedName, shapeErr)
		}
	}

	sarifBytes := payload
	if binding.Convert != "" {
		convertPath := filepath.Join(packRoot, filepath.FromSlash(binding.Convert))
		if info, statErr := os.Stat(convertPath); statErr != nil || info.IsDir() {
			return nil, fmt.Errorf("broken pack %s: missing convert script %s", manifest.NormalizedName, convertPath)
		}
		converted, convErr := resolveSandboxedRunStdout()(convertPath, nil, packRoot, payload)
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

	// PERMANENT declared build-exemption bridge (SPEC-041 REQ-004/REQ-007/CLM-012,
	// Sharp Edge 5). Each produced gate.Violation.ProjectWide is stamped from ITS
	// producing binding's DECLARED ExemptFromScopeFilter value — the bridge the
	// engine path never had (it previously used ScopeKind only for arg-shaping and
	// never set ProjectWide). ProjectWide is consumed by pkg/gate/scope.go's
	// filterViolations to keep an exempt engine's UNCHANGED-file violation out of the
	// diff-scope filter, so an unchanged-file build break still REDs (CLM-013).
	//
	// This REPLACES the SPEC-040 transitional seam (`binding.GateType ==
	// engine.GateTypeBuild`): no GateType identity and no CheckType enum identity
	// decides scope — only the explicit per-binding property (CLM-017). go-build
	// declares it true; golangci/go-test/findings false/unset (CLM-014/015/016).
	// Resolution is PER-VIOLATION: each violation carries ITS binding's value, with
	// no gate-type-level aggregation (REQ-007/CLM-018). A true-conflict (same
	// file+line+rule from two sources with differing values) resolves to the
	// exempting value at the union of violations — the louder, safe-against-
	// under-broad-filtering direction (CLM-019).
	exempt := binding.ExemptFromScopeFilter
	out := make([]gate.Violation, 0, len(checkViolations))
	for _, v := range checkViolations {
		out = append(out, gate.Violation{
			Rule: pack.NamespacedRuleID(manifest.NormalizedName, v.Rule),
			// Canonicalize the SARIF-echoed path to ONE repo-relative form (ISSUE-046)
			// so every bridged violation carries a stable File everywhere it is
			// consumed — identity AND the raw-path consumers isExistingCodeViolation /
			// CompareBaseline's scope.Contains(old.File). projectRoot rel-ifies an
			// absolute artifactLocation.uri; a "./"-prefixed walk form collapses to the
			// explicit-arg form. The scope-branch invocation shape (:632-636) is
			// untouched — we normalize the OUTPUT path, never the engine INPUTS
			// (CLM-006, ISSUE-010 preserved).
			File:        gate.NormalizePath(projectRoot, v.File),
			// Carry the SARIF-reported start line so the SPEC-049 waiver
			// reconciliation can byte-scan the finding's own line for a @waiver token.
			// It rides through to gate.Violation.Line, which is line-INDEPENDENT of
			// baseline identity (json:"-").
			Line:        v.Line,
			Message:     v.Message,
			Severity:    nonEmpty(v.Severity, "error"),
			SourcePack:  manifest.NormalizedName,
			ProjectWide: exempt,
			RegionHash:  v.Fingerprint,
		})
	}
	return out, nil
}

// excludeTestdataPaths returns files minus any path that has a `testdata`
// directory SEGMENT (ISSUE-040). It is a pure path-string filter — no language,
// tool, or extension nouns — honoring the universal `testdata` convention
// (`go help packages`: a directory named testdata is inert data, never compiled or
// vetted) generically for ANY pack's rule-fed findings engine. Matching is on
// slash-split segment equality, so a look-alike file whose name merely contains
// the substring (e.g. "testdata_util.go", or a "mytestdata/" directory) is NOT
// excluded. Applied only INSIDE runFindingsEngine's diff-scope else branch; a
// testdata-only diff filters to an empty slice and the caller appends nothing.
func excludeTestdataPaths(files []string) []string {
	kept := make([]string, 0, len(files))
	for _, f := range files {
		if hasTestdataSegment(f) {
			continue
		}
		kept = append(kept, f)
	}
	return kept
}

// hasTestdataSegment reports whether path has a slash-separated segment exactly
// equal to "testdata" (an exact directory-segment match, not a substring).
func hasTestdataSegment(path string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if seg == "testdata" {
			return true
		}
	}
	return false
}

// checkEngineToolAllowed is the dispatch-time trust gate (SPEC-035 REQ-002): it
// returns a *check.ConfigError (exit 2) naming the tool and pack when the engine's
// tool is not on the trusted-tool allowlist OR is not lock-pinned to its
// allowlisted version, and nil when the gate passes. It is the dispatch half of the
// SAME engine.CheckToolAllowed check validateEngine and provisionEngines run, so an
// un-allowlisted tool is rejected at every resolveEngineRegistry caller that leads
// to running a command.
//
// The gate is keyed on the binding carrying a non-nil Provision — the
// backstop-introduced tools that ride the backstop.lock pin (semgrep, ast-grep, or
// any pack-declared provisioned tool). A nil-Provision binding is either the
// sandbox engine (no command at all — exempt, CLM-009), an assume-present Layer-0
// toolchain engine (go build/test, golangci — governed by provisionEngines'
// on-PATH fail-loud, not the allowlist+lock-pin), or the config-file shape with no
// command. The lockedVersion passed to CheckToolAllowed is Provision.Version (the
// lock-resolved pin), NOT a second literal (CLM-029).
func checkEngineToolAllowed(manifest *pack.Manifest, binding engine.EngineBinding) error {
	if binding.Provision == nil {
		return nil
	}
	if err := engine.CheckToolAllowed(
		resolveTrustedToolAllowlist(),
		binding.Provision.Tool,
		binding.Provision.Version,
	); err != nil {
		return &check.ConfigError{Message: fmt.Sprintf(
			"pack %s: %s", manifest.NormalizedName, err.Error(),
		)}
	}
	return nil
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
