package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/schema"
	"github.com/backstop-ai/backstop-core/pkg/validate"
	"github.com/backstop-ai/backstop-core/pkg/waiver"
	"github.com/spf13/cobra"
)

// GateResult is re-exported from pkg/gate for the contract.
type GateResult = gate.GateResult

// StepResult is re-exported from pkg/gate for the contract.
type StepResult = gate.StepResult

// newGateCommand creates the Cobra command for backstop gate.
func newGateCommand(jsonFlag *bool) *cobra.Command {
	var allFlag bool
	var fileFlag string
	var baseFlag string
	cmd := &cobra.Command{
		Use:   "gate [--all | --file FILE [FILE...] | --base REV]",
		Short: "Run full verification gate",
		Long: `Runs the complete backstop gate: the full reconciliation kill chain
that orchestrates artifact validation, code checking, test verification,
test substantiveness checks, coverage threshold verification, contract
signature verification, baseline comparison, waiver resolution, and ledger
integrity verification. This is the primary enforcement checkpoint — if
it's green, it ships.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGate(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&allFlag, "all", false, "run the full project sweep")
	cmd.Flags().StringVar(&fileFlag, "file", "", "scope gate to one or more explicit files")
	cmd.Flags().StringVar(&baseFlag, "base", "",
		"scope gate to files changed since the merge-base with REV, plus untracked files. "+
			"For CI: a fresh checkout has a clean working tree, so the default diff scope "+
			"resolves to nothing and checks zero files. Pass the pull-request base sha or the "+
			"push before-sha. An unresolvable REV is a config error, never a silent full sweep.")
	return cmd
}

// runGate is the Cobra RunE handler that orchestrates all nine gate steps.
func runGate(cmd *cobra.Command, args []string) error {
	// Load config via CLI foundation config loader.
	cfg, cfgErr := config.LoadConfig()

	// If config loading fails, return immediately with exit code 2.
	if cfgErr != nil {
		return &ExitCodeError{
			Code:    ExitConfigError,
			Message: fmt.Sprintf("config: %s", cfgErr),
		}
	}

	// Resolve project root from backstop.yml location.
	cfgPath, discoverErr := config.DiscoverConfigPath()
	projectRoot := "."
	if discoverErr == nil {
		projectRoot = filepath.Dir(cfgPath)
	}

	// THE ONE ARTIFACT-ROOT RESOLUTION for this run. Every consumer below reads this
	// value; a second resolution anywhere in the same run is the class of bug this
	// whole spec is about. Note projectRoot is the literal "." whenever config-path
	// discovery failed — ResolveRoot absolutizes it, which is what makes that safe.
	artifactRoot, rootErr := artifact.ResolveRoot(projectRoot, cfg.ArtifactRoot)
	if rootErr != nil {
		// THE FAILURE IS DISTINGUISHED BY TYPE, never by string match. The two error
		// types mean genuinely different things — a CONFIGURED root that is absent from
		// disk (REQ-008's loud failure: the consumer said where its artifacts live and
		// they are not there) versus a declared value that is malformed — and a caller
		// that had to parse messages could not tell them apart. SPEC-070's doctor
		// branches on exactly these two.
		//
		// An existing-but-EMPTY root is NEITHER of these: it resolves cleanly, and the
		// type-directory walkers' os.IsNotExist tolerance is what keeps it a pass.
		var missing *artifact.RootMissingError
		if errors.As(rootErr, &missing) {
			return &ExitCodeError{
				Code:    ExitConfigError,
				Message: fmt.Sprintf("config: configured artifact_root %q does not exist at %s — nothing was scanned", missing.Declared, missing.Path),
			}
		}
		var invalid *artifact.RootInvalidError
		if errors.As(rootErr, &invalid) {
			return &ExitCodeError{
				Code:    ExitConfigError,
				Message: fmt.Sprintf("config: invalid artifact_root %q: %s", invalid.Declared, invalid.Reason),
			}
		}
		return &ExitCodeError{
			Code:    ExitConfigError,
			Message: fmt.Sprintf("config: %s", rootErr),
		}
	}

	allFlag, allErr := cmd.Flags().GetBool("all")
	fileValue, fileErr := cmd.Flags().GetString("file")
	baseValue, baseErr := cmd.Flags().GetString("base")
	if flagErr := firstNonNil(allErr, fileErr, baseErr); flagErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", flagErr)}
	}
	// The three scope selectors are mutually exclusive. This EXTENDS the existing
	// check rather than adding a parallel one, so there is a single place that
	// decides which scope a run uses.
	if allFlag && fileValue != "" {
		return &ExitCodeError{Code: ExitConfigError, Message: "config: --all and --file are mutually exclusive"}
	}
	if baseValue != "" && allFlag {
		return &ExitCodeError{Code: ExitConfigError, Message: "config: --base and --all are mutually exclusive"}
	}
	if baseValue != "" && fileValue != "" {
		return &ExitCodeError{Code: ExitConfigError, Message: "config: --base and --file are mutually exclusive"}
	}

	scopeMode := gate.GateScopeModeDiff
	explicitFiles := []string{}
	if allFlag {
		scopeMode = gate.GateScopeModeAll
	}
	if fileValue != "" {
		scopeMode = gate.GateScopeModeFile
		explicitFiles = append([]string{fileValue}, args...)
	} else if len(args) > 0 {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: unexpected gate arguments: %s", strings.Join(args, " "))}
	}

	// A base-resolution failure surfaces as exit 2 carrying the scope layer's own
	// message unchanged — it already names the ref and why it could not resolve.
	// Exit 2 rather than 1 is the point: exit 1 would claim violations were found.
	scope, scopeErr := gate.ComputeGateScopeWithBase(projectRoot, scopeMode, explicitFiles, baseValue)
	if scopeErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", scopeErr)}
	}

	// Build gate with step implementations.
	var opts []gate.Option

	steps := buildGateSteps(projectRoot, artifactRoot, scope)
	opts = append(opts, gate.WithSteps(steps))
	opts = append(opts, gate.WithScope(scope))

	baselinePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	ttl, ttlErr := cfg.BaselineTTLDuration()
	if ttlErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %v", ttlErr)}
	}
	baselineArtifact, baselineWarning, baselineModTime := resolveBaselineCache(baselinePath, ttl)
	opts = append(opts, gate.WithBaseline(baselineArtifact), gate.WithBaselineWarning(baselineWarning), gate.WithBaselineCacheMeta(baselinePath, ttl, baselineModTime))

	// SPEC-049 REQ-016: ENABLE AND FEED the waiver reconciliation pass at the
	// shipped construction site (mirroring the WithBaseline call above), or the
	// shipped `backstop gate` stays dark and suppresses nothing. The production
	// Policy is EXTRACTED from the installed pack manifests (buildWaiverPolicy) —
	// the CLM-027 "declared, not hardcoded" mechanism realized in production — and
	// the LineReader yields RAW source bytes over the active scope.
	// A pack that's declared-but-not-installed (or an unreadable manifest) must NOT
	// abort the gate here — the gate's own pack-resolution reports that loudly with
	// output. Degrade to the severity-only policy (critical secrets stay non-waivable
	// regardless of packs) and let the run proceed. An uninstalled pack emits no
	// findings anyway, so it has nothing to protect via its declared non-waivable set.
	waiverPolicy, waiverPolicyErr := buildWaiverPolicy(projectRoot)
	if waiverPolicyErr != nil {
		waiverPolicy = waiver.NewDeclaredPolicy(nil, []string{"critical"})
	}
	opts = append(opts, gate.WithWaiver(buildWaiverLineReader(projectRoot, scope), waiverPolicy, time.Now()))

	allowSeeding, changedFiles := ruleSetChangeSeedingContext(projectRoot, scope)
	opts = append(opts, gate.WithRuleSetChangeSeedingAllowed(allowSeeding), gate.WithRuleSetChangeFiles(changedFiles))

	if policy := gatePolicyFromConfig(cfg); len(policy) > 0 {
		opts = append(opts, gate.WithPolicy(policy))
	}

	g := gate.New(opts...)
	result, exitCode := g.Run(context.Background())

	// ISSUE-059 provenance: stamp the HEAD SHA and completion time on the result before
	// formatting, mirroring the baseline artifact (gitSHA returns "" on a non-repo, with no
	// dirty check and no -dirty suffix). SchemaVersion stays "gate/v1" — this is additive.
	result.GitSHA = gitSHA(projectRoot)
	result.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

	// SPEC-068 identity + root, stamped the same additive way. Every value here comes
	// from a resolution this run ALREADY made — artifactRoot from the one root
	// resolution above, the identity from the same accessor `backstop version` reads —
	// so both renderings below read ONE resolved value and cannot report different
	// things about one run.
	identity := effectiveBuildIdentity()
	result.BinaryVersion = identity.Version
	result.ArtifactRoot = artifactRoot.Path
	result.ArtifactRootConfigured = artifactRoot.Configured
	if cohort, cohortErr := schema.ComputeCohort(SchemaFS); cohortErr == nil {
		result.SchemaCohort = cohort.ID
		result.SchemaIdentities = schemaIdentityList(cohort)
	}

	// Format output based on --json flag.
	jsonFlag, jsonErr := cmd.Flags().GetBool("json")
	if jsonErr != nil {
		return fmt.Errorf("reading --json flag: %w", jsonErr)
	}
	if jsonFlag {
		data, err := gate.FormatJSON(result)
		if err != nil {
			return fmt.Errorf("formatting gate JSON output: %w", err)
		}
		cmd.Println(string(data))
	} else {
		noColor := gate.NoColorFromEnv()
		output := gate.FormatHuman(result, noColor)
		cmd.Print(output)
		// Informational: report terminal (retired) artifacts excluded from
		// enforcement (ISSUE-031 CLM-017). Purely informational — it is NOT a
		// warning or violation and does not affect the gate verdict; retirement
		// is deliberate. Silent when zero (no noise).
		// Reads the SAME already-resolved root threaded to buildGateSteps above — no
		// second resolution. terminalExclusionNotice swallows a read error and returns
		// "", so under a .backstop/-rooted project the "N retired artifacts excluded"
		// line would otherwise vanish silently. Purely informational, never a verdict.
		if notice := terminalExclusionNotice(artifactRoot.Dir(artifact.KindSpec)); notice != "" {
			cmd.Println(notice)
		}
	}

	if exitCode != 0 {
		return &ExitCodeError{
			Code:    exitCode,
			Message: fmt.Sprintf("gate: exit code %d", exitCode),
			// Explained ONLY for the violations verdict (SPEC-055 REQ-011 / CLM-077).
			// This return is reachable only after the formatter above wrote the full
			// result — human report or JSON document — so the human line would duplicate
			// it, and under --json it would corrupt the document a consumer parses.
			//
			// The condition is the CODE, not the command: every exit-2 construction in
			// runGate above returns before anything is printed, so suppressing on
			// "this is the gate" would make a gate misconfiguration silent. Should Run
			// ever return a config-class code here, it stays loud by the same rule
			// (CLM-078 is the falsifier).
			Explained: exitCode == ExitViolations,
		}
	}
	return nil
}

// terminalExclusionNotice returns an informational one-line summary of the
// terminal (retired) specs in specDir that the gate excluded from enforcement
// (ISSUE-031 CLM-017). It returns "" when no terminal specs are present (or the
// dir is unreadable) so the gate emits no noise in the common case. The line is
// informational only — it never affects the gate verdict.
func terminalExclusionNotice(specDir string) string {
	count, err := gate.CountTerminalSpecs(specDir)
	if err != nil || count == 0 {
		return ""
	}
	noun := "retired artifact"
	if count != 1 {
		noun += "s"
	}
	return fmt.Sprintf("ℹ %d %s excluded from enforcement (terminal status)", count, noun)
}

func resolveBaselineCache(path string, ttl time.Duration) (*gate.BaselineArtifact, string, time.Time) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		refreshed, refreshErr := refreshBaselineFromRemote(path)
		if refreshErr == nil {
			return refreshed, "baseline cache fetched from remote main baseline artifact", time.Now()
		}
		return nil, fmt.Sprintf("baseline unavailable: no cached baseline found at .backstop/baseline.json and remote baseline fetch failed (%v); run `backstop baseline pull`", refreshErr), time.Time{}
	}
	baseline, loadErr := gate.LoadBaseline(path)
	if loadErr != nil {
		refreshed, refreshErr := refreshBaselineFromRemote(path)
		if refreshErr == nil {
			return refreshed, "baseline cache was unreadable and refreshed from remote main baseline artifact", time.Now()
		}
		return nil, fmt.Sprintf("baseline unavailable: cached baseline at .backstop/baseline.json is unreadable (%v) and remote baseline fetch failed (%v); run `backstop baseline pull`", loadErr, refreshErr), info.ModTime()
	}
	if time.Since(info.ModTime()) <= ttl {
		return baseline, "", info.ModTime()
	}
	refreshed, refreshErr := refreshBaselineFromRemote(path)
	if refreshErr == nil {
		return refreshed, "baseline cache refreshed from remote main baseline artifact", time.Now()
	}
	return baseline, fmt.Sprintf("baseline refresh failed; using stale cached baseline from .backstop/baseline.json (%v)", refreshErr), info.ModTime()
}

func refreshBaselineFromRemote(path string) (*gate.BaselineArtifact, error) {
	if err := runBaselinePull(nil, nil); err != nil {
		return nil, fmt.Errorf("pulling remote baseline: %w", err)
	}
	artifact, err := gate.LoadBaseline(path)
	if err != nil {
		return nil, fmt.Errorf("refreshed baseline cache is unreadable: %w", err)
	}
	return artifact, nil
}

func ruleSetChangeSeedingContext(projectRoot string, scope *gate.GateScope) (bool, []string) {
	if scope == nil || scope.Mode != gate.GateScopeModeAll {
		return false, nil
	}
	changed, err := changedFilesAgainstOriginMain(projectRoot)
	if err != nil {
		return false, nil
	}
	for _, file := range changed {
		if file == "backstop.yml" || file == "backstop.lock" || strings.HasPrefix(file, ".backstop/packs/") || strings.HasPrefix(file, ".backstop/rules/") {
			return true, changed
		}
	}
	return false, changed
}

func changedFilesAgainstOriginMain(projectRoot string) ([]string, error) {
	baseCmd := exec.Command("git", "merge-base", "HEAD", "origin/main")
	baseCmd.Dir = projectRoot
	baseOut, err := baseCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("resolving merge-base against origin/main: %w", err)
	}
	base := strings.TrimSpace(string(baseOut))
	if base == "" {
		return nil, fmt.Errorf("empty merge-base")
	}
	diffCmd := exec.Command("git", "diff", "--name-only", base)
	diffCmd.Dir = projectRoot
	out, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing changed files against merge-base: %w", err)
	}
	files := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files, nil
}

// buildGateSteps constructs the nine ordered step functions with concrete
// implementations wired to real pkg/gate logic.
// Steps 1-2: delegate to real artifact validation and code check.
// Steps 3-6: mechanical verification using grep/AST parsing.
// Steps 7-9: deferred (baseline, waivers, ledger).
// gateConfig loads the project's backstop.yml for the gate wiring, returning a
// zero-value-safe config. An unreadable config yields an empty config (no language
// seed — SPEC-046 retired the single-language field), so the traceability classifier
// still derives a concrete CapabilityState rather than panicking.
func gateConfig(projectRoot string) *config.Config {
	cfg, err := config.LoadConfigFromPath(filepath.Join(projectRoot, "backstop.yml"))
	if err != nil || cfg == nil {
		return &config.Config{}
	}
	return cfg
}

// bunAcceptanceEnabled reports whether the external Bun-fork acceptance (SPEC-047
// REQ-005) may run: true ONLY when the acceptance env var is set AND the bun
// toolchain is on PATH. The guarded acceptance test calls t.Skip() when this is
// false, so backstop-core's Go CI never invokes the real bun/oxlint/tsc/prettier
// toolchain (CLM-029) — the executed proof runs in the fork's own environment. This
// is the ONLY new production symbol this spec adds to gate.go; the live dispatch of
// the bun pack is the UNIFORM toolchain-pack dispatch SPEC-046 already provides, so
// there is NO per-pack branch here — the guard only gates whether the fork
// acceptance test executes.
func bunAcceptanceEnabled() bool {
	if os.Getenv("BACKSTOP_BUN_ACCEPTANCE") == "" {
		return false
	}
	// The acceptance requires the real bun toolchain; its presence on PATH is the
	// second condition, so an env var set in a bun-free environment still skips.
	_, err := exec.LookPath("bun")
	return err == nil
}

// gatePolicyFromConfig converts the declared enforcement.policy table into the gate's
// per-dimension policy map. Returns nil when none is declared, leaving every dimension
// at the default (block, all-code).
func gatePolicyFromConfig(cfg *config.Config) map[string]gate.DimensionPolicy {
	if cfg == nil || len(cfg.Enforcement.Policy) == 0 {
		return nil
	}
	policy := make(map[string]gate.DimensionPolicy, len(cfg.Enforcement.Policy))
	for dim, p := range cfg.Enforcement.Policy {
		policy[dim] = dimensionPolicyFromConfig(p)
	}
	return policy
}

// dimensionPolicyFromConfig maps one config.DimensionPolicy row (including its
// OPTIONAL per-PACK/per-rule-SOURCE overrides) into the gate's gate.DimensionPolicy
// (SPEC-047 REQ-007). The per-source scope carries through verbatim so a scoped entry
// (e.g. backstop/self → block + all-code) reaches gate.ApplyPolicy's per-source
// filtering. An entry with no sources maps to the dimension-only shape (unchanged).
func dimensionPolicyFromConfig(p config.DimensionPolicy) gate.DimensionPolicy {
	out := gate.DimensionPolicy{Level: p.Level, AppliesTo: p.AppliesTo}
	if len(p.Sources) > 0 {
		out.Sources = make(map[string]gate.DimensionPolicy, len(p.Sources))
		for src, sp := range p.Sources {
			out.Sources[src] = gate.DimensionPolicy{Level: sp.Level, AppliesTo: sp.AppliesTo}
		}
	}
	return out
}

// deriveCapabilityState computes a dimension's CapabilityState from installed-pack
// presence (SPEC-037/038/041) and STAMPS the cosmetic stack label (SPEC-046
// REQ-004) carried on CapabilityState.Stack. The stack label is computed once in
// buildGateSteps via declaredToolchainStackLabel(packs) and threaded here through
// wrapTraceabilityStep (the sole caller); pkg/gate renders it instead of the retired
// language-derived stackLabel. The capability classification keys are
// installed-pack-presence and are UNCHANGED by the rehome.
func deriveCapabilityState(packs []*pack.Manifest, dim gate.TraceabilityDimension, stack string) gate.CapabilityState {
	cap := capabilityStateForDimension(packs, dim)
	cap.Stack = stack
	return cap
}

// packDeclaresGateType reports whether any installed pack DECLARES an engine whose
// gate_type equals the traceability dimension (ISSUE-063 REQ-001). Capability presence
// is keyed on the DECLARATION — manifest.Engines[].Binding.GateType, the parsed
// gate_type enum ParseManifest resolves — never on the pack's name or org. A pack from
// ANY org (backstop, acme, a local pack) that declares a `gate_type: contracts` engine
// provides the contracts capability (REQ-003/CLM-005). The dimension string and the
// gate_type spelling are identical, but the map goes through ParseGateType so an
// accidental drift fails closed (absent) rather than silently mis-detecting. A manifest
// is counted at most once even when it declares several engines with the same gate_type.
func packDeclaresGateType(packs []*pack.Manifest, dim gate.TraceabilityDimension) bool {
	return len(packsDeclaringGateType(packs, dim)) > 0
}

// packsDeclaringGateType returns the installed packs that DECLARE an engine with the
// dimension's gate_type (ISSUE-063 REQ-001/REQ-004), deduped to one entry per pack (a
// pack declaring several engines with the same gate_type counts once) and sorted by
// NormalizedName so resolution is deterministic. It is the shared primitive behind both
// capability presence (packDeclaresGateType) and dispatch selection (resolveCapabilityPack).
// The dimension string and the gate_type spelling are identical, but the map goes through
// ParseGateType so an accidental drift yields no matches rather than a mis-selection.
func packsDeclaringGateType(packs []*pack.Manifest, dim gate.TraceabilityDimension) []*pack.Manifest {
	gt, err := engine.ParseGateType(string(dim))
	if err != nil {
		return nil
	}
	var out []*pack.Manifest
	for _, m := range packs {
		if m == nil {
			continue
		}
		for _, spec := range m.Engines {
			if spec.Binding.GateType == gt {
				out = append(out, m)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NormalizedName < out[j].NormalizedName })
	return out
}

// declaresTestVerdictEngine reports whether ANY of the given manifests declares an
// engine binding with `gate_type: test` (ISSUE-118 CLM-006).
//
// It is keyed on the DECLARATION, never on a pack name — the same rule
// packsDeclaringGateType follows for the traceability dimensions. It answers the
// question test_verification needs in order to distinguish "the mandated tests
// passed" from "nothing can tell me whether they passed": without a declared
// test-verdict engine, an empty verdict stream means the capability is absent, not
// that the suite was green, and reporting an unqualified pass there is exactly the
// silent green ISSUE-118 was filed about.
func declaresTestVerdictEngine(packs []*pack.Manifest) bool {
	for _, m := range packs {
		if m == nil {
			continue
		}
		for _, spec := range m.Engines {
			if spec.Binding.GateType == engine.GateTypeTest {
				return true
			}
		}
	}
	return false
}

// resolveCapabilityPack selects THE installed pack that provides a dimension's capability
// by its declared gate_type engine (ISSUE-063 REQ-004), never by name. Zero matches → nil
// (capability-absent no-op, governed by the polarity classifier upstream). Exactly one →
// that pack. More than one → a fail-loud config error naming the ambiguous packs, so a
// multi-provider install can never silently pick one. The SAME by-declaration selection
// drives both capability presence (REQ-001) and this engine dispatch (REQ-004).
func resolveCapabilityPack(installed []*pack.Manifest, dim gate.TraceabilityDimension) (*pack.Manifest, error) {
	matches := packsDeclaringGateType(installed, dim)
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.NormalizedName)
		}
		return nil, fmt.Errorf(
			"ambiguous %s capability: %d installed packs declare a %q gate_type engine (%s); exactly one pack may provide a given traceability dimension",
			dim, len(matches), string(dim), strings.Join(names, ", "),
		)
	}
}

// capabilityStateForDimension computes the Present/Working/PackOrCommand capability
// state for a dimension from what the installed packs DECLARE (ISSUE-063 REQ-001): a
// dimension is present iff some installed pack declares an engine with the matching
// gate_type (packDeclaresGateType). This replaces the three name-keyed re-keys
// (SPEC-037/038/041) that bound each capability to a baked pack coordinate
// (`backstop/contracts`, `backstop/substantiveness`) or a naming convention
// (`*-toolchain`) — a pack from ANY org that declares the gate_type now fills the slot
// (REQ-003). The PackOrCommand string is a human DISPLAY label only; it names the
// declared capability, never a distribution coordinate. The cosmetic Stack label is
// stamped by the caller (deriveCapabilityState). Absent+undeclared lands class-2 (warn,
// exit 0) and absent+declared class-3 (block) via the SPEC-036 classifier upstream.
func capabilityStateForDimension(packs []*pack.Manifest, dim gate.TraceabilityDimension) gate.CapabilityState {
	if packDeclaresGateType(packs, dim) {
		return gate.CapabilityState{
			Present:       true,
			Working:       true,
			PackOrCommand: "the installed pack declaring a " + string(dim) + " engine",
		}
	}
	return gate.CapabilityState{
		Present:       false,
		Working:       false,
		PackOrCommand: "a pack declaring a " + string(dim) + " engine (install it: `backstop pack add`)",
	}
}

// coverageToolchainPackInstalled reports whether the coverage capability is present:
// some installed pack DECLARES a `gate_type: coverage` engine (ISSUE-063 REQ-002). It
// no longer keys on a `*-toolchain` name suffix — the coverage producer is identified
// by its declaration, not its pack name, so a coverage pack under any name/org fills
// the slot. A thin delegator to packDeclaresGateType, kept as the named coverage-arm
// entry point.
func coverageToolchainPackInstalled(packs []*pack.Manifest) bool {
	return packDeclaresGateType(packs, gate.DimensionCoverage)
}

// substantivenessPackInstalled reports whether the substantiveness capability is
// present: some installed pack DECLARES a `gate_type: substantiveness` engine
// (ISSUE-063 REQ-002). It no longer matches the `backstop/substantiveness` coordinate;
// capability keys on the declaration, so a substantiveness pack under any name/org
// fills the slot. A thin delegator to packDeclaresGateType.
func substantivenessPackInstalled(packs []*pack.Manifest) bool {
	return packDeclaresGateType(packs, gate.DimensionSubstantiveness)
}

// contractsPackInstalled reports whether the contracts capability is present: some
// installed pack DECLARES a `gate_type: contracts` engine (ISSUE-063 REQ-002). It no
// longer matches the `backstop/contracts` coordinate; capability keys on the
// declaration, so a contracts pack under any name/org fills the slot (REQ-003/CLM-005).
// A thin delegator to packDeclaresGateType.
func contractsPackInstalled(packs []*pack.Manifest) bool {
	return packDeclaresGateType(packs, gate.DimensionContracts)
}

// wrapTraceabilityStep wraps a traceability analyzer step (the delegate) with
// the SPEC-036 polarity classifier so the classifier runs IN FRONT OF the
// analyzer: it derives the CapabilityState (installed-pack presence), classifies
// the dimension, and for class 1/2/3 returns PolarityStepResult WITHOUT reaching
// the analyzer (intercept). Only the none/proceed outcome (declared-and-working or
// undeclared-but-present) falls through to the unchanged delegate (REQ-008). The
// delegate is taken as a parameter so the wiring test can spy on whether the
// analyzer is reached (CLM-028). The `stack` parameter is the declared-toolchain
// stack label (SPEC-046 REQ-004), threaded into deriveCapabilityState so the
// rehomed classifier renders it on CapabilityState.Stack — deriveCapabilityState is
// reached ONLY through this wrapper, so the label is stamped here, never past it.
func wrapTraceabilityStep(packs []*pack.Manifest, cfg *config.Config, dim gate.TraceabilityDimension, stepName string, stack string, delegate gate.StepFunc) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		cap := deriveCapabilityState(packs, dim, stack)
		class := gate.ClassifyDimension(cfg, dim, cap)
		if class != gate.ClassNone {
			// Intercept — do NOT reach the analyzer.
			return gate.PolarityStepResult(stepName, dim, class, cfg, cap)
		}
		// Fall through to the unchanged analyzer.
		return delegate(ctx)
	}
}

// toolchainEnforcementStepName is the stable step name for the no-toolchain-pack
// WARN-ONLY loud state (SPEC-040 REQ-005/REQ-006). It is a function (not a
// package-level var/const) to keep the file free of package-level mutable state.
func toolchainEnforcementStepName() string { return "toolchain_enforcement" }

// noToolchainPackMessage is the stable, recognizable loud message the
// no-toolchain-pack WARN-ONLY state renders on the gate's human report surface
// and reflects in the machine summary (SPEC-040 REQ-006). It must never be
// collapsed into a normal green — "nothing ran" must be impossible to mistake
// for "everything passed."
func noToolchainPackMessage() string { return "enforcement not configured (0 toolchain packs)" }

// declaredToolchainStackLabel derives the cosmetic traceability stack label from the
// SET of DECLARED toolchain packs (SPEC-046 REQ-004 / SQ-1, rehomed by ISSUE-064 REQ-003).
// Membership is by declaresToolchainMechanism — the SAME by-declaration primitive
// countToolchainPacks uses — so the label set and the count set share one membership
// source and can never disagree. Each label VALUE is the pack's DECLARED
// manifest.Language (e.g. a pack declaring language: go -> "go"), NOT the name with a
// "-toolchain" suffix stripped; a mechanism pack that declares no language contributes
// no token. The resulting stack names are joined as a SET (deduped + sorted, NO
// precedence and NO overlap winner) — a polyglot repo's label names EVERY declared stack.
//
// Returns "unspecified" when the declared-language set is empty — the SINGLE
// authoritative empty-fallback signal. SPEC-043's SourceClassifier.HasSourceGlobs() is
// CORROBORATING ONLY and MUST NOT drive this fallback: it can diverge from the declared
// set (a mechanism pack with no `classification` source globs still labels its declared
// language yet HasSourceGlobs() == false). This is a declared-language-set helper, NOT a
// glob classifier — SPEC-043's single gate.SourceClassifier remains the ONLY glob
// classifier (no fork, CLM-019).
func declaredToolchainStackLabel(packs []*pack.Manifest) string {
	seen := map[string]bool{}
	var stacks []string
	for _, m := range packs {
		if !declaresToolchainMechanism(m) {
			continue
		}
		lang := strings.TrimSpace(m.Language)
		if lang == "" || seen[lang] {
			continue
		}
		seen[lang] = true
		stacks = append(stacks, lang)
	}
	if len(stacks) == 0 {
		return "unspecified"
	}
	sort.Strings(stacks)
	return strings.Join(stacks, ", ")
}

// toolchainMechanismGateTypes returns the enforcement-MECHANISM dimensions a toolchain
// pack provides (lint/typecheck/test/coverage). A pack declaring any of these IS a
// toolchain pack, BY DECLARATION (the ISSUE-063 principle applied to the SPEC-040
// enforcement-configured signal) — independent of its name or org, so a third-party
// toolchain pack not named `*-toolchain` still counts. `findings` is deliberately
// excluded: a standalone rules pack (secrets, standards) declares findings but is not
// the enforcement mechanism.
func toolchainMechanismGateTypes() []gate.TraceabilityDimension {
	return []gate.TraceabilityDimension{"lint", "build", "test", "coverage"}
}

// declaresToolchainMechanism reports whether a manifest declares any enforcement-
// mechanism engine, via the shared packsDeclaringGateType by-declaration primitive.
func declaresToolchainMechanism(m *pack.Manifest) bool {
	one := []*pack.Manifest{m}
	for _, gt := range toolchainMechanismGateTypes() {
		if packDeclaresGateType(one, gt) {
			return true
		}
	}
	return false
}

// countToolchainPacks counts the DECLARED packs that provide the enforcement
// mechanism — every declared pack that DECLARES a lint/build/test/coverage engine
// (declaresToolchainMechanism), NOT by the `-toolchain` name convention. A toolchain
// is an ordinary declared pack (SPEC-046 REQ-002); this keys on the declared `packs:`
// set alone. It is the signal the no-toolchain-pack WARN-ONLY loud state keys on
// (SPEC-040 REQ-005/REQ-006) — "0 mechanism packs" means nothing actually ran, which
// must never read as a real green.
func countToolchainPacks(declared []*pack.Manifest) int {
	n := 0
	for _, m := range declared {
		if declaresToolchainMechanism(m) {
			n++
		}
	}
	return n
}

// toolchainEnforcementStatus returns the no-toolchain-pack WARN-ONLY loud
// StepResult and true when ZERO <lang>-toolchain packs are DECLARED — the
// anti-vacuous-green guardrail (SPEC-040 REQ-005/REQ-006, Sharp Edge 3). A
// toolchain is an ordinary declared pack (SPEC-046 REQ-002), so the state keys on
// the declared `packs:` set alone — no language-derived bridge input. The step is
// a NON-FAILING "warning" carrying the stable loud message in Reason, counted in
// GateResult.StepsWarned and rendered on the human report surface; gate.Pass stays
// true and exit code stays 0. When at least one toolchain pack IS declared it
// returns (_, false): the toolchain passes run through dispatchPackEngines and
// produce their own normal pass/fail, so no warn state is emitted. Reusing the
// SPEC-036 "warning"/StepsWarned mechanism — no new status vocabulary is invented.
func toolchainEnforcementStatus(declared []*pack.Manifest) (gate.StepResult, bool) {
	if countToolchainPacks(declared) > 0 {
		return gate.StepResult{}, false
	}
	return gate.StepResult{
		StepName:   toolchainEnforcementStepName(),
		Status:     "warning",
		Reason:     noToolchainPackMessage(),
		Violations: []gate.Violation{},
	}, true
}

// buildGateSteps assembles the ordered gate step list.
//
// projectRoot and root are DELIBERATELY SEPARATE and must stay so. projectRoot is the
// CODE directory these steps walk for test files and coverage; root is the resolved
// ARTIFACT root the spec-reading dimensions read. Under a `.backstop/` layout the two
// are genuinely different directories, and collapsing them would point test discovery
// at `.backstop/` and find no source at all — a different vacuous green than the one
// this change removes.
//
// The scope tail stays LAST and variadic, which is what lets the call sites that omit
// it keep omitting it; the new parameter therefore goes BEFORE it, since Go cannot
// express a parameter after a variadic.
func buildGateSteps(projectRoot string, root artifact.Root, scope ...*gate.GateScope) []gate.StepFunc {
	var activeScope *gate.GateScope
	if len(scope) > 0 {
		activeScope = scope[0]
	}
	specDir := root.Dir(artifact.KindSpec)
	packs, packErr := loadInstalledPacks(projectRoot)
	if packErr != nil {
		return []gate.StepFunc{
			func(context.Context) gate.StepResult {
				return gate.StepResult{
					StepName:   "pack_loading",
					Status:     "fail",
					ConfigErr:  true,
					Violations: []gate.Violation{{Rule: "pack_loading", Message: packErr.Error(), Severity: "error"}},
				}
			},
		}
	}

	// THE ARTIFACT-CORPUS EXCLUSION SET, DERIVED ONCE (ISSUE-122 CLM-005). It is the
	// tool-agnostic base core carries unioned with the dependency directory names the
	// INSTALLED PACKS declare via classification.dependency_dirs — the same `packs`
	// manifest set mergeSourceClassifier reads below, so the gate cannot disagree with
	// itself about which trees are corpus. Every consumer downstream takes THIS
	// variable; none re-loads the packs and none writes a literal.
	nonCorpus := artifact.NewNonCorpusDirs(mergeDependencyDirs(packs))

	// Step 1: Artifact validation — delegates to ValidateArtifacts.
	artifactValidator := &realArtifactValidator{projectRoot: projectRoot, root: root, nonCorpus: nonCorpus}

	// Step 2 ("Code check") is GONE as a gate step (SPEC-040 REQ-001): lint/build/
	// test now run only through dispatchPackEngines. The baked shared `go test ./...`
	// runner is ERADICATED (SPEC-041 REQ-002): coverage no longer reuses an in-binary
	// whole-module exec. Coverage's per-FILE signal is now the declared toolchain
	// coverage pass (SPEC-042's dispatchPackCoverage producer over the DECLARED
	// <lang>-toolchain packs), consumed per-FILE by the re-implemented coverage step
	// (SPEC-041 REQ-001).

	// LIVE DISCOVERY WIRING (SPEC-045 REQ-006): the merged SourceClassifier (test +
	// source globs) and merged TestNameMatcher (test-name patterns) are built ONCE from
	// the UNION of the declared-pack manifests and threaded into BOTH the
	// test-verification and substantiveness steps, closing the integration gap where a
	// correct discovery unit never reaches the live gate. mergeSourceClassifier /
	// mergeTestNameMatcher take the wholesale declared `packs` set (no toolchain-only
	// pre-filter) — a toolchain is just an ordinary declared pack (SPEC-046), so there
	// is no language-derived bridge set to thread. An invalid pack-declared test-name
	// regex is a LOUD config-error step (never a silently-empty matcher that would make
	// discovery find nothing and mass-fail every mandated test).
	classifier := mergeSourceClassifier(packs)
	matcher, matcherErr := mergeTestNameMatcher(packs)
	if matcherErr != nil {
		return []gate.StepFunc{
			func(context.Context) gate.StepResult {
				return gate.StepResult{
					StepName:   "pack_loading",
					Status:     "fail",
					ConfigErr:  true,
					Violations: []gate.Violation{{Rule: "pack_loading", Message: "compiling pack-declared test_name_patterns: " + matcherErr.Error(), Severity: "error"}},
				}
			},
		}
	}

	// Steps 3-4: Test verification and substantiveness need spec dir and code dir.
	// We use the project root as the code directory for walking test files. Both
	// consume the merged classifier + matcher (the de-Go'd pack-declared discovery).
	//
	// THE TEST-VERDICT COLLECTOR (ISSUE-118). pack_engines computes a failing
	// mandated test's finding correctly and then throws it away: the finding's
	// reported position is a bare basename that no scope's canonicalized file set
	// can match, so the diff-scope filter drops it silently. This collector carries
	// the UNFILTERED dispatch stream — captured BEFORE that filter — across to
	// test_verification, which joins it to the mandated tests BY NAME.
	//
	// ORDERING IS WHAT MAKES A SUPPLIER SAFE. `packed` (below) assembles lock,
	// steps[0], pack_engines, then steps[1:] — which begins at test_verification.
	// So the collector is ALWAYS populated before it is read. If that assembly order
	// is ever changed, THIS is what breaks, and the verdict would silently go empty.
	//
	// The zero value is the honest default for the paths that never reach dispatch:
	// no verdict engine declared, so test_verification emits its capability-absent
	// advisory rather than an unqualified pass. That covers the len(packs) == 0 early
	// return below, where no pack_engines step is assembled at all.
	var collectedVerdicts []gate.Violation
	verdictEngineDeclared := false
	testVerifyStep := gate.StepTestVerificationVerdictFunc(specDir, projectRoot, activeScope, classifier, matcher,
		func() ([]gate.Violation, bool) { return collectedVerdicts, verdictEngineDeclared })

	// Step 4: Test substantiveness needs the resolved mandated tests with file paths.
	// We extract mandated tests and resolve their file paths, then pass to substantiveness.
	testSubstantivenessStep := buildTestSubstantivenessStep(specDir, projectRoot, projectRoot, activeScope, classifier, matcher)

	// Step 5: Coverage threshold consumes the canonical per-FILE
	// []check.CoverageRecord PRODUCED by SPEC-042's dispatchPackCoverage over the
	// DECLARED toolchain packs — NOT a binary-resident `go test` runner (re-baking one
	// would re-violate REQ-002). The records are sourced lazily at step-run time so the
	// producer is exercised inside the gate (CLM-003). The merged classifier (above) is
	// also threaded into the coverage step (SPEC-043 REQ-005).
	coverageStep := buildCoverageStep(specDir, projectRoot, activeScope, classifier, coverageRecordsProducer(packs, projectRoot))

	// Step 6: Contract signature needs contract entries extracted from specs.
	contractStep := buildContractStep(specDir, projectRoot, activeScope)

	// ISSUE-042: the native status/reality drift dimension (CLM-007/008/009). It runs a
	// FULL-SWEEP existence check — resolving EVERY artifact under projectRoot and checking
	// each mandated test name against the whole-repo found-test set — so a stale-status
	// artifact whose file is out of the diff is still caught. It threads NO pass/fail and
	// re-runs NO suite (a present-but-failing mandated test is pack_engines' job, CLM-005).
	// The wiring emits TWO surfaces: the policied BLOCK step (StepArtifactStatusDrift) and
	// the non-policied WARN advisory (StepArtifactStatusDriftAdvisory), so the WARN
	// direction is structurally non-blocking (no policy can upgrade it).
	driftBlockStep, driftAdvisoryStep := buildStatusDriftSteps(projectRoot, root, classifier, matcher, mergePackClaimIndex(packs))
	traceBlockStep, traceAdvisoryStep := buildRequirementTraceabilitySteps(projectRoot, root, nonCorpus)

	// SPEC-036: wrap the three traceability analyzer steps with the polarity
	// classifier so it runs IN FRONT OF each analyzer. The classifier derives the
	// CapabilityState from installed-pack presence (no pack, no engine), classifies
	// the dimension, and intercepts for class 1/2/3 (analyzer not reached); only
	// declared-and-working / undeclared-but-present fall through to the UNCHANGED
	// analyzer (REQ-008). The analyzer files themselves are untouched — the wrapper
	// intercepts at the wiring boundary. SPEC-046 REQ-004: the cosmetic stack label is
	// computed ONCE from the declared toolchain-pack-NAME set and threaded into all
	// three wrappers (and thence deriveCapabilityState) so the rehomed classifier
	// renders it on CapabilityState.Stack — no language read.
	traceabilityCfg := gateConfig(projectRoot)
	stackLabel := declaredToolchainStackLabel(packs)
	testSubstantivenessStep = wrapTraceabilityStep(packs, traceabilityCfg, gate.DimensionSubstantiveness, gate.StepTestSubstantiveness, stackLabel, testSubstantivenessStep)
	coverageStep = wrapTraceabilityStep(packs, traceabilityCfg, gate.DimensionCoverage, gate.StepCoverageThreshold, stackLabel, coverageStep)
	contractStep = wrapTraceabilityStep(packs, traceabilityCfg, gate.DimensionContracts, gate.StepContractSignature, stackLabel, contractStep)

	// SPEC-040 KEYSTONE CUTOVER (REQ-001/CLM-001/CLM-008): the bespoke code-check /
	// gate.StepCodeCheckScopedFunc Step-2 entry is GONE from the step list. Lint,
	// build, and test enforcement now runs ONLY via dispatchPackEngines over the
	// DECLARED <lang>-toolchain packs (the pack_engines step below) — no dual-run, no
	// parallel dispatcher, no pkg/check->pkg/pack/engine import. The
	// SPEC-040 transitional shared go-test runner is ERADICATED by SPEC-041 REQ-002:
	// coverage now consumes the per-FILE []check.CoverageRecord produced by the
	// declared toolchain coverage pass (coverageRecordsProducer), not an in-binary
	// `go test`.
	steps := []gate.StepFunc{
		gate.StepArtifactValidationScopedFunc(artifactValidator, activeScope),
		testVerifyStep,
		testSubstantivenessStep,
		coverageStep,
		contractStep,
		driftBlockStep,
		driftAdvisoryStep,
		traceBlockStep,
		traceAdvisoryStep,
		// SPEC-049 REQ-017: waiver resolution MUST run BEFORE baseline comparison so
		// the accumulated violation set is ALREADY waiver-subtracted when
		// baseline_comparison captures its NewViolations — otherwise an active-waived
		// finding still counts against the ISSUE-050 ratchet (REQ-013). Ordering here
		// (waiver ahead of baseline) is what makes an active waiver satisfy the ratchet.
		gate.StepWaiverResolutionScopedFunc(activeScope),
		gate.StepBaselineComparisonScopedFunc(activeScope),
		gate.StepLedgerIntegrityScopedFunc(activeScope),
	}

	// The no-toolchain-pack WARN-ONLY loud state (SPEC-040 REQ-005/REQ-006, Sharp
	// Edge 3): when ZERO <lang>-toolchain packs are DECLARED, append a NON-FAILING
	// "warning" step carrying the stable loud "enforcement not configured (0 toolchain
	// packs)" message. gate.Pass stays true and exit 0, but "nothing ran" is impossible
	// to mistake for "everything passed." A toolchain is an ordinary declared pack
	// (SPEC-046), so this keys on the declared `packs:` set alone. Wired here so it
	// flows through BOTH the no-declared-packs early return and the pack-dispatch path
	// below.
	if warnStep, emitted := toolchainEnforcementStatus(packs); emitted {
		warn := warnStep
		steps = append(steps, func(context.Context) gate.StepResult { return warn })
	}

	if len(packs) == 0 {
		return steps
	}

	packNames := make([]string, 0, len(packs))
	for _, manifest := range packs {
		packNames = append(packNames, manifest.NormalizedName)
	}

	lockStep := func(context.Context) gate.StepResult {
		err := verifyPackLock(projectRoot, packNames)
		if err != nil {
			return gate.StepResult{
				StepName:   "pack_lock_verification",
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "pack_lock_verification", Message: err.Error(), Severity: "error"}},
			}
		}
		return gate.StepResult{
			StepName:   "pack_lock_verification",
			Status:     "pass",
			Violations: []gate.Violation{},
		}
	}

	packValidatorStep := func(context.Context) gate.StepResult {
		runner := &check.ExecCommandRunner{Dir: projectRoot}
		// A <lang>-toolchain pack dispatches its lint/build/test engine bindings
		// through the SAME substrate as every other declared pack (SPEC-046): a
		// toolchain is an ordinary declared pack, so it flows through this uniform
		// pack_engines dispatch — there is no language-derived bridge set.
		// The generic pack_engines findings dispatch runs only the generic stages
		// (lint/build/test/findings). Rules bound to a dedicated-step gate_type
		// (substantiveness/contracts/coverage) are dispatched by their own gate step;
		// running them here too would scan context-free and emit garbage findings.
		dispatchPacks := excludeDedicatedStepRules(packs)
		// PRESENCE CHECK OVER THE WHOLE INSTALLED-PACK SET (ISSUE-112) — note the
		// argument is `packs`, NOT `dispatchPacks`, and that is the entire point.
		//
		// Before dispatch, verify every declared engine tool resolves on PATH:
		// assume-present Layer-0 tools (go/golangci-lint) and pinned ones (semgrep,
		// ast-grep) alike, since backstop installs neither. A missing one is a
		// *check.ConfigError surfaced as a config-error step (exit 2) naming the tool.
		//
		// The exclusion above answers "which step dispatches this rule"; it must not
		// double as "whose tools have to exist". Tool presence is a property of the
		// TOOL, not of the dispatching step. Passing the post-exclusion set here left
		// engines routed to a dedicated step — substantiveness, contracts, coverage —
		// never probed at all, so a project missing their tool got a clean dimension
		// over an engine that never ran.
		//
		// OWNED CONSEQUENCE: for those dedicated-step engines the allowlist TRUST
		// refusal now surfaces from this step instead of from dispatch. Same verdict,
		// same message, reported earlier — the un-allowlisted tool was always refused.
		if provErr := provisionEngines(packs); provErr != nil {
			return gate.StepResult{
				StepName:   "pack_engines",
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "pack_engines", Message: provErr.Error(), Severity: "error"}},
			}
		}
		// Thread the gate's diff scope (activeScope) so rule-fed findings engines
		// scan only the changed files, not the whole repository (ISSUE-010). A nil
		// activeScope or GateScopeModeAll restores the whole-repo sweep; the
		// project-wide toolchain passes stay project-wide regardless.
		violations, err := resolveDispatchPackEngines()(dispatchPacks, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, activeScope, runner)
		if err != nil {
			return gate.StepResult{
				StepName:   "pack_engines",
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "pack_engines", Message: err.Error(), Severity: "error"}},
			}
		}
		// PUBLISH THE UNFILTERED STREAM TO THE VERDICT COLLECTOR — BEFORE THE FILTER
		// BELOW, WHICH IS THE ENTIRE POINT (ISSUE-118). A failing mandated test's
		// finding reports a bare basename that resolves nowhere, so the diff-scope
		// filter drops it and test_verification never learns the test failed. Routing
		// is by the DECLARED gate_type carried on each violation — never a pack, rule
		// or message-name sniff.
		//
		// This ADDS a reader; it changes NOTHING about what this step itself reports.
		// The filtered set below is untouched, deliberately: whether pack_engines
		// should also keep an out-of-scope test failure is a separate question with
		// its own issue, and quietly answering it here would make both unreviewable.
		collectedVerdicts = gate.RouteTestVerdictFindings(violations)
		verdictEngineDeclared = declaresTestVerdictEngine(dispatchPacks)

		// ISSUE-070: apply the diff-scope filter the delegate/baseline paths already
		// use. Project-wide engines (golangci) scan ./... and legitimately produce
		// violations across the whole repo; unchanged-file NON-exempt violations must be
		// filtered out here or they leak past diff-scope. ProjectWide (exempt, e.g.
		// go-build) violations are structurally retained by the filter.
		violations = activeScope.FilterViolations(violations)
		return gate.StepResult{
			StepName:   "pack_engines",
			Status:     gate.StepVerdict(violations),
			Violations: violations,
		}
	}

	// Order: lock, artifact (steps[0]), pack_engines dispatch, then the remaining
	// steps (test_verification onward — the code_check Step-2 entry is gone, so
	// steps[1:] now begins at test_verification).
	packed := make([]gate.StepFunc, 0, len(steps)+2)
	packed = append(packed, lockStep)
	packed = append(packed, steps[0], packValidatorStep)
	packed = append(packed, steps[1:]...)
	return packed
}

// buildStatusDriftSteps wires the ISSUE-042 native status/reality drift dimension into
// the gate (CLM-007/008/009). It returns two StepFuncs sharing ONE lazy resolution:
//
//   - block (StepArtifactStatusDrift): the error-severity broken-promise surface
//     (success-terminal artifact + absent mandated test). This is the POLICIED dimension
//     (backstop.yml level: block, applies-to: new-code) — pre-existing findings
//     grandfather against the baseline, net-new ones block.
//   - advisory (StepArtifactStatusDriftAdvisory): the warning-severity delivered-but-open
//     surface. It carries NO policy entry, so its "warning" status is structurally
//     non-blocking (CLM-006).
//
// EXISTENCE is the ONLY signal and it runs FULL-SWEEP: ResolveMandatedTestPaths resolves
// each mandated test name against the whole-repo found-test set (collectTestFuncNames
// walks all of projectRoot, NOT activeScope), so a stale-status artifact whose file is out
// of the diff is still caught (CLM-007). No pass/fail is threaded and no suite is re-run
// (CLM-005/008) — a present-but-failing mandated test is caught by the pack_engines step.
func buildStatusDriftSteps(projectRoot string, root artifact.Root, classifier gate.SourceClassifier, matcher gate.TestNameMatcher, packClaims gate.PackClaimIndex) (gate.StepFunc, gate.StepFunc) {
	var (
		once           sync.Once
		blockResult    gate.StepResult
		advisoryResult gate.StepResult
	)
	compute := func() {
		res, err := gate.ResolveArtifactStatus(root.Path)
		if err != nil {
			// A resolve failure is a config error under the BLOCK name (it halts the gate);
			// the advisory surface stays a clean pass so it never invents a false warning.
			blockResult = gate.StepResult{
				StepName:   gate.StepArtifactStatusDrift,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: gate.StepArtifactStatusDrift, Message: "resolving artifact status: " + err.Error(), Severity: "error"}},
			}
			advisoryResult = gate.StepResult{StepName: gate.StepArtifactStatusDriftAdvisory, Status: "pass", Violations: []gate.Violation{}}
		} else {
			blockResult, advisoryResult = computeDriftSurfaces(projectRoot, res, classifier, matcher, packClaims)
		}
	}
	block := func(context.Context) gate.StepResult {
		once.Do(compute)
		return blockResult
	}
	advisory := func(context.Context) gate.StepResult {
		once.Do(compute)
		return advisoryResult
	}
	return block, advisory
}

// computeDriftSurfaces runs the FULL-SWEEP existence resolution over the resolved artifact
// records and returns the split block/advisory surfaces. EXISTENCE is the only signal:
// ResolveMandatedTestPaths resolves each mandated test name against the whole-repo
// found-test set (NOT activeScope), so an out-of-diff stale artifact is still caught. No
// pass/fail is threaded (CLM-005/008).
//
// EXISTENCE resolves against BOTH mandated-test vocabularies (ISSUE-098). A mandated name
// is present when a source test FUNCTION carries it OR when an installed pack DECLARES a
// claim by that name — a pack's claims are that pack's tests, and no test-file glob or
// test-name regex can ever match a claim id in a pack.yml, so the source sweep alone
// reports pack-side evidence as a false broken promise.
//
// The union happens HERE and deliberately NOT inside ResolveMandatedTestPaths: a
// pack-claim hit must never fill MandatedTest.FilePath, because the test substantiveness
// Q2 noTarget join below skips a mandated test only when FilePath is empty. Giving a
// pack-resolved test the path of its pack.yml would make that join ask whether a manifest
// references the target package and emit a false "does not call package" violation
// against it (CLM-004).
func computeDriftSurfaces(projectRoot string, res *gate.ArtifactStatusResolution, classifier gate.SourceClassifier, matcher gate.TestNameMatcher, packClaims gate.PackClaimIndex) (gate.StepResult, gate.StepResult) {
	var all []gate.MandatedTest
	for _, rec := range res.Records {
		all = append(all, rec.MandatedTests...)
	}
	all = gate.ResolveMandatedTestPaths(all, projectRoot, classifier, matcher)
	present := gate.ResolvePresentTestNames(all, packClaims)
	combined := gate.ClassifyStatusDrift(res.Records, present)
	// Normalize each violation's File to the ONE canonical repo-relative form so its
	// baseline identity is scope-stable (ISSUE-046), matching test_verification.
	for i := range combined.Violations {
		combined.Violations[i].File = gate.NormalizePath(projectRoot, combined.Violations[i].File)
	}
	return gate.SplitDriftResult(combined)
}

// buildRequirementTraceabilitySteps takes the artifact-corpus exclusion set because it
// is the outermost hop on the path down to collectTraceRefs's DiscoverArtifacts call —
// buildGateSteps is where the installed-pack manifests are actually in hand, so the set
// travels from there rather than being re-derived (or defaulted) further in.
func buildRequirementTraceabilitySteps(projectRoot string, root artifact.Root, nonCorpus artifact.NonCorpusDirs) (gate.StepFunc, gate.StepFunc) {
	var (
		once           sync.Once
		blockResult    gate.StepResult
		advisoryResult gate.StepResult
	)
	compute := func() {
		blockResult, advisoryResult = computeRequirementTraceabilitySurfaces(projectRoot, root, nonCorpus)
	}
	block := func(context.Context) gate.StepResult {
		once.Do(compute)
		return blockResult
	}
	advisory := func(context.Context) gate.StepResult {
		once.Do(compute)
		return advisoryResult
	}
	return block, advisory
}

func computeRequirementTraceabilitySurfaces(projectRoot string, root artifact.Root, nonCorpus artifact.NonCorpusDirs) (gate.StepResult, gate.StepResult) {
	res, err := gate.ResolveArtifactStatus(root.Path)
	if err != nil {
		return gate.StepResult{
			StepName:   gate.StepRequirementTraceability,
			Status:     "fail",
			ConfigErr:  true,
			Violations: []gate.Violation{{Rule: gate.StepRequirementTraceability, Message: "resolving artifact status: " + err.Error(), Severity: "error"}},
		}, gate.StepResult{StepName: gate.StepRequirementTraceabilityAdvisory, Status: "pass", Violations: []gate.Violation{}}
	}
	refs, err := collectTraceRefs(root, nonCorpus)
	if err != nil {
		return gate.StepResult{
			StepName:   gate.StepRequirementTraceability,
			Status:     "fail",
			ConfigErr:  true,
			Violations: []gate.Violation{{Rule: gate.StepRequirementTraceability, Message: "collecting supports refs: " + err.Error(), Severity: "error"}},
		}, gate.StepResult{StepName: gate.StepRequirementTraceabilityAdvisory, Status: "pass", Violations: []gate.Violation{}}
	}
	for i := range res.Records {
		res.Records[i].Path = gate.NormalizePath(projectRoot, res.Records[i].Path)
	}
	for i := range refs {
		refs[i].CitingPath = gate.NormalizePath(projectRoot, refs[i].CitingPath)
	}
	combined := gate.ClassifyRequirementTraceability(res.Records, refs)
	for i := range combined.Violations {
		combined.Violations[i].File = gate.NormalizePath("", combined.Violations[i].File)
	}
	return gate.SplitTraceabilityResult(combined)
}

func collectTraceRefs(root artifact.Root, nonCorpus artifact.NonCorpusDirs) ([]gate.TraceRef, error) {
	discovered, err := DiscoverArtifacts(root, []string{"spec", "issue"}, nonCorpus)
	if err != nil {
		return nil, fmt.Errorf("discovering spec/issue artifacts: %w", err)
	}
	var parsed []*artifact.ParsedArtifact
	pathByBase := map[string]string{}
	for _, da := range discovered {
		art, parseErr := artifact.ParseFile(da.Path)
		if parseErr != nil {
			continue
		}
		parsed = append(parsed, art)
		pathByBase[art.Filename] = da.Path
	}
	refs := validate.CollectSupportRefs(parsed)
	out := make([]gate.TraceRef, 0, len(refs))
	for _, ref := range refs {
		citingPath := ref.File
		if pathByBase[ref.File] != "" {
			citingPath = pathByBase[ref.File]
		}
		out = append(out, gate.TraceRef{
			BundleName: ref.BundleName,
			ReqID:      ref.ReqID,
			PinVersion: ref.Version,
			Pinned:     ref.Version != "",
			CitingPath: citingPath,
		})
	}
	return out, nil
}

// resolveSubstantivenessPacksFn is a test seam: nil in production (the resolver below
// returns the INSTALLED substantiveness pack manifest set), overridden by the wiring
// tests that inject a manifest set so the dispatch-seam spy can observe routing without
// a real on-disk install. Declared WITHOUT an initializer so it holds no package-level
// mutable default — production resolves lazily via resolveSubstantivenessPacks.
var resolveSubstantivenessPacksFn func(projectRoot string) ([]*pack.Manifest, error)

// resolveSubstantivenessPacks returns the substantiveness pack manifest set the
// substantiveness step dispatches. In production it filters the INSTALLED packs
// (loadInstalledPacks) to the substantiveness pack — the capability is
// installed-pack-resolvable, NOT built into the binary and NOT resolved from testdata
// (REQ-009 / CLM-030 / CLM-035). A test seam may override it. An empty result means the
// substantiveness pack is not installed; the caller treats that as a capability-absent
// no-op (no findings, no violation), which the capability classifier governs upstream.
func resolveSubstantivenessPacks(projectRoot string) ([]*pack.Manifest, error) {
	if resolveSubstantivenessPacksFn != nil {
		return resolveSubstantivenessPacksFn(projectRoot)
	}
	installed, err := loadInstalledPacks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving substantiveness packs: loading installed packs: %w", err)
	}
	// Select the substantiveness pack by its DECLARED gate_type: substantiveness engine
	// (ISSUE-063 REQ-004), not by NormalizedName — org-agnostic, fail-loud on ambiguity.
	m, err := resolveCapabilityPack(installed, gate.DimensionSubstantiveness)
	if err != nil {
		return nil, fmt.Errorf("resolving substantiveness pack: %w", err)
	}
	if m == nil {
		return []*pack.Manifest{}, nil
	}
	return []*pack.Manifest{m}, nil
}

// buildTestSubstantivenessStep re-implements the substantiveness step (SPEC-037 REQ-005)
// to CONSUME the substantiveness pack's Q1 findings + Q2 extraction through the EXISTING
// resolveDispatchPackEngines() / dispatchPackEnginesFn seam (the same dispatcher code
// check and the pack_engines step use) — NOT a re-implemented dispatcher and NOT the
// deleted go/parser analyzer. It then routes the flat []gate.Violation result by
// namespaced rule ID, keys extraction findings per mandated test, and runs the
// language-agnostic gate set-join (NoTargetViolation) + hollow → test_substantiveness
// conversion. A spy on the real dispatchPackEnginesFn seam mechanically proves the step
// reached the dispatcher (CLM-015..017). When the substantiveness pack is not installed,
// the step is a no-op (the capability classifier governs the warn/block polarity).
func buildTestSubstantivenessStep(specDir, codeDir, projectRoot string, scope *gate.GateScope, classifier gate.SourceClassifier, matcher gate.TestNameMatcher) gate.StepFunc {
	return func(_ context.Context) gate.StepResult {
		mandated, err := gate.ExtractMandatedTests(specDir)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "failed to extract mandated tests: " + err.Error(), Severity: "error"}},
			}
		}

		// Implemented-only scope (ISSUE-054): the mandated-test-keyed noTarget join runs
		// only for `implemented` specs' mandated tests — draft / ready-for-implementation
		// specs describe planned code and must not produce false "does not call package X"
		// findings. Applied at the CONSUMER (the shared ExtractMandatedTests stays
		// unfiltered for artifact_status_drift) and BEFORE ResolveMandatedTestPaths / the
		// Q2 join. Q1 hollow findings are pack-produced over test files and stay unchanged.
		due := mandated[:0:0]
		for _, mt := range mandated {
			if gate.ContractsAreDue(mt.Status) {
				due = append(due, mt)
			}
		}
		mandated = due

		// Resolve file paths for found tests (the keying join's FilePath side) via the
		// SAME pack-declared discovery the verification step uses (classifier + matcher).
		mandated = gate.ResolveMandatedTestPaths(mandated, codeDir, classifier, matcher)

		packs, err := resolveSubstantivenessPacks(projectRoot)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "resolving substantiveness pack: " + err.Error(), Severity: "error"}},
			}
		}
		if len(packs) == 0 {
			// The substantiveness pack is not installed — no-op (capability-absent is
			// governed by the SPEC-036 classifier upstream, not a silent pass here).
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "pass",
				Violations: []gate.Violation{},
			}
		}

		// Reach the substantiveness pack's findings + extraction through the SAME
		// dispatch seam code check and the pack_engines step use (REQ-005). NOT a
		// re-implemented dispatcher; NOT the deleted analyzer.
		runner := &check.ExecCommandRunner{Dir: projectRoot}
		flat, err := resolveDispatchPackEngines()(packs, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, scope, runner)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "dispatching substantiveness pack: " + err.Error(), Severity: "error"}},
			}
		}

		// Route the flat stream by the pack-DECLARED substantiveness_role property (the
		// ISSUE-062 structured channel), not by matching a baked rule-name literal
		// (ISSUE-064). The pack stamps `hollow`/`referenced-symbol` on each finding, so
		// core carries no rule-name — or pack-name — routing key.
		hollow, extraction := gate.RouteSubstantivenessFindings(flat)

		var violations []gate.Violation
		// Q1 hollow → one test_substantiveness violation per routed hollow finding,
		// scope-filtered so out-of-scope test files are suppressed (REQ-008/CLM-029).
		for _, v := range gate.HollowFindingsToViolations(hollow) {
			if scope != nil && scope.Mode != gate.GateScopeModeAll && v.File != "" && !scope.Contains(v.File) {
				continue
			}
			violations = append(violations, v)
		}

		// Q2 noTarget set-join, in TWO passes (ISSUE-113).
		//
		// Pass 1 collects, and raises nothing. The two skips below are the SAME two the
		// single-pass loop applied, in the same order, and that order matters: an
		// unresolved or out-of-scope test must never inflate the eligible count, or a
		// diff-scoped run could refuse over tests it was not even looking at.
		var eligible []gate.MandatedTest
		for _, mt := range mandated {
			if mt.FilePath == "" {
				continue // not found — already reported by the verification step
			}
			if scope != nil && scope.Mode != gate.GateScopeModeAll && !scope.Contains(mt.FilePath) {
				continue
			}
			if gate.JoinEligibleForNoTarget(mt, testFileColocatedWithTarget(mt.FilePath, mt.TargetPkg)) {
				eligible = append(eligible, mt)
			}
		}

		// THE REFUSAL. When at least one test is join-eligible and BOTH routed partitions
		// are empty, the pack gave core no evidence on which to found any per-test verdict,
		// so the step reports ONE honest config-error instead of one unfounded "does not
		// call package X" violation per mandated test (397 of them in the incident
		// ISSUE-113 was filed from).
		//
		// len(hollow) is the ROUTED partition length, deliberately NOT len(violations) (the
		// scope-FILTERED hollow violations). Refusing is the drastic action, so any pack
		// output at all — even for a file this run is not looking at — must block it.
		// "Tidying" this to the filtered count makes the refusal MORE likely to fire, which
		// is the wrong direction.
		//
		// WHAT THIS RETURN DISCARDS, precisely: it drops whatever is in `violations`, which
		// at this point can only be scope-filtered hollow violations. But the refusal
		// requires len(hollow) == 0, and a scope filter cannot manufacture violations out of
		// an empty partition — so on this path `violations` is provably EMPTY and no hollow
		// verdict is lost. The per-test noTarget verdicts pass 2 would have raised ARE
		// declined by design, and under the bare-helper-assertion shape they can be TRUE;
		// that is why the refusal message names it as a candidate cause rather than
		// diagnosing the pack. This is not a claim that no true verdict is ever lost.
		//
		// The hollow == 0 guard is what makes even the narrow claim true. An earlier design
		// refused on (eligible, extraction) alone and therefore DID discard real hollow
		// violations, silently converting a correct RED into an exit-2 that blamed the
		// pack's config for a finding the operator needed to see.
		// TestE2E_HollowEvidenceBlocksZeroMatchRefusal pins that boundary end-to-end. Do
		// not simplify len(hollow) out of this call.
		//
		// ConfigErr is doing three mechanical jobs here, not one: Gate.Run halts the
		// remaining steps and returns exit 2 (pkg/gate/gate.go); waiver_resolution is
		// ordered after this step so it never runs and the refusal cannot be waived; and
		// ApplyPolicy skips ConfigErr steps so it cannot be baseline-grandfathered
		// (pkg/gate/policy.go). Removing ConfigErr for tidiness removes all three at once.
		if v, refuse := gate.SubstantivenessEvidenceRefusal(len(eligible), len(extraction), len(hollow)); refuse {
			return gate.StepResult{
				StepName:   gate.StepTestSubstantiveness,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{v},
			}
		}

		// Pass 2 — the raise, now over the eligible set. NoTargetViolationForTest is still
		// called rather than inlined: eligibility only ever spoke about the EMPTY case, and
		// the table remains the authority on whether the ACTUAL evidence satisfies the test.
		for _, mt := range eligible {
			referenced := gate.ReferencedSetForTest(extraction, mt)
			samePackage := testFileColocatedWithTarget(mt.FilePath, mt.TargetPkg)
			if v, raised := gate.NoTargetViolationForTest(mt, referenced, samePackage); raised {
				v.File = mt.FilePath
				violations = append(violations, v)
			}
		}

		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		if violations == nil {
			violations = []gate.Violation{}
		}
		return gate.StepResult{
			StepName:   gate.StepTestSubstantiveness,
			Status:     status,
			Violations: violations,
		}
	}
}

// testFileColocatedWithTarget reports whether a test file is the SAME UNIT as its
// target by LANGUAGE-NEUTRAL directory-leaf comparison (SPEC-045 REQ-003): the leaf
// of the test file's directory equals targetPkg. It REPLACES the deleted Go
// `package`-clause reader — no file read, no `package` clause, carrying no Go
// assumption, so a TS `.test.ts` co-located with its target is same-unit without any
// clause existing. An empty targetPkg yields false (the preserved guard).
func testFileColocatedWithTarget(filePath, targetPkg string) bool {
	return targetPkg != "" && filepath.Base(filepath.Dir(filePath)) == targetPkg
}

// mergeTestNameMatcher unions Manifest.TestNamePatterns across the declared toolchain
// packs and compiles ONE gate.TestNameMatcher (SPEC-045 REQ-006/CLM-018/CLM-036). It
// is built where the manifests are visible (cmd/backstop) so pkg/gate takes no
// pkg/pack dependency, and takes the wholesale declared-pack set (a manifest with no
// test_name_patterns contributes nothing to the union — no toolchain-only pre-filter,
// so it is NOT orphaned when SPEC-046 deletes the language: bridge). An INVALID
// pack-declared regex surfaces as a LOUD construction error (the caller turns it into
// a config-error step), never a silently-empty matcher that makes discovery find
// nothing and then mass-fail every mandated test.
func mergeTestNameMatcher(packSets ...[]*pack.Manifest) (gate.TestNameMatcher, error) {
	var patterns []string
	for _, set := range packSets {
		for _, manifest := range set {
			if manifest == nil {
				continue
			}
			patterns = append(patterns, manifest.TestNamePatterns...)
		}
	}
	return gate.NewTestNameMatcher(patterns)
}

// coverageRecordsFn produces the canonical per-FILE []check.CoverageRecord the
// coverage step consumes (SPEC-041 CLM-003). It is the typed seam between the gate
// and SPEC-042's dispatchPackCoverage producer.
type coverageRecordsFn func(scope *gate.GateScope) ([]check.CoverageRecord, error)

// coverageRecordsProducer returns a coverageRecordsFn that sources the canonical
// per-FILE []check.CoverageRecord from SPEC-042's dispatchPackCoverage producer
// over the DECLARED toolchain packs. It NEVER constructs a binary-resident
// `go test` runner (re-baking one would re-violate REQ-002); the per-file coverage
// signal originates from the declared toolchain coverage pass. A toolchain is an
// ordinary declared pack (SPEC-046), so it keys on the declared `packs:` set alone
// — there is no language-derived bridge set.
func coverageRecordsProducer(declared []*pack.Manifest, projectRoot string) coverageRecordsFn {
	return func(scope *gate.GateScope) ([]check.CoverageRecord, error) {
		if len(declared) == 0 {
			return nil, nil
		}
		runner := &check.ExecCommandRunner{Dir: projectRoot}
		packDir := filepath.Join(projectRoot, ".backstop", "packs")
		return dispatchPackCoverage(declared, packDir, projectRoot, scope, runner)
	}
}

// mergeSourceClassifier unions the classification.source and classification.test
// globs across the FULL declared-pack manifest set into one gate.SourceClassifier
// (SPEC-043 REQ-005/CLM-021/CLM-022). It takes the wholesale []*pack.Manifest set
// loadInstalledPacks resolves over `backstop.yml packs:` — NOT a toolchain-only
// pre-filter: a manifest with no `classification:` block contributes empty globs
// (zero to the union), so no filter is needed or wanted. A toolchain is just an
// ordinary declared pack (SPEC-046), so this merge sources from the declared
// `packs:` set with no language-derived bridge input and is never orphaned. Built
// where the manifests are visible (cmd/backstop) so pkg/gate takes no pkg/pack
// dependency.
func mergeSourceClassifier(packs []*pack.Manifest) gate.SourceClassifier {
	var source, test []string
	for _, manifest := range packs {
		if manifest == nil {
			continue
		}
		source = append(source, manifest.Classification.Source...)
		test = append(test, manifest.Classification.Test...)
	}
	return gate.NewSourceClassifier(source, test)
}

// buildCoverageStep creates a StepFunc that extracts spec verifications and checks
// the spec-declared coverage threshold PER FILE over the canonical
// []check.CoverageRecord produced by the records producer (SPEC-042's
// dispatchPackCoverage) — NOT a binary-resident `go test` runner (REQ-002). The
// records are produced lazily inside the step so the producer is exercised in the
// gate (CLM-003).
func buildCoverageStep(specDir, projectRoot string, scope *gate.GateScope, classifier gate.SourceClassifier, records coverageRecordsFn) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		specs, err := gate.ExtractSpecVerifications(specDir)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepCoverageThreshold,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: "coverage_threshold", Message: "failed to extract spec verifications: " + err.Error(), Severity: "error"}},
			}
		}

		var coverage []check.CoverageRecord
		if records != nil {
			coverage, err = records(scope)
			if err != nil {
				return gate.StepResult{
					StepName:   gate.StepCoverageThreshold,
					Status:     "fail",
					ConfigErr:  true,
					Violations: []gate.Violation{{Rule: "coverage_threshold", Message: "failed to produce coverage records: " + err.Error(), Severity: "error"}},
				}
			}
		}
		step := gate.StepCoverageThresholdScopedFunc(coverage, specs, scope, classifier)
		return step(ctx)
	}
}

// contractEngineResultsFn is a test seam: nil in production (the producer below
// runs the PACK path — real ast-grep signature presence + real grep symbol
// absence over the in-repo traceability pack — to build one ContractEngineResult
// per ContractEntry), overridden by the wiring/E2E tests so a spy can observe that
// buildContractStep consumes the PACK-produced results rather than the deleted
// go/parser analyzer (REQ-006/CLM-020/021/022). Declared WITHOUT an initializer so
// it holds no package-level mutable default.
var contractEngineResultsFn func(projectRoot string, contracts []gate.ContractEntry) ([]gate.ContractEngineResult, error)

// produceContractEngineResults builds the []ContractEngineResult buildContractStep
// verdicts off. In production it runs the PACK path (dispatchContractEntry ->
// resolveDispatchPackEngines -> sandboxed convert -> SARIF) over
// each declared contract entry; it NEVER routes to the deleted go/parser analyzer
// (CLM-021). A test seam may override it so the wiring spy can assert the pack path
// is the one consumed (CLM-020) and an unwired path fails (CLM-022).
func produceContractEngineResults(projectRoot string, contracts []gate.ContractEntry) ([]gate.ContractEngineResult, error) {
	if contractEngineResultsFn != nil {
		return contractEngineResultsFn(projectRoot, contracts)
	}
	// Resolve the contracts pack from the INSTALLED declaration (loadInstalledPacks) —
	// NOT from testdata and NOT from a baked path. When the pack is not installed, the
	// step produces NO results — a capability-absent no-op the SPEC-036 classifier
	// governs upstream (warn/exit 0 undeclared, block declared), which keeps an
	// uninstalled workspace from passing vacuously (REQ-014/CLM-048).
	packs, err := resolveContractsPacks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving contracts pack: %w", err)
	}
	if len(packs) == 0 {
		return []gate.ContractEngineResult{}, nil
	}
	manifest := packs[0]

	results := make([]gate.ContractEngineResult, 0, len(contracts))
	for _, c := range contracts {
		r, err := dispatchContractEntry(projectRoot, manifest, c)
		if err != nil {
			return nil, fmt.Errorf("dispatching contract entry for %s: %w", c.Name, err)
		}
		results = append(results, r)
	}
	return results, nil
}

// resolveContractsPacksFn is a test seam: nil in production (resolveContractsPacks below
// filters the INSTALLED packs to the contracts pack), overridable by tests. Declared
// WITHOUT an initializer so it holds no package-level mutable default.
var resolveContractsPacksFn func(projectRoot string) ([]*pack.Manifest, error)

// resolveContractsPacks returns the contracts pack manifest set the contract step
// dispatches. In production it filters the INSTALLED packs (loadInstalledPacks) to the
// contracts pack — the capability is installed-pack-resolvable, NOT built into the binary
// and NOT resolved from testdata (REQ-013). An empty result means the pack is not
// installed (capability-absent, governed upstream).
func resolveContractsPacks(projectRoot string) ([]*pack.Manifest, error) {
	if resolveContractsPacksFn != nil {
		return resolveContractsPacksFn(projectRoot)
	}
	installed, err := loadInstalledPacks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading installed packs: %w", err)
	}
	// Select the contracts pack by its DECLARED gate_type: contracts engine (ISSUE-063
	// REQ-004), not by NormalizedName — so a contracts pack under any name/org dispatches,
	// and a two-provider install fails loud rather than silently picking one.
	m, err := resolveCapabilityPack(installed, gate.DimensionContracts)
	if err != nil {
		return nil, fmt.Errorf("resolving contracts pack: %w", err)
	}
	if m == nil {
		return []*pack.Manifest{}, nil
	}
	return []*pack.Manifest{m}, nil
}

// dispatchContractEntry runs ONE contract entry through the REAL pack-engine dispatch
// seam (resolveDispatchPackEngines → dispatchPackEngines → runFindingsEngine), so the
// engine command clears the trusted-tool allowlist (CheckToolAllowed — REQ-005) and the
// convert script runs under the REAL sandbox (packval.SandboxedRunStdout — CLM-049). It
// builds a SINGLE-RULE in-memory manifest from the contracts pack's DECLARED engines: a
// signature entry compiles its Signature via the pack's compiler script (the COMPILER
// stays pack-side — the binary never compiles a pattern) and feeds the COMPILED pattern
// as the pattern-arg of the declared ast-grep engine; an absence entry feeds the forbidden
// symbol as the pattern-arg of the declared grep engine. Matched = the dispatch produced a
// finding for that contract's file; Scanned = the declared file/scope exists on disk (the
// file-scanned guard signal). NO raw exec, NO sandbox bypass.
func dispatchContractEntry(projectRoot string, manifest *pack.Manifest, c gate.ContractEntry) (gate.ContractEngineResult, error) {
	// Determine the scan target (file for a present signature; file-OR-path scope for an
	// absence) and the file-scanned guard signal.
	target := c.File
	if c.Absent && c.Scope != "" {
		target = c.Scope
	}
	if _, statErr := os.Stat(target); statErr != nil {
		// Unscanned/missing scope — no probe runs; the gate verdict raises a loud config
		// error for an absence entry (file-scanned guard) and a no-match for a present one.
		return gate.ContractEngineResult{Entry: c, Matched: false, Scanned: false}, nil
	}

	// Build the single rule the dispatch runs for this contract.
	var rule pack.Rule
	if c.Absent {
		rule = pack.Rule{ID: "contract-absence", Engine: "grep", Pattern: c.Name}
	} else {
		pattern, compileErr := compileContractSignature(projectRoot, manifest, c.Signature)
		if compileErr != nil {
			return gate.ContractEngineResult{}, fmt.Errorf("compiling signature for %s: %w", c.Name, compileErr)
		}
		rule = pack.Rule{ID: "contract-signature", Engine: contractSignatureEngine(manifest), Pattern: pattern}
	}

	// A single-rule manifest carrying the SAME declared engines as the installed pack, so
	// dispatch resolves the pack's engine bindings + convert scripts from packRoot.
	single := &pack.Manifest{
		Name:           manifest.Name,
		NormalizedName: manifest.NormalizedName,
		Language:       manifest.Language,
		Engines:        manifest.Engines,
		Content:        pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{rule}}},
	}

	// Scope the dispatch to exactly this contract's target via a FILE-mode gate scope, so
	// the engine scans only the declared file/path.
	scope, scopeErr := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{target})
	if scopeErr != nil {
		return gate.ContractEngineResult{}, fmt.Errorf("computing scope for %s: %w", c.Name, scopeErr)
	}

	runner := &check.ExecCommandRunner{Dir: projectRoot}
	packDir := filepath.Join(projectRoot, ".backstop", "packs")
	violations, dispErr := resolveDispatchPackEngines()([]*pack.Manifest{single}, packDir, projectRoot, scope, runner)
	if dispErr != nil {
		return gate.ContractEngineResult{}, fmt.Errorf("dispatching contract engine for %s: %w", c.Name, dispErr)
	}

	locs := make([]gate.SarifLocation, 0, len(violations))
	for _, v := range violations {
		locs = append(locs, gate.SarifLocation{File: v.File})
	}
	return gate.ContractEngineResult{Entry: c, Matched: len(violations) > 0, Scanned: true, Locations: locs}, nil
}

// contractSignatureEngine returns the pattern-arg ast-grep engine name the contracts pack
// declares for signature presence. It prefers a pack-declared "ast-grep-contracts" engine
// (the traceability dispatch pack) and falls back to the built-in "ast-grep" (the
// installable packs/contracts/ pack, which rides the DefaultRegistry ast-grep binding).
func contractSignatureEngine(manifest *pack.Manifest) string {
	if _, ok := manifest.Engines["ast-grep-contracts"]; ok {
		return "ast-grep-contracts"
	}
	return "ast-grep"
}

// compileContractSignature invokes the contracts pack's pack-relative signature compiler
// (scripts/compile-signature.sh) to turn the declared human-readable Signature into an
// ast-grep pattern. The COMPILER is pack-side (CLM-006): the binary shells the pack script
// and reads back the pattern; it never compiles or renders a signature itself. The
// resulting pattern is fed to the real ast-grep dispatch as the pattern-arg.
func compileContractSignature(projectRoot string, manifest *pack.Manifest, signature string) (string, error) {
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", filepath.FromSlash(manifest.NormalizedName))
	script := filepath.Join(packRoot, "scripts", "compile-signature.sh")
	out, err := exec.Command("/bin/sh", script, signature).Output()
	if err != nil {
		return "", fmt.Errorf("running pack signature compiler %s: %w", script, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// buildContractStep extracts contract entries from specs, routes them to the
// PACK-produced contracts SARIF path (ast-grep signature + grep absence) to build
// []ContractEngineResult, then feeds the rewritten gate.StepContractSignatureScopedFunc
// pack-SARIF consumer (REQ-006). It MUST NOT route to the deleted go/parser analyzer.
func buildContractStep(specDir, projectRoot string, scope *gate.GateScope) gate.StepFunc {
	return func(ctx context.Context) gate.StepResult {
		contracts, err := gate.ExtractContractEntries(specDir, projectRoot)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepContractSignature,
				Status:     "fail",
				Violations: []gate.Violation{{Rule: "contract_signature", Message: "failed to extract contracts: " + err.Error(), Severity: "error"}},
			}
		}

		results, err := produceContractEngineResults(projectRoot, contracts)
		if err != nil {
			return gate.StepResult{
				StepName:   gate.StepContractSignature,
				Status:     "fail",
				ConfigErr:  true,
				Violations: []gate.Violation{{Rule: "contract_signature", Message: "dispatching contract pack: " + err.Error(), Severity: "error"}},
			}
		}

		step := gate.StepContractSignatureScopedFunc(results, scope)
		return step(ctx)
	}
}

// buildWaiverPolicy builds the PRODUCTION waiver Policy by EXTRACTING the declared
// non-waivable sets from the INSTALLED pack manifests (SPEC-049 REQ-016 / CLM-069)
// — the CLM-027 "declared, not hardcoded" mechanism realized in production. A pack
// rule self-declares itself un-waivable via `non_waivable: true` in its manifest;
// the backstop/self pack marks its rules that way. Critical-severity secrets ship
// non-waivable as a severity-level policy (not a hardcoded rule list). cmd/backstop
// holds NO hardcoded list of protected rule-ids.
func buildWaiverPolicy(projectRoot string) (waiver.Policy, error) {
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading installed packs: %w", err)
	}
	var nonWaivableRules []string
	for _, m := range packs {
		if m == nil {
			continue
		}
		for _, r := range m.Content.Ruleset.Rules {
			if r.NonWaivable {
				nonWaivableRules = append(nonWaivableRules, pack.NamespacedRuleID(m.NormalizedName, r.ID))
			}
		}
	}
	return waiver.NewDeclaredPolicy(nonWaivableRules, []string{"critical"}), nil
}

// buildWaiverLineReader constructs the LineReader the waiver reconciliation pass
// consumes (SPEC-049 REQ-016): it yields the RAW bytes of a requested source line,
// resolved against the project root, with NO language knowledge and NO comment
// parsing. The scope is accepted to mirror the baseline analog's signature; reads
// are keyed by the repo-relative file path each finding carries.
func buildWaiverLineReader(projectRoot string, _ *gate.GateScope) waiver.LineReader {
	return func(file string, line int) (string, bool) {
		if line <= 0 {
			return "", false
		}
		f, err := os.Open(filepath.Join(projectRoot, filepath.FromSlash(file)))
		if err != nil {
			return "", false
		}
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		n := 0
		for scanner.Scan() {
			n++
			if n == line {
				return scanner.Text(), true
			}
		}
		return "", false
	}
}

// schemaIdentityList renders a cohort's per-SCHEMA identities, sorted so a gate result
// is byte-stable across runs.
//
// This is the per-SCHEMA surface. The per-ARTIFACT binding lives on the validate
// envelope's record array and is a genuinely different fact — a flat list of schema
// identities cannot express which artifact was validated against which schema.
func schemaIdentityList(cohort schema.Cohort) []string {
	out := make([]string, 0, len(cohort.Digests))
	for version := range cohort.Digests {
		if identity, ok := cohort.SchemaIdentity(version); ok {
			out = append(out, identity)
		}
	}
	sort.Strings(out)
	return out
}

// realArtifactValidator implements gate.ArtifactValidator by calling
// ValidateArtifacts with the embedded schema FS.
type realArtifactValidator struct {
	projectRoot string
	root        artifact.Root
	// cohort is the schema identity this validator asserts against. It is a FIELD
	// rather than a per-call computation so the gate's artifact_validation step
	// asserts on the same values the rest of the run reports. A zero cohort is
	// resolved lazily below, which is what keeps the several test constructions of
	// this struct working unchanged.
	cohort schema.Cohort
	// nonCorpus is the artifact-corpus exclusion set derived once in buildGateSteps
	// from the installed packs (ISSUE-122). It is a FIELD for the same reason cohort
	// is: so the gate's two corpus consumers below — the per-artifact validation
	// config and the ungated scan — cannot measure different corpora.
	//
	// The ZERO VALUE IS REACHABLE AND CORRECT. Every test construction of this struct
	// uses a keyed literal and wires no packs, so it gets artifact.NonCorpusDirs{},
	// which excludes the tool-agnostic base and nothing else — today-minus-declarations
	// rather than a walk into `.git`.
	nonCorpus artifact.NonCorpusDirs
}

func (v *realArtifactValidator) ValidateAll(_ context.Context) ([]gate.Violation, error) {
	// The gate asserts against the SAME cohort the CLI does, computed from the same
	// embedded schemas — the two cannot assert against different values.
	cohort := v.cohort
	if cohort.ID == "" {
		computed, err := schema.ComputeCohort(SchemaFS)
		if err != nil {
			return nil, &gate.ConfigError{Err: fmt.Errorf("computing schema cohort: %w", err)}
		}
		cohort = computed
	}
	cfg := ValidateConfig{
		ProjectRoot: v.projectRoot,
		Root:        v.root,
		All:         true,
		SchemaFS:    SchemaFS,
		Cohort:      cohort,
		NonCorpus:   v.nonCorpus,
	}

	result, err := ValidateArtifacts(cfg)
	if err != nil {
		return nil, &gate.ConfigError{Err: err}
	}

	// Convert validate.Violation to gate.Violation.
	var violations []gate.Violation
	for _, vv := range result.Violations {
		violations = append(violations, gate.Violation{
			Rule:     vv.Rule,
			File:     vv.File,
			Message:  vv.Message,
			Severity: vv.Severity,
		})
	}

	// THE ONE PLACE THE UNGATED SCAN IS INVOKED. A second call site would let two
	// scans disagree about which root they measured against, which is the class of bug
	// this whole spec is about.
	//
	// The findings arrive already ProjectWide-marked from the shared conversion, which
	// is what keeps them alive through diff-scoped filtering — an ungated artifact is
	// by definition a file nobody just edited.
	ungated, ungatedErr := gate.FindUngatedArtifacts(v.projectRoot, v.root, v.nonCorpus)
	if ungatedErr != nil {
		return nil, &gate.ConfigError{Err: fmt.Errorf("scanning for ungated artifacts: %w", ungatedErr)}
	}
	violations = append(violations, gate.UngatedFindingsToViolations(ungated)...)

	return violations, nil
}

// firstNonNil returns the first non-nil error from errs, or nil if all are nil.
// Used to collapse several independent flag-lookup errors into one guard.
func firstNonNil(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Verify that specDir exists before building steps — avoid confusing
// errors when specs directory is missing.
func specsExist(specDir string) bool {
	info, err := os.Stat(specDir)
	return err == nil && info.IsDir()
}
