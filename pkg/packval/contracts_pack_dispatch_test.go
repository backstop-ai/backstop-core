package packval_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// contractsPackRel is the in-repo pack SOURCE this test drives. It is the pack
// ISSUE-142 names: 7 rules, 100% pattern-arg-declared, and therefore 100% undispatched
// before this lane.
const contractsPackRel = "packs/contracts"

// contractsExpectedDispatches is 7 rules x 2 fixtures (one positive, one negative on
// each rule's single claim). Zero is the ISSUE-142 vacuous green; the count is asserted
// so a regression back to it cannot pass as a green.
const contractsExpectedDispatches = 14

// packvalRepoRoot walks up to the directory holding go.mod, the same shape
// pkg/pack/engine/contracts_kind_signature_test.go uses for its packs/contracts reads.
func packvalRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate module root (go.mod) from test working dir")
	return ""
}

// requireAstGrep skips LOUDLY when the engine binary is absent, mirroring
// requireSemgrep. The six signature rules in this pack ride ast-grep, pinned at 0.43.0
// by the trusted-tool allowlist. A silently-skipped verdict assertion is the same
// vacuous-green defect class this lane exists to kill.
func requireAstGrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Skipf("SKIPPING A REAL-ENGINE ASSERTION: the ast-grep binary is not on PATH (%v). "+
			"This sub-test proves packs/contracts' fixtures discriminate by actually running the "+
			"engine; it cannot be satisfied by a mock and it is NOT being reported as a pass.", err)
	}
}

// TestContractsPack_PatternArgFixturesDispatchAndDiscriminate (CLM-008) closes the
// measured consequence. packs/contracts is the pack ISSUE-142 names, and every one of
// its rules declares its engine input as an inline `pattern:` with no rule file. Both
// halves are required:
//
//	COUNT   — the fixtures actually reach the engine, 14 times, each carrying the
//	          declaring rule's pattern as the first argument.
//	VERDICT — through the REAL engine, every positive fixture produces zero findings
//	          and every negative produces at least one, so the phase reports pass for a
//	          reason rather than because it executed nothing.
func TestContractsPack_PatternArgFixturesDispatchAndDiscriminate(t *testing.T) {
	root := packvalRepoRoot(t)
	packDir := filepath.Join(root, contractsPackRel)
	m, err := packval.ParseManifest(filepath.Join(packDir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse %s manifest: %v", contractsPackRel, err)
	}

	// patternByRule records what each rule DECLARED, so the dispatch assertion below
	// cannot be satisfied by a path-shaped argv that merely happens to be non-empty.
	patternByRule := map[string]string{}
	for _, r := range m.Content.Ruleset.Rules {
		if r.Pattern == "" {
			t.Fatalf("rule %q declares no inline pattern; this pack is 100%% pattern-arg and "+
				"a rule without one would be silently undispatchable", r.ID)
		}
		patternByRule[r.ID] = r.Pattern
	}
	if len(patternByRule) != 7 {
		t.Fatalf("expected the contracts pack to declare 7 pattern-arg rules, got %d", len(patternByRule))
	}

	t.Run("every declared fixture reaches the engine", func(t *testing.T) {
		rec := &recordingExecutor{}

		packval.RunFixtures(m, packDir, rec)

		if len(rec.calls) == 0 {
			t.Fatal("packs/contracts dispatched ZERO engine runs — this is the ISSUE-142 vacuous " +
				"green in its original habitat: 7 rules, 14 declared fixtures, none executed, " +
				"phase3-fixtures reports pass")
		}
		if len(rec.calls) != contractsExpectedDispatches {
			t.Fatalf("expected %d dispatches (7 rules x 2 fixtures), got %d: %v",
				contractsExpectedDispatches, len(rec.calls), rec.calls)
		}
		declared := map[string]bool{}
		for _, p := range patternByRule {
			declared[p] = true
		}
		for i, targets := range rec.calls {
			if len(targets) != 2 {
				t.Fatalf("dispatch %d: expected exactly [pattern, fixture file], got %v", i, targets)
			}
			if !declared[targets[0]] {
				t.Fatalf("dispatch %d: first target %q is not any rule's declared `pattern:` — "+
					"a pattern-arg dispatch must carry the inline pattern, not a path", i, targets[0])
			}
		}
	})

	t.Run("the fixtures discriminate through the real engine", func(t *testing.T) {
		requireAstGrep(t)

		res := packval.RunFixtures(m, packDir, &packval.DefaultExecutor{})

		if len(res.Errors) != 0 {
			for _, e := range res.Errors {
				t.Errorf("phase3 error: check=%s rule=%s claim=%s message=%s", e.Check, e.Rule, e.Claim, e.Message)
			}
			t.Fatalf("packs/contracts must report NO phase-3 errors under live dispatch; got %d", len(res.Errors))
		}
		if res.Status != "pass" {
			t.Fatalf("expected phase3-fixtures status \"pass\", got %q", res.Status)
		}
	})
}
