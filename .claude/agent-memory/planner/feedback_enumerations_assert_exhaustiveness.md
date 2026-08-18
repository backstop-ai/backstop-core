---
name: enumerations-assert-exhaustiveness
description: An existence-in-world enumeration or "the ONE live sibling lane" claim asserts a count — re-derive it from the real grep at write time, and expect it to go stale within hours in a many-lane tree
metadata:
  type: feedback
---

Any enumeration a plan writes down ("every plan referencing X", "THE ONE live sibling
lane") is read as EXHAUSTIVE. State the deriving command and its literal result count
alongside the list, and re-run it at write time — never carry a count forward from an
earlier round.

**Why:** PLAN-ISSUE-144 burned two extra review rounds on this. Round 3 said "eleven
other plans" when `grep -rln "packval/executor.go" plans/` returned thirteen files
(= twelve others), miscounted a citation-only artifact (PLAN-SPEC-048) as a grep-set
member, and missed two same-day untracked drafts entirely. Worse, its "THE ONE LIVE
SIBLING LANE IN THIS PACKAGE" sentence had become actively false — a second lane
(PLAN-ISSUE-151) was writing `pkg/packval` source with its own red-first `type: test`
windows — so the plan was telling its implementer that no second lane existed, the
exact opposite of what the cross-lane section is for.

**How to apply:**
- Paste the grep command and its file count into the notes; enumerate every member; if
  you cite an artifact that is NOT in the set (a capability-origin plan, an ADR), say so
  explicitly and exclude it from the count.
- `git status`-untracked, same-day `status: draft` plans ARE members. They are the ones
  most likely to be missed and the ones most likely to collide.
- Prefer "TWO live sibling lanes" over "THE ONE" — a superlative claim is a promise you
  cannot keep in a tree with dozens of concurrent lanes. When a sibling lane exists,
  name its file surface and its red-first task IDs in EVERY attribution/check-first list
  in the plan, not only in the notes; an implementer reads the verification task, not
  the preamble.
- Reciprocity is cheap: if a sibling plan already names yours as disjoint, say so and
  return the acknowledgment.

**THE ENUMERATING COMMAND ITSELF CAN BE THE BUG — sanity-check its shape before trusting a
low count.** Authoring PLAN-ISSUE-157 I enumerated a Go test package's symbols with
`for f in *_test.go; do p=$(head -1 "$f"); if [ "$p" = "package engine_test" ]; ...`. `head -1`
assumes the package clause is line 1; two of the three files open with a doc comment, so the
loop silently skipped them and I wrote "all three symbols" into the plan. The real count was
TEN across THREE files — and the one I missed (`conventionRepoRoot`) was the FIRST thing the
implementer would need. Reviewer caught it. A filter that returns suspiciously few members
deserves one falsifying probe (grep the pattern across ALL candidates unfiltered and compare
counts) before the number goes in the artifact. Use `grep -q '^package X' "$f"` over
whole-file matching, not a positional `head`/`sed` read.

Corollary: when the enumeration is FOR an implementer joining an existing package, hand them
the deriving command too, and say the list was accurate as of a date — packages grow, and a
stale "these are the only helpers" line causes exactly the redeclaration collision the guard
existed to prevent.

Related: [[shared-tree-assertions-cannot-attribute]], [[cite-by-name-in-contended-files]].
