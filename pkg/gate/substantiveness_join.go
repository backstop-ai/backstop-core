package gate

import (
	"path/filepath"
	"strings"
)

// substantiveness_join.go is the language-agnostic, gate-side half of the
// SPEC-037 substantiveness pack split (BUNDLE-009 Seed 3). The PACK does the
// language-specific work (Q1 hollow-test findings rule + Q2 referenced-symbol
// extraction rule); this file consumes the FLAT pack-dispatch SARIF
// ([]Violation with only {Rule, File, Message, Severity, SourcePack} — Line/region
// dropped, no GateType field) and performs:
//   - routing substantiveness findings out of the flat pack_engines stream by
//     NAMESPACED rule ID (RouteSubstantivenessFindings),
//   - keying extraction findings back to a MandatedTest by (FilePath, FuncName)
//     parsed from the SARIF Message (ReferencedSetForTest),
//   - the noTarget SET-JOIN decision table (NoTargetViolation),
//   - the relocated, behavior-preserving target-package derivation
//     (TargetPackageName).
// It re-bakes NO go/parser / go/ast: routing and keying are string/set operations
// over pack SARIF, and the noTarget verdict is a pure set-membership test.

// ReferencedSymbolSet is the set of package/symbol names a single test references,
// assembled from the Q2 extraction findings keyed to that test.
type ReferencedSymbolSet map[string]bool

// TargetPackageName reduces a declared subject to the OPAQUE token the noTarget
// set-join checks membership against. Core bakes in NO layout knowledge (ISSUE-047):
// there is no "cmd/"/"pkg/" prefix literal and no layout special-case. A subject is
// an opaque path-or-bare token; this reduces it to its trailing `/`-segment via the
// SAME language-neutral last-segment op testFileColocatedWithTarget uses on the
// test-file side (filepath.Base of the directory leaf) — a bare token passes through
// unchanged, and `cmd/foo`/`internal/foo`/`pkg/foo` all reduce symmetrically to
// `foo`. The ONLY special case is the EMPTY subject, which returns "" (the
// empty-target input the set-join SKIPS — CLM-009): filepath.Base("") returns "."
// (not ""), so the empty subject is intercepted BEFORE Base to preserve that skip.
func TargetPackageName(subject string) string {
	if subject == "" {
		return ""
	}
	return filepath.Base(subject)
}

// NoTargetViolation is the exhaustive, language-agnostic noTarget set-join decision
// table (REQ-003). It returns (violation, true) only for a genuine noTarget; the
// three satisfied/skipped dispositions return (Violation{}, false). No fourth
// disposition exists, and the verdict is a pure set/string test — never a re-baked
// AST analysis (Sharp Edge 1/2).
//
//	targetPkg == ""                          → SKIP (no violation)
//	targetPkg != "" && samePackage           → satisfied (no violation)
//	targetPkg != "" && !samePackage && in set → satisfied (no violation)
//	targetPkg != "" && !samePackage && !in set → noTarget violation
func NoTargetViolation(funcName, targetPkg string, referenced ReferencedSymbolSet, samePackage bool) (Violation, bool) {
	if targetPkg == "" {
		return Violation{}, false
	}
	if samePackage {
		return Violation{}, false
	}
	if referenced[targetPkg] {
		return Violation{}, false
	}
	return Violation{
		Rule:     StepTestSubstantiveness,
		Message:  "test function " + funcName + " does not call package " + targetPkg,
		Severity: "error",
	}, true
}

// NoTargetViolationForTest is the thin PRE-JOIN skip wrapper that applies the opt-in
// `kind: absence` annotation (ISSUE-035 Category 2) WITHOUT altering NoTargetViolation's
// exhaustive decision table or its strangler-harness caller. When the mandated test's
// claim declared `kind: absence` (mt.IsAbsence), the noTarget set-join is SKIPPED (the
// test proves an ABSENCE, so by design it does not call its target package). Otherwise
// it delegates unchanged to NoTargetViolation, so the full decision table — and the
// false-negative guard (an UNannotated not-in-set test STILL raises) — stays intact and
// unit-testable here in pkg/gate. The skip mirrors how terminal specs are pre-filtered
// in ExtractMandatedTests and how the gate loop already `continue`s on unfound tests.
func NoTargetViolationForTest(mt MandatedTest, referenced ReferencedSymbolSet, samePackage bool) (Violation, bool) {
	if mt.IsAbsence {
		return Violation{}, false
	}
	return NoTargetViolation(mt.FuncName, mt.TargetPkg, referenced, samePackage)
}

// RouteSubstantivenessFindings partitions the FLAT pack_engines []Violation stream
// into the substantiveness hollow-findings and extraction-findings by matching the
// pack's stable NAMESPACED rule IDs (pack.NamespacedRuleID form) on each violation's
// Rule. All other pack rules are ignored. NO gate_type field is consulted — the
// Violation carries none (Sharp Edge 5 / REQ-007 / CLM-024).
func RouteSubstantivenessFindings(violations []Violation, hollowRuleID, extractionRuleID string) (hollow, extraction []Violation) {
	for _, v := range violations {
		switch v.Rule {
		case hollowRuleID:
			hollow = append(hollow, v)
		case extractionRuleID:
			extraction = append(extraction, v)
		}
	}
	return hollow, extraction
}

// ReferencedSetForTest joins the routed Q2 extraction findings to a MandatedTest by
// (FilePath, FuncName) and assembles that test's ReferencedSymbolSet. FuncName and the
// referenced symbol are parsed from each finding's PINNED Message
// ("referenced-symbol func=<TestFuncName> symbol=<pkg>") — NOT from Line/region
// (dropped by the flat conversion) and NOT by re-walking the test AST (Sharp Edge 2).
// A test with no matching finding yields an empty set, which the decision table
// handles unchanged (CLM-025/CLM-026).
func ReferencedSetForTest(extraction []Violation, test MandatedTest) ReferencedSymbolSet {
	set := ReferencedSymbolSet{}
	for _, v := range extraction {
		if !sameFile(v.File, test.FilePath) {
			continue
		}
		fn, symbol := parseExtractionMessage(v.Message)
		if fn != test.FuncName {
			continue
		}
		if symbol != "" {
			set[symbol] = true
		}
	}
	return set
}

// IsTestHollow turns the routed hollow partition into the per-test hollow verdict
// the substantiveness step raises: it maps each routed hollow finding to its mandated
// test by (File, FuncName parsed from the finding's PINNED `func=<FN>` Message) and
// reports whether the given test is hollow. Language-agnostic — Go and TS hollow
// findings flow identically through the same `func=` shape (CLM-003/004/012/013/014).
func IsTestHollow(hollow []Violation, test MandatedTest) bool {
	for _, v := range hollow {
		if !sameFile(v.File, test.FilePath) {
			continue
		}
		if funcNameFromMessage(v.Message) == test.FuncName {
			return true
		}
	}
	return false
}

// HollowFindingsToViolations converts each routed hollow finding into one
// test_substantiveness Violation, preserving the deleted analyzer's
// "test X has no assertions (hollow)" report-surface message (CLM-005). The hollow
// rule already embeds that message text (plus the pinned `func=<FN>` key the gate uses
// for routing), so the conversion forwards the finding's File + Message under the
// test_substantiveness rule name — one violation per hollow finding. The Violation
// carries no gate_type field (none exists — Sharp Edge 5).
func HollowFindingsToViolations(hollow []Violation) []Violation {
	out := make([]Violation, 0, len(hollow))
	for _, v := range hollow {
		out = append(out, Violation{
			Rule: StepTestSubstantiveness,
			// Canonical repo-relative File (ISSUE-046). This join has no ProjectRoot
			// threaded; the carried finding's File is a pack-reported repo-relative
			// path, so NormalizePath("", …) — the idempotent Clean+ToSlash+strip-"./"
			// subset of the SINGLE helper — canonicalizes it. Any absolute path is
			// handled by the Phase-1 identity chokepoint.
			File:     NormalizePath("", v.File),
			Message:  stripFuncToken(v.Message),
			Severity: "error",
		})
	}
	return out
}

// stripFuncToken removes the trailing pinned ` func=<FN>` routing token (and any
// ` symbol=<pkg>` token) from a report message so the user-facing report surface keeps
// the clean "test X has no assertions (hollow)" semantics while the raw finding still
// carries the gate's routing key. Whitespace before the first pinned token is trimmed.
func stripFuncToken(message string) string {
	for _, key := range []string{" func=", " symbol="} {
		if idx := strings.Index(message, key); idx >= 0 {
			message = message[:idx]
		}
	}
	return strings.TrimRight(message, " \t")
}

// funcNameFromMessage parses the enclosing test function name out of a routed
// finding's PINNED message (both the hollow and extraction rules embed `func=<FN>`).
// It is the gate-side per-test key that replaces the dropped Line/region identity.
func funcNameFromMessage(message string) string {
	// parseExtractionMessage returns (funcName, symbol); the discarded symbol is a
	// string, not an error.
	fn, _ := parseExtractionMessage(message) // nosemgrep: go.core.no-ignored-errors — discards a string, not an error
	return fn
}

// parseExtractionMessage extracts the func= and symbol= tokens from a pinned
// substantiveness SARIF message. Both tokens are whitespace-delimited; a missing
// token yields "". This is a pure string operation over pack SARIF.
func parseExtractionMessage(message string) (funcName, symbol string) {
	return tokenValue(message, "func="), tokenValue(message, "symbol=")
}

// tokenValue returns the whitespace-delimited value following the first occurrence
// of key (e.g. "func=") in s, or "" if absent.
func tokenValue(s, key string) string {
	idx := strings.Index(s, key)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(key):]
	if end := strings.IndexAny(rest, " \t\n"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// sameFile compares two file paths for the (FilePath, FuncName) join. Extraction
// findings carry the file as reported by the pack (which scans a fixture/scope dir),
// while a MandatedTest carries the resolved path; comparing by cleaned base-aware
// suffix keeps the join robust to absolute-vs-relative path shape without re-walking
// the filesystem. An exact match (after Clean) is preferred; otherwise a basename
// match is the fallback, since extraction findings for the SAME test always share the
// same file basename.
func sameFile(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ca, cb := filepath.Clean(a), filepath.Clean(b)
	if ca == cb {
		return true
	}
	return filepath.Base(ca) == filepath.Base(cb)
}
