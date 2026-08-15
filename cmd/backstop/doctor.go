package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/spf13/cobra"
)

// doctor.go is `backstop doctor` (SPEC-070): the command you run when everything else
// refuses.
//
// THE LOAD-BEARING POSTURE is the inverse of the gate's. runGate turns a config load
// failure into an immediate exit-2 (gate.go:67-73); runDoctor turns the SAME failure
// into a reported check result and keeps going, because a diagnostic that will not start
// on a broken setup cannot diagnose a broken setup. There is no branch here that returns
// a config-error exit code, and that absence is asserted by a source scan rather than
// left to review.
//
// EVERY CONSUMER READS ONE REGISTRY. The --check selector, the human renderer, the
// --json renderer and the exit-code computation all read the results ONE enumeration of
// doctorRegistry() produced, so a check cannot be present in one consumer and missing
// from another.

// The ONE check-id source. doctorRegistry builds its entries from these constants and no
// other code may write a check id as a literal — which is what makes a rename a
// compile-time event rather than a silent desync between the registry and the guidance
// init prints.
const (
	doctorCheckConfigPresent  = "config-present"
	doctorCheckConfigLoads    = "config-loads"
	doctorCheckGitRepository  = "git-repository"
	doctorCheckPacksInstalled = "packs-installed"
	doctorCheckBuildIdentity  = "build-identity"
	doctorCheckToolchainRuns  = "toolchain-runs"
	doctorCheckArtifactLayout = "artifact-layout"
)

// The four statuses REQ-002 declares. `skipped` is produced by the one-condition-one-
// owner rule: a check whose input could not be gathered is skipped naming the check that
// owns the condition, never failed a second time.
const (
	doctorStatusPass    = "pass"
	doctorStatusWarn    = "warn"
	doctorStatusFail    = "fail"
	doctorStatusSkipped = "skipped"
)

// doctorSchemaVersion is the --json payload's declared shape.
const doctorSchemaVersion = "doctor/v1"

// doctorCheck is one registry entry.
type doctorCheck struct {
	// ID is the stable lowercase-kebab identifier init prints and --check selects on.
	ID string
	// Title is the human name.
	Title string
	// Run returns EXACTLY ONE result. A check that executes several things — the
	// toolchain check is the case — enumerates them inside its single message and
	// reports the worst outcome as its status.
	Run func(ctx doctorContext) doctorResult
}

// doctorResult is one check's reported outcome.
//
// ONE STRUCT FEEDS BOTH RENDERERS, so the human form and the --json payload cannot
// disagree about the check set or the statuses. The json tags ARE the doctor/v1
// per-check shape.
//
// Remediation carries NO omitempty DELIBERATELY: REQ-001 requires the key present for
// every check that ran, so a passing check emits an empty string rather than dropping the
// key and making the payload's shape depend on its status.
type doctorResult struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

// doctorReport is the --json payload.
type doctorReport struct {
	SchemaVersion string         `json:"schema_version"`
	Checks        []doctorResult `json:"checks"`
}

// doctorRegistry is THE single source of doctor's checks.
//
// A SLICE, NOT A MAP, because report order is a requirement — map iteration would make
// the report non-deterministic across runs. The four setup checks come FIRST and
// permanently, so a reader sees why a later check skipped before reading the skip.
//
// It has exactly TWO non-test readers with distinct roles, and no third: runDoctor, the
// ONE site that ENUMERATES it — every report's check SET comes from that one enumeration
// — and doctorGuidance, which resolves a SINGLE id against it, returns no set and feeds
// no report. A second ENUMERATION is what would let two consumers disagree about the
// check set; a keyed single-id lookup cannot.
func doctorRegistry() []doctorCheck {
	return []doctorCheck{
		{ID: doctorCheckConfigPresent, Title: "backstop.yml is discoverable", Run: checkConfigPresent},
		{ID: doctorCheckConfigLoads, Title: "backstop.yml loads and validates", Run: checkConfigLoads},
		{ID: doctorCheckGitRepository, Title: "the project root is a git work tree", Run: checkGitRepository},
		{ID: doctorCheckPacksInstalled, Title: "declared packs are installed", Run: checkPacksInstalled},
		{ID: doctorCheckBuildIdentity, Title: "the running binary's build identity", Run: checkBuildIdentity},
		{ID: doctorCheckToolchainRuns, Title: "pack-declared test/build entrypoints execute", Run: checkToolchainRuns},
		{ID: doctorCheckArtifactLayout, Title: "artifacts sit where the resolved root expects them", Run: checkArtifactLayout},
	}
}

// doctorGuidance resolves ONE check id against the registry and returns the printable
// guidance, or ("", false) when no entry carries it.
//
// IT IS A KEYED LOOKUP, NEVER AN ENUMERATION — slices.IndexFunc rather than a range —
// which is what keeps it the registry's second permitted reader without becoming a
// second source of the check set. Any code outside doctor.go that names a doctor check
// goes through here, so an unregistered id is UNPRINTABLE rather than printed wrong.
func doctorGuidance(checkID string) (string, bool) {
	registry := doctorRegistry()
	index := slices.IndexFunc(registry, func(entry doctorCheck) bool { return entry.ID == checkID })
	if index < 0 {
		return "", false
	}
	return fmt.Sprintf("run `backstop doctor --check %s` to diagnose this", registry[index].ID), true
}

// newDoctorCommand builds `backstop doctor`.
//
// IT TAKES THE ROOT PERSISTENT --json FLAG BY POINTER, exactly as newGateCommand and the
// nine newPack*Command constructors do. Declaring a local --json here would SHADOW the
// root one: `backstop doctor --json` would pass every test that types it while
// `backstop --json doctor` — the form users type — silently rendered human text.
//
// It owns exactly one flag of its own, --check, and nothing else. Every decision it
// renders comes from runDoctor.
func newDoctorCommand(jsonFlag *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose a backstop setup, including the conditions that make other commands refuse",
		Long: `Reports on a project's backstop setup, one result per registered check.

Doctor is the command you run when everything else refuses. An absent, unparseable, or
gate-fatal backstop.yml is reported as a CHECK RESULT carrying remediation rather than as
a reason not to start, so a broken setup is diagnosed instead of merely rejected.

A bare invocation runs EVERY registered check. --check <id> reports on one of them, and
an unknown id is a loud error naming the registered ids rather than an empty successful
run. Warnings and skips do not fail the run: the exit code is 1 when at least one check
FAILS and 0 otherwise.`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd, jsonFlag, args)
		},
	}
	cmd.Flags().String("check", "", "report on ONE registered check by id, instead of every check")
	return cmd
}

// runDoctor gathers the context ONCE, enumerates the registry ONCE, renders, and maps the
// exit code.
//
// NO PATH HERE RETURNS A CONFIG-ERROR EXIT. Every gathering failure reaches the checks as
// DATA on the context, which is the whole mechanism behind REQ-003.
func runDoctor(cmd *cobra.Command, jsonFlag *bool, args []string) error {
	// args is unused: doctor takes no positional arguments (cobra.NoArgs enforces that).
	// It stays in the signature because the declared contract carries it.
	_ = args

	selector, flagErr := cmd.Flags().GetString("check")
	if flagErr != nil {
		return &ExitCodeError{Code: ExitViolations, Message: flagErr.Error()}
	}

	ctx := gatherDoctorContext()

	// THE ONE ENUMERATION. Every consumer below reads the results it produced.
	var results []doctorResult
	registered := []string{}
	for _, entry := range doctorRegistry() {
		registered = append(registered, entry.ID)
		if selector != "" && entry.ID != selector {
			continue
		}
		results = append(results, entry.Run(ctx))
	}

	// An unknown selector is a LOUD error naming the registered ids — never an empty,
	// successful run, which would be a diagnostic reporting nothing and calling it fine.
	if selector != "" && len(results) == 0 {
		return &ExitCodeError{
			Code: ExitViolations,
			Message: fmt.Sprintf("unknown check %q; the registered checks are: %s",
				selector, strings.Join(registered, ", ")),
		}
	}

	if err := renderDoctorReport(cmd, jsonFlag, results); err != nil {
		return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
	}

	for _, result := range results {
		if result.Status == doctorStatusFail {
			return &ExitCodeError{
				Code:      ExitViolations,
				Explained: true,
				Message:   "doctor reported at least one failing check; see the report above",
			}
		}
	}
	return nil
}

// renderDoctorReport prints the results. BOTH renderings consume the same slice.
func renderDoctorReport(cmd *cobra.Command, jsonFlag *bool, results []doctorResult) error {
	if jsonFlag != nil && *jsonFlag {
		payload := doctorReport{SchemaVersion: doctorSchemaVersion, Checks: results}
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		cmd.Println(string(encoded))
		return nil
	}

	cmd.Println("backstop doctor")
	for _, result := range results {
		cmd.Printf("  %-16s %-8s %s — %s\n", result.ID, result.Status, result.Title, indentDoctorContinuation(result.Message))
		if result.Remediation != "" {
			cmd.Printf("  %-16s %-8s %s\n", "", "", indentDoctorContinuation(result.Remediation))
		}
	}
	return nil
}

// indentDoctorContinuation aligns a multi-line message under the report's message column.
//
// A check that enumerates several things inside its one message — toolchain-runs is the
// case — otherwise emits continuation lines flush against the left margin, where they read
// as separate checks rather than as detail belonging to the line above.
func indentDoctorContinuation(text string) string {
	return strings.ReplaceAll(text, "\n", "\n"+strings.Repeat(" ", 28))
}

// gatherDoctorContext builds the ONE context every check reads.
//
// THIS IS THE ONLY PLACE A LOADER IS CALLED, and each is called exactly once, BEFORE the
// registry runs. Every error is CARRIED on the context rather than raised: that inversion
// is the whole of REQ-003, and it is why no check can turn a load failure into a refusal
// to start. A loader call inside a check would be a second abort path, and it would read
// as ordinary defensive error handling in review.
func gatherDoctorContext() doctorContext {
	ctx := doctorContext{}

	searchDir, wdErr := os.Getwd()
	if wdErr != nil {
		// Even this is carried rather than raised: the checks report on what they were
		// given, and an unresolvable working directory is a condition to state, not a
		// reason to refuse.
		searchDir = "."
	}
	ctx.SearchDir = searchDir

	ctx.ConfigPath, ctx.ConfigPathErr = config.DiscoverConfigPath()

	// The project-root fallback mirrors runGate's (gate.go:75-80): the directory of the
	// discovered config, or the directory the search started from.
	ctx.ProjectRoot = searchDir
	if ctx.ConfigPathErr == nil {
		ctx.ProjectRoot = filepath.Dir(ctx.ConfigPath)
	}

	if ctx.ConfigPathErr == nil {
		ctx.Config, ctx.ConfigErr = config.LoadConfig()
	}
	if ctx.ConfigPathErr == nil && ctx.ConfigErr == nil {
		ctx.Packs, ctx.PacksErr = loadInstalledPacks(ctx.ProjectRoot)
	}

	// The runner is ROOTED AT THE PROJECT ROOT here, at construction, because
	// check.CommandRunner has no working-directory argument — the directory is a
	// construction-time property of check.ExecCommandRunner. Every consumer of this
	// context inherits that rooting rather than introducing a second one.
	ctx.Runner = &check.ExecCommandRunner{Dir: ctx.ProjectRoot}

	return ctx
}
