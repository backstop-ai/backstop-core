package packval

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ── FIXTURE CONSTRUCTION (ISSUE-160, crash-guard cycle) ──────────────────────
// Runtime-built fixture packs in a temp dir, reusing this package's existing
// helpers (convertFixturePackDir, writeConvertFixtureScript, stdoutArtifactSarif)
// rather than re-declaring them. binding.Command is always the ABSOLUTE path the
// helper returns: exec.Command resolves a separator-less name through PATH and a
// relative one against the PROCESS cwd, either of which would trip
// check.NeverStarted and stay red after the fix, indistinguishable from the
// defect. No Provision block anywhere — a fixture script is not on the
// trusted-tool allowlist and every test would die at the trust gate first.
//
// Nothing here runs a convert, so no test in this file guards on the sandbox
// platform: doing so would skip real coverage on Linux, the only platform that
// gates merges.

// crashGuardSilentCrashScript exits with the given non-zero status having
// printed NOTHING — the tool/infra crash shape. An empty payload parses to ZERO
// findings with NO error, so without the guard this run is indistinguishable
// from a clean scan.
const crashGuardSilentCrashScript = "#!/bin/sh\nexit 2\n"

// crashGuardReportingScript emits a SARIF log with one result and THEN exits
// non-zero — the ordinary findings-engine shape. A build or test engine exits 1
// precisely BECAUSE it found something; that is working correctly, not crashing.
func crashGuardReportingScript() string {
	return "#!/bin/sh\ncat <<'SARIF'\n" + stdoutArtifactSarif(1) + "\nSARIF\nexit 1\n"
}

// TestPackVal_EngineCrashGuard_NonZeroExitWithNoFindingsFailsLoud is THE DEFECT
// (CLM-006). DefaultExecutor.RunEngine never consulted binding.CrashGuard, so a
// declaring engine that crashed — non-zero exit, nothing parseable emitted —
// returned the same verdict as a clean run that simply found nothing.
//
// Measured 2026-08-17 against HEAD 958b7b0: Passed=false, ExitCode=0, err=nil.
// phase3.go's POSITIVE fixture loop appends an error only on `case r.Passed:`,
// so that verdict is a positive fixture's SILENT clean pass — a crashed engine
// validates the pack over a run that produced nothing at all. (Against a
// NEGATIVE fixture the same verdict reds, but as "negative fixture not
// triggered": loud and misattributed to the fixture rather than to the crash.
// This test calls RunEngine directly and exercises neither loop; the polarity is
// stated only so the defect is understood correctly.)
func TestPackVal_EngineCrashGuard_NonZeroExitWithNoFindingsFailsLoud(t *testing.T) {
	packDir := convertFixturePackDir(t)
	enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", crashGuardSilentCrashScript)

	binding := engine.EngineBinding{Command: enginePath, CrashGuard: true, Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err == nil {
		t.Fatalf("THE LYING VERDICT: a CrashGuard engine exited non-zero having emitted nothing parseable, yet the "+
			"run returned a nil error — the verdict a positive phase-3 fixture accepts as a silent clean pass, so a "+
			"crashed engine validates the pack. got result %+v", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
	}
	if !strings.Contains(err.Error(), enginePath) {
		t.Fatalf("the refusal must name the engine %q so the failure is attributable; got: %v", enginePath, err)
	}
	if !strings.Contains(err.Error(), "crashed") {
		t.Fatalf("the refusal must describe the CRASH condition; got: %v", err)
	}
}

// TestPackVal_EngineCrashGuard_UnsetLeavesNonZeroExitUnchanged keeps CLM-006
// honest (CLM-007) and is what makes the guard OPT-IN. The IDENTICAL script with
// CrashGuard FALSE keeps today's behaviour exactly: a nil error and Passed=false.
//
// A findings engine's exit code is not its contract — the SARIF is — so every
// rule-fed engine leaves the flag false. Without this leg, CLM-006 could be
// satisfied by a change that turns every non-zero exit into an error and reds
// every ordinary engine that reports nothing.
//
// Green both before and after the fix, by design.
func TestPackVal_EngineCrashGuard_UnsetLeavesNonZeroExitUnchanged(t *testing.T) {
	packDir := convertFixturePackDir(t)
	enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", crashGuardSilentCrashScript)

	binding := engine.EngineBinding{Command: enginePath, CrashGuard: false, Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err != nil {
		t.Fatalf("THE GUARD IS NOT OPT-IN: a binding that declares NO CrashGuard must keep its current behaviour — "+
			"a non-zero exit with zero findings is an ordinary not-fired verdict, not an error. got: %v (result %+v)", err, res)
	}
	if res.Passed {
		t.Fatalf("expected the finding-free run to report not-fired, got %+v", res)
	}
}

// TestPackVal_EngineCrashGuard_NonZeroExitWithFindingsStillReports pins CLM-008:
// the guard must NOT fire when the engine actually reported. A CrashGuard engine
// that exits non-zero WITH parseable findings is the normal findings-engine case.
//
// This is the leg that stops the fix collapsing into "CrashGuard means a
// non-zero exit is fatal", which would red a build or test engine on every run
// that finds something — precisely the outage class PLAN-ISSUE-067 hit.
//
// The package's existing TestExecutor_RunEngineStartedNonZeroExitStillReportsFindings
// covers the same shape with CrashGuard UNSET; this is its declaring twin, not a
// duplicate of it. Green both before and after the fix.
// The SECOND leg is the opposite-direction one SE14 mandates, and it is here
// rather than left implicit because a mutation run against the finished
// implementation MEASURED its absence: dropping the `runErr != nil` conjunct was
// NOT CAUGHT by any other test in this package. A CLEAN exit with zero findings
// is an ordinary POSITIVE fixture passing — the clean example produced no
// finding, exactly what phase3.go's positive loop expects — so a guard missing
// that conjunct would fail-loud on every clean positive fixture a CrashGuard
// engine validates. The two legs together pin both conjuncts.
func TestPackVal_EngineCrashGuard_NonZeroExitWithFindingsStillReports(t *testing.T) {
	t.Run("non_zero_exit_with_findings", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", crashGuardReportingScript())

		binding := engine.EngineBinding{Command: enginePath, CrashGuard: true, Provision: nil}

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		if err != nil {
			t.Fatalf("OVER-FIRING: a CrashGuard engine that exited non-zero BECAUSE it reported findings is working "+
				"correctly and must not error — the guard requires ZERO parseable findings, not merely a non-zero exit. "+
				"got: %v (result %+v)", err, res)
		}
		if !res.Passed {
			t.Fatalf("expected the reported finding to yield Passed=true, got %+v", res)
		}
	})

	t.Run("clean_exit_with_no_findings", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", "#!/bin/sh\nexit 0\n")

		binding := engine.EngineBinding{Command: enginePath, CrashGuard: true, Provision: nil}

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		if err != nil {
			t.Fatalf("OVER-FIRING: a CrashGuard engine that exited CLEANLY having found nothing is an ordinary "+
				"positive fixture passing, not a crash — the guard requires a NON-ZERO exit as well as zero findings. "+
				"got: %v (result %+v)", err, res)
		}
		if res.Passed {
			t.Fatalf("expected the finding-free clean run to report not-fired, got %+v", res)
		}
	})
}

// TestPackVal_EngineCrashGuard_NeverStartedPrecedesCrashGuard pins CLM-009: the
// never-started refusal still comes FIRST. "crashed: non-zero exit with no
// parseable findings" mis-describes a binary that never ran at all — the same
// misattribution class ISSUE-112 and ISSUE-140 both exist to kill.
//
// Green both before and after the fix: it guards the INSERTION POINT rather than
// the new behaviour. If it reds after the guard lands, the guard was inserted
// ABOVE the never-started refusal.
func TestPackVal_EngineCrashGuard_NeverStartedPrecedesCrashGuard(t *testing.T) {
	packDir := convertFixturePackDir(t)
	absent := filepath.Join(packDir, "engine-that-does-not-exist.sh")

	binding := engine.EngineBinding{Command: absent, CrashGuard: true, Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err == nil {
		t.Fatalf("an engine whose process never started must fail loud, got nil error (result %+v)", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
	}
	if !strings.Contains(err.Error(), "never started") {
		t.Fatalf("the refusal must name the NEVER-STARTED condition; got: %v", err)
	}
	if strings.Contains(err.Error(), "crashed") {
		t.Fatalf("MISATTRIBUTION: a process that never started did not crash — reporting it as a crash sends a pack "+
			"author to debug a tool that never ran. The crash guard must sit AFTER the never-started refusal; got: %v", err)
	}
}
