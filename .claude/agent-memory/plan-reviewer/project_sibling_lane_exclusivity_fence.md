---
name: sibling-lane-exclusivity-fence
description: Same-night sibling plans often carve an explicit file-exclusivity split in ONE plan's notes naming the other issue; grep the SIBLING's notes for your issue ID before passing on scope
metadata:
  type: project
---

When two issues are filed the same day on adjacent surfaces, one planner often writes an
explicit file-exclusivity fence into ITS notes naming the other lane — and the other plan,
authored concurrently, never sees it and crosses the line.

**Why:** PLAN-ISSUE-140 (2026-08-16) reviewed FAIL because PLAN-ISSUE-092, uncommitted/draft
in the same working tree, carried a section "F6. WHY pkg/packval/executor.go IS NOT IN SCOPE"
that read: "the negative loop discards the error text entirely ... THIS LANE OWNS (a) — CLM-005"
and "executor.go ... EXACTLY the surface of ISSUE-140 ... THIS LANE DOES NOT TOUCH executor.go.
This is a deliberate file-exclusivity split between two in-flight lanes." PLAN-092 honored it;
PLAN-140 crossed it, and with a CONTRADICTORY mechanism (keep `Check: "semgrep-negative"` +
append err vs. move the error to a dedicated engine-error Check and suppress the fixture
verdict). Both plans validated clean and were internally congruent to their issues.

**How to apply:** existence-in-world is not just "does another plan touch these files."
1. `git status --porcelain plans/` — UNCOMMITTED drafts are the live collision surface, not
   the committed corpus. Sort by `created:`; same-day siblings are the risk.
2. grep the sibling plan's NOTES for your issue ID and for the words "scope", "owns",
   "not in scope", "split". A fence naming your lane is dispositive.
3. Overlap alone is not the finding — check whether the two MECHANISMS can coexist. Ask:
   whose test reds if the other lands second? Name that test in the report.
4. Also check whether the sibling INVERTS a predicate your plan's at-HEAD falsification
   baseline depends on. PLAN-092's polarity flip would have silently evaporated PLAN-140's
   "clean pass IS the vacuous green" evidence — an order-dependent falsification that the
   plan never disclosed.

Related: [[verified-enumeration-do-not-rederive]], [[shortcircuit-dependent-guard]]
