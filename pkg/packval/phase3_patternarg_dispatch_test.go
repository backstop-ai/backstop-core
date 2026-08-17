package packval_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// patternArgPackPattern is the inline pattern testdata/patternarg-pack declares. The
// tests below assert against it VERBATIM so a dispatch that quietly re-derived a path
// from it could not pass.
const patternArgPackPattern = "forbiddenCall(...)"

// TestPackVal_Manifest_PatternArgRuleCapturesInlinePattern (CLM-001): packval's
// authoring-time Rule model must carry the YAML `pattern:` key the gate-runtime model
// (pkg/pack.Rule) has always carried. Without the field the key is silently discarded
// at unmarshal and every pattern-arg rule loses the only engine input it declares.
func TestPackVal_Manifest_PatternArgRuleCapturesInlinePattern(t *testing.T) {
	m, _ := testdataPack(t, "patternarg-pack")

	rules := m.Content.Ruleset.Rules
	if len(rules) != 1 {
		t.Fatalf("expected the fixture pack to declare exactly one rule, got %d", len(rules))
	}
	r := rules[0]
	if r.Pattern != patternArgPackPattern {
		t.Fatalf("the declared `pattern:` key must round-trip through ParseManifest; got %q, want %q",
			r.Pattern, patternArgPackPattern)
	}
	if r.RuleSourcePath() != "" {
		t.Fatalf("this rule declares NO rule source file — that is the whole point of the "+
			"pattern-arg declaration style; got source %q", r.RuleSourcePath())
	}
}

// TestPackVal_Manifest_DispatchInputIsSingleAuthority (CLM-004): exactly ONE function
// decides "pattern or path". It is named beside RuleSourcePath, which ISSUE-092
// established as the single authority for the rule-file keys, so a future caller
// cannot choose differently.
func TestPackVal_Manifest_DispatchInputIsSingleAuthority(t *testing.T) {
	cases := []struct {
		name string
		rule packval.Rule
		mode engine.InputMode
		want string
	}{
		{
			name: "pattern-arg reads the inline pattern",
			rule: packval.Rule{Pattern: "forbidden(...)"},
			mode: engine.InputModePatternArg,
			want: "forbidden(...)",
		},
		{
			name: "rule-flags reads the rule source path",
			rule: packval.Rule{RulePath: "rules/no-global.yml", Pattern: "forbidden(...)"},
			mode: engine.InputModeRuleFlags,
			want: "rules/no-global.yml",
		},
		{
			name: "config-file reads the rule source path",
			rule: packval.Rule{RulePath: "config/.eslintrc", Pattern: "forbidden(...)"},
			mode: engine.InputModeConfigFile,
			want: "config/.eslintrc",
		},
		{
			name: "none reads the rule source path",
			rule: packval.Rule{File: "rules/legacy.yml"},
			mode: engine.InputModeNone,
			want: "rules/legacy.yml",
		},
		{
			// pkg/gate/testdata/traceability-pack's real shape: BOTH declared. A
			// pattern-arg engine consumes the pattern, never the path.
			name: "both declared, pattern-arg still picks the pattern",
			rule: packval.Rule{RulePath: "rules/no-global.yml", Pattern: "forbidden(...)"},
			mode: engine.InputModePatternArg,
			want: "forbidden(...)",
		},
		{
			name: "the file alias still resolves for a non-pattern-arg mode",
			rule: packval.Rule{File: "rules/legacy.yml"},
			mode: engine.InputModeRuleFlags,
			want: "rules/legacy.yml",
		},
		{
			// CLM-006's caller needs an empty return to test against: a rule
			// declaring nothing this engine can consume.
			name: "declares nothing consumable under pattern-arg",
			rule: packval.Rule{File: "rules/legacy.yml"},
			mode: engine.InputModePatternArg,
			want: "",
		},
		{
			name: "declares nothing at all",
			rule: packval.Rule{},
			mode: engine.InputModeRuleFlags,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule.RuleDispatchInput(tc.mode); got != tc.want {
				t.Fatalf("RuleDispatchInput(%q) = %q, want %q", tc.mode, got, tc.want)
			}
		})
	}
}

// TestPackVal_P3_PatternArgRuleDispatchesFixtures (CLM-002): a rule declaring ONLY
// `pattern:` must actually reach executor.RunEngine, once per declared fixture. The
// assertion is on the CALL COUNT and never on the verdict alone — zero dispatches also
// report `pass`, which is exactly the ISSUE-142 defect.
func TestPackVal_P3_PatternArgRuleDispatchesFixtures(t *testing.T) {
	m, dir := testdataPack(t, "patternarg-pack")
	rec := &recordingExecutor{}

	packval.RunFixtures(m, dir, rec)

	if len(rec.calls) == 0 {
		t.Fatal("phase 3 dispatched ZERO engine runs for a pattern-arg rule — this is exactly " +
			"the ISSUE-142 vacuous green: no fixture executed, phase reports pass")
	}
	// One positive fixture + one negative fixture on the single declared claim.
	if len(rec.calls) != 2 {
		t.Fatalf("expected one dispatch per declared fixture (2), got %d: %v", len(rec.calls), rec.calls)
	}
}

// TestPackVal_P3_PatternArgTargetsCarryTheInlinePatternNotAPath (CLM-003): for a
// pattern-arg engine the FIRST target is the inline pattern STRING, not a filesystem
// path. That is what makes packval's argv agree with the gate's own gatherEngineInputs
// shape without forking a second argv authority.
func TestPackVal_P3_PatternArgTargetsCarryTheInlinePatternNotAPath(t *testing.T) {
	m, dir := testdataPack(t, "patternarg-pack")
	rec := &recordingExecutor{}

	packval.RunFixtures(m, dir, rec)

	if len(rec.calls) == 0 {
		t.Fatal("no dispatch recorded; the target shape cannot be asserted")
	}
	for i, targets := range rec.calls {
		if len(targets) != 2 {
			t.Fatalf("dispatch %d: expected exactly [pattern, fixture file], got %v", i, targets)
		}
		if targets[0] != patternArgPackPattern {
			t.Fatalf("dispatch %d: first target must be the declared inline pattern verbatim, got %q", i, targets[0])
		}
		if _, err := os.Stat(filepath.Join(dir, targets[0])); err == nil {
			t.Fatalf("dispatch %d: the pattern target %q resolved to a real path under packDir — "+
				"a pattern is a command ARGUMENT and must never be stat-ed or joined to packDir", i, targets[0])
		}
		info, err := os.Stat(filepath.Join(dir, targets[1]))
		if err != nil {
			t.Fatalf("dispatch %d: fixture target %q does not resolve to a real path: %v", i, targets[1], err)
		}
		if info.IsDir() {
			t.Fatalf("dispatch %d: fixture target %q is a DIRECTORY — ISSUE-091's undercount trap; "+
				"targets must be explicit files", i, targets[1])
		}
	}
}

// TestPackVal_P3_PatternArgRuleWithoutPatternFailsLoud (CLM-006): a rule that DECLARES
// AN ENGINE INPUT but whose resolved engine cannot consume it — here a pattern-arg
// engine on a rule declaring a rule source and no pattern — is a NAMED, BLOCKING
// phase-3 error, never a quiet skip.
func TestPackVal_P3_PatternArgRuleWithoutPatternFailsLoud(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rules/legacy.yml", "rules:\n  - id: R1\n")
	writeFile(t, dir, "fixtures/p.go", "package p")
	writeFile(t, dir, "fixtures/n.go", "package p")
	m := &packval.PackManifest{
		Name: "acme/patternless", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Engines: map[string]engine.EngineBinding{
			"pattern-engine": {
				Command:   "true",
				InputMode: engine.InputModePatternArg,
				InputFlag: "-e",
				ScopeKind: engine.ScopeKindFileArgs,
			},
		},
		Content: packval.Content{Ruleset: packval.Ruleset{Rules: []packval.Rule{{
			ID: "R1", Engine: "pattern-engine", File: "rules/legacy.yml", RiskClass: "correctness",
			Claims: []packval.Claim{{ID: "C1", Fixtures: packval.Fixtures{
				Positive: []packval.FixtureRef{{Path: "fixtures/p.go"}},
				Negative: []packval.FixtureRef{{Path: "fixtures/n.go"}},
			}}},
		}}}},
	}

	res := packval.RunFixtures(m, dir, &recordingExecutor{})

	if res.Status == "pass" && len(res.Errors) == 0 {
		t.Fatal("a pattern-arg rule declaring no `pattern:` was reported as a PASSING phase with " +
			"zero errors — that silent skip is the ISSUE-142 defect, not an acceptable outcome")
	}
	if res.Status != "fail" {
		t.Fatalf("expected status \"fail\", got %q with errors %+v", res.Status, res.Errors)
	}
	named := false
	for _, e := range res.Errors {
		if e.Rule == "R1" && strings.Contains(e.Message, string(engine.InputModePatternArg)) {
			named = true
		}
	}
	if !named {
		t.Fatalf("the failure must name the rule and its declared input mode; got %+v", res.Errors)
	}
}

// TestPackVal_P3_RulePathDispatchUnchangedByPatternArgSupport (CLM-005) is the
// NON-REGRESSION CONTROL. A rule_path-declared rule must dispatch exactly as before —
// rule source path first, fixture second. Green before this lane and green after; it
// reds only if widening dispatch eligibility broke the path style.
func TestPackVal_P3_RulePathDispatchUnchangedByPatternArgSupport(t *testing.T) {
	m, dir := testdataPack(t, "rulepath-pack")
	rec := &recordingExecutor{}

	packval.RunFixtures(m, dir, rec)

	if len(rec.calls) != 2 {
		t.Fatalf("expected one dispatch per declared fixture (2), got %d: %v", len(rec.calls), rec.calls)
	}
	for i, targets := range rec.calls {
		if len(targets) != 2 {
			t.Fatalf("dispatch %d: expected exactly [rule file, fixture file], got %v", i, targets)
		}
		if targets[0] != "rules/no-global.yml" {
			t.Fatalf("dispatch %d: first target must remain the declared rule SOURCE PATH, not a pattern; got %q",
				i, targets[0])
		}
	}
}

// TestPackVal_P3_PatternArgPhaseThreeFailsWhenNegativeFixtureNoLongerViolates (CLM-007)
// is THE CENTRAL FALSIFICATION for this lane. It runs the REAL DefaultExecutor — a
// mocked engine cannot prove a tool ran — over two packs that differ in exactly one
// respect: whether the declared negative fixture actually violates the rule.
//
// Both halves are required. A test that reds on both discriminates on nothing.
func TestPackVal_P3_PatternArgPhaseThreeFailsWhenNegativeFixtureNoLongerViolates(t *testing.T) {
	requireSemgrep(t)

	t.Run("broken negative fixture fails the phase", func(t *testing.T) {
		m, dir := testdataPack(t, "patternarg-pack-broken-negative")

		res := packval.RunFixtures(m, dir, &packval.DefaultExecutor{})

		if res.Status != "fail" {
			t.Fatalf("a pattern-arg negative fixture that no longer violates its rule MUST fail phase 3; "+
				"got status %q with errors %+v", res.Status, res.Errors)
		}
		named := false
		for _, e := range res.Errors {
			if e.Rule == "no-forbidden-call" && e.Claim == "C-001" {
				named = true
			}
		}
		if !named {
			t.Fatalf("the failure must name the offending rule and claim; got %+v", res.Errors)
		}
	})

	t.Run("honest fixtures pass the phase", func(t *testing.T) {
		m, dir := testdataPack(t, "patternarg-pack")

		res := packval.RunFixtures(m, dir, &packval.DefaultExecutor{})

		if res.Status != "pass" {
			t.Fatalf("the honest pattern-arg pack must PASS — otherwise the failing case above is not "+
				"discriminating on fixture content; got status %q with errors %+v", res.Status, res.Errors)
		}
	})
}
