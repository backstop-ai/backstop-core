---
name: go-distribution-pack-shipped
description: backstop-ai/go-distribution v0.1.0 is PUBLIC and remotely consumable — the release trinity as a scaffolding recipe + 9 rules; hash equality across local/remote installs proven, stash is the first consumer
metadata:
  type: project
---

Shipped 2026-07-29 (PLAN-ISSUE-101). `backstop-ai/go-distribution` v0.1.0 is public:
a `kind: scaffolding` recipe (`go-release`) emitting .goreleaser.yml, release.yml,
tag-integrity.yml and a SELF-CONTAINED version starter + table test, plus nine
semgrep rules (6 ERROR / 3 WARNING) guarding what the recipe establishes.
Consume with `pack add backstop-ai/go-distribution@0.1.0`.

Load-bearing facts a future session should not re-derive:
- **The pack MUST declare exactly one provisioned engine binding**, because
  `recipe apply` calls provisionedEngineBinding UNCONDITIONALLY before Apply,
  regardless of op family. Ours is named `semgrep-godist`, NOT `semgrep` — pack
  bindings OVERRIDE the embedded base-engines defaults, and redefining `semgrep`
  would silently re-resolve the engine for every OTHER pack in a consumer.
  Verified: go-standards/go-toolchain/backstop-self still dispatched alongside it.
- **Foreign templates ride self-emitting pass-through params** (name = the
  template's inner text, default = that text wrapped). Eight of them. Values are
  never rescanned, which is the whole mechanism. See
  [[project_recipe_payload_sharp_edges]] — including that the substituter reads
  `{{ }}` inside COMMENTS.
- **`pack remove` BEFORE any re-add** (ISSUE-095: a plain add over an installed
  same-name pack silently no-ops, so a local install would masquerade as a remote
  proof). Doing it right yields `source_type: git`, `git_ref: v0.1.0`, and a
  content_hash EQUAL to a fresh local install of the same tree — that equality is
  what proves the published tree is the proven tree.
- **stash is the first consumer** and deliberately ships NO `brews:` block and no
  tap token (publication coordinates are an open founder question), so it carries
  exactly ONE non-blocking warning. Its `.gitignore` needed `dist/` added BY HAND:
  a create op refuses to touch a file the consumer already owns, and .gitignore is
  not a mergeable format.

**Why:** this is the reference for "what a real pack looks like end to end" —
recipe + rules + captured fixtures + falsification harness + acceptance harness.

**How to apply:** copy its SHAPE for the next pack; do not re-solve the engine
binding, the template escape, or the remote-proof sequence. Related:
[[project_pack_recipes_archetype_gate_order]], [[project_sarif_warning_severity_lost]].
