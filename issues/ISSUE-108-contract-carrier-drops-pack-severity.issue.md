---
title: "Contract Carrier Drops Pack Severity"
schema_version: issue/v1

issue:
  id: ISSUE-108
  title: "Contract Carrier Drops Pack Severity"
  type: bug
  status: open
  created: "2026-07-29"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Contract Carrier Drops Pack Severity

## Problem

`contract_signature` is the one converted verdict site in the `ISSUE-105` inventory where a pack
still cannot declare a non-blocking finding at all — not because the step verdict ignores severity
(it doesn't, since `ISSUE-105` routed it through `StepVerdict`), but because severity is dropped one
hop EARLIER, at the carrier type, before it ever reaches the step.

**The carrier has no severity field.** `ContractEngineResult` (`pkg/gate/contract_verdict.go:31-36`):

```go
type ContractEngineResult struct {
    Entry     ContractEntry
    Matched   bool
    Scanned   bool
    Locations []SarifLocation
}
```

There is no `Severity` field for a pack to populate. `produceContractEngineResults`
(`cmd/backstop/gate.go`, ~line 1501, in `buildContractStep`) constructs these values from the pack's
contracts SARIF dispatch but has no field to carry a declared severity into even if the pack emitted
one.

**`VerifyContractVerdict` hardcodes `"error"` on all three of its violation-returning branches**
(`pkg/gate/contract_verdict.go:77,85,101`): the unscanned-scope config-error branch, the
forbidden-symbol-present absence violation, and the missing-signature present-contract violation all
construct their `Violation` with `Severity: "error"` literally, regardless of anything the carrier
could theoretically hold — because the carrier holds nothing to read.

**Consequence: `contract_signature` is structurally warning-free by construction, not by policy.**
This is DIFFERENT in kind from `ISSUE-106`/`107`, where a pack-declared value exists somewhere
upstream and is discarded en route. Here there is no channel for the value to exist in at all — the
type itself cannot represent it. That distinction is already established as evidence in
`PLAN-ISSUE-105`'s record: `TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy`
(`pkg/gate/step_verdict_severity_test.go:163-193`) documents, in its own comment
(`step_verdict_severity_test.go:148-162`), that this site was MEASURED WHILE IMPLEMENTING and
CONTRADICTED the plan's original site classification: *"this site cannot be handed a declared warning
at all... `ContractEngineResult` carries no severity field for a pack to populate. So
`contract_signature` is structurally warning-free (the plan's CLASS 3) rather than a slice carrying a
pack-resolved severity (CLASS 1)."*

**The mandated coupling: a self-reporting premise guard, not a static assertion.**
`TestClass3Sites_ViolationsAreErrorSeverityByConstruction`
(`pkg/gate/step_verdict_severity_test.go:265-339`) is the regression lock for three sites the
`ISSUE-105` lane deliberately left as raw-count because their inputs are structurally error-only by
construction. `TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy` is the sibling test
for THIS site specifically, and it is written to self-report the moment this issue's fix lands: it
asserts `result.Violations[0].Severity != "error"` would indicate staleness
(`step_verdict_severity_test.go:182-186`, *"If severity now flows here, the CLASS-3 reading above is
stale and this site needs a genuine warning-input test"*), and separately asserts the CONTINGENT half
directly on `StepVerdict` rather than on a constructible input (`step_verdict_severity_test.go:188-192`,
*"were a declared warning ever to reach this slice, the converted step reports it non-blocking"*) —
because today no input CAN construct that case. Once this issue adds a `Severity` field to
`ContractEngineResult`, that test's premise flips from true to false, and it must be updated
DELIBERATELY as part of this issue's fix, not left to fail unnoticed or edited away without
understanding why it existed.

## Impact

Any pack that wants to declare a contract-signature check as advisory (`level: warning` — for example,
a soft-migration signature that should be flagged but not gate) has no mechanism to do so today: every
`contract_signature` finding blocks, unconditionally, regardless of what the pack's contract
definition or engine dispatch declares. This is the last of the three residual sites in the pack
severity contract family (`ISSUE-104` fixed SARIF parsing, `ISSUE-105` fixed step verdicts,
`ISSUE-106` covers the substantiveness join, `ISSUE-107` covers coverage's silent-pass inversion) —
after this issue, contract_signature is the only converted verdict site where the contract still
cannot be expressed at all, rather than being expressed and then mishandled.

## Solution

**Fix shape: thread severity through the carrier, then let it flow.**

1. Add a `Severity` field to `ContractEngineResult` (`pkg/gate/contract_verdict.go:31-36`).
2. Populate it in `produceContractEngineResults` (`cmd/backstop/gate.go` ~line 1501) from whatever the
   pack's contracts SARIF dispatch declares for that entry — this requires the contracts pack engine
   dispatch to actually carry a severity value through to this point; verify what's available there
   before assuming the field is a trivial pass-through (contracts findings may need their own
   `nonEmptySeverity`-equivalent default, mirroring `substantiveness_q1_dispatch.go:71,110-113`, so an
   empty/absent value still fails closed to `"error"`).
3. In `VerifyContractVerdict` (`contract_verdict.go:77,85,101`), replace the three hardcoded
   `Severity: "error"` literals with the carrier's resolved value (defaulted fail-closed if empty, per
   the same fail-closed law `blocksVerdict` already documents — `pkg/gate/policy.go:63-67`).
4. Decide explicitly whether ALL THREE violation-returning branches should be severity-driven, or only
   some. The unscanned-scope config-error branch (`contract_verdict.go:77`) is arguably a different
   kind of failure (a config/scan-completeness problem, not a pack-declared advisory) — compare to how
   `ISSUE-105` left `ConfigErr` branches untouched everywhere else in the codebase
   (`step_verdict_severity_test.go:122-143`,
   `TestDelegateSteps_ConfigErrorStillFailsRegardlessOfSeverity`). If the unscanned-scope branch should
   stay hardcoded `"error"` regardless of pack severity, say so in the fix rather than defaulting to
   "thread it everywhere the field now exists."

**Update the coupled test.** `TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy`
(`step_verdict_severity_test.go:163-193`) must be revised as part of this fix: once `Severity` flows
through the carrier, its CLASS-3 framing (comment at `step_verdict_severity_test.go:148-162`) is
stale, and the test needs a genuine warning-input case constructed via the now-populated `Severity`
field on `ContractEngineResult{}`, asserting `StepContractSignatureScopedFunc` reports `"warning"` for
it — the same shape `TestStepArtifactValidation_DeclaredWarningDoesNotFailWithoutPolicy`
(`step_verdict_severity_test.go:76-98`) already provides for its site. Name this coupling in the PR/
plan so the update is deliberate, not an incidental side effect discovered by a failing test.

**Blast-radius discipline.** Measure `contract_signature` step verdicts on backstop-core's own dogfood
run and at least one fixture consumer before and after; today's behavior (every contract violation
blocks) should be the expected floor — any flip must be limited to a pack that explicitly declares a
non-default severity on a contract entry, and backstop-core's own contract usage should show zero
flips unless it has such a declaration.

## Notes / references

- Ratified contract record: `PLAN-ISSUE-020`.
- Hop 1: `ISSUE-104` (SARIF severity descriptor fallback, `a42b065`).
- Hop 2: `ISSUE-105` (step verdicts ignore severity without a policy entry, `d7d777c`) — the plan
  record (`PLAN-ISSUE-105`, `pkg/gate/step_verdict_severity_test.go:145-193`) is where the class-3
  reclassification of this exact site was discovered mid-implementation and documented as evidence;
  this issue is the fix that reclassification was deferred pending.
- Live witness / self-reporting coupling:
  `TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy`
  (`pkg/gate/step_verdict_severity_test.go:163-193`) — its premise (`Violations[0].Severity == "error"`
  by construction) flips false the moment this issue's fix lands; update it deliberately as part of
  the fix, per its own inline warning.
- Family: `ISSUE-106` (substantiveness join discards pack severity, value exists upstream and is
  discarded) and `ISSUE-107` (coverage warning-only step reads as pass, inverted silent-not-blocking
  failure) are sibling residuals from the same audit; this issue is the only one where the value
  cannot be represented at all rather than being represented and then mishandled.
