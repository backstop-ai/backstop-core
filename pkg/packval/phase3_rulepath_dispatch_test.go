package packval_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// recordingExecutor captures every RunEngine dispatch so a test can assert on the
// COUNT and on the exact targets. Asserting only on a phase verdict would reproduce
// ISSUE-092's own defect inside its regression test: zero dispatches also report pass.
type recordingExecutor struct {
	packval.MockExecutor
	calls [][]string
}

func (r *recordingExecutor) RunEngine(packDir string, binding engine.EngineBinding, targets []string) (packval.ExecutionResult, error) {
	r.calls = append(r.calls, append([]string(nil), targets...))
	return r.MockExecutor.RunEngine(packDir, binding, targets)
}

func testdataPack(t *testing.T, name string) (*packval.PackManifest, string) {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	m, err := packval.ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse %s manifest: %v", name, err)
	}
	return m, dir
}

// requireSemgrep skips LOUDLY when the engine binary is absent. A silently-skipped
// falsification test is the same vacuous-green defect class this lane exists to kill,
// so the skip names the tool and never asserts success on a skipped run.
func requireSemgrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("semgrep"); err != nil {
		t.Skipf("SKIPPING A FALSIFICATION TEST: the semgrep binary is not on PATH (%v). "+
			"This test proves phase 3 can fail by actually running the engine; it cannot be "+
			"satisfied by a mock and it is NOT being reported as a pass.", err)
	}
}

// TestPackVal_P3_RulePathDeclaredRuleDispatchesFixtures (CLM-002): a rule declaring
// its source with `rule_path:` — the way every real pack does — must actually reach
// executor.RunEngine. Before ISSUE-092 the dispatch was guarded on packval's `File`
// field, so the count here was ZERO and the phase still reported pass.
func TestPackVal_P3_RulePathDeclaredRuleDispatchesFixtures(t *testing.T) {
	m, dir := testdataPack(t, "rulepath-pack")
	rec := &recordingExecutor{}

	packval.RunFixtures(m, dir, rec)

	if len(rec.calls) == 0 {
		t.Fatal("phase 3 dispatched ZERO engine runs for a rule_path-declared rule — " +
			"this is exactly the ISSUE-092 vacuous green: no fixture executed, phase reports pass")
	}
	// One positive fixture + one negative fixture on the single declared claim.
	if len(rec.calls) != 2 {
		t.Fatalf("expected one dispatch per declared fixture (2), got %d: %v", len(rec.calls), rec.calls)
	}
}

// TestPackVal_P3_EngineTargetsAreExplicitFilesNotDirectory (CLM-002) pins ISSUE-091's
// measurement trap: the engine must be invoked with EXPLICIT FILE targets (the rule
// source file plus one fixture file), never a directory. A later refactor to a
// directory scan would silently undercount findings in files the scan skips.
func TestPackVal_P3_EngineTargetsAreExplicitFilesNotDirectory(t *testing.T) {
	m, dir := testdataPack(t, "rulepath-pack")
	rec := &recordingExecutor{}

	packval.RunFixtures(m, dir, rec)

	if len(rec.calls) == 0 {
		t.Fatal("no dispatch recorded; the target shape cannot be asserted")
	}
	for i, targets := range rec.calls {
		if len(targets) != 2 {
			t.Fatalf("dispatch %d: expected exactly [rule file, fixture file], got %v", i, targets)
		}
		if targets[0] != "rules/no-global.yml" {
			t.Fatalf("dispatch %d: first target must be the declared rule source file, got %q", i, targets[0])
		}
		for _, tgt := range targets {
			info, err := os.Stat(filepath.Join(dir, tgt))
			if err != nil {
				t.Fatalf("dispatch %d: target %q does not resolve to a real path: %v", i, tgt, err)
			}
			if info.IsDir() {
				t.Fatalf("dispatch %d: target %q is a DIRECTORY — ISSUE-091's undercount trap; "+
					"targets must be explicit files", i, tgt)
			}
		}
	}
}

// TestPackVal_P3_PhaseThreeFailsWhenNegativeFixtureNoLongerViolates (CLM-003) is THE
// CENTRAL TEST of this lane. It runs the REAL DefaultExecutor — a mocked engine could
// not prove the tool ran — over two packs that differ in exactly one respect: whether
// the declared negative fixture actually violates the rule.
//
// It DISCRIMINATES on purpose. A test that reds on both packs would be keying on
// something other than the fixture's violating content and would prove nothing.
func TestPackVal_P3_PhaseThreeFailsWhenNegativeFixtureNoLongerViolates(t *testing.T) {
	requireSemgrep(t)

	t.Run("broken negative fixture fails the phase", func(t *testing.T) {
		m, dir := testdataPack(t, "rulepath-pack-broken-negative")

		res := packval.RunFixtures(m, dir, &packval.DefaultExecutor{})

		if res.Status != "fail" {
			t.Fatalf("a negative fixture that no longer violates its rule MUST fail phase 3; got status %q with errors %+v",
				res.Status, res.Errors)
		}
		named := false
		for _, e := range res.Errors {
			if e.Rule == "no-global-registry-access" && e.Claim == "C-001" {
				named = true
			}
		}
		if !named {
			t.Fatalf("the failure must name the offending rule and claim; got %+v", res.Errors)
		}
	})

	t.Run("honest fixtures pass the phase", func(t *testing.T) {
		m, dir := testdataPack(t, "rulepath-pack")

		res := packval.RunFixtures(m, dir, &packval.DefaultExecutor{})

		if res.Status != "pass" {
			t.Fatalf("the honest pack must PASS — otherwise the failing case above is not "+
				"discriminating on fixture content; got status %q with errors %+v", res.Status, res.Errors)
		}
	})
}

// TestPackVal_P3_RulePathDispatchSurvivesTheFileAlias guards the back-compat half of
// CLM-001 at the DISPATCH site rather than only at the accessor: a legacy manifest
// declaring `file:` must still dispatch, so reconciling onto rule_path is additive.
func TestPackVal_P3_RulePathDispatchSurvivesTheFileAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rules/legacy.yml", "rules:\n  - id: R1\n")
	writeFile(t, dir, "fixtures/p.go", "package p")
	writeFile(t, dir, "fixtures/n.go", "package p")
	m := &packval.PackManifest{
		Name: "acme/legacy", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Content: packval.Content{Ruleset: packval.Ruleset{Rules: []packval.Rule{{
			ID: "R1", Engine: "semgrep", File: "rules/legacy.yml", RiskClass: "correctness",
			Claims: []packval.Claim{{ID: "C1", Fixtures: packval.Fixtures{
				Positive: []packval.FixtureRef{{Path: "fixtures/p.go"}},
				Negative: []packval.FixtureRef{{Path: "fixtures/n.go"}},
			}}},
		}}}},
	}
	rec := &recordingExecutor{}

	packval.RunFixtures(m, dir, rec)

	if len(rec.calls) == 0 {
		t.Fatal("a legacy file:-declared rule must still dispatch after the rule_path reconciliation")
	}
	for _, targets := range rec.calls {
		if !strings.HasSuffix(targets[0], "legacy.yml") {
			t.Fatalf("the alias must resolve to the declared file, got targets %v", targets)
		}
	}
}
