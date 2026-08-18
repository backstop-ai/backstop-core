---
name: inheritance-proof-by-head-worktree-gate-all
description: Proving a `gate --all` red is inherited needs a clean-HEAD worktree with its OWN `pack install`, and the two violation lists must be PATH-NORMALIZED before diffing or every finding looks like a delta
metadata:
  type: project
---

`backstop gate --all` in backstop-core is red by default (measured 2026-08-18:
71 blocking pack_engines + 34 substantiveness + 13 coverage + 2 contract_signature
+ 1 artifact_validation, at a completely clean HEAD). "gate --all must be green"
in a plan is therefore not a readable bar on its own — the only legible statement
is a CONTROL/TREATMENT DIFFERENTIAL.

Recipe that worked:

    git worktree add --detach <scratch>/ctrl-head HEAD
    cd <scratch>/ctrl-head && backstop pack install     # packs are gitignored
    cd <scratch>/ctrl-head && backstop gate --all

`pack install` inside the control is what makes it faithful: it reads the
control's OWN `backstop.yml`, so a lane that relocked a pack gets the PRE-relock
fleet in the control and the post-relock fleet in the treatment. Copying
`.backstop/` across instead would give the control the treatment's installed
tree and a lock/hash mismatch.

**★ NORMALIZE PATHS BEFORE `comm`.** Findings carry ABSOLUTE paths, so the
control's live in the worktree dir and the treatment's in the repo dir — a raw
diff of the two sorted lists reports ~36 bogus deltas in each direction. Strip
both roots first:

    sed -e "s#<repoRoot>/##g" -e "s#<ctrlRoot>/##g" | sort

With that, the real delta was six lines: 4 blocking (all traceable to a SIBLING
lane's uncommitted `ci.yml`/`workflows_test.go`) + 2 advisories, and exactly one
line ONLY IN CONTROL — the finding this lane's relock removed. That single
"only in control" line is the strongest evidence a lane can produce: it shows the
change FIXING something, not just failing to break anything.

**Why:** PLAN-ISSUE-157 required proving inherited reds rather than waiving them.
Related: [[project_redproof_by_worktree_flip]] (single-test flip),
[[project_control_vs_treatment_by_preserved_binary]],
[[project_local_baseline_makes_gate_permissive]].
