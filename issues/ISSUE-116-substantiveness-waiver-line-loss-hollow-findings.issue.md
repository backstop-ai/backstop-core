---
title: "Substantiveness Waiver Line Loss Hollow Findings"
schema_version: issue/v1

issue:
  id: ISSUE-116
  title: "Substantiveness Waiver Line Loss Hollow Findings"
  type: bug
  status: closed
  created: "2026-08-09"
  closed: "2026-08-10"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate

delivered_by: PLAN-ISSUE-116
---

# Substantiveness Waiver Line Loss Hollow Findings

## Resolution

Delivered by PLAN-ISSUE-116 (`plans/PLAN-ISSUE-116-substantiveness-waiver-line-carry.plan.yml`).

**Fix:** `HollowFindingsToViolations` (`pkg/gate/substantiveness_join.go`) now sets `Line: v.Line`
on the constructed `Violation`, matching what the shared pack dispatch already preserves at
`cmd/backstop/pack_gate.go:748`. The two stale docstrings that asserted the false "Line/region is
already dropped" premise — the premise that caused this function to be written without the field
in the first place — were corrected in the same change so the invariant can't be re-broken by a
future edit made against the old comment.

**Verification — real production gate+waiver path, not an isolated unit check:**
- `TestQ1_HollowFindingsToViolations_ForwardsSourceLine`
  (`pkg/gate/substantiveness_hollow_violation_test.go`) — unit pin proving the join forwards each
  finding's own line positionally.
- `TestE2E_SubstantivenessHollowWaiver_SuppressesFindingOnItsOwnLine` and
  `TestE2E_SubstantivenessHollowWaiver_MismatchedRuleIDSurfacesAsUnused`
  (`cmd/backstop/gate_substantiveness_waiver_e2e_test.go`) — behavioral e2e tests that run a real
  hollow finding with an inline `@waiver:test_substantiveness:...` token through the actual gate +
  waiver reconciliation path (`computeWaiverResult` / `waiver.Adjudicate`), not
  `HollowFindingsToViolations` called in a vacuum.
- All three tests were confirmed true falsifiers: independently re-verified RED against the
  pre-fix code by both the implementer and a separate impl-reviewer pass, with the pre-fix RED
  values matching what the plan's own planning-time measurement predicted.
- impl-reviewer: clean PASS on correctness, test substantiveness, scope-fence adherence, and
  claim mapping.
- Full gate run: PASS, 10/10 steps (195 pre-existing advisory warnings, unrelated to this change).

**Scope note:** ISSUE-117 ("Substantiveness noTarget Findings Have No Waivable Line") is the
architecturally-separate sibling case named in this issue's own References as scoped OUT of
PLAN-ISSUE-116 — `noTarget` findings carry no line at all, a different mechanism than the hollow
findings this issue covers. ISSUE-117 remains open, untouched by this close.

## Problem

An inline `@waiver:test_substantiveness:<reason>:<expiry>` token does not suppress a hollow-test
`test_substantiveness` finding, even when placed on the exact line SARIF reports the finding at.
`pack_engines` waivers work correctly on the same gate run; `test_substantiveness` waivers silently
do nothing — no suppression, and the token doesn't even surface in `backstop waiver list`'s
unused/dangling bucket, so from the user's side it reads as "not recognized at all," not "recognized
but rejected."

**Root cause: `HollowFindingsToViolations` drops `Line` on the way into the `Violation`.**

The shared pack dispatch preserves the SARIF-reported line specifically so waiver reconciliation can
byte-scan a finding's own line for a token:

```go
// cmd/backstop/pack_gate.go:744-748
// Carry the SARIF-reported start line so the SPEC-049 waiver
// reconciliation can byte-scan the finding's own line for a @waiver token.
// It rides through to gate.Violation.Line, which is line-INDEPENDENT of
// baseline identity (json:"-").
Line:        v.Line,
```

`HollowFindingsToViolations` (`pkg/gate/substantiveness_join.go:172-188`) takes those already-dispatched
hollow findings and re-packages each one into a `test_substantiveness` `Violation` — but only copies
`File`, not `Line`:

```go
// pkg/gate/substantiveness_join.go:172-188
func HollowFindingsToViolations(hollow []Violation) []Violation {
	out := make([]Violation, 0, len(hollow))
	for _, v := range hollow {
		out = append(out, Violation{
			Rule:     StepTestSubstantiveness,
			File:     NormalizePath("", v.File),
			Message:  stripFuncToken(v.Message),
			Severity: "error",
		})
	}
	return out
}
```

Every hollow-finding `Violation` this function produces carries `Line == 0` (Go zero value). The
waiver adjudicator builds its lookup key from that same struct:

```go
// pkg/gate/step_waiver.go:89
findings = append(findings, waiver.Finding{RuleID: v.Rule, File: v.File, Line: v.Line, Severity: v.Severity})
```

and re-keys suppression the same way on the subtraction pass (`step_waiver.go:112`,
`findingKey(v.File, keyLine(v), v.Rule)`). With `Line` always `0`, the adjudicator scans line 0 —
never the line the `@waiver` token is actually placed on — and no token it finds there can ever match.
This is deterministic, not flaky: it fails for every hollow finding, regardless of what token text or
placement the user tries.

**This is not "the step isn't wired to waivers."** `test_substantiveness` IS a correctly-declared
waivable dimension:

```go
// pkg/gate/step_waiver.go:66-68
func waivableDimension(step string) bool {
	return step == StepPackEngines || step == StepTestSubstantiveness
}
```

`pack_engines` findings go through the same dispatch (`pack_gate.go:744-748`) and keep their `Line`
all the way to adjudication, which is why `pack_engines` waivers work and is the asymmetry that makes
this diagnosable: one waivable dimension's join step silently zeroes the field the other one carries
through untouched.

**The docstring's own stated assumption is stale and is presumably why the bug was introduced.**
`substantiveness_join.go:11-13` describes the flat stream this file consumes as:

```
// ([]Violation with only {Rule, File, Message, Severity, SourcePack} — Line/region
// dropped, no GateType field)
```

That's no longer true — the shared dispatch at `pack_gate.go:748` deliberately preserves `Line` before
this file ever sees the finding. `HollowFindingsToViolations` was written against the stale "Line is
already gone" assumption and only copies the fields the docstring says survive. The docstring must be
corrected alongside the fix so the invariant isn't re-broken by a future change made against the same
stale comment.

**A near-miss in the test suite that looks like prior evidence but isn't.**
`pkg/gate/baseline_test.go:87-88` carries an inline `@waiver:test_substantiveness:...:... PROBE` token
on `TestBaseline_IdentityHash_DeterministicContract`. This looks like a working example of the
mechanism but is not one: that test has real, substantive assertions, so no `test_substantiveness`
hollow finding is ever raised against it, and the token is inert — it suppresses nothing because there
is nothing to suppress. Do not treat this as prior evidence the waiver-on-hollow-finding path has ever
been exercised successfully; it hasn't.

### How it surfaced

`bclabs-brief-builder` installed `backstop-ai/typescript-substantiveness@1.1.1` for the first time and
hit 75 hollow-test `test_substantiveness` findings that were legitimately false positives for that
project's fixtures. Every `@waiver:test_substantiveness:...` token placed to suppress them was
ignored — this is likely the first time this pack's waiver path has been exercised against a real,
non-trivial hollow finding outside backstop-core's own (inert, per above) self-check.

## Impact

Any consumer repo using the `test_substantiveness` hollow-test rule has no working inline suppression
escape-hatch. Legitimate false positives on the hollow-test rule cannot be waived — the only
workarounds are restructuring real code to dodge the rule or disabling the rule globally in the pack,
the same two unacceptable options ISSUE-017 documented for the (already-fixed) `pack_engines`/nosemgrep
case. Because the waiver silently no-ops rather than erroring, a user has no signal that anything is
wrong with their token — it just doesn't work, and `backstop waiver list` gives no diagnostic trail
either (see Solution's note on the unused/dangling bucket).

## Solution

**Direction, not final code — exact shape is the plan's job.**

Carry `Line: v.Line` through in the `Violation` `HollowFindingsToViolations` constructs
(`pkg/gate/substantiveness_join.go:172-188`), matching what the shared dispatch already preserves at
`pack_gate.go:748`. Correct the stale "Line/region dropped" docstring at `substantiveness_join.go:11-13`
in the same change so the false assumption isn't reintroduced later.

**Acceptance must be behavioral, not a parser unit test in isolation.** The mandated test needs a real
hollow finding with a real `File`+`Line`, a real inline `@waiver:test_substantiveness:...` token placed
on that line in a fixture, run through the actual waiver reconciliation path (`computeWaiverResult` /
`waiver.Adjudicate`, not `HollowFindingsToViolations` called in a vacuum), asserting the finding is
suppressed and the `test_substantiveness` step flips to pass. RED before the fix (finding still active
despite the token), GREEN after.

Worth checking as part of the same fix: whether a `test_substantiveness` waiver token that fails to
match currently surfaces anywhere in `backstop waiver list`'s unused/dangling accounting — per the
report that follows this issue, it currently doesn't, which is itself part of what makes this defect
read as "not recognized" rather than "recognized but not working."

## References

- `pkg/gate/substantiveness_join.go:172-188` — `HollowFindingsToViolations`, drops `Line`
- `pkg/gate/substantiveness_join.go:11-13` — stale docstring claiming Line/region is dropped upstream
- `cmd/backstop/pack_gate.go:744-748` — shared pack dispatch, preserves `Line` for waiver reconciliation
- `pkg/gate/step_waiver.go:66-68` — `waivableDimension`, confirms `test_substantiveness` is correctly
  declared waivable
- `pkg/gate/step_waiver.go:89` — waiver finding collection, keys on `v.Line`
- `pkg/gate/step_waiver.go:112` — suppression subtraction pass, re-keys on the same `Line`
- `pkg/gate/baseline_test.go:87-88` — inert `@waiver:test_substantiveness` token on a non-hollow test;
  not evidence the mechanism works
- ISSUE-017 — same "silent inline-suppression no-op" shape, previously fixed for `pack_engines`/nosemgrep
- Surfaced by: `bclabs-brief-builder` installing `backstop-ai/typescript-substantiveness@1.1.1`,
  75 unwaivable hollow-test false positives
- ISSUE-117 — sibling case (noTarget findings have no line at all); architecturally separate,
  scoped out on its own
