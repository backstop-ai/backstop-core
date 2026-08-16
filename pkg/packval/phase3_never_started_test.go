package packval

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// TestRunFixtures_NegativeFixtureUnstartableEngineFailsLoud is the end-to-end
// falsification for CLM-003, driving the REAL production path: RunFixtures over a
// NEGATIVE fixture whose declared engine never started. A MockExecutor cannot
// exercise the classification under test, so this uses &DefaultExecutor{}.
//
// At HEAD this returned a CLEAN PASS — Status "pass", zero Errors — because the
// executor's *exec.Error-only check missed the path-ful fork/exec shape, stdout was
// empty, and Passed:false on a NEGATIVE fixture is exactly the success condition.
// That clean phase result IS the vacuous green ISSUE-140 reports.
//
// This file is TEST-ONLY: it exercises pkg/packval/phase3.go but never edits it —
// that file belongs to PLAN-ISSUE-092. The assertions below are therefore written to
// be ORDER-INDEPENDENT with respect to that lane: they must NOT key on the error's
// Check string, because PLAN-ISSUE-092 re-homes engine errors onto a dedicated
// engine-error Check. "The phase FAILED and the rule/claim are named" is true in
// either world.
func TestRunFixtures_NegativeFixtureUnstartableEngineFailsLoud(t *testing.T) {
	dir := t.TempDir()

	// Requirement 3: the rule file must EXIST and CONTAIN the declared rule ID.
	// RunFixtures cross-checks it BEFORE dispatching any fixture and appends a
	// semgrep-rule-id error otherwise — which would make the phase non-zero-error at
	// HEAD for a reason unrelated to ISSUE-140, falsifying the baseline narrative.
	writeSrc(t, dir, "rules/r.yml", "rules:\n  - id: R1\n")

	script := writeUnstartableEngineScript(t, dir, "unstartable-engine.sh")

	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		// Requirement 1: a PACK-LEVEL engines: override. resolveEngine merges
		// pack.Engines OVER baseengines.Registry(), so a bare Engine: "semgrep" with
		// no override would resolve to the REAL base binding — a startable `semgrep …`
		// command whose process starts fine, so the never-started predicate would
		// never fire and this test could never go green.
		//
		// Requirement 2: Provision is NIL. RunEngine runs engine.CheckToolAllowed
		// FIRST whenever Provision is non-nil, so inheriting the base binding's pinned
		// provision would fail the run at the ALLOWLIST gate rather than at process
		// start — a red that keeps firing after the fix.
		Engines: map[string]engine.EngineBinding{
			"semgrep": {Command: script, Provision: nil},
		},
		Content: Content{Ruleset: Ruleset{Rules: []Rule{{
			ID: "R1", Engine: "semgrep", File: "rules/r.yml", RiskClass: "correctness",
			Layer: 3, Category: "presence", InputScope: "single-file",
			// No Validator: the layer-3 validator branches are guarded on it, and a
			// validator failure would add errors unrelated to the engine dispatch.
			Claims: []Claim{{ID: "C1", Fixtures: Fixtures{
				// ONLY a negative fixture. A positive one would ALSO error at HEAD
				// (Passed:false fails a positive), destroying the clean-pass baseline.
				Negative: []FixtureRef{{Path: "neg"}},
			}}},
		}}}},
	}

	res := RunFixtures(pack, dir, &DefaultExecutor{})

	if res.Status != "fail" {
		t.Fatalf("expected phase status %q for a negative fixture whose engine never started, got %q with errors %+v",
			"fail", res.Status, res.Errors)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected at least one validation error; an engine that never ran must not read as a clean negative")
	}
	named := false
	for _, e := range res.Errors {
		if e.Rule == "R1" && e.Claim == "C1" {
			named = true
		}
	}
	if !named {
		t.Fatalf("expected an error naming rule R1 and claim C1; got %+v", res.Errors)
	}
}
