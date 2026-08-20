---
title: "Ci Repo Wide Sweep Debt Inventory"
schema_version: issue/v1

issue:
  id: ISSUE-094
  title: "Ci Repo Wide Sweep Debt Inventory"
  type: technical-debt
  status: open
  created: "2026-07-28"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Ci Repo Wide Sweep Debt Inventory

## Problem

CI's blocking job (`gate`, `.github/workflows/ci.yml:13-36`) is being flipped from whole-repo
`golangci-lint run ./...` to a diff-scoped `backstop gate` under PLAN-ISSUE-020 (in flight — see
`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml`, tracked by ISSUE-020). That flip is
correct and wanted: it makes CI green mean "the diff is clean," matching local dev workflow and
avoiding the false-red-on-untouched-code failure mode diff scoping exists to prevent.

But it has a side effect: repo-wide debt that a whole-repo sweep would have caught becomes
**invisible** unless something else surfaces it. Once `gate` only looks at the diff, a PR that
doesn't touch `pkg/config` can merge clean while `pkg/config` sits on lint violations nobody sees
in CI output ever again — not blocking, not even visible, just silently absent from every future
run's signal.

Founder ruling (Brandon, 2026-07-28), on being shown this gap: *"Anything else caught as part of
the repo-wide scan gets noted in a GitHub issue."* This issue is the tracker for building that
loud-but-non-blocking half: a repo-wide sweep job that never fails the build, but keeps a durable,
visible record of what it finds.

### Known baseline (measured, not projected)

The first-ever real CI run (2026-07-28) failed on whole-repo `golangci-lint`: **23 errcheck +
unparam findings**, concentrated in `pkg/config` and `pkg/check`. This is the known grandfathered
toolchain baseline — reproduces identically with a local `go tool golangci-lint run ./...`. It is
the concrete seed the rolling inventory issue starts from, not a hypothetical.

## Reference implementation and adaptation

The pattern comes from the `slotly` repo's `.github/workflows/enforce-architecture.yml`: a
diff-blocking job paired with an advisory job that auto-files GitHub issues for out-of-diff
violations. Its mechanics worth carrying over:

- **Idempotency via signature**: an HTML-comment signature embedded in the issue body (e.g.
  `<!-- arch-violation: path -->`) lets a re-run recognize "this is the same finding" and update
  in place instead of duplicating.
- **Labeling**: a dedicated label marks these auto-managed issues so they're filterable and
  distinguishable from human-filed issues.
- **Auto-close on compliance**: when a touched file comes into compliance (a subsequent diff-scoped
  gate run on that file is clean), the tracking entry closes itself with an explanatory comment
  rather than requiring manual bookkeeping.

**Adaptation required, not a straight port**: slotly's pattern files ONE ISSUE PER VIOLATING
FILE. Backstop's repo-wide golangci-lint sweep currently produces roughly **1,727 lines of
output**. Per-file issues at that volume would spam `backstop-ai/backstop-core`'s issue tracker on
day one of CI going live. The adaptation (orchestrator-recommended, founder saw and accepted the
recommendation): **one rolling inventory issue** (or at most one issue per linter/rule class, not
per file), with an internal checklist that preserves the per-file granularity and per-file
auto-close semantics *inside* that single issue's body — a checklist item per file/finding,
checked off (or the line removed/edited) as files come into compliance, rather than one GitHub
issue per file.

## Scope

An advisory (never build-failing) CI job, triggered on push to `main`, alongside the diff-blocking
`gate` job PLAN-ISSUE-020 is standing up:

- **Today**: whole-repo `golangci-lint run ./...` (the same tool/config the current `gate` job
  runs, just no longer gating the build once the diff-scoped `backstop gate` takes over that
  role).
- **Once ISSUE-091 is fixed**: `backstop gate --all`, replacing the hand-baked golangci-lint
  invocation. ISSUE-091 (`gate --all` under-reports vs. diff-scoped `backstop gate` — confirmed
  111 diff-only findings on files `--all` reported zero for) is the explicit retirement trigger:
  using `--all` for this sweep before that bug is fixed would under-report the very debt this job
  exists to surface, defeating its purpose.
- Findings are synced into a rolling inventory GitHub issue (or small, bounded set of issues split
  by linter/rule class — not by file) on `backstop-ai/backstop-core`, following the
  idempotent-signature + label + per-file-checklist-auto-close pattern above, adapted for volume.

## Honest tension to flag, not resolve here

This creates a **second debt ledger** — GitHub issues — running alongside `.backstop/baseline.json`
(the existing ratchet/baseline mechanism, see `[[project_baseline_design]]`). Both are "here is
debt backstop knows about but isn't blocking on" mechanisms, tracked in different systems with
different lifecycles. That duplication is acceptable for launch — the baseline ratchet is scoped
to what `backstop gate` itself evaluates and doesn't have an issue-per-finding UX, while this
sweep needs a human-visible, closeable unit of work — but it should not be treated as a permanent
design. Flagged here for future unification rather than solved now.

## Cross-references

- **ISSUE-091** (`gate --all` under-reports test-file findings, open, critical) — the retirement
  trigger. This sweep job stays on hand-baked whole-repo `golangci-lint` until ISSUE-091 closes;
  switching to `backstop gate --all` earlier would silently under-report relative to what the
  diff-scoped gate itself would find on the same files.
- **ISSUE-086** (published baseline generated packless, open, technical-debt) — a sibling
  "CI signal isn't what it appears to be" gap at the baseline layer; this issue's sweep job and
  ISSUE-086's baseline-packs gap are both instances of CI silently not exercising the full
  verification surface it appears to.
- **ISSUE-020** (cross-platform sandbox Linux no-op, in-progress via PLAN-ISSUE-020) — owns the
  `gate` job's cutover from hand-baked golangci-lint to `backstop gate`, diff-scoped. This issue's
  advisory sweep job is a sibling addition alongside that cutover, not a dependency of it — the
  sweep can land independently, though it's most useful once the diff-scoped `gate` job is live
  and repo-wide visibility genuinely becomes CI's blind spot.
- **BUNDLE-015 REQ-018** (CI recipe pack, the packs-only acceptance test for CI workflow
  scaffolding) — this sweep job is, in its current form, more hand-baked CI: a bespoke GitHub
  Actions job with bespoke issue-sync logic, not something a backstop pack declares. That is
  named honestly rather than papered over. A future backstop capability (once REQ-018's CI recipe
  pack exists) could plausibly absorb "sweep repo-wide, sync findings to a tracking issue" as a
  native, packs-only capability instead of a hand-written workflow step — but REQ-018 has no
  committed timeline (per DIR-019/DIR-027), so this issue's own scope does not wait on it.

## Severity and sequencing

Launch-adjacent, **not blocking v0.1.0**. The diff-blocking half (PLAN-ISSUE-020's `backstop gate`
cutover) is what unblocks release — a project can ship with CI enforcing the diff even before
repo-wide debt has a visible home. This issue is the loud-debt half: it makes existing repo-wide
debt honest and trackable once the blocking gate stops seeing it, but its absence does not block
shipping.

## Verification

No code change accompanies this filing. Verification criteria for the eventual fix (recorded so
`ready`-promotion inherits a target):

- A CI job runs on push to `main`, never fails the build regardless of findings, and syncs
  repo-wide sweep findings (whole-repo golangci-lint today; `backstop gate --all` after ISSUE-091
  closes) into a rolling GitHub issue (or a small fixed set split by linter class) on
  `backstop-ai/backstop-core`.
- Re-running the job against an unchanged violation set does not create a duplicate issue or
  duplicate checklist entries — governed by an idempotent signature per finding.
- A finding whose file comes into compliance (confirmed clean on a subsequent sweep) is checked
  off / removed from the inventory with an explanatory note, without manual bookkeeping.
- The known baseline (23 errcheck/unparam findings in `pkg/config`, `pkg/check`) appears in the
  first live run of this job as the inventory's seed content.

## Note (2026-08-19)

Reviewed by the founder as part of a full backlog-pm investigation sweep and **parked** — real
and unbuilt, but nothing is currently blocked by its absence, CI is green, and no existing
directive charter fits it cleanly. DIR-003, DIR-019, DIR-024, and DIR-025 were all checked and
ruled out as a home — two of them (DIR-019, DIR-025) are already disclaimed by this issue's own
text above. Status stays `open`; this is not a scope or priority change, just a record that the
park decision was made deliberately rather than by neglect.

A concrete precedent worth citing for whoever eventually plans this: `slotly`
(`/Users/bmanson/src/projects/slotly/.github/workflows/enforce-architecture.yml`, a sibling
project on this machine — backstop's own origin lineage) runs a working, proven version of this
idea, but via a **different mechanism** than the "one rolling issue with a per-file checklist"
shape proposed above. It auto-creates **one GitHub issue per violation**, tagged with an
idempotent HTML-comment signature (`<!-- arch-violation: $filepath -->`) so re-runs don't
duplicate, and auto-closes that issue when a later PR fixes the file. Supporting scripts:
`.github/scripts/check-file-size.sh`, `.github/scripts/check-function-complexity.sh`,
`.github/scripts/post-complexity-comment.js`. Worth weighing per-violation issues with
auto-close against the single-rolling-checklist shape when this is eventually planned — it may
be a better fit for backstop's own gate findings.
