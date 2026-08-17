package packval

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ── FIXTURE CONSTRUCTION (ISSUE-160, strict-SARIF cycle) ─────────────────────
// Runtime-built fixture packs in a temp dir, reusing this package's existing
// helpers rather than re-declaring them: convertFixturePackDir,
// writeConvertFixtureScript (ABSOLUTE path — see the crash-guard file's note on
// why binding.Command must never be a bare basename), stdoutArtifactSarif,
// stdoutArtifactEngineScript, nonSarifEngineScript and toSarifConvertScript.
// No Provision block anywhere.
//
// EXACTLY ONE test in this file guards on the sandbox platform:
// TestPackVal_EngineStrictSarif_ConvertDeclaringBindingExempt, the only binding
// here that declares a Convert (the convert step is the only sandboxed one).
// Guarding the others would skip this cycle's central falsification on Linux —
// the only platform that gates merges — and turn it into a vacuous green in CI.

// strictSarifNonSarifJSON is the shape a v1/too-old lint binary emits: a valid
// JSON OBJECT with no `runs` key. check.ParsePackFindings unmarshals it into the
// SARIF log struct, finds no runs, and returns ZERO findings with NO error — the
// lenient read this guard exists to refuse for a declaring binding.
const strictSarifNonSarifJSON = `{"Issues":[{"Text":"boom"}]}`

// strictSarifEchoScript prints body to stdout and exits 0.
func strictSarifEchoScript(body string) string {
	if body == "" {
		return "#!/bin/sh\nexit 0\n"
	}
	return "#!/bin/sh\ncat <<'PAYLOAD'\n" + body + "\nPAYLOAD\n"
}

// runStrictSarifFixture builds a one-off fixture pack whose engine emits body,
// and runs the production DefaultExecutor over it.
func runStrictSarifFixture(t *testing.T, body string, strict bool) (ExecutionResult, error) {
	t.Helper()
	packDir := convertFixturePackDir(t)
	enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", strictSarifEchoScript(body))
	binding := engine.EngineBinding{Command: enginePath, StrictSarif: strict, Provision: nil}
	return (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
}

// TestPackVal_EngineStrictSarif_NonSarifJsonFailsLoud is THE DEFECT (CLM-010).
// DefaultExecutor.RunEngine never consulted binding.StrictSarif, so a binding
// declaring that its engine emits native SARIF got the LENIENT parse anyway.
//
// Measured 2026-08-17 against HEAD 958b7b0: Passed=false, ExitCode=0, err=nil —
// parseSarif json.Unmarshals any valid JSON object into the sarifLog struct,
// finds no runs, and returns zero findings with no error. phase3.go's POSITIVE
// fixture loop appends an error only on `case r.Passed:`, so that verdict is a
// positive fixture's SILENT clean pass over output the parser could not read.
//
// THE REFUSAL MUST NAME NO TOOL. The gate's twin names a specific linter because
// it lives in a lint-specific file; this is the GENERIC pack-validation dispatch,
// where a baked tool literal violates the zero-baked-language first principle.
// TestPackVal_EngineGateParity_ExecutorCarriesNoToolLiteral makes that
// mechanical over the whole file; the assertion here pins the one message an
// implementer is most likely to copy wholesale.
func TestPackVal_EngineStrictSarif_NonSarifJsonFailsLoud(t *testing.T) {
	res, err := runStrictSarifFixture(t, strictSarifNonSarifJSON, true)
	if err == nil {
		t.Fatalf("THE LYING VERDICT: a binding declaring strict SARIF emitted valid JSON that is NOT a SARIF log, "+
			"yet the run returned a nil error — the lenient parse read it as zero findings, which is the SUCCESS "+
			"condition a positive phase-3 fixture accepts silently. got result %+v", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
	}
	if !strings.Contains(err.Error(), "runs") {
		t.Fatalf("the refusal must name the missing `runs` array so a pack author knows what shape was expected; got: %v", err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "golangci") {
		t.Fatalf("BAKED TOOL LITERAL: the generic pack-validation dispatch must name NO tool and NO language — "+
			"the gate's wording is lint-specific because it lives in a lint-specific file, and copying it here "+
			"violates the zero-baked-language first principle; got: %v", err)
	}
}

// TestPackVal_EngineStrictSarif_UnsetLeavesLenientParseUnchanged keeps CLM-010
// honest (CLM-011) and is what makes the guard OPT-IN. The IDENTICAL non-SARIF
// JSON with StrictSarif FALSE keeps today's lenient behaviour: a nil error and
// Passed=false. Engines that do not promise SARIF must keep it.
//
// Green both before and after the fix, by design.
func TestPackVal_EngineStrictSarif_UnsetLeavesLenientParseUnchanged(t *testing.T) {
	res, err := runStrictSarifFixture(t, strictSarifNonSarifJSON, false)
	if err != nil {
		t.Fatalf("THE GUARD IS NOT OPT-IN: a binding that declares NO strict SARIF must keep the lenient parse — "+
			"non-SARIF JSON reads as zero findings with no error. got: %v (result %+v)", err, res)
	}
	if res.Passed {
		t.Fatalf("expected the leniently-parsed non-SARIF payload to report not-fired, got %+v", res)
	}
}

// TestPackVal_EngineStrictSarif_ValidSarifAndEmptyPayloadPass pins CLM-012. Two
// legs, both declaring StrictSarif and no Convert.
//
// Leg (b) is the one that stops the guard being "harden"ed into a rejection of
// empty output: a genuinely clean run may emit nothing, and the
// emptiness-is-suspicious discipline belongs to the crash guard and the
// never-started refusal, not to a SHAPE check. An implementation that rejects an
// empty payload as malformed fails here — and would red every clean run.
//
// Both legs green before and after the fix, by design.
func TestPackVal_EngineStrictSarif_ValidSarifAndEmptyPayloadPass(t *testing.T) {
	t.Run("well_formed_sarif_reports_fired", func(t *testing.T) {
		res, err := runStrictSarifFixture(t, stdoutArtifactSarif(1), true)
		if err != nil {
			t.Fatalf("a well-formed SARIF log must pass the shape guard; got error: %v (result %+v)", err, res)
		}
		if !res.Passed {
			t.Fatalf("expected the SARIF result to yield Passed=true, got %+v", res)
		}
	})

	t.Run("empty_payload_is_not_malformed", func(t *testing.T) {
		res, err := runStrictSarifFixture(t, "", true)
		if err != nil {
			t.Fatalf("AN EMPTY PAYLOAD IS NOT MALFORMED: a genuinely clean run may emit nothing, so the SHAPE guard "+
				"must return early on empty input — rejecting it would red every clean run. got: %v (result %+v)", err, res)
		}
		if res.Passed {
			t.Fatalf("expected an empty payload to report not-fired, got %+v", res)
		}
	})
}

// TestPackVal_EngineStrictSarif_ConvertDeclaringBindingExempt pins CLM-013: the
// guard is gated on binding.Convert == "", exactly as the gate gates its own.
// Non-SARIF input is PRECISELY what a convert exists to reshape, so applying the
// shape guard to a convert-declaring binding's PRE-convert payload would
// fail-loud on a correctly-authored pack.
//
// THE ONE TEST IN THIS PLAN THAT MAY SKIP: it is the only binding here declaring
// a Convert, and only the convert step runs sandboxed.
//
// toSarifConvertScript's output is DERIVED from its input — it counts the
// "ruleId" occurrences it actually read — so a converter printing a constant
// document could not satisfy this by accident.
//
// Green both before and after the fix; it goes RED if the guard is written
// without the convert gating.
func TestPackVal_EngineStrictSarif_ConvertDeclaringBindingExempt(t *testing.T) {
	requireSandboxPlatform(t)

	packDir := convertFixturePackDir(t)
	enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", nonSarifEngineScript(2))
	writeConvertFixtureScript(t, packDir, "to-sarif.sh", toSarifConvertScript)

	binding := engine.EngineBinding{
		Command:     enginePath,
		StrictSarif: true,
		Convert:     "to-sarif.sh",
		Provision:   nil,
	}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err != nil {
		t.Fatalf("OVER-FIRING: a binding declaring BOTH strict SARIF and a convert must NOT have the shape guard "+
			"applied to its PRE-convert payload — non-SARIF input is exactly what the convert exists to reshape, so "+
			"guarding there fails loud on a correctly-authored pack. got: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("expected the converted findings to yield Passed=true, got %+v", res)
	}
}

// TestPackVal_EngineStrictSarif_GuardReadsSelectedPayload pins CLM-014: the
// guard reads the SELECTED payload, not raw stdout. A binding declaring
// StrictSarif alongside a StdoutArtifact is judged on the ARTIFACT's bytes,
// because the artifact-selection stage has already chosen them by that point.
//
// BOTH DIRECTIONS ARE MANDATED because either alone could pass by accident, and
// the two legs have DIFFERENT PRE-FIX VERDICTS by design:
//   - leg (a) is ALREADY GREEN pre-fix. The selection block picks the artifact's
//     well-formed SARIF, which parses to one finding and returns Passed=true with
//     a nil error before this cycle changes anything. It is a NON-REGRESSION
//     guard over the prior lane's fix, not this cycle's falsification.
//   - leg (b) is RED pre-fix and is the falsifying half: with no shape guard, the
//     artifact's non-SARIF JSON parses leniently to zero findings and returns a
//     nil error.
//
// Not platform-guarded: no convert is involved on this path.
func TestPackVal_EngineStrictSarif_GuardReadsSelectedPayload(t *testing.T) {
	run := func(t *testing.T, stdoutBody, artifactBody string) (ExecutionResult, error) {
		t.Helper()
		packDir := convertFixturePackDir(t)
		script := stdoutArtifactEngineScript(stdoutBody, "results.sarif", artifactBody)
		enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", script)
		binding := engine.EngineBinding{
			Command:        enginePath,
			StrictSarif:    true,
			StdoutArtifact: "results.sarif",
			Provision:      nil,
		}
		return (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	}

	t.Run("well_formed_artifact_beats_non_sarif_stdout", func(t *testing.T) {
		res, err := run(t, strictSarifNonSarifJSON, stdoutArtifactSarif(1))
		if err != nil {
			t.Fatalf("WRONG SUBJECT: the shape guard must judge the SELECTED payload — the declared artifact's "+
				"well-formed SARIF — not raw stdout. Reading stdout here fails loud on a correct run. got: %v (result %+v)", err, res)
		}
		if !res.Passed {
			t.Fatalf("expected the artifact's SARIF result to decide the verdict, got %+v", res)
		}
	})

	t.Run("non_sarif_artifact_fails_despite_well_formed_stdout", func(t *testing.T) {
		res, err := run(t, stdoutArtifactSarif(1), strictSarifNonSarifJSON)
		if err == nil {
			t.Fatalf("WRONG SUBJECT: the declared artifact carried non-SARIF JSON and stdout carried a well-formed "+
				"SARIF log, yet the run returned a nil error — the guard is reading stdout, so a declaring binding's "+
				"real output is never shape-checked at all. got result %+v", res)
		}
		if res.Passed {
			t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
		}
	})
}
