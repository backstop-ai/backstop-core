---
title: "Directive / Issue / Plan Lifecycle Hardening"
number: DIR-016
created: "2026-07-06"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-042"
    - "ISSUE-043"
    - "ISSUE-044"
---

## Description

Backstop has invested heavily in hardening the FEATURE track
(bundle → spec → plan → implementation) — maturity gates, OQ discipline,
promotion structure. The REACTIVE/ROADMAP track (directive → issue → plan)
has had much less: the artifact state machine is defined
(`open → ready → … → closed` for issues, `queued → active → … → done` for
directives) but the transitions are manual and entirely unenforced. Nothing
keeps `status` in sync with reality, and closing an issue is high-friction
enough that issues rot at `open` instead of being closed promptly. This
directive hardens that track so status stays honest and transitions are
cheap, bringing it toward parity with the feature track.

Three concrete symptoms surfaced in the 2026-07-06 session that closed out
DIR-015/ISSUE-018 et al.:

- **Status↔reality drift is unguarded in one direction and unguarded worse in
  the other.** Delivered-but-still-`open` drift had to be cleaned up by hand
  on four issues (ISSUE-018, ISSUE-034, ISSUE-035, ISSUE-036) at that
  session's close — nothing in the gate or hooks caught it automatically.
  The inverse is more dangerous and entirely unguarded: an issue or
  directive can be marked `closed`/`done` while its mandated tests are
  absent or failing — a vacuous "done" with no check to catch it.
- **Closing a plan-backed issue re-authors work that already exists.**
  Closing an issue currently requires re-deriving its requirements/claims
  onto the issue by hand, even though issue → plan → implement work already
  holds those claims in the PLAN. That re-authoring tax is exactly what
  causes issues to sit at `open` well past the point their work actually
  landed.
- **Config self-consistency is unenforced.** Adding a new sub-agent requires
  a matching case in the agent-guard hook or the agent silently can't write
  artifacts at all — this was hit live while authoring this very directive
  (`directive-author` had no agent-guard case). Nothing detects the config
  gap; it's discovered only when an agent mysteriously fails to write.

## Acceptance Criteria

- The gate makes status drift LOUD, using the existing broken-promise model
  (loud ≠ automatically blocking — see backstop's own enforcement
  philosophy): delivered-but-still-`open`/`queued`/`active` drift WARNs;
  `closed`/`done` while the artifact's mandated tests are absent or failing
  BLOCKS, since that is a broken promise, not just un-adopted capability.
- Closing a plan-backed issue is a low-friction status+date flip — claims
  trace back to the plan that implemented them rather than being
  re-authored onto the issue.
- A config self-check detects gaps like a missing agent-guard case for a
  registered sub-agent, so a new agent role fails loudly at config-check
  time rather than silently at write time.

## Notes

This is the directive/issue/plan-track analogue of the maturity work the
bundle/spec/plan track already has (structural promotion gates, OQ
discipline) — the goal is parity, not novelty. The pattern is expected to
extend to further track-hardening work beyond this directive's initial
scope. Grounded in the 2026-07-06 session that surfaced all three symptoms
in the course of closing out DIR-015 and the thin-executor eradication
backlog (ISSUE-018 et al.).
