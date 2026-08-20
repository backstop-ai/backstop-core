---
name: source-issue-fences-out-delivered-claim
description: A plan that widens scope past its source issue often leaves the issue's own explicit "deliberately left out / that's ISSUE-NNN's territory" section un-retracted — grep the source for a scope-fence section, not just for factual errors
metadata:
  type: project
---

When a plan DELIVERS work its own source issue explicitly fenced OUT, check whether any task
routes a retraction of that fence. Grep the source issue for section headings like
"Structural facet", "deliberately left out of this issue's fix", "is ISSUE-NNN's territory,
not this issue's", "would scope-creep this issue".

**Why:** PLAN-ISSUE-180 (2026-08-19, round 2) reversed a judgment call from "narrow" to
"generalize the roster predicate" and added CLM-003 for it. Its TASK-005 routed three factual
corrections into ISSUE-180 and ISSUE-164 — but ISSUE-180 §"Structural facet" (lines 126-144)
says in so many words that generalizing the roster is ISSUE-164's territory and folding it in
here "would scope-creep this issue past its own confirmed evidence", and §"Relationship"
line 122 recommends NARROWING ISSUE-164 while the plan now recommends CLOSING it. On close with
`delivered_by: PLAN-ISSUE-180` the issue would ship a body disclaiming a claim the plan
delivered.

**How to apply:** for any scope-widening plan, enumerate every site in the source artifact that
constrains scope or attributes the work elsewhere, and require the routing task to name those
sites individually. Routing "correct the X section" is under-inclusive when the same premise
appears in Problem, Known, References and an Existence-in-world check.
Related: [[retracted-claim-survives-in-shipping-text]], [[completeness-claimed-comment-set]],
[[import-premise-grep-matches-forbidden-lists]].
