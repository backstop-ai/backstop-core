---
title: "Land Approved Artifact Lifecycle Reference"
schema_version: issue/v1

issue:
  id: ISSUE-199
  title: "Land Approved Artifact Lifecycle Reference"
  type: enhancement
  status: open
  created: "2026-09-02"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Land Approved Artifact Lifecycle Reference

## Problem

`/reference/#artifact-lifecycle-and-closure` on `main` is still the Seed 1
summary. SPEC-072 parked CLAIM-030 and CLAIM-031 there. CLAIM-030 still says
plans progress from `draft` to `approved` and `in-progress`. Live plan
statuses are `draft` → `ready` → `implementing` → `completed`.

The section lists some states and issue-close fields. It does not lock the
state machine: legal transitions, what must be true before each move, what
validators and the gate begin enforcing, what entering the state enables, or
how terminal/closure traceability is established.

Entity pages already list per-noun statuses. Model already owns the two work
paths and ARCH-001. `/reference/#source-traceability` owns gate/evidence
provenance (CLAIM-007 / CLAIM-008). None of those surfaces is the canonical
lifecycle machine.

A local prototype of the approved expansion exists on
`cursor/evaluate-copy-local-preview-e296`. This issue records the approved
direction so it can be planned and landed. The prototype is evidence of
intent, not a close and not a substitute for issue → plan.

ISSUE-197 owns the entity lookup pages. This issue owns the Reference
section only. Do not create a new URL. Do not add more lifecycle detail to
Model.

Do not `/plan` this issue in this filing. Plans wait until Brandon approves
the copy.

## Solution

Land the approved `/reference/#artifact-lifecycle-and-closure` expansion:
live states, legal transitions, preconditions, validator/gate enforcement,
what each state enables, and terminal/closure traceability. Keep CLAIM-030
and CLAIM-031 IDs on this section. Keep `/reference/#source-traceability`
scoped to gate/evidence provenance.

Do not reopen BUNDLE-032's visitor-journey charter. Do not restyle the whole
`/reference/` page in this issue. Do not `/plan` this issue in this filing.

### Approved section contract

Heading stays `## Artifact lifecycle and closure {#artifact-lifecycle-and-closure}`.

The section must answer:

- What states exist?
- What transitions are legal?
- What must be true before each transition?
- What validators/gates begin enforcing at that state?
- What does entering that state enable?
- How is terminal/closure traceability established?

Use live artifact states, not stale Seed 1 language.

**Plan:** `draft` → `ready` → `implementing` → `completed`, with terminal
alternatives `replaced`, `canceled`, and `obsoleted` documented separately.
`draft`, `ready`, and `implementing` are the same shape to the validator.
Terminal plans are exempt from phase and task completeness. A `completed`
plan whose mandated tests are missing may pass the validator; the gate does
not.

**Issue:** `open` → `ready` → `in-progress` / `blocked` → `closed`.
Requirements, claims, verification, implementation, and contracts are
optional at `open` and required from `ready` onward. Close requires a
`## Resolution` section and at most one of `delivered_by` (completed
`PLAN-ISSUE-NNN`) or `resolved-by` (typed artifact id, commit SHA, or
pull-request URL). Both on the same close is illegal. `replaced` /
`canceled` / `obsoleted` are terminals with successor fields where the
schema names one. An issue does not get a spec.

**Spec:** `draft` → `ready-for-implementation` → `implemented`. Live specs
carry requirements and claims. Each spec has exactly one plan. A spec comes
from a bundle. Terminals: `replaced`, `canceled`, `deprecated`, `obsoleted`.

**Bundle:** `idea` → `exploring` → `defined` → `ready` → `delivered`.
Bundles start at `exploring` with real open questions. The user drives
promotion. `defined` and `ready` require Draft Requirements, Draft Design
Decisions, Spec Seeds, Version History, `requirements[]`, and
`solution.approach`. `ready` additionally requires success criteria and
assumptions. Terminals: `delivered`, `replaced`, `canceled`, `deprecated`.

Keep these Seed 1 needles on this section:

- `Bundles progress through \`idea\`, \`exploring\`, \`defined\`, and \`ready\``
  (ART-003 / CLAIM-030)
- `` `delivered_by` names a completed plan `` (ART-004 / CLAIM-031)

CLAIM-030 must use live plan states (`ready` / `implementing` / `completed`),
not `approved` / `in-progress`. Update `docs/_data/evidence-inventory.yml`
`statement` / `statement_markdown` to match the claim bytes on the page.

Directive, ADR, and Capability status vocabularies stay on their entity
pages. This section may point at those pages. It does not absorb them.

Do not mix artifact lifecycle/status semantics into
`/reference/#source-traceability`.

### Presentation (paper/ink)

The lifecycle machine lives on `/reference/`, not a new URL. `page_kind` is
already `reference`. Landing must give `/reference/` the same paper/ink
chrome as Evaluate, Model, Adopt, Extend, and entity pages. Brandon asked
for that match after the first draft left Reference on the dark field-guide
surface.

When editing `site.css` media queries, do not drop
`[data-page-kind="home"] .nav` rules. Playwright `testMatch` stays
`public-site.spec.ts`. Packs stay external.

### Rejected — do not restore

- CLAIM-030 plan language `approved` and `in-progress`
- A new lifecycle URL
- Adding this machine onto `/model/`
- Dumping per-noun Status tables from the entity pages into Reference
- Mixing CLAIM-007 / CLAIM-008 gate-evidence provenance into this section
- Folding ISSUE-198 (Extend) into this issue
- Independent "improvement" of locked copy
- Slide-frame chrome, slogans, manufactured paradoxes

### Lockstep landing will need

Closed-world allowances for every new non-claim, non-jlink, non-heading
block under this section.

CLAIM-030 / CLAIM-031 owner route `/reference/` and anchor
`artifact-lifecycle-and-closure` stay. Page test
`extend-reference-contributing.sh` must still find the ART-003 and ART-004
needles and the claim bytes.

Seed 1 SPECs still quote the old CLAIM-030 plan statuses. Landing must
update those contract literals. Do not update SPECs in this filing — only
record that landing will.

### Scope constraint

This issue owns `/reference/#artifact-lifecycle-and-closure` only, plus the
smallest corresponding inventory, allowance, page-test, and Seed 1 contract
updates needed to land that section without silent divergence from SPEC-072
/ SPEC-075 / SPEC-076.

Out of scope — do not absorb:

- Homepage (ISSUE-190)
- Evaluate (ISSUE-193)
- Model (ISSUE-195), including ARCH-001 / `#delivery-lifecycle`
- Adopt (ISSUE-196)
- Entity lookup pages (ISSUE-197)
- Extend pack-authoring rewrite (ISSUE-198)
- `/reference/#source-traceability`
- `/reference/#pack-artifact`
- Other Reference sections
- Vendoring packs
- `/plan` of this issue in this filing

## References

- Local prototype: branch `cursor/evaluate-copy-local-preview-e296`. Primary
  source `docs/reference.md` `#artifact-lifecycle-and-closure`.
- Live states: `artifacts/issue/v1/schema.json`, `artifacts/spec/v1/schema.json`,
  `artifacts/bundle/v2/schema.json`, `artifacts/plan/v1/schema.json`.
- Enforcement: `pkg/validate/issue.go` (`traceabilityRequired` from `ready`
  onward; `delivered_by` / `resolved-by` close), `pkg/validate/bundle.go`
  (maturity gates), `pkg/validate/plan.go` (terminal exemption),
  `pkg/validate/spec.go` (live-work completeness).
- SPEC-072 CLAIM-030 / CLAIM-031 / ART-003 / ART-004.
- ISSUE-197: entity pages remain lookup-by-noun Status tables.
- Live site still old: `https://backstop.sh/reference/#artifact-lifecycle-and-closure`.

### Existence-in-world check

Before filing, `issues/` and `bundles/` were searched for artifact lifecycle
reference, state machine, CLAIM-030, and `/reference/#artifact-lifecycle-and-closure`.

- BUNDLE-032 / SPEC-072 own visitor-journey IA and Seed 1 Reference. This
  issue does not reopen that charter. It records the approved rewrite of
  this one section after visitor review.
- ISSUE-197 owns per-noun Status tables on entity pages.
- ISSUE-031 (closed) added terminal states; it does not own visitor copy.
- No open issue in this working tree owns this Reference section rewrite.
