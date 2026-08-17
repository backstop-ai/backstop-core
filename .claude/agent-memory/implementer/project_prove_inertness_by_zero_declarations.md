---
name: prove-inertness-by-zero-declarations
description: When a gate red lands in shared fixture-validation e2e tests, prove your change is inert by grepping every pack for the DECLARATION your new branch keys on — zero matches means the branch never executes in-repo
metadata:
  type: project
---

A change that adds a guarded branch (`if binding.X != "" { ... }`) can be proven
INERT for the whole in-repo corpus with one grep: search every pack for the
declaration the branch keys on. Zero matches = the branch is unreachable in this
repo, so it cannot be the cause of any red.

**Why:** ISSUE-144 added a `binding.StdoutArtifact` payload-selection branch to
`pkg/packval`'s `RunEngine`. The diff-scoped gate came back with five `go-test`
failures in pack-validation e2e tests (`phase3-fixtures: N validation error(s)`
for `packs/contracts` and the substantiveness zeromatch pack) — all in the same
package family the change lives in, all shaped exactly like something the change
could have broken. `grep -rn "stdout_artifact" packs/` returned ZERO, which
settled attribution in one command: no local pack declares the field, so the new
branch never runs during any in-repo fixture validation. The reds were
PLAN-ISSUE-142's and ISSUE-148's known windows.

**How to apply:** Reach for this before reading a single failing e2e test body,
whenever your change is additive-behind-a-declaration and the reds are in a
shared package. It is stronger and far cheaper than a worktree control run, and
unlike a test-name check it survives the case where the failing test has no
obvious owner. It does NOT apply to changes that alter an unconditional path —
there the branch is always live and you still need
[[project_control_vs_treatment_by_preserved_binary]].

Pairs with [[project_gate_contention_in_shared_tree]]: these gate runs took ~20
and ~27 minutes with 6–9 concurrent gate processes, so a cheap textual
attribution avoids a second half-hour run you would otherwise burn to be sure.
