---
name: ciyml-byte-identity-guard
description: SPEC-067 CLM-007's TestCIRecipes_CoreAdoptsThePackWithoutApplyingAnyRecipe reds on ANY uncommitted .github/workflows/ci.yml edit — it is a dirtiness check, not a content check, and self-clears on commit
metadata:
  type: project
---

`TestCIRecipes_CoreAdoptsThePackWithoutApplyingAnyRecipe`
(`cmd/backstop/ci_recipes_pack_identity_test.go`) calls `ciGitDiffIsDirty`, which
compares `git show HEAD:.github/workflows/ci.yml` against the on-disk bytes. It
therefore fails on ANY uncommitted edit to that file regardless of what the edit
says — including a plan-MANDATED comment correction.

**Why:** the test proves SPEC-067 CLM-007 ("core adopts the PACK, never a
recipe") and uses working-tree cleanliness as its proxy for "bespoke, not
recipe-generated". The proxy is coarser than the claim.

**How to apply:** if a plan mandates a ci.yml edit, expect this red for the whole
pre-commit window and do NOT revert the edit or weaken the test. It surfaces in
TWO dimensions at once — as a go-test failure AND as `mandated_test_failed`
(SPEC-067 CLM-007) — so it looks like two problems. Prove it self-clears rather
than asserting it: detached worktree at HEAD, `git apply` the ci.yml patch,
COMMIT it, symlink `.backstop/packs` in (the worktree lacks the gitignored packs),
then run the test. Partial credit is still evidence — before the packs were
linked the test failed LATER, at the manifest lookup, which already proves the
dirtiness assertion passed. Related: [[project_green_gate_by_scope_exit]],
[[project_init_gate_guard_fires_on_sibling_lanes]].
