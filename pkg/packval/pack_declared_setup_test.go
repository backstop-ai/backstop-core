package packval

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// TestPhase3_DispatchErrorBranches covers the remaining fixture-dispatch error
// branches: an empty-ID tool_config is skipped, a negative fixture whose engine run
// ERRORS is reported under the DEDICATED engine-error check, and a failing multi-file
// validator is a validator-multi-file failure.
//
// The engine-error half was previously asserted as `Check == "semgrep-negative" &&
// Message contains "run failed"`. ISSUE-092 CLM-005 deliberately re-homes engine
// errors onto their own check, precisely so a broken run stops masquerading as a
// fixture verdict, so that assertion could not survive. The name and the multi-file
// half are unchanged.
func TestPhase3_DispatchErrorBranches(t *testing.T) {
	dir := t.TempDir()
	writeSrc(t, dir, "rules/r.yml", "rules:\n  - id: R1\n")
	writeSrc(t, dir, "v.sh", "#!/bin/sh\nexit 0\n")
	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		ToolConfig: []ToolConfigEntry{{ID: ""}}, // skipped by the empty-ID guard
		Content: Content{Ruleset: Ruleset{Rules: []Rule{{
			ID: "R1", Engine: "semgrep", File: "rules/r.yml", RiskClass: "correctness",
			Layer: 3, Category: "presence", InputScope: "multi-file", Validator: "v.sh",
			Claims: []Claim{{ID: "C1", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "p"}},
				Negative: []FixtureRef{{Path: "neg"}},
			}}},
		}}}},
	}
	mock := &MockExecutor{
		EngineFn: func(_ string, _ engine.EngineBinding, targets []string) (ExecutionResult, error) {
			for _, tg := range targets {
				if strings.Contains(tg, "neg") {
					return ExecutionResult{}, errors.New("engine run failed")
				}
			}
			return ExecutionResult{Passed: false}, nil
		},
		ValidatorFn: func(_, _ string, _ []string) (ExecutionResult, error) {
			return ExecutionResult{Passed: false}, nil
		},
	}
	res := RunFixtures(pack, dir, mock)
	var sawEngineError, sawMultiFile bool
	for _, e := range res.Errors {
		if e.Check == "engine-error" && strings.Contains(e.Message, "engine run failed") {
			sawEngineError = true
			if e.Rule != "R1" || e.Claim != "C1" {
				t.Errorf("the engine error must name the rule and claim; got Rule=%q Claim=%q", e.Rule, e.Claim)
			}
		}
		if e.Check == "semgrep-negative" {
			t.Errorf("a broken engine run must not be reported as a negative fixture verdict; got %+v", e)
		}
		if e.Check == "validator-multi-file" {
			sawMultiFile = true
		}
	}
	if !sawEngineError || !sawMultiFile {
		t.Fatalf("expected a dedicated engine-error carrying the underlying message and a validator-multi-file error; got %+v", res.Errors)
	}
}

// TestPhase3_ToolConfigUnknownEngineFailsLoud (CLM-009): a tool_config entry naming
// an engine present in neither the base registry nor the pack's engines: block fails
// loud through the generic dispatch, never a silent pass.
func TestPhase3_ToolConfigUnknownEngineFailsLoud(t *testing.T) {
	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		ToolConfig: []ToolConfigEntry{{
			ID: "tc1", Engine: "no-such-engine", Tool: "x", File: "cfg",
			Claims: []Claim{{ID: "c1", Fixtures: Fixtures{Positive: []FixtureRef{{Path: "p"}}}}},
		}},
	}
	res := RunFixtures(pack, t.TempDir(), &MockExecutor{})
	found := false
	for _, e := range res.Errors {
		if e.Check == "engine-resolve" && strings.Contains(e.Message, "no-such-engine") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected fail-loud engine-resolve error naming the unknown engine; got %+v", res.Errors)
	}
}

// TestPhase3_ToolConfigDispatchPositiveNegative exercises the generic tool_config
// dispatch under the BUNDLE-005 REQ-011 contract: a positive fixture that DOES fire is
// a tool-config-positive failure (a false positive), and a negative fixture that does
// NOT fire is a tool-config-negative failure (an untested claim).
func TestPhase3_ToolConfigDispatchPositiveNegative(t *testing.T) {
	pack := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "generic", Archetype: "enforcement",
		ToolConfig: []ToolConfigEntry{{
			ID: "tc1", Engine: "config-file", Tool: "x", File: "cfg",
			Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "good"}},
				Negative: []FixtureRef{{Path: "bad"}},
			}}},
		}},
	}
	// Fire (Passed=true) only for the POSITIVE "good" target: the positive then fires
	// (a false positive → tool-config-positive) and the negative does not fire (an
	// untested claim → tool-config-negative). Both failure branches at once.
	mock := &MockExecutor{EngineFn: func(_ string, _ engine.EngineBinding, targets []string) (ExecutionResult, error) {
		for _, tg := range targets {
			if strings.Contains(tg, "good") {
				return ExecutionResult{Passed: true}, nil
			}
		}
		return ExecutionResult{Passed: false}, nil
	}}
	res := RunFixtures(pack, t.TempDir(), mock)
	var sawPos, sawNeg bool
	for _, e := range res.Errors {
		if e.Check == "tool-config-positive" {
			sawPos = true
		}
		if e.Check == "tool-config-negative" {
			sawNeg = true
		}
	}
	if !sawPos || !sawNeg {
		t.Fatalf("expected both tool-config-positive and tool-config-negative errors; got %+v", res.Errors)
	}
}

// TestPhase3_GoModTidyPreflightGone (CLM-006): the baked Go module-tidy pre-flight
// is gone — no goModTidyTempCopy, no exec.Command("go", "mod", "tidy") — so the
// tool_config fixture path bakes no Go toolchain step.
func TestPhase3_GoModTidyPreflightGone(t *testing.T) {
	src, err := os.ReadFile("phase3.go")
	if err != nil {
		t.Fatalf("read phase3.go: %v", err)
	}
	s := string(src)
	for _, banned := range []string{
		"goModTidyTempCopy",
		`exec.Command("go", "mod", "tidy")`,
		`"go", "mod", "tidy"`,
	} {
		if strings.Contains(s, banned) {
			t.Fatalf("phase3.go still bakes a Go module-tidy pre-flight: %s", banned)
		}
	}
	// The engine-agnostic rule-id YAML parse is NOT a baked tool invocation and stays.
	if !strings.Contains(s, "semgrepFileContainsRuleID") {
		t.Fatal("semgrepFileContainsRuleID (engine-agnostic YAML id parse) should be retained")
	}
}

// TestScaffoldSkeleton_UsesPackDeclaredIndicator (CLM-007): the skeleton-scaffold
// check keys off a pack-DECLARED test indicator, not a Go-hardwired "_test.go" /
// "func Test" scan. A NON-Go skeleton file satisfying the declared indicator passes;
// absence of the indicator warns GENERICALLY.
func TestScaffoldSkeleton_UsesPackDeclaredIndicator(t *testing.T) {
	// Case A: a non-Go skeleton file that satisfies the declared indicator passes
	// with no test-indicator warning.
	packOK := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "typescript", Archetype: "code",
		Content: Content{Scaffolds: []Scaffold{{
			ID: "S1", Path: "scaf", Tier: "skeleton", TestIndicator: "describe(",
		}}},
	}
	dirOK := t.TempDir()
	writeSrc(t, dirOK, "scaf/app.spec.ts", "describe('thing', () => {})\n")
	rOK := RunFixtures(packOK, dirOK, &MockExecutor{})
	for _, w := range rOK.Warnings {
		if w.Check == "scaffold-skeleton-test-indicator" {
			t.Fatal("a non-Go skeleton satisfying the declared indicator must not warn")
		}
	}

	// Case B: same declared indicator, but no file contains it → generic warning
	// (no Go-specific wording).
	dirWarn := t.TempDir()
	writeSrc(t, dirWarn, "scaf/app.spec.ts", "export const x = 1\n")
	rWarn := RunFixtures(packOK, dirWarn, &MockExecutor{})
	found := false
	for _, w := range rWarn.Warnings {
		if w.Check == "scaffold-skeleton-test-indicator" {
			found = true
			if strings.Contains(w.Message, "_test.go") || strings.Contains(w.Message, "func Test") || strings.Contains(strings.ToLower(w.Message), "go ") {
				t.Fatalf("warning must be language-neutral, got: %q", w.Message)
			}
		}
	}
	if !found {
		t.Fatal("expected a generic test-indicator warning when the declared indicator is absent")
	}
}

// TestScaffoldSkeleton_NoGoTestLiteral (CLM-007): the skeleton block no longer
// contains the "_test.go" suffix scan or the "func Test" literal.
func TestScaffoldSkeleton_NoGoTestLiteral(t *testing.T) {
	src, err := os.ReadFile("phase3.go")
	if err != nil {
		t.Fatalf("read phase3.go: %v", err)
	}
	s := string(src)
	for _, banned := range []string{"_test.go", "func Test"} {
		if strings.Contains(s, banned) {
			t.Fatalf("phase3.go skeleton block still hardwires a Go literal: %s", banned)
		}
	}
}
