---
name: debake-literal-sweeps-all-specs
description: A plan that de-bakes a literal must grep ALL of specs/ for that literal, not just the spec it thinks owns it — sibling specs enumerate the same list in requirement prose
metadata:
  type: project
---

When a plan removes a baked literal (`vendor`, `node_modules`, a language extension,
a tool name), sweep `grep -rn '<literal>' specs/` across the WHOLE corpus before
accepting the plan's list of drifting specs. Planners reliably grep only the spec they
believe owns the symbol.

**Why:** PLAN-ISSUE-122 round 3 stated with confidence "TWO implemented specs drift on
this change, not one" (SPEC-068 + SPEC-043) after grepping SPEC-068 alone. A corpus-wide
grep found SPEC-070 REQ-007 (`:207-213`) enumerating the same five names as a property of
`FindUngatedArtifacts` — live requirement prose in an `implemented` spec, on a file
(`doctor_checks.go`) the plan edits — plus SPEC-069 Sharp Edge 5. Worse than a silent
omission: the plan carried an affirmative "SPEC-070 … DELIBERATELY NOT IN SCOPE, citations
verified" block that had only checked the CONTRACT-SIGNATURE question, so an implementer
would never look again.

**How to apply:** Two orthogonal sweeps per de-baking plan, and the plan must show both:
1. `grep -rn '<literal>' specs/` — prose/claim/requirement enumerations, corpus-wide.
2. `grep -rn '<symbol>' specs/` — contract-signature literals (see
   [[contract-signature-block-drift]]).
Neither grep finds the other's hits. A plan that ran only one has an unexamined half.
An affirmative "X is out of scope, verified" clause is a red flag, not reassurance —
check WHICH question it verified.
