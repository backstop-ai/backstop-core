---
name: datadriven-red-uniform-quantifier
description: A data-driven mandated test's "PRE-FIX STATE: ALL/BOTH X fail" is a per-subtest claim; evaluate the plan's own literal predicate against each measured row — the "right id for the wrong reason" row is usually already green
metadata:
  type: project
---

When a plan mandates a DATA-DRIVEN test (walk the manifest, one subtest per
fixture/row) and states a uniform pre-fix RED — "both negatives fail", "all N
positives fire", "every row is red" — evaluate the plan's OWN literal predicate
against each measured row separately. The uniform quantifier is frequently false
for exactly one row, and that row is the one the planner already flagged as
passing "for the wrong reason".

**Why:** PLAN-ISSUE-148 (2026-08-17) stated "both negatives fail the rule-id
equality" under a predicate of "at least one result whose ruleId EQUALS the
enclosing rule's own id". Measured against the combined sgconfig:
`hollow-test-go/negative.go` fired only `referenced-symbol-go` (fails — sibling
rule), but `referenced-symbol-go/negative.go` fired `referenced-symbol-go` —
**its own id** — so that subtest was GREEN pre-fix. The plan's own parenthetical
conceded it ("the right id for the wrong reason") while the leading sentence and
the claim text (CLM-002: "the way both of them do today") asserted the opposite.
The remedy offered was "assert the SPECIFIC expectation strongly enough that the
corrected body is what satisfies it" — a wish, not a predicate. An implementer
had nothing defined to write and would hit a green subtest where the task text
promised red.

The tell: a planner writes the correct nuance in a parenthetical or a table row,
then over-generalizes it into the summary sentence and the claim text. The
parenthetical is the measurement; the summary is the narrative. Trust the table.

**How to apply:** For any plan whose test task enumerates rows from a manifest or
registry, build the row×predicate matrix yourself from real tool output and mark
each cell red/green. Do NOT accept "N/N red" — count them. If a row is green
pre-fix, the plan needs either (a) a strictly stronger predicate that makes it
red, or (b) a corrected pre-fix statement that admits the row is a
non-discriminating guard. Option (a) is almost always available and is what makes
the claim fully discriminating: in the 148 case, requiring the negative's
own-rule finding to carry a symbol other than the test receiver `t` turned
3-red/1-green into 4-red (pre-fix refsym/negative yields only `symbol=t`;
post-fix it yields `strings` and `t`).

Check the CLAIM TEXT too, not just the task description — the same false
quantifier propagates into `claims:` prose and strands the claim even after the
task is corrected. See [[project_repurposed_test_claim_text_drift]].

Related: [[project_inert_engine_premise_wrong_reason]] (right conclusion, wrong
reason — the mirror image of this), [[project_shortcircuit_dependent_guard]]
(a guard green at HEAD with the defect live), [[project_packval_real_execution_premises]].
