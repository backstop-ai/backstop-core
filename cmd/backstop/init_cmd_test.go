package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/initialize"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/backstop-ai/backstop-core/pkg/recipe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// init_cmd_test.go drives `backstop init` through the ASSEMBLED root command, so every
// claim here is about the command a consumer actually runs — its flags, its report, and
// above all its EXIT CODE.
//
// NO t.Parallel ANYWHERE IN THIS SUITE: runInitCommand calls t.Chdir, which mutates
// process-wide state, and the hermetic pack redirects use t.Setenv.
//
// ★ THE EXIT-CODE MATRIX IS THE POINT OF THIS FILE. Its organizing rule is REQ-014's
// precedence clause: PRE-EXISTING FINDINGS ARE NEVER AN INIT FAILURE, but an init STEP
// that failed to deliver what it promised ALWAYS is. Capability-absent outcomes are
// neither — nothing promised them.

// runInitCommand runs `backstop init` in project through the real command tree and
// returns its combined output and its TRUE exit code.
//
// The exit code comes from reportError — main's OWN mapping — rather than from a
// reimplementation here, so a change to how the binary classifies errors shows up in
// these assertions instead of silently diverging from them.
func runInitCommand(t *testing.T, project string, args ...string) (string, int) {
	t.Helper()
	t.Chdir(project)

	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"init"}, args...))

	err := root.Execute()
	if err == nil {
		return out.String(), ExitPass
	}
	// reportError WRITES the diagnostic, so it must run before the buffer is read.
	// Returning `out.String(), reportError(&out, err)` evaluates the read first and
	// silently drops the very message these claims assert on.
	code := reportError(&out, err)
	return out.String(), code
}

// initCmdHermeticPack publishes a fixture pack source as a hermetic remote and returns
// its pinned ref plus a fresh empty project.
func initCmdHermeticPack(t *testing.T, fixture string) (packRef, project string) {
	t.Helper()
	source, err := filepath.Abs(filepath.Join("testdata", "hermetic-remote", fixture))
	if err != nil {
		t.Fatalf("resolving the %s fixture: %v", fixture, err)
	}
	remote := newHermeticRemote(t, source, "v1.0.0")
	redirectPackURL(t, remoteE2EOrg, fixture, remote.Path)
	assertPackURLRedirected(t, remoteE2EOrg, fixture, remote)
	return remoteE2EOrg + "/" + fixture + "@1.0.0", t.TempDir()
}

// ═══════════════════════════════════════════════════════════════════════════════
// THE COMMAND, END TO END
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_SingleInvocationReachesFirstValue (SPEC-069 CLM-001).
//
// ONE invocation in an EMPTY directory leaves backstop.yml, the .backstop/ layout and a
// .gitignore on disk and prints ONE report. No second command is required, and init must
// not instruct the consumer to run a further command to complete a step it itself
// claimed.
func TestInit_SingleInvocationReachesFirstValue(t *testing.T) {
	project := t.TempDir()

	output, code := runInitCommand(t, project)
	if code != ExitPass {
		t.Fatalf("a bare init in an empty directory exited %d\n%s", code, output)
	}

	for _, path := range []string{
		"backstop.yml", ".gitignore",
		".backstop/bundles", ".backstop/specs", ".backstop/plans",
		".backstop/issues", ".backstop/adrs", ".backstop/directives",
	} {
		if _, err := os.Stat(filepath.Join(project, path)); err != nil {
			t.Fatalf("%s is not on disk after a single invocation: %v\n%s", path, err, output)
		}
	}

	// ONE report, naming every step it ran.
	if strings.Count(output, "backstop init —") != 1 {
		t.Fatalf("the run printed %d reports, want exactly one\n%s", strings.Count(output, "backstop init —"), output)
	}
	// The steps init CLAIMED must not tell the consumer to go and RUN something to
	// finish them. The skipped steps legitimately do name a follow-up command — that is
	// the honest-skip report — so only the delivered ones are checked here.
	for _, claimed := range []string{"git", "config", "layout", "gitignore"} {
		line := reportLineFor(t, output, claimed)
		if strings.Contains(line, "run `backstop") {
			t.Fatalf("the %s step told the consumer to run a further command to complete what init claimed:\n%s", claimed, line)
		}
	}
}

// reportLineFor returns the report line for the named step.
func reportLineFor(t *testing.T, output, step string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), step+" ") {
			return line
		}
	}
	t.Fatalf("no report line for step %q\n%s", step, output)
	return ""
}

// ═══════════════════════════════════════════════════════════════════════════════
// CONFIG ERRORS — EXIT 2
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_OnlyAndNoFlagsCombinedIsAConfigError (SPEC-069 CLM-012).
func TestInit_OnlyAndNoFlagsCombinedIsAConfigError(t *testing.T) {
	output, code := runInitCommand(t, t.TempDir(), "--only", "git", "--no-observe")

	if code != ExitConfigError {
		t.Fatalf("combining --only and --no- exited %d, want %d — the two express contradictory intents about the same set\n%s",
			code, ExitConfigError, output)
	}
	if !strings.Contains(output, "--only") || !strings.Contains(output, "--no-") {
		t.Fatalf("the error does not name both flags.\n%s", output)
	}
}

// TestInit_UnknownCapabilityNameIsAConfigErrorNamingTheValidSet (SPEC-069 CLM-013).
//
// The message LISTS the seven valid names. Seven names cost one line and answer the
// operator's actual question; a suggestion engine would be init guessing at intent.
func TestInit_UnknownCapabilityNameIsAConfigErrorNamingTheValidSet(t *testing.T) {
	output, code := runInitCommand(t, t.TempDir(), "--only", "not-a-capability")

	if code != ExitConfigError {
		t.Fatalf("an unrecognized capability exited %d, want %d\n%s", code, ExitConfigError, output)
	}
	for _, name := range []string{"git", "sdlc", "gitignore", "packs", "toolchain", "baseline", "observe"} {
		if !strings.Contains(output, name) {
			t.Fatalf("the error does not list the valid capability %q.\n%s", name, output)
		}
	}
}

// TestInit_LocalPathPackRefIsRefusedForLockPortability (SPEC-069 CLM-087).
//
// It names the lock-portability REASON and points at `backstop pack add` after init. A
// bare refusal would leave the consumer to guess why a path they can see on disk is not
// acceptable.
func TestInit_LocalPathPackRefIsRefusedForLockPortability(t *testing.T) {
	output, code := runInitCommand(t, t.TempDir(), "--pack", "./a-local-pack")

	if code != ExitConfigError {
		t.Fatalf("a local-path --pack exited %d, want %d\n%s", code, ExitConfigError, output)
	}
	lowered := strings.ToLower(output)
	if !strings.Contains(lowered, "local_path") && !strings.Contains(lowered, "portab") {
		t.Fatalf("the refusal does not give the lock-portability reason.\n%s", output)
	}
	if !strings.Contains(output, "backstop pack add") {
		t.Fatalf("the refusal does not point at `backstop pack add` after init.\n%s", output)
	}
}

// TestInit_LocalPathClassificationMatchesTheShippedClassifier (SPEC-069 CLM-110).
//
// EVERY form the shipped classifier accepts is refused by init, and every form it
// rejects is accepted — so the two CANNOT DISAGREE about any ref. A second definition
// of "is this a local path" drifting from the add path's is exactly what would produce
// the machine-specific lock entry the refusal exists to prevent.
// ★ IT IS HERMETIC. The REMOTE forms below are coordinates the insteadOf redirect
// points at a local repository, so accepting one resolves against that repository
// instead of reaching github.com. An earlier version used an invented `some-org/`
// coordinate, which made a real network clone on every run — the verdict was identical
// offline, so it was never a false green, but a suite that silently depends on the
// network is one that starts failing for reasons that have nothing to do with the claim.
func TestInit_LocalPathClassificationMatchesTheShippedClassifier(t *testing.T) {
	// One redirect, installed before any ref is exercised, covering both remote forms.
	const fixture = "acceptance-lint-pack"
	source, absErr := filepath.Abs(filepath.Join("testdata", "hermetic-remote", fixture))
	if absErr != nil {
		t.Fatalf("resolving the %s fixture: %v", fixture, absErr)
	}
	remote := newHermeticRemote(t, source, "v1.0.0")
	redirectPackURL(t, remoteE2EOrg, fixture, remote.Path)
	assertPackURLRedirected(t, remoteE2EOrg, fixture, remote)
	redirected := remoteE2EOrg + "/" + fixture

	refs := []string{
		"/opt/packs/some-pack",
		"./relative-pack",
		"../sibling-pack",
		filepath.Join(string(filepath.Separator), "var", "packs", "some-pack"),
		redirected + "@1.0.0",
		redirected,
	}

	sawLocal, sawRemote := false, false
	for _, ref := range refs {
		shippedSaysLocal := distribution.IsLocalPath(ref)
		if shippedSaysLocal {
			sawLocal = true
		} else {
			sawRemote = true
		}

		output, code := runInitCommand(t, t.TempDir(), "--pack", ref)
		initRefused := code == ExitConfigError && strings.Contains(output, "local filesystem path")

		if shippedSaysLocal != initRefused {
			t.Fatalf("init and the shipped classifier DISAGREE about %q: distribution.IsLocalPath = %v, init refused = %v.\nA ref one accepted and the other classified local is precisely the machine-specific lock entry REQ-018 exists to prevent.\n%s",
				ref, shippedSaysLocal, initRefused, output)
		}
	}

	// Both sides of the classifier must actually have been exercised, or "they agree"
	// is a statement about an empty half.
	if !sawLocal || !sawRemote {
		t.Fatalf("the ref table exercised local=%v remote=%v; it must cover BOTH verdicts or the agreement it asserts is vacuous", sawLocal, sawRemote)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// NO PROMPT, NO STDIN
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_HeadlessRunIsIdenticalToInteractiveRun (SPEC-069 CLM-014).
//
// With stdin CLOSED and no TTY the run produces the same files, the same report and the
// same exit code. Init does not prompt, so it must not depend on stdin being readable —
// and a command that blocks on a closed stdin is one nobody can put in CI.
func TestInit_HeadlessRunIsIdenticalToInteractiveRun(t *testing.T) {
	interactive := t.TempDir()
	interactiveOutput, interactiveCode := runInitCommand(t, interactive)

	headless := t.TempDir()
	closed, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("opening the null device: %v", err)
	}
	original := os.Stdin
	os.Stdin = closed
	if closeErr := closed.Close(); closeErr != nil {
		t.Fatalf("closing stdin: %v", closeErr)
	}
	headlessOutput, headlessCode := runInitCommand(t, headless)
	os.Stdin = original

	if headlessCode != interactiveCode {
		t.Fatalf("headless exited %d and interactive %d", headlessCode, interactiveCode)
	}

	// The reports differ only in the project path, which is the directory basename by
	// construction. Compare everything after the header line.
	normalize := func(output, root string) string {
		return strings.ReplaceAll(strings.ReplaceAll(output, root, "<project>"), filepath.Base(root), "<name>")
	}
	if normalize(headlessOutput, headless) != normalize(interactiveOutput, interactive) {
		t.Fatalf("the two reports differ.\nheadless:\n%s\ninteractive:\n%s", headlessOutput, interactiveOutput)
	}

	for _, path := range []string{"backstop.yml", ".gitignore", ".backstop/specs"} {
		if _, statErr := os.Stat(filepath.Join(headless, path)); statErr != nil {
			t.Fatalf("the headless run did not produce %s: %v", path, statErr)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// COMMAND TREE — DENYLIST
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_AddsNoCIVerbToTheCommandTree (SPEC-069 CLM-077).
//
// CI is governed SOLELY by `--ci` presence. A `ci` verb would give one outcome two
// entry points and two justifications.
func TestInit_AddsNoCIVerbToTheCommandTree(t *testing.T) {
	assertNoVerbAnywhere(t, "ci")
}

// TestInit_AddsNoScaffoldVerbOrNegationFlag (SPEC-069 CLM-133).
//
// No `--no-scaffold` flag and no `scaffold` verb or subcommand. `--scaffold` is a flag
// on init and nothing else — omission IS the opt-out, exactly as with `--ci`.
func TestInit_AddsNoScaffoldVerbOrNegationFlag(t *testing.T) {
	assertNoVerbAnywhere(t, "scaffold")

	initCmd := initCommandFrom(t)
	if initCmd.Flags().Lookup("no-scaffold") != nil {
		t.Fatal("init declares a --no-scaffold flag; the scaffold step is governed solely by the presence of --scaffold")
	}
	if initCmd.Flags().Lookup("scaffold") == nil {
		t.Fatal("init declares no --scaffold flag at all, so the absence above proves nothing")
	}
	// And no --no-ci either, for the same reason.
	if initCmd.Flags().Lookup("no-ci") != nil {
		t.Fatal("init declares a --no-ci flag; omission is the opt-out")
	}
}

// assertNoVerbAnywhere fails when a command by that name exists at ANY depth.
//
// The whole tree is walked rather than just init's children: a `ci` or `scaffold` verb
// parked under some other namespace would be exactly as much a second entry point.
func assertNoVerbAnywhere(t *testing.T, verb string) {
	t.Helper()

	var visit func(cmd *cobra.Command)
	visit = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.Name() == verb {
				t.Fatalf("the command tree carries a %q verb at %q; that step is governed solely by its flag on init", verb, child.CommandPath())
			}
			visit(child)
		}
	}
	visit(NewRootCommand())
}

// initCommandFrom returns the init command out of a freshly assembled tree, failing
// when it is not registered at all — so a flag-absence assertion cannot pass because
// the command is missing.
func initCommandFrom(t *testing.T) *cobra.Command {
	t.Helper()
	for _, cmd := range NewRootCommand().Commands() {
		if cmd.Name() == "init" {
			return cmd
		}
	}
	t.Fatal("the root command tree carries no init command")
	return nil
}

// TestInit_ExposesExactlySevenNegationFlags is additive: it pins the flag surface to
// the capability vocabulary, so an eighth negation cannot appear without an eighth
// capability existing first.
func TestInit_ExposesExactlySevenNegationFlags(t *testing.T) {
	initCmd := initCommandFrom(t)

	negations := 0
	initCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if strings.HasPrefix(flag.Name, "no-") {
			negations++
		}
	})
	if negations != 7 {
		t.Fatalf("init declares %d --no- flags, want exactly seven — one per capability", negations)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// EXIT 0
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_CleanGateAndAllStepsDeliveredExitZero (SPEC-069 CLM-067).
func TestInit_CleanGateAndAllStepsDeliveredExitZero(t *testing.T) {
	output, code := runInitCommand(t, t.TempDir())

	if code != ExitPass {
		t.Fatalf("a run whose every step delivered exited %d\n%s", code, output)
	}
}

// TestInit_PreExistingFindingsAloneExitZero (SPEC-069 CLM-066).
//
// Inherited violations are NOT an init failure. The project below carries a pack whose
// engine really dispatches, and whatever it notices is reported as observation.
func TestInit_PreExistingFindingsAloneExitZero(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "acceptance-lint-pack")

	output, code := runInitCommand(t, project, "--pack", ref)

	if code != ExitPass {
		t.Fatalf("pre-existing findings made init exit %d; inherited violations are never an init failure\n%s", code, output)
	}
	if !strings.Contains(output, "What the gate noticed") {
		t.Fatalf("the run did not report the gate's observations at all, so the exit code above proves nothing\n%s", output)
	}
}

// TestInit_CIOmittedIsADeliberateNoOpExitingZero (SPEC-069 CLM-076).
func TestInit_CIOmittedIsADeliberateNoOpExitingZero(t *testing.T) {
	output, code := runInitCommand(t, t.TempDir())

	if code != ExitPass {
		t.Fatalf("an omitted --ci exited %d; a skipped optional step is not an error\n%s", code, output)
	}
	if !strings.Contains(reportLineFor(t, output, "ci"), "skipped") {
		t.Fatalf("the CI step did not report a skip\n%s", output)
	}
}

// TestInit_ScaffoldOmittedIsADeliberateNoOpExitingZero (SPEC-069 CLM-129).
func TestInit_ScaffoldOmittedIsADeliberateNoOpExitingZero(t *testing.T) {
	output, code := runInitCommand(t, t.TempDir())

	if code != ExitPass {
		t.Fatalf("an omitted --scaffold exited %d; not every pack ecosystem ships a scaffold recipe\n%s", code, output)
	}
	if !strings.Contains(reportLineFor(t, output, "scaffold"), "skipped") {
		t.Fatalf("the scaffold step did not report a skip\n%s", output)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// EXIT NON-ZERO
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_EveryCIResolveFailureExitsNonZero (SPEC-069 CLM-084).
//
// All five shipped resolve failure modes, driven through the COMMAND. Each is a promise
// init was asked for and did not keep.
func TestInit_EveryCIResolveFailureExitsNonZero(t *testing.T) {
	cases := map[string]string{
		"unpinned ref":     "some-org/pack:some-recipe",
		"malformed ref":    "not-a-ref-at-all",
		"uninstalled pack": "never-installed/pack:some-recipe@1.0.0",
		"empty recipe id":  "some-org/pack:@1.0.0",
		"non-semver pin":   "some-org/pack:some-recipe@latest",
	}

	for name, ref := range cases {
		t.Run(name, func(t *testing.T) {
			output, code := runInitCommand(t, t.TempDir(), "--ci", ref)
			if code == ExitPass {
				t.Fatalf("--ci %q exited 0; CI was asked for and not delivered\n%s", ref, output)
			}
		})
	}
}

// TestInit_UnresolvableScaffoldRefExitsNonZero (SPEC-069 CLM-135).
//
// The consumer asked for a source file and init did not deliver one: a broken promise,
// not an un-adopted capability.
func TestInit_UnresolvableScaffoldRefExitsNonZero(t *testing.T) {
	output, code := runInitCommand(t, t.TempDir(), "--scaffold", "never-installed/pack:first-source@1.0.0")

	if code == ExitPass {
		t.Fatalf("an unresolvable --scaffold exited 0\n%s", output)
	}
	if !strings.Contains(output, "scaffold") {
		t.Fatalf("the failure is not attributed to the scaffold step\n%s", output)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// PRECEDENCE — BOTH FACTS REPORTED IN FULL
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_CIResolveFailureWinsOverObservationOnTheExitCode (SPEC-069 CLM-069).
//
// A run that triggers BOTH an observation and a CI resolve failure exits non-zero on
// the failed step — and reports BOTH facts in full.
func TestInit_CIResolveFailureWinsOverObservationOnTheExitCode(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "acceptance-lint-pack")

	output, code := runInitCommand(t, project, "--pack", ref, "--ci", "never-installed/pack:ci@1.0.0")

	if code == ExitPass {
		t.Fatalf("a CI resolve failure alongside an observation exited 0\n%s", output)
	}
	if !strings.Contains(output, "What the gate noticed") {
		t.Fatalf("the observation was dropped; BOTH facts are reported in full\n%s", output)
	}
	if !strings.Contains(reportLineFor(t, output, "ci"), "not delivered") {
		t.Fatalf("the CI step's own failure is not in the report\n%s", output)
	}
}

// TestInit_BrownfieldPreserveWinsOverObservationOnTheExitCode (SPEC-069 CLM-068).
//
// The same precedence, driven by a REAL preserve: the consumer's own file already sits
// at the recipe's declared target, so the apply SUCCEEDS while preserving it, and the
// resulting gap is what carries the exit code.
func TestInit_BrownfieldPreserveWinsOverObservationOnTheExitCode(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "scaffold-recipe-pack")
	packName := remoteE2EOrg + "/scaffold-recipe-pack"

	// The consumer's own file, already at the path the recipe declares.
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatalf("creating the consumer's source directory: %v", err)
	}
	consumersOwn := "the consumer's own file, which predates backstop\n"
	if err := os.WriteFile(filepath.Join(project, "src", "first-source.txt"), []byte(consumersOwn), 0o644); err != nil {
		t.Fatalf("writing the consumer's own file: %v", err)
	}

	output, code := runInitCommand(t, project, "--pack", ref, "--ci", packName+":first-source@1.0.0")

	if code == ExitPass {
		t.Fatalf("a user-owned brownfield preserve exited 0; it is a broken promise, not an un-adopted capability\n%s", output)
	}
	if !strings.Contains(output, "What the gate noticed") {
		t.Fatalf("the observation was dropped; BOTH facts are reported in full\n%s", output)
	}
	// And the consumer's file was NOT clobbered.
	body, err := os.ReadFile(filepath.Join(project, "src", "first-source.txt"))
	if err != nil {
		t.Fatalf("reading the consumer's file after the run: %v", err)
	}
	if string(body) != consumersOwn {
		t.Fatalf("init overwrote the consumer's own file.\nbefore: %q\nafter:  %q", consumersOwn, body)
	}
}

// TestInit_UserOwnedPreserveExitsNonZeroDespiteSuccessfulApply (SPEC-069 CLM-099).
//
// The underlying apply SUCCEEDED — nothing errored — and the broken promise is what the
// exit code carries. An implementation that derived the code from whether the apply
// returned an error would exit 0 here.
func TestInit_UserOwnedPreserveExitsNonZeroDespiteSuccessfulApply(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "scaffold-recipe-pack")
	packName := remoteE2EOrg + "/scaffold-recipe-pack"

	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatalf("creating the consumer's source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "src", "first-source.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("writing the consumer's own file: %v", err)
	}

	output, code := runInitCommand(t, project, "--pack", ref, "--scaffold", packName+":first-source@1.0.0")

	if code == ExitPass {
		t.Fatalf("a successful apply that preserved a user-owned file exited 0\n%s", output)
	}
	if !strings.Contains(output, "user-owned") {
		t.Fatalf("the report does not classify the preserve as user-owned\n%s", output)
	}
}

// TestInit_ScaffoldUserOwnedPreserveExitsNonZeroWithANextAction (SPEC-069 CLM-141).
func TestInit_ScaffoldUserOwnedPreserveExitsNonZeroWithANextAction(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "scaffold-recipe-pack")
	packName := remoteE2EOrg + "/scaffold-recipe-pack"

	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatalf("creating the consumer's source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "src", "first-source.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("writing the consumer's own file: %v", err)
	}

	output, code := runInitCommand(t, project, "--pack", ref, "--scaffold", packName+":first-source@1.0.0")

	if code == ExitPass {
		t.Fatalf("a user-owned scaffold preserve exited 0\n%s", output)
	}
	if !strings.Contains(output, "backstop recipe apply") {
		t.Fatalf("the scaffold gap gives the consumer no next action\n%s", output)
	}
	if !strings.Contains(strings.ToLower(output), "was not written") {
		t.Fatalf("the scaffold gap does not state that the recipe's declared source file was NOT written\n%s", output)
	}
}

// TestInit_ScaffoldTemplatingKindEmptyPairPreserveIsReportedAsIndeterminateGap
// (SPEC-069 CLM-142).
//
// The recipe's declared KIND comes from the PACK — the two fixture packs differ in
// exactly that one field — so the classifier's observable is pack data rather than
// something this test supplied.
func TestInit_ScaffoldTemplatingKindEmptyPairPreserveIsReportedAsIndeterminateGap(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "scaffold-templating-pack")
	packName := remoteE2EOrg + "/scaffold-templating-pack"

	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatalf("creating the consumer's source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "src", "first-source.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("writing the consumer's own file: %v", err)
	}

	output, code := runInitCommand(t, project, "--pack", ref, "--scaffold", packName+":first-source@1.0.0")

	if code == ExitPass {
		t.Fatalf("an indeterminate preserve exited 0; DD-15's refuse posture scores it conservatively as a gap\n%s", output)
	}
	if !strings.Contains(output, "indeterminate") {
		t.Fatalf("the preserve was not classified indeterminate\n%s", output)
	}
	if !strings.Contains(strings.ToLower(output), "cannot determine") {
		t.Fatalf("the report does not state that init cannot determine whether the recipe's output is present\n%s", output)
	}
	if strings.Contains(strings.ToLower(output), "no backstop gate was wired") {
		t.Fatalf("the indeterminate report asserts that no gate was wired — the half init cannot know\n%s", output)
	}
	if !strings.Contains(output, "backstop recipe apply") {
		t.Fatalf("the indeterminate report gives no next action\n%s", output)
	}
}

// TestInit_IndeterminatePreserveExitsNonZeroAndAssertsNoUnwiredGate (SPEC-069 CLM-114).
//
// BOTH halves asserted together: an implementation that exits 0 fails, and one that
// borrows the USER-OWNED sentence fails.
func TestInit_IndeterminatePreserveExitsNonZeroAndAssertsNoUnwiredGate(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "scaffold-templating-pack")
	packName := remoteE2EOrg + "/scaffold-templating-pack"

	if err := os.MkdirAll(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatalf("creating the consumer's source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "src", "first-source.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("writing the consumer's own file: %v", err)
	}

	output, code := runInitCommand(t, project, "--pack", ref, "--ci", packName+":first-source@1.0.0")

	if code == ExitPass {
		t.Fatalf("an indeterminate CI preserve exited 0\n%s", output)
	}
	if strings.Contains(strings.ToLower(output), "no backstop gate was wired") {
		t.Fatalf("the indeterminate report borrowed the USER-OWNED sentence, asserting something about a one-shot that may already have materialized\n%s", output)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// THE ACCOUNTABLE CLASS — EXIT 0
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_WaiverCoveredPreserveExitsZeroAndClaimsNoUnwiredGate (SPEC-069 CLM-116).
// TestInit_ApplyWithOnlyWaiverCoveredPreservesIsSuccessAndExitsZero (CLM-117).
// TestInit_PopulatedWaiverPairIsWaiverCoveredAtEveryRecipeKind (CLM-125).
// TestInit_ScaffoldWaiverCoveredPreserveIsSuccessAndExitsZero (CLM-143).
//
// These four share one setup: a recipe-owned file the consumer legitimately customized
// and ACCOUNTED FOR with a valid waiver token. The gate IS wired, the customization is
// accountable, and the run therefore exits 0 with no gap.
//
// Reaching that state through the REAL applier requires the recipe to have been adopted
// on a PRIOR apply and then diverged, so each test applies twice: once to adopt and
// materialize, then edits the file with a covering token, then applies again.
func TestInit_WaiverCoveredPreserveExitsZeroAndClaimsNoUnwiredGate(t *testing.T) {
	output, code := runWaiverCoveredScenario(t, "scaffold-recipe-pack", "--ci")

	if code != ExitPass {
		t.Fatalf("a waiver-covered preserve exited %d; the accountable class is the one preserve class that leaves an apply successful\n%s", code, output)
	}
	if strings.Contains(strings.ToLower(output), "no backstop gate was wired") {
		t.Fatalf("the waiver-covered report claims no gate was wired; the gate demonstrably IS wired and the divergence is accounted for\n%s", output)
	}
}

// TestInit_ApplyWithOnlyWaiverCoveredPreservesIsSuccessAndExitsZero (SPEC-069 CLM-117).
func TestInit_ApplyWithOnlyWaiverCoveredPreservesIsSuccessAndExitsZero(t *testing.T) {
	output, code := runWaiverCoveredScenario(t, "scaffold-recipe-pack", "--ci")

	if code != ExitPass {
		t.Fatalf("an apply whose every preserve is waiver-covered exited %d\n%s", code, output)
	}
	if !strings.Contains(output, "waiver-covered") {
		t.Fatalf("the preserve was not classified waiver-covered, so the exit code above proves nothing\n%s", output)
	}
}

// TestInit_PopulatedWaiverPairIsWaiverCoveredAtEveryRecipeKind (SPEC-069 CLM-125).
//
// THE PAIR OUTRANKS THE KIND, `templating` INCLUDED. The kind is consulted ONLY to
// split the empty-pair case, so an implementation that tested the kind FIRST would
// misclassify a populated pair at `templating` as indeterminate and exit non-zero.
//
// ★ ONE HALF OF THIS CLAIM IS UNREACHABLE THROUGH THE SHIPPED APPLIER, AND THAT IS A
// FACT ABOUT pkg/recipe RATHER THAN A GAP IN INIT. preserveOrRegenerate returns its
// `own.kind == KindTemplating` branch "without reading the file or consulting the
// waiver seam" (pkg/recipe/apply.go), so it emits an EMPTY Rule/CoveringWaiver pair at
// that kind ALWAYS. A populated pair at `templating` therefore cannot arise from a real
// apply, however the fixture is arranged — which is verified below rather than assumed,
// because "the combination is unreachable" is exactly the sort of claim that quietly
// stops being true.
//
// So the claim is asserted in the two places it is actually available:
//
//	REACHABLE HALF — a populated pair from a REAL apply at a real declared kind is
//	classified waiver-covered and exits 0, end to end through the command.
//
//	ORDERING HALF — the same REAL pair, carried at the kind the TEMPLATING fixture pack
//	declares. Both observables still come from packs; only their COMBINATION is composed
//	here, and it is composed precisely because the applier cannot produce it.
func TestInit_PopulatedWaiverPairIsWaiverCoveredAtEveryRecipeKind(t *testing.T) {
	t.Run("reachable through a real apply", func(t *testing.T) {
		output, code := runWaiverCoveredScenario(t, "scaffold-recipe-pack", "--ci")
		if code != ExitPass {
			t.Fatalf("a populated waiver pair from a real apply exited %d\n%s", code, output)
		}
		if !strings.Contains(output, "waiver-covered") {
			t.Fatalf("the real populated pair was not classified waiver-covered\n%s", output)
		}
	})

	t.Run("the templating combination is unreachable from the applier", func(t *testing.T) {
		// The FALSIFIER for the note above. The file on disk carries a valid covering
		// waiver, and the templating apply STILL returns an empty pair — proving the
		// class init reports follows the pair the APPLIER returned rather than anything
		// init sniffed for itself, and proving the combination below is composed for a
		// real reason.
		output, code := runWaiverCoveredScenario(t, "scaffold-templating-pack", "--ci")
		if code == ExitPass {
			t.Fatalf("the templating apply reported an accountable customization; the shipped applier never adjudicates at that kind, so it cannot have\n%s", output)
		}
		if !strings.Contains(output, "indeterminate") {
			t.Fatalf("the templating apply's EMPTY pair was not classified indeterminate, so the reachability fact this claim rests on no longer holds\n%s", output)
		}
	})

	t.Run("the pair outranks the kind at the templating kind", func(t *testing.T) {
		// BOTH pack-derived observables are read BEFORE anything changes the working
		// directory: capturedRealWaiverPair drives a real run, which chdirs into a temp
		// project, and a relative fixture path read afterwards resolves against that.
		templatingKind := declaredRecipeKind(t, "scaffold-templating-pack")
		realPair := capturedRealWaiverPair(t)

		if templatingKind != "templating" {
			t.Fatalf("the templating fixture declares kind %q; this sub-test needs the kind that makes the ordering load-bearing", templatingKind)
		}

		project := t.TempDir()
		output, code := runInitWithReplayedApply(t, project, initialize.ApplyOutcome{
			Preserved:  []recipe.PreservedDivergence{realPair},
			RecipeKind: templatingKind,
		})

		if code != ExitPass {
			t.Fatalf("a POPULATED pair at kind %q exited %d; the pair outranks the kind at EVERY kind, and an implementation that tested the kind first fails exactly here\n%s",
				templatingKind, code, output)
		}
		if !strings.Contains(output, "waiver-covered") {
			t.Fatalf("a populated pair at kind %q was not classified waiver-covered\n%s", templatingKind, output)
		}
		if strings.Contains(strings.ToLower(output), "cannot determine") {
			t.Fatalf("a populated pair at kind %q was reported as indeterminate; the kind is consulted ONLY to split the EMPTY-pair case\n%s", templatingKind, output)
		}
		if strings.Contains(strings.ToLower(output), "no backstop gate was wired") {
			t.Fatalf("an accountable customization was reported as an unwired gate\n%s", output)
		}
	})
}

// replayedApplier hands the runner an outcome CAPTURED from a real apply.
//
// It exists for exactly one combination — a populated Rule/CoveringWaiver pair at the
// `templating` kind — which the shipped applier cannot produce because it returns that
// kind's branch before it ever adjudicates a divergence. Both observables it carries
// still come from packs; only their pairing is composed. Every other assertion in this
// file drives the real adapter.
type replayedApplier struct{ outcome initialize.ApplyOutcome }

func (r replayedApplier) Apply(projectRoot string, ref string) (initialize.ApplyOutcome, error) {
	return r.outcome, nil
}

// runInitWithReplayedApply runs the real runner with every production adapter EXCEPT
// the applier, which replays the supplied outcome.
func runInitWithReplayedApply(t *testing.T, project string, outcome initialize.ApplyOutcome) (string, int) {
	t.Helper()
	t.Chdir(project)

	runner, err := initialize.NewRunner(
		initPackInstaller{},
		replayedApplier{outcome: outcome},
		initGateRunner{},
		&packToolchainProber{Runner: &check.ExecCommandRunner{Dir: project}},
		unavailableBaselineSeeder{},
	)
	if err != nil {
		t.Fatalf("assembling the runner: %v", err)
	}

	capabilities, resolveErr := initialize.ResolveCapabilities(nil, nil)
	if resolveErr != nil {
		t.Fatalf("resolving capabilities: %v", resolveErr)
	}

	result, runErr := runner.Run(initialize.Options{
		ProjectRoot:  project,
		Capabilities: capabilities,
		CIRecipeRef:  "some-org/some-pack:some-recipe@1.0.0",
	})
	if runErr != nil {
		t.Fatalf("the run errored: %v", runErr)
	}

	var rendered strings.Builder
	for _, step := range result.Steps {
		rendered.WriteString(step.Step + " " + step.Outcome.String() + " " + step.Detail + "\n")
	}
	for _, preserve := range result.Preserved {
		rendered.WriteString(preserve.Path + " " + preserve.Class.String() + "\n")
	}
	if result.BrokenPromise {
		return rendered.String(), ExitViolations
	}
	return rendered.String(), ExitPass
}

// capturedRealWaiverPair drives a REAL apply to the waiver-covered state and returns
// the PreservedDivergence the shipped applier produced, so the pair replayed above is
// machinery-produced rather than invented.
func capturedRealWaiverPair(t *testing.T) recipe.PreservedDivergence {
	t.Helper()

	ref, project := initCmdHermeticPack(t, "scaffold-recipe-pack")
	packName := remoteE2EOrg + "/scaffold-recipe-pack"
	recipeRef := packName + ":first-source@1.0.0"

	if _, code := runInitCommand(t, project, "--pack", ref, "--ci", recipeRef); code != ExitPass {
		t.Fatal("the adopting run did not succeed")
	}

	target := filepath.Join(project, "src", "first-source.txt")
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading the materialized target: %v", err)
	}
	covering := "@" + "waiver:recipe.first-source.present:accepted-risk:2029-01-01"
	if writeErr := os.WriteFile(target, []byte(string(body)+"\n"+covering+" deliberate customization\n"), 0o644); writeErr != nil {
		t.Fatalf("writing the diverged target: %v", writeErr)
	}

	outcome, applyErr := initRecipeApplier{}.Apply(project, recipeRef)
	if applyErr != nil {
		t.Fatalf("the capturing apply failed: %v", applyErr)
	}
	if len(outcome.Preserved) != 1 {
		t.Fatalf("the capturing apply returned %d preserves, want 1", len(outcome.Preserved))
	}
	captured := outcome.Preserved[0]
	if captured.Rule == "" || captured.CoveringWaiver == "" {
		t.Fatalf("the captured pair is not POPULATED (%+v); the replay below would assert nothing", captured)
	}
	return captured
}

// declaredRecipeKind reads a fixture pack's declared recipe kind off its own
// recipe.yml, so the kind under test is PACK DATA rather than a literal here.
func declaredRecipeKind(t *testing.T, fixture string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "hermetic-remote", fixture, "recipes", "first-source", "recipe.yml"))
	if err != nil {
		t.Fatalf("reading the %s recipe manifest: %v", fixture, err)
	}
	manifest, parseErr := recipe.ParseRecipeManifest(body)
	if parseErr != nil {
		t.Fatalf("parsing the %s recipe manifest: %v", fixture, parseErr)
	}
	return manifest.Kind
}

// TestInit_ScaffoldWaiverCoveredPreserveIsSuccessAndExitsZero (SPEC-069 CLM-143).
func TestInit_ScaffoldWaiverCoveredPreserveIsSuccessAndExitsZero(t *testing.T) {
	output, code := runWaiverCoveredScenario(t, "scaffold-recipe-pack", "--scaffold")

	if code != ExitPass {
		t.Fatalf("a waiver-covered SCAFFOLD preserve exited %d\n%s", code, output)
	}
	if !strings.Contains(output, "waiver-covered") {
		t.Fatalf("the scaffold preserve was not classified waiver-covered\n%s", output)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// NOT REGRESSING THE SCOPE DEFAULT
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_PostInitGateWithNoScopeFlagsIsDiffScoped (SPEC-069 CLM-070).
//
// REQ-015 is satisfied by NOT REGRESSING the shipped default, so the assertion is that
// a `backstop gate` run with no scope flags in a project init produced still resolves
// the diff scope — and that the generated config wrote nothing that could change it.
func TestInit_PostInitGateWithNoScopeFlagsIsDiffScoped(t *testing.T) {
	project := t.TempDir()
	if _, code := runInitCommand(t, project); code != ExitPass {
		t.Fatalf("the init run exited %d", code)
	}

	// ★ THE SCOPE IS READ OFF A REAL `backstop gate` RUN WITH NO SCOPE FLAGS, NOT
	// COMPUTED HERE.
	//
	// An earlier version of this test called ComputeGateScope(project,
	// GateScopeModeDiff, nil) and asserted the mode came back as diff — it HANDED
	// ITSELF THE ANSWER. Flipping runGate's own default from GateScopeModeDiff to
	// GateScopeModeAll left it passing, which is exactly the regression REQ-015 exists
	// to catch: this requirement is satisfied by NOT REGRESSING the shipped default, so
	// the only assertion worth anything is one that reads what the SHIPPED COMMAND
	// resolves when the operator supplies no scope flag.
	mode := gateScopeModeFromJSONRun(t, project)
	if mode != string(gate.GateScopeModeDiff) {
		t.Fatalf("a `backstop gate` run with no scope flags resolved scope mode %q, want %q. Init must write no configuration that changes, overrides or pins the gate's default scope.",
			mode, gate.GateScopeModeDiff)
	}

	// And the config init wrote carries nothing that could override it.
	loaded, loadErr := config.LoadConfigFromPath(filepath.Join(project, "backstop.yml"))
	if loadErr != nil {
		t.Fatalf("loading the generated config: %v", loadErr)
	}
	if loaded.Enforcement.TestCommand != "" || len(loaded.Enforcement.Toolchain) > 0 {
		t.Fatal("the generated config wrote enforcement keys init has no business setting")
	}
}

// gateScopeModeFromJSONRun runs the SHIPPED `backstop gate --json` with NO scope flags
// in project and returns the scope mode the command itself resolved.
//
// Reading the mode out of the emitted result is what makes the assertion about the
// BINARY's default rather than about a value the test chose: the scope arrives through
// runGate's own flag handling, so flipping that default reds this test.
func gateScopeModeFromJSONRun(t *testing.T, project string) string {
	t.Helper()
	t.Chdir(project)

	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"gate", "--json"})

	// The VERDICT is irrelevant here and is deliberately not asserted — a freshly
	// initialized project may legitimately carry advisories. What is being read is the
	// scope the command resolved, which it reports either way.
	_ = root.Execute()

	var result struct {
		Scope struct {
			Mode string `json:"mode"`
		} `json:"scope"`
	}
	body := out.Bytes()
	start := bytes.IndexByte(body, '{')
	if start < 0 {
		t.Fatalf("`gate --json` emitted no JSON object:\n%s", body)
	}
	if err := json.Unmarshal(body[start:], &result); err != nil {
		t.Fatalf("parsing the gate's JSON result: %v\n%s", err, body)
	}
	if result.Scope.Mode == "" {
		t.Fatalf("the gate's JSON result carries no scope mode, so this assertion would be vacuous:\n%s", body)
	}
	return result.Scope.Mode
}

// runWaiverCoveredScenario drives a REAL apply to the WAIVER-COVERED state.
//
// The state is only reachable through prior ADOPTION: preserveOrRegenerate returns the
// user-owned branch for any recipe no apply ever adopted, so the file must first be
// materialized BY the recipe, then diverged with a covering token, and only then applied
// again. Hand-building the state would make the classifier's discriminator test-supplied
// rather than machinery-produced.
func runWaiverCoveredScenario(t *testing.T, fixture, flag string) (string, int) {
	t.Helper()

	ref, project := initCmdHermeticPack(t, fixture)
	packName := remoteE2EOrg + "/" + fixture
	recipeRef := packName + ":first-source@1.0.0"

	// FIRST run: installs the pack and materializes the recipe's declared target, which
	// is what records the adoption.
	if _, code := runInitCommand(t, project, "--pack", ref, flag, recipeRef); code != ExitPass {
		t.Fatalf("the adopting run did not succeed; the waiver-covered state is unreachable without it")
	}

	target := filepath.Join(project, "src", "first-source.txt")
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading the materialized target: %v", err)
	}

	// Diverge it, and ACCOUNT for the divergence with a covering waiver token. The
	// token is assembled rather than written as a literal: the gate byte-scans source
	// for the waiver grammar, and this file is source.
	rule := "recipe.first-source.present"
	covering := "@" + "waiver:" + rule + ":accepted-risk:2029-01-01"
	diverged := string(body) + "\n" + covering + " the consumer's deliberate customization\n"
	if writeErr := os.WriteFile(target, []byte(diverged), 0o644); writeErr != nil {
		t.Fatalf("writing the diverged target: %v", writeErr)
	}

	// SECOND run: the apply now finds recipe-owned output that diverged, and adjudicates
	// the divergence as covered.
	return runInitCommand(t, project, flag, recipeRef)
}

// TestInit_EveryFlagLookupFailureIsAConfigErrorCarryingNoOptions covers the flag
// translation layer's refusal arms — one per lookup, exhaustively.
//
// initOptionsFromFlags is the ONLY place the command line becomes an initialize.Options,
// and its whole contract is that nothing has been written when it refuses. A lookup that
// fails must therefore produce (a) the ZERO Options — never a half-filled one a caller
// could act on — and (b) a *check.ConfigError, which is what maps to exit 2, the code
// that says "change the invocation" rather than "a promise was broken".
//
// The failure is induced by handing it a command missing exactly ONE of the flags it
// reads, walking the reads in order, so every arm is exercised and none is reachable only
// through the arm before it.
func TestInit_EveryFlagLookupFailureIsAConfigErrorCarryingNoOptions(t *testing.T) {
	defineOnly := func(flags *pflag.FlagSet) { flags.StringArray("only", nil, "") }
	definePack := func(flags *pflag.FlagSet) { flags.StringArray("pack", nil, "") }
	defineCI := func(flags *pflag.FlagSet) { flags.String("ci", "", "") }
	defineScaffold := func(flags *pflag.FlagSet) { flags.String("scaffold", "", "") }

	// The first negation flag the capability loop reads. Derived from the vocabulary,
	// never spelled as a literal, so an eighth capability cannot silently change which
	// flag this case is about.
	firstNegation := "no-" + string(initialize.DefaultCapabilities()[0])

	cases := []struct {
		name        string
		define      []func(*pflag.FlagSet)
		missingFlag string
	}{
		{"only", nil, "only"},
		{"pack", []func(*pflag.FlagSet){defineOnly}, "pack"},
		{"ci", []func(*pflag.FlagSet){defineOnly, definePack}, "ci"},
		{"scaffold", []func(*pflag.FlagSet){defineOnly, definePack, defineCI}, "scaffold"},
		{firstNegation, []func(*pflag.FlagSet){defineOnly, definePack, defineCI, defineScaffold}, firstNegation},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "init"}
			for _, define := range tc.define {
				define(cmd.Flags())
			}

			options, err := initOptionsFromFlags(cmd)

			if err == nil {
				t.Fatalf("initOptionsFromFlags accepted a command with no %q flag; an unreadable flag must refuse, not fall through to a default", tc.missingFlag)
			}
			var configErr *check.ConfigError
			if !errors.As(err, &configErr) {
				t.Fatalf("the refusal is %T, want a *check.ConfigError: nothing has been written yet, so this is exit 2 (fix the invocation), not a broken promise.\ngot: %v", err, err)
			}
			if !strings.Contains(err.Error(), tc.missingFlag) {
				t.Fatalf("the refusal does not name the flag it could not read.\nwant to contain: %s\ngot:             %s", tc.missingFlag, err.Error())
			}
			if options.ProjectRoot != "" || options.Capabilities != nil || options.PackRefs != nil ||
				options.CIRecipeRef != "" || options.ScaffoldRecipeRef != "" {
				t.Fatalf("a refusal returned a populated Options (%+v); a caller must not be handed a half-translated invocation", options)
			}
		})
	}
}
