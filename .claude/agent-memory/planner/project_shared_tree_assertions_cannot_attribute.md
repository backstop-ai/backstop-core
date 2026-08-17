---
name: shared-tree-assertions-cannot-attribute
description: A test asserting "this change touched no file under X" via git status is unfixable in a shared tree — invert the scan to "does X reference us", which reads content and always has teeth
metadata:
  type: project
---

A test that enforces "this implementation changed no file under `<pkg>`" by reading
`git status --porcelain -- <pkg>` cannot be repaired by narrowing it. This repo runs
several implementer lanes against ONE shared working tree, so a status snapshot has no
lane provenance: it blames whoever runs it. Worse, once the lane commits, the same check
passes over an empty status — zero true teeth when clean, false teeth when dirty.

**Why:** ISSUE-139 — SPEC-069 CLM-063's purity check fataled PLAN-ISSUE-118's lane for
`pkg/gate` edits that lane legitimately owned, and its own "skip if my package is clean"
guard sat AFTER the `t.Fatalf`, so it was unreachable in exactly the steady state it was
written for.

**How to apply:** when planning a fix for this shape, invert the scan direction. Instead
of "did anything under X change" (a property of a change set, unobservable from a tree
snapshot), assert "does any file under X reference US" — read file CONTENT, never
working-tree state. That is attributable (X has no legitimate reason to name us),
shared-tree-independent, and holds in a committed tree where the git check has none.
Then say plainly in the plan that deleting the `t.Fatalf` is a STRENGTHENING, with the
teeth arithmetic, or a reviewer will read it as weakening a mandated test.

Two traps worth carrying forward:

- **Check the dependency direction before forbidding an import.** ISSUE-139 proposed
  "assert init's source imports no `pkg/gate` symbol"; `cmd/backstop/init_seams.go` is in
  init's own globbed source set and legitimately imports `pkg/gate` to build the gate
  runner. A denylist claim forbids CHANGING a package, not CONSUMING its exported API —
  verify which one the requirement means before scoping the scan. See
  [[verify-issue-premises]].
- **For a falsification pass that needs a dirty tree, prescribe an UNTRACKED probe file**
  the implementer creates and deletes, never an edit to a tracked file and never
  `git stash`. Untracked files show in `--porcelain` and revert without touching another
  agent's work. See [[cite-by-name-in-contended-files]].
