package main

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// gate_substantiveness_waiver_e2e_test.go is the ISSUE-116 behavioral acceptance
// proof: an inline @waiver token placed on a hollow finding's OWN reported line
// suppresses that finding through the REAL reconciliation path.
//
// WHY THIS LIVES IN cmd/backstop AND NOT pkg/gate. Two reasons, both structural:
//   - the Line the waiver reconciliation needs is placed on the Violation by
//     cmd/backstop's shared pack dispatch (pack_gate.go:744-748), which pkg/gate
//     cannot import — so only here is the whole chain the real one;
//   - computeWaiverResult is unexported, so the only honest reach into it is the
//     shipped seam, gate.New(WithSteps(...), WithWaiver(...)).Run(ctx).
//
// Every hop between the installed pack and the verdict is production code: real
// distribution.Add → real dispatchPackEngines → real ast-grep → real
// convert-under-sandbox → route → join → gate Run → waiver.Adjudicate.

// substantivenessWaiverNow is the fixed clock for these runs. The fixtures' tokens
// expire in 2999 so they are active and not expiring-soon at this instant. A func
// rather than a package-level var: time.Date is not const-expressible, and the
// go-standards no-global-mutable-state rule (correctly) rejects the var form.
func substantivenessWaiverNow() time.Time {
	return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
}

// hollowFixtureWithToken renders the hollow test source carrying `token` as a
// trailing comment on the test's DECLARATION line (line 5) — the exact line the
// hollow finding is reported at, which is the placement ISSUE-116 is about.
//
// TWO DETAILS ARE LOAD-BEARING; DO NOT "TIDY" EITHER:
//  1. the token must sit on the func declaration line, because that is the line
//     SARIF reports the hollow finding at and the line the adjudicator byte-scans;
//  2. the body calls `gate.Build()` and NOT the harness's default `doSubject()`.
//     `gate` is the spec's mandated target package, so referencing it satisfies the
//     noTarget set-join and keeps the hollow finding the ONLY violation. With
//     `doSubject()` a SECOND, unrelated noTarget violation is raised (that finding
//     carries no line at all — ISSUE-117, explicitly out of scope here), and the
//     step could never flip to pass no matter what the waiver did. `Build` is still
//     genuinely hollow: it matches none of the hollow rule's assertion vocabulary
//     (require|assert|check|verify|expect|must|Fatal|Error|Fail|Skip).
func hollowFixtureWithToken(token string) string {
	return "package sample_test\n\nimport \"testing\"\n\n" +
		"func TestE2EHollowSubject(t *testing.T) { // " + token + "\n\tgate.Build()\n}\n"
}

// newWaiverE2EWorkspace scaffolds the standard substantiveness e2e workspace, then
// overwrites its hollow fixture with the token-carrying variant and installs the
// real local pack.
func newWaiverE2EWorkspace(t *testing.T, token string) *e2eWorkspace {
	t.Helper()
	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	if err := os.WriteFile(ws.hollowFile, []byte(hollowFixtureWithToken(token)), 0o644); err != nil {
		t.Fatalf("writing the token-carrying hollow fixture: %v", err)
	}
	if err := ws.installSubstantivenessLocalPack(repoRoot(t)); err != nil {
		t.Fatalf("installing local pack: %v", err)
	}
	return ws
}

// waiverWorkspaceLineReader builds a waiver LineReader over the temp workspace.
// The hollow violation's File arrives repo-relative (canonicalized by
// NormalizePath) while other violations may arrive absolute, so a relative path is
// joined onto root and an absolute path is passed through unchanged.
func waiverWorkspaceLineReader(root string) func(string, int) (string, bool) {
	return func(file string, line int) (string, bool) {
		path := filepath.FromSlash(file)
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		f, err := os.Open(path)
		if err != nil {
			return "", false
		}
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		n := 0
		for scanner.Scan() {
			n++
			if n == line {
				return scanner.Text(), true
			}
		}
		return "", false
	}
}

// runWaiverEnabledGate feeds the PRODUCTION substantiveness StepResult into a
// waiver-enabled gate Run — the only exported reach into computeWaiverResult.
func runWaiverEnabledGate(ws *e2eWorkspace, res gate.StepResult) gate.GateResult {
	steps := []gate.StepFunc{
		func(context.Context) gate.StepResult { return res },
		gate.StepWaiverResolutionScopedFunc(nil),
	}
	g := gate.New(
		gate.WithSteps(steps),
		gate.WithWaiver(waiverWorkspaceLineReader(ws.root), nil, substantivenessWaiverNow()),
	)
	out, _ := g.Run(context.Background())
	return out
}

// requireOnlyHollowViolation fails unless the production step produced EXACTLY the
// hollow violation. A second violation means the fixture drifted (see
// hollowFixtureWithToken) and every downstream assertion would be misleading.
func requireOnlyHollowViolation(t *testing.T, res gate.StepResult) gate.Violation {
	t.Helper()
	if len(res.Violations) != 1 {
		t.Fatalf("fixture drift: the production step must yield EXACTLY ONE violation (the hollow one); got %d: %#v", len(res.Violations), res.Violations)
	}
	v := res.Violations[0]
	if v.Rule != gate.StepTestSubstantiveness || !strings.Contains(v.Message, "has no assertions (hollow)") {
		t.Fatalf("the single violation must be the hollow one; got %#v", res.Violations)
	}
	return v
}

// substantivenessStep returns the accumulated test_substantiveness StepResult.
func substantivenessStep(t *testing.T, res gate.GateResult) gate.StepResult {
	t.Helper()
	return stepNamed(t, res, gate.StepTestSubstantiveness)
}

// stepNamed returns the accumulated StepResult with the given name.
func stepNamed(t *testing.T, res gate.GateResult, name string) gate.StepResult {
	t.Helper()
	for _, s := range res.Steps {
		if s.StepName == name {
			return s
		}
	}
	t.Fatalf("gate result carries no %q step: %#v", name, res.Steps)
	return gate.StepResult{}
}

// TestE2E_SubstantivenessHollowWaiver_SuppressesFindingOnItsOwnLine (CLM-002) —
// THE FALSIFIER for ISSUE-116. An @waiver:test_substantiveness token on the hollow
// finding's own reported line suppresses that finding through the real production
// path, flipping the test_substantiveness step fail→pass with exactly one active
// waiver. Assertions are ordered so a reader can tell WHICH hop broke.
func TestE2E_SubstantivenessHollowWaiver_SuppressesFindingOnItsOwnLine(t *testing.T) {
	requireAstGrepE2E(t)

	ws := newWaiverE2EWorkspace(t, "@waiver:test_substantiveness:false-positive:2999-01-01")
	stepRes := ws.runProductionSubstantivenessStep()

	// 1. exactly one violation, and it is the hollow one.
	v := requireOnlyHollowViolation(t, stepRes)

	// 2. the mechanism, at the integration surface: the hollow violation carries the
	// finding's reported line. Pre-ISSUE-116 this is 0 and the red lands here first.
	if v.Line == 0 {
		t.Fatalf("the hollow violation must carry the finding's reported Line so the waiver can byte-scan it; got Line=0 in %#v", v)
	}

	res := runWaiverEnabledGate(ws, stepRes)

	// 3. the finding is really gone and the step flipped.
	sub := substantivenessStep(t, res)
	if len(sub.Violations) != 0 {
		t.Fatalf("the co-located @waiver must SUPPRESS the hollow finding; %d violation(s) still stand: %#v", len(sub.Violations), sub.Violations)
	}
	if sub.Status != "pass" {
		t.Fatalf("with its only finding suppressed the test_substantiveness step must be %q; got %q", "pass", sub.Status)
	}

	// 4. the waiver is accounted for as active and reported as such.
	if len(res.ActiveWaivers) != 1 {
		t.Fatalf("expected exactly 1 active waiver over the real path, got %d: %#v", len(res.ActiveWaivers), res.ActiveWaivers)
	}
	if reason := stepNamed(t, res, gate.StepWaiverResolution).Reason; !strings.Contains(reason, "PASS · 1 waivers") {
		t.Fatalf("waiver_resolution must render the PASS·N-waivers state; got Reason=%q", reason)
	}
}

// TestE2E_SubstantivenessHollowWaiver_MismatchedRuleIDSurfacesAsUnused (CLM-003) —
// the diagnostic trail. A token on the finding's own line whose rule-id does NOT
// match (here a realistic near-miss typo) must suppress NOTHING and must surface in
// the waiver step's unused/dangling accounting. Pre-ISSUE-116 the token was never
// harvested at all — windowLines on a Line-0 finding yields only line 0, which no
// LineReader can return — so it could not even be classified, which is exactly what
// made the defect read as "not recognized" rather than "recognized but rejected".
func TestE2E_SubstantivenessHollowWaiver_MismatchedRuleIDSurfacesAsUnused(t *testing.T) {
	requireAstGrepE2E(t)

	ws := newWaiverE2EWorkspace(t, "@waiver:test_substantivenes:false-positive:2999-01-01")
	stepRes := ws.runProductionSubstantivenessStep()

	requireOnlyHollowViolation(t, stepRes)

	res := runWaiverEnabledGate(ws, stepRes)

	// A non-matching token must never suppress anything.
	sub := substantivenessStep(t, res)
	if len(sub.Violations) != 1 {
		t.Fatalf("a token naming a DIFFERENT rule-id must not suppress the hollow finding; got %d violation(s): %#v", len(sub.Violations), sub.Violations)
	}
	if sub.Status != "fail" {
		t.Fatalf("the unsuppressed hollow finding must keep the step %q; got %q", "fail", sub.Status)
	}
	if len(res.ActiveWaivers) != 0 {
		t.Fatalf("a non-matching token must not become an active waiver; got %#v", res.ActiveWaivers)
	}

	// And it must be visible: harvested, then classified unused/dangling.
	reason := stepNamed(t, res, gate.StepWaiverResolution).Reason
	if !strings.Contains(reason, "unused/dangling") || !strings.Contains(reason, "test_substantivenes") {
		t.Fatalf("the mistyped token must surface in the unused/dangling accounting instead of vanishing; got Reason=%q", reason)
	}
}
