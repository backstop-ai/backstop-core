package gate

import (
	"path/filepath"
	"strconv"
	"strings"
)

// substantiveness_join.go is the language-agnostic, gate-side half of the
// SPEC-037 substantiveness pack split (BUNDLE-009 Seed 3). The PACK does the
// language-specific work (Q1 hollow-test findings rule + Q2 referenced-symbol
// extraction rule); this file consumes the FLAT pack-dispatch SARIF
// ([]Violation with {Rule, File, Line, Message, Severity, SourcePack, GateType} —
// ISSUE-118 added GateType, but it is stamped PER PRODUCING ENGINE BINDING and
// packs/substantiveness/pack.yml declares both the Q1 hollow rule and the Q2
// extraction rule under a single gate_type binding, so it is uniform across
// both roles here and cannot make the Q1/Q2 partition this file needs) and
// performs:
//   - routing substantiveness findings out of the flat pack_engines stream by the
//     pack-declared substantiveness_role property (RouteSubstantivenessFindings, ISSUE-064),
//   - keying extraction findings back to a MandatedTest by (FilePath, func) read
//     from the finding's structured SARIF Properties (ReferencedSetForTest),
//   - the noTarget SET-JOIN decision table (NoTargetViolation),
//   - the relocated, behavior-preserving target-package derivation
//     (TargetPackageName).
// It re-bakes NO go/parser / go/ast: routing and keying are string/set operations
// over pack SARIF, and the noTarget verdict is a pure set-membership test.
//
// ISSUE-116: this header PREVIOUSLY claimed the flat conversion dropped Line/region.
// It does not — the shared pack dispatch (cmd/backstop/pack_gate.go:744-748)
// deliberately PRESERVES the SARIF-reported start line on Violation.Line so the
// SPEC-049 waiver reconciliation can byte-scan a finding's own line for a @waiver
// token, and every join here must FORWARD it. HollowFindingsToViolations was written
// against the stale claim and silently zeroed the field, which made inline
// test_substantiveness waivers a no-op. Do not re-derive that bug from this comment.
//
// ISSUE-106 (hop 3 of the pack-severity chain, after ISSUE-104's parser and ISSUE-105's
// step verdicts): HollowFindingsToViolations ALSO hardcoded Severity "error", discarding
// the severity the pack itself declared, so a pack whose hollow rule declared `warning`
// still blocked the gate. It now FORWARDS the source finding's own severity verbatim.
// The noTarget half deliberately does NOT — it synthesizes a severity it was never
// handed, and that fixed "error" is a ratified decision recorded on NoTargetViolation.
// Do not re-derive either bug, or the asymmetry, from a stale comment.

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
//
// THE FIXED "error" SEVERITY IS A DECISION, NOT AN OVERSIGHT (ISSUE-106). Its sibling
// HollowFindingsToViolations FORWARDS a pack's declared severity, and the asymmetry
// between them is deliberate:
//   - this violation is SYNTHESIZED by the decision table above from a set-membership
//     test, not CONVERTED from any one pack finding, so there is no contributing finding
//     whose severity could be forwarded — the sibling forwards because it was HANDED a
//     value; this one has none to forward;
//   - ReferencedSymbolSet is map[string]bool, presence only. The extraction findings that
//     populate it carry no severity into the set at all, so there is nowhere for one to
//     ride;
//   - a noTarget verdict is a GATE-COMPUTED defect about the consumer's tests, not an
//     advisory the pack authored. A fixed severity is the honest reading of what it is.
//
// Giving packs a knob here therefore means inventing a NEW declaration channel (a
// severity carried alongside presence, or a noTarget-specific rule property). That is a
// capability expansion, not a bug fix, and it belongs in its own issue rather than in an
// edit here. TestNoTarget_SynthesizedSeverityIsFixedByDesign pins the decision.
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

// JoinEligibleForNoTarget reports whether a mandated test is JOIN-ELIGIBLE: whether an
// EMPTY evidence set would make it raise a noTarget violation (CLM-001, ISSUE-113).
//
// It is implemented AS the decision table rather than as a copy of its predicates. That
// is the whole point: NoTargetViolationForTest stays the SINGLE authority on which tests
// the join speaks about, so the guard below and the loop it guards can never disagree.
// Anyone who replaces this body with an explicit FilePath/absence/TargetPkg/same-package
// conjunction has re-opened exactly the drift this construction closes — and the
// conjunction would silently go stale the moment a fifth disposition is added to the
// table.
//
// FilePath is NOT part of the decision table; the caller filters unresolved tests (and
// out-of-scope ones) BEFORE consulting this.
func JoinEligibleForNoTarget(mt MandatedTest, samePackage bool) bool {
	_, raised := NoTargetViolationForTest(mt, ReferencedSymbolSet{}, samePackage)
	return raised
}

// SubstantivenessEvidenceRefusal decides whether the substantiveness step should REFUSE
// to report per-test noTarget verdicts because it has no evidence on which to found any
// of them (CLM-002/CLM-003, ISSUE-113).
//
// THE DEFECT IT EXISTS FOR. The Q2 noTarget set-join runs per mandated test and raises
// "does not call package X" whenever the target token is absent from that test's
// referenced-symbol set. It cannot, by itself, tell "this one test genuinely does not
// call its target" from "the pack produced no extraction evidence AT ALL, so EVERY set is
// empty". In the second case the step emitted one FALSE violation per mandated test — 397
// of them in the observed bclabs-portal incident — and named the real cause nowhere. A
// starved join does not produce a WRONG verdict about the code; it produces NO verdict at
// all, dressed as N verdicts. That is a broken TOOL, so the disposition follows ISSUE-020:
// one config-error naming the cause, instead of N unfounded findings.
//
// This function DECIDES; it does not dispose. The CALLER sets StepResult.ConfigErr — and
// that flag is doing three mechanical jobs, not one: Gate.Run halts the remaining steps
// and returns exit 2 so the operator reads one message rather than a wall; waiver
// resolution is ordered AFTER this step and therefore never runs, making the refusal
// structurally UNWAIVABLE; and ApplyPolicy skips ConfigErr steps, so it cannot be
// baseline-grandfathered into silence either. A future reader who "simplifies" ConfigErr
// away removes all three at once.
//
// THE hollow == 0 TERM IS LOAD-BEARING — it is the term a later reader is most likely to
// delete as redundant, and deleting it reintroduces a measured false refusal. Hollow
// evidence is core's ONLY independent proof that the pack's engine ran and actually
// classified test files; hollow > 0 therefore FALSIFIES the very diagnosis this message
// makes. Refusing there would also discard the real hollow violations the caller has
// already accumulated — and there is no vacuous green to prevent, because the gate is
// already RED with true findings pointing at the same files. The concrete measured case:
// the shipped newE2EWorkspace fixture (eligible 1, extraction 0, hollow 1) refuses
// without this term, discards its true hollow violation, and breaks
// TestE2E_SubstantivenessInstalledLocalPack_RealGate_HollowRed.
//
// TWO DELIBERATE RESIDUALS, so this comment does not read as a claim that the predicate
// is exact. Neither is a bug to quietly widen or narrow:
//   - UNDER-refusal, at hollow > 0: a pack that bakes its globs on the Q2 rule ONLY,
//     leaving Q1 healthy, lands in the non-refusing branch and its noTarget wall is not
//     collapsed. Covering it costs either true hollow findings or a vacuous green, so it
//     is left uncovered on purpose. Both incidents ISSUE-113 was filed from land in the
//     hollow == 0 branch and ARE covered.
//   - OVER-refusal, at hollow == 0: a workspace whose tests assert only through
//     UNQUALIFIED helper calls reaches this branch with TRUE noTarget verdicts at stake,
//     and this function suppresses them. Empirically verified against real ast-grep, not
//     hypothetical: `assertEqual(t, got, "x")` matches the pack's assertion-vocabulary
//     regex (so no Q1 finding) while the Q2 rule requires a selector_expression callee
//     (so no Q2 finding either) — zero findings from both rules. It is ACCEPTED because
//     core has no observable separating that case from a genuinely starved pack: packs
//     are opaque by design, and all three causes present as the identical empty result.
//     The mitigation is HONESTY — the message names it as one of three candidate causes,
//     which is why that third cause is load-bearing and not padding. Tightening the
//     predicate to exclude it would require information packs do not provide; do not
//     attempt it here.
//
// The returned Violation deliberately carries NO File. The refusal is about the pack's
// configuration, not about any one file, and attaching an arbitrary test file would
// re-personalize a finding whose whole point is that it belongs to none of them.
func SubstantivenessEvidenceRefusal(eligible, extraction, hollow int) (Violation, bool) {
	if eligible < 1 || extraction > 0 || hollow > 0 {
		return Violation{}, false
	}
	count := strconv.Itoa(eligible)
	return Violation{
		Rule: StepTestSubstantiveness,
		// ONE message, no fork on the hollow count: the hollow > 0 state no longer
		// refuses, so there is exactly one diagnostic state to describe. It states the
		// OBSERVED FACT first (the only thing core actually knows), then names all THREE
		// candidate causes without asserting any of them, then says what it is refusing
		// INSTEAD OF — the operator who has seen the violation wall needs to recognize
		// this as its replacement. Cause three is worded WITHOUT the word "hollow" on
		// purpose: a message discussing hollow findings would describe a state this
		// refusal can no longer reach.
		Message: "the substantiveness pack produced no findings of any kind while " + count +
			" mandated tests were join-eligible — its engine did not run, its classification " +
			"matched 0 test files, or those tests genuinely make no package-qualified calls " +
			"while still satisfying the pack's assertion vocabulary; refusing instead of " +
			"reporting " + count + " unsubstantiated \"does not call package\" violations",
		Severity: "error",
	}, true
}

// substantiveness role vocabulary (ISSUE-064). The consuming pack STAMPS one of these
// role values into each substantiveness finding's structured Properties channel (the
// ISSUE-062 `Violation.Properties` lift), declaring what the finding IS — a `hollow`
// finding (a test with no assertion) or a `referenced-symbol` finding (a symbol the test
// references, used for the subject-join). Core routes purely on this DECLARED role, so a
// pack may name its rules anything (`hollow-test-ts`, `hollow-test-go`, an org-specific
// name); the language-suffixed rule NAME is no longer a routing key (REQ-001/REQ-002).
const (
	substantivenessRoleProperty   = "substantiveness_role"
	substantivenessRoleHollow     = "hollow"
	substantivenessRoleReferenced = "referenced-symbol"
)

// RouteSubstantivenessFindings partitions the FLAT pack_engines []Violation stream into
// the substantiveness hollow-findings and extraction-findings by the pack-DECLARED role
// carried in each finding's structured Properties (`substantiveness_role`, the ISSUE-062
// channel) — NOT by matching a baked namespaced rule-id literal (ISSUE-064). A finding
// whose role is `hollow` joins the hollow partition; `referenced-symbol` joins the
// extraction partition; any other role (or no role property) is ignored, exactly as a
// non-substantiveness pack rule is. NO gate_type field is consulted — it is stamped
// uniformly across both roles for this pack (Sharp Edge 5 / REQ-007 / CLM-024) and
// so cannot substitute for the role property.
func RouteSubstantivenessFindings(violations []Violation) (hollow, extraction []Violation) {
	for _, v := range violations {
		switch v.Properties[substantivenessRoleProperty] {
		case substantivenessRoleHollow:
			hollow = append(hollow, v)
		case substantivenessRoleReferenced:
			extraction = append(extraction, v)
		}
	}
	return hollow, extraction
}

// ReferencedSetForTest joins the routed Q2 extraction findings to a MandatedTest by
// (FilePath, func) and assembles that test's ReferencedSymbolSet. The enclosing test
// name and the referenced symbol come from each finding's STRUCTURED Properties
// (`func`/`symbol`, ISSUE-062) — NOT parsed out of the free-text Message, NOT from
// Line/region (which the flat conversion DOES carry, per the header's ISSUE-116 note,
// but which is not a join key: this join is by structured identity, not by position),
// and NOT by re-walking the test AST (Sharp Edge 2).
// Properties["func"] is compared to MandatedTest.FuncName VERBATIM, so
// the join is correct for a name containing spaces or quotes (a string-named it()/test()
// description), not only a single-token Go TestXxx name. A test with no matching finding
// yields an empty set, which the decision table handles unchanged (CLM-025/CLM-026).
func ReferencedSetForTest(extraction []Violation, test MandatedTest) ReferencedSymbolSet {
	set := ReferencedSymbolSet{}
	for _, v := range extraction {
		if !sameFile(v.File, test.FilePath) {
			continue
		}
		if v.Properties["func"] != test.FuncName {
			continue
		}
		if symbol := v.Properties["symbol"]; symbol != "" {
			set[symbol] = true
		}
	}
	return set
}

// IsTestHollow turns the routed hollow partition into the per-test hollow verdict the
// substantiveness step raises: it maps each routed hollow finding to its mandated test
// by (File, Properties["func"]) — the enclosing test name read VERBATIM from the
// finding's structured Properties (ISSUE-062), never parsed from the message — and
// reports whether the given test is hollow. Language-agnostic and whitespace-safe: Go
// and TS hollow findings flow identically, and a spaced/quoted test name matches
// exactly (CLM-003/004/012/013/014, CLM-006).
func IsTestHollow(hollow []Violation, test MandatedTest) bool {
	for _, v := range hollow {
		if !sameFile(v.File, test.FilePath) {
			continue
		}
		if v.Properties["func"] == test.FuncName {
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
// test_substantiveness rule name — one violation per hollow finding. The Violation's
// GateType field (ISSUE-118) is left unset here deliberately — it is stamped uniformly
// across the hollow and extraction roles for this pack, so it carries no role
// information this conversion could use (Sharp Edge 5).
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
			File: NormalizePath("", v.File),
			// Carry the SARIF-reported start line so the SPEC-049 waiver
			// reconciliation can byte-scan the finding's own line for a @waiver
			// token (ISSUE-116) — matching what the shared pack dispatch already
			// preserves at cmd/backstop/pack_gate.go:744-748. Safe for baseline
			// grandfathering: Line is json:"-" and excluded from identity hashing
			// (pkg/gate/result.go:81-88), so identity stays line-INDEPENDENT.
			Line:    v.Line,
			Message: stripFuncToken(v.Message),
			// FORWARD the pack's own declared severity, VERBATIM (ISSUE-106). The
			// severity a pack declares IS its blockingness declaration (the ratified
			// contract on blocksVerdict, pkg/gate/policy.go), so overwriting it here
			// silently converted a pack's advisory into a blocker — the ISSUE-104 /
			// ISSUE-105 defect recurring one hop later, inside a converter both of
			// those fixes bypass.
			//
			// NO nonEmptySeverity WRAPPER, DELIBERATELY, for two reasons. The value has
			// ALREADY been defaulted upstream: the production bridge in
			// cmd/backstop/pack_gate.go applies nonEmpty(v.Severity, "error") before
			// routing, so a second default here would be a second spelling of one rule.
			// And forwarding an empty value is safe anyway, because blocksVerdict treats
			// anything that is not "warning" as blocking — this join fails CLOSED by
			// construction, never open. TestQ1_HollowFindingsToViolations_ForwardsPackDeclaredSeverity
			// (substantiveness_severity_test.go) pins both halves. Do not "harden" this
			// back into a second authority.
			Severity: v.Severity,
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
