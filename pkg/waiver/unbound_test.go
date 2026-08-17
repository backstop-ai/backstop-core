package waiver

import (
	"strings"
	"testing"
)

// unbound_test.go — ISSUE-097. Adjudicate NEVER scans the tree: it reads only the
// two-line association window of each finding it was HANDED, so a token on a line
// where its own rule no longer fires is never harvested at all and cannot even reach
// the Unused bucket. These lock the tree-driven half that has no such dependency.
//
// Every fixture here is SYNTHETIC and in-memory. After the live tree is re-keyed the
// repository holds zero unbound tokens, so a test measuring the real repo would be
// vacuously green forever; exactly one test in this lane reads the real tree and it
// lives in cmd/backstop.

// staleRuleID is the pre-rename rule id both cmd/backstop tokens carried: a real
// three-segment pack-namespaced id whose <org>/<pack> half no longer names any pack.
const staleRuleID = "backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine"

// currentRuleID is the same rule under the post-rename namespace.
const currentRuleID = "backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules.no-structural-name-split-on-spine"

// mark prefixes a fixture body with the waiver marker AT RUNTIME, so this file's own
// source carries no complete literal token.
//
// THAT IS NOT COSMETIC. The production harvest byte-scans the whole repository, this
// file included, so a fully-literal fixture token here is a REAL token in the tree: an
// earlier revision planted `a/b/first`, and TestRepo_CarriesNoUnboundWaiverTokens
// correctly reported six unbound tokens instead of five. Assembling the marker keeps the
// fixture a fixture.
func mark(body string) string { return "@" + "waiver:" + body }

// TestWaiver_Unbound_FlagsTokenWhosePackNamespaceIsNotInstalled is the falsification
// (CLM-004): a token whose extracted <org>/<pack> matches no known namespace yields
// exactly one unbound Diagnostic, carrying the FULL rule-id (the string the reader
// must edit — the extracted pack name alone is not editable text) and the token's own
// file and line so the warning is navigable.
func TestWaiver_Unbound_FlagsTokenWhosePackNamespaceIsNotInstalled(t *testing.T) {
	lines := []string{
		"package main",
		"// " + mark(staleRuleID+":false-positive:2027-07-17 legitimate tokenization"),
		"func splitCommand(s string) []string { return nil }",
	}
	tokens := HarvestTokens("cmd/backstop/pack_gate.go", lines)
	if len(tokens) != 1 {
		t.Fatalf("expected the harvest to return the single well-formed token, got %d", len(tokens))
	}

	diags := Unbound(tokens, []string{"backstop-ai/backstop-self"})
	if len(diags) != 1 {
		t.Fatalf("a token keyed to a pack namespace no lock records must yield exactly one "+
			"diagnostic, got %d (%#v)", len(diags), diags)
	}
	d := diags[0]
	if d.Kind != DiagnosticUnbound {
		t.Errorf("diagnostic Kind = %q, want %q", d.Kind, DiagnosticUnbound)
	}
	if d.RuleID != staleRuleID {
		t.Errorf("diagnostic RuleID = %q, want the token's FULL rule-id %q; the extracted pack "+
			"name alone is not the string the reader has to edit", d.RuleID, staleRuleID)
	}
	if d.File != "cmd/backstop/pack_gate.go" {
		t.Errorf("diagnostic File = %q, want the token's own file", d.File)
	}
	if d.Line != 2 {
		t.Errorf("diagnostic Line = %d, want 2 (the token's own 1-indexed line); an unnavigable "+
			"warning is one nobody acts on", d.Line)
	}
	if !strings.Contains(d.Message, "backstop/self") {
		t.Errorf("the message must name the unresolvable pack namespace so the reader knows what "+
			"is wrong, got %q", d.Message)
	}
	if !strings.Contains(d.Message, "backstop.lock") {
		t.Errorf("the message must name backstop.lock as the authority it was checked against, "+
			"got %q", d.Message)
	}
}

// TestWaiver_Unbound_BindsTokenWhosePackNamespaceIsInstalled keeps the test above
// honest (CLM-004). Without this leg a function returning one diagnostic per token
// unconditionally would satisfy the falsification.
func TestWaiver_Unbound_BindsTokenWhosePackNamespaceIsInstalled(t *testing.T) {
	lines := []string{
		"// " + mark(currentRuleID+":false-positive:2027-07-17 legitimate tokenization"),
	}
	tokens := HarvestTokens("cmd/backstop/pack_gate.go", lines)
	if len(tokens) != 1 {
		t.Fatalf("expected one harvested token, got %d", len(tokens))
	}

	if diags := Unbound(tokens, []string{"backstop-ai/backstop-self"}); len(diags) != 0 {
		t.Fatalf("a token keyed to a namespace the lock DOES record must yield no diagnostics, "+
			"got %d (%#v)", len(diags), diags)
	}
}

// TestWaiver_Unbound_ExemptsRuleIDWithoutExtractablePackName is filter F3 (CLM-005): a
// rule id with fewer than three `/`-separated segments carries no extractable pack
// name and is UNCLASSIFIABLE, not unbound.
//
// The ids below were measured as real F3 removals in this repository, not invented:
// without this filter the whole-tree scan reports 26 candidates of which 5 are real,
// and a warning at a 1-in-5 signal ratio is silent green wearing a costume.
func TestWaiver_Unbound_ExemptsRuleIDWithoutExtractablePackName(t *testing.T) {
	cases := []struct {
		name   string
		ruleID string
	}{
		{"no slash at all — a real core dimension id", "coverage_threshold"},
		{"two segments", "pkg-rule"},
		{"single letter", "r"},
		{"two segments, hyphenated", "mystery-rule"},
		{"dotted but slash-free", "some.other.rule"},
	}
	namespaces := []string{"backstop-ai/backstop-self", "backstop-ai/go-standards"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := []string{"// " + mark(tc.ruleID+":deferred:2999-01-01 note")}
			tokens := HarvestTokens("pkg/thing/thing.go", lines)
			if len(tokens) != 1 {
				t.Fatalf("expected the fixture token to parse, got %d tokens", len(tokens))
			}
			if diags := Unbound(tokens, namespaces); len(diags) != 0 {
				t.Fatalf("rule id %q carries no extractable <org>/<pack> name and must be SKIPPED, "+
					"not flagged; got %d diagnostics (%#v)", tc.ruleID, len(diags), diags)
			}
		})
	}
}

// TestWaiver_Unbound_EmptyNamespaceSetYieldsNoDiagnostics is SE3 (CLM-006). "No
// namespaces known" is indistinguishable from "no pack is legitimate", and the
// fail-loud reading flags every pack-namespaced waiver in the repository on any tree
// whose lock is missing or unreadable.
func TestWaiver_Unbound_EmptyNamespaceSetYieldsNoDiagnostics(t *testing.T) {
	lines := []string{"// " + mark(staleRuleID+":false-positive:2027-07-17 note")}
	tokens := HarvestTokens("cmd/backstop/pack_gate.go", lines)
	if len(tokens) != 1 {
		t.Fatalf("expected one harvested token, got %d", len(tokens))
	}

	// The premise: this exact token DOES produce a diagnostic under a populated list,
	// so a zero result below proves the empty-set guard fired rather than that the
	// token happened to be unclassifiable.
	if diags := Unbound(tokens, []string{"backstop-ai/backstop-self"}); len(diags) != 1 {
		t.Fatalf("premise broken: the fixture token must be flaggable under a populated "+
			"namespace list, got %d diagnostics", len(diags))
	}

	if diags := Unbound(tokens, nil); len(diags) != 0 {
		t.Errorf("a nil namespace slice must yield ZERO diagnostics, got %d (%#v); flagging "+
			"everything on a tree with no lock is a false-positive storm on a supported state",
			len(diags), diags)
	}
	if diags := Unbound(tokens, []string{}); len(diags) != 0 {
		t.Errorf("an empty non-nil namespace slice must yield ZERO diagnostics, got %d (%#v)",
			len(diags), diags)
	}
}

// TestWaiver_HarvestTokens_ReadsTokenWithNoFindingPresent states the architectural
// break as a test (CLM-007): the harvest byte-scans raw lines for the literal marker
// with NO Finding ever constructed, which is what makes the whole check independent of
// whether a rule currently fires at the token's location.
func TestWaiver_HarvestTokens_ReadsTokenWithNoFindingPresent(t *testing.T) {
	lines := []string{
		"package smoke",
		"\t// " + mark("backstop-ai/backstop-self/some.rules.no-baked-tool-exec:deferred:2026-10-24 escalated, see ISSUE-097"),
		"// two on one line: " + mark("a/b/first:accepted-risk:2999-01-01") + " " + mark("a/b/second:third-party:2999-06-30"),
		"// " + mark("a/b/broken:not-a-reason-code:2999-01-01 malformed and must be skipped"),
	}
	tokens := HarvestTokens("tests/smoke/smoke_test.go", lines)

	if len(tokens) != 3 {
		t.Fatalf("expected 3 well-formed tokens (one on line 2, two on line 3, the malformed "+
			"line-4 token skipped), got %d: %#v", len(tokens), tokens)
	}

	first := tokens[0]
	if first.RuleID != "backstop-ai/backstop-self/some.rules.no-baked-tool-exec" {
		t.Errorf("RuleID = %q, want it parsed from the raw bytes", first.RuleID)
	}
	if first.Reason != ReasonDeferred {
		t.Errorf("Reason = %q, want %q", first.Reason, ReasonDeferred)
	}
	if got := first.Expiry.Format("2006-01-02"); got != "2026-10-24" {
		t.Errorf("Expiry = %q, want 2026-10-24", got)
	}
	if first.Line != 2 {
		t.Errorf("Line = %d, want 2 — lines are 1-indexed, matching every other location in "+
			"this package", first.Line)
	}
	if first.File != "tests/smoke/smoke_test.go" {
		t.Errorf("File = %q, want the file it was handed", first.File)
	}

	// Multi-occurrence scanning: both tokens on line 3 are returned, in column order.
	if tokens[1].RuleID != "a/b/first" || tokens[2].RuleID != "a/b/second" {
		t.Errorf("both tokens on a shared line must be returned in column order, got %q then %q",
			tokens[1].RuleID, tokens[2].RuleID)
	}
	if tokens[1].Line != 3 || tokens[2].Line != 3 {
		t.Errorf("both line-3 tokens must report line 3, got %d and %d", tokens[1].Line, tokens[2].Line)
	}

	// The malformed token is SKIPPED, not returned half-parsed: Adjudicate owns
	// malformed reporting and duplicating it here would double-report.
	for _, tok := range tokens {
		if strings.Contains(tok.RuleID, "broken") {
			t.Errorf("a malformed token must be skipped, not returned; got %#v", tok)
		}
	}
}
