package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// THE PACK SEVERITY CONTRACT, LOCKED END TO END.
//
// RATIFIED SEMANTIC: a SARIF `level: warning` from ANY pack is NON-BLOCKING, by
// contract. Severity IS how a pack author declares blockingness — "loud != blocking"
// expressed on the wire — so a pack that wants to inform without gating says
// `level: warning` and that is a supported, intended thing to say, not an accident
// inherited from the coverage-notice fix.
//
// FAIL-CLOSED ON SILENCE: only an explicit "warning" is exempt. `level: error` blocks,
// and an ABSENT level blocks too (pkg/check/parsers.go sarifSeverity maps everything
// that is not "warning" to error). A pack that declares nothing therefore gets the
// strict reading, so forgetting to declare can never buy a pack a free pass.
//
// WHY THESE RUN THROUGH THE WHOLE CHAIN. pkg/gate/policy_severity_test.go already
// covers the policy layer, but it hand-builds gate.Violation values with Severity
// already set — so it locks blocksVerdict and nothing upstream of it. The contract a
// PACK AUTHOR relies on spans three hops in two packages: the SARIF `level` string →
// check.Violation.Severity (check.ParsePackFindings) → gate.Violation.Severity
// (runFindingsEngine's bridge) → the verdict (gate.ApplyPolicy). Any hop could drop or
// rewrite the field and every existing test would still pass. These drive the REAL
// runFindingsEngine over real SARIF bytes and then the REAL ApplyPolicy, so the mapping
// is locked where a pack author actually experiences it.

// packSarifAtLevel builds a one-finding SARIF log at the given level. An EMPTY level
// OMITS the key entirely rather than emitting `"level":""` — the absent-declaration
// case is a missing field, and a test that sent an empty string would exercise a
// different input than the pack it stands in for.
func packSarifAtLevel(level string) []byte {
	levelField := ""
	if level != "" {
		levelField = `"level":"` + level + `",`
	}
	return []byte(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"fake"}},"results":[` +
		`{"ruleId":"house-rule",` + levelField +
		`"message":{"text":"the pack has something to say"},` +
		`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/app.go"},` +
		`"region":{"startLine":7}}}]}]}]}`)
}

// packFindingsAtLevel runs the PRODUCTION findings dispatch over a pack whose engine
// emits one SARIF finding at the given level, returning the gate violations the gate
// would actually consume.
func packFindingsAtLevel(t *testing.T, level string) []gate.Violation {
	t.Helper()

	runner := &capturingRunner{out: packSarifAtLevel(level)}
	binding := engine.EngineBinding{
		Command:   "fake scan",
		InputMode: engine.InputModeNone,
		ScopeKind: engine.ScopeKindProjectWide,
	}

	violations, err := runFindingsEngine(
		&pack.Manifest{NormalizedName: "test-org/house-pack"},
		t.TempDir(), t.TempDir(), nil, binding, nil, runner,
	)
	if err != nil {
		t.Fatalf("runFindingsEngine over %s-level SARIF: %v", levelName(level), err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected the pack's single %s finding to survive the dispatch, got %d: %#v",
			levelName(level), len(violations), violations)
	}
	return violations
}

// verdictFor applies the REAL policy layer to a pack_engines step carrying the given
// violations under a BLOCKING dimension policy — the strictest setting a consumer can
// declare, so a warning that survives here survives everywhere.
func verdictFor(violations []gate.Violation) gate.StepResult {
	step := gate.StepResult{StepName: gate.StepPackEngines, Status: "pass", Violations: violations}
	policy := map[string]gate.DimensionPolicy{
		gate.StepPackEngines: {Level: gate.PolicyBlock, AppliesTo: gate.AppliesToAllCode},
	}
	return gate.ApplyPolicy([]gate.StepResult{step}, nil, policy, &gate.GateScope{Mode: gate.GateScopeModeAll})[0]
}

func levelName(level string) string {
	if level == "" {
		return "absent"
	}
	return level
}

// TestPackFinding_WarningLevelSurvivesAsNonBlocking is the contract itself: a pack
// says `level: warning` and the gate does not fail.
//
// It asserts the severity at the BRIDGE as well as the verdict, because those are two
// different failures with the same symptom. A bridge that dropped the field would
// produce a blocking finding via the fail-closed default — correct-looking machinery
// arriving at the wrong answer — and asserting only the verdict would leave a reader
// unable to tell which hop broke.
func TestPackFinding_WarningLevelSurvivesAsNonBlocking(t *testing.T) {
	violations := packFindingsAtLevel(t, "warning")

	if violations[0].Severity != "warning" {
		t.Fatalf("the SARIF level \"warning\" did not survive to gate.Violation.Severity (got %q). The "+
			"finding then blocks by the fail-closed default, and a pack author's declared non-blocking "+
			"notice gates the build instead", violations[0].Severity)
	}

	got := verdictFor(violations)

	if got.Status == "fail" {
		t.Errorf("a pack finding declared `level: warning` FAILED a blocking dimension. Severity is how a " +
			"pack declares blockingness; if warning blocks, a pack has no way to inform without gating and " +
			"\"loud != blocking\" has no wire representation")
	}
	if len(got.Violations) != 1 {
		t.Errorf("the warning was dropped from the report (%d left). Non-blocking must never mean "+
			"invisible — the pack said it for a reason", len(got.Violations))
	}
}

// TestPackFinding_ErrorLevelBlocks guards the other direction, so the contract cannot
// be satisfied by a gate that stopped blocking on pack findings altogether.
func TestPackFinding_ErrorLevelBlocks(t *testing.T) {
	violations := packFindingsAtLevel(t, "error")

	if violations[0].Severity != "error" {
		t.Fatalf("expected the SARIF level \"error\" to reach gate.Violation.Severity, got %q",
			violations[0].Severity)
	}

	if got := verdictFor(violations); got.Status != "fail" {
		t.Fatalf("a pack finding declared `level: error` did not fail a blocking dimension (status %q). "+
			"The warning exemption has widened into a hole that disarms pack enforcement entirely",
			got.Status)
	}
}

// TestPackFinding_AbsentLevelDefaultsToErrorAndBlocks locks the fail-closed half, and
// it is the half most likely to rot.
//
// SARIF makes `level` optional, so a pack can emit a finding that declares no severity
// at all. The parser maps that to error (pkg/check/parsers.go), which means silence is
// read as the STRICT answer. The alternative — treating an undeclared level as
// non-blocking — would let any pack disable enforcement by omission, which is the
// vacuous green this project exists to prevent.
func TestPackFinding_AbsentLevelDefaultsToErrorAndBlocks(t *testing.T) {
	violations := packFindingsAtLevel(t, "")

	if violations[0].Severity != "error" {
		t.Fatalf("a finding with NO declared level arrived as severity %q; it must default to error. "+
			"Any other default lets a pack escape enforcement by omitting a field", violations[0].Severity)
	}

	if got := verdictFor(violations); got.Status != "fail" {
		t.Fatalf("a finding with NO declared level did not block (status %q) — declaring nothing bought "+
			"the pack a free pass, and the fail-closed default is what stops that", got.Status)
	}
}
