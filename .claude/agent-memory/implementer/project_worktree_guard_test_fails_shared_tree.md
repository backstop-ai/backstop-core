---
name: worktree-guard-test-fails-shared-tree
description: SPEC-069's TestInit_ChangesNoGatePackageFile fatals on ANY dirty pkg/gate file, so it reds for every lane in a shared multi-agent tree — inherited, self-clearing, not a code defect
metadata:
  type: project
---

`TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage`
(`pkg/initialize/sourceset_scan_test.go`, SPEC-069 CLM-063) fatals whenever
`git status --porcelain -- pkg/gate` is NON-EMPTY. It asserts attribution against
the WORKING TREE vs HEAD, deliberately, "because a merge-base diff carries every
commit anyone landed since the branch point."

**Why it misfires:** that reasoning assumes ONE lane owns the working tree. In
this repo's multi-agent shared tree it is false — measured 2026-08-16 during
ISSUE-122: 18 dirty entries under `pkg/gate`, 16 belonging to sibling lanes and 2
to ISSUE-122 itself (and ISSUE-122's were MANDATED by its own plan's TASK-008,
which threads `nonCorpus` through `FindUngatedArtifacts`). So the test reds for
whoever runs the gate, regardless of whose work caused it, and it surfaces TWICE
in one run — once as a `pack_engines`/go-test finding and once as a
`test_verification` `mandated_test_failed`. One root cause, two dimensions.

**How to apply:** when a gate run shows this test failing, do NOT hunt for a code
defect and do NOT change the test to pass. Run `git status --porcelain --
pkg/gate` and attribute the entries. It self-clears once lanes commit (and its
own non-vacuity guard then SKIPS it, since it needs uncommitted `pkg/initialize`
files to attribute anything). Report it as inherited/structural. The trustworthy
signal for your own work in a contaminated tree is a scoped
`backstop gate --file <your files>` run, not the diff-scoped verdict.

See also [[feedback_never_stash_shared_tree]],
[[feedback_netnegative_gate_baseline]], [[project_green_gate_by_scope_exit]].
