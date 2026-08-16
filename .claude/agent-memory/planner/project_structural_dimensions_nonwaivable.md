---
name: structural-dimensions-nonwaivable
description: contract_signature / artifact_status_drift / test_substantiveness are excluded from waivableDimension() by SPEC-049 REQ-010 — never plan an interim waiver for them
metadata:
  type: project
---

Structural gate dimensions — `contract_signature`, `artifact_status_drift`,
`test_substantiveness` — are DELIBERATELY excluded from `waivableDimension()` per SPEC-049
REQ-010's exhaustive waivable-surface matrix. A `@waiver:` token written against one is INERT:
it is never harvested, so it neither suppresses the red nor leaves an accountable record — it
just sits in the repo as a lie.

**Why:** discovered the expensive way during PLAN-ISSUE-129 (2026-08-16). A pre-existing dormant
`contract_signature` false-red on a YAML-target contract (ISSUE-053's class — the compiler is
Go-syntax-only) surfaced when a fixture edit pulled the file into diff scope. The implementer
reached for an interim waiver, found it structurally impossible, and correctly REVERTED the
token rather than leave a false record. ISSUE-053 had to be amended (ecf0f61) to correct its own
stale claim that a waiver was being applied.

**How to apply:**
- Never write a task that prescribes an interim waiver for a structural dimension. The
  accountable paths are retire / replace / resolved-by / obsoleted, or a real fix.
- When a plan's file scope will pull a fixture or artifact into diff scope, expect DORMANT reds
  from dimensions that were previously never evaluated on that file. Plan for surfacing +
  filing, not for suppressing — see [[feedback_state_a_sweep_once]] and the no-grandfather rule.
- A dormant red on a file you touch is real scope: the "touch it, fix ALL its violations" rule
  means the plan should either budget for the fix or explicitly accept-and-file it as a
  follow-on issue with the pre-existence PROVEN (an identical-shape violation on an untouched
  sibling file is the proof shape that worked).
