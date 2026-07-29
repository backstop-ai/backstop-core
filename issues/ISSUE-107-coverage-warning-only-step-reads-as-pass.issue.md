---
title: "Coverage Warning Only Step Reads As Pass"
schema_version: issue/v1

issue:
  id: ISSUE-107
  title: "Coverage Warning Only Step Reads As Pass"
  type: bug
  status: open
  created: "2026-07-29"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Coverage Warning Only Step Reads As Pass

## Problem

`StepCoverageThreshold`'s own verdict computation (`pkg/gate/step_coverage.go:213-219`) is the
INVERSE of `ISSUE-104`/`ISSUE-105`: those issues were "a pack declares warning, the step blocks
anyway" (loud becomes blocking, when it must not). This is "a pack declares warning, the step reads
as clean" (loud becomes silent, when it must still be loud) — the opposite failure mode, and no
better.

```go
status := "pass"
for _, v := range violations {
    if v.Severity == "error" {
        status = "fail"
    }
}
```

This loop sets `status = "fail"` when an `"error"`-severity violation exists, but there is no `else`
branch — a coverage step whose violations are ALL `severity: "warning"` never flips `status` off its
initial `"pass"`. The step reports clean.

**This is the mirror-image bug of the one `blocksVerdict`'s own doc comment names as the founding
incident.** `pkg/gate/policy.go:47-52` records: *"a warning that fails the gate is a contradiction in
terms — and it shipped: CI run 30395875188 failed `coverage_threshold` whose ONLY violation was the
severity=warning coverage-exclusion NOTICE, because both verdict paths here counted entries without
reading this field."* That incident was fixed at the POLICY layer (`blocksVerdict`) and, per
`ISSUE-105`, at four raw-count step-builder sites — but `step_coverage.go`'s own verdict loop was
explicitly carved OUT of `ISSUE-105`'s scope as already-severity-aware and therefore correct
(`ISSUE-105`'s Problem section lists it under *"Severity-aware sites, verified NOT in scope as
defects... `pkg/gate/step_coverage.go:214-219` — `if v.Severity == "error" { status = "fail" }`"*). It
IS severity-aware — that part of the ISSUE-105 read is accurate — but severity-aware-in-the-blocking-
direction is not the whole contract: it also has to be severity-aware in the WARNING direction, and
today it is not. A step that is aware enough to skip failing on a warning must also be aware enough to
report `"warning"` instead of silently defaulting to `"pass"`.

**The live instance class already exists in this codebase.** `step_coverage.go:174-181` emits exactly
this shape today: a `coverage_exclusion` violation at `Severity: "warning"` whenever an in-scope
changed file's coverage requirement is suppressed by a pack-declared exclusion — "LOUDLY SURFACED... a
NON-blocking warning" per its own comment. It currently surfaces as loud only because backstop-core's
own `backstop.yml` happens to declare a coverage policy entry, routing the step through
`ApplyPolicy`'s severity-aware override; a policy-sparse consumer with the exact same
exclusion-only finding set gets `status: "pass"` straight out of the step builder, with no downstream
layer to correct it (`ApplyPolicy` only overrides steps that HAVE a policy entry — see
`ISSUE-105`'s "no-entry passthrough" finding, `policy.go:132-136`).

**Consequence for `GateResult`.** A `"pass"` status never increments `GateResult.StepsWarned`
(`pkg/gate/result.go:147-223`) and never appears distinguished from a genuinely clean run in the
summary line — the coverage-exclusion notice, whose entire purpose is to be visible, becomes invisible
to exactly the audience (a policy-sparse consumer) `ISSUE-105` identified as the population the
warning tier exists to protect.

## Impact

Pairs with `ISSUE-100`'s renderer half: this changes what a consumer SEES on a coverage step that
reads `"pass"` today, for any policy-sparse consumer whose coverage findings are entirely
warning-severity (the `coverage_exclusion` notice is the one live instance class in this codebase, but
any future warning-severity coverage rule inherits the same silent-pass defect). This is not a
question of blocking — a warning-only step correctly does not fail the gate either way — it is a
question of whether the non-blocking finding is REPORTED as a distinguishable state
(`StepsWarned`, summary visibility) or laundered into indistinguishable-from-clean. Same "loud !=
blocking" contract as `ISSUE-104`/`105`/`106`, opposite direction of the same defect class.

## Solution

**Fix shape (small, local, `pkg/gate/step_coverage.go:213-219`).** Replace the `pass`-initialized,
`error`-only loop with a tri-state verdict matching `StepVerdict`'s existing shape
(`pkg/gate/policy.go:117-130`): no violations → `"pass"`; at least one `severity: "error"` violation
→ `"fail"`; violations exist but none are `"error"` → `"warning"`. The step already has a full
`[]Violation` slice with severity populated correctly (`step_coverage.go:176,197` — both violation
constructors set `Severity` explicitly, `"warning"` for `coverage_exclusion` and `"error"` for
`coverage_threshold`/`coverage_metric_missing`), so this is a verdict-computation fix only; no
upstream severity-plumbing change is needed here the way `ISSUE-106`/`108` require.

**Consider reusing `StepVerdict` directly** (`pkg/gate/policy.go:117-130`) rather than re-deriving the
tri-state locally — it is the single existing severity-aware predicate this whole issue family
converges on, and `step_coverage.go` re-implementing the same three-way logic by hand is exactly the
kind of second spelling `StepVerdict`'s own doc comment (`policy.go:104-105`) says it exists to avoid
("there is exactly ONE severity predicate in this codebase and this is a thin wrapper over it, never a
second spelling"). If `step_coverage.go` has a structural reason not to call `StepVerdict` directly
(e.g., ordering with respect to other coverage-step branches not shown here), that reason should be
stated in the fix.

**Test gap to close.** Add a coverage-step test with an all-`severity:"warning"` violation set (the
`coverage_exclusion`-only shape) and assert `status == "warning"` (not `"pass"`), plus the
`GateResult`-level assertion that `StepsWarned` increments and `Pass` stays true — the same shape
`assertNonBlockingResult` (`pkg/gate/step_verdict_severity_test.go:345-358`) already provides for the
four `ISSUE-105` sites. A mixed error+warning coverage violation set should continue to report
`"fail"`, unchanged.

**Blast-radius discipline.** Measure coverage-step verdicts on backstop-core's own dogfood run and at
least one fixture consumer before and after; the only expected flip is `coverage_exclusion`-only steps
moving from `"pass"` to `"warning"` on a policy-sparse consumer (backstop-core itself already sees this
correctly via its policy entry, so its own dogfood run should show zero flips).

## Notes / references

- Founding incident this loop was originally written to fix: CI run `30395875188`, documented at
  `pkg/gate/policy.go:47-52` — the coverage_threshold-fails-on-a-notice defect this issue's fix must
  not reintroduce in the opposite direction.
- Found by: plan-reviewer-105 during `PLAN-ISSUE-105` review; confirmed by implementer-105,
  2026-07-29, while enumerating the sites `ISSUE-105` deliberately left untouched as already
  severity-aware.
- `ISSUE-105`'s Problem section explicitly carves this site out as *"Severity-aware sites, verified
  NOT in scope as defects"* — that read is correct for the blocking direction and incomplete for the
  warning direction; this issue is the completion, not a contradiction of that finding.
- Pairs with: `ISSUE-100` (step-tally/renderer half of the severity-contract neighborhood) — this
  issue changes what a consumer sees on a step that reports `"pass"` today, which is exactly `ISSUE-100`'s
  surface.
- Ratified contract record: `PLAN-ISSUE-020`.
- Family: `ISSUE-106` (substantiveness join discards pack severity) and `ISSUE-108` (contract carrier
  drops pack severity) are sibling residuals from the same audit; this is the one member of the family
  that inverts the failure direction (silent, not blocking) rather than repeating it.
