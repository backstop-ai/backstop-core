package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// THE PACK SEVERITY CONTRACT, LOCKED END TO END, ON CAPTURED TOOL OUTPUT.
//
// RATIFIED SEMANTIC: a SARIF severity of `warning` from ANY pack is NON-BLOCKING, by
// contract. Severity IS how a pack author declares blockingness — "loud != blocking"
// expressed on the wire — so a pack that wants to inform without gating declares
// warning and that is a supported, intended thing to say, not an accident inherited
// from the coverage-notice fix.
//
// WHERE THE SEVERITY IS READ FROM (ISSUE-104). A SARIF log may state a result's level
// in either of two places, and the parser resolves them in this order:
//  1. the result's own `level` (SARIF §3.27.10) — golangci-lint v2 emits this;
//  2. else the PRODUCING RULE'S DESCRIPTOR, tool.driver.rules[].defaultConfiguration
//     .level, joined to the result by ruleId and scoped PER RUN — semgrep emits ONLY
//     this, and omits `level` from the result object entirely;
//  3. else the fail-closed default: error.
//
// FAIL-CLOSED ON SILENCE: only an explicit warning is exempt. An error blocks, and a
// level absent from BOTH places blocks too (pkg/check/parsers.go sarifSeverity maps
// everything that is not "warning" to error). A pack that declares nothing therefore
// gets the strict reading, so forgetting to declare can never buy a pack a free pass.
//
// WHY THE INPUT CHANGED. This file used to build its SARIF BY HAND, writing `level`
// onto the RESULT — a shape real semgrep never emits. The contract was therefore
// locked against an input the parser would never receive, which is precisely why
// these tests passed green through the whole of ISSUE-104: every declared-WARNING
// semgrep rule was arriving at the gate as a blocking error, and the test that exists
// to catch that could not see it. The two capturable directions now run on UNMODIFIED
// semgrep bytes (cmd/backstop/testdata/semgrep/fixtures/, provenance in that
// directory's PROVENANCE.md), so the contract is locked on the wire shape a pack
// author actually produces.
//
// WHY THESE RUN THROUGH THE WHOLE CHAIN. pkg/gate/policy_severity_test.go already
// covers the policy layer, but it hand-builds gate.Violation values with Severity
// already set — so it locks blocksVerdict and nothing upstream of it. The contract a
// PACK AUTHOR relies on spans three hops in two packages: the SARIF severity →
// check.Violation.Severity (check.ParsePackFindings) → gate.Violation.Severity
// (runFindingsEngine's bridge) → the verdict (gate.ApplyPolicy). Any hop could drop or
// rewrite the field and every existing test would still pass. These drive the REAL
// runFindingsEngine over real SARIF bytes and then the REAL ApplyPolicy, so the mapping
// is locked where a pack author actually experiences it.

// readSemgrepCapture reads UNMODIFIED semgrep output from the captured fixture
// corpus. The bytes were never hand-edited; PROVENANCE.md beside them records the
// tool version, the exact command, the capture date and the sha256 of each file,
// and capture.sh re-runs the capture.
func readSemgrepCapture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "semgrep", "fixtures", name))
	if err != nil {
		t.Fatalf("read captured semgrep fixture %s: %v", name, err)
	}
	return b
}

// packSarifAtLevel returns the SARIF a pack's engine emits when it declares the given
// severity.
//
// The two capturable directions are REAL SEMGREP OUTPUT, read verbatim: semgrep states
// the rule's severity on the rule descriptor and emits no result-level `level` at all,
// so these are the bytes the parser actually receives from a pack in production.
//
// The absent case is HAND-BUILT and could not be otherwise: semgrep always emits a
// descriptor, golangci-lint always emits a result level, and no producer on hand emits
// a finding that declares severity in neither place. It is a synthetic SARIF shape
// standing in for such a producer, retained because the fail-closed floor is the half
// most likely to rot. An EMPTY level OMITS the key entirely rather than emitting
// `"level":""` — the absent-declaration case is a missing field, and a test that sent
// an empty string would exercise a different input than the pack it stands in for.
func packSarifAtLevel(t *testing.T, level string) []byte {
	t.Helper()
	switch level {
	case "warning":
		return readSemgrepCapture(t, "descriptor-warning.sarif")
	case "error":
		return readSemgrepCapture(t, "descriptor-error.sarif")
	case "":
		return []byte(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"fake"}},"results":[` +
			`{"ruleId":"house-rule",` +
			`"message":{"text":"the pack has something to say"},` +
			`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/app.go"},` +
			`"region":{"startLine":7}}}]}]}]}`)
	default:
		t.Fatalf("no SARIF input for level %q", level)
		return nil
	}
}

// packFindingsAtLevel runs the PRODUCTION findings dispatch over a pack whose engine
// emits one SARIF finding at the given severity, returning the gate violations the gate
// would actually consume.
func packFindingsAtLevel(t *testing.T, level string) []gate.Violation {
	t.Helper()

	runner := &capturingRunner{out: packSarifAtLevel(t, level)}
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
// declares warning and the gate does not fail. It runs on CAPTURED semgrep output, so
// the severity it starts from is the one a real pack rule actually puts on the wire —
// on the rule descriptor, with no level on the result.
//
// It asserts the severity at the BRIDGE as well as the verdict, because those are two
// different failures with the same symptom. A bridge that dropped the field would
// produce a blocking finding via the fail-closed default — correct-looking machinery
// arriving at the wrong answer — and asserting only the verdict would leave a reader
// unable to tell which hop broke.
func TestPackFinding_WarningLevelSurvivesAsNonBlocking(t *testing.T) {
	violations := packFindingsAtLevel(t, "warning")

	if violations[0].Severity != "warning" {
		t.Fatalf("the declared severity \"warning\" did not survive to gate.Violation.Severity (got %q). The "+
			"finding then blocks by the fail-closed default, and a pack author's declared non-blocking "+
			"notice gates the build instead", violations[0].Severity)
	}

	got := verdictFor(violations)

	if got.Status == "fail" {
		t.Errorf("a pack finding declared at warning severity FAILED a blocking dimension. Severity is how a " +
			"pack declares blockingness; if warning blocks, a pack has no way to inform without gating and " +
			"\"loud != blocking\" has no wire representation")
	}
	if len(got.Violations) != 1 {
		t.Errorf("the warning was dropped from the report (%d left). Non-blocking must never mean "+
			"invisible — the pack said it for a reason", len(got.Violations))
	}
}

// TestPackFinding_ErrorLevelBlocks guards the other direction, so the contract cannot
// be satisfied by a gate that stopped blocking on pack findings altogether. Same
// producer, same capture recipe, one word different in the rule — so the two captures
// differ in severity and nothing else.
func TestPackFinding_ErrorLevelBlocks(t *testing.T) {
	violations := packFindingsAtLevel(t, "error")

	if violations[0].Severity != "error" {
		t.Fatalf("expected the declared severity \"error\" to reach gate.Violation.Severity, got %q",
			violations[0].Severity)
	}

	if got := verdictFor(violations); got.Status != "fail" {
		t.Fatalf("a pack finding declared at error severity did not fail a blocking dimension (status %q). "+
			"The warning exemption has widened into a hole that disarms pack enforcement entirely",
			got.Status)
	}
}

// TestPackFinding_AbsentLevelDefaultsToErrorAndBlocks locks the fail-closed half, and
// it is the half most likely to rot.
//
// ITS INPUT IS HAND-BUILT, AND SAYS SO. SARIF makes severity optional in both places,
// so a pack could emit a finding that declares none — but no producer on hand does:
// semgrep always writes a rule descriptor and golangci-lint always writes a result
// level. This is a synthetic SARIF shape standing in for such a producer, and it is
// the only input in this file that is not captured tool output.
//
// The parser maps that silence to error (pkg/check/parsers.go), which means an
// undeclared severity is read as the STRICT answer. The alternative — treating it as
// non-blocking — would let any pack disable enforcement by omission, which is the
// vacuous green this project exists to prevent.
func TestPackFinding_AbsentLevelDefaultsToErrorAndBlocks(t *testing.T) {
	violations := packFindingsAtLevel(t, "")

	if violations[0].Severity != "error" {
		t.Fatalf("a finding with NO declared severity arrived as %q; it must default to error. "+
			"Any other default lets a pack escape enforcement by omitting a field", violations[0].Severity)
	}

	if got := verdictFor(violations); got.Status != "fail" {
		t.Fatalf("a finding with NO declared severity did not block (status %q) — declaring nothing bought "+
			"the pack a free pass, and the fail-closed default is what stops that", got.Status)
	}
}
