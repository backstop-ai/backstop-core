package waiver

import "strings"

// unbound.go — ISSUE-097. The TREE-DRIVEN half of waiver inspection.
//
// Adjudicate is finding-driven: it byte-scans only the two-line association window of
// each finding it was HANDED, so a token sitting where its own rule no longer fires is
// never read into its token map at all — it cannot even reach the Unused bucket, which
// classifies only tokens that WERE harvested. A waiver orphaned by a pack rename is
// therefore architecturally invisible, not merely unreported: it stops suppressing the
// moment its target rule would otherwise fire, and nothing warns.
//
// These two functions are the tree-driven scan that has no such dependency. They are
// ADDITIVE and sit beside Adjudicate — suppression semantics are untouched. The
// zero-baked property is preserved unchanged: raw bytes only, no language identifier,
// no comment lexer, reusing the same scanOccurrences and ParseToken the suppression
// path uses.

// DiagnosticUnbound marks a well-formed token whose pack namespace matches no pack the
// project's lock records — dead config that suppresses nothing and says nothing.
const DiagnosticUnbound = "unbound"

// packNameSegments is how many `/`-separated segments a rule id must have before an
// <org>/<pack> name can be read off its head. A pack-namespaced id is built as
// <org>/<pack> + "/" + the engine's own rule id, so two segments name a pack and the
// third is the rule. Fewer than three means the id carries no pack name at all and is
// UNCLASSIFIABLE, not unbound.
const packNameSegments = 3

// HarvestTokens returns every well-formed waiver token in a file's raw lines, with NO
// Finding involved — the architectural break from Adjudicate's finding-driven window.
// lines is the file's content in order; the returned Line is 1-indexed, matching every
// other location in this package.
//
// Malformed tokens are SKIPPED rather than returned half-parsed: Adjudicate owns
// malformed reporting, and reporting them here too would double-report the ones it
// already sees. Tree-wide malformed detection is a real residual gap, tracked
// separately.
func HarvestTokens(file string, lines []string) []Waiver {
	var out []Waiver
	for i, text := range lines {
		for _, col := range scanOccurrences(text) {
			w, err := ParseToken(text[col:], Location{File: file, Line: i + 1})
			if err != nil {
				continue
			}
			out = append(out, w)
		}
	}
	return out
}

// Unbound returns one diagnostic per harvested token whose extracted <org>/<pack> name
// appears in none of namespaces, preserving input order so the rendered gate string is
// byte-stable across runs.
//
// An EMPTY namespace set yields nothing. "No namespaces known" is indistinguishable
// from "no pack is legitimate", and the fail-loud reading would flag every
// pack-namespaced waiver in a repository whose lock is simply absent or unreadable — a
// false-positive storm triggered by a supported state.
func Unbound(tokens []Waiver, namespaces []string) []Diagnostic {
	if len(namespaces) == 0 {
		return nil
	}
	known := make(map[string]struct{}, len(namespaces))
	for _, ns := range namespaces {
		known[ns] = struct{}{}
	}

	var out []Diagnostic
	for _, tok := range tokens {
		name, ok := packNameOf(tok.RuleID)
		if !ok {
			continue
		}
		if _, bound := known[name]; bound {
			continue
		}
		out = append(out, Diagnostic{
			RuleID:  tok.RuleID,
			File:    tok.File,
			Line:    tok.Line,
			Kind:    DiagnosticUnbound,
			Message: "waiver token names pack " + name + ", which matches no pack in backstop.lock; it suppresses nothing until it is re-keyed to a recorded pack or removed",
		})
	}
	return out
}

// packNameOf reads the <org>/<pack> name off the head of a namespaced rule id, and
// reports ok=false when the id carries no such name.
func packNameOf(ruleID string) (string, bool) {
	segments := strings.Split(ruleID, "/")
	if len(segments) < packNameSegments {
		return "", false
	}
	return segments[0] + "/" + segments[1], true
}
