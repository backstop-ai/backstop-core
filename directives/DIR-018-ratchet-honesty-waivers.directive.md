---
title: "Ratchet Honesty Waivers"
number: DIR-018
created: "2026-07-09"
schema_version: directive/v1

directive:
  status: done
  source:
    - "BUNDLE-013"
    - "ISSUE-050"
  spec: "SPEC-049"
  completed: "2026-07-10"
---

## Description

Close the silent-grandfathering gap in the gate's baseline system. Today the
baseline can hide pre-existing debt indefinitely with no justification and no
expiry (it is gitignored, CI-regenerated, and by design not a durable record
of a human decision), and the ratchet only prevents NEW debt from landing on
touched files — it never forces paydown of the debt already sitting in a file
an author is actively editing. This directive makes "chosen not to fix"
always explicit and always accountable, via two complementary pieces of
work:

1. **BUNDLE-013 (waiver subsystem)** — give backstop a native, engine-neutral,
   tracked, justified, gate-visible suppression primitive. This fills gate
   step 8 ("waiver resolution"), which SPEC-010 REQ-006 reserved and left
   stubbed (`pkg/gate/step_deferred.go` currently returns "skipped / waivers
   not implemented"), and the seam SPEC-019 deliberately preserved for
   waiver-aware baseline comparison. Today the only escape valves are
   engine-native suppressions (`// nosemgrep`, `//nolint`, `//lint:ignore`,
   `#gitleaks:allow` — invisible to backstop's own accounting) and silent
   baseline grandfathering. A waiver is the accountable third option: a
   first-class ledger of deliberate deferrals, each with a justification,
   that stays loud in gate output while still letting the gate go green.
2. **ISSUE-050 (strict file-level ratchet)** — the other half of the ratchet.
   Touching a file revokes baseline grandfathering for ALL of that file's
   pre-existing findings, not just new ones added to it, forcing each one to
   be fixed or explicitly waived. This drives paydown of existing debt at
   the exact point an author already has full context: the file they are
   editing right now.

**Sequencing (decided, not open):** waivers land first, or at minimum
progress to a spec+plan first, because they are the strict ratchet's
accountable escape valve. Shipping ISSUE-050's strict revocation without an
accountable "acknowledge and defer" mechanism already in place would leave
only the invisible engine-native suppressions as relief — exactly what this
theme exists to move away from. ISSUE-050 implements after BUNDLE-013 has
progressed to at least a spec+plan.

**Relationship to other directives (do not conflate):**

- DIR-003 (Baseline Implementation) builds the baseline MECHANISM (gate step
  7, CI generation, TTL pull). DIR-018 is the ACCOUNTABILITY layer on top of
  that mechanism — gate step 8 (waivers) plus the honest bidirectional
  ratchet. Complementary, not overlapping; DIR-018 does not redo DIR-003's
  work.
- DIR-015 (Gate Checker Hardening) fixes what the gate actually checks today
  — scope bugs, the `applies-to` rename (ISSUE-041), contract drift.
  ISSUE-050 extends the applies-to/new-code grandfathering semantics that
  ISSUE-041 established, so DIR-018 is downstream of that work, but it is its
  own coherent theme (the ratchet's second half plus the waiver primitive) —
  not more of DIR-015's correctness cleanup.

## Notes

BUNDLE-013 is at `exploring` maturity with 8 open questions (granularity,
justification model, lifecycle/expiry, storage/format, baseline/ratchet
relationship, migration off engine-native suppressions, CLI ergonomics, gate
reporting) — all deliberately left for founder-driven resolution before any
design decisions or spec seeds are committed. ISSUE-050 is a decided design
(founder verbatim, 2026-07-09) but is explicitly blocked on BUNDLE-013
landing an accountable escape valve before it implements.

**Closed out 2026-07-10.** Both halves delivered in sequence as planned:

- BUNDLE-013 resolved all 8 OQs and promoted exploring → defined
  (`cbe83df`), then implemented via SPEC-049 (`eee4700`); bundle status is
  `delivered`.
- ISSUE-050 implemented after BUNDLE-013's escape valve landed
  (`ba3169e`), closed via `delivered_by: PLAN-ISSUE-050` (`bd50aa8`).

The ratchet-honesty theme (accountable waivers + strict file-level
grandfathering revocation, on by default) is fully realized. No follow-on
directive opened — any future waiver/ratchet work is new scope, not a
reopening of this theme.
