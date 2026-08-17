---
title: "Step Tally Counts Warnings As Violations"
schema_version: issue/v1

issue:
  id: ISSUE-100
  title: "Step Tally Counts Warnings As Violations"
  type: bug
  status: closed
  created: "2026-07-28"
  closed: "2026-08-17"

complexity:
  scope: contained
  uncertainty: known
  risk: safe

delivered_by: PLAN-ISSUE-100
---

# Step Tally Counts Warnings As Violations

## Resolution

Fixed by PLAN-ISSUE-100 (`status: completed`, commit `2d69e14`). The tally/render half retained by
this issue (see Amendment — the verdict half was already fixed separately) is now severity-aware:

- `pkg/gate/result.go` gained `tallyBySeverity(violations []Violation) (blocking, warnings int)`,
  implemented strictly over `pkg/gate/policy.go`'s existing `blocksVerdict` predicate — never a
  hand-rolled `Severity` comparison of its own. `GateResult` gained a new `TotalWarnings` field
  (JSON `total_warnings`, no `omitempty`). `TotalViolations` now counts only blocking entries
  instead of every entry regardless of severity, fixing the CI-run-30389988184 defect described
  above where one blocking error plus one warning-severity notice reported as "2 violations".
- `pkg/gate/output.go`'s human report splits per-step counts into blocking/warnings — the summary
  row, the "Total violations:" footer, and per-entry `[warning]` detail markers — with zero
  rendering churn on the common (no-warnings) case.

**Important correction on the shipped spelling.** This issue's own Solution (b) suggested
`total_violations` become a count of `severity == "error"` entries. That spelling was deliberately
**not** used: an unset `Severity` still blocks per the ratified `blocksVerdict` contract
(`pkg/gate/policy.go`, landed under the Amendment's verdict-half fix), so `== "error"` would
misclassify a real, blocking failure as a mere notice — fail-open. The shipped implementation
calls `blocksVerdict` directly instead, keeping the tally and the verdict permanently in agreement.

**Companion artifact corrections landed alongside**, filed as their own follow-ons rather than
folded into this close: SPEC-010 REQ-008's prose (which also described `total_violations` as an
unconditional count) was corrected in commit `0670133`; DIR-032 item 7's stale `severity ==
"error"` prescription — the same refuted spelling this issue's own Solution text carried — is
being corrected separately, in flight at close time.

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

> **Superseded (2026-07-28) — the claim above is INCOMPLETE, not wrong.** `step_coverage.go`'s
> own status computation is correct as described. But it is not the last word: the gate's policy
> layer runs after the step and can overwrite `status`. See the Amendment section below — a
> sibling verdict-level defect was found downstream of this one and has been split into its own
> fix lane. This issue's own scope (the tally/render half) is unchanged by that split.

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

## Amendment (2026-07-28) — a verdict-defect half was found and split out

Continued investigation past the original display-only framing above surfaced a second,
verdict-level defect in the same neighborhood, proven in CI run 30395875188: a `coverage_threshold`
step reported status `"fail"` with exactly ONE violation, and that violation's own `Severity` is
`"warning"` (the coverage_exclusion notice) — the structured JSON for that run confirms no hidden
`severity: "error"` entry is present. So this is not the tally miscounting two problems as one
line; this is the gate BLOCKING on a violation that is, by its own severity field, non-blocking.

**Mechanism.** `step_coverage.go:214-219`'s own status computation is correct, exactly as the
original Problem section above states — but it is not the layer that has the last word. The gate's
policy layer runs after the step and overwrites `status` without consulting `Violation.Severity`
at all:

- `pkg/gate/policy.go:126-131` — `applyPolicy`'s default (block) branch: `if len(counted) > 0 {
  s.Status = "fail" }`. `counted` is `s.Violations` (or the baseline-grandfathered subset of it);
  its length is severity-blind.
- `pkg/gate/policy.go:190-205` — `applyScopedPolicy` builds a `blocking` slice by appending every
  violation whose EFFECTIVE POLICY LEVEL (`eff.Level`, resolved from the dimension/source policy
  configuration) is not `PolicyWarn` (line 191: `if lvl == PolicyWarn { warned++; continue }`).
  That `lvl` is the POLICY's configured level for the dimension/source — never `v.Severity`. A
  violation the step itself marked `Severity: "warning"` still lands in `blocking` and flips
  `status` to `"fail"` whenever its governing policy level defaults to `block` (the default for
  any dimension without an explicit `warn` override).
- Confirmed severity-blind by inspection: `grep -c Severity pkg/gate/policy.go` returns 0 — the
  policy layer never reads the field the step layer populates.

Net effect: the coverage-exclusion mechanism exists specifically to make an intentional coverage
gap VISIBLE-BUT-NON-BLOCKING (a `severity: "warning"` notice), and the policy layer blocks on its
own notice — the opposite of what the mechanism was built to do, and a direct violation of the
founder's loud-not-blocking law (CLAUDE.md, "Enforcement philosophy": *"Block defects + broken
promises; warn-with-guidance for un-adopted capability. The enemy is silent/vacuous green, not
passing."*). A notice is not a defect; blocking on it is exactly the failure mode that law rules
out.

**Disposition.** This verdict-defect half is being fixed in-lane under PLAN-ISSUE-020's scope
extension (orchestrator-ruled 2026-07-28, with falsifiers required in both directions plus a
measured blast-radius check before landing) rather than folded into this issue — the fix touches
`pkg/gate/policy.go`'s severity handling, a different surface than this issue's renderer/tally
scope. **ISSUE-100 retains only the original tally/render half** (`result.go:225`,
`output.go:61,80`) described in the Problem section above; the Solution section's two options
still apply to that half unchanged.

**Fixed.** The verdict-defect half landed in two commits under PLAN-ISSUE-020's scope
extension: `2e49745` (`pkg/gate/policy.go`'s `blocksVerdict` now honors `Violation.Severity` —
an explicit `warning` is exempt from verdict computation; UNSET severity still blocks,
fail-closed) and `3ac6e7f` (`cmd/backstop/pack_severity_contract_test.go`, the founder-ratified
severity contract locked across all three hops — parser `sarifSeverity`, the
`runFindingsEngine` bridge, and `blocksVerdict` — each falsified per-hop).

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
  mechanism that produced the motivating instance; also the lane now carrying the split-off
  verdict-defect fix (see Amendment section above).
- Cross-reference: ISSUE-099 — sibling gate-output ergonomics issue (single-invocation
  human+JSON emission), same reporting-layer neighborhood, independent defect.
- CI run 30395875188 — the verdict-defect evidence (coverage_threshold `"fail"` with a single
  `severity: "warning"` violation and no hidden error entry); see Amendment section above.

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
