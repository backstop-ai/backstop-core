---
title: "Pack Recipe Capability"
number: DIR-019
created: "2026-07-14"
schema_version: directive/v1

directive:
  status: active
  source:
    - "BUNDLE-015"
    - "SPEC-054"
    - "ISSUE-079"
    - "ISSUE-081"
    - "ISSUE-080"
    - "ISSUE-085"
---

## Description

Build the **pack scaffolding-recipe capability**: packs carry scaffolding
recipes (a template plus a pack-declared target path), and core applies them
via a generic, language/platform-blind mechanism — copy template to
pack-declared target, nothing more. No recipe logic, target-path convention,
or platform knowledge is baked into backstop itself; every recipe's shape and
destination is entirely pack-declared. This preserves the thin-executor
invariant (zero baked language/tool knowledge) that `backstop/self` already
enforces — a baked recipe mechanism would be exactly the kind of violation
that invariant exists to catch.

Two consumers of this capability:

1. **Language project scaffolding** — e.g. `pack add ts` bootstraps a starter
   TypeScript project via its recipe; the same mechanism works for any future
   language pack with zero core changes.
2. **CI recipe pack** — per-platform gate-workflow templates (GitHub Actions,
   etc.) that `backstop init` applies by default to wire CI gating out of the
   box.

**This directive is a blocking dependency for DIR-002 (`backstop init`
Command).** BUNDLE-003 resolved (all 7 OQs) that `backstop init` bakes zero
language/platform knowledge itself — it delegates all project scaffolding and
CI gate-workflow templating to packs via this recipe mechanism. `backstop
init`'s spec cannot be written until the recipe capability it depends on
exists, so DIR-019 must land — at least through a spec — before DIR-002's
implementation proceeds.

This directive is sourced from BUNDLE-015 (pack-scaffolding-recipes), which
has since been promoted by the founder to `defined` maturity, and SPEC-054
has been written and implemented against it (see below). The bundle still
carries scope this directive has not yet delivered — the CI recipe pack
consumer in particular (see Notes) — so the directive remains `active`
pending that follow-on work.

### Delivery status

SPEC-054 (`implemented`) shipped the MECHANISM: `pkg/recipe` + the `recipe
apply` CLI, resolving a recipe by `<pack>:<recipe>@<recipe_version>` and
applying its declared ops against a pack-declared manifest. First dogfood of
that mechanism surfaced two defects that must close before it is trustworthy
for the CI-recipe-pack consumer: ISSUE-079 (risk: critical — `recipe apply`
substitutes only payload/fragment CONTENT, never the site fields
(Op.Target/Anchor/Snippet), so it silently writes literal `{{ }}` paths and
still exits 0 — silent wrong output, not a crash), and ISSUE-081 (the
authoring surface itself is underspecified — merge fragment form, `--param`
CLI input, and insert splice semantics are all unpinned). Mechanism-complete,
not yet trustworthy.

## Notes

**Delivered so far:** SPEC-054 covers BUNDLE-015 REQ-001..010/021/023/024 —
13 of 24 REQs. The remaining 11 are uncovered, including REQ-018 (the CI
recipe pack — this directive's own packs-only acceptance test). Zero packs
in the published fleet declare a `recipes:` block in their `pack.yml` yet.
With remote pack consumption (DIR-026) and the fleet migration (DIR-027)
now underway, recipes are the sole remaining tier-1 launch long-pole.

Added to BACKLOG.yml immediately before DIR-002, since DIR-002 depends on
it, but after DIR-001 (Release Workflow) and DIR-003 (Baseline
Implementation) — both are also `backstop init` prerequisites and were
already prioritized ahead of DIR-002. This placement is a defensible default,
not a unilateral reprioritization; the founder should reposition if they
disagree with where this lands relative to other queued work.

The repo's `recipes/` directory currently contains three EMPTY subdirectories
(`go`, `meta`, `typescript` — each just a `.gitkeep`), and no pack in the
installed fleet declares a `recipes:` block in its `pack.yml`. The apply
mechanism is complete (modulo ISSUE-079/081) but has zero consumers:
BUNDLE-015's "CI recipe pack" consumer — the default-CI-wiring story that
`backstop init` depends on — remains unbuilt.

ISSUE-080's original two-problem report split on reconciliation (2026-07-26):
the shared `ExitViolations`/`main.go` stderr-suppression root cause (dropped
`manual:` fallback, silenced diagnostics) was fixed generically, repo-wide, by
SPEC-055 (`reportError`/`Explained` in `cmd/backstop/main.go`) — see SPEC-055
for that half's delivery. What remains open under ISSUE-080, and is now cited
by this directive, is the recipe-specific remainder: `recipe apply` silently
clobbers an operator's manually-diverged edit to a recipe-owned file when the
guarding `@waiver:` token carries a malformed reason code — a data-loss defect
in `pkg/recipe/apply.go`'s waiver-divergence adjudication that `reportError`
never touches, since it fails silently at exit 0.

**Sequencing (founder ack 2026-07-26, backlog-pm ISSUE-079 escalation):**
ISSUE-079 is next work under this directive, ahead of ISSUE-081. ISSUE-079
(risk: critical — untemplated site fields, silent wrong output) is the
correctness defect blocking trust in the mechanism; ISSUE-081 (authoring-
surface polish — merge/insert/`--param` semantics) follows it.

**ISSUE-085 (recipe-pack archetype gap) — added 2026-07-27.** A pack whose
whole point is handing out scaffolds + recipes with no ruleset has no valid
archetype: `pkg/packval`'s model recognizes only `code` (requires rules) and
`enforcement` (forbids scaffolds), so a recipes-first pack satisfies neither
branch. This was latent behind the pre-SPEC-055 nil-`Validator` skip (the
retired ISSUE-073) and was EXPOSED by SPEC-055 REQ-008 making `pack
check`/`pack test` validation unconditional on every `pack add` — turning a
`pack check`-time gap into an install-time hard blocker. It sits on REQ-018's
critical path: BUNDLE-015's CI recipe pack — this directive's own
packs-only acceptance test — is precisely this pack shape (scaffolds +
recipes, no rules) and will hit the same wall. DIR-026 caused the exposure
(REQ-008 shipped there); DIR-019 owns the fix, since it's this directive's
archetype/recipe-capability surface. Founder-decided 2026-07-27: direction 1
— a new `recipes`/scaffolding archetype, WITH TEETH: every non-templating
(`scaffolding`- and `implementing`-kind) recipe in the pack must declare its
own `enforcement.rules`; `templating`-kind recipes are exempt, since the
applier itself is the drift enforcement for the other two kinds.
