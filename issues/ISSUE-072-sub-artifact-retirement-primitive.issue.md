---
title: "Sub-Artifact Retirement Primitive + Finding-Resolved-By-Retirement Close Path"
schema_version: issue/v1

issue:
  id: ISSUE-072
  title: "Sub-Artifact Retirement Primitive + Finding-Resolved-By-Retirement Close Path"
  type: enhancement
  status: open
  created: "2026-07-21"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Sub-Artifact Retirement Primitive + Finding-Resolved-By-Retirement Close Path

## Problem

The traceability model is **add-only at the requirement/claim level**, and **whole-artifact-only**
at the retirement level. There is no first-class way to retire *part* of a still-live artifact, and
no clean way to close an issue whose resolution *is* such a retirement. Two intertwined gaps:

### Gap 1 — no sub-artifact retirement primitive

Retirement statuses are **artifact-level terminals only**: `replaced`, `obsoleted`, `canceled`,
`deprecated` (`pkg/validate/terminal.go:25`; `obsoleted` is purpose-built for delivered-then-removed
work and requires an `obsoleted-by` pointer). Individual **requirements and claims carry no `status`
/ `retired` field** — the spec schema has no per-item lifecycle marker.

So when a spec stays live but *some* of its requirements must be retired (their claims, mandated
tests, and code deleted), the only available move is **excision**: delete the requirement + claims +
plan tasks + tests + code in one commit and bump `spec_version`. That keeps the gate clean (no
dangling claim, no orphaned mandated test → no `artifact_status_drift`), but:

- The structured record that those requirements **ever existed, were implemented, then retired** is
  **lost from the artifact**. It survives only in git history + prose + whatever ADR/issue narrates
  the "why." The amended artifact reads as if the retired requirements never existed.
- There is no mechanical link from a retired requirement to the decision that retired it (contrast
  the *required, typed* `replaced-by` / `obsoleted-by` pointers that whole-artifact retirement gets).

### Gap 2 — no close path for a finding resolved by retirement

An issue whose resolution is "**retire part of another artifact**" has nowhere clean to land.
Closing an issue wants a self-contained proof chain (implementation / verification / claims /
mandated tests), per the `closed-requires-traceability` enforcement in
`artifacts/issue/v1/schema.json`. A *retirement* has none of that — its "delivery" is a **deletion**
plus edits to a *different* artifact (the amended spec, the ADR). So the issue is forced to stay
`open` even though the work is genuinely, verifiably **done** (the code is gone; the gate is green).

This is the same missing-primitive showing up as a loose end, and it is adjacent to but distinct
from [[ISSUE-071]] (vacuous *closed*): ISSUE-071 is "closed with no proof at all"; this is "the proof
is a deletion + a cross-artifact amendment, which the close path has no shape for."

## Surfacing evidence (real case)

bclabs-portal, 2026-07-21. A real-Supabase deploy of SPEC-010 (evidence storage) proved its
`storage.objects` RLS **cannot deploy** (owner constraint) — filed as portal **ISSUE-002**, decided
in portal **ADR-0001** (enforce evidence tenancy at the DB pointer layer; the `storage.objects` RLS
was a redundant second copy of SPEC-007's `rm_evidence` gate). Resolution = **retire** SPEC-010
REQ-004/REQ-005 + their claims + the `storage-policy.ts` contract, delete the code/tests, and gut
`drizzle/0007` — while SPEC-010 stays live (REQ-001/002/003/006 remain).

Handled by excision + `spec_version 1.0.0 → 1.1.0` + the ISSUE-002/ADR-0001 narrative. The gate went
green (and the drift check correctly caught a first-pass miss where the *plan* still mandated the
deleted tests — the machinery worked). But **portal ISSUE-002 is stuck `open`**: there is no way to
close a finding whose resolution is a retirement, because closure wants a self-contained test chain a
retirement doesn't have. That stuck-open issue is the tell that the primitive is missing.

## Direction (to be specified in a plan when prioritized)

Explore a **sub-artifact retirement** concept and its close path. Candidate shape (not committed):

- A per-requirement / per-claim lifecycle marker (e.g. `retired: true` or `status: retired`) that
  keeps the item **in the artifact for history** rather than excising it, carrying a **required,
  typed `retired-by` pointer** to the deciding ADR/issue — mirroring `replaced-by`/`obsoleted-by`
  at the sub-artifact grain.
- Drift/coverage/contract dimensions treat a retired requirement like an absent one for
  *live-work* purposes (no mandated-test obligation, no dangling), but the trace is preserved and
  the "why" is mechanically linked.
- A **finding-resolved-by-retirement** close path: an issue may close when its resolution is a
  retirement recorded via the above (issue → ADR → retired requirement(s)), without fabricating a
  self-contained implementation/claims block it doesn't have — analogous to the existing
  `resolved-by` / `delivered_by` relaxations, extended to the retirement case.

Open questions for the plan: excise-vs-mark (does keeping retired items bloat artifacts over time?);
whether this reuses `obsoleted-by` semantics at a finer grain or needs a distinct field; interaction
with `spec_version` bumps and requirement versioning (`REQ-NNN@X.Y.Z`); whether a bundle requirement
revision (portal REQ-013 mechanism→guarantee) needs the same treatment.

## Impact

Foundational, not urgent. Nothing reds today — excision + version bump already passes the gate. The
cost is **silent**: retirement history lives outside the artifacts, retired requirements have no
mechanical link to their deciding ADR, and findings-resolved-by-retirement are forced to stay `open`
(a growing class as the issue→ADR reactive track scales and as more delivered work gets walked back).
Worth designing before sub-artifact retirement becomes common, but not blocking in-flight work.

## Acceptance

- A requirement/claim can be marked retired in-place (history preserved) with a required typed
  pointer to the deciding ADR/issue; live-work dimensions (`artifact_status_drift`, coverage,
  contract) treat it as non-obligating, exactly like excision does today, with a regression test
  proving no drift and a preserved+linked trace.
- An issue whose resolution is a recorded retirement can reach a success-terminal status without a
  fabricated self-contained proof block, via an explicit retirement-close relaxation — with a
  regression test proving the portal ISSUE-002 shape (finding → ADR → retired requirements, code
  deleted) closes cleanly, while a bare vacuous close still trips [[ISSUE-071]]'s check.
- Whole-artifact retirement (`replaced`/`obsoleted`/etc.) behavior is unchanged.

## Notes / references

- Surfaced 2026-07-21 from the bclabs-portal ISSUE-002 / SPEC-010 amendment / ADR-0001 change:
  a deploy-blocker finding resolved by retiring two requirements of a still-live spec.
- Verified against the model: `pkg/validate/terminal.go` (artifact-level retirement statuses +
  typed `replaced-by`/`obsoleted-by` refs); no per-requirement/claim `status` field in the spec
  schema; `artifacts/issue/v1/schema.json` `closed-requires-traceability`.
- Related: [[ISSUE-071]] (vacuous done issue) — complementary; this is the retirement-shaped close
  path, that is the zero-proof close hole.
- Backlog capture only. Will receive a PLAN when prioritized; mandated test names deferred to that
  plan, not declared here.
