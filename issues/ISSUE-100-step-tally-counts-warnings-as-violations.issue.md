---
title: "Step Tally Counts Warnings As Violations"
schema_version: issue/v1

issue:
  id: ISSUE-100
  title: "Step Tally Counts Warnings As Violations"
  type: bug
  status: open
  created: "2026-07-28"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Step Tally Counts Warnings As Violations

## Problem

A `StepResult`'s displayed violation count and `GateResult.total_violations` count every entry
in `StepResult.Violations` identically, regardless of `Violation.Severity`. Warning-severity
entries (advisory notices, e.g. a coverage-exclusion suppression notice) inflate the same tally
as blocking `severity: error` entries, so a step that has exactly one real, blocking problem can
read as having two.

Two call sites do this blind `len()` count:

- `pkg/gate/result.go:225` — `r.TotalViolations += len(s.Violations)`, the JSON envelope's
  `total_violations` field.
- `pkg/gate/output.go:61,80` — `violationCount := len(step.Violations)` /
  `fmt.Sprintf("  (%d violations)", violationCount)`, the human table's per-step tally line.

Measured instance (implementer-020-final, analyzing CI run 30389988184, 2026-07-28):
`coverage_threshold` rendered as `fail  (2 violations)` in the human table. The JSON for the same
run proves that "2" is one `severity: error` entry (the actual coverage shortfall,
`pkg/gate/step_coverage.go:204-209`) plus one `severity: warning` entry (a coverage-exclusion
suppression notice) — one blocking problem, not two. A reader sees "2 violations" and reasonably
concludes there are two things to fix; there is one.

The step's PASS/FAIL verdict logic is correct and was verified separately: `step_coverage.go:214-219`
only flips `status` to `"fail"` when it finds a `severity == "error"` entry, so the gate never
mis-blocks or mis-passes because of this. This is a display/reporting defect, not a verdict
defect — `total_violations` and the human tally are the surfaces that lie, not `Pass`/`status`.

**Systemic, not coverage-specific.** Any step that emits a `warning`-severity `Violation` rides
the same blind count. `pkg/gate/requirement_traceability.go` and `pkg/gate/status_drift.go` both
already distinguish `Severity == "error"` from `Severity == "warning"` for their own verdict
logic (`requirement_traceability.go:290,307,311`; `status_drift.go:99,103`) — the severity
distinction exists and is populated correctly at the point violations are created. It is only the
tally at the reporting layer that discards it.

The exclusion-notice case is the motivating instance and illustrates why this matters beyond
cosmetics: a coverage-exclusion suppression notice exists specifically to tell the reader "this
gap is intentional, here's why" — but by riding the same violation count as a real failure, it
makes the step look like it has MORE problems, which partially defeats the notice's own purpose.

## Impact

Any consumer that reads `total_violations` (JSON) or the per-step `(N violations)` line (human
table) as a severity-blind count of "how many things are wrong" gets an inflated number whenever
a step carries advisory warnings alongside — or instead of — real failures. This is loud-not-
blocking debt (the verdict is correct; only the count is misleading), but it directly undermines
trust in the tally as a first-read signal, which is the tally's entire purpose.

## Notes / references

- Found by implementer-020-final analyzing CI run 30389988184, 2026-07-28, while investigating a
  `coverage_threshold` step that displayed `fail (2 violations)`.
- Verified: the verdict logic at `pkg/gate/step_coverage.go:214-219` is correct (flips `fail` only
  on `severity == "error"`); this issue is scoped to the count/display layer only.
- Cross-reference: PLAN-ISSUE-020 — discovery context; the coverage-exclusion suppression-notice
  mechanism that produced the motivating instance.
- Cross-reference: ISSUE-099 — sibling gate-output ergonomics issue (single-invocation
  human+JSON emission), same reporting-layer neighborhood, independent defect.

## Solution

Two fix shapes, both viable; recommending the second.

**(a) Split warnings out of `StepResult`.** Add a separate slice (e.g. `Notices []Violation`) so
warning-severity entries never enter `Violations` at all. Correct by construction but a shape
change touching every step that can emit a warning-severity entry, plus the JSON schema and any
consumer that iterates `Violations` expecting it to include warnings (e.g. baseline
comparison, which currently walks `Violations` including warnings for fingerprinting —
`pkg/gate/baseline.go:218`).

**(b) Render counts by severity, recommended.** Keep `Violations` as the single carrier (no shape
change, no schema bump) and change only the two counting sites to split by `Severity` when they
render: `total_violations` becomes a count of `severity == "error"` entries (or the JSON gains a
second field alongside it, e.g. `total_warnings`), and the human table's per-step line changes
from `(N violations)` to something like `(1 blocking, 1 notice)`. This is local to the renderer
arithmetic in `pkg/gate/result.go:225` and `pkg/gate/output.go:61,80` — no step function, no
schema, and no other consumer needs to change.
