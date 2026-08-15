package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/initialize"
	"github.com/spf13/cobra"
)

// newInitCommand builds `backstop init` (SPEC-069).
//
// IT IS THIN BY CONSTRUCTION, mirroring recipe_apply.go. It holds FOUR things and
// nothing else: flag definitions, flag->Options translation, report rendering, and
// exit-code mapping. Every decision about what init DOES lives in pkg/initialize; every
// piece of machinery it reaches lives behind a seam.
//
// IT CONSTRUCTS THE RUNNER FROM THE PRODUCTION ADAPTERS AND BUILDS NONE OF ITS OWN. A
// command wired to a test double passes every unit test in the plan and does nothing in
// a consumer's repository, so the five arguments below are the real ones.
//
// ★ THIS FILE IS SHARED CONTRACT SURFACE WITH SPEC-070 (`backstop doctor`), which owns
// exactly ONE symbol here: doctorGuidanceForSteps. Whichever spec lands second EDITS
// what the first left — it does not recreate the file, does not reformat it, and does
// not move or delete the other spec's symbol. Doctor guidance attaches to the TOOLCHAIN
// step ONLY and must never reach either recipe step; the no-CI-detection claim is the
// guard SPEC-070 explicitly defers to.
func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Take a project from nothing to a first gated run",
		Long: `Initializes backstop in a project directory, in ONE prompt-free invocation.

Bare ` + "`backstop init`" + ` runs the full default capability set: it initializes a git
repository if there is none, writes the profile-correct backstop.yml, scaffolds the
artifact layout, installs the packs you named, writes the canonical .gitignore, runs
each installed pack's declared test/build entrypoint once, delegates baseline seeding,
and finally runs the gate once and reports what it noticed.

Narrow the run with --no-<capability> or --only <capability>. The capability set is
exactly seven backstop names: git, sdlc, gitignore, packs, toolchain, baseline, observe.
Generating backstop.yml is not among them — it is unconditional, because an init that
does not write it produces nothing you can use.

Packs enter only through --pack, and only as PORTABLE git references: a local
filesystem path is refused, because it would record a machine-specific path in the
backstop.lock you commit.

CI wiring and the first source file are governed solely by --ci and --scaffold. Each
takes a WHOLE PINNED recipe reference, <pack>:<recipe>@<version>, which is handed to the
shipped recipe machinery verbatim — backstop constructs no part of it and holds no
default. Omitting either is a deliberate no-op that is reported, not an error.

Init never prompts and never reads standard input, so it behaves identically with no
TTY. Findings the gate reports are OBSERVATION: pre-existing violations in a project you
just started governing are never an init failure. The exit code carries broken promises
and nothing else.`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd)
		},
	}

	// Seven negations, one per capability, generated FROM the vocabulary so an eighth
	// flag cannot appear without an eighth capability existing first.
	for _, capability := range initialize.DefaultCapabilities() {
		cmd.Flags().Bool("no-"+string(capability), false,
			fmt.Sprintf("skip the %s capability", capability))
	}
	cmd.Flags().StringArray("only", nil,
		"run ONLY the named capability (repeatable; may not be combined with any --no- flag)")
	cmd.Flags().StringArray("pack", nil,
		"install a pack by PORTABLE git reference, e.g. <org>/<pack>@<version> (repeatable; a local filesystem path is refused)")
	// BOTH REF FLAGS CARRY WHOLE OPAQUE STRINGS. There is no --ci-param or
	// --scaffold-param, and no --no-ci or --no-scaffold: omission is the opt-out.
	cmd.Flags().String("ci", "",
		"wire CI by applying a pinned recipe an installed pack declares, as <pack>:<recipe>@<version>")
	cmd.Flags().String("scaffold", "",
		"scaffold a first source file by applying a pinned recipe an installed pack declares, as <pack>:<recipe>@<version>")

	return cmd
}

// runInit is the command body: translate flags, run, render, map the exit code.
func runInit(cmd *cobra.Command) error {
	options, err := initOptionsFromFlags(cmd)
	if err != nil {
		return err
	}

	// FAIL-CLOSED, AND A CONSTRUCTION FAILURE IS A CONFIG ERROR. There is no fallback
	// to a partial runner and no substituting a double for a seam that failed to
	// construct: a half-wired init is precisely what the constructor exists to make
	// unrepresentable.
	runner, constructErr := initialize.NewRunner(
		initPackInstaller{},
		initRecipeApplier{},
		initGateRunner{},
		&packToolchainProber{Runner: &check.ExecCommandRunner{Dir: options.ProjectRoot}},
		unavailableBaselineSeeder{},
	)
	if constructErr != nil {
		return &check.ConfigError{Message: constructErr.Error()}
	}

	result, runErr := runner.Run(options)
	if runErr != nil {
		// A step that refused — a local-path pack ref, an un-allowlisted entrypoint
		// tool — is a CONFIG error: the invocation or the installation is what must
		// change, and nothing was half-done.
		var configErr *check.ConfigError
		if errors.As(runErr, &configErr) {
			return configErr
		}
		return &check.ConfigError{Message: runErr.Error()}
	}

	renderInitReport(cmd, options, result)

	// THE EXIT CODE CARRIES BROKEN PROMISES AND NOTHING ELSE, and it is computed from
	// THE REQUEST rather than from the resulting filesystem state. Two structurally
	// identical no-ops carry different codes — `--ci` omitted exits 0 while `--ci`
	// supplied-and-unresolvable does not — and only the invocation separates them.
	// Pre-existing findings never contribute.
	if result.BrokenPromise {
		return &ExitCodeError{
			Code:      ExitViolations,
			Explained: true,
			Message:   "init did not deliver everything it was asked for; see the report above",
		}
	}
	return nil
}

// initOptionsFromFlags translates the command line into initialize.Options.
//
// EVERY REFUSAL HERE IS A CONFIG ERROR (exit 2), because in each case nothing has been
// written and the invocation is what must change.
func initOptionsFromFlags(cmd *cobra.Command) (initialize.Options, error) {
	only, onlyErr := cmd.Flags().GetStringArray("only")
	if onlyErr != nil {
		return initialize.Options{}, &check.ConfigError{Message: onlyErr.Error()}
	}
	packs, packErr := cmd.Flags().GetStringArray("pack")
	if packErr != nil {
		return initialize.Options{}, &check.ConfigError{Message: packErr.Error()}
	}
	ci, ciErr := cmd.Flags().GetString("ci")
	if ciErr != nil {
		return initialize.Options{}, &check.ConfigError{Message: ciErr.Error()}
	}
	scaffold, scaffoldErr := cmd.Flags().GetString("scaffold")
	if scaffoldErr != nil {
		return initialize.Options{}, &check.ConfigError{Message: scaffoldErr.Error()}
	}

	excluded := []string{}
	for _, capability := range initialize.DefaultCapabilities() {
		set, flagErr := cmd.Flags().GetBool("no-" + string(capability))
		if flagErr != nil {
			return initialize.Options{}, &check.ConfigError{Message: flagErr.Error()}
		}
		if set {
			excluded = append(excluded, string(capability))
		}
	}

	capabilities, resolveErr := initialize.ResolveCapabilities(only, excluded)
	if resolveErr != nil {
		return initialize.Options{}, &check.ConfigError{Message: resolveErr.Error()}
	}

	root, rootErr := os.Getwd()
	if rootErr != nil {
		return initialize.Options{}, &check.ConfigError{Message: fmt.Sprintf("resolving the working directory: %v", rootErr)}
	}

	return initialize.Options{
		ProjectRoot:       root,
		Capabilities:      capabilities,
		PackRefs:          packs,
		CIRecipeRef:       ci,
		ScaffoldRecipeRef: scaffold,
	}, nil
}

// renderInitReport prints ONE report naming every step and its outcome.
//
// It goes to stdout undivided. A report split across two streams makes every stream
// assertion about it unfalsifiable, and this report is the whole of what a consumer
// sees after their first invocation.
func renderInitReport(cmd *cobra.Command, options initialize.Options, result initialize.Result) {
	cmd.Printf("backstop init — %s\n", options.ProjectRoot)

	for _, step := range result.Steps {
		cmd.Printf("  %-10s %-18s %s\n", step.Step, step.Outcome, step.Detail)
	}

	if len(result.Observations) > 0 {
		cmd.Printf("\nWhat the gate noticed, by dimension:\n")
		for _, observation := range result.Observations {
			cmd.Printf("  %-28s %d\n", observation.Dimension, observation.Count)
		}
	}

	if len(result.Preserved) > 0 {
		cmd.Printf("\nFiles a recipe left in place:\n")
		for _, preserve := range result.Preserved {
			cmd.Printf("  %-40s %s\n", preserve.Path, preserve.Class)
		}
	}
}
