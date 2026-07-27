---
title: "Baseline Implementation"
number: DIR-003
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-007"
    - "SPEC-010"
    - "ISSUE-056"
    - "ISSUE-086"
---

## Description

Implement gate step 7 (baseline comparison) and the CI baseline generation workflow. CI runs `backstop gate` post-merge and publishes the violation set as an immutable baseline artifact. Locally, `backstop gate` auto-pulls the latest baseline with TTL-based caching and reports differentials ("3 new violations beyond baseline") instead of absolute counts.

Includes: `backstop baseline pull` command, `.backstop/baseline.json` caching, TTL logic (default 15 minutes), GitHub Actions artifact publishing, structural diff algorithm for violation identity. Per ISSUE-086, the CI `baseline` job must install packs before generating, so the published artifact reflects pack-engine findings rather than a structurally empty engine set.

Depends on DIR-001 (release workflow — CI must exist to generate baselines).

## Notes

- **ISSUE-086 gates this directive's completion, independent of backlog position.** DIR-003's
  pull model presumes CI generates a baseline `backstop gate` can trust and pull from. The
  currently published `backstop-baseline-v1` artifact is generated with zero packs installed
  (`.github/workflows/ci.yml:39-64` runs `./backstop baseline generate` with no `pack install`
  step), so every pack-engine-sourced dimension is structurally absent from it — not clean,
  never evaluated. Any DIR-003 work that consumes today's artifact inherits that vacuum.
  Reordering this directive in BACKLOG.yml does not change this fact. ISSUE-086 must be fixed
  before or as part of DIR-003's delivery, not after.
- **The approved coverage-baseline refresh is HELD, founder-ratified 2026-07-27, until the CI
  `baseline` job installs packs.** Refreshing the tracked coverage baseline against today's
  packless artifact would ratchet-declare every pack-engine dimension clean without ever having
  evaluated it — the silent/vacuous green this project exists to prevent (see CLAUDE.md
  "Enforcement philosophy"). This is a hold, not a PM suggestion, and should not be lifted
  without an explicit founder go once ISSUE-086 lands.
