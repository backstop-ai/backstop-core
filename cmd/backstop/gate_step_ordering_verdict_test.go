package main

import (
	"context"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// gate_step_ordering_verdict_test.go pins the CROSS-STEP ORDERING CONTRACT that
// `(*Gate).Run` structurally cannot see (ISSUE-172, CLM-002/CLM-003/CLM-004).
//
// WHY IT EXISTS. ISSUE-172 proposed dispatching the gate's steps CONCURRENTLY,
// reasoning that `pack_engines` and `coverage_threshold` looked independent. The
// investigation found they are not, and that the dependency lives in the WIRING —
// `buildGateSteps` in cmd/backstop/gate.go — not in `pkg/gate`'s dispatch loop.
// Until this file existed the contract was protected by nothing but a comment
// (`cmd/backstop/gate.go`, the "ORDERING IS WHAT MAKES A SUPPLIER SAFE" block just
// above the `collectedVerdicts` declaration). This converts that comment into an
// executable contract.
//
// THE TWO DEPENDENCIES HAVE DIFFERENT FAILURE MODES AND MUST NOT BE CONFLATED.
// A future reader who sees only "an ordering assertion" may delete the wrong one:
//
//   - pack_engines -> test_verification is a CORRECTNESS dependency. `pack_engines`
//     WRITES `collectedVerdicts` / `verdictEngineDeclared`; the `TestVerdictSupplier`
//     closure handed to `gate.StepTestVerificationVerdictFunc` READS them at
//     step-RUN time. Losing the order does not error — it SILENTLY NARROWS
//     enforcement, by either of two paths (CLM-003): a clean reorder downgrades a
//     `critical` `mandated_test_failed` to the non-blocking
//     `test_verification_verdict_capability_absent` advisory, and a genuine data
//     race that observes the bool write but not the slice write joins an EMPTY
//     verdict set and returns an UNQUALIFIED PASS with not even the advisory.
//     TestGateStepOrdering_VerdictJoinIsEmptyWhenTestVerificationRunsFirst below
//     demonstrates the first path executably.
//
//   - pack_engines -> coverage_threshold is a SPEED dependency, created by
//     ISSUE-172's own fix (CLM-004). go-toolchain's `go-test` engine now runs
//     `go test -coverprofile=cover.out`, and the coverage producer REUSES that
//     profile when it finds the freshness stamp the test producer wrote in the SAME
//     gate process. Losing the order is BENIGN by design — no stamp means no reuse
//     and the producer runs the suite exactly as it did before, slow-but-correct,
//     never wrong. But it is also SILENT: nothing reds, the gate simply goes back to
//     paying for two whole-module `go test` runs. That is why it is asserted here
//     rather than left to a comment.
//
// SCOPE: this file adds TESTS ONLY. ISSUE-172's in-process-concurrency approach was
// DECLINED; nothing here implements it, and "just adding a mutex while I'm here"
// would be a different, founder-gated decision.

// TestGateStepOrdering_PackEnginesPrecedesItsDependentSteps asserts BOTH ordering
// dependencies against the REAL assembled pipeline `backstop gate` ships.
//
// It reuses the shipped assembly (`buildGateSteps`) and the shared `stepNameOrder`
// / `indexOf` helpers rather than restating the order from the source, so a change
// to the assembly is what moves this test — not a change to a duplicated list.
func TestGateStepOrdering_PackEnginesPrecedesItsDependentSteps(t *testing.T) {
	project, _ := verdictE2EWorkspace(t, "test-verdict-e2e")
	// The dispatch is seamed so pack_engines is cheap and hermetic; the ORDER of the
	// assembled steps is what this test reads, and the seam does not touch it.
	seamDispatch(t, verdictE2ESarif(t))

	scope, err := gate.ComputeGateScope(project, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("computing the gate scope: %v", err)
	}
	names := stepNameOrder(t, buildGateSteps(project, rootAtDir(t, project), scope))

	packIdx := indexOf(names, gate.StepPackEngines)
	verifyIdx := indexOf(names, gate.StepTestVerification)
	coverageIdx := indexOf(names, gate.StepCoverageThreshold)
	if packIdx < 0 || verifyIdx < 0 || coverageIdx < 0 {
		t.Fatalf("all three steps must be present in the assembled pipeline; order=%v", names)
	}

	// (a) THE CORRECTNESS DEPENDENCY (CLM-002). Losing this silently narrows
	// enforcement — see the file comment and the sibling test below.
	if packIdx > verifyIdx {
		t.Fatalf("%s (%d) MUST precede %s (%d): the verdict collector is written by the former and read by the latter, "+
			"and reordering silently downgrades a failing mandated test to a non-blocking advisory (ISSUE-172 CLM-002/CLM-003); order=%v",
			gate.StepPackEngines, packIdx, gate.StepTestVerification, verifyIdx, names)
	}

	// (b) THE SPEED DEPENDENCY (CLM-004). Losing this is benign in verdict terms —
	// the coverage producer falls back to running the suite — but it is SILENT, and
	// it un-does ISSUE-172's entire fix with nothing red to say so.
	if packIdx > coverageIdx {
		t.Fatalf("%s (%d) MUST precede %s (%d): the coverage producer reuses the cover.out + freshness stamp the "+
			"go-test engine writes in the SAME gate process, and reordering silently reverts the gate to two "+
			"whole-module test runs (ISSUE-172 CLM-004); order=%v",
			gate.StepPackEngines, packIdx, gate.StepCoverageThreshold, coverageIdx, names)
	}
}

// TestGateStepOrdering_VerdictJoinIsEmptyWhenTestVerificationRunsFirst is the leg
// that PROVES the two steps are not independent — the inference ISSUE-172 got wrong.
//
// It drives the SAME wiring twice, over two INDEPENDENT assemblies (each
// `buildGateSteps` call closes over its own fresh collector), and compares:
//
//   - IN ORDER: pack_engines then test_verification -> the `critical`
//     `mandated_test_failed` violation, i.e. enforcement works.
//   - OUT OF ORDER: test_verification BEFORE pack_engines -> NO
//     `mandated_test_failed` at all, and the non-blocking
//     `test_verification_verdict_capability_absent` advisory in its place.
//
// The second half is the finding: reordering does not error, it SILENTLY DOWNGRADES
// enforcement (CLM-003 path 1). Nothing in `pkg/gate` can observe this, because the
// channel is a closure variable in `cmd/backstop/gate.go`.
func TestGateStepOrdering_VerdictJoinIsEmptyWhenTestVerificationRunsFirst(t *testing.T) {
	project, _ := verdictE2EWorkspace(t, "test-verdict-e2e")
	seamDispatch(t, verdictE2ESarif(t))

	scope, err := gate.ComputeGateScope(project, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("computing the gate scope: %v", err)
	}
	root := rootAtDir(t, project)

	// ── IN ORDER ────────────────────────────────────────────────────────────
	// The shipped assembly, run exactly as `(*Gate).Run` runs it.
	inOrder := buildGateSteps(project, root, scope)
	names := make([]string, 0, len(inOrder))
	results := make([]gate.StepResult, 0, len(inOrder))
	for _, step := range inOrder {
		res := step(context.Background())
		names = append(names, res.StepName)
		results = append(results, res)
	}

	packIdx := indexOf(names, gate.StepPackEngines)
	verifyIdx := indexOf(names, gate.StepTestVerification)
	if packIdx < 0 || verifyIdx < 0 {
		t.Fatalf("both steps must be present; order=%v", names)
	}
	if !violationRulePresent(results[verifyIdx].Violations, "mandated_test_failed") {
		t.Fatalf("IN ORDER, test_verification must carry the mandated_test_failed violation; got %#v", results[verifyIdx].Violations)
	}
	for _, v := range results[verifyIdx].Violations {
		if v.Rule == "mandated_test_failed" && v.Severity != "critical" {
			t.Fatalf("mandated_test_failed must be severity critical, got %q", v.Severity)
		}
	}

	// ── OUT OF ORDER ────────────────────────────────────────────────────────
	// A SECOND, INDEPENDENT assembly, so its collector is pristine. Indices are
	// reused from the in-order run: the assembly is deterministic, and identifying a
	// step requires RUNNING it, which would defeat the point here.
	outOfOrder := buildGateSteps(project, root, scope)
	if len(outOfOrder) != len(inOrder) {
		t.Fatalf("the two assemblies must be structurally identical; got %d vs %d steps", len(outOfOrder), len(inOrder))
	}
	verifyFirst := outOfOrder[verifyIdx](context.Background())
	if verifyFirst.StepName != gate.StepTestVerification {
		t.Fatalf("index %d of the second assembly must be %s, got %s", verifyIdx, gate.StepTestVerification, verifyFirst.StepName)
	}
	// pack_engines runs AFTER, which is precisely the point: its write to the
	// collector arrives too late to be read.
	if got := outOfOrder[packIdx](context.Background()); got.StepName != gate.StepPackEngines {
		t.Fatalf("index %d of the second assembly must be %s, got %s", packIdx, gate.StepPackEngines, got.StepName)
	}

	if violationRulePresent(verifyFirst.Violations, "mandated_test_failed") {
		t.Fatalf("OUT OF ORDER, the verdict channel is unpopulated, so no mandated_test_failed can be produced; got %#v", verifyFirst.Violations)
	}
	if !violationRulePresent(verifyFirst.Violations, "test_verification_verdict_capability_absent") {
		t.Fatalf("OUT OF ORDER, test_verification must fall back to the NON-BLOCKING capability-absent advisory — "+
			"this is the silent downgrade ISSUE-172 CLM-003 names; got %#v", verifyFirst.Violations)
	}
	// The downgrade is silent precisely because the advisory does not block: state
	// that explicitly, so the severity contract is pinned alongside the routing.
	if verifyFirst.Status == "fail" {
		t.Fatalf("OUT OF ORDER, the step must NOT read as a failure — that is what makes the enforcement loss SILENT; status=%q", verifyFirst.Status)
	}
}

// violationRulePresent reports whether any violation carries rule.
func violationRulePresent(violations []gate.Violation, rule string) bool {
	for _, v := range violations {
		if v.Rule == rule {
			return true
		}
	}
	return false
}
