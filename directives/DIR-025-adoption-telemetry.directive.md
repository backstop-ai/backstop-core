---
title: "Adoption Telemetry"
number: DIR-025
created: "2026-07-16"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-016"
---

## Description

Build anonymized CLI usage telemetry and ecosystem/adoption metrics, live
from **day one of first release** — the founder is staking this claim now,
despite no spare bandwidth, because the numbers only start compounding once
collection starts. Two halves:

1. **Usage telemetry.** Anonymized, transparent signal on how `backstop` is
   actually run in the wild — the substrate for any future "how many repos
   run backstop" claim. **Trust-first is the hard constraint, not a nice-to-have:**
   backstop's brand is trust (verification, no vacuous green, no baked
   surveillance) — telemetry that isn't obviously anonymized and openly
   documented would directly undercut the thing the product sells. Whatever
   ships must be opt-out-or-better, disclosed in the open, and defensible
   under the same scrutiny backstop asks of everyone else's tooling.
2. **Ecosystem/adoption metrics.** The aggregate, publicly-tellable numbers —
   repos using backstop, packs that exist, packs installed — the
   "podcast-braggable" figures that make the OSS-flag-planting /
   agency-capture business model (`project_business_model_agency`)
   credible with a number instead of an assertion.

**Hard coupling to the release timeline.** This directive must land WITH OR
BEFORE DIR-001 (Release Workflow) — that is the entire reason it is being
staked out now rather than deferred. Telemetry collection that starts after
day one of the first public release loses the day-zero baseline permanently;
there is no way to retroactively backfill "how many repos adopted on day
one." This coupling, not a bandwidth argument, is why it sits immediately
after DIR-001 in the backlog rather than further down with the other
hardening directives.

BUNDLE-016 is still at `exploring` maturity as of this writing, with its
Overview/Components sections not yet populated — this directive is sourced
against the bundle's existence and the founder's stated intent, not against
a promoted/defined bundle. Per standard bundle workflow, no spec work begins
here until the founder drives BUNDLE-016 past `exploring` with real open
questions resolved (`feedback_bundle_workflow`, `feedback_oq_workflow`).

## Notes

Positioned directly after DIR-001 (Release Workflow) in BACKLOG.yml,
inheriting the release timeline rather than being prioritized independently
— it is a release-day dependency, not a general hardening item. The founder
reorders at will; this placement should not be read as this directive
outranking release-blocking work on its own merits, only as riding along
with DIR-001's schedule.
