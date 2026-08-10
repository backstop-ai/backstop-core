---
name: pack-rename-migration-recipe
description: renaming an installed local pack (DIR-027 name==coordinate) is a 7-step migration — pack add leaves the OLD backstop.yml key, dir, and lock entry behind, and silently unbinds waivers and dogfood tests
metadata:
  type: project
---

`backstop pack add <local-path>` installs under the pack.yml `name:`, so a renamed
pack lands at a NEW coordinate and everything keyed to the old one is orphaned —
silently. Executed 2026-07-27 for `backstop/go-standards` →
`backstop-ai/go-standards` and `backstop/self` → `backstop-ai/backstop-self`.

Full sequence (each step is separately load-bearing):
1. `pack add <source-path>` — installs the new coordinate and ADDS a
   `backstop.yml` packs key. It does NOT remove the old one; you end up with both.
2. Remove the stale `backstop.yml` packs key by hand.
3. Re-point `enforcement.policy.<dim>.sources.<pack>` keys — a per-pack policy
   override is keyed on the pack NAME and goes inert, not error, when it drifts.
4. `rm -rf .backstop/packs/<old-org>/<old-name>` — `detectExtraUnlocked`
   (`verify.go:91`) walks packsDir org/pack and FAILS any dir not in the lock.
5. Prune the stale `backstop.lock` entries. `pack remove <old-name>` CANNOT do it
   ("pack is not installed"); edit the lock. Note `VerifyLock` skips
   `source_type: local`, so a stale local entry passes verification while lying.
6. Re-key `@waiver:` comments — the waiver's rule-ID embeds the pack path
   (`backstop.packs.<org>.<pack>.rules…`). An unbound waiver does not warn; the
   finding just reappears. Get the new ID by reading `rule` from
   `gate --file <f> --json`, never by hand-deriving it. Verify each adjudicates
   via `active_waivers` in the JSON.
7. Update dogfood/ratchet TESTS that assert the real backstop.yml/lock/install
   path. 8 failed here. Synthetic-fixture tests using their own temp pack of the
   same name must NOT be touched — only the ones reading the real repo.

**Why:** steps 2-7 are all silent failures; only the tests fail loudly.

**How to apply:** prefer deriving install paths from the pack-name constant
(`filepath.Join(root,".backstop","packs",filepath.FromSlash(packName))`) so the
next rename moves one constant. Related:
[[project_pack_copies_and_stale_gate_binary]], [[project_selfpack_b2_token_rule_scope]].
