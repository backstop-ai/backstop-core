package packval_test

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// ruleIDProbeManifest builds a one-rule pack whose declared rule id does NOT appear in
// its rule source file, so the semgrep-rule-id cross-check fires wherever it runs.
func ruleIDProbeManifest(engineName string, engines map[string]engine.EngineBinding) *packval.PackManifest {
	return &packval.PackManifest{
		Name: "acme/ruleid-probe", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		Engines: engines,
		Content: packval.Content{Ruleset: packval.Ruleset{Rules: []packval.Rule{{
			ID: "PACK_RULE_ID", Engine: engineName, RulePath: "rules/source.yml", RiskClass: "correctness",
			Claims: []packval.Claim{{ID: "C1", Fixtures: packval.Fixtures{
				Positive: []packval.FixtureRef{{Path: "fixtures/p.txt"}},
				Negative: []packval.FixtureRef{{Path: "fixtures/n.txt"}},
			}}},
		}}}},
	}
}

func ruleIDProbeDir(t *testing.T, sourceContent string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "rules/source.yml", sourceContent)
	writeFile(t, dir, "fixtures/p.txt", "clean\n")
	writeFile(t, dir, "fixtures/n.txt", "violating\n")
	return dir
}

func hasRuleIDError(errs []packval.ValidationError) bool {
	for _, e := range errs {
		if e.Check == "semgrep-rule-id" {
			return true
		}
	}
	return false
}

// TestPackVal_P3_RuleIDCrossCheckSkippedForConfigFileEngine (CLM-010): the rule-ID
// cross-check assumes the declared file IS A LIST OF RULES each carrying an `id` —
// which is precisely the semantics of `input_mode: rule-flags`. Under
// `input_mode: config-file` the declared file is a PROJECT CONFIG naming rule
// DIRECTORIES and carrying no ids at all, so the check is not merely failing there, it
// is categorically inapplicable.
//
// This is the packs/substantiveness case verbatim: it declares
// `rule_path: ast-grep/sgconfig.yml`, whose entire content is `ruleDirs: [rules]`, and
// the unconditional check made that a second, independent reason the pack could never
// reach phase3-fixtures: pass.
func TestPackVal_P3_RuleIDCrossCheckSkippedForConfigFileEngine(t *testing.T) {
	// The ast-grep shape: a project config naming rule directories, no ids anywhere.
	dir := ruleIDProbeDir(t, "ruleDirs:\n  - rules\n")
	m := ruleIDProbeManifest("ast-grep", nil)

	res := packval.RunFixtures(m, dir, newFixtureMock(false, true))

	if hasRuleIDError(res.Errors) {
		t.Fatalf("a config-file engine's project config must not be run through a check written "+
			"for a semgrep rule-file shape; got %+v", res.Errors)
	}
}

// TestPackVal_P3_RuleIDCrossCheckStillRunsForNonSemgrepRuleFlagsEngine (CLM-010) is
// THE ANTI-REGRESSION TEST, and the more important of the pair.
//
// ⚠ IT EXISTS TO FAIL LOUDLY IF SOMEONE "SIMPLIFIES" THE CONDITION TO
// `rule.Engine == "semgrep"`. That would be two defects at once: a baked engine-name
// literal in the binary, which CLAUDE.md's thin-executor first principle forbids
// outright; and a silent behavior change to
// cmd/backstop/testdata/hermetic-remote/fixture-fail-pack, which declares the
// PACK-DECLARED engine `marker-scan` and a CLAIMLESS rule — so the rule-ID mismatch is
// its ONLY failure mechanism. Under a name check that pack would start PASSING
// `pack test`, breaking
// TestPackvalValidator_RunPackTest_FixtureFailureReturnsValidationError and SPEC-069's
// in-flight work. `marker-scan` declares `input_mode: rule-flags`, so under the
// INPUT-MODE condition the check still runs on it and it still fails.
func TestPackVal_P3_RuleIDCrossCheckStillRunsForNonSemgrepRuleFlagsEngine(t *testing.T) {
	dir := ruleIDProbeDir(t, "rules:\n  - id: SOME_OTHER_ID\n")
	m := ruleIDProbeManifest("marker-scan", map[string]engine.EngineBinding{
		"marker-scan": {
			Command:   "marker-scan --sarif",
			InputMode: engine.InputModeRuleFlags,
			InputFlag: "--rules",
		},
	})

	res := packval.RunFixtures(m, dir, newFixtureMock(false, true))

	if !hasRuleIDError(res.Errors) {
		t.Fatalf("the rule-ID cross-check must key on the DECLARED input mode, not on the engine's "+
			"name: a rule-flags engine that is not called semgrep must still be checked; got %+v", res.Errors)
	}
}

// TestPackVal_P3_RuleIDCrossCheckSkippedWhenEngineDoesNotResolve (CLM-010) pins the
// ORDERING the conditioning requires. With no resolved binding there is no declared
// input mode, so the cross-check has nothing to key on and must not run — but the
// fail-loud engine-resolve error must survive untouched (ISSUE-019: an unknown engine
// is never a silent skip).
func TestPackVal_P3_RuleIDCrossCheckSkippedWhenEngineDoesNotResolve(t *testing.T) {
	dir := ruleIDProbeDir(t, "rules:\n  - id: SOME_OTHER_ID\n")
	m := ruleIDProbeManifest("no-such-engine", nil)

	res := packval.RunFixtures(m, dir, newFixtureMock(false, true))

	resolved := false
	for _, e := range res.Errors {
		if e.Check == "engine-resolve" && strings.Contains(e.Message, "no-such-engine") {
			resolved = true
		}
	}
	if !resolved {
		t.Fatalf("an unresolvable engine must still fail loud, naming it; got %+v", res.Errors)
	}
	if hasRuleIDError(res.Errors) {
		t.Fatalf("with no binding there is no declared input mode, so the rule-ID cross-check has "+
			"nothing to key on and must not ride along; got %+v", res.Errors)
	}
}

// TestPackVal_P3_RuleIDCrossCheckStillRunsForSemgrep is the control: the base
// registry's semgrep declares input_mode: rule-flags, so the check that has always run
// there still runs. Without this, an implementation that skipped the check everywhere
// would satisfy the two skip-tests above.
func TestPackVal_P3_RuleIDCrossCheckStillRunsForSemgrep(t *testing.T) {
	dir := ruleIDProbeDir(t, "rules:\n  - id: SOME_OTHER_ID\n")
	m := ruleIDProbeManifest("semgrep", nil)

	res := packval.RunFixtures(m, dir, newFixtureMock(false, true))

	if !hasRuleIDError(res.Errors) {
		t.Fatalf("semgrep declares input_mode: rule-flags, so the rule-ID cross-check must still run; got %+v", res.Errors)
	}
}
