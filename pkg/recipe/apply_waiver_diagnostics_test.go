package recipe

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/waiver"
)

// ISSUE-080. The regenerate suite next door proves a divergence is preserved on a
// VALID token and regenerated on none. This file holds the third case those two
// cannot tell apart: a token that does not PARSE.
//
// waiver.Adjudicate already classifies an unparseable reason code as a
// DiagnosticMalformed on Result.Malformed. The defect was that the applier's
// reader returned a bare bool, so the diagnostic was dropped before any decision
// could see it and a malformed token became indistinguishable from no token —
// the operator's edit regenerated over, exit 0, in silence.

// malformedTokenLine is the line divergedWithTokenLine places its token on. The
// diagnostic must report THAT line, not the first line of the file: a reader that
// scanned only the first association window would still find a token here and
// report the wrong coordinate.
const malformedTokenLine = 4

// malformedReasonCode is outside the closed ReasonCode enum
// (pkg/waiver/waiver.go), so ParseToken fails on it and every field of the token
// becomes untrustworthy. It is the live field repro from ISSUE-080.
const malformedReasonCode = "intentional-fork"

// malformedToken and validToken build the two tokens the cases below contrast.
// Both are written by the CONSUMER; the applier authors neither.
func malformedToken(rule string) string {
	return "# @waiver:" + rule + ":" + malformedReasonCode + ":2099-01-01 hand-edited on purpose"
}

func validToken(rule string) string {
	return "# @waiver:" + rule + ":accepted-risk:2099-01-01 divergence accepted by the consumer"
}

// divergedWithTwoTokenLines places first on line 3 and second on line 4 of the
// diverged file, so a file can carry a VALID covering token AND a separate
// malformed one at once. It mirrors divergedWithTokenLine's shape; the covering
// token goes FIRST because coveringWaiverText reports the first matching line,
// and the reported token must be the one that actually accounted for the
// divergence.
func divergedWithTwoTokenLines(first string, second string) string {
	return "# recipe-owned output\nalpha\n" + first + "\n" + second + "\ncharlie\n"
}

// multiRuleRecipe declares THREE enforcement rules over the same output. It is the
// only fixture in the repo that can falsify the accumulation: coveredDivergence
// adjudicates once PER DECLARED RULE, so the same token yields three identical
// Malformed entries unless they are deduped by token identity. Every single-rule
// fixture passes a naive accumulator.
const multiRuleRecipe = `
kind: scaffolding
version: 1.0.0
enforcement:
  rules:
    - recipe.output.divergence
    - recipe.output.second
    - recipe.output.third
ops:
  - id: op-generate
    kind: create
    target: generated/recipe-owned.conf
    payload: body.txt
`

// noEnforcementRecipe declares NO enforcement block, so enforcementRules returns
// nothing and the waiver seam is never consulted. It bounds the blast radius of
// the malformed-token block: a recipe with no rule id to adjudicate against cannot
// be blocked by a token in its output.
const noEnforcementRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-generate
    kind: create
    target: generated/recipe-owned.conf
    payload: body.txt
`

// countingWaiverReader wraps another reader and records the rule of every
// adjudication request. The count is load-bearing in the multi-rule case: the
// dedupe assertion alone would also pass for an applier that only ever consulted
// the FIRST declared rule, which is a different bug with the same symptom.
type countingWaiverReader struct {
	calls    []string
	delegate WaiverReader
}

func (c *countingWaiverReader) read(rule string, file string) DivergenceVerdict {
	c.calls = append(c.calls, rule)
	return c.delegate(rule, file)
}

// malformedDiagnostics filters a result's diagnostics down to the malformed kind.
func malformedDiagnostics(diagnostics []waiver.Diagnostic) []waiver.Diagnostic {
	malformed := make([]waiver.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Kind == waiver.DiagnosticMalformed {
			malformed = append(malformed, diagnostic)
		}
	}
	return malformed
}

// TestApply_WaiverReaderVerdict_CarriesAdjudicationDiagnostics is the TYPE-level
// claim (CLM-001/CLM-002): the seam reports a verdict, not a bool, so "not covered
// because a token was malformed" and "not covered because there was no token"
// reach the decision point as DIFFERENT values.
//
// The two supplied-reader sub-cases differ ONLY in the verdict's Diagnostics —
// Covered is false in both — and produce opposite outcomes. A bool seam cannot
// express that difference, which is what makes this the claim's own falsifier.
//
// The third sub-case drops the stub entirely and drives the PRODUCTION reader,
// because a seam that carries diagnostics no real adjudication ever produces would
// satisfy the first two and still ship the defect.
func TestApply_WaiverReaderVerdict_CarriesAdjudicationDiagnostics(t *testing.T) {
	t.Run("verdict_with_a_malformed_diagnostic_reaches_the_decision", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))
		rule := declaredEnforcementRule(t, resolved)

		diverged := divergedWithTokenLine(malformedToken(rule))
		writeUnder(t, projectRoot, createOp.Target, diverged)

		reader := func(string, string) DivergenceVerdict {
			return DivergenceVerdict{Covered: false, Diagnostics: []waiver.Diagnostic{{
				File:    createOp.Target,
				Line:    malformedTokenLine,
				Message: `waiver: unknown reason-code "` + malformedReasonCode + `"`,
				Kind:    waiver.DiagnosticMalformed,
			}}}
		}

		_, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, ReadWaivers: reader})
		if err == nil {
			t.Fatalf("Apply succeeded; a verdict carrying a malformed diagnostic must fail the apply rather than regenerate over the operator's edit")
		}
		if got := snapshotTree(t, projectRoot)[createOp.Target]; got != diverged {
			t.Errorf("file on disk = %q, want the consumer's bytes untouched %q", got, diverged)
		}
	})

	t.Run("verdict_with_no_diagnostics_regenerates", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))

		writeUnder(t, projectRoot, createOp.Target, divergedWithTokenLine("# no token at all"))

		reader := func(string, string) DivergenceVerdict {
			return DivergenceVerdict{Covered: false}
		}

		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, ReadWaivers: reader})
		if err != nil {
			t.Fatalf("re-apply over an uncovered divergence with no diagnostics: unexpected error: %v", err)
		}
		if got := snapshotTree(t, projectRoot)[createOp.Target]; got != regeneratedPayload {
			t.Errorf("file on disk = %q, want the regenerated payload %q — an uncovered divergence with nothing to report still regenerates", got, regeneratedPayload)
		}
		if len(result.Diagnostics) != 0 {
			t.Errorf("result.Diagnostics = %+v, want none — the verdict reported none", result.Diagnostics)
		}
	})

	t.Run("production_reader_reports_one_diagnostic_per_token", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))
		rule := declaredEnforcementRule(t, resolved)

		// A covering token plus a malformed one, so the apply SUCCEEDS and the
		// diagnostics ride out on the result where they can be counted. The
		// diverged file is five lines and every line becomes a synthesized
		// finding, so an implementation that reported per FINDING rather than per
		// TOKEN would report several here.
		diverged := divergedWithTwoTokenLines(validToken(rule), malformedToken(rule))
		writeUnder(t, projectRoot, createOp.Target, diverged)

		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err != nil {
			t.Fatalf("re-apply over a covered divergence: unexpected error: %v", err)
		}

		malformed := malformedDiagnostics(result.Diagnostics)
		if len(malformed) != 1 {
			t.Fatalf("production reader reported %d malformed diagnostics (%+v), want exactly 1 for the single malformed token", len(malformed), malformed)
		}
		if malformed[0].Line != malformedTokenLine {
			t.Errorf("diagnostic line = %d, want %d — the line the consumer's malformed token actually sits on", malformed[0].Line, malformedTokenLine)
		}
		if !strings.Contains(malformed[0].Message, malformedReasonCode) {
			t.Errorf("diagnostic message = %q, want it to name the illegal reason code %q", malformed[0].Message, malformedReasonCode)
		}
	})
}

// TestApply_DivergedWithMalformedWaiverToken_FailsLoudAndPreservesConsumerBytes is
// the issue's core claim (CLM-003): the apply FAILS and destroys nothing.
//
// The two rejected alternatives, and why the assertions below are shaped to
// exclude them: regenerate-but-warn still destroys the operator's edit (so the
// bytes are asserted byte-for-byte), and preserve-silently would turn a typo into
// a permanent opt-out of regeneration (so the apply is asserted to ERROR rather
// than to succeed quietly).
func TestApply_DivergedWithMalformedWaiverToken_FailsLoudAndPreservesConsumerBytes(t *testing.T) {
	resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))
	rule := declaredEnforcementRule(t, resolved)

	diverged := divergedWithTokenLine(malformedToken(rule))
	writeUnder(t, projectRoot, createOp.Target, diverged)

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err == nil {
		t.Fatalf("Apply succeeded over a divergence carrying a malformed token; a token that does not PARSE cannot be read as absent")
	}

	got := snapshotTree(t, projectRoot)[createOp.Target]
	if got != diverged {
		t.Errorf("file on disk = %q, want the consumer's diverged bytes verbatim %q — a failed apply rewrites nothing", got, diverged)
	}
	if got == regeneratedPayload {
		t.Errorf("the recipe's payload is back on disk; the operator's edit was destroyed by the very apply that refused to accept the token")
	}

	var zero ApplyResult
	if len(result.Written) != len(zero.Written) || len(result.Preserved) != len(zero.Preserved) || len(result.Regenerated) != len(zero.Regenerated) || len(result.Diagnostics) != len(zero.Diagnostics) {
		t.Errorf("failed Apply returned %+v, want the ZERO result — an apply either produces a verdict or it fails, never both", result)
	}
}

// TestApply_DivergedWithMalformedWaiverToken_DiagnosticNamesFileLineAndReasonCode
// holds the diagnostic's CONTENT (CLM-003). A bare "apply failed" would be the
// same silence in a different costume: the operator has to be told which file,
// which line, and which reason code so they can take one of the two honest exits.
//
// The target is asserted AS REPORTED — the substituted declared target the
// operator sees on disk, never a raw {{ … }} declaration (ISSUE-079).
func TestApply_DivergedWithMalformedWaiverToken_DiagnosticNamesFileLineAndReasonCode(t *testing.T) {
	resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))
	rule := declaredEnforcementRule(t, resolved)

	tokenLine := malformedToken(rule)
	writeUnder(t, projectRoot, createOp.Target, divergedWithTokenLine(tokenLine))

	_, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err == nil {
		t.Fatalf("Apply succeeded over a malformed token; there is no diagnostic to assert on")
	}
	message := err.Error()

	if !strings.Contains(message, createOp.Target) {
		t.Errorf("diagnostic %q does not name the reported target %q", message, createOp.Target)
	}
	if !strings.Contains(message, fmt.Sprintf("line %d", malformedTokenLine)) {
		t.Errorf("diagnostic %q does not name line %d, where the consumer's token sits", message, malformedTokenLine)
	}
	// Read the reason code back off the token the test wrote rather than retyping
	// it, and assert on the CODE rather than pkg/waiver's whole sentence, so a
	// wording change there does not falsely red this.
	if !strings.Contains(message, malformedReasonCode) {
		t.Errorf("diagnostic %q does not name the illegal reason code %q from the token %q", message, malformedReasonCode, tokenLine)
	}
}

// TestApply_CoveredDivergenceWithSeparateMalformedToken_PreservesAndReportsDiagnostic
// holds the independence of coverage and hygiene (CLM-012).
//
// Getting this wrong reproduces ISSUE-080's own defect one `if` later: an applier
// that returns as soon as a rule covers, before the diagnostics are collected,
// drops the malformed token exactly as the bool seam did. Blocking instead was
// rejected too — the operator DID account for the divergence with a valid token,
// and revoking that over an unrelated typo punishes the accountable path.
//
// The falsifying twin is in this test on purpose: without it, an implementation
// that stuffed every adjudication artifact into Diagnostics would pass.
func TestApply_CoveredDivergenceWithSeparateMalformedToken_PreservesAndReportsDiagnostic(t *testing.T) {
	t.Run("covered_and_malformed_preserves_and_reports", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))
		rule := declaredEnforcementRule(t, resolved)

		diverged := divergedWithTwoTokenLines(validToken(rule), malformedToken(rule))
		writeUnder(t, projectRoot, createOp.Target, diverged)

		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err != nil {
			t.Fatalf("Apply failed over a divergence a VALID token covers: %v — an unrelated malformed token must not revoke an accountable divergence", err)
		}

		if got := snapshotTree(t, projectRoot)[createOp.Target]; got != diverged {
			t.Errorf("file on disk = %q, want the consumer's bytes preserved verbatim %q", got, diverged)
		}
		preserved, reported := preservedFor(result, createOp.Target)
		if !reported {
			t.Fatalf("result.Preserved = %+v, want an entry for the covered divergence %q", result.Preserved, createOp.Target)
		}
		if preserved.Rule != rule {
			t.Errorf("preserved.Rule = %q, want the recipe's declared enforcement rule %q", preserved.Rule, rule)
		}
		if !strings.Contains(preserved.CoveringWaiver, ":accepted-risk:") {
			t.Errorf("preserved.CoveringWaiver = %q, want the VALID token that accounted for the divergence, not the malformed one", preserved.CoveringWaiver)
		}

		malformed := malformedDiagnostics(result.Diagnostics)
		if len(malformed) != 1 {
			t.Fatalf("result reported %d malformed diagnostics (%+v), want exactly 1 — the covered branch must not drop what adjudication found", len(malformed), malformed)
		}
		if malformed[0].Line != malformedTokenLine {
			t.Errorf("diagnostic line = %d, want %d — the malformed token's own line, not the covering token's", malformed[0].Line, malformedTokenLine)
		}
		if !strings.Contains(malformed[0].Message, malformedReasonCode) {
			t.Errorf("diagnostic message = %q, want it to name the illegal reason code %q", malformed[0].Message, malformedReasonCode)
		}
	})

	t.Run("covered_with_no_malformed_token_reports_nothing", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))
		rule := declaredEnforcementRule(t, resolved)

		// The identical divergence minus the malformed token. If this reported a
		// diagnostic, the assertion above would be satisfied by an applier that
		// reports something on every covered divergence, which proves nothing.
		diverged := divergedWithTwoTokenLines(validToken(rule), "# an ordinary comment, no token")
		writeUnder(t, projectRoot, createOp.Target, diverged)

		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err != nil {
			t.Fatalf("Apply failed over a cleanly covered divergence: %v", err)
		}
		if got := snapshotTree(t, projectRoot)[createOp.Target]; got != diverged {
			t.Errorf("file on disk = %q, want the consumer's bytes preserved verbatim %q", got, diverged)
		}
		if len(result.Diagnostics) != 0 {
			t.Errorf("result.Diagnostics = %+v, want NONE — there is no token hygiene problem in this file", result.Diagnostics)
		}
	})
}

// TestApply_MultipleEnforcementRules_MalformedTokenReportedOnce holds the
// cross-rule dedupe (CLM-013).
//
// coveredDivergence adjudicates the file once per DECLARED rule, and each
// adjudication re-discovers the same token, so three declared rules produce three
// identical Malformed entries unless the accumulation dedupes by token identity
// {File, Line, Message}. RuleID is deliberately NOT part of that key: it is the
// one field that legitimately DIFFERS between the three, because each adjudication
// was run against a different rule's findings.
//
// The invocation count is the second half and is not redundant: the dedupe
// assertion alone also passes for an applier that consults only the first declared
// rule, which would silently stop honoring the other two.
func TestApply_MultipleEnforcementRules_MalformedTokenReportedOnce(t *testing.T) {
	resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, multiRuleRecipe)

	declaredRules := resolved.Manifest.Enforcement.Rules
	if len(declaredRules) < 2 {
		t.Fatalf("fixture declares %v; the dedupe is unfalsifiable below two rules", declaredRules)
	}

	// One token, naming the FIRST declared rule. It is malformed, so it covers
	// nothing and every declared rule is consulted.
	writeUnder(t, projectRoot, createOp.Target, divergedWithTokenLine(malformedToken(declaredRules[0])))

	counting := &countingWaiverReader{delegate: adjudicateDivergence}
	_, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, ReadWaivers: counting.read})
	if err == nil {
		t.Fatalf("Apply succeeded over a malformed token; there is no reported diagnostic set to count")
	}

	if len(counting.calls) != len(declaredRules) {
		t.Errorf("the seam was consulted %d times (%v), want once per declared rule (%v) — an applier that stops at the first rule silently ignores the rest", len(counting.calls), counting.calls, declaredRules)
	}

	// The failure names ONE token, not one per rule: the diagnostic set the error
	// was built from collapsed to a single entry.
	message := err.Error()
	if occurrences := strings.Count(message, malformedReasonCode); occurrences != 1 {
		t.Errorf("the diagnostic names the reason code %d times in %q, want exactly 1 — %d declared rules re-adjudicate the SAME token and must collapse to one report", occurrences, message, len(declaredRules))
	}
}

// TestApply_MalformedTokenForAnotherRule_AlsoBlocks pins a DELIBERATE asymmetry
// (CLM-005), so it cannot be quietly "fixed" into symmetry later.
//
// A VALID token naming another rule does not preserve, because it PARSED — we can
// read its rule id and know it is not about us. A MALFORMED token declares no
// trustworthy rule at all: ParseToken fails before any field is credible, so it
// cannot be dismissed as someone else's business. The contrast is the test.
func TestApply_MalformedTokenForAnotherRule_AlsoBlocks(t *testing.T) {
	const otherRule = "some.other.rule"

	t.Run("malformed_token_for_another_rule_blocks", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))

		diverged := divergedWithTokenLine(malformedToken(otherRule))
		writeUnder(t, projectRoot, createOp.Target, diverged)

		_, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err == nil {
			t.Fatalf("Apply succeeded; an unparseable token names no rule we can trust, so it cannot be dismissed as another rule's")
		}
		if got := snapshotTree(t, projectRoot)[createOp.Target]; got != diverged {
			t.Errorf("file on disk = %q, want the consumer's bytes untouched %q", got, diverged)
		}
	})

	t.Run("valid_token_for_another_rule_still_regenerates", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))

		writeUnder(t, projectRoot, createOp.Target, divergedWithTokenLine("# @waiver:"+otherRule+":accepted-risk:2099-01-01 waives a different rule entirely"))

		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err != nil {
			t.Fatalf("Apply failed over a VALID wrong-rule token: %v — it parsed, so it is knowably not about this recipe's rule", err)
		}
		if got := snapshotTree(t, projectRoot)[createOp.Target]; got != regeneratedPayload {
			t.Errorf("file on disk = %q, want the regenerated payload %q — a valid token for another rule does not cover this divergence", got, regeneratedPayload)
		}
		if preserved, reported := preservedFor(result, createOp.Target); reported {
			t.Errorf("result reports %q as preserved (%+v); this token covers a different rule", createOp.Target, preserved)
		}
	})
}

// TestApply_NoEnforcementRules_MalformedTokenCannotBlock bounds the blast radius of
// the new block (CLM-006). A recipe declaring no enforcement rule has no rule id a
// consumer could waive against, so there is nothing to adjudicate and the seam is
// never consulted — a malformed token in its output is simply invisible.
//
// Without this bound, adding the block would start failing applies for recipes
// that never opted into divergence adjudication at all.
func TestApply_NoEnforcementRules_MalformedTokenCannotBlock(t *testing.T) {
	resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, noEnforcementRecipe)

	if len(enforcementRules(resolved)) != 0 {
		t.Fatalf("fixture declares enforcement rules %v; this case requires none", enforcementRules(resolved))
	}

	writeUnder(t, projectRoot, createOp.Target, divergedWithTokenLine(malformedToken("recipe.output.divergence")))

	counting := &countingWaiverReader{delegate: adjudicateDivergence}
	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, ReadWaivers: counting.read})
	if err != nil {
		t.Fatalf("Apply failed for a recipe with no declared enforcement rule: %v — there was nothing to adjudicate the token against", err)
	}
	if len(counting.calls) != 0 {
		t.Errorf("the waiver seam was consulted %v; a recipe declaring no enforcement rule adjudicates nothing", counting.calls)
	}
	if got := snapshotTree(t, projectRoot)[createOp.Target]; got != regeneratedPayload {
		t.Errorf("file on disk = %q, want the regenerated payload %q — this path is unchanged by ISSUE-080", got, regeneratedPayload)
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("result.Diagnostics = %+v, want none — nothing was adjudicated", result.Diagnostics)
	}
}

// TestApply_DivergedNoWaiver_RecordsRegeneratedOverDivergence holds the reporting
// data the CLI reads (CLM-008). Regenerated is a strict SUBSET of Written in the
// SAME value form, so the CLI can mark a clobber without re-deriving the
// distinction — and the unchanged-file half proves the two lists are not aliases.
func TestApply_DivergedNoWaiver_RecordsRegeneratedOverDivergence(t *testing.T) {
	t.Run("a_write_over_nothing_is_not_a_regeneration", func(t *testing.T) {
		// adoptRecipeOwnedOutput performs the FIRST apply onto an empty project
		// root: the target was WRITTEN, but nothing existed to diverge from, so
		// nothing was overwritten.
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))

		// Re-apply with the file UNCHANGED: still no divergence and still nothing
		// clobbered.
		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err != nil {
			t.Fatalf("re-apply over an unchanged file: unexpected error: %v", err)
		}
		if len(result.Regenerated) != 0 {
			t.Errorf("result.Regenerated = %v, want empty — %q holds exactly the bytes the recipe declares", result.Regenerated, createOp.Target)
		}
	})

	t.Run("an_uncovered_divergence_is_written_and_regenerated", func(t *testing.T) {
		resolved, projectRoot, createOp := adoptRecipeOwnedOutput(t, fmt.Sprintf(regenerateRecipe, KindScaffolding))

		writeUnder(t, projectRoot, createOp.Target, divergedWithTokenLine("# a divergence with no waiver at all"))

		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err != nil {
			t.Fatalf("re-apply over an uncovered divergence: unexpected error: %v", err)
		}
		if got := snapshotTree(t, projectRoot)[createOp.Target]; got != regeneratedPayload {
			t.Errorf("file on disk = %q, want the regenerated payload %q", got, regeneratedPayload)
		}

		if len(result.Regenerated) != 1 || result.Regenerated[0] != createOp.Target {
			t.Fatalf("result.Regenerated = %v, want exactly [%q]", result.Regenerated, createOp.Target)
		}

		// The SUBSET property: every Regenerated entry must appear in Written,
		// compared by value. A Regenerated list carrying a raw declaration while
		// Written carries a substituted path would break this silently, and only
		// for templated targets.
		for _, regenerated := range result.Regenerated {
			var inWritten bool
			for _, written := range result.Written {
				if written == regenerated {
					inWritten = true
				}
			}
			if !inWritten {
				t.Errorf("result.Regenerated entry %q is absent from result.Written %v; Regenerated is a strict subset in the same value form", regenerated, result.Written)
			}
		}
	})
}
