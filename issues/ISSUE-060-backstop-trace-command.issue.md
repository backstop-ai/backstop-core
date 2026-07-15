---
title: "Backstop Trace Command"
schema_version: issue/v1

issue:
  id: ISSUE-060
  title: "Backstop Trace Command"
  type: enhancement
  status: open
  created: "2026-07-15"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: safe

requirements:
  - id: REQ-001
    text: >
      `backstop trace [bundle[:REQ-NNN[@version]]] --json` must exist as a
      read-only command emitting the requirement's full proof attribution
      graph: the bundle REQ and its version log, the supporting spec
      requirement(s) at their pinned version(s) with status, the claims and
      mandated tests backing each, and any lineage issues tracing to the REQ
      (present for context, never counted as coverage). Vocabulary must say
      "covered", never "delivered", for REQ status — only an `implemented`
      spec requirement closes coverage.
    supports: "requirement-traceability:REQ-011@1.0.0"
---

# ISSUE-060: `backstop trace` — the requirement-interrogation command

## Problem

With requirement traceability shipped (BUNDLE-014, delivered 2026-07-15), the
gate answers "what's covered / what's not" — but by design it only emits the
**negative** space: coverage gaps, unresolved refs, broken chains
(`requirement_traceability` step, see BUNDLE-014 REQ-006…REQ-012). The
**positive** proof graph is fully derivable from the corpus but has no
queryable surface:

- which `implemented` spec proves a given bundle REQ, and at which pinned
  version (REQ-009's coverage resolution);
- which claims and mandated tests back that proof;
- which issues carry lineage against the REQ (REQ-011 — resolution-checked,
  queryable, but explicitly NOT coverage);
- the REQ's own version log (REQ-004) — what changed, across which versions.

Today, answering "prove REQ-011 is satisfied, and show me everything that
touches it" requires manually grepping bundles/specs/issues and reconstructing
the chain by hand. There is no command that walks the graph and emits it.

This has been explicitly agreed with the portal (gate-payload) consumer:
proof attribution stays OUT of gate violation payloads — different shape,
different freshness cadence than a violation feed — and belongs in a
dedicated, read-only command instead.

## Solution

Not committed — this is a FUTURE / backlog seed, not near-term work. Direction:

Add `backstop trace [bundle[:REQ-NNN[@version]]] --json` as a new, read-only
CLI command emitting the requirement's full attribution chain as a JSON
graph:

- the bundle REQ (text, current version, and full version log);
- the supporting spec requirement(s) at their pinned version(s), with each
  supporting spec's status (only `implemented` closes coverage — the command
  must use "covered" vocabulary for REQs, never "delivered", per BUNDLE-014's
  status vocabulary);
- claims and mandated tests backing each supporting spec requirement;
- lineage issues tracing to the REQ (present for context, never counted as
  coverage — REQ-011).

Positioned as an ENTERPRISE-tier capability: primary consumers are the bclabs
portal's requirement drill-down view, forensic replay, and audits — not the
day-to-day gate loop.

Explicitly out of scope for this issue:

- the exact output JSON schema (design at spec/plan time, once picked up);
- any write behavior — `trace` is read-only, always, no exceptions;
- wiring into the gate payload or violation feed (the portal boundary this
  issue exists to respect).

## References

- BUNDLE-014 (`requirement-traceability`, delivered 2026-07-15) — the
  traceability graph this command reads: REQ-001…REQ-003 (resolution/pin/
  log-match, `artifact validate`), REQ-004 (per-REQ version log), REQ-006…
  REQ-010 (`requirement_traceability` gate step, coverage semantics), REQ-011
  (issue lineage, resolution-checked but non-coverage — the requirement this
  issue directly extends into a queryable surface)
- Portal consumer agreement: proof attribution graph is out of gate violation
  payloads (different shape/freshness); lives in a dedicated command instead
