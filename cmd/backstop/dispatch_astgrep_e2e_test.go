package main

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// TestGateDispatch_AstGrepProofRuleEndToEnd drives the ast-grep proof rule all the
// way through the dispatch path (CLM-030 / REQ-008/REQ-011): declaration ->
// group-by-engine -> gather config-file (the pack-shipped sgconfig.yml via
// --config, ISSUE-028) -> run (fake runner) -> convert (the pack's real
// to-sarif.sh) -> parseSarif -> namespaced violation. Substantive: asserts
// the converted finding's namespaced id, file, and message all reach gate output,
// so "ast-grep wired end-to-end" cannot be satisfied without the whole pipe.
func TestGateDispatch_AstGrepProofRuleEndToEnd(t *testing.T) {
	stubSandboxedRunStdout(t, nil) // runs the pack's real convert script directly
	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}

	violations, err := dispatchPackEngines([]*pack.Manifest{astGrepDispatchManifest()}, engineDispatchPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (ast-grep e2e): %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected the ast-grep proof rule to produce 1 violation end-to-end, got %d: %#v", len(violations), violations)
	}
	v := violations[0]
	if v.Rule != "test-org/engine-pack/ast-grep-proof" {
		t.Errorf("end-to-end violation must be namespaced pack/rule, got %q", v.Rule)
	}
	if v.File != "main.go" || v.Message != "forbiddenCall is not allowed" {
		t.Errorf("end-to-end finding must reach gate output via the real convert, got %#v", v)
	}
	// The ast-grep engine command really ran (the e2e is real dispatch, not a stub
	// returning canned violations).
	if len(runner.calls) != 1 || runner.calls[0].name != "ast-grep" {
		t.Errorf("expected the ast-grep engine command to run once, got %#v", runner.calls)
	}
}

// TestGateDispatch_ReplacesSemgrepOnlyFeeder proves dispatch produces violations
// via group-by-engine execution with NO semgrep-only feeder (CLM-031 / REQ-011):
// a NON-semgrep engine (ast-grep) fires through the generic dispatch path AND
// the semgrep engine command is NEVER invoked when the pack declares no semgrep
// rule. The retired model funneled every rule into a single semgrep feeder; this
// proves rules now run only through their declared engine. Substantive:
// behavioral on both halves — the ast-grep violation IS produced, and the semgrep
// executable is provably not run for an ast-grep-only pack.
func TestGateDispatch_ReplacesSemgrepOnlyFeeder(t *testing.T) {
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}

	violations, err := dispatchPackEngines([]*pack.Manifest{astGrepDispatchManifest()}, engineDispatchPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("a non-semgrep engine must still fire through group-by-engine dispatch, got %d violations", len(violations))
	}
	if violations[0].Rule != "test-org/engine-pack/ast-grep-proof" {
		t.Errorf("the non-semgrep engine violation must be namespaced, got %q", violations[0].Rule)
	}

	// The retired model funneled every rule into a single semgrep feeder. With an
	// ast-grep-only pack, the semgrep command must NEVER run — proving the
	// semgrep-only feeder is gone and rules dispatch only through their own engine.
	for _, call := range runner.calls {
		if call.name == "semgrep" {
			t.Errorf("an ast-grep-only pack must never invoke the semgrep engine; the semgrep-only feeder must be retired, got call %#v", call)
		}
	}
	// And the ast-grep engine genuinely ran (the no-semgrep result is real
	// dispatch, not a skipped pack).
	ranAstGrep := false
	for _, call := range runner.calls {
		if call.name == "ast-grep" {
			ranAstGrep = true
		}
	}
	if !ranAstGrep {
		t.Error("expected the ast-grep engine command to run for the ast-grep-only pack")
	}
}

// TestGateDispatch_MixedEnginesNotCrossFed proves a pack carrying BOTH semgrep and
// ast-grep rules dispatches each rule to its own engine, never cross-feeding rule
// files (CLM-032 / REQ-011): the semgrep invocation must carry only the semgrep
// rule path and the ast-grep invocation only the ast-grep rule dir. Substantive:
// captures every per-engine invocation and asserts the rule inputs do not bleed
// across engines.
func TestGateDispatch_MixedEnginesNotCrossFed(t *testing.T) {
	stubSandboxedRunStdout(t, nil)
	// Both engines return empty SARIF (semgrep) / empty ast-grep JSON; we only care
	// about HOW each engine was invoked, not the findings.
	runner := &fixtureRunner{byCmd: map[string][]byte{
		"semgrep":       []byte(`{"version":"2.1.0","runs":[]}`),
		"ast-grep scan": []byte(`[]`),
	}}

	manifest := &pack.Manifest{
		NormalizedName: "test-org/engine-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "semgrep-no-eval", Engine: "semgrep", RulePath: "semgrep/no-eval.yml", Standard: "x"},
			{ID: "ast-grep-proof", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml", Standard: "x"},
		}}},
	}

	if _, err := dispatchPackEngines([]*pack.Manifest{manifest}, engineDispatchPacksDir(t), t.TempDir(), nil, runner); err != nil {
		t.Fatalf("dispatchPackEngines (mixed engines): %v", err)
	}

	var semgrepCall, astGrepCall *fixtureCall
	for i := range runner.calls {
		switch runner.calls[i].name {
		case "semgrep":
			semgrepCall = &runner.calls[i]
		case "ast-grep":
			astGrepCall = &runner.calls[i]
		}
	}
	if semgrepCall == nil || astGrepCall == nil {
		t.Fatalf("expected both engines to run; got calls %#v", runner.calls)
	}

	semgrepArgs := strings.Join(semgrepCall.args, " ")
	astGrepArgs := strings.Join(astGrepCall.args, " ")

	// The semgrep invocation must carry the semgrep rule file and NEVER the ast-grep
	// rule dir/file.
	if !strings.Contains(semgrepArgs, "semgrep/no-eval.yml") {
		t.Errorf("semgrep invocation must carry its own rule file, got args %q", semgrepArgs)
	}
	if strings.Contains(semgrepArgs, "ast-grep") {
		t.Errorf("semgrep invocation must NOT be fed the ast-grep rule input, got args %q", semgrepArgs)
	}
	// The ast-grep invocation must carry its own pack-shipped sgconfig.yml (via
	// --config) and NEVER the semgrep rule file.
	if !strings.Contains(astGrepArgs, "ast-grep/sgconfig.yml") {
		t.Errorf("ast-grep invocation must carry its own pack-shipped sgconfig.yml, got args %q", astGrepArgs)
	}
	if !strings.Contains(astGrepArgs, "--config") {
		t.Errorf("ast-grep invocation must inject its config via --config, got args %q", astGrepArgs)
	}
	if strings.Contains(astGrepArgs, "semgrep/no-eval.yml") {
		t.Errorf("ast-grep invocation must NOT be fed the semgrep rule file, got args %q", astGrepArgs)
	}
}
