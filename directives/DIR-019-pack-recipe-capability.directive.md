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

ISSUE-080 (`recipe apply` discards its declared `manual:` fallback; violation-
class failures exit 1 silently) is a recipe-capability defect by subject
matter, but its root cause is the shared `ExitViolations`/`main.go`
stderr-suppression convention that DIR-026's BUNDLE-006 seeds claim by name.
Home unresolved — escalated to Brandon 2026-07-26, not yet cited by this
directive.

**Sequencing (founder ack 2026-07-26, backlog-pm ISSUE-079 escalation):**
ISSUE-079 is next work under this directive, ahead of ISSUE-081. ISSUE-079
(risk: critical — untemplated site fields, silent wrong output) is the
correctness defect blocking trust in the mechanism; ISSUE-081 (authoring-
surface polish — merge/insert/`--param` semantics) follows it.
