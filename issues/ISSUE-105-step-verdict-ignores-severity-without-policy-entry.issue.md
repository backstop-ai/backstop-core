---
title: "Step Verdict Ignores Severity Without Policy Entry"
schema_version: issue/v1

issue:
  id: ISSUE-105
  title: "Step Verdict Ignores Severity Without Policy Entry"
  type: bug
  status: closed
  created: "2026-07-29"
  closed: "2026-07-29"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

delivered_by: PLAN-ISSUE-105
---

# Step Verdict Ignores Severity Without Policy Entry

## Resolution

Delivered by PLAN-ISSUE-105 (status: completed), commit `d7d777c`. Exported `StepVerdict`
(`pkg/gate/policy.go`, +55/-0 — `ApplyPolicy`'s existing logic left byte-untouched, per the
passthrough-now-correct layering argument) and routed the four class-1 raw-count sites through it:
`pack_engines` (`cmd/backstop/gate.go`), `artifact_validation` and `code_check`
(`pkg/gate/step_delegate.go`), and `contract_signature` (`pkg/gate/step_contract.go`).
`ConfigErr` branches were left untouched.

**Three-stage red proof.** Compile-red (the helper didn't exist yet), then behavioral-red at the
helper stage alone (before any call site was converted), then green only once the call sites were
routed — with the inherited worktree's pre-existing failures proven to be the identical set before
and after, so the fix's own effect could be isolated from ambient noise.

**Blast radius: zero verdict flips on core.** Measured on backstop-core's own dogfood run and on a
fixture consumer per the PLAN-ISSUE-020 precedent for this class of change. The fixture consumer
supplies the one intended flip — no policy entry, `severity: warning` finding: step status
`fail`/exit 1 becomes `warning`/exit 0, with the finding still reported (loud, non-blocking). No
flip occurred that didn't fit that exact pattern.

**`step_contract` reclassified class-3 in evidence**, not converted: its carrier type has no
severity field, so there is nothing for a severity-aware helper to read. The mandated test instead
asserts the true invariant — a carrier without severity cannot participate in the severity
contract — guarded by a self-reporting premise check, closing that file's prior 0/13 coverage debt
to 100%.

**Completes the two-hop severity-contract chain with ISSUE-104.** Hop 1 (`a42b065`, ISSUE-104)
made the resolved severity value correct at the SARIF parse. Hop 2 (`d7d777c`, this issue) makes
the step verdict actually honor that value for every consumer, not only ones with a policy entry
declared for the dimension.

**Cross-lane acceptance met 2026-07-29.** The TASK-013 re-run against the stash consumer confirms
the full chain: exit 0, exactly one non-blocking warning reported, and the error-direction control
still blocks — proving hop 2 was necessary-but-insufficient alone (a prior re-run with only hop 1
landed still failed) and that both hops together are what close the loop.

**Residual family, named rather than swept.** "Warnings are non-blocking" is proven true today for
`pack_engines` and the four converted class-1 sites — it is NOT yet a universal property of every
step in the gate. Filed honestly as ISSUE-106/107/108 rather than implied by this closure.

## Problem

## Problem

Most step builders compute PASS/FAIL by RAW COUNT — `if len(violations) > 0 { status = "fail" }`
— never consulting `Violation.Severity`. Site inventory (`grep -n 'len(violations) > 0'` plus
`status = "fail"`, non-test):

- `cmd/backstop/gate.go:863` — the `pack_engines` step (the measured instance below).
- `pkg/gate/step_delegate.go:58-59` — `StepArtifactValidation`.
- `pkg/gate/step_delegate.go:106-107` — `StepCodeCheck`.
- `pkg/gate/step_contract.go:63-64` — `StepContractSignature`.

**Severity-aware sites, verified NOT in scope as defects.** Several steps compute their own
verdict by iterating violations and checking `Severity` explicitly — these are correct as-is and
this issue does not touch them:

- `pkg/gate/step_coverage.go:214-219` — `if v.Severity == "error" { status = "fail" }`.
- `pkg/gate/status_drift.go:96-104` and `pkg/gate/requirement_traceability.go:303-311` — same
  error/warning loop pattern (`driftStepResult`, `traceabilityStepResult`).
- `pkg/gate/step_waiver.go:152` (`if len(wv) > 0 { status = "fail" }`) is a raw count but is
  provably equivalent to a severity-aware one: every entry in `wv` is built by
  `waiverDiagToViolation` (`step_waiver.go:169-177`), which hardcodes `Severity: "error"` — a
  waiver-resolution diagnostic can never carry `severity: warning`, so there is no severity signal
  for a raw count to discard here. Left unchanged.

**The severity-aware predicate has exactly one non-test caller.** `blocksVerdict` /
`blockingViolations` (`pkg/gate/policy.go:80-95`) — the founder-ratified predicate that exempts an
explicit `severity: "warning"` from blocking a verdict (see the contract block at
`policy.go:55-79`) — is called from exactly one production site: `policy.go:173`, inside
`applyScopedPolicy`. Every step builder listed above computes its own `status` before
`ApplyPolicy` ever runs, using the severity-blind `len() > 0` check.

**`ApplyPolicy` only overrides steps a consumer happens to configure.** `ApplyPolicy`
(`pkg/gate/policy.go:119-194`) walks `policy map[string]DimensionPolicy` — the consumer's
`backstop.yml` `enforcement.policy` table — and for each step: `p, ok := policy[s.StepName]; if
!ok ... { out = append(out, s); continue }` (`policy.go:132-136`). A step with NO entry in that
map is appended UNCHANGED — its severity-blind `status` from the step builder stands, and
`blockingViolations` is never consulted for it at all.

**Net effect: the pack-author severity contract holds only for consumers who happen to declare a
policy entry for the dimension.** The contract ratified at `policy.go:55-61` — *"a SARIF `level:
warning` from ANY pack is NON-BLOCKING BY CONTRACT... severity IS how a pack author declares
blockingness"* — is, in practice, conditional on adopter configuration the pack author has no
visibility into or control over. backstop-core's own `backstop.yml` declares a `pack_engines`
policy entry, which is why this defect never surfaced in dogfooding. A consumer that declares no
`pack_engines` entry (or none at all) gets the severity-blind step-builder status instead.

**Measured, not theoretical.** implementer-101, 2026-07-29, one-config-line A/B probe against a
scratch consumer with byte-identical trees and one `severity: warning` finding from a real pack
rule:

- RUN A — no `pack_engines` policy entry in `backstop.yml`: step status `fail`, gate exit 1.
- RUN B — identical tree plus `enforcement.policy: { pack_engines: { level: block } }`: step
  status `warning`, gate exit 0, the finding STILL REPORTED (loud, non-blocking).

The only delta between the two runs is the presence of the policy entry. The `level: block` entry
in RUN B does not relax anything — it is the DEFAULT level — its only effect here is routing the
step through `ApplyPolicy`'s `blockingViolations` path instead of leaving the step-builder's raw
count untouched.

**Why this is core's defect, not a consumer-config gap.** A pack author cannot rely on a contract
whose enforcement depends on whether the adopting consumer happens to have written a policy table
entry for the dimension — that is precisely the scenario the WARNING tier exists to cover
(un-adopted-capability signals reaching consumers who have configured nothing at all for that
dimension). The one-line consumer-side workaround (add a `pack_engines` policy entry) was
deliberately NOT applied as the fix here, because it would leave every OTHER policy-sparse
consumer, and every other severity-blind step, exposed to the same defect.

**Not in tension with fail-closed.** The existing fail-closed law (an UNSET severity blocks,
`policy.go:63-67`) governs an ABSENT severity value. This defect is different: it discards a
severity value the finding DOES carry — a pack explicitly declared `level: warning` and the step
verdict never looked at it. Fail-closed-on-absence and honor-what-is-declared are not in tension;
this issue is squarely the second one.

## Impact

Any consumer whose `backstop.yml` has no (or a sparse) `enforcement.policy` table — which is the
common case for a project just adopting a pack, before it has written any per-dimension
enforcement config — gets severity-blind step verdicts. A pack's own advisory
(`severity: warning`) findings block the gate exactly as if they were blocking findings, for any
step listed in the inventory above, defeating the loud-not-blocking contract for exactly the
population it exists to protect (see also `feedback_loud_not_blocking`, CLAUDE.md).

Blocks `PLAN-ISSUE-101` TASK-013/014/015 (go-distribution pack v0.1.0 tag still held): even with
`ISSUE-104`'s SARIF-descriptor fix landed, a policy-sparse consumer adopting the pack still reds
on the pack's own `level: warning` advisory rules, because the step verdict never reaches
`blockingViolations` without a policy entry.

## Solution

**Fix shape.** Step-level verdicts should count only `blockingViolations` (the existing
severity-aware helper, `pkg/gate/policy.go:87-95`) regardless of whether the consumer has declared
a policy entry — the severity contract is a property of the finding, not of adopter
configuration. Smallest honest form: either (a) change each raw-count site in the inventory above
to call `blockingViolations(violations)` and check its length instead of the full slice's, or (b)
factor a shared `severityBlockingStatus(violations []Violation) string` helper in `pkg/gate` and
call it from all four sites plus `ApplyPolicy`'s unscoped default-block path
(`policy.go:171-190`), so there is exactly one place that decides "does this violation block."
Either way, `StepResult.Violations` keeps every entry (severity-blind reporting is correct and
must not change — only blindly).

The three severity-aware sites (`step_coverage.go`, `status_drift.go`,
`requirement_traceability.go`) and `step_waiver.go` do not need to change; the fix should say so
explicitly in its own audit rather than assume it, since a future edit to `waiverDiagToViolation`
that ever emits a non-error severity would silently reintroduce this defect there too.

**Test gap to close.** The existing end-to-end severity-contract test
(`cmd/backstop/pack_severity_contract_test.go`, landed closing ISSUE-100's split-off verdict
half) exercises a consumer WITH a policy table entry for the dimension under test — which is
exactly why both this issue and ISSUE-104 shipped past it undetected. Extend coverage with a
fixture consumer that has an EMPTY or ABSENT `enforcement.policy` table; the A/B probe shape above
(same tree, only the policy entry's presence differs, assert exit code AND per-step status differ
only in the expected way) is the acceptance fixture shape.

**Blast-radius discipline (per the PLAN-ISSUE-020 precedent for this exact class of fix).**
Measure step verdicts on backstop-core's own dogfood run and on at least one non-trivial fixture
consumer before and after the change; enumerate every verdict that flips. Every flip must fit the
pattern "was severity-blind-blocking a `severity: warning` finding, should never have blocked" —
any flip that does not fit that pattern is itself a new defect, not a side effect to wave through.

## Notes / references

- Hop 1 of this defect family: `ISSUE-104` (SARIF severity descriptor fallback) — fixed in commit
  `a42b065`. That fix makes the resolved severity value correct; this issue is hop 2, ensuring the
  step verdict actually reads it once resolved.
- Ratified contract record: `PLAN-ISSUE-020` (the founder-ratified severity contract, and the
  `blocksVerdict`/`blockingViolations` predicate this issue's fix reuses,
  `pkg/gate/policy.go:47-95`).
- Sibling: `ISSUE-100` (step-tally counts warnings as violations) — same severity-contract
  neighborhood; that issue's split-off verdict half is what delivered `blocksVerdict` honoring
  `Violation.Severity` in the first place, at the ONE call site (`policy.go:173`) this issue is
  about extending coverage past.
- Evidence: `PLAN-ISSUE-101` TASK-013 (the A/B probe run described above); blocks TASK-013/014/015
  there — v0.1.0 tag for the go-distribution pack is held pending this fix.
- Memory: `project_sarif_warning_severity_lost` — needs its re-open caveat honored; the RUN
  B-style consumer-side policy-entry workaround did NOT reverse the underlying defect, it only
  routes around it for consumers who happen to configure one.
