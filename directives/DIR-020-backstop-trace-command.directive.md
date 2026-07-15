---
title: "Backstop Trace Command"
number: DIR-020
created: "2026-07-15"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-060"
---

## Description

Build `backstop trace [bundle[:REQ-NNN[@version]]] --json` — a new, read-only
CLI command that emits the **positive** proof-attribution graph for a
requirement: the bundle REQ and its version log, the supporting spec
requirement(s) at their pinned version(s) with status, the claims and
mandated tests backing each, and any lineage issues tracing to the REQ
(present for context, never counted as coverage — only an `implemented` spec
requirement closes coverage). Vocabulary must say "covered", never
"delivered", for REQ status, per BUNDLE-014's status vocabulary.

This is the deliberate positive counterpart to the negative-space coverage
gaps the `requirement_traceability` gate step already emits (BUNDLE-014,
delivered 2026-07-15). The two are kept structurally separate on purpose:
gate violation payloads answer "what's broken," `trace` answers "prove this
is satisfied and show me everything that touches it." That split was agreed
with the bclabs portal (gate-payload) consumer — gap facts and proof facts
have different shapes and different freshness cadences, so proof attribution
does not get folded into the gate payload.

Positioned as an **enterprise-tier** capability. Primary consumers are the
bclabs portal's requirement drill-down view, forensic replay, and audits —
not the day-to-day gate loop. This is future backlog, not near-term work: no
spec exists yet, and the output JSON schema is explicitly left for spec/plan
time per ISSUE-060.

## Notes

Placed at the bottom of BACKLOG.yml, below all current active/queued work.
This is deliberately low priority — an enterprise-tier, forensic/audit
surface with an explicit "not committed, future backlog seed" framing in its
source issue (ISSUE-060). Nothing above it in the backlog should be
displaced by this addition; the founder should reposition if priority
changes.

## References

- ISSUE-060 (`backstop-trace-command`, open, enhancement) — the source
  issue; REQ-001 is the sole requirement, scoped exploratory/cross-cutting/
  safe, supporting BUNDLE-014 REQ-011 (issue lineage tracing).
- BUNDLE-014 (`requirement-traceability`, delivered 2026-07-15) — the
  traceability graph this command reads: REQ-001…REQ-003 (resolution/pin/
  log-match), REQ-004 (per-REQ version log), REQ-006…REQ-010
  (`requirement_traceability` gate step, coverage semantics), REQ-011 (issue
  lineage, resolution-checked but non-coverage).
