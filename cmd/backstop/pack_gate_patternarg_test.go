package main

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// SPEC-035 TASK-023 — pattern-arg gather tests (REQ-004/CLM-016/017/018).
//
// These drive gatherEngineInputs for a pattern-arg engine: the BUNDLE-009 seam
// where each rule's inline `pattern:` is passed as a command argument (via the
// engine's InputFlag) INSTEAD of resolving a rule file on disk. The four existing
// gather modes (config-file/rule-flags/none) are unchanged; this is the
// fifth case. The empty-pattern fail-loud mirrors resolveRulePath's missing-path
// fail-loud (a broken pack naming pack + rule), and the no-filesystem-touch
// assertion proves a pattern-arg rule never os.Stats a rule path.

// patternArgBinding is the pattern-arg engine binding under test: it carries the
// InputFlag the gather emits before each rule's pattern, and never a rule-path
// shape.
func patternArgBinding() engine.EngineBinding {
	return engine.EngineBinding{
		Command:   "acme-query --json",
		InputMode: engine.InputModePatternArg,
		InputFlag: "--pattern",
		ScopeKind: engine.ScopeKindFileArgs,
		Category:  engine.EngineCategoryOpinion,
	}
}

// patternArgManifest builds an in-memory manifest naming the pack so the
// fail-loud error can be asserted to name pack + rule.
func patternArgManifest() *pack.Manifest {
	return &pack.Manifest{NormalizedName: "acme/pattern-arg-pack"}
}

// TestGatherInputs_PatternArgEmitsFlagAndPattern proves a pattern-arg engine
// emits [InputFlag, rule.Pattern] for each rule, in rule order, instead of
// resolving a rule file path on disk (CLM-016).
func TestGatherInputs_PatternArgEmitsFlagAndPattern(t *testing.T) {
	m := patternArgManifest()
	binding := patternArgBinding()
	rules := []pack.Rule{
		{ID: "forbid-todo", Engine: "acme-query", Pattern: "comment.contains('TODO')"},
		{ID: "forbid-fixme", Engine: "acme-query", Pattern: "comment.contains('FIXME')"},
	}

	// packRoot is irrelevant for pattern-arg (no filesystem resolution), so a
	// bogus path must NOT cause a failure — the gather never touches it.
	got, err := gatherEngineInputs(m, "/nonexistent/pack/root", binding, rules)
	if err != nil {
		t.Fatalf("pattern-arg gather must succeed without filesystem resolution, got: %v", err)
	}

	want := []string{
		"--pattern", "comment.contains('TODO')",
		"--pattern", "comment.contains('FIXME')",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d gathered args %v, got %d: %v", len(want), want, len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("gathered arg[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestGatherInputs_PatternArgEmptyPatternFailsLoud proves a pattern-arg rule
// with an EMPTY pattern is a blocking broken-pack config error naming the pack
// and the rule — never a silently emitted empty arg (CLM-017).
func TestGatherInputs_PatternArgEmptyPatternFailsLoud(t *testing.T) {
	m := patternArgManifest()
	binding := patternArgBinding()
	rules := []pack.Rule{
		{ID: "empty-pattern-rule", Engine: "acme-query", Pattern: ""},
	}

	got, err := gatherEngineInputs(m, "/nonexistent/pack/root", binding, rules)
	if err == nil {
		t.Fatalf("an empty pattern under a pattern-arg engine must fail loud, got nil with args %v — that is a silent empty-arg emission", got)
	}
	msg := err.Error()
	if !strings.Contains(msg, "acme/pattern-arg-pack") {
		t.Errorf("broken-pack error must name the pack, got: %v", err)
	}
	if !strings.Contains(msg, "empty-pattern-rule") {
		t.Errorf("broken-pack error must name the offending rule, got: %v", err)
	}
}

// TestGatherInputs_PatternArgIgnoresRulePath proves a pattern-arg rule with a
// pattern but NO rule_path still gathers inputs successfully and does NOT
// os.Stat a rule path: a (deliberately wrong) rule_path that would fail
// resolveRulePath's on-disk check is ignored, because pattern-arg never resolves
// a file (CLM-018).
func TestGatherInputs_PatternArgIgnoresRulePath(t *testing.T) {
	m := patternArgManifest()
	binding := patternArgBinding()
	// This rule carries a rule_path that does NOT exist on disk. A mode that
	// resolved rule paths would fail loud on the missing file; pattern-arg must
	// ignore it entirely and gather off the pattern.
	rules := []pack.Rule{
		{ID: "todo", Engine: "acme-query", Pattern: "x", RulePath: "rules/does-not-exist.yml"},
	}

	got, err := gatherEngineInputs(m, "/nonexistent/pack/root", binding, rules)
	if err != nil {
		t.Fatalf("pattern-arg must ignore rule_path (no filesystem resolution); a missing rule file must not fail it, got: %v", err)
	}
	want := []string{"--pattern", "x"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected %v gathered off the pattern, got %v", want, got)
	}
}
