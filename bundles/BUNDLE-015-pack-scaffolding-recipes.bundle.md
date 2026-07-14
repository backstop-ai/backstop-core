---
title: "Pack Scaffolding Recipes"
number: BUNDLE-015
created: "2026-07-14"
schema_version: bundle/v2

bundle:
  name: pack-scaffolding-recipes
  version: "0.2.0"
  created: "2026-07-14"
  updated: "2026-07-14"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Backstop is a thin executor with a hard invariant: detection, framework/language,
    and CI-platform knowledge live in PACKS as data, NEVER in core/CLI (backstop/self
    enforces it). But onboarding needs to SCAFFOLD things that are inherently
    language- and platform-specific — a starter TS or Go project skeleton, a
    `.github/workflows/backstop.yml` gate workflow, and so on. The only way to
    scaffold without baking that knowledge into core is for PACKS to carry the
    templates ("recipes") and for a GENERIC core mechanism to apply them. That
    capability does not exist yet, and it is a BLOCKING dependency for `backstop init`
    (BUNDLE-003 DD-12): init consumes recipes but cannot be built until they exist.
    This bundle defines how a pack declares and ships a scaffolding recipe, and how
    init / `pack add` apply one — copy-template-to-declared-path, with both the
    template and its target path owned entirely by the pack.

  user_story: >
    As a pack author, I want to ship a starter project skeleton and a CI gate-workflow
    template inside my pack so that a consumer running `backstop init` (or
    `backstop pack add <me>`) gets those files materialized in their repo — without
    backstop core containing a single line that knows my language or CI platform. As
    the consumer, I want that scaffolding applied non-destructively: it fills in what's
    missing and never clobbers files I already have.

solution:
  approach: >
    Define a pack-recipe capability: a recipe is template CONTENT plus a self-declared
    target PATH (the recipe says where it installs, e.g. `.github/workflows/backstop.yml`);
    core supplies no path knowledge. Application is a single GENERIC
    copy-template-to-declared-path mechanism with no language- or platform-aware
    branches — dumb selectors (`--ci github`, `pack add ts`) index into a pack's
    recipes. Application is non-destructive (converge, never clobber), aligning with
    init's stance. The CI recipe pack — a cross-cutting pack holding per-platform
    gate-workflow templates (github/gitlab/bitbucket/jenkins) — is the first and
    canonical consumer. The MECHANISM details (recipe file format, manifest
    declaration shape, invocation/selection model, conflict handling, whether a
    templating engine is needed) are the open questions this bundle exists to resolve;
    the FRAME above (template + self-declared path, generic apply, all knowledge in
    pack data) is inherited as settled input from the BUNDLE-003 init session.
---

# Pack Scaffolding Recipes

## Current Thinking

### Provenance: spun out of the init bundle

This bundle was carved out of **BUNDLE-003 (onboarding / `backstop init`)** during the
2026-07-14 OQ-resolution session. Init resolved (OQ-7, DD-12/13) that it does ZERO baked
language/platform knowledge: both (a) language project scaffolding and (b) CI workflow
templates are delivered by PACKS as "recipes," and init / `pack add` only APPLY those
recipes through a generic mechanism. That apply mechanism does not yet exist. BUNDLE-003
explicitly records it as a **BLOCKING DEPENDENCY** ("init consumes recipes but cannot be
built until this exists — likely its own bundle"). This is that bundle.

### The settled frame (inputs, not open)

The init session decided the SHAPE of the capability; those decisions arrive here as
inputs (captured as Draft Design Decisions below), not as things to re-litigate:

- A recipe = template content + a **self-declared target path**. The recipe owns where
  it installs; the applier supplies no path knowledge.
- Application is a **generic copy-template→declared-path** mechanism — no
  language/platform-aware branches in core. `--ci github` / `pack add ts` are dumb
  selectors that index into a pack's recipes.
- **HARD INVARIANT (inherited, BUNDLE-003 DD-13):** all language/platform knowledge stays
  in recipe DATA. A language/framework/platform name appearing in core CLI code is the
  bug; `backstop/self` enforces the boundary.
- **Non-destructive** application (aligns with init's converge-never-clobber): a recipe
  must not clobber existing files.
- The **CI recipe pack** is the first/canonical consumer — a cross-cutting pack holding
  per-platform gate-workflow templates.

### What's genuinely open

Everything about the MECHANISM below the frame: the on-disk recipe format, how a pack
DECLARES recipes in its manifest, how application is invoked while staying prompt-free
under init's omakase model, how conflicts are handled, whether one pack can ship many
recipes and how they're addressed, whether "CI recipe" is a distinct kind, and whether
any templating (variable substitution) is needed — and if so, whether introducing it
re-imports baked knowledge through the back door. These are the Open Questions.

### Naming-collision hazard (surfaced early, load-bearing for OQ-2)

The word "scaffold" is already overloaded in this codebase and MUST NOT be conflated with
this capability:

- `pkg/pack/manifest.go` `Content.Scaffolds []Scaffold` — a rule's paired TEST scaffold
  (tier, `test_command`, `pairs_with.rules`). This is authoring metadata about a rule's
  fixtures, not a consumer-repo file template.
- `pkg/pack/scaffold.go` / `backstop artifact new` (`ValidPackTypes = engine / mechanism /
  toolchain`) — scaffolds a NEW PACK's own `pack.yml`. This is about authoring a pack,
  not applying a pack's contents to a consumer.

Neither is the "pack ships a template that init materializes into a consumer repo" concept
here. OQ-2 (manifest declaration) must pick a name/shape that does not collide with these.
The working term in this bundle is **recipe**, deliberately distinct from **scaffold**.

## Draft Design Decisions

These record the frame inherited from the BUNDLE-003 init session. They are settled inputs
(the founder resolved them there); the mechanism that realizes them is what the OQs explore.

- **DD-1: A recipe = template content + a self-declared target path.** The recipe names its
  own install destination (e.g. `.github/workflows/backstop.yml`). The applier (init /
  `pack add`) contributes NO path knowledge. Source: BUNDLE-003 DD-12.

- **DD-2: Application is a generic copy-template→declared-path mechanism.** Core has no
  language- or platform-aware branch. `--ci github` and `pack add ts` are dumb selectors
  that index into a pack's declared recipes; the applier resolves the selected recipe and
  copies its template to the recipe-declared path. Source: BUNDLE-003 DD-12 / OQ-7.

- **DD-3: HARD INVARIANT — all language/platform knowledge lives in recipe DATA, never in
  core.** A language, framework, or platform name appearing in core CLI code is the bug;
  `backstop/self` enforces it. Recipes, CI templates, and starter skeletons are all data
  supplied by packs. Inherited verbatim from BUNDLE-003 DD-13 — the invariant that makes
  this whole capability safe.

- **DD-4: Application is non-destructive.** A recipe must not clobber an existing target
  file. This aligns with init's converge-never-clobber stance (BUNDLE-003 OQ-6). The exact
  behavior when the target already exists (skip / error / merge) is OQ-4.

- **DD-5: The CI recipe pack is the first and canonical consumer.** A cross-cutting pack
  holding per-platform gate-workflow templates (github / gitlab / bitbucket / jenkins) as
  data, applied via the generic mechanism. It is a spec seed / first consumer here, NOT a
  separate artifact. It is also the forcing function that keeps the mechanism honest — if
  the CI pack needs a language/platform branch in core to work, the design has failed.
  Source: BUNDLE-003 OQ-7.

## Open Questions

Genuinely open — the mechanism below the settled frame. To be worked one at a time with the
founder; none are pre-resolved. Maturity stays `exploring` until the founder resolves these
and triggers promotion.

- **OQ-1 — Recipe FORMAT.** What is a recipe on disk? Options: (a) a template DIRECTORY
  copied as a tree; (b) individual declared files; (c) a single template file per recipe.
  And is it PURE STATIC COPY, or is there variable SUBSTITUTION (project name, module path,
  etc.)? Static copy is the simplest and safest against baking knowledge; substitution is
  more useful for real starter skeletons but opens OQ-7. Lean: TBD — depends on whether the
  first real recipes (CI workflow, TS/Go starter) actually need any variables.

- **OQ-2 — Manifest DECLARATION.** How does a pack declare its recipes? Options: (a) an
  explicit `recipes:` block in `pack.yml` listing id → template → target path; (b)
  convention-based directory (e.g. a `recipes/` dir the applier discovers). Must NOT collide
  with the existing `content.scaffolds` (rule test scaffolds) or `pack scaffold`/`artifact
  new` (pack authoring) — see the naming-collision hazard above. Lean: explicit block, since
  the self-declared target path (DD-1) is data that has to live SOMEWHERE the manifest can
  validate.

- **OQ-3 — INVOCATION under omakase.** Is applying a recipe opt-in per recipe (a
  flag/selector like `--ci github`), and how does it stay PROMPT-FREE per init's omakase
  model (BUNDLE-003 OQ-2)? Does `pack add <lang>` auto-apply that pack's recipe, or
  is scaffolding a separate explicit act? Reconcile with: init installs the omakase base
  prompt-free and you SUBTRACT via flags. Lean: TBD — the tension is auto-apply (fewer steps)
  vs. surprise file creation on `pack add`.

- **OQ-4 — CONFLICT handling.** When the recipe's declared target path already exists, what
  happens? Options: (a) skip silently; (b) skip with a loud, actionable notice; (c) hard
  error; (d) attempt a merge. Must reconcile with DD-4 (non-destructive) and init's
  converge-never-clobber. Lean: skip-with-loud-notice (honors non-destructive AND the
  loud-≠-blocking principle), but the founder decides.

- **OQ-5 — MULTIPLICITY & addressing.** Can one pack ship MULTIPLE recipes (e.g. a TS
  toolchain pack shipping a starter-skeleton recipe AND the CI pack shipping N per-platform
  recipes)? If yes, how are recipes ADDRESSED/SELECTED — by id? by a selector key the CLI
  maps to a recipe (`--ci github` → recipe `github`)? This is tightly coupled to OQ-2 and
  OQ-6. Lean: yes, multiple; addressed by a stable recipe id the selector resolves.

- **OQ-6 — Is "CI recipe" a distinct KIND?** Is a CI workflow template a distinct recipe
  KIND (with its own semantics), or is it just an ordinary recipe whose self-declared target
  path happens to be under `.github/`? A distinct kind risks re-importing platform knowledge
  as a taxonomy; "just a path" keeps the mechanism uniform. Lean: just a path — no CI-specific
  kind — but confirm nothing (selection, conflict policy) actually needs a CI distinction.

- **OQ-7 — Templating engine (and its baking risk).** If OQ-1 chooses substitution, WHAT
  does the substitution — a real templating engine, or a tiny fixed variable set? Does
  introducing templating risk baking knowledge into core (e.g. if core has to KNOW what
  `{{module_path}}` means for a given language)? The invariant (DD-3) says the variable
  vocabulary, like everything else, must be pack-supplied data, not core knowledge. Lean:
  start with pure static copy (no engine); only add substitution if a real recipe proves it
  necessary, and keep the variable set data-driven.

- **OQ-8 — Recipe LIFECYCLE on pack upgrade.** When a pack ships a new or updated recipe and
  the target path already exists, DD-4 (non-destructive / never-clobber) means it is simply
  never re-applied. Is that the INTENDED answer — recipes are one-shot bootstraps, and keeping
  a project current after a pack upgrade is the consumer's problem — or does there need to be
  an explicit update / re-apply / diff / merge path? And if so, how does that path NOT violate
  non-destructive (DD-4)? This is the fork: one-shot bootstrap vs. a managed-file relationship
  the applier keeps current. Lean: non-destructive dissolves it into one-shot bootstrap
  (scaffolded files become consumer-owned code the moment they land), but keep it explicitly
  open — a real update path may be wanted for CI workflows that should track the pack.

- **OQ-9 — Cross-pack target COLLISION (recipe-vs-recipe).** Two packs both declaring a recipe
  at the SAME target path (e.g. both wanting `.github/workflows/backstop.yml`). How does the
  generic applier DETECT and resolve it — fail loudly? first-wins / last-wins? namespaced
  paths? This is DISTINCT from OQ-4: OQ-4 is recipe-vs-existing-*user*-file (non-destructive
  toward the consumer's own files); OQ-9 is recipe-vs-recipe ACROSS packs (a config-time
  ambiguity the applier must arbitrate before any file is written). Cross-reference OQ-4 but
  they resolve separately. Lean: fail loudly at config time (honors loud-≠-blocking; silent
  first/last-wins would make which pack you installed first load-bearing), but the founder
  decides.

### Non-forks (recorded, not open)

- **Recipe removal / uninstall.** When a pack is removed, its scaffolded files STAY — they
  became the consumer's own code the moment they landed (the one-shot-bootstrap lean of OQ-8).
  Removal does not touch them. Recorded as a settled default rather than an OQ; if OQ-8
  resolves toward a managed-file relationship instead, revisit this.

## Spec Seeds

Provisional — these firm up once the OQs resolve (especially OQ-1/OQ-2, which shape the
first two). Recorded now so the decomposition is visible; not yet load-bearing.

- **Recipe apply mechanism (core)** — the generic copy-template→declared-path applier
  (DD-1/DD-2), non-destructive conflict handling (DD-4/OQ-4), selector resolution
  (OQ-5). Contains ZERO language/platform literals (DD-3); this seed is what `backstop/self`
  guards. The piece BUNDLE-003 `backstop init` is blocked on.

- **Manifest recipe declaration (schema)** — how a pack declares recipes in `pack.yml`
  (OQ-2), validated by pack-manifest validation, distinct from `content.scaffolds`.

- **CI recipe pack (backstop-packs, first consumer)** — the cross-cutting pack holding
  per-platform gate-workflow templates as data (DD-5). The forcing function / proof that the
  mechanism needs no core branch. Lives in backstop-packs, not core.

## Notes / Ideas

- **The CI pack is the acceptance test for the invariant.** If wiring GitHub-vs-GitLab CI
  requires a branch in core, DD-3 has been violated. The CI recipe pack existing and working
  packs-only IS the evidence that the mechanism is genuinely thin.
- **Relationship to `content.scaffolds` and `pack scaffold`** — see the naming-collision
  hazard in Current Thinking. Resolving OQ-2 should explicitly state how the new declaration
  coexists with these without overloading "scaffold" further.

## Version History

- 0.1.0 (2026-07-14): Initial bundle at `exploring`, spun out of BUNDLE-003 (init) during the
  2026-07-14 OQ-resolution session as the BLOCKING pack-recipe dependency init consumes but
  cannot build without (BUNDLE-003 DD-12). Captured the settled frame as five Draft Design
  Decisions inherited from the init session (recipe = template + self-declared path; generic
  copy-to-path apply; hard invariant that all language/platform knowledge stays in pack data;
  non-destructive application; CI recipe pack as first consumer). Posed seven genuine open
  questions on the unresolved MECHANISM (recipe format, manifest declaration, invocation under
  omakase, conflict handling, multiplicity/addressing, whether CI is a distinct kind,
  templating engine and its baking risk). Surfaced the "scaffold" naming-collision hazard
  against existing `content.scaffolds` and `pack scaffold`/`artifact new`. Three provisional
  spec seeds (core apply mechanism, manifest declaration schema, CI recipe pack). No OQs
  pre-resolved; no self-promotion — the founder drives both.
- 0.2.0 (2026-07-14): Bundle-reviewer pass (nits only). Captured the two genuinely-missing
  design forks it flagged as OQ-8 (recipe lifecycle on pack upgrade — one-shot bootstrap vs. a
  managed update/re-apply path, and how that squares with DD-4) and OQ-9 (cross-pack
  recipe-vs-recipe target collision, distinct from OQ-4's recipe-vs-user-file case). Recorded
  recipe removal/uninstall as a settled non-fork (scaffolded files become consumer-owned;
  removal doesn't touch them). Swept two cosmetic nits: References now cites BUNDLE-003 OQ-6
  for converge-never-clobber (was mis-grouped under DD-11), and OQ-3 no longer says "scaffold
  recipe" (re-overloaded the very word the naming-collision hazard warns about). OQ count now
  9; maturity unchanged (exploring) — new OQs left open, founder drives resolution and promotion.

## References

- **BUNDLE-003 (onboarding / `backstop init`)** — the consumer and origin. DD-12 (packs
  carry scaffolding recipes), DD-13 (hard thin-executor invariant, inherited here as DD-3),
  OQ-6 (converge-never-clobber), DD-14 (ecosystem-scaffolder composition), OQ-7 (CI wired via
  a recipe pack). Records this capability as a BLOCKING dependency in its Out of Scope /
  Dependencies note.
- **backstop/self pack** — enforces the zero-baked-language boundary the apply mechanism must
  respect (DD-3).
- **Pack manifest** (`pkg/pack/manifest.go`) — existing `Content.Scaffolds` (rule test
  scaffolds) and the shape a `recipes:` declaration (OQ-2) would extend or sit beside.
- **`pkg/pack/scaffold.go` / `backstop artifact new`** — the pack-authoring scaffolder
  (`engine`/`mechanism`/`toolchain`); the other current meaning of "scaffold" to stay clear of.
- **BUNDLE-001 / BUNDLE-002 (pack distribution / publishing)** — how packs (including the CI
  recipe pack) are distributed and installed; recipes ride along inside a pack's content.
