---
name: dir033-and-the-three-way-gate-boundary
description: DIR-033 "Gate Verdict Honesty Residual Tail" exists (queued, 9 sources, ABSENT from BACKLOG.yml) — homing a gate/engine defect is now a THREE-way call between DIR-024 / DIR-032(done) / DIR-033, decided by filing provenance, not topic
metadata:
  type: project
---

`DIR-033 "Gate Verdict Honesty Residual Tail"` was authored 2026-08-17 (`status: queued`,
9 sources: ISSUE-149/150/152/154/155/156/157/159/161). It is **absent from BACKLOG.yml's
`directives:` list** — placement is a founder call, deliberately deferred by the directive's
own Notes. Do not read its absence as an error to fix; do not slot it yourself.

**Why:** `DIR-032 "Gate Verdict Honesty"` reached `done` 2026-08-17 while its member plans
were still filing mandated follow-ons. Founder ruled a *successor* directive rather than
reopening DIR-032 (protects its earned `done`) or folding into DIR-024 (dilutes the catch-all).
DIR-033's own Description generalizes the mechanism: a directive's roster is a CLOSED set fixed
at authoring; plan follow-on filings are an OPEN set generated during delivery. Expect a
tail-sweep after every directive closure.

**How to apply — the three-way test, in this order:**
1. **DIR-033** if the issue was *filed by a DIR-032 member plan* (ISSUE-091/093/136/142/146/148
   lanes). Provenance, not topic — check the issue's own "filed as PLAN-X TASK-N" line.
2. **DIR-032** never (it is `done`; citations stay as provenance, per the done-directive
   RELEASE rule — see [[project_homed_but_orphaned_bundles]]).
3. **DIR-024 "Gate/Engine Quality"** otherwise. Its charter, drawn explicitly SEVEN times now
   (items 15/ISSUE-135, 16/145, 17/147, 18/158, 20/163): **loud red with a wrong or missing
   legible name**. DIR-032's charter sentence — "computes a result internally but reports the
   wrong verdict about it" — is the discriminator. If it goes RED, it is DIR-024.

**Two DIR-024 sub-precedents worth reusing verbatim:**
- **Delivery residuals home to their parent item's directive**, as clear fits, not roster calls.
  Item 19 (ISSUE-131) is item 3's tail; item 20 (ISSUE-163) is item 1's. The directive's own
  text blesses this shape: "a clear fit, not a founder roster call."
- **The v0.2.0 Linux-CI investigation family all lands here**: ISSUE-158 (item 18),
  ISSUE-162 (→ folded to ISSUE-147/item 17), ISSUE-163 (item 20).

**Standing sequencing debt:** DIR-024 is a 20-item catch-all at BACKLOG.yml position 5 with
TWO items each asking to be the next lane (ISSUE-147, ISSUE-163). Never propose promoting the
directive to carry one item — ask for a sequencing ack instead. See
[[project_pack_rule_precision_family]].
