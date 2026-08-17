package packval_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// countingExecutor delegates every seam to the REAL DefaultExecutor and counts the
// engine dispatches. It is defined HERE, in the test file, because pkg/packval/
// executor.go belongs to ISSUE-140's lane and this one must not touch it.
type countingExecutor struct {
	packval.DefaultExecutor
	engineCalls int
}

func (c *countingExecutor) RunEngine(packDir string, binding engine.EngineBinding, targets []string) (packval.ExecutionResult, error) {
	c.engineCalls++
	return c.DefaultExecutor.RunEngine(packDir, binding, targets)
}

func phaseResult(t *testing.T, res *packval.Result, name string) packval.PhaseResult {
	t.Helper()
	for _, p := range res.Phases {
		if p.Phase == name {
			return p
		}
	}
	t.Fatalf("phase %q never ran; phases were %+v", name, res.Phases)
	return packval.PhaseResult{}
}

// TestPackVal_PipelineDispatchesFixturesAndCanFail (CLM-008, CLM-003) drives the WHOLE
// `pack test` pipeline — not just RunFixtures — with the REAL DefaultExecutor, so the
// proof covers the path a pack author actually takes and a real engine actually runs.
// A mocked engine could not prove the tool ran.
//
// THE DISPATCH COUNT IS THE POINT, NOT THE PASS. Asserting only Status == "pass" would
// reproduce ISSUE-092's own defect inside its regression test: zero dispatches also
// report pass, which is exactly what every real pack in the fleet was doing.
//
// The corpus is deliberately SEMGREP-ONLY and synthetic. Three deeper defects
// (packval's Rule model has no Pattern field; the executor never applies
// binding.Convert — ISSUE-141; and, before this lane, the unconditional rule-ID
// cross-check) stop the real in-repo packs from completing phase 3 for reasons that
// are not this defect. A semgrep-only corpus is the one that exercises THIS lane's
// mechanism and nothing else.
func TestPackVal_PipelineDispatchesFixturesAndCanFail(t *testing.T) {
	if _, err := exec.LookPath("semgrep"); err != nil {
		t.Skipf("SKIPPING THE END-TO-END DISPATCH PROOF: the semgrep binary is not on PATH (%v). "+
			"This test exists to show the engine genuinely runs; it is NOT being reported as a pass.", err)
	}

	run := func(t *testing.T, pack string) (*packval.Result, *countingExecutor) {
		t.Helper()
		dir, err := filepath.Abs(filepath.Join("testdata", pack))
		if err != nil {
			t.Fatal(err)
		}
		exec := &countingExecutor{}
		res := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "test", Executor: exec}).Run()
		return res, exec
	}

	t.Run("honest pack passes having actually dispatched", func(t *testing.T) {
		res, counter := run(t, "rulepath-pack")

		// The skipped check is load-bearing: Pipeline.Run stops at the first failing
		// phase, so a manifest defect in phases 1-2 would render phase 3 "skipped" and
		// a naive assertion could read that as "not failed".
		for _, p := range res.Phases {
			if p.Status == "skipped" {
				t.Fatalf("no phase may be skipped for the honest pack, or phase 3's verdict is not "+
					"evidence of anything; %q was skipped (reason: %s). All phases: %+v", p.Phase, p.Reason, res.Phases)
			}
		}
		p3 := phaseResult(t, res, "phase3-fixtures")
		if p3.Status != "pass" {
			t.Fatalf("phase3-fixtures = %q for the honest pack; errors: %+v", p3.Status, p3.Errors)
		}
		if counter.engineCalls == 0 {
			t.Fatal("phase3-fixtures reported PASS having dispatched ZERO engine runs — " +
				"that is the ISSUE-092 vacuous green, not a passing pack")
		}
	})

	t.Run("broken-negative pack fails, naming the rule and claim", func(t *testing.T) {
		res, counter := run(t, "rulepath-pack-broken-negative")

		p3 := phaseResult(t, res, "phase3-fixtures")
		if p3.Status != "fail" {
			t.Fatalf("phase3-fixtures = %q for a pack whose negative fixture no longer violates its "+
				"rule; want fail. errors: %+v", p3.Status, p3.Errors)
		}
		if counter.engineCalls == 0 {
			t.Fatal("the failure must come from an executed engine, not from a skipped dispatch")
		}
		named := false
		for _, e := range p3.Errors {
			if e.Rule == "no-global-registry-access" && e.Claim == "C-001" {
				named = true
			}
		}
		if !named {
			t.Fatalf("the failure must name the offending rule and claim; got %+v", p3.Errors)
		}
		if res.Status == "pass" {
			t.Fatal("a failing phase 3 must make the overall pack test result non-passing")
		}
	})
}
