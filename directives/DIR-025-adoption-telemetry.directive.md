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

**Decoupled from the release timeline (ruling 2026-07-27).** This directive
does not need to land with or before DIR-001 (Release Workflow). The founder's
ruling: "we don't need fancy telemetry from day 1. We'll get downloads/clones
from GitHub and we'll also bake in homebrew and go install from day 1." That
splits the two halves cleanly:

- **Half 2 (ecosystem/adoption metrics)** is largely reconstructable without
  this directive shipping. GitHub already surfaces downloads and clone/traffic
  analytics, and Homebrew publishes install analytics for taps/formulae —
  with Homebrew and `go install` distribution baked in from day one, the
  publicly-tellable adoption numbers have a source at launch regardless of
  whether bespoke instrumentation exists yet.
- **Half 1 (usage telemetry)** does lose its day-zero baseline permanently if
  collection doesn't start on day one — there is no way to retroactively
  backfill "how many repos adopted on day one." That insight is still true.
  But at day-zero launch scale the cohort is small enough that the lost
  baseline costs little, while instrumenting it early buys noise rather than
  signal. The founder has consciously accepted that cost rather than treating
  it as a scheduling constraint on this directive.

Decoupling also reinforces the trust-first constraint above: designing
opt-out-or-better, openly-disclosed telemetry properly is easier unhurried,
on BUNDLE-016's own timeline, than rushed to hit a release date. Rushing
trust-sensitive collection to meet a deadline is the failure mode this
directive is staked out to avoid.

BUNDLE-016 is still at `exploring` maturity as of this writing, with its
Overview/Components sections not yet populated — this directive is sourced
against the bundle's existence and the founder's stated intent, not against
a promoted/defined bundle. Per standard bundle workflow, no spec work begins
here until the founder drives BUNDLE-016 past `exploring` with real open
questions resolved (`feedback_bundle_workflow`, `feedback_oq_workflow`).

## Notes

**2026-07-27 — hard-coupling-to-DIR-001 claim retracted.** An earlier version
of this directive asserted it must land with or before DIR-001 and was
positioned in BACKLOG.yml on that basis. The backlog PM flagged that both the
Description's coupling claim and this section's position claim had gone
stale relative to BACKLOG.yml's actual order; the founder ruled to retire the
coupling rather than reorder. See the Description's "Decoupled from the
release timeline" paragraph for the founder's reasoning and its date. This
directive's position in BACKLOG.yml is not restated here — position is
recorded in BACKLOG.yml itself, not duplicated in directive prose.
