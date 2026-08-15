package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/recipe"
	"github.com/spf13/cobra"
)

// newRecipeCommand builds the `recipe` namespace. Both this parent and its `apply`
// subcommand carry Short AND Long: the agent-discovery tree walk
// (TestCLI_Help_AllCommandsHaveDescriptions) fails any command missing either.
func newRecipeCommand() *cobra.Command {
	recipeCmd := &cobra.Command{
		Use:   "recipe",
		Short: "Recipe adoption commands",
		Long: `Commands for adopting the file-operation recipes an installed pack declares.
A recipe is pack-declared DATA — its ops, targets, payloads and rewrite rules all
live in the pack — so this namespace carries no knowledge of any language,
framework, or CI platform.`,
	}
	recipeCmd.AddCommand(newRecipeApplyCommand())
	return recipeCmd
}

// newRecipeApplyCommand builds `backstop recipe apply <pack>:<recipe>@<version>`.
func newRecipeApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <pack>:<recipe>@<version>",
		Short: "Apply a pinned recipe from an installed pack",
		Long: `Applies one pinned recipe into the current project.

The reference is always fully pinned — <pack>:<recipe>@<version>; there is no
"latest" form. The recipe is resolved out of the INSTALLED pack corpus, its ops run
in the order the recipe declares them, and the adoption is recorded in the tracked
project-root record.

Use --param to supply a value for a param the recipe DECLARES. The flag is
repeatable, and any param left unsupplied falls back to the default the recipe
declares for it. A param declared required with no default can ONLY be supplied
this way — without it the apply cannot resolve, so it fails rather than writing at
an unresolved site.

A recipe's transform op is dispatched to the engine its pack DECLARES, and that
engine's tool must clear the same trusted-tool allowlist gate every pack-declared
enforcement command clears — an un-allowlisted or wrongly-pinned tool is refused
before any command is built.`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read INSIDE RunE rather than binding a captured variable:
			// NewRootCommand() is constructed fresh per invocation, and a captured
			// var would share state across them.
			supplied, flagErr := cmd.Flags().GetStringArray("param")
			if flagErr != nil {
				return &check.ConfigError{Message: flagErr.Error()}
			}

			projectRoot, rootErr := recipeProjectRoot()
			if rootErr != nil {
				return rootErr
			}

			// The declared kind is init's; `recipe apply` reports what the apply DID
			// and has no use for it.
			result, _, err := runRecipeApply(args[0], projectRoot, supplied)
			if err != nil {
				// A ConfigError is returned UNWRAPPED so it keeps its exit-2 shape
				// (main.go maps an untyped error to ExitConfigError) and stays
				// inspectable by errors.As. Every other failure is a violation: the
				// message already carries the op's declared manual instruction
				// verbatim, relayed by the applier.
				var configErr *check.ConfigError
				if errors.As(err, &configErr) {
					return configErr
				}
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
			}

			reportRecipeApply(cmd, args[0], result)
			return nil
		},
	}

	// StringArray, NOT StringSlice: StringSlice comma-splits its values, which
	// would silently turn `--param greeting=a,b` into two params and truncate a
	// value the operator supplied whole.
	cmd.Flags().StringArray("param", nil, "supply a value for a param the recipe declares, as key=value (repeatable; unsupplied params fall back to the recipe's declared defaults)")

	return cmd
}

// reportRecipeApply prints what the apply ACTUALLY DID.
//
// The gap this closes: the command used to print one static line whatever
// happened, so a run that CLOBBERED a local edit and a clean re-apply were
// indistinguishable to the operator — the same silence, in the success channel,
// that ISSUE-080 closes in the failure channel.
//
// Entries are emitted in the RESULT'S OWN SLICE ORDER, which is the recipe's
// declared op order. Sorting would be a second, invented ordering over an artifact
// whose sequence is its contract, for the same reason Apply never sorts ops.
//
// STREAM SPLIT, deliberately: the whole REPORT goes to stdout undivided — a report
// split across two streams makes every stream assertion about it unfalsifiable —
// while a WARNING is not part of the report of what was done and goes to stderr,
// where it is separately assertable. A clean preserve leaving stderr EMPTY is what
// makes the warning meaningful when it appears.
func reportRecipeApply(cmd *cobra.Command, ref string, result recipe.ApplyResult) {
	cmd.Printf("applied recipe %s\n", ref)

	for _, written := range result.Written {
		if clobberedDivergence(result, written) {
			cmd.Printf("  wrote %s (REGENERATED over a local divergence no active waiver accounted for)\n", written)
			continue
		}
		cmd.Printf("  wrote %s\n", written)
	}

	for _, preserved := range result.Preserved {
		if strings.TrimSpace(preserved.CoveringWaiver) == "" {
			// The user-owned branch never adjudicated anything, so there is no
			// token to quote — saying so is more honest than an empty citation.
			cmd.Printf("  preserved %s (the consumer's own file)\n", preserved.Path)
			continue
		}
		cmd.Printf("  preserved %s (accounted for by %s)\n", preserved.Path, preserved.CoveringWaiver)
	}

	if len(result.Written) == 0 && len(result.Preserved) == 0 {
		// Said explicitly rather than left as a bare headline, which reads
		// identically to a run that did work.
		cmd.Printf("  nothing was written or preserved\n")
	}

	for _, diagnostic := range result.Diagnostics {
		cmd.PrintErrf("warning: %s:%d: %s\n", diagnostic.File, diagnostic.Line, diagnostic.Message)
	}
}

// clobberedDivergence reports whether a written target overwrote an
// unaccounted-for divergence. It reads the applier's own Regenerated list — a
// strict subset of Written — rather than re-deriving the distinction here, so the
// CLI reports DATA instead of guessing.
func clobberedDivergence(result recipe.ApplyResult, target string) bool {
	for _, regenerated := range result.Regenerated {
		if regenerated == target {
			return true
		}
	}

	return false
}

// recipeProjectRoot resolves the project root from the discovered backstop.yml, the
// same way the gate does, so a recipe writes beneath the project the config declares
// rather than beneath whatever directory the operator happened to be in.
func recipeProjectRoot() (string, error) {
	cfgPath, err := config.DiscoverConfigPath()
	if err != nil {
		return "", &check.ConfigError{Message: fmt.Sprintf("config: %s", err)}
	}
	return filepath.Dir(cfgPath), nil
}

// runRecipeApply applies one pinned recipe reference into projectRoot.
//
// It is thin, and it is THE HOME OF THE TRANSFORM TRUST GATE: this is the only layer
// that can see the pack's declared `engines:` block, so it is the only layer that has
// a tool name and a declared version to check. The ordering below is load-bearing —
// resolve, select the engine from declared data, GATE, and only then build the
// dispatch — so an un-allowlisted tool's command is never even constructed.
//
// The exit-code split follows the boundary the operator cares about: everything that
// happens BEFORE the apply — an unusable reference, an uninstalled pack, a recipe the
// pack does not index, an engine backstop will not run, and every `--param` rejection
// below (malformed, duplicate, undeclared) — is a *check.ConfigError (exit 2), because
// nothing was applied and the invocation or the installation is what must change. Only
// a failure of the apply ITSELF is a violation (exit 1), and it carries the op's
// declared manual instruction verbatim.
//
// suppliedParams arrives as the raw repeated `--param key=value` arguments so the
// rejections can quote what the operator actually typed.
//
// IT ALSO RETURNS THE RESOLVED RECIPE'S DECLARED KIND (SPEC-069 REQ-035). The kind is
// already in hand after the single recipe.ResolveRecipe call below — this function just
// did not return it. `backstop init` needs it because REQ-035's preserve classifier
// splits an empty Rule/CoveringWaiver pair on the recipe's declared kind, so init's
// RecipeApplier must report the kind of the recipe that was ACTUALLY APPLIED. Its only
// alternative would be to call ParseRecipeRef + ResolveRecipe a SECOND time to ask what
// kind it was — a second derivation of "which recipe is this ref", drifting from the one
// the apply used, which is the exact one-authority-not-a-copy hazard REQ-018 refuses for
// local-path classification. Take it from the resolve.
//
// BEHAVIOR IS OTHERWISE UNCHANGED: the same steps in the same order, the same
// *check.ConfigError-versus-violation split, and the same ZERO recipe.ApplyResult on
// every failure path — now paired with an EMPTY kind, because a run that resolved
// nothing has no kind to report.
func runRecipeApply(ref string, projectRoot string, suppliedParams []string) (recipe.ApplyResult, string, error) {
	// Purely syntactic and needs no filesystem, so it runs first: a malformed
	// invocation is reported without depending on the project being resolvable.
	params, err := parseParamFlags(suppliedParams)
	if err != nil {
		return recipe.ApplyResult{}, "", fmt.Errorf("parsing --param flags: %w", err)
	}

	parsed, err := recipe.ParseRecipeRef(ref)
	if err != nil {
		return recipe.ApplyResult{}, "", &check.ConfigError{Message: err.Error()}
	}

	manifests, err := loadInstalledPacks(projectRoot)
	if err != nil {
		return recipe.ApplyResult{}, "", &check.ConfigError{Message: err.Error()}
	}
	corpus := make(map[string]*pack.Manifest, len(manifests))
	for _, manifest := range manifests {
		corpus[manifest.NormalizedName] = manifest
	}

	packsDir := filepath.Join(projectRoot, ".backstop", "packs")
	resolved, err := recipe.ResolveRecipe(parsed, corpus, filepath.Join(packsDir, filepath.FromSlash(parsed.Pack)))
	if err != nil {
		return recipe.ApplyResult{}, "", &check.ConfigError{Message: err.Error()}
	}
	// AFTER resolution (the recipe's declared param schema is only knowable now)
	// and BEFORE the apply, so a typo'd name is refused while nothing has been
	// written. Deliberately at the CLI layer only: recipe.Apply and every
	// SDLC-mediated library caller are unchanged, because this is invocation
	// hygiene rather than applier policy.
	if err := checkUndeclaredParams(resolved.Manifest, params); err != nil {
		return recipe.ApplyResult{}, "", fmt.Errorf("checking supplied params against %q: %w", ref, err)
	}

	// Resolution succeeded, so the pack IS in the corpus under this key.
	packManifest := corpus[parsed.Pack]

	binding, err := provisionedEngineBinding(packManifest)
	if err != nil {
		return recipe.ApplyResult{}, "", fmt.Errorf("select the transform engine for recipe %s: %w", ref, err)
	}

	// TRUST GATE — the SAME check every pack-declared enforcement command clears
	// (checkEngineToolAllowed → engine.CheckToolAllowed over
	// resolveTrustedToolAllowlist). It is reused rather than re-implemented so there
	// is exactly ONE gate, and it runs HERE, before the dispatch closure below is
	// built, mirroring runFindingsEngine's gate-before-command-construction order.
	if gateErr := checkEngineToolAllowed(packManifest, binding); gateErr != nil {
		return recipe.ApplyResult{}, "", fmt.Errorf("gate the transform engine for recipe %s: %w", ref, gateErr)
	}

	result, err := recipe.Apply(resolved, recipe.ApplyOptions{
		Mode:        recipe.ModeDirect,
		ProjectRoot: projectRoot,
		Params:      params,
		Dispatch:    transformDispatch(binding, projectRoot),
		// ReadWaivers is deliberately nil: that selects the applier's OWN reader over
		// the real pkg/waiver read path. Supplying a second reader here would fork the
		// adjudication into two implementations.
	})
	if err != nil {
		// The ZERO result on failure, matching Apply's own contract: an apply
		// either produces a verdict or it fails, never both. The wrap names the
		// project the apply ran against, which the applier's own error — scoped to
		// the recipe and the op — does not carry.
		return recipe.ApplyResult{}, "", fmt.Errorf("apply into project %q: %w", projectRoot, err)
	}

	return result, resolved.Manifest.Kind, recordRecipeAdoption(projectRoot, result)
}

// parseParamFlags turns the repeated `--param key=value` arguments into the
// supplied param scope.
//
// The grammar, pinned because a plausible alternative is silently wrong in each
// case: the split takes the FIRST `=` only, so a VALUE may contain `=`; an EMPTY
// value is legal and supplies the empty string, which is meaningfully distinct
// from unsupplied; a missing `=` or an empty KEY is refused; and a DUPLICATE key
// is refused rather than last-wins. A map write silently keeps the last value,
// which would discard something the operator typed without ever saying so.
func parseParamFlags(supplied []string) (map[string]string, error) {
	params := make(map[string]string, len(supplied))

	for _, raw := range supplied {
		key, value, separated := strings.Cut(raw, "=")
		if !separated {
			return nil, &check.ConfigError{Message: fmt.Sprintf(
				"--param %q is malformed: a param is supplied as key=value", raw)}
		}
		if strings.TrimSpace(key) == "" {
			return nil, &check.ConfigError{Message: fmt.Sprintf(
				"--param %q supplies an empty param name: a param is supplied as key=value", raw)}
		}
		if _, duplicate := params[key]; duplicate {
			return nil, &check.ConfigError{Message: fmt.Sprintf(
				"--param %q supplies %q a second time; a param may be supplied once, and silently keeping the last value would discard one you typed", raw, key)}
		}
		params[key] = value
	}

	return params, nil
}

// checkUndeclaredParams refuses a supplied param the resolved recipe does not
// declare.
//
// Without it a typo (`--param app_nmae=x`) is silently dropped and resurfaces as
// an unresolvable-placeholder failure the operator cannot attribute to their own
// invocation — the same undiagnosable shape ISSUE-081 was filed about. The names
// are SORTED because map iteration order would make the message nondeterministic
// across runs.
func checkUndeclaredParams(manifest *recipe.RecipeManifest, supplied map[string]string) error {
	declared := make(map[string]struct{}, len(manifest.Params))
	declaredNames := make([]string, 0, len(manifest.Params))
	for _, spec := range manifest.Params {
		declared[spec.Name] = struct{}{}
		declaredNames = append(declaredNames, spec.Name)
	}

	undeclared := make([]string, 0, len(supplied))
	for name := range supplied {
		if _, known := declared[name]; !known {
			undeclared = append(undeclared, name)
		}
	}
	if len(undeclared) == 0 {
		return nil
	}
	sort.Strings(undeclared)
	sort.Strings(declaredNames)

	return &check.ConfigError{Message: fmt.Sprintf(
		"--param supplied %v, which this recipe does not declare; it declares %v",
		undeclared, declaredNames)}
}

// provisionedEngineBinding selects the transform engine from DECLARED DATA: the
// pack's single provisioned engine binding.
//
// An op declares no engine, so the pack's declaration is the only source. EXACTLY ONE
// provisioned binding is required — `Manifest.Engines` is a map, so picking among
// several would be nondeterministic, and none means there is no tool to gate or run.
// Both are fail-loud config errors naming the pack and the count rather than a
// silently-chosen engine.
func provisionedEngineBinding(manifest *pack.Manifest) (engine.EngineBinding, error) {
	provisioned := make([]string, 0, len(manifest.Engines))
	for name, spec := range manifest.Engines {
		if spec.Binding.Provision != nil {
			provisioned = append(provisioned, name)
		}
	}
	sort.Strings(provisioned)

	if len(provisioned) != 1 {
		return engine.EngineBinding{}, &check.ConfigError{Message: fmt.Sprintf(
			"pack %s declares %d provisioned engine bindings (%v): applying a recipe requires EXACTLY ONE, because a recipe op declares no engine and choosing among several would be nondeterministic",
			manifest.NormalizedName, len(provisioned), provisioned,
		)}
	}

	binding := manifest.Engines[provisioned[0]].Binding
	// A binding with no command is unrunnable, and inventing one would be precisely the
	// baked tool knowledge the applier must not carry. Refuse HERE, with selection,
	// rather than mid-apply: nothing has run yet, so this is a configuration defect the
	// pack must fix, not a violation of the consumer's project.
	if strings.TrimSpace(binding.Command) == "" {
		return engine.EngineBinding{}, &check.ConfigError{Message: fmt.Sprintf(
			"pack %s declares engine %q with no command, so it cannot run a transform",
			manifest.NormalizedName, provisioned[0],
		)}
	}

	return binding, nil
}

// transformDispatch builds the production transform seam over the pack's declared
// binding. It is reached only AFTER selection accepted the binding and the trust gate
// passed, so the command is known to be present and its tool allowlisted.
//
// The argv is shaped exactly as the enforcement dispatch shapes it — the declared
// command split into program + leading args, then the declared input flag, the rule
// file, and the target — and it runs through the same check.CommandRunner path, with
// the working directory pinned to the project root.
//
// It is NOT sandboxed, and must not be: the sandboxed runner is the convert-step
// runner, whose profile denies all file writes. A transform's entire purpose is to
// WRITE the consumer's file, so it is structurally impossible under that profile —
// and relaxing the profile to fit would be a security regression. The enforcement
// dispatch does not run engine commands sandboxed either.
func transformDispatch(binding engine.EngineBinding, projectRoot string) recipe.TransformDispatch {
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	return func(rule string, target string) error {
		name, args := splitCommand(binding.Command)
		if binding.InputFlag != "" {
			args = append(args, binding.InputFlag)
		}
		args = append(args, rule, target)

		// COMBINED output, not the clean-stdout capture the findings dispatch uses: a
		// findings engine's stdout IS its SARIF contract, whereas a transform's product
		// is the rewritten FILE and its stdout carries nothing. An engine that refuses
		// to run explains itself on stderr, so capturing stdout alone would reduce a
		// precise diagnostic to a bare exit code.
		output, runErr := runner.Run(context.Background(), name, args...)
		if runErr == nil {
			return nil
		}
		return fmt.Errorf("the declared engine %q exited non-zero (%w): %s", binding.Command, runErr, strings.TrimSpace(string(output)))
	}
}

// recordRecipeAdoption persists the apply's thin adoption entry at the project root.
//
// It is written only when the run actually WROTE something: a run that changed no
// file adopted nothing, and recording it anyway would tell the NEXT apply that a
// file the consumer owns is recipe-owned and safe to regenerate over. The upsert is
// idempotent, so it composes with the applier's own record write rather than
// competing with it.
func recordRecipeAdoption(projectRoot string, result recipe.ApplyResult) error {
	if len(result.Written) == 0 {
		return nil
	}

	path := filepath.Join(projectRoot, recipe.AdoptionRecordName)
	record, err := recipe.ReadAdoptions(path)
	if err != nil {
		return err
	}
	if record.Recipes == nil {
		record.Recipes = make(map[string]recipe.AdoptionEntry, 1)
	}
	record.Recipes[result.Adoption.Recipe] = result.Adoption

	return recipe.WriteAdoptions(path, record)
}
