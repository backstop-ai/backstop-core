---
name: closeout-verify-the-fix-landed
description: At close-out, verify the PRODUCTION fix reached HEAD — a plan can read `completed` at HEAD while its core diff sits uncommitted, shipping tests without their subject
metadata:
  type: project
---

At close-out, confirm the production change is IN HEAD — not just that the plan's
`status:` field says `completed`. Check the fix's own code at HEAD by its distinguishing
structure, and diff the closeout commit's file list against the lane's declared files.

**Why:** PLAN-ISSUE-091 (2026-08-16). Commit b113d12 "closeout: PLAN/ISSUE-091" shipped 12
files — both new test files, the inverted mandated assertion, the doc corrections, four
follow-on issues, and the AS-BUILT banner declaring the issue delivered — but OMITTED
`cmd/backstop/pack_gate.go`, which held the entire fix. A sibling then closed ISSUE-091 as
delivered (4e0b852) on top of it. Net result: HEAD carried tests asserting behavior HEAD did
not implement, a closed issue, a `completed` plan, and doc/issue text describing a change
that was never committed. Every artifact agreed the work was done; the code disagreed.

**How to apply:**
- **Grepping for a line the fix touches is NOT proof the fix landed.** At HEAD,
  `excludeTestdataPaths(scope.Files)` was present and looked like the fix — `git log -S` on
  that string attributed it to 92d5c2e (ISSUE-040), sitting inside the OLD two-branch
  structure. Assert on the DISTINGUISHING STRUCTURE instead: the defect read
  `if scope == nil || scope.Mode == gate.GateScopeModeAll`, the fix reads `if scope == nil`.
  Verify with `git show HEAD:<file> | sed -n '<range>p'`, and use `git log -S` to date any
  line you are tempted to read as evidence.
- **`git show --stat <closeout-commit>` and reconcile against the lane's files.** A missing
  production file is invisible from the plan artifact alone.
- **The specific failure mode is a SPLIT LANE, and shared files are what cause it.** The
  omitted files were exactly the ones shared with other live lanes
  (PLAN-ISSUE-067/140 both had uncommitted work in `pack_gate.go`). Whoever staged the
  commit excluded them to avoid sweeping the other lanes — protecting them, but splitting
  this lane across the fix boundary.
- **Therefore say BOTH halves when advising on a shared-tree commit.** "Stage explicit paths,
  never `git add -A`" is only half the instruction and, alone, invites this exact omission.
  Name the lane's MUST-INCLUDE production files in the same breath. I gave only the
  protective half and the omission followed from it.
- Closed-requires-traceability checks verify the plan's STATUS FIELD, not that the plan's
  code landed. That blind spot is why this reached HEAD unchallenged.
- Remedy needs the lead's call when the missing files are shared: either the sibling lanes
  commit first and the fix lands as a follow-up, or `git add -p` stages only this lane's
  hunks. Do not unilaterally commit a shared file
  ([[feedback_git_stash_shared_tree_hazard]] is the same hazard by a different verb).

- **A close-out can be asked on a lane that is ENTIRELY uncommitted, plan file included.**
  PLAN-ISSUE-178 (2026-08-20): all four code files plus the plan artifact itself were
  working-tree-only at close; HEAD was an unrelated commit. That is legitimate — the fix and
  its verification were real — but it changes what the banner may claim. Two things must be
  said out loud instead of implied: there is NO Linux CI confirmation and cannot be one until
  the diff is pushed (so do not borrow the CI-run-citation shape of a committed close-out),
  and there is no `git show <sha>:` as-reviewed baseline to diff the banner against, so
  "left byte-identical below this banner" rests on the close-out's word, not on a commit.
  Verify the fix by READING THE WORKING TREE for its distinguishing structure, exactly as
  you would read HEAD.

Related: [[project_plan_closeout_convention]] (uncommitted-shared-tree over-attribution, and
verifying the hand-off's promised follow-ups actually landed — the same "trust the report vs
read the tree" failure in two other shapes, all three hit on the same plan).
