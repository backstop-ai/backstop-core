---
name: issue-self-retraction
description: A closed issue's Problem/Evidence section can be RETRACTED by its own later Root Cause or Resolution section while both remain in the file — cite only the live section, and check which one is live before quoting evidence
metadata:
  type: project
---

An issue file is not uniformly authoritative. A later section can retract an earlier one
**in place**, leaving both in the file with no marker on the dead text. Quoting the dead half
into a plan's claim ledger ships a retracted finding as a founder-signed fact.

**Why:** authoring PLAN-ISSUE-172 (2026-08-19) I cited `ISSUE-068:40-46` — "`pack_engines`
≈ 174s and `coverage_threshold` ≈ 174s — near-identical and **overlapping**… the gate pays ~2x
the single-run cost, **concurrently**. The concurrency ALSO forced the consumer to cap vitest
parallelism" — as empirical support for "the two dominant steps contend for CPU." The same
issue's Root Cause section at `:56-68` retracts exactly that: *"Corrected 2026-07-18 by
code-grounded investigation — the original 'serialize / concurrency' framing below was WRONG
and is replaced, not extended… there is no concurrent double-run and nothing to 'serialize'…
The portal's vitest fork-cap… is NOT evidence of a backstop concurrency bug."* The Problem
section was never rewritten, so the file contradicts itself and only the later section is live.
Plan-reviewer caught it before TASK-001 would have written it into ISSUE-172's permanent
Investigation section.

**How to apply:**
- Before quoting an issue as EVIDENCE, read its Root Cause / Resolution / Corrected sections
  FIRST, then decide which text is live. On a `closed` issue, the Resolution section is usually
  the most recent authorship and beats everything above it.
- Grep the file for `Corrected`, `WRONG`, `replaced, not extended`, `retract`, `superseded` —
  this repo's issues mark self-corrections in prose, not in frontmatter.
- Partial retraction is the common case, so split the citation rather than dropping it whole:
  in ISSUE-068 the *double-run cost* (~2x a single run) SURVIVES and is confirmed by the live
  Resolution, while the *overlap/concurrency/fork-cap* framing does not. I kept the former and
  dropped the latter.
- Prefer re-grounding a claim on your OWN measured finding over any inherited citation. CLM-009's
  conclusion stood on this plan's own double-run discovery with the ISSUE-068 reference removed
  entirely — the citation was never load-bearing, only decorative.
- When you find one, leave a **★ CITATION TRAP** note in the claim itself naming the retracted
  passage and why it looks quotable. The next planner will otherwise re-find the same seductive
  paragraph. Related: [[verify-issue-premises]], [[dir032-stale-premises]].
