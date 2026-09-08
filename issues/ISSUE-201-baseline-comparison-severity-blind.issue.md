---
title: "Baseline Comparison Severity Blind"
schema_version: issue/v1

issue:
  id: ISSUE-201
  title: "Baseline Comparison Severity Blind"
  type: bug
  status: open
  created: "2026-09-08"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Baseline Comparison Severity Blind

## Problem

The aggregate `baseline_comparison` step fails the gate on ANY net-new entry, counted raw,
regardless of severity. `pkg/gate/gate.go:203-211` (at `7546cc9`, the shipped 0.2.0 build):

```go
if len(comparison.NewViolations) == 0 { ... return pass }
return StepResult{StepName: StepBaselineComparison, Status: "fail", ...
    Reason: fmt.Sprintf("%d new violations beyond baseline%s", len(comparison.NewViolations), ...)}
```

Every other verdict in the gate is severity-aware. `pkg/gate/policy.go:80-82` defines
`blocksVerdict` as "anything not `warning`", `StepVerdict` (`policy.go:125-133`) returns
`warning`, never `fail`, when only warning-severity entries remain, and `ApplyPolicy`
(`policy.go:230-245`) fails a dimension only on `countedBlocking`. The aggregate step is the one
place that ignores this contract: a warning-severity advisory that is "new" against the baseline
fails the whole gate, while the same advisory in the dimension that produced it reports
`⚠ warning` and passes.

## Why it bites in practice (observed 2026-09-08, `bclabs-brief-builder`)

Violation identity includes a per-region content hash (`pkg/gate/baseline.go:216-221`). An
ordinary prose edit to a bundle — reworded requirement text, a versions-log line — re-hashes the
region under every requirement in that bundle. Each `requirement_traceability_advisory` entry
beneath it ("requirement REQ-NNN has no implemented-spec coverage") is then "new" by identity even
though the baseline already lists the same requirement, same file, same message.

Observed gate output on a tree with a reviewed, green implementation waiting to land:

```
  requirement_traceability_advisory ⚠ warning  (124 warnings)
  baseline_comparison               fail       (13 warnings)  (13 new violations beyond baseline)
  Steps: 10 passed, 1 failed, 1 skipped, 2 warned
  Total violations: 0 (+142 warnings)
FAIL
```

Zero violations, thirteen re-hashed advisories, gate red. The pre-commit hook refused the
commit — and the only thing that refreshes the baseline is a green run on the commit it is
refusing. A `git stash` of the bundle edits alone turned the gate green, isolating the cause.

Two consequences:

1. **Editing a bundle's prose is gate-blocking** in any consumer whose bundles carry `defined`
   requirements not yet covered by an implemented spec — which is every bundle between `defined`
   and delivery. The advisory exists precisely to tolerate that window.
2. **The advisory tier is not advisory.** The `_advisory` step names, the `⚠ warning` rendering,
   and `Total violations: 0` all tell the consumer nothing blocks; the aggregate step contradicts
   them silently, and the reason string calls warnings "violations".

## Root cause

`baseline_comparison` decides its verdict on `len(comparison.NewViolations)` instead of on
`blockingViolations(comparison.NewViolations)` / `StepVerdict`. The region-hash identity is a
separate design choice (it makes an advisory "new" when only its surrounding prose moved) and is
worth a look, but the verdict bug is what turns that churn into a red gate.

## Sanctioned escape in the shipped build

Any configured `enforcement.policy` neutralizes the aggregate step (`policy.go:180-186`,
"superseded by per-dimension enforcement policy") and each policied dimension applies
severity-aware ratcheting via `ApplyPolicy`. backstop-core's own `backstop.yml` runs this way.
`bclabs-brief-builder` adopted the same block on 2026-09-08 to land its work, with a comment
pointing at this issue. Consumers without a policy block — the documented default — still hit
the bug.

## Fix shape (for the planner to derive)

- Verdict: `baseline_comparison` returns `fail` only when `blockingViolations(NewViolations)` is
  non-empty; `warning` when only warning-severity entries are new; `pass` otherwise. Report the
  full `NewViolations` list unchanged (non-blocking must not mean invisible — same rule as
  `StepVerdict`). Reason string distinguishes "N new violations" from "N new warnings".
- Lock it: a test where the accumulated steps carry only warning-severity net-new entries and the
  aggregate step's verdict is `warning`, plus the existing pass/fail cells unchanged. Pair with
  `TestPolicy_BlockAllCodeAgreesWithStepVerdict`'s intent: a consumer with no policy block and a
  consumer with every dimension `block`/`new-code` reach the same verdict on the same tree.
- Consider (may be a separate issue): whether `requirement_traceability_advisory` identity should
  key on requirement id + file rather than region hash, so prose churn does not re-mint it.

## Related

- `bclabs-brief-builder` `ISSUE-071` — the consumer-side write-up of the coupling (a flaky CI
  journey froze the baseline; this bug turned the frozen baseline into a red local gate).
- `pkg/gate/policy.go` `StepVerdict` / `blocksVerdict` — the contract the aggregate step breaks.
