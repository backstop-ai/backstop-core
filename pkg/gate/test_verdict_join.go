package gate

import "strings"

// test_verdict_join.go is the ISSUE-118 mandated-test verdict join: the pure,
// I/O-free half that turns a flat pack dispatch stream into "the tests this spec
// MANDATED passed" — or did not. It follows RouteSubstantivenessFindings /
// HollowFindingsToViolations (substantiveness_join.go), the established precedent
// for routing a flat pack stream and joining it to a spec promise. It runs no
// engine, reads no file, and holds no language or tool noun.

// mandatedTestFailedRule is the rule id of the blocking verdict violation. It is
// NEW rather than a reuse of `test_verification` deliberately: no existing
// baseline entry can suppress it, and a failing mandated test is the one finding a
// ratchet must never inherit (ISSUE-118 sharp edge 2).
const mandatedTestFailedRule = "mandated_test_failed"

// gateTypeTest is the canonical gate_type spelling this package routes on. It is
// a STRING because that is what a Violation carries: pack_gate.go stamps each
// dispatched finding with its producing binding's declared gate_type already
// rendered, so the join joins string-to-string and never needs the enum type.
//
// It is DECLARED HERE rather than imported from pkg/pack/engine on purpose. The
// architecture contract for this package is `gate: mayDependOn: [artifact, check,
// config, pack, waiver]` — pack/engine is deliberately not among them, and
// pkg/gate keeps no dependency on the engine enum anywhere else. This mirrors
// traceability_polarity.go, which declares its own DimensionSubstantiveness /
// DimensionCoverage / DimensionContracts spellings for exactly the same reason:
// they join against a DECLARED gate_type across the same boundary.
//
// The two spellings are held in lockstep by a test, not by an import.
// TestRouteTestVerdictFindings_LockstepWithDeclaredGateTypeEnum drives real
// engine.GateType values through RouteTestVerdictFindings and asserts that
// GateTypeTest routes and the other six do not — so a rename or re-spelling on
// either side goes red behaviorally, which is a stronger pin than a shared
// constant reference would be. (Test files are exempt from the architecture
// rule via its `excludeFiles: ^.*_test\.go$`, so the guard can import the enum
// where production code may not.)
const gateTypeTest = "test"

// RouteTestVerdictFindings returns the members of a flat dispatch stream whose
// producing engine binding DECLARED gate_type test, preserving input order.
//
// Routing is by DECLARATION ALONE — never a pack name, rule name, message shape or
// command sniff (the ISSUE-064 rule). The gate type is spelled ONCE in this package,
// as gateTypeTest, and a lockstep test pins that spelling to the pack-facing
// engine.GateTypeTest enum so the two cannot drift apart. A violation with an empty
// GateType was not produced by a pack engine binding at all (every core-emitted
// violation) and is never routed.
func RouteTestVerdictFindings(violations []Violation) []Violation {
	routed := make([]Violation, 0, len(violations))
	for _, v := range violations {
		if v.GateType == gateTypeTest {
			routed = append(routed, v)
		}
	}
	return routed
}

// MandatedTestFailures joins routed test-verdict findings to the mandated tests an
// `implemented` spec promised, emitting one blocking violation per mandated test a
// finding NAMES.
//
// THE JOIN IS BY NAME, NOT BY PATH, AND THAT IS THE WHOLE POINT. A test runner
// reports a failure position as a BARE BASENAME — the committed capture reports
// `widget_test.go` for a package whose import path is `.../pkg/widget`, and reports
// no position at all for a failure with no `file:line` block. Neither form can ever
// match a gate scope's canonicalized repo-relative file set, so a path-keyed join
// silently drops every one of these findings. That silent drop IS ISSUE-118. The
// test's own function NAME is the only identifier that survives: it is what the
// spec mandated, and it is what the converter already puts in the message.
//
// Emitted violations are attributed to the MANDATED TEST'S own resolved file
// (falling back to the spec that declared it when path resolution has not run),
// NEVER to the finding's unresolvable reported path — letting that path through
// would put an unmatchable File into identity and scope and re-open the defect.
//
// A NIL scope MEANS "DO NOT SCOPE-FILTER AT ALL", PROJECT-WIDE, AND IS A
// LOAD-BEARING PART OF THIS CONTRACT — not a defensive nil check.
// MandatedTestFailures(mandated, verdicts, nil) returns a violation for every named
// mandated test regardless of any run's scope mode. That is the call shape the
// test-discovery-capability-absent path uses: above that guard ResolveMandatedTestPaths
// has not run, so mt.FilePath is structurally empty and GateScope.Contains("") is
// false in diff mode — a scoped call there would keep a mandated test only when its
// SPEC FILE happened to land in the diff, which in an all-test-file diff (the
// canonical ISSUE-118 shape) it does not, and the verdict this function exists to
// surface would be dropped. Do not "tidy" that nil call site into a scoped one.
//
// When a scope IS supplied, the guard is the SAME one the surrounding step applies
// to name-presence findings (keep when the mandated test's own file OR its spec
// file is in scope; all-mode keeps everything), so the two halves of the dimension
// agree. A failing mandated test whose files are entirely outside the diff stays
// out of scope BY DESIGN — that is ISSUE-129's surface and is deliberately not
// absorbed here.
func MandatedTestFailures(mandated []MandatedTest, verdicts []Violation, scope *GateScope) []Violation {
	projectRoot := ""
	if scope != nil {
		projectRoot = scope.ProjectRoot
	}

	violations := []Violation{}
	for _, mt := range mandated {
		// Implemented-only scope (ISSUE-054, sharp edge 6): a mandated test is a live
		// promise only once its spec is `implemented`. A draft spec's not-yet-written
		// test must never produce a verdict violation. This reuses the shared
		// predicate rather than re-implementing the filter.
		if !contractsAreDue(mt.Status) {
			continue
		}
		if scope != nil && scope.Mode != GateScopeModeAll {
			if mt.FilePath != "" && !scope.Contains(mt.FilePath) && !scope.Contains(mt.SpecFile) {
				continue
			}
			if mt.FilePath == "" && !scope.Contains(mt.SpecFile) {
				continue
			}
		}

		// One violation per mandated test even when several findings name it (a
		// subtest and its parent both fail); the extra detail folds into the single
		// message rather than becoming duplicate findings.
		var details []string
		for _, finding := range verdicts {
			if verdictNamesTest(finding.Message, mt.FuncName) {
				details = append(details, finding.Message)
			}
		}
		if len(details) == 0 {
			continue
		}

		file := mt.FilePath
		if file == "" {
			file = mt.SpecFile
		}
		violations = append(violations, Violation{
			Rule: mandatedTestFailedRule,
			File: NormalizePath(projectRoot, file),
			Message: "mandated test " + mt.FuncName + " FAILED (spec " + mt.SpecID +
				", claim " + mt.ClaimID + "): " + strings.Join(details, "; "),
			// `critical`, NOT `error`. Both BLOCK identically — blocksVerdict is
			// !EqualFold(Severity, "warning") — but `critical` is the ONLY route by
			// which a CORE-emitted rule reaches the production waiver policy's
			// non-waivable set: waiver.NewDeclaredPolicy(rules, []string{"critical"})
			// keys non-waivability on that severity, and its rule list is harvested
			// exclusively from PACK manifests, where a core-emitted rule has no
			// manifest to declare itself in. Downgrading this to `error` "for
			// consistency" would keep the block and silently make a failing mandated
			// test WAIVABLE, which is the one outcome this dimension exists to prevent.
			Severity: "critical",
		})
	}
	return violations
}

// verdictNamesTest reports whether a finding's message names funcName as a WHOLE
// identifier: present, and not immediately preceded or followed by an identifier
// character (letter, digit or underscore).
//
// The anchoring is what stops a finding naming TestFooBar from satisfying — or
// implicating — a mandate for TestFoo, in both directions. It must not be tighter
// than that: a test name in real runner output is surrounded by ordinary
// punctuation ("TestFoo: got 3 want 4", "--- FAIL: TestFoo"), and an anchor that
// rejected those would silently stop the join from ever firing.
//
// Pure string work — no regex compiled per call, no language noun, no assumption
// about the message's shape beyond the name appearing in it.
func verdictNamesTest(message, funcName string) bool {
	if funcName == "" {
		return false
	}
	for offset := 0; ; {
		idx := strings.Index(message[offset:], funcName)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(funcName)
		beforeOK := start == 0 || !isIdentifierByte(message[start-1])
		afterOK := end == len(message) || !isIdentifierByte(message[end])
		if beforeOK && afterOK {
			return true
		}
		offset = start + 1
	}
}

// isIdentifierByte reports whether b is a letter, digit or underscore — the
// generic identifier-character class the boundary anchor uses. It is deliberately
// ASCII-mechanical and names no language.
func isIdentifierByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_':
		return true
	default:
		return false
	}
}
