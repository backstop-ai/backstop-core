package main

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// multiRulePacksDir returns the .backstop/packs dir holding the two-rule ast-grep
// fixture pack (test-org/multirule-pack) authored for the un-stubbed E2E.
func multiRulePacksDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "astgrep-multirule", ".backstop", "packs")
}

// TestGateDispatch_AstGrepMultiRuleRealBinaryEndToEnd is the load-bearing
// acceptance test for ISSUE-028 (CLM-004/CLM-005/CLM-006). It drives a TWO-rule
// ast-grep pack through the PRODUCTION dispatchPackEngines with the REAL ast-grep
// binary — NO fixtureRunner stubbing of "ast-grep scan", real convert (the pack's
// to-sarif.sh), real SARIF parse — and asserts findings from BOTH rules arrive.
//
// This is deliberately un-stubbed: every prior ast-grep dispatch test stubbed the
// runner and shipped a one-rule pack, so `--rule <DIR>` never reached real
// ast-grep and the multi-rule drop stayed invisible (a stub is what hid the bug).
// Against the BROKEN rule-dir/"--rule" binding this test genuinely fails (ast-grep
// errors on a directory and reports zero findings); against the config-file fix it
// passes with findings from rule-one AND rule-two.
//
// If the ast-grep binary is absent it FAILS LOUD via t.Fatalf naming ast-grep and
// the install command — NEVER t.Skip (a skip is silent vacuous green and is
// exactly the failure mode this issue exists to kill).
func TestGateDispatch_AstGrepMultiRuleRealBinaryEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Fatalf("ast-grep binary not found on PATH: %v — install ast-grep 0.43.0 "+
			"(e.g. `brew install ast-grep` or `cargo install ast-grep --locked --version 0.43.0`); "+
			"this E2E hard-requires the real binary and MUST NOT be skipped (a skip would "+
			"re-hide the multi-rule dispatch defect)", err)
	}

	// Real convert: shell the pack's REAL to-sarif.sh on the engine's real stdout
	// (the sandbox wrapper is ISSUE-020-fenced; the convert SCRIPT itself is real
	// and un-stubbed — this is the same direct-shell real-convert path the other
	// e2e dispatch tests use).
	stubSandboxedRunStdout(t, nil)

	packsDir := multiRulePacksDir(t)
	packRoot := filepath.Join(packsDir, "test-org", "multirule-pack")
	// projectRoot is the fixture source dir whose positive.go trips BOTH rules in
	// one scan; dispatch appends it as the scan target (scope == nil escape hatch).
	projectRoot := filepath.Join(packRoot, "fixtures", "rules", "rule-one")

	// The REAL production command runner — the same os/exec runner production uses,
	// so `ast-grep scan --config <sgconfig.yml> --json <projectRoot>` actually runs.
	runner := &check.ExecCommandRunner{}

	manifest := &pack.Manifest{
		NormalizedName: "test-org/multirule-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "rule-one", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml", Standard: "x"},
			{ID: "rule-two", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml", Standard: "x"},
		}}},
	}

	violations, err := dispatchPackEngines([]*pack.Manifest{manifest}, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (real ast-grep multi-rule e2e): %v", err)
	}

	// BOTH rules must report. Collect the namespaced violation ids and the
	// per-rule messages so a single-rule or empty result cannot pass.
	const (
		wantRuleOne = "test-org/multirule-pack/rule-one"
		wantRuleTwo = "test-org/multirule-pack/rule-two"
	)
	gotRules := map[string]bool{}
	gotMessages := map[string]bool{}
	for _, v := range violations {
		gotRules[v.Rule] = true
		gotMessages[v.Message] = true
	}

	if len(violations) < 2 {
		t.Fatalf("a two-rule ast-grep pack must report findings from BOTH rules in one "+
			"invocation (len >= 2); got %d: %#v — the broken --rule dispatch drops every "+
			"finding (vacuous green)", len(violations), violations)
	}
	if !gotRules[wantRuleOne] {
		t.Errorf("rule-one's finding is MISSING; got rules %v — a multi-rule pack must not "+
			"drop rule-one", keysOf(gotRules))
	}
	if !gotRules[wantRuleTwo] {
		t.Errorf("rule-two's finding is MISSING; got rules %v — a multi-rule pack must not "+
			"drop rule-two", keysOf(gotRules))
	}
	// Each finding must carry its rule-distinct message so neither rule's result is
	// a stand-in for the other.
	if !gotMessages["forbiddenCallOne is not allowed"] {
		t.Errorf("rule-one's distinct message missing; got messages %v", keysOf(gotMessages))
	}
	if !gotMessages["forbiddenCallTwo is not allowed"] {
		t.Errorf("rule-two's distinct message missing; got messages %v", keysOf(gotMessages))
	}
}

// keysOf returns the keys of a bool set for diagnostic messages.
func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
