package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/initialize"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// These are cmd/backstop tests, and that placement is FORCED: checkEngineToolAllowed
// and splitCommand are unexported in package main, so the prober cannot live in
// pkg/initialize without a second copy of both.
//
// They are written against INIT'S ADAPTER (packToolchainProber, the spec-mandated
// ToolchainProber) and therefore drive the SHARED packEntrypointProber underneath it
// end to end: selection, the trust gate, the splitter, the runner and the outcome split
// are all exercised through these fifteen. They are deliberately NOT split into
// "shared" and "init" suites — the mandated names are the spec's and the claim subjects
// are cmd/backstop.

// toolchainFixtureDir is the directory the fixture manifests' relative commands resolve
// against. The runner's Dir is what a relative declared command is resolved from, so
// rooting it here is what makes `bin/exit-zero.sh` runnable.
const toolchainFixtureDir = "testdata/init-toolchain"

// recordingRunner wraps a real runner and records every command it was asked to run,
// so a selection claim can assert on WHAT RAN rather than on what was reported.
type recordingRunner struct {
	inner check.CommandRunner
	// commands holds one entry per Run call, as "<name> <args...>".
	commands []string
	// stdoutCalls counts RunStdout invocations. It must stay ZERO: the shared prober
	// enters through Run, and a non-zero count here means the capture method drifted.
	stdoutCalls int
}

func (r *recordingRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	return r.inner.Run(ctx, name, args...)
}

func (r *recordingRunner) RunStdout(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.stdoutCalls++
	r.commands = append(r.commands, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	return r.inner.RunStdout(ctx, name, args...)
}

// toolchainFixture parses one of the init-toolchain manifest fixtures through the
// SHIPPED pack.ParseManifest, so every claim below is asserted against the pack's OWN
// declared YAML shape rather than a fabricated Go struct.
func toolchainFixture(t *testing.T, name string) *pack.Manifest {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(toolchainFixtureDir, name))
	if err != nil {
		t.Fatalf("reading the toolchain fixture %s: %v", name, err)
	}
	manifest, parseErr := pack.ParseManifest(body)
	if parseErr != nil {
		t.Fatalf("parsing the toolchain fixture %s: %v", name, parseErr)
	}
	return manifest
}

// probeFixture runs init's adapter over one fixture manifest and returns the reports,
// the recording runner, and the error.
//
// The error is LAST, which is the Go convention and what staticcheck's ST1008 pins.
func probeFixture(t *testing.T, name string) ([]initialize.StepReport, *recordingRunner, error) {
	t.Helper()
	runner := &recordingRunner{inner: &check.ExecCommandRunner{Dir: toolchainFixtureDir}}
	prober := &packToolchainProber{
		Packs:  []*pack.Manifest{toolchainFixture(t, name)},
		Runner: runner,
	}
	reports, err := prober.Probe(toolchainFixtureDir)
	return reports, runner, err
}

// ranCommandContaining reports whether any recorded command carries the marker.
func ranCommandContaining(runner *recordingRunner, marker string) bool {
	for _, command := range runner.commands {
		if strings.Contains(command, marker) {
			return true
		}
	}
	return false
}

// TestInit_ExecutesDeclaredTestEntrypointOnceAndPassesOnZeroExit (SPEC-069 CLM-053).
func TestInit_ExecutesDeclaredTestEntrypointOnceAndPassesOnZeroExit(t *testing.T) {
	reports, runner, err := probeFixture(t, "gatetype-matrix-pack.yml")
	if err != nil {
		t.Fatalf("probing the gate-type matrix errored: %v", err)
	}

	executions := 0
	for _, command := range runner.commands {
		if strings.Contains(command, "matrix-test") {
			executions++
		}
	}
	if executions != 1 {
		t.Fatalf("the declared TEST entrypoint ran %d times, want exactly once.\nran: %v", executions, runner.commands)
	}

	found := false
	for _, report := range reports {
		if strings.Contains(report.Detail, "matrix-test") {
			found = true
			if report.Outcome != initialize.OutcomeDelivered {
				t.Fatalf("a zero-exit test entrypoint reported %v, want OutcomeDelivered: %s", report.Outcome, report.Detail)
			}
		}
	}
	if !found {
		t.Fatalf("no report names the executed test entrypoint.\nreports: %v", reports)
	}
}

// TestInit_ExecutesDeclaredBuildEntrypointOnceAndPassesOnZeroExit (SPEC-069 CLM-054).
func TestInit_ExecutesDeclaredBuildEntrypointOnceAndPassesOnZeroExit(t *testing.T) {
	reports, runner, err := probeFixture(t, "gatetype-matrix-pack.yml")
	if err != nil {
		t.Fatalf("probing the gate-type matrix errored: %v", err)
	}

	executions := 0
	for _, command := range runner.commands {
		if strings.Contains(command, "matrix-build") {
			executions++
		}
	}
	if executions != 1 {
		t.Fatalf("the declared BUILD entrypoint ran %d times, want exactly once.\nran: %v", executions, runner.commands)
	}

	for _, report := range reports {
		if strings.Contains(report.Detail, "matrix-build") && report.Outcome != initialize.OutcomeDelivered {
			t.Fatalf("a zero-exit build entrypoint reported %v: %s", report.Outcome, report.Detail)
		}
	}
}

// assertGateTypeNotExecuted is the body the five non-entrypoint gate-type claims share.
//
// They are FIVE claims and five tests rather than one table, deliberately: "only test
// and build" asserted in aggregate hides a member, and each of the five is a gate type
// a future reader might plausibly decide belongs in the toolchain step.
func assertGateTypeNotExecuted(t *testing.T, marker string) {
	t.Helper()
	_, runner, err := probeFixture(t, "gatetype-matrix-pack.yml")
	if err != nil {
		t.Fatalf("probing the gate-type matrix errored: %v", err)
	}

	if ranCommandContaining(runner, marker) {
		t.Fatalf("the %s engine WAS executed by init's toolchain step. Selection is by declared gate_type — test and build ONLY — because those two name a STAGE of backstop's own kill chain; every other stage has its own dispatch.\nran: %v",
			marker, runner.commands)
	}
	// The falsifying half: the two that SHOULD run did, so the assertion above is not
	// passing because nothing ran at all.
	if !ranCommandContaining(runner, "matrix-test") || !ranCommandContaining(runner, "matrix-build") {
		t.Fatalf("neither entrypoint ran, so %q not running proves nothing.\nran: %v", marker, runner.commands)
	}
}

// TestInit_LintGateTypeEngineIsNotExecuted (SPEC-069 CLM-055).
func TestInit_LintGateTypeEngineIsNotExecuted(t *testing.T) {
	assertGateTypeNotExecuted(t, "matrix-lint")
}

// TestInit_FindingsGateTypeEngineIsNotExecuted (SPEC-069 CLM-118).
func TestInit_FindingsGateTypeEngineIsNotExecuted(t *testing.T) {
	assertGateTypeNotExecuted(t, "matrix-findings")
}

// TestInit_CoverageGateTypeEngineIsNotExecuted (SPEC-069 CLM-119).
func TestInit_CoverageGateTypeEngineIsNotExecuted(t *testing.T) {
	assertGateTypeNotExecuted(t, "matrix-coverage")
}

// TestInit_SubstantivenessGateTypeEngineIsNotExecuted (SPEC-069 CLM-120).
func TestInit_SubstantivenessGateTypeEngineIsNotExecuted(t *testing.T) {
	assertGateTypeNotExecuted(t, "matrix-substantiveness")
}

// TestInit_ContractsGateTypeEngineIsNotExecuted (SPEC-069 CLM-121).
func TestInit_ContractsGateTypeEngineIsNotExecuted(t *testing.T) {
	assertGateTypeNotExecuted(t, "matrix-contracts")
}

// reportFor returns the report whose detail names the engine's command marker.
func reportFor(t *testing.T, reports []initialize.StepReport, marker string) initialize.StepReport {
	t.Helper()
	for _, report := range reports {
		if strings.Contains(report.Detail, marker) {
			return report
		}
	}
	t.Fatalf("no report names %q.\nreports: %v", marker, reports)
	return initialize.StepReport{}
}

// TestInit_UnstartableEntrypointIsReportedAsOwedSetupAndExitsNonZero (SPEC-069
// CLM-056).
//
// Case (b): the declared executable CANNOT BE STARTED AT ALL. It is reported as a SETUP
// step the consumer still owes, naming the pack whose entrypoint could not run and
// pointing at THAT PACK's own documented install steps — inventing no install command
// and installing nothing.
func TestInit_UnstartableEntrypointIsReportedAsOwedSetupAndExitsNonZero(t *testing.T) {
	reports, _, err := probeFixture(t, "entrypoint-cases-pack.yml")
	if err != nil {
		t.Fatalf("probing the entrypoint cases errored: %v", err)
	}

	report := reportFor(t, reports, "no-such-entrypoint.sh")

	// The exit half, asserted on the Outcome the command's exit mapping reads.
	if report.Outcome != initialize.OutcomeBrokenPromise {
		t.Fatalf("an unstartable entrypoint reported %v, want OutcomeBrokenPromise: init promised to verify the toolchain RUNS and did not verify it", report.Outcome)
	}

	detail := strings.ToLower(report.Detail)
	if !strings.Contains(detail, "could not be started") {
		t.Fatalf("the report does not say the executable could not be STARTED, which is the whole distinction from a non-zero exit.\ngot: %s", report.Detail)
	}
	if !strings.Contains(detail, "setup you still owe") {
		t.Fatalf("the report does not label this as setup the consumer still owes.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, "hermetic/entrypoint-cases-pack") {
		t.Fatalf("the report does not name the pack whose entrypoint could not run.\ngot: %s", report.Detail)
	}
	if !strings.Contains(detail, "install") {
		t.Fatalf("the report does not point at the pack's own documented install steps.\ngot: %s", report.Detail)
	}
}

// TestInit_NonZeroEntrypointExitIsReportedVerbatimWithItsExitCode (SPEC-069 CLM-105).
//
// Case (c): the command STARTED and exited non-zero. The exit code plus the captured
// output VERBATIM, attributed to the pack and the command.
func TestInit_NonZeroEntrypointExitIsReportedVerbatimWithItsExitCode(t *testing.T) {
	reports, _, err := probeFixture(t, "entrypoint-cases-pack.yml")
	if err != nil {
		t.Fatalf("probing the entrypoint cases errored: %v", err)
	}

	report := reportFor(t, reports, "exit-nonzero-stdout.sh")

	if report.Outcome != initialize.OutcomeBrokenPromise {
		t.Fatalf("a non-zero entrypoint exit reported %v, want OutcomeBrokenPromise", report.Outcome)
	}
	if !strings.Contains(report.Detail, "exited 3") {
		t.Fatalf("the report does not carry the entrypoint's own exit code.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, "entrypoint failed: 3 checks did not pass") {
		t.Fatalf("the report does not carry the entrypoint's captured output VERBATIM.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, "hermetic/entrypoint-cases-pack") {
		t.Fatalf("the report does not attribute the failure to the pack.\ngot: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, "exit-nonzero-stdout.sh") {
		t.Fatalf("the report does not attribute the failure to the command.\ngot: %s", report.Detail)
	}
}

// TestInit_CapturedOutputIncludesStderrOnlyDiagnostics (SPEC-069 CLM-122).
//
// ★★ THE CAPTURE-METHOD TRIPWIRE, AND IT IS INVISIBLE IN A DIFF REVIEW.
//
// The fixture's stdout is EMPTY and its whole diagnostic is on stderr. An
// implementation that copied runFindingsEngine's three steps AND its method choice
// passes every structural claim in this file — the allowlist is bound, no shell is
// invoked, no second execution path exists, the exit code is right — and ONLY this
// claim goes red.
//
// The test asserts the fixture's stdout really is empty before asserting anything
// else: a fixture that also wrote to stdout would pass under BOTH capture methods and
// this claim would go vacuous while looking exactly like it is working.
func TestInit_CapturedOutputIncludesStderrOnlyDiagnostics(t *testing.T) {
	// FIRST, prove the fixture is the shape the claim needs.
	direct := &check.ExecCommandRunner{Dir: toolchainFixtureDir}
	stdoutOnly, _ := direct.RunStdout(context.Background(), "bin/exit-nonzero-stderr-only.sh")
	if len(stdoutOnly) != 0 {
		t.Fatalf("the CLM-122 fixture wrote %d bytes to stdout (%q). It must write NOTHING to stdout, or it passes under both Run and RunStdout and this claim asserts nothing.",
			len(stdoutOnly), stdoutOnly)
	}

	reports, runner, err := probeFixture(t, "entrypoint-cases-pack.yml")
	if err != nil {
		t.Fatalf("probing the entrypoint cases errored: %v", err)
	}
	if runner.stdoutCalls != 0 {
		t.Fatalf("the prober called RunStdout %d times; the entrypoint path enters the runner through Run (combined stdout+stderr) precisely so a stderr-only diagnostic is not lost", runner.stdoutCalls)
	}

	report := reportFor(t, reports, "exit-nonzero-stderr-only.sh")
	if !strings.Contains(report.Detail, "dependency resolution error (diagnostic on stderr only)") {
		t.Fatalf("the captured-output report is missing the entrypoint's STDERR diagnostic. Binding RunStdout instead of Run prints an EMPTY 'captured output verbatim' for exactly the failures this case exists to surface.\ngot: %s", report.Detail)
	}
}

// TestInit_NonZeroEntrypointExitClaimsNoCause (SPEC-069 CLM-106, denylist).
//
// A case-(c) report does NOT use the owed-setup label, does NOT name dependencies or a
// package manager, and attributes the failure to nothing beyond the pack and the
// command. Treating any non-zero exit as owed setup commits the exact exit-code
// cause-inference REQ-011 forbids.
func TestInit_NonZeroEntrypointExitClaimsNoCause(t *testing.T) {
	reports, _, err := probeFixture(t, "entrypoint-cases-pack.yml")
	if err != nil {
		t.Fatalf("probing the entrypoint cases errored: %v", err)
	}

	for _, marker := range []string{"exit-nonzero-stdout.sh", "exit-nonzero-stderr-only.sh"} {
		report := reportFor(t, reports, marker)
		detail := strings.ToLower(report.Detail)

		if strings.Contains(detail, "setup you still owe") || strings.Contains(detail, "owed setup") {
			t.Fatalf("%s: a started-and-exited-non-zero entrypoint was labelled owed setup. Init cannot tell WHY it exited non-zero, and claiming it can is the exit-code cause-inference REQ-011 exists to forbid.\ngot: %s", marker, report.Detail)
		}
		// The nouns live in this TEST, not in the source set, so naming them here is an
		// assertion rather than a bake.
		for _, cause := range []string{"dependencies are missing", "package manager", "npm", "pnpm", "yarn", "cargo", "pip", "wrapper"} {
			if strings.Contains(detail, cause) {
				t.Fatalf("%s: the report attributes the failure to %q. It may attribute it to the pack and the command and to NOTHING ELSE.\ngot: %s", marker, cause, report.Detail)
			}
		}
	}
}

// TestInit_EachEntrypointOutcomeIsIndependentOfTheOthers (SPEC-069 CLM-057).
//
// One pack, several entrypoint engines, some failing and one passing: each produces its
// OWN separately-reported outcome, and a failing one does not turn a passing one into a
// failure. Each engine's outcome is ITS OWN command's exit status and nothing adjacent.
func TestInit_EachEntrypointOutcomeIsIndependentOfTheOthers(t *testing.T) {
	reports, _, err := probeFixture(t, "entrypoint-cases-pack.yml")
	if err != nil {
		t.Fatalf("probing the entrypoint cases errored: %v", err)
	}

	if len(reports) != 4 {
		t.Fatalf("the four declared entrypoints produced %d reports, want 4 — one each.\nreports: %v", len(reports), reports)
	}

	passing := reportFor(t, reports, "exit-zero.sh case-a")
	if passing.Outcome != initialize.OutcomeDelivered {
		t.Fatalf("the passing entrypoint reported %v alongside three failing siblings; outcomes are independent.\ngot: %s", passing.Outcome, passing.Detail)
	}
	for _, failing := range []string{"no-such-entrypoint.sh", "exit-nonzero-stdout.sh", "exit-nonzero-stderr-only.sh"} {
		if reportFor(t, reports, failing).Outcome != initialize.OutcomeBrokenPromise {
			t.Fatalf("%s did not report its own failure", failing)
		}
	}
}

// TestInit_UnallowlistedEntrypointToolIsRefusedBeforeExecution (SPEC-069 CLM-107).
//
// The trust gate sits BEFORE splitCommand and before the runner: the command is NEVER
// handed to anything, and the refusal surfaces as a CONFIG ERROR rather than as a
// toolchain pass or fail. NON-EXECUTION is asserted, not just the error — the fixture's
// command points at the real zero-exit executable precisely so a recorded call would be
// unambiguous proof the gate was bypassed.
func TestInit_UnallowlistedEntrypointToolIsRefusedBeforeExecution(t *testing.T) {
	reports, runner, err := probeFixture(t, "unallowlisted-tool-pack.yml")

	if err == nil {
		t.Fatalf("an un-allowlisted entrypoint tool was accepted.\nreports: %v", reports)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("the refused command WAS executed: %v. The trust gate sits before the splitter and before the runner, so the command must never be handed to anything.", runner.commands)
	}
	if len(reports) != 0 {
		t.Fatalf("a refusal produced %d toolchain reports; it is neither a toolchain pass nor a toolchain fail, so there is no verdict to report.\nreports: %v", len(reports), reports)
	}

	var configErr *check.ConfigError
	if !errors.As(err, &configErr) {
		t.Fatalf("the refusal is not a *check.ConfigError (%T), so it would not map to exit 2: %v", err, err)
	}
	if !strings.Contains(err.Error(), "not-a-trusted-tool") {
		t.Fatalf("the refusal does not name the tool it refused.\ngot: %s", err.Error())
	}
}

// TestInit_EntrypointRunsAsArgvThroughTheSharedRunnerNeverAShell (SPEC-069 CLM-108).
//
// Shell metacharacters pass through as LITERAL arguments. Init executes arbitrary
// pack-supplied command strings, so an unbound execution path here would be a hole in
// the trusted-tool invariant rather than a style preference.
func TestInit_EntrypointRunsAsArgvThroughTheSharedRunnerNeverAShell(t *testing.T) {
	_, runner, err := probeFixture(t, "metachar-command-pack.yml")
	if err != nil {
		t.Fatalf("probing the metachar fixture errored: %v", err)
	}

	if len(runner.commands) != 1 {
		t.Fatalf("the metachar entrypoint ran %d times, want once.\nran: %v", len(runner.commands), runner.commands)
	}
	// THE PROGRAM IS THE DECLARED EXECUTABLE, NEVER A SHELL. splitCommand tokenizes the
	// declared command on whitespace into program + argv; a shell-based implementation
	// would have `sh` as the program and the whole command as one `-c` argument.
	if !strings.HasPrefix(runner.commands[0], "bin/exit-zero.sh ") {
		t.Fatalf("the executed program is not the declared one: %q. Nothing on this path may invoke a shell.", runner.commands[0])
	}
	for _, shell := range []string{"/bin/sh", "/bin/bash", "sh -c", "bash -c"} {
		if strings.Contains(runner.commands[0], shell) {
			t.Fatalf("the command was run through %q.\nran: %s", shell, runner.commands[0])
		}
	}

	// THE METACHARACTERS REACHED THE PROCESS AS LITERAL ARGV. The declared command's
	// echo of its own argv is the proof, read off the SHARED prober's raw record —
	// init's report does not carry a PASSING entrypoint's output, and adding it there
	// just to satisfy a test would put noise in every consumer's successful run.
	probes := (&packEntrypointProber{
		Packs:  []*pack.Manifest{toolchainFixture(t, "metachar-command-pack.yml")},
		Runner: &check.ExecCommandRunner{Dir: toolchainFixtureDir},
	}).Probe(context.Background())
	if len(probes) != 1 {
		t.Fatalf("the shared prober produced %d probes, want 1", len(probes))
	}
	received := string(probes[0].Output)

	if !strings.Contains(received, "$(echo pwned)") {
		t.Fatalf("the command substitution was EXPANDED rather than passed through literally.\nthe entrypoint received: %s", received)
	}
	if strings.Contains(received, "pwned\n") && !strings.Contains(received, "$(echo pwned)") {
		t.Fatalf("a shell evaluated the substitution.\nthe entrypoint received: %s", received)
	}
	if !strings.Contains(received, "&& echo chained") {
		t.Fatalf("`&&` chained a second command instead of arriving as literal argv.\nthe entrypoint received: %s", received)
	}
	if !strings.Contains(received, "a;b|c") {
		t.Fatalf("`;` and `|` did not arrive as literal argv.\nthe entrypoint received: %s", received)
	}
}

// TestInit_NoDeclaredEntrypointReportsCapabilityAbsentWithoutFailing (SPEC-069
// CLM-058).
//
// When no installed pack declares a test or build engine there is nothing to execute:
// the step reports capability-absent and does NOT fail the run. An un-adopted
// capability is a missing benefit, not a broken promise.
func TestInit_NoDeclaredEntrypointReportsCapabilityAbsentWithoutFailing(t *testing.T) {
	reports, runner, err := probeFixture(t, "no-entrypoint-pack.yml")
	if err != nil {
		t.Fatalf("a pack with no declared entrypoint errored: %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("something was executed for a pack declaring no test or build engine: %v", runner.commands)
	}
	if len(reports) != 1 {
		t.Fatalf("want exactly one capability-absent report, got %d: %v", len(reports), reports)
	}
	if reports[0].Outcome != initialize.OutcomeCapabilityAbsent {
		t.Fatalf("reported %v, want OutcomeCapabilityAbsent", reports[0].Outcome)
	}
	if !strings.Contains(strings.ToLower(reports[0].Detail), "not a failure") {
		t.Fatalf("the report does not say this is not a failure, which is the whole point of the capability-absent class.\ngot: %s", reports[0].Detail)
	}
}
