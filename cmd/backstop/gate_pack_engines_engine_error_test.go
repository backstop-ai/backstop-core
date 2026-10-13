package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// ISSUE-209 at the pack_engines STEP altitude. The dispatcher now returns the
// findings it collected alongside the error of an engine that failed; the step's
// error branch must report them (after the diff-scope filter) instead of replacing
// them with one synthetic violation, while staying a config error (exit 2). The
// dispatcher is stubbed at the dispatchPackEnginesFn seam, returning violations
// shaped exactly as the real dispatcher emits them.

// stepErrTestEngineError is the D-1 engine-error violation the dispatcher emits for
// a failed engine.
func stepErrTestEngineError(msg string) gate.Violation {
	return gate.Violation{
		Rule:        gate.StepPackEngines,
		Severity:    "error",
		File:        "",
		ProjectWide: true,
		Message:     msg,
	}
}

// stepErrTestFinding is a namespaced pack-rule finding on file.
func stepErrTestFinding(file, msg string) gate.Violation {
	return gate.Violation{Rule: "org/pack/no-foo", File: file, Message: msg, Severity: "error", SourcePack: "org/pack"}
}

// stepErrTestSetSeam overrides dispatchPackEnginesFn to return violations together
// with err (restored on cleanup).
func stepErrTestSetSeam(t *testing.T, violations []gate.Violation, err error) {
	t.Helper()
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func(_ []*pack.Manifest, _, _ string, _ *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
		return append([]gate.Violation(nil), violations...), err
	}
}

// stepErrTestPackEnginesStep returns the pack_engines StepFunc built by
// buildGateSteps, so the test can run it alone through a real gate.Gate.
func stepErrTestPackEnginesStep(t *testing.T, projectRoot string, scope *gate.GateScope) gate.StepFunc {
	t.Helper()
	for _, step := range buildGateSteps(projectRoot, rootAtDir(t, projectRoot), scope) {
		if step(context.Background()).StepName == gate.StepPackEngines {
			return step
		}
	}
	t.Fatal("expected a pack_engines step in the gate step list")
	return nil
}

// TestPackEnginesStep_EngineErrorPreservesFindingsAndStaysConfigErr (CLM-005): on a
// dispatch error the step reports every preserved finding and every engine-error
// violation in dispatch order, stays fail + ConfigErr, and the gate exits 2. The
// human output renders engine errors un-namespaced ("[pack_engines] <msg>").
func TestPackEnginesStep_EngineErrorPreservesFindingsAndStaysConfigErr(t *testing.T) {
	projectRoot, _ := newDiffScopedPackGateProject(t, "changed.go")

	e1 := `dispatching findings engine "eng-1" for pack org/pack: ENGINE-ONE-BROKE`
	e2 := `dispatching findings engine "eng-2" for pack org/other: ENGINE-TWO-BROKE`
	want := []gate.Violation{
		stepErrTestFinding("a.go", "finding A"),
		stepErrTestEngineError(e1),
		stepErrTestFinding("b.go", "finding B"),
		stepErrTestEngineError(e2),
	}
	stepErrTestSetSeam(t, want, errors.Join(errors.New(e1), errors.New(e2)))

	step := stepErrTestPackEnginesStep(t, projectRoot, nil)
	res := step(context.Background())
	if res.Status != "fail" {
		t.Errorf("a dispatch error must fail the step, got status %q", res.Status)
	}
	if !res.ConfigErr {
		t.Error("a dispatch error must stay a config error (exit 2)")
	}
	if !reflect.DeepEqual(res.Violations, want) {
		t.Errorf("the step must report the preserved violations in dispatch order:\n got: %#v\nwant: %#v", res.Violations, want)
	}

	result, code := gate.New(gate.WithSteps([]gate.StepFunc{step})).Run(context.Background())
	if code != 2 {
		t.Errorf("a pack_engines config error must exit 2, got %d", code)
	}
	human := gate.FormatHuman(result, true)
	for _, msg := range []string{e1, e2} {
		if !strings.Contains(human, "[pack_engines] "+msg) {
			t.Errorf("human output must render the engine error as %q; got:\n%s", "[pack_engines] "+msg, human)
		}
	}
	if strings.Contains(human, "/pack_engines]") {
		t.Errorf("engine errors must NOT render as a namespaced pack rule; got:\n%s", human)
	}
}

// TestPackEnginesStep_EngineErrorScopeFiltersFindingsButNotEngineErrors (CLM-005):
// the preserved findings pass through the diff-scope filter like any other — an
// out-of-scope non-exempt finding is dropped, an out-of-scope ProjectWide finding is
// kept — and the engine-error violation (File "") always survives.
func TestPackEnginesStep_EngineErrorScopeFiltersFindingsButNotEngineErrors(t *testing.T) {
	changed := "changed.go"
	projectRoot, scope := newDiffScopedPackGateProject(t, changed)

	inScope := stepErrTestFinding(changed, "in-scope finding")
	outOfScope := stepErrTestFinding("pkg/untouched/other.go", "out-of-scope finding")
	projectWide := stepErrTestFinding("pkg/untouched/build.go", "project-wide finding")
	projectWide.ProjectWide = true
	engineErr := stepErrTestEngineError(`dispatching findings engine "eng-1" for pack org/pack: BROKE`)
	stepErrTestSetSeam(t, []gate.Violation{inScope, outOfScope, projectWide, engineErr}, errors.New(engineErr.Message))

	res := runPackEnginesStep(t, projectRoot, scope)
	want := []gate.Violation{inScope, projectWide, engineErr}
	if !reflect.DeepEqual(res.Violations, want) {
		t.Errorf("want the out-of-scope non-exempt finding dropped and the rest kept in order:\n got: %#v\nwant: %#v", res.Violations, want)
	}
	if !res.ConfigErr || res.Status != "fail" {
		t.Errorf("the step must stay fail + ConfigErr, got status=%q configErr=%v", res.Status, res.ConfigErr)
	}
}

// TestPackEnginesStep_EngineErrorWithoutErrorViolationKeepsSyntheticViolation
// (CLM-005, D-6; control): when the dispatch returns an error but no engine-error
// violation (a seam stub returning (nil, err), for instance), the step still names
// its error with today's synthetic violation, so it can never fail silently.
func TestPackEnginesStep_EngineErrorWithoutErrorViolationKeepsSyntheticViolation(t *testing.T) {
	changed := "changed.go"
	dispatchErr := errors.New("broken pack org/pack: the seam failed")
	synthetic := gate.Violation{Rule: "pack_engines", Message: dispatchErr.Error(), Severity: "error"}
	finding := stepErrTestFinding(changed, "in-scope finding")

	cases := []struct {
		name     string
		returned []gate.Violation
		want     []gate.Violation
	}{
		{name: "nil violations", returned: nil, want: []gate.Violation{synthetic}},
		{name: "finding without engine-error", returned: []gate.Violation{finding}, want: []gate.Violation{finding, synthetic}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projectRoot, scope := newDiffScopedPackGateProject(t, changed)
			stepErrTestSetSeam(t, tc.returned, dispatchErr)

			res := runPackEnginesStep(t, projectRoot, scope)
			if !reflect.DeepEqual(res.Violations, tc.want) {
				t.Errorf("violations mismatch:\n got: %#v\nwant: %#v", res.Violations, tc.want)
			}
			if res.Status != "fail" || !res.ConfigErr {
				t.Errorf("the step must stay fail + ConfigErr, got status=%q configErr=%v", res.Status, res.ConfigErr)
			}
		})
	}
}
