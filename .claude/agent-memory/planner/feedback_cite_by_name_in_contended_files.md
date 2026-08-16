---
name: cite-by-name-in-contended-files
description: In files concurrently edited by sibling lanes (cmd/backstop/gate.go especially), cite edit sites by function/construct name, never by line number
metadata:
  type: feedback
---

When a plan's edit sites live in a file that concurrent sibling lanes are also
editing, cite them by FUNCTION or CONSTRUCT NAME and tell the implementer to
locate them by reading the file fresh. Never cite a line number.

**Why:** PLAN-ISSUE-122 tracked four sites in `cmd/backstop/gate.go` across seven
review rounds and got three different line triples — :1652/:1663/:1674/:1705,
then :1706/:1717/:1728/:1759, then :1718/:1729/:1740/:1771 — purely from
PLAN-ISSUE-113 and PLAN-ISSUE-118 churning the same file. Each round "fixed" the
numbers and each fix was stale before the next reviewer opened the file. A line
number in a contended file is a guaranteed future defect, and re-deriving it
burns a review round every time.

**How to apply:** `cmd/backstop/gate.go` is the standing example — it is huge, it
is where most gate wiring lands, and multiple lanes touch it at once. Before
citing any line there, run `git status`; if it is dirty, or if any sibling plan
scopes it, go name-based. Quote a line number only as a dated aid ("it sat at
:1401 in the 2026-08-16 tree"), never as the locator. Add one standing
"do not trust a line number in this file" note rather than repeating the caveat
at each site. Uncontended files (pkg/artifact, specs) can keep line numbers.

**Corollary — re-verify the surrounding claim, not just the number.** Chasing
gate.go's line drift surfaced a real defect no reviewer had caught: the plan
asserted `computeRequirementTraceabilitySurfaces`'s caller was "a closure inside
buildGateSteps," when it is actually the package-level
`buildRequirementTraceabilitySteps` — a whole missing threading hop. Stale line
numbers are often a symptom that the prose around them was written against a
tree nobody has re-read. See [[project_extending_a_shipped_plan]].
