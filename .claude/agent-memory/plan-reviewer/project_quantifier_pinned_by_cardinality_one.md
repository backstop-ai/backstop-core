---
name: quantifier-pinned-by-cardinality-one
description: "A plan claiming an ANY/ALL quantifier is 'already pinned by corpus pack X' — count the quantified dimension per unit; cardinality 1 makes ANY and ALL identical and the pin vacuous
metadata:
  type: project
---

When a plan mandates a quantifier (ANY vs ALL, EVERY vs SOME) and says one half is
"already pinned by existing fixture X", COUNT THE QUANTIFIED DIMENSION inside X. If
the set being quantified over has cardinality 1, ANY and ALL are the same predicate
and the pin is vacuous — a wrong implementation passes the whole corpus.

**Why:** PLAN-ISSUE-151 round 3 (2026-08-17). Round 2 flagged "the quantifier is
undetermined by the corpus". The fix added a pack pinning the per-fixture-SET half
(EVERY) and asserted the per-fixture half (ANY) was "already pinned by
`fixture-masked`, whose two fixtures are matched by one slash-free pattern each,
never both by the same one." The task that OWNED that pack specified exactly ONE
slash-free pattern matching BOTH fixtures. Tabulating every pack: max slash-free
patterns per semgrep rule = 1, so ANY == ALL corpus-wide. An ALL implementation
passed everything and then produced the wrong answer on the real flagship rule
(2 hooks, each matching exactly one of 2 fixtures, disjointly).

**How to apply:** For every quantifier a plan mandates, build the table yourself —
one row per corpus unit, column = size of the quantified set. Any row with size 0 or
1 contributes nothing. Then check the REAL-WORLD case the plan cites as motivation
and confirm the corpus reproduces its cardinality, not just its shape. Two
independent tells that this is happening: (a) the plan asserts a pin in a task that
does NOT own the fixture files, contradicting the task that does; (b) the cited
"mirrors the real rule" pack differs from the real rule precisely in the
load-bearing dimension. Related: [[datadriven_red_uniform_quantifier]],
[[inert_decoy_fixtures_vacuous]].
