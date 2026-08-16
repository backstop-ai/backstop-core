---
name: pack-release-push-order
description: Push a pack release's main BEFORE its tag, and git fetch before trusting ahead/behind — a stale origin ref let a tag land on a commit unreachable from main
metadata:
  type: project
---

When shipping a pack release (version bump + commit + tag + push), **push `main`
first and the tag second**, and run `git fetch origin` before reading
`git rev-list --count origin/main..main`.

**Why:** shipping bun-toolchain v1.3.0 for ISSUE-122, the pre-flight ahead/behind
check read a STALE `origin/main` ref (no fetch first), so it reported "0 ahead /
in sync" while origin had actually gained a commit months earlier. `git push
origin main` was then rejected non-fast-forward — but the tag push, issued as a
separate command, SUCCEEDED. For a minute the remote carried tag `v1.3.0`
pointing at a commit unreachable from any branch. Because packs resolve BY TAG
(`pack add`/`pack update`), a consumer resolving that version would have gotten
correct content from an orphaned commit — the release was live and the repo was
inconsistent at the same time, which is the worst combination to debug later.

**How to apply:** the fix is NOT to force-move or delete the published tag — the
plan text forbids re-tagging, and force-moving a tag a consumer may have hashed
is indistinguishable from tampering. Instead `git fetch`, MERGE `origin/main`
into the release commit (a release bump touches `pack.yml`; the divergent commit
is usually CI/workflow, so there is no conflict), then push `main`. Verify with
`git merge-base --is-ancestor <tag-sha> origin/main` and read the answer — that
is the check that the tag is genuinely on the branch, and it is one command.

Ordering main-then-tag makes the whole failure unreachable: a rejected `main`
push stops you before the tag exists anywhere.

See also [[project_pack_ships_after_the_core_release]],
[[project_pack_rename_migration_recipe]].
