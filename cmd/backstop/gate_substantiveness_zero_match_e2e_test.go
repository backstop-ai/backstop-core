package main

import (
	"context"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// gate_substantiveness_zero_match_e2e_test.go is the ISSUE-113 end-to-end regression over
// the REAL dispatch path: real local pack install → real ast-grep → real
// convert-under-sandbox → route → join. Nothing here is hand-constructed; every violation
// asserted on was produced by the production step.
//
// The proof is THREE-WAY over the SAME workspace, because "one violation after the fix"
// on its own proves nothing — a step that always returned one violation would pass it:
//
//	control  UNPATCHED pack           → 0 violations   (the tests are genuinely fine)
//	before   ZERO-MATCH pack, pre-fix → 3 noTarget violations, all FALSE
//	after    ZERO-MATCH pack, post-fix→ exactly 1 refusal, ConfigErr true
//
// The control is what makes the middle row a proven FALSEHOOD rather than an assertion
// about one. Three mandated tests rather than one is also load-bearing: the defect is
// "N misleading violations instead of 1", and N must be > 1 for the fix to be visibly a
// COLLAPSE rather than a rewording.
//
// ast-grep absence is a t.Fatal via requireAstGrepE2E, NEVER a t.Skip — a skipped
// real-engine test is silent vacuous green.

// zeroMatchRefusalSubstring is the stable fragment of the refusal message: the OBSERVED
// FACT the refusal reports, as opposed to any of the three causes it declines to pick
// between.
const zeroMatchRefusalSubstring = "produced no findings of any kind"

// TestE2E_ZeroMatchClassification_ControlPackReportsNoViolations (CLM-006) — THE
// NON-VACUITY CONTROL. Unlike the other two tests in this file it is GREEN on both sides
// of the fix; that is exactly its job. The same workspace run against the UNPATCHED pack
// produces ZERO violations, which is what proves the three violations the zero-match run
// produces are FALSE rather than merely unwanted.
//
// IF THIS TEST IS RED, THE FIXTURE IS WRONG (bad target token, tests placed same-package,
// hollow bodies, ast-grep not seeing the files) AND THE OTHER TWO TESTS IN THIS FILE
// PROVE NOTHING. Read the violation dump below before touching anything else.
func TestE2E_ZeroMatchClassification_ControlPackReportsNoViolations(t *testing.T) {
	requireAstGrepE2E(t)

	ws, err := newZeroMatchE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding the zero-match workspace: %v", err)
	}
	if err := ws.installSubstantivenessLocalPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the UNPATCHED local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()

	if len(res.Violations) != 0 {
		t.Fatalf("CONTROL BROKEN — the zero-match workspace's three tests genuinely call package `gate` "+
			"and genuinely assert, so the UNPATCHED pack must produce NO violations. Got %d; every other "+
			"test in this file is meaningless until this is fixed: %#v", len(res.Violations), res.Violations)
	}
	if res.Status != "pass" {
		t.Fatalf("CONTROL BROKEN — the substantiveness step must PASS over the unpatched pack; got status %q", res.Status)
	}
	if res.ConfigErr {
		t.Fatal("CONTROL BROKEN — a healthy pack over a healthy workspace must not raise a config error")
	}
}

// TestE2E_ZeroMatchClassification_RefusesInsteadOfPerTestViolations (CLM-005) — THE
// FALSIFIER. The same workspace, but the pack's Q2 referenced-symbol classification is
// baked to a layout that matches nothing, so the set-join is starved. Pre-fix this
// produces three FALSE "does not call package gate" violations and names the real cause
// nowhere; post-fix it produces exactly one config-error refusal.
//
// Assertions are ordered so a reader can tell WHICH hop broke.
func TestE2E_ZeroMatchClassification_RefusesInsteadOfPerTestViolations(t *testing.T) {
	requireAstGrepE2E(t)

	ws, err := newZeroMatchE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding the zero-match workspace: %v", err)
	}
	if err := ws.installZeroMatchSubstantivenessPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the zero-match (Q2-starved) local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()

	// 1. THE COLLAPSE. Pre-fix this is 3 and the red lands here; the dump below is the
	// false "does not call package gate" wall ISSUE-113 is about, printed verbatim.
	if len(res.Violations) != 1 {
		t.Fatalf("a starved Q2 join must collapse to ONE refusal instead of one violation per mandated test; got %d:\n%s",
			len(res.Violations), renderViolations(res.Violations))
	}

	// 2. the surviving violation is the refusal, not a per-test verdict.
	//
	// PINNED ON "test function ", NOT ON "does not call package". The refusal message
	// deliberately QUOTES the latter — CLM-003 requires it to say what it is refusing
	// INSTEAD OF reporting, so the operator who saw the wall recognizes this as its
	// replacement — which makes that substring useless for telling the two apart. The
	// per-test verdict's actual distinguishing shape is its `test function <NAME> does not
	// call package <PKG>` prefix (pkg/gate/substantiveness_join.go, NoTargetViolation), and
	// that is what must be absent here.
	if strings.Contains(res.Violations[0].Message, "test function ") {
		t.Fatalf("the single violation must be the refusal, not a per-test noTarget verdict; got %q", res.Violations[0].Message)
	}
	for _, name := range []string{"TestZeroMatchAlpha", "TestZeroMatchBravo", "TestZeroMatchCharlie"} {
		if strings.Contains(res.Violations[0].Message, name) {
			t.Fatalf("the refusal must belong to NO individual test — naming %s re-personalizes a finding whose whole point is that it belongs to none of them; got %q", name, res.Violations[0].Message)
		}
	}

	// 3. it is a CONFIG error, which is what halts the run, makes the refusal unwaivable,
	// and keeps it out of baseline grandfathering.
	if !res.ConfigErr {
		t.Fatalf("the refusal must be a config error (ConfigErr true) — a broken classifier is a broken TOOL, not broken code; got %#v", res)
	}

	// 4. and the step still fails.
	if res.Status != "fail" {
		t.Fatalf("the refusing step must report status %q; got %q", "fail", res.Status)
	}

	// 5. the message names the cause. These fixtures are deliberately NON-hollow (each
	// body carries a t.Fatalf), so the routed hollow partition is empty and condition (c)
	// is satisfied — that is WHY the refusal is reachable here at all. If this assertion
	// fails on the count while 1-4 pass, suspect a hollow finding leaking into the
	// workspace (a fixture body that lost its t.Fatalf), not the message wording.
	msg := res.Violations[0].Message
	if !strings.Contains(msg, "3") {
		t.Fatalf("the refusal must report the join-eligible count (3) so the operator sees the scale of what it declined to report; got %q", msg)
	}
	if !strings.Contains(msg, zeroMatchRefusalSubstring) {
		t.Fatalf("the refusal must state the observed fact — that the pack produced no findings of any kind; got %q", msg)
	}
}

// TestE2E_ZeroMatchClassification_RefusalIsNotWaivable (CLM-008) — the refusal is
// STRUCTURALLY unsuppressible, not merely un-waived-in-practice. Because ConfigErr halts
// Gate.Run at this step, waiver_resolution (which is ordered after substantiveness) never
// executes at all, so no inline token can reach it. The LineReader below hands out a
// syntactically valid, non-expired @waiver:test_substantiveness token for EVERY line
// queried — the most permissive reader constructible — and it still suppresses nothing.
func TestE2E_ZeroMatchClassification_RefusalIsNotWaivable(t *testing.T) {
	requireAstGrepE2E(t)

	ws, err := newZeroMatchE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding the zero-match workspace: %v", err)
	}
	if err := ws.installZeroMatchSubstantivenessPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the zero-match (Q2-starved) local pack: %v", err)
	}
	stepRes := ws.runProductionSubstantivenessStep()

	alwaysWaived := func(string, int) (string, bool) {
		return "// @waiver:test_substantiveness:false-positive:2999-01-01", true
	}
	g := gate.New(
		gate.WithSteps([]gate.StepFunc{
			func(context.Context) gate.StepResult { return stepRes },
			gate.StepWaiverResolutionScopedFunc(nil),
		}),
		gate.WithWaiver(alwaysWaived, nil, substantivenessWaiverNow()),
	)
	out, exit := g.Run(context.Background())

	// A config error is exit 2 — the operator reads one message instead of scrolling past
	// downstream noise.
	if exit != 2 {
		t.Fatalf("a ConfigErr refusal must exit 2 (config error), not 1 (violations); got %d", exit)
	}
	// The waiver step never ran, so there was nothing for the token to be adjudicated by.
	for _, s := range out.Steps {
		if s.StepName == gate.StepWaiverResolution {
			t.Fatalf("the ConfigErr halt must break the loop BEFORE waiver_resolution runs — a broken classifier "+
				"must not be papered over with an inline token; got step %#v", s)
		}
	}
	if len(out.ActiveWaivers) != 0 {
		t.Fatalf("no waiver can become active when the run halts before waiver_resolution; got %#v", out.ActiveWaivers)
	}
	// And the refusal itself survives into the reported result.
	sub := substantivenessStep(t, out)
	if len(sub.Violations) != 1 || !strings.Contains(sub.Violations[0].Message, zeroMatchRefusalSubstring) {
		t.Fatalf("the refusal violation must still be reported after the halt; got %s", renderViolations(sub.Violations))
	}
}

// renderViolations formats a violation slice one-per-line so a recorded failure literally
// shows the wall of messages under discussion rather than a single %#v blob.
func renderViolations(violations []gate.Violation) string {
	var b strings.Builder
	for i, v := range violations {
		b.WriteString("  [")
		b.WriteString(strings.TrimSpace(v.Rule))
		b.WriteString("] ")
		b.WriteString(v.File)
		b.WriteString(": ")
		b.WriteString(v.Message)
		if i < len(violations)-1 {
			b.WriteString("\n")
		}
	}
	if b.Len() == 0 {
		return "  (none)"
	}
	return b.String()
}
