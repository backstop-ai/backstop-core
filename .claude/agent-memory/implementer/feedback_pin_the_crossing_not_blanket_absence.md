---
name: pin-the-crossing-not-blanket-absence
description: A blanket "X must never appear here" guard goes RED against the sanctioned seam it was written to protect — assert the allowed crossing exactly-once (this callee, from this caller) instead
metadata:
  type: feedback
---

When a plan says "assert file A contains no call to X and no function that reaches
it", check for a DELIBERATE, MEASURABLE crossing before writing the blanket rule.
Real boundaries usually have exactly one sanctioned door, and the blanket form
condemns it.

Shape the guard as: this callee, from this caller, EXACTLY ONCE. Assert the COUNT,
not just the absence — at zero the seam was deleted or renamed and the guard is
watching nothing; above one the file grew a second path. Name the sanctioned pair in
a `const` and say in the comment why that one is legitimate.

**Why:** ISSUE-020 TASK-032 mandated "sandbox_linux.go contains no call to unix.Exec
and no function that reaches it". But `MaybeRunSandboxHelper` dispatches to
`runSandboxHelper` (sandbox_linux.go:124) — that call IS the measurement seam, and it
is honestly covered, because a malformed spec makes it return at the decode step
before any restriction is installed and before any exec. My literal first draft went
red against the very seam it existed to protect. The plan wording was unsatisfiable as
written and went back to the planner as an amendment.

**How to apply:** when a mandated absence assertion fires on arrival, do NOT weaken it
to make it pass and do NOT assume the code is wrong — first ask whether the hit is the
sanctioned seam. If it is, pin it and report the plan wording as an amendment rather
than silently reinterpreting the task. Falsify both directions in a detached worktree:
deleting the seam must go red just as loudly as adding a second crossing. Related:
[[absence-tests-via-goast]], [[choose-compile-red-or-behavioral-red]],
[[never-stash-shared-tree]].
