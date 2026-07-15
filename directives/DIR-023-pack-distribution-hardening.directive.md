---
title: "Pack Distribution Hardening"
number: DIR-023
created: "2026-07-15"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-055"
    - "ISSUE-058"
---

## Description

Harden pack distribution correctness on the road to the registry era. Two
threads at very different time horizons:

1. **Local provenance cache for local packs (ISSUE-055) — near-term, live
   bug.** Local-source packs (`pack add ../backstop-packs/ts-toolchain`, the
   founder's actual day-to-day workflow for the TypeScript pack suite) don't
   durably record where they came from. `LockEntry.LocalPath` is written
   straight into the **committed** `backstop.lock`, but this repo's own lock
   file already exhibits the failure today: five `source_type: local`
   entries (`backstop/contracts`, `backstop/go-standards`,
   `backstop/go-toolchain`, `backstop/self`, `backstop/substantiveness`),
   zero carrying a `local_path` — meaning `pack install` from a clean clone
   of this exact repo would fail loud on all five right now. BUNDLE-003
   OQ-4 already resolved the direction: the committed lock holds only
   portable git-ref packs; local-path provenance moves to a **gitignored**
   local cache (e.g. `.backstop/pack-sources.local.json`) written at `pack
   add` time, restoring local packs on the same machine only; promoting a
   local pack to shareable stays an explicit git-ref republish, never
   automatic. Real fresh-clone/reinstall e2e is the bar — a stubbed cache
   is not sufficient proof (`project_pack_provisioning_integration_gap`).
   Includes reconciling this repo's own five local-pack entries as part of
   the fix, not as a follow-up.
2. **Registry-era pack-declared detection (ISSUE-058) — explicitly
   deferred, future work.** BUNDLE-003 (OQ-5, resolved/dissolved) settled
   that day-zero `backstop init` does zero language detection — languages
   enter only via explicit `pack add`. A future auto-detect-and-offer
   capability hits a genuine catch-22: detecting a language without baking
   `go.mod → Go` etc. into core requires packs to declare their own
   activation signals, but reading those signals requires the packs to
   already be installed — which is what detection was supposed to
   determine. The only clean break is a pack registry/index consultable
   **before** any pack installs; neither a pack-manifest `detect:` field
   nor any registry/catalog exists today. This issue is explicitly gated on
   that registry infrastructure existing — it is not to be picked up ahead
   of that dependency landing, and even then must hold the thin-executor
   invariant (language-to-pack mapping stays pack-declared data, never a
   baked lookup/switch in core).

## Notes

ISSUE-055 is the near-term, live half of this directive (a bug in this
repo's own lock file today); ISSUE-058 is explicitly future/deferred per
its own issue file ("DEFERRED — registry-era, not near-term. Do not pick
this up ahead of that dependency landing"). Grouped under one directive
because both are pack-distribution-correctness themes on the same road
(from ad-hoc git-ref/local-path adds today toward a registry-backed future),
not because they're equally ready to pick up — the plan/spec work for
ISSUE-055 can proceed independently and well ahead of ISSUE-058.
