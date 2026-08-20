---
title: "The Published Baseline Artifact Is Generated With Zero Packs Installed"
schema_version: issue/v1

issue:
  id: ISSUE-086
  title: "The Published Baseline Artifact Is Generated With Zero Packs Installed"
  type: technical-debt
  status: open
  created: "2026-07-27"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# The Published Baseline Artifact Is Generated With Zero Packs Installed

## Problem

This is a **known, founder-acknowledged launch checklist item**, not a fresh discovery. Founder,
2026-07-27, on being shown this: "yes i am aware of that. that'll be a launch checklist item."
This issue is the durable tracker for that checklist item — filed so the gap has an ID,
dependencies, and a resolution path instead of living only in a chat transcript.

`.github/workflows/ci.yml:39-64` (the `baseline` job, gated to `push` on `main`, `needs: gate`)
builds the CLI (`ci.yml:51-52`) and runs `./backstop baseline generate` (`ci.yml:54-58`)
directly — there is **no `pack install` (or equivalent) step anywhere in this job**.
`.backstop/packs/` is gitignored (`.gitignore:41`, consistent with the pack-distribution model:
packs install like `node_modules`, `backstop.lock` is the durable record — see CLAUDE.md
"Packs"). With no install step, the directory backstop's pack-engine dispatch reads from is
simply absent on the baseline runner.

The result: `backstop-baseline-v1` (published via `actions/upload-artifact@v4` at `ci.yml:60-64`,
path `.backstop/baseline.json`) is generated with **zero pack engines present**. Every
pack-engine-sourced finding (semgrep/ast-grep dims, anything routed through
`dispatchPackEngines`) is structurally absent from the published reference baseline — not because
the codebase is clean on those dims, but because nothing ran them.

This bears directly on **DIR-003**'s pull model: DIR-003 (Baseline Implementation) presumes CI
generates a baseline that gate can trust and pull from; a packless baseline is not that — it's a
partial baseline masquerading as the full one. It also bears on the pending
coverage-baseline-refresh sequencing noted in project memory
([[project_baseline_design]]/[[project_gate_dogfood_mostly_dark]]): refreshing the tracked
baseline against this packless artifact would ratchet the project against a vacuum on every
pack-engine dimension, silently declaring those dims "clean" when they were never evaluated.

## Split history

This issue was narrowed on 2026-07-27, per founder ruling, from a two-fact filing. The original
filing coupled this packless-baseline gap with a second fact — that core `ci.yml`'s `gate` job
hand-bakes the Go toolchain instead of calling `backstop gate`. That fact is a duplicate: it
already lives inside **ISSUE-020**'s acceptance criterion, and the founder's 2026-07-26 ruling on
ISSUE-020 was explicit — "do not re-file this as a standalone issue." The founder's clarification
on 2026-07-27: "wire CI to backstop gate, but no separate issue — it's welded into ISSUE-020's
acceptance." This issue now carries only the packless-baseline defect; the CI-wiring fact stays
welded to ISSUE-020 and should not be re-filed.

## Dependencies and path to resolution

The fix is narrow and independently actionable: add a `pack install` (or equivalent) step to the
`baseline` job (`ci.yml:39-64`) before `./backstop baseline generate` runs (`ci.yml:54-58`), so
the packs backstop's pack-engine dispatch depends on are actually present on the baseline runner.

This does **not** require waiting on **ISSUE-020** or **BUNDLE-021**. Those track the `gate`
job's Linux sandbox blocker for the hand-baked-Go → `backstop gate` wiring — a separate, harder
problem gated on BUNDLE-021's OQ-2/OQ-3. The `baseline` job doesn't need any of that: it only
needs `backstop baseline generate` to see installed packs, not to run gate's sandboxed
dimensions. A `pack install` step ahead of `baseline generate` is sufficient on its own.

There is overlap worth naming so it doesn't read as a contradiction: the CI-runs-`backstop gate`
work tracked in ISSUE-020 would fix this issue as a side effect once its jobs install packs — a
`gate` job that installs packs to run `backstop gate` for real also leaves packs present for
`baseline generate`, if the two jobs share a runner state or the `baseline` job is later folded
into the same install step. But this issue's own fix does not depend on that landing first, and
should not wait on it — it can and should ship on its own timeline.

## Cross-references

- **ISSUE-020** — owns the CI-wiring gap (the `gate` job hand-baking Go instead of calling
  `backstop gate`) as part of its own acceptance criterion, per founder ruling on 2026-07-26,
  reaffirmed 2026-07-27. That fact is NOT part of this issue and should not be re-filed here or
  elsewhere — this line exists so the split stays durable.
- **ISSUE-084** — published pack *repos* have no CI (`pack check`/`pack test` on push); a sibling
  gap at the pack-repo layer to this issue's gap at the core-repo layer. Different surface, same
  category of thesis violation (CI that doesn't exercise backstop's own verification).
- **DIR-003** (Baseline Implementation) — the owning directive; depends on the published baseline
  being a trustworthy, full-scope artifact, which this issue's defect breaks today.
- **DIR-001** (Release Workflow) — release-pipeline territory; a self-gating release depends on
  the published baseline being trustworthy, so this issue sits as a lighter-weight dependency of
  DIR-001's premise holding, alongside (but independent of) the CI-wiring gap ISSUE-020 owns.

## Verification

No code change accompanies this filing — it is the checklist tracker. Verification criteria for
the eventual fix (recorded here so `ready`-promotion inherits a target, not a blank page):

- The `baseline` job (`ci.yml:39-64`) installs packs before `./backstop baseline generate` runs,
  so `backstop-baseline-v1` reflects pack-engine findings rather than an empty engine set.
- A subsequent gate run against the refreshed baseline does not silently ratchet-clean any
  pack-engine dimension that was previously unevaluated — any newly-surfaced findings from
  now-present engines are visible, not swallowed by the baseline swap.

## Note (2026-08-19)

Status check, no change: criterion 1 above (the `baseline` job installs packs before `baseline
generate`) has landed, via ISSUE-020's commits — confirmed by direct read of `ci.yml`: `./backstop
pack install` runs at `ci.yml:312`, ahead of `./backstop baseline generate` at `ci.yml:321`.

Criterion 2 is **not yet demonstrated**: no baseline has ever been committed in this repo
(`.backstop/baseline.json` is untracked, zero git history for it), which is exactly what DIR-003's
founder-ratified HOLD (reaffirmed 2026-08-10) protects against. Criterion 1 landing does not prove
criterion 2 safe — a baseline refresh that silently ratchet-cleans a previously-unevaluated
pack-engine dimension remains unverified until a baseline is actually generated and pulled.

`status: open` remains accurate. No re-homing — DIR-003 already sources this issue.
