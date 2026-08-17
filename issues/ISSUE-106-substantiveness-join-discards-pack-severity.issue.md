---
title: "Substantiveness Join Discards Pack Severity"
schema_version: issue/v1

issue:
  id: ISSUE-106
  title: "Substantiveness Join Discards Pack Severity"
  type: bug
  status: closed
  created: "2026-07-29"
  closed: "2026-08-17"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

delivered_by: PLAN-ISSUE-106
---

# Substantiveness Join Discards Pack Severity

## Resolution

Delivered by PLAN-ISSUE-106 (status: completed), commit `c4dfabf`.

`HollowFindingsToViolations` (`pkg/gate/substantiveness_join.go`) no longer hardcodes
`Severity: "error"` on every hollow-test violation — it now forwards the pack's own declared
severity (already defaulted by `nonEmptySeverity` at Q1 dispatch), so a substantiveness pack that
ships a warning-severity rule can produce a genuine non-blocking warning instead of being silently
upgraded to a hard failure. This is the direct 1:1-conversion fix shape this issue's Solution
section called for.

`cmd/backstop/gate.go`'s `buildTestSubstantivenessStep` correspondingly switched from a raw
violation-count status to `gate.StepVerdict`, so a warning-only result no longer reports `fail` —
closing the gap on the step-verdict side as well, consistent with the `ISSUE-105` precedent this
issue's family follows.

**Blast radius measured byte-identical**, per the `PLAN-ISSUE-020` discipline this issue's Solution
section mandated: a control-vs-treatment comparison on this repo's own findings showed no verdict
flips, because neither currently-installed substantiveness rule declares a `severity:` key today.
The forwarding mechanism is proven correct by the fix and by regression coverage, and is ready for
when a pack does declare a non-default severity — nothing changed value in practice yet, which is
the expected result of a mechanism-correctness fix against an unpopulated input.

**`NoTargetViolation` (the harder design-decision half) was NOT changed** — this issue's Solution
section explicitly separated the two sites as not interchangeable, and the plan's scope was the
direct 1:1 conversion. `NoTargetViolation` remains a fixed-severity synthesized violation; whether
it should ever carry a pack-declared severity is not resolved by this closure and would need its
own follow-on if raised again.

**One claim (CLM-004, the end-to-end e2e proof) is written but its two e2e tests are deliberately
left red pending `ISSUE-148`** (substantiveness fixture polarity — unrelated to this lane). This
was landed red-on-purpose rather than skipped or weakened: the fix itself (CLM-001/002/003/005/006)
is fully proven by unit-level coverage; only the e2e demonstration of CLM-004 awaits ISSUE-148's
fixture fix. `TestClass3Sites_ViolationsAreErrorSeverityByConstruction`'s SITE 2 assertions were
updated per this issue's named coupling requirement, flipping the premise from "warning-in,
error-out" (the defect) to asserting severity is now preserved.

## Problem

The substantiveness JOIN — the gate-side half of the SPEC-037 substantiveness pack split
(`pkg/gate/substantiveness_join.go`) — discards a pack-declared severity before the step verdict
ever sees it, one hop past where `ISSUE-104`/`ISSUE-105` closed the same gap for other sites.

**Pack severity survives dispatch, then is overwritten at the join.** The Q1 dispatch converter
(`pkg/gate/substantiveness_q1_dispatch.go:71`) correctly preserves whatever severity the pack
declares on a hollow-test finding: `Severity: nonEmptySeverity(v.Severity)` — a genuine pass-through
that only defaults an *empty* value to `"error"` (`substantiveness_q1_dispatch.go:110-113`), leaving
a declared `"warning"` untouched. The join then throws that value away:

- `HollowFindingsToViolations` (`substantiveness_join.go:184`) converts each routed hollow finding
  into a `test_substantiveness` `Violation` and hardcodes `Severity: "error"` on every one,
  regardless of what `v.Severity` carried in.
- `NoTargetViolation` (`substantiveness_join.go:68`) — the noTarget set-join decision table —
  likewise hardcodes `Severity: "error"` on the violation it constructs.

**Already caught in the codebase's own regression lock, filed rather than fixed.**
`TestClass3Sites_ViolationsAreErrorSeverityByConstruction`
(`pkg/gate/step_verdict_severity_test.go:291-308`) feeds `HollowFindingsToViolations` a hollow
finding with `Severity: "warning"` and asserts the *output* is `"error"` — the exact warning-in/
error-out witness for this defect, with an inline comment naming it directly: *"the pack's declared
severity survives Q1 dispatch (`nonEmptySeverity`) and is then discarded here, so a pack declaring a
`warning` substantiveness rule still blocks after this lane. Same contract, different axis — severity
lost BEFORE the verdict rather than ignored BY it. Filed, not fixed here."* That test's SITE 1/3
guards (`waiverDiagToViolation`, `StepTestVerificationScopedFunc`) are genuinely structurally
warning-free by construction; SITE 2 (this issue) is not — it is discarding a real input value, not
locking an invariant that has no other input.

**The two sites are not the same shape of fix.** `HollowFindingsToViolations` is a 1:1 conversion —
one hollow finding in, one violation out — so forwarding `v.Severity` (already defaulted by
`nonEmptySeverity` upstream) instead of the hardcoded literal is a direct substitution. `NoTargetViolation`
is structurally different: it does not convert a single input finding at all. It fires on a
SET-MEMBERSHIP test — whether the target package name is present in a test's `ReferencedSymbolSet`
(itself a `map[string]bool`, `substantiveness_join.go:26`, assembled from zero or more
`referenced-symbol` extraction findings via `ReferencedSetForTest`) — so there is no single
contributing finding whose severity to carry forward, and the extraction findings that DO exist in
the set carry no severity value today (the map is boolean presence only). Declaring a severity for
the noTarget violation therefore requires either extending `ReferencedSymbolSet` to carry severity
alongside presence, or having the extraction/hollow *rule* declare the noTarget severity directly
(e.g., a `substantiveness_role` property convention distinct from presence), not merely forwarding
an existing field.

## Impact

Any pack that declares a substantiveness rule (hollow-test or noTarget-adjacent) at `level: warning`
— intending an advisory, non-blocking signal — has that declaration silently overwritten to `error`
at the join, and the finding blocks the gate exactly as if the pack had declared it blocking. This is
the same "loud != blocking" contract violation `ISSUE-104`/`ISSUE-105` closed at the SARIF-parse and
step-verdict layers; this issue is the same defect recurring one hop later, inside a step-specific
converter that both of those fixes bypass entirely (the substantiveness join runs downstream of
`StepVerdict`'s severity-aware sites, feeding it violations that are already `"error"` by the time it
sees them).

## Solution

**Fix shape — `HollowFindingsToViolations` (the direct fix).** Forward the finding's already-resolved
`v.Severity` instead of the hardcoded literal at `substantiveness_join.go:184`. `nonEmptySeverity` at
dispatch already guarantees the value is never empty, so no additional defaulting is needed at the
join.

**Fix shape — `NoTargetViolation` (the design decision).** This is the harder half: the noTarget
violation is SYNTHESIZED by the gate's own decision table, not converted from one input finding, so
the fix must decide what severity a JOINED/synthesized violation inherits when it is not a 1:1
conversion — for example, whether the pack declares a severity for the noTarget rule itself (carried
through a new channel, since presence-only `ReferencedSymbolSet` has nowhere to put it today), or
whether the noTarget violation stays a fixed severity by design because it represents a genuine
gate-computed defect rather than an advisory the pack can tune. Either resolution should be stated
explicitly in the fix rather than left implicit, since the two sites are not interchangeable and a
uniform "just forward severity" instruction does not apply cleanly to both.

**Blast-radius discipline (per the `PLAN-ISSUE-020` precedent this family follows).** Measure
substantiveness step verdicts on backstop-core's own dogfood run and at least one fixture consumer
before and after; every flip must fit "was severity-blind-overwriting a declared `warning` finding,
should never have blocked."

**Test gap.** `TestClass3Sites_ViolationsAreErrorSeverityByConstruction`'s SITE 2 assertions
(`step_verdict_severity_test.go:282-308`) currently LOCK the defective behavior (warning-in,
error-out) as a "filed, not fixed here" regression guard. Once this issue's fix lands, that test's
premise flips — it must be updated deliberately to assert severity IS preserved, not accidentally
left asserting the old (now wrong) behavior. Name this coupling explicitly in the fix so it is not
missed the way a bare test-suite pass could miss it.

## Notes / references

- Ratified contract record: `PLAN-ISSUE-020` (the founder-ratified severity contract) and
  `pkg/gate/policy.go:47-95` (`blocksVerdict`/`blockingViolations`/`StepVerdict`) — the predicate
  this family of fixes exists to make the JOIN honor, not merely the step verdict.
- Hop 1: `ISSUE-104` (SARIF severity descriptor fallback, `a42b065`) — makes the parsed severity
  value correct.
- Hop 2: `ISSUE-105` (step verdicts ignore severity without a policy entry, `d7d777c`) — makes the
  step-level verdict read that value regardless of adopter policy config.
- This issue is a residual enumerated by implementer-105 (2026-07-29) after both hops landed: the
  substantiveness JOIN sits between a pack's declared severity (correctly resolved and dispatched)
  and the step verdict (now severity-aware), and silently substitutes its own hardcoded value in
  between — a defect neither hop's fix touches.
- Live witness: `TestClass3Sites_ViolationsAreErrorSeverityByConstruction`
  (`pkg/gate/step_verdict_severity_test.go:255-339`), SITE 2 block
  (`step_verdict_severity_test.go:282-308`), which documents this exact gap inline as "SIBLING 1...
  Filed, not fixed here."
- Family: `ISSUE-107` (coverage warning-only step reads as pass) and `ISSUE-108` (contract carrier
  drops pack severity) are sibling residuals from the same audit, each a different hop in the same
  pack-severity contract.
