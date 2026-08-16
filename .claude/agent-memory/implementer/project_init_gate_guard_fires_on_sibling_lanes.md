---
name: init-gate-guard-fires-on-sibling-lanes
description: SPEC-069 CLM-063's TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage asserts `git status -- pkg/gate` is EMPTY, so it reds for ANY lane touching pkg/gate in a shared dirty tree
metadata:
  type: project
---

`TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage`
(`pkg/initialize/sourceset_scan_test.go`, SPEC-069 CLM-063) fails whenever
`git status --porcelain -- pkg/gate` is non-empty — regardless of WHICH lane
made the change. Any plan that legitimately edits `pkg/gate` reds this
mandated test, and it surfaces in `gate --all` as
`test_verification: mandated_test_failed` attributed to SPEC-069.

**Why:** the test's own ★ comment says it uses working-tree-vs-HEAD rather than
a merge-base diff precisely to avoid blaming other sessions' work. That choice
only excludes *committed* sibling work. Against *uncommitted* sibling work in a
shared tree it has no attribution power at all — the predicate is just
"is pkg/gate dirty", and every concurrent lane trips it equally.

**How to apply:** if this test reds and your plan touched `pkg/gate`, do NOT
assume it is yours and do NOT "fix" it. Attribute it:
`git status --porcelain -- pkg/gate` and count whose files are listed. Then
prove it with a control — copy the tree WITH `.git` to scratch,
`git checkout HEAD --` *only your* pkg/gate files, delete your untracked ones,
and re-run. Measured 2026-08-16 during PLAN-ISSUE-122: 18 dirty entries, 5 mine
and 13 from PLAN-ISSUE-113/118, and the test failed identically with all 5 of
mine reverted. Inherited, not caused.

Pairs with [[never_stash_shared_tree]] and
[[project_gate_all_underreports_vs_diff]]: derive YOUR blocking set by
intersecting `gate --all --json` findings against your own changed-file list.
On that same run 130 findings named zero files I had touched.
