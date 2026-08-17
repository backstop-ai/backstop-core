---
name: mutation-matrix-beats-sequence-red
description: Prove a falsification corpus by mutating the finished impl into each WRONG variant and recording which test catches it — but first confirm the mutation actually CHANGES behavior, or a no-op mutation reads as a weak test
metadata:
  type: project
---

For a lane whose whole product is a set of falsifiers (a validation corpus, a
detection rule, a set of sharp-edge guards), the strongest evidence is not
red-then-green sequence and not even a single worktree flip. It is a **mutation
matrix**: take the FINISHED implementation, mutate it into each specific wrong
variant the plan's sharp edges name, and record which mandated test catches
each one.

PLAN-ISSUE-151 (2026-08-17): 10 packval sharp edges, each mutated one at a time
in a detached worktree, each caught by exactly the test the plan predicted —
union-across-sibling-rules → `FixtureMaskAttributedPerSemgrepRule`, ANY-instead-of-EVERY
→ `RequiresEveryFixtureCovered`, ALL-instead-of-ANY → `FixtureMaskAdvisoryFires`,
pack-wide fixtures → `FixtureMaskScopedPerRule`, dropped InputMode gate →
`NonRuleFlagsEngineNotScanned`, and so on. That table is what makes "the corpus
is non-vacuous" a measurement instead of a claim.

★ **THE TRAP: A MUTATION THAT DOES NOT CHANGE BEHAVIOR READS AS A WEAK TEST.**
Two of my first-pass mutations came back "NOT CAUGHT (suite still green)" and
both were *bad mutations*, not weak tests:

  * I deleted an `if len(hooks) == 0 { continue }` guard to simulate
    "raise the mask whenever an inert pattern exists". But the very next line
    already returned false for an empty hook set, so the guard was redundant and
    deleting it changed nothing. The real mutation had to remove BOTH guards.
  * I swapped a collapse-first loop for a per-item loop to simulate "no dedup".
    But an explicit dedup map downstream still caught it. The real mutation had
    to remove both the collapse AND the map.

**Why it matters:** the instinct on seeing "NOT CAUGHT" is to go strengthen the
test. That would have been wasted work twice, and would have added assertions
defending against a defect that cannot occur. Before concluding a test is weak,
verify the mutant actually behaves differently — the cheapest check is that the
mutated build produces a different output on the fixture the test reads.

**How to apply:** script it (python driving `str.replace` + `go test`, restoring
the original after each run) so all N mutations run unattended in one pass, and
report the matrix. Expect some mutants to fail to COMPILE for mechanical reasons
(unused import, unused variable) — that is not a result, fix the mutant and
re-run it rather than recording it as caught. And a genuinely redundant guard
that no mutation can expose is fine to keep as defense-in-depth; say so plainly
rather than deleting it or pretending it was pinned.

Related: [[project_redproof_by_worktree_flip]],
[[feedback_noregression_guards_cannot_be_red]],
[[feedback_never_stash_shared_tree]].
