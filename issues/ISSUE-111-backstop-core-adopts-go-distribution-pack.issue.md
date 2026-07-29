---
title: "Backstop Core Adopts Go Distribution Pack"
schema_version: issue/v1

issue:
  id: ISSUE-111
  title: "Backstop Core Adopts Go Distribution Pack"
  type: enhancement
  status: open
  created: "2026-07-29"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Backstop Core Adopts Go Distribution Pack

## Problem

ISSUE-101's Verification section names backstop-core **adopting** the `backstop-ai/go-distribution`
pack as part of the pack's acceptance bar: "backstop-core itself adopts the pack and its rules
gate the existing hand-written trinity — proving the rules assert invariants generically, not
against backstop-core's own literals." PLAN-ISSUE-101 proved that consumption claim, but only in
**discarded git worktrees** — TASK-010 (local install) and TASK-017 (remote install, post-
publication) both install the pack into a detached worktree, run the rule checks, and then
discard the worktree. Neither task installs the pack durably into core's own committed
`backstop.yml` / `backstop.lock`. Confirmed against the live tree: as of 2026-07-29,
`backstop.lock` records six installed packs (`backstop-self`, `cobra-cli-standards`,
`go-contracts`, `go-standards`, `go-substantiveness`, `go-toolchain`) and `go-distribution` is not
among them.

This was a deliberate omission, not an oversight — PLAN-ISSUE-101 records why: mutating the live
repo was out of scope for that plan, and "adding a pack to core's committed backstop.yml /
backstop.lock can move core's gate verdict for reasons that have nothing to do with the trinity."
That's a real worry worth taking seriously before adopting any new pack. But the residual — ISSUE-
101's own Verification section names an outcome nothing yet owns — needs an artifact, not prose in
a completed plan, per this repo's alignment rule that a predating artifact's promise gets tracked
openly rather than left to drift.

**Adoption starts from measurement, not hope.** PLAN-ISSUE-101's TASK-016 re-derivation
(implementer-101, 2026-07-29) already measured the two facts that make this a low-risk, bounded
piece of work rather than an open question:

- Core's shipped release trinity (`.goreleaser.yml`, `.github/workflows/release.yml`,
  `.github/workflows/tag-integrity.yml`) yields **ZERO findings** from all nine `go-distribution`
  rules, confirmed against **both** a local pack install and a remote install
  (content hash `d5835e18...`, equal across both install paths — the reproducibility proof
  ISSUE-101's own Verification section asks for).
- The existing installed packs dispatch **alongside** `go-distribution` unchanged, measured in a
  full `backstop gate --all` run: `go-standards` 32 findings, `go-toolchain` 53, `backstop-self` 2,
  `go-distribution` 0.

Together these make the "could-move-the-verdict" worry from PLAN-ISSUE-101 **measurable rather
than hypothetical**, and the measurement says adoption is verdict-neutral: the new pack adds zero
findings of its own and does not disturb what the other packs already report.

## Scope

- `pack add backstop-ai/go-distribution@0.1.0` (or the then-current published version) into
  backstop-core's live checkout, committing the resulting `backstop.yml` and `backstop.lock`
  changes.
- One verification run: `backstop gate --all` (or equivalent full-scope gate) confirming the
  verdict does not move — i.e. `go-distribution` continues to report zero findings against core's
  trinity and the other packs' finding counts are unchanged, consistent with the pre-adoption
  measurement above.

This issue does not ask for anything beyond durable installation and that one confirmation run.

## What this issue is NOT

Full retirement of backstop-core's hand-baked CI (`ci.yml`'s `gate`/`baseline` jobs invoking Go
tooling directly instead of going through `backstop gate`) is a **different, larger** piece of
work than adopting one release-tooling pack, and this issue does not claim to close it. Both
ISSUE-087 and DIR-001 cite a now-stale pointer for that gap ("ISSUE-086"); ISSUE-101 already
corrected that citation once (its own "Stale-citation correction" section), and the corpus has
moved again since:

- **ISSUE-020** ("Cross-Platform Sandbox: Linux Is a Hard Error"), which the hand-baked-CI fact
  was welded onto by 2026-07-27 founder ruling, is now **CLOSED** — delivered by PLAN-ISSUE-020
  across five forced-order phases, Linux sandbox work included. It is no longer a blocker for
  anything.
- **ISSUE-086** ("The Published Baseline Artifact Is Generated With Zero Packs Installed") remains
  **open** and covers a narrower, different defect: the `baseline` job in `ci.yml` runs
  `backstop baseline generate` with no `pack install` step, so the published reference baseline
  has zero pack-engine findings structurally, not because the codebase is clean.
- **BUNDLE-015 REQ-018** (the committed-but-unbuilt CI recipe pack — the packs-only delivery
  mechanism that would eventually let `ci.yml` itself be recipe-sourced rather than hand-written)
  has, per DIR-019, no committed timeline yet; 13 of BUNDLE-015's 24 requirements are covered, and
  REQ-018 is one of the 11 that are not.

So: the remaining hand-baked-CI territory today lives in REQ-018/ISSUE-086, not on Linux sandbox
work. This issue's adoption of `go-distribution` is orthogonal to that territory — it gates the
release *trinity* backstop-core already ships, not the `ci.yml` `gate`/`baseline` job shapes.

## References

- `issues/ISSUE-101-go-distribution-pack.issue.md` — the Verification section naming core's
  adoption as acceptance criteria; its "Stale-citation correction" section, the first correction
  of the ISSUE-086 pointer.
- `plans/PLAN-ISSUE-101-go-distribution-pack.plan.yml` — TASK-010 (worktree proof, local install),
  TASK-017 (worktree proof, remote install, hash-equality check), and TASK-016 (which named this
  issue as a deliberate deferral, citing "adding a pack to core's committed backstop.yml /
  backstop.lock can move core's gate verdict for reasons that have nothing to do with the
  trinity").
- `issues/ISSUE-086-published-baseline-generated-packless.issue.md` — current, narrower owner of
  the packless-CI defect class.
- `issues/ISSUE-020-cross-platform-sandbox-linux-noop.issue.md` — closed; no longer the owner of
  any hand-baked-CI fact.
- `directives/DIR-019-pack-recipe-capability.directive.md` — REQ-018 coverage status (13 of 24
  BUNDLE-015 requirements covered; REQ-018 among the uncovered 11).
- `backstop.yml` / `backstop.lock` — current installed-pack roster, confirming `go-distribution`
  is not yet present.
