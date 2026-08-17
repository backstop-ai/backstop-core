package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ── FIXTURE CONSTRUCTION (ISSUE-160, producer cycle) ─────────────────────────
// Every fixture is built at RUNTIME into a temp dir, reusing the helpers this
// package already owns: writeConvertFixtureScript (returns the ABSOLUTE path,
// which buildEngineArgv needs so exec.Command resolves against neither PATH nor
// the PROCESS cwd), convertFixturePackDir, writeUnstartableEngineScript and
// stdoutArtifactSarif. Re-declaring any of them would be a duplicate declaration
// in package packval.
//
// No test here calls requireSandboxPlatform: only the convert step is sandboxed
// and nothing on this path runs a convert, so guarding these would skip real
// coverage on Linux — the only platform that gates merges.
//
// binding.Producer is declared PACK-RELATIVE throughout, deliberately. An
// absolute producer path would resolve identically under any base and destroy
// the packDir-resolution assertion below. That is the mirror of the absolute
// binding.Command rule, not a contradiction of it: Command is handed to
// exec.Command raw, while Producer is joined against packDir first.

// producerSarifScript emits a SARIF log carrying resultCount results to stdout.
func producerSarifScript(resultCount int) string {
	return "#!/bin/sh\ncat <<'SARIF'\n" + stdoutArtifactSarif(resultCount) + "\nSARIF\n"
}

// producerSilentScript emits nothing and exits 0 — the plain command a producer
// is declared to replace. An empty payload parses to ZERO findings with NO
// error, which is precisely the lying verdict this cycle exists to kill.
const producerSilentScript = "#!/bin/sh\nexit 0\n"

// TestPackVal_EngineProducer_ProducerInvokedInPlaceOfCommand is THE DEFECT
// (CLM-001). DefaultExecutor.RunEngine never consulted binding.Producer, so a
// pack declaring one silently got the plain command run instead.
//
// The verdict that produced was a LIE, not a crash. Measured 2026-08-17 against
// HEAD 958b7b0: Passed=false, ExitCode=0, err=nil — the plain command ran,
// emitted nothing, and check.ParsePackFindings read empty output as zero
// findings with no error. phase3.go's POSITIVE fixture loop appends an error
// only on `case r.Passed:`, so that verdict is a positive fixture's SILENT clean
// pass over a run that invoked the wrong program entirely (against a NEGATIVE
// fixture the same verdict reds, but as "negative fixture not triggered" —
// loud and misattributed to the fixture).
func TestPackVal_EngineProducer_ProducerInvokedInPlaceOfCommand(t *testing.T) {
	packDir := convertFixturePackDir(t)
	plainPath := writeConvertFixtureScript(t, packDir, "plain.sh", producerSilentScript)
	writeConvertFixtureScript(t, packDir, "produce.sh", producerSarifScript(1))

	binding := engine.EngineBinding{Command: plainPath, Producer: "produce.sh", Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err != nil {
		t.Fatalf("a declared producer emitting real SARIF must yield findings; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("THE LYING VERDICT: the binding declares a producer that emits one real SARIF result, yet the run "+
			"reports not-fired — the plain command was invoked instead and its empty output parsed to zero findings "+
			"with no error, which is the SUCCESS condition a positive phase-3 fixture accepts silently. got %+v", res)
	}
}

// TestPackVal_EngineProducer_AbsentDeclarationRunsPlainCommand is the assertion
// that keeps CLM-001 honest (CLM-002). A binding with an EMPTY Producer runs
// binding.Command exactly as before and stats no file. The pack dir deliberately
// contains no producer script at all, so an implementation that resolves or runs
// a producer unconditionally surfaces as a refusal rather than passing silently.
//
// Green both before and after the fix, by design.
func TestPackVal_EngineProducer_AbsentDeclarationRunsPlainCommand(t *testing.T) {
	packDir := convertFixturePackDir(t)
	plainPath := writeConvertFixtureScript(t, packDir, "plain.sh", producerSarifScript(1))

	binding := engine.EngineBinding{Command: plainPath, Producer: "", Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err != nil {
		t.Fatalf("a binding with no declared producer must run the plain command; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("expected the plain command's SARIF finding to yield Passed=true, got %+v", res)
	}
	if !strings.Contains(res.Output, "fixture/artifact-0") {
		t.Fatalf("the plain command's own stdout must reach the caller unchanged on the no-declaration path; got Output %q", res.Output)
	}
}

// TestPackVal_EngineProducer_ArgvPreservedIncludingCommandTail pins CLM-003: the
// substitution replaces the invoked NAME ONLY, never the arg shaping. This is
// the leg that distinguishes the FINDINGS-path semantics (runFindingsEngine
// swaps the name because it owns scope contracts a bare call would evaporate)
// from the COVERAGE-path semantics (runCoverageEngine invokes its producer bare
// because a coverage producer shapes its own whole invocation).
//
// buildEngineArgv consumes only fields[0] of binding.Command as the name, so a
// command WITH a tail hands that tail to the producer as $1. The producer here
// therefore forwards "$@" verbatim rather than re-stating a subcommand — a
// producer written as `mytool build "$@"` would run `mytool build build ...`,
// the exact class that cost a full gate outage on PLAN-ISSUE-067.
func TestPackVal_EngineProducer_ArgvPreservedIncludingCommandTail(t *testing.T) {
	packDir := convertFixturePackDir(t)
	plainPath := writeConvertFixtureScript(t, packDir, "plain.sh", producerSilentScript)

	// The producer records its OWN argv, one argument per line, into a file in
	// its cwd (RunEngine sets cmd.Dir = packDir and the engine run is not
	// sandboxed, so a plain redirect lands in packDir), then emits SARIF.
	recorder := "#!/bin/sh\n: > argv.txt\nfor a in \"$@\"; do printf '%s\\n' \"$a\" >> argv.txt; done\ncat <<'SARIF'\n" +
		stdoutArtifactSarif(1) + "\nSARIF\n"
	writeConvertFixtureScript(t, packDir, "produce.sh", recorder)

	binding := engine.EngineBinding{
		Command:   plainPath + " build",
		InputFlag: "--config",
		Producer:  "produce.sh",
		Provision: nil,
	}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"rules/sample.yml", "fixtures/case.txt"})
	if err != nil {
		t.Fatalf("the producer must run and emit its SARIF; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("expected the producer's SARIF result to yield Passed=true, got %+v", res)
	}

	recorded, readErr := os.ReadFile(filepath.Join(packDir, "argv.txt"))
	if readErr != nil {
		t.Fatalf("the producer did not record its argv — it was never invoked: %v", readErr)
	}
	got := strings.Split(strings.TrimRight(string(recorded), "\n"), "\n")
	if len(got) == 1 && got[0] == "" {
		got = nil
	}
	want := []string{"build", "--config", "rules/sample.yml", "fixtures/case.txt"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("ARGV NOT PRESERVED: the producer must receive exactly the args buildEngineArgv shaped for the "+
			"plain command — the Command's own tail, the InputFlag, then the fixture targets, in that order. A BARE "+
			"invocation (the coverage-path semantics) records an empty argv and would drop the rule file and the "+
			"fixture path, so every positive fixture would validate as a silent clean pass over a run that checked "+
			"nothing.\n got %v\nwant %v", got, want)
	}
}

// TestPackVal_EngineProducer_MissingProducerFailsLoud pins CLM-004. A
// declared-but-missing producer is a BROKEN PACK naming both the declared value
// and the resolved path — never a silent fall-back to the plain command, which
// would make a mis-typed declaration look like it worked: strictly worse than
// today, because it would look fixed.
//
// Pre-fix this test is only PARTIALLY red, and that is correct rather than a
// fixture problem. Measured 2026-08-17 against HEAD 958b7b0: Passed=false,
// ExitCode=0, err=nil — the declared producer is silently ignored, the plain
// command runs cleanly and its empty output parses to zero findings. So the
// not-Passed leg ALREADY HOLDS pre-fix while the non-nil-error assertion and
// both substring assertions are RED. The non-nil-error leg is the falsifying
// one: a nil error here means the run silently succeeded off the plain command.
func TestPackVal_EngineProducer_MissingProducerFailsLoud(t *testing.T) {
	const declared = "scripts/absent-producer.sh"

	packDir := convertFixturePackDir(t)
	// The plain command is present and exits cleanly, so a silent fall-back
	// reads as a clean pass rather than surfacing as an unrelated failure.
	plainPath := writeConvertFixtureScript(t, packDir, "plain.sh", producerSilentScript)

	binding := engine.EngineBinding{Command: plainPath, Producer: declared, Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err == nil {
		t.Fatalf("a declared-but-missing producer must fail loud — a nil error here means the run silently fell back "+
			"to the plain command, the SOFT-FAILURE regression CLM-004 forbids. got result %+v", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
	}
	if !strings.Contains(err.Error(), declared) {
		t.Fatalf("the refusal must name the DECLARED value %q so a pack author can find it in pack.yml; got: %v", declared, err)
	}
	want := filepath.Join(packDir, filepath.FromSlash(declared))
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("the refusal must name the RESOLVED path %q; got: %v", want, err)
	}
}

// TestPackVal_EngineProducer_ResolvedAgainstPackDirAndNamedOnNeverStarted pins
// CLM-005 — the base directory (D2) and the never-started subject (D4).
//
// Leg (a) proves the base operationally through a NESTED pack-relative path: a
// resolution joined against the TEST PROCESS's cwd (pkg/packval during go test)
// finds nothing. Leg (b) proves the refusal names the PRODUCER: blaming the tool
// for a producer that never ran is the misattribution class ISSUE-112 and
// ISSUE-140 both exist to kill.
func TestPackVal_EngineProducer_ResolvedAgainstPackDirAndNamedOnNeverStarted(t *testing.T) {
	t.Run("nested_relative_producer_resolves_under_pack_dir", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		plainPath := writeConvertFixtureScript(t, packDir, "plain.sh", producerSilentScript)
		if err := os.MkdirAll(filepath.Join(packDir, "scripts"), 0o755); err != nil {
			t.Fatalf("mkdir nested producer directory: %v", err)
		}
		writeConvertFixtureScript(t, packDir, filepath.Join("scripts", "produce.sh"), producerSarifScript(1))

		binding := engine.EngineBinding{Command: plainPath, Producer: "scripts/produce.sh", Provision: nil}

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		if err != nil {
			t.Fatalf("a nested pack-relative producer must resolve against packDir; got error: %v (result %+v)", err, res)
		}
		if !res.Passed {
			t.Fatalf("expected the nested producer's SARIF result to decide the verdict, got %+v", res)
		}
	})

	t.Run("unstartable_producer_named_on_never_started", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		// The plain command is present and executable, so anything the refusal
		// says about IT would be wrong: it was never reached.
		plainPath := writeConvertFixtureScript(t, packDir, "plain.sh", producerSarifScript(1))
		// EXISTS but is NOT executable: it passes the os.Stat existence check and
		// then fails to exec, which is what routes it to the never-started refusal
		// rather than to the missing-producer refusal.
		producerPath := writeUnstartableEngineScript(t, packDir, "unstartable-producer.sh")

		binding := engine.EngineBinding{Command: plainPath, Producer: "unstartable-producer.sh", Provision: nil}

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		if err == nil {
			t.Fatalf("a producer that cannot be executed must fail loud, got nil error (result %+v)", res)
		}
		if res.Passed {
			t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
		}
		if !strings.Contains(err.Error(), "never started") {
			t.Fatalf("the refusal must name the NEVER-STARTED condition; got: %v", err)
		}
		if !strings.Contains(err.Error(), producerPath) {
			t.Fatalf("the refusal must name the RESOLVED PRODUCER PATH %q — that is what failed to start; got: %v", producerPath, err)
		}
		if strings.Contains(err.Error(), plainPath) {
			t.Fatalf("MISATTRIBUTION: the producer is what failed to start, so the refusal must not name "+
				"binding.Command %q — that sends a pack author to debug a tool that was never reached; got: %v", plainPath, err)
		}
	})
}
