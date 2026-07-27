---
title: "A Pack Shipping Recipes + Scaffolds but No Ruleset Has No Valid Archetype — Uninstallable Since SPEC-055"
schema_version: issue/v1

issue:
  id: ISSUE-085
  title: "A Pack Shipping Recipes + Scaffolds but No Ruleset Has No Valid Archetype — Uninstallable Since SPEC-055"
  type: bug
  status: open
  created: "2026-07-26"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# A Pack Shipping Recipes + Scaffolds but No Ruleset Has No Valid Archetype — Uninstallable Since SPEC-055

## Problem

`pkg/packval`'s archetype model recognizes exactly two archetypes — `code` and `enforcement`
(`pkg/packval/phase1.go:42`) — and both demand a non-empty ruleset:

- `archetype: code` fails phase4 with `"code pack requires rules"` whenever
  `Content.Ruleset.Rules` is empty (`pkg/packval/phase4.go:14`), regardless of what else the pack
  ships.
- `archetype: enforcement` fails phase4 with `"enforcement pack must not include scaffolds"`
  whenever `Content.Scaffolds` is non-empty (`pkg/packval/phase4.go:37-39`), and separately
  forbids `content.sdk` (`phase4.go:41-43`).

There is no archetype that admits **scaffolds + recipes, no rules** — a pack whose whole point is
to hand a consumer file-op recipes and paired scaffolds rather than enforce anything. A pack
shaped that way satisfies neither branch: declare `archetype: code` and phase4 rejects it for
having no rules; declare `archetype: enforcement` and phase4 rejects it for having scaffolds.
There is no third option.

This is compounded by a second, independent gap already named in SPEC-054: `pkg/packval`'s
`Content` struct (`pkg/packval/manifest.go:51-54`) has **no `Recipes` field at all** — only
`Ruleset`, `Scaffolds`, `SDK`. `recipes:` is parsed by the separate runtime manifest model
(`pkg/pack/manifest.go:63`, used by `pack add`/gate), but packval's own structural model is blind
to it. SPEC-054 names this explicitly as a deliberate, tracked gap ("A RECIPES-ONLY pack does not
validate yet — `recipes:` alone is not 'content'", `specs/SPEC-054-recipe-apply-and-manifest.spec.md:1419-1427`):
"content is required" is asserted in three independent places (`pkg/pack/manifest.go`,
`pkg/pack/validate_manifest.go`'s `ValidateManifest`, and `pkg/packval`'s phase-1 structural
check) and none of them counts `recipes:`, so a pack shipping *only* recipes fails validation by
design, with a **NAMED FOLLOW-UP** to widen all three sites consistently — deliberately not
built in SPEC-054.

**This issue is the adjacent gap SPEC-054 did not name**, not a duplicate of it: a pack that has
BOTH `recipes:` AND real `content:` (scaffolds paired with the recipes) still has nowhere to land,
because the *archetype* layer — one level above the emptiness checks SPEC-054 scoped out — has no
category that permits "scaffolds + recipes, no ruleset." SPEC-054's follow-up would make an
empty-content recipes-only pack validate; it says nothing about the archetype taxonomy, which is
the wall this issue hits.

### Why this matters now

`pkg/pack/distribution/add.go` REQ-008 (SPEC-055) made `pack check`/`pack test` validation
**unconditional on every `pack add`** — previously a nil `Validator` silently skipped validation
(see the retired ISSUE-073), so this archetype gap was latent. Now every `pack add` runs the full
packval pipeline, so a recipes-primary pack with content is rejected at install time, not just at
`pack check` time.

### Live reproduction (2026-07-27, during ISSUE-079's live verification)

A demo "recipes field-guide" pack — `archetype: code`, content = scaffolds + a `recipes:` index,
deliberately no ruleset (the pack's entire purpose is to hand out recipes, not enforce a rule) —
failed `phase4-archetype`'s `"code pack requires rules"` check. Making it validate required
whack-a-mole additions that no recipes-first pack author actually wants:

1. Add a ruleset with at least one rule, purely to satisfy `code-rules`.
2. Add an `engines:` binding for that rule, because phase2-coherence then fails with `"unknown
   engine"` for a rule with no resolvable engine.
3. Add `pairs_with.rules` on each scaffold (phase4 `pairs_with` check, `phase4.go:26-27`).
4. Add `pairs_with.scaffolds`/`pairs_with.sdk` on the rule itself (phase4 `pairs_with` check,
   `phase4.go:21-22`).

Every one of these exists to satisfy the `code` archetype's enforcement-pack assumptions, not
because the recipes field-guide pack needs a rule at all.

### Impact

- The field-guide demo pack — meant to *teach* this exact pack shape (recipes + scaffolds,
  no enforcement) — cannot validate as authored.
- BUNDLE-015 REQ-018's CI recipe pack (the packs-only acceptance test / first real consumer of
  the recipe capability) will hit the same wall.
- Every future recipe-first pack author hits this at their very first `pack add`, and the only
  documented escape is padding the pack with a decoy rule + engine + pairs_with wiring that
  serves no enforcement purpose — a workaround that actively teaches the wrong pack shape.

## Solution

Recorded as options, not decided — this is a design gap needing a direction call, likely under
DIR-019 (Pack Recipe Capability) or wherever BUNDLE-015/019 land it:

1. **New archetype** (e.g. `archetype: recipes` or `archetype: scaffolding`) admitting
   scaffolds + recipes with no ruleset requirement, and no `pairs_with`/engine demands tied to
   enforcement semantics. Cleanest taxonomically, but grows the archetype enum and needs its own
   phase4 branch (and likely a phase1 content-required accommodation once `Recipes` is modeled at
   all — see below).
2. **Relax `code`/`enforcement`** to admit recipes-bearing content without a ruleset — e.g. `code`
   no longer requires rules when `recipes:` is present. Smaller surface change, but blurs what
   `code` archetype means and risks silently admitting genuinely rule-less "code" packs that
   aren't recipe packs at all.
3. **Document rules-required as intentional** and fix the field guide (and any other
   recipes-primary demo/reference pack) to always carry a token ruleset + engine + `pairs_with`
   wiring. Zero validator change, but bakes the whack-a-mole workaround into the canonical
   teaching example — the opposite of what a field guide is for.

Whichever direction is chosen, it is coupled to (but not blocked by) widening the three
`content-is-required` sites SPEC-054 deliberately left alone — `pkg/pack/manifest.go`,
`pkg/pack/validate_manifest.go`'s `ValidateManifest`, and `pkg/packval`'s phase-1 structural
check — since `pkg/packval`'s `Content` struct does not even parse `recipes:` today
(`pkg/packval/manifest.go:51-54`). A recipes/scaffolding archetype without that widening would
still require the pack to carry a non-recipes content field (scaffolds, as in the field-guide
case) to clear phase1's `"content is required"` check.

## References

- `pkg/packval/phase1.go:42-44` — only two archetypes recognized (`code`, `enforcement`)
- `pkg/packval/phase1.go:45-47` — phase1 `"content is required"`: satisfied by
  `Ruleset.Rules`/`Scaffolds`/`SDK` only; `recipes:` is not in the `Content` struct at all
- `pkg/packval/phase4.go:14-15` — `code` archetype: `"code pack requires rules"`
- `pkg/packval/phase4.go:17-35` — `code` archetype `pairs_with` co-occurrence requirements (rule
  needs `pairs_with.scaffolds`/`sdk`; scaffold needs `pairs_with.rules` resolving to a real rule)
- `pkg/packval/phase4.go:37-44` — `enforcement` archetype: forbids `Content.Scaffolds` and
  `Content.SDK`
- `pkg/packval/manifest.go:51-54` — `Content` struct: `Ruleset`, `Scaffolds`, `SDK` only, no
  `Recipes` field
- `pkg/pack/manifest.go:55-63` — the SEPARATE runtime manifest model that DOES parse the
  top-level `recipes:` index (used by `pack add`/gate, not by packval)
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md:1419-1427` — "A RECIPES-ONLY pack does not
  validate yet" — names the three `content-is-required` sites as a deliberate, tracked, NOT-here
  follow-up; this issue is the archetype-layer gap adjacent to that follow-up, not a duplicate
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md` REQ-008 — validation is now
  unconditional on every `pack add`, which turned this from a latent `pack check` gap into an
  install-time hard blocker
- `issues/ISSUE-073-pack-add-nil-git-cloner-panic.issue.md` — the retired issue describing the
  PRE-REQ-008 world where a nil `Validator` silently skipped validation (why this gap was latent
  until now)
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` — REQ-008 traces here
- `bundles/BUNDLE-015-pack-scaffolding-recipes.bundle.md` REQ-018 — the CI recipe pack that will
  hit this same wall as the packs-only acceptance test / first real consumer
- `directives/DIR-019-pack-recipe-capability.directive.md` — likely backlog home; PM to slot
- Discovered live-reproducing 2026-07-27 during ISSUE-079's live verification, against a demo
  "recipes field-guide" pack (`archetype: code`, scaffolds + `recipes:`, no ruleset)
