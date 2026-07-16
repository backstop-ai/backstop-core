---
title: "Pack Scaffolding Recipes"
number: BUNDLE-015
created: "2026-07-14"
schema_version: bundle/v2

bundle:
  name: pack-scaffolding-recipes
  version: "0.3.0"
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

- **DD-4: Application is non-destructive toward USER-OWNED files.** A recipe must not clobber
  a consumer's own file. This aligns with init's converge-never-clobber stance (BUNDLE-003
  OQ-6). REFINED by DD-6/DD-8: for RECIPE-OWNED output (scaffolding/implementing kinds), the
  model is regenerate-by-default with divergence recorded as a WAIVER — so "never clobber"
  protects files the recipe does not own; recipe-owned output is regenerable and your edits
  to it are accountable deviations, not silently preserved. The apply-time regenerate-vs-skip
  behavior is the residual of OQ-4.

- **DD-5: The CI recipe pack is the first and canonical consumer.** A cross-cutting pack
  holding per-platform gate-workflow templates (github / gitlab / bitbucket / jenkins) as
  data, applied via the generic mechanism. It is a spec seed / first consumer here, NOT a
  separate artifact. It is also the forcing function that keeps the mechanism honest — if
  the CI pack needs a language/platform branch in core to work, the design has failed.
  Source: BUNDLE-003 OQ-7.

### The recipe model — two orthogonal axes + the waiver as the dial

Founder-driven, decided in the 2026-07-14 brain-dump. These are the founder's vision recorded
as decided (not pre-resolved by me); they resolve/reframe several OQs below.

- **DD-6 (headline): Recipe KIND is first-class — THREE kinds, defined by ownership AFTER
  generation.** (Resolves OQ-6.)
  - **Scaffolding** — writes the EXACT files, placeholders resolved from passed params, in
    exactly the places the recipe dictates. Recipe-OWNED ("regenerate, don't hand-edit").
    **CI recipes are the scaffolding kind** — CI is NOT its own concept, just this kind (this
    is what resolves OQ-6).
  - **Implementing** — a CANONICAL, blessed implementation of something (an HTTP client, an
    event-bus listener). You pass config values; the implementation itself is fixed.
    CONFIG-SURFACE-owned.
  - **Templating** — a starter SHELL / SDK-boilerplate shortcut. One-shot drop, then fully
    YOURS to customize.
  These form a **spectrum of post-generation ownership**: recipe-owned → config-owned → yours.

- **DD-7: Enforcement is an ORTHOGONAL, opt-in axis on ANY kind.** A recipe MAY ship a paired
  enforcement suite — gate rules SCOPED to its generated output — that keeps the output in
  bounds. It is not tied to any one kind: Scaffolding+enforce = locked structure;
  Implementing+enforce = canonical stays canonical; Templating+enforce = "guided freedom"
  (customize the shell, but declared invariants hold). **Mechanism: just pack gate-rules scoped
  to the recipe's output — no new enforcement primitive; it reuses the existing gate.** (The
  declaration/scoping of that paired suite is new OQ-10.)

- **DD-8: The WAIVER is the dial between locked and free.** To scaffold-then-customize, apply a
  WAIVER via the existing `@waiver:<rule>:<reason>:<expiry>` subsystem. This makes the three
  kinds a SPECTRUM, not silos — the waiver dials any enforced recipe from locked toward free.
  Scaffold + waivers-where-you-diverged = ACCOUNTABLE customization (each deviation recorded
  with reason + expiry), strictly better than raw templating's silent drift. This is the answer
  to "your version diverges from the recipe": a waiver, NOT a bespoke merge/upgrade mechanism
  (reframes OQ-4 and OQ-8).

## Open Questions

The 2026-07-14 founder brain-dump (DD-6/DD-7/DD-8) resolved OQ-6 and reframed OQ-4/OQ-8, and
heavily INFORMED OQ-1/OQ-3/OQ-7 without closing them. Resolved OQs are kept here marked
RESOLVED with decision + rationale (not deleted) so the reasoning survives. Maturity stays
`exploring` — the founder drives the remaining resolutions and triggers promotion.

- **OQ-1 — Recipe FORMAT. (INFORMED, still open.)** What is a recipe on disk? Options: (a) a
  template DIRECTORY copied as a tree; (b) individual declared files; (c) a single template
  file per recipe. And is it PURE STATIC COPY, or is there variable SUBSTITUTION (project name,
  module path, etc.)? New constraint from DD-6: recipes DO take params (placeholders resolved
  from passed values), so pure-static-copy is off the table for the scaffolding/implementing
  kinds — the format must carry DECLARED placeholders. What stays open: directory-tree vs.
  per-file, and whether the KIND changes the on-disk shape (a templating one-shot shell may
  differ from a param-driven scaffolding tree). Lean: per-kind declared template + declared
  placeholder set; exact file layout TBD.

- **OQ-2 — Manifest DECLARATION. (STILL OPEN — now BIGGER.)** How does a pack declare its
  recipes? Options: (a) an explicit `recipes:` block in `pack.yml`; (b) a convention-based
  directory the applier discovers. The declaration must now carry MORE than template → target
  path: per DD-6/DD-7 it must declare the recipe's **kind**, its **param schema** (placeholders
  + types/defaults), its **target paths**, AND its **paired enforcement suite** (OQ-10). Must
  NOT collide with the existing `content.scaffolds` (rule test scaffolds) or `pack
  scaffold`/`artifact new` (pack authoring) — see the naming-collision hazard above. Lean:
  explicit block, since the self-declared path + params + kind + enforcement are all data the
  manifest has to validate.

- **OQ-3 — INVOCATION under omakase. (INFORMED, still open.)** New constraint from DD-6:
  recipes are applied via a scaffold/apply COMMAND that takes PARAMS — so invocation is not a
  bare flag, it's a parameterized apply. What stays open: how a parameterized apply stays
  PROMPT-FREE under init's omakase model (BUNDLE-003 OQ-2) — where do param values come from
  without prompting (config? flags? defaults declared in the recipe)? Does `pack add <lang>`
  auto-apply that pack's recipe, or is applying a separate explicit act? Reconcile with: init
  installs the omakase base prompt-free and you SUBTRACT via flags. Lean: TBD — auto-apply
  (fewer steps) vs. surprise parameterized file creation on `pack add`.

- **OQ-4 — CONFLICT handling. (RESOLVED-via-waiver, residual noted.)** Decision: divergence
  between a consumer's file and the recipe is a WAIVER (DD-8), NOT a bespoke merge. The
  conflict MODEL is settled by per-kind ownership (DD-6) + waiver-as-dial (DD-8): recipe-owned
  output regenerates, and if you edited it you carry an accountable waiver; user-owned files
  are never clobbered (DD-4). **Residual (still open):** the apply-time MECHANICS — when the
  recipe-owned target already exists on a re-apply, does the applier overwrite/regenerate,
  skip, or three-way it, and how is that surfaced? This residual is shared with OQ-8 and folds
  into the apply mechanism spec. Distinct from OQ-9 (recipe-vs-recipe).

- **OQ-5 — MULTIPLICITY & addressing. (STILL OPEN, unchanged.)** Can one pack ship MULTIPLE
  recipes (e.g. a TS toolchain pack shipping a starter-skeleton recipe AND the CI pack shipping
  N per-platform recipes)? If yes, how are recipes ADDRESSED/SELECTED — by id? by a selector
  key the CLI maps to a recipe (`--ci github` → recipe `github`)? Coupled to OQ-2. Lean: yes,
  multiple; addressed by a stable recipe id the selector resolves.

- **OQ-6 — Is "CI recipe" a distinct KIND? (RESOLVED.)** Decision: KIND is first-class, and
  there are THREE (scaffolding / implementing / templating — DD-6). **A CI recipe is the
  SCAFFOLDING kind** — CI is not its own concept, just a scaffolding-kind recipe whose target
  paths land under `.github/` (or the platform equivalent). Rationale: making kind first-class
  captures the real distinction (post-generation ownership) without re-importing platform
  knowledge as a taxonomy; CI collapses into an existing kind rather than needing a bespoke
  one. Supersedes the earlier "just a path, no kind" lean — the founder's model says kind DOES
  matter, but CI is not the axis it matters on.

- **OQ-7 — Templating engine (and its baking risk). (INFORMED, still open.)** DD-6 confirms
  substitution IS needed (params → placeholders), so the question is no longer whether but
  WHAT resolves them. New hard constraint: the substitution must be DECLARATIVE, NOT
  Turing-complete — a declared placeholder set resolved from passed params, never executable
  logic in the recipe (that is the Nx divergence in the References: "generators minus the
  executable function"). Executable templating would reopen the baked-knowledge door (DD-3).
  What stays open: which declarative substitution scheme, and whether the placeholder
  vocabulary is fully pack-declared (it must be — core cannot know what `{{module_path}}`
  means). Lean: minimal declarative placeholder substitution, pack-declared vocabulary, no
  engine that can execute.

- **OQ-8 — Recipe LIFECYCLE on pack upgrade. (RESOLVED-via-waiver.)** Decision: the
  one-shot-vs-managed fork collapses via DD-8. There is NO bespoke merge/upgrade mechanism; a
  pack upgrade re-applies (regenerates) recipe-owned output, and wherever the consumer diverged
  they carry a WAIVER (reason + expiry) — accountable, not silently stale, and not a custom
  three-way merger. Keeping current after upgrade is therefore a regenerate-plus-waiver
  operation on existing machinery, not new substrate. **Residual:** the concrete re-apply
  mechanics are the same residual noted in OQ-4 (overwrite/skip/surface); no separate lifecycle
  mechanism is needed.

- **OQ-9 — Cross-pack target COLLISION (recipe-vs-recipe). (STILL OPEN, unchanged.)** Two packs both declaring a recipe
  at the SAME target path (e.g. both wanting `.github/workflows/backstop.yml`). How does the
  generic applier DETECT and resolve it — fail loudly? first-wins / last-wins? namespaced
  paths? This is DISTINCT from OQ-4: OQ-4 is recipe-vs-existing-*user*-file (non-destructive
  toward the consumer's own files); OQ-9 is recipe-vs-recipe ACROSS packs (a config-time
  ambiguity the applier must arbitrate before any file is written). Cross-reference OQ-4 but
  they resolve separately. Lean: fail loudly at config time (honors loud-≠-blocking; silent
  first/last-wins would make which pack you installed first load-bearing), but the founder
  decides.

- **OQ-10 — Enforcement DECLARATION + SCOPING. (NEW — from DD-7.)** DD-7 says a recipe MAY
  ship a paired enforcement suite reusing the existing gate — but HOW does a recipe declare it,
  and how is it SCOPED? A normal pack's gate rules apply repo-wide; recipe-paired enforcement
  must be scoped to the recipe's GENERATED OUTPUT (the files it wrote), not the whole repo —
  otherwise a scaffolding recipe's "keep this structure" rule would police unrelated code.
  Open: (a) how the manifest declares which rules pair with which recipe (folds into OQ-2);
  (b) what defines the enforcement SCOPE — the recipe's declared target paths? a recorded
  manifest of generated files? — and how that scope survives the consumer moving/renaming the
  output; (c) whether output-scoped enforcement needs any gate change or is expressible with
  existing path-scoping. Lean: scope to the recipe's declared target paths, declared alongside
  the recipe in the manifest, reusing the existing gate's path scoping — but confirm the gate
  can already express output-scoped rules.

### Non-forks (recorded, not open)

- **Recipe removal / uninstall.** When a pack is removed, its scaffolded files STAY — they
  became the consumer's own code the moment they landed (the one-shot-bootstrap lean of OQ-8).
  Removal does not touch them. Recorded as a settled default rather than an OQ; if OQ-8
  resolves toward a managed-file relationship instead, revisit this.

## Spec Seeds

Provisional — these firm up once the OQs resolve (especially OQ-1/OQ-2, which shape the
first two). Recorded now so the decomposition is visible; not yet load-bearing.

- **Recipe apply mechanism (core)** — the generic applier: resolves a recipe by id/selector,
  substitutes DECLARED placeholders from passed params (DD-6, declarative-only per OQ-7),
  writes to recipe-declared paths (DD-1/DD-2), handles the per-kind ownership + regenerate/skip
  residual (DD-4/DD-6/OQ-4/OQ-8). Contains ZERO language/platform literals (DD-3); this seed is
  what `backstop/self` guards. The piece BUNDLE-003 `backstop init` is blocked on. GENERATION
  is the only genuinely new primitive here — enforcement/ratchet/waiver are existing machinery
  (see the strategic note below).

- **Manifest recipe declaration (schema)** — how a pack declares recipes in `pack.yml`
  (OQ-2): kind (DD-6), param schema, target paths, and paired enforcement (DD-7/OQ-10).
  Validated by pack-manifest validation, distinct from `content.scaffolds`.

- **CI recipe pack (backstop-packs, first consumer)** — the cross-cutting pack holding
  per-platform gate-workflow templates as data (DD-5). The forcing function / proof that the
  mechanism needs no core branch. Lives in backstop-packs, not core.

## Notes / Ideas

- **Strategic strength — recipes are a new USE of the substrate, not new substrate.**
  GENERATION is the ONLY new primitive recipes require. Everything that keeps generated output
  honest already exists: the ENFORCEMENT is the gate (DD-7), the RATCHET keeps output from
  rotting, and the WAIVER is the accountable-deviation dial (DD-8) — some of it shipped this
  very session. Recipes compound existing machinery rather than adding a parallel stack. This
  is the "integrate-don't-build / the bundle is the product / the pieces compound" thesis
  proving itself on a concrete feature.
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
- 0.3.0 (2026-07-14): **Founder brain-dump — the core recipe model.** Recorded the founder's
  vision (their own resolutions, not pre-resolved by the author) as three design decisions:
  DD-6 (recipe KIND is first-class — THREE kinds by post-generation ownership: scaffolding →
  implementing → templating; CI is the scaffolding kind), DD-7 (enforcement is an ORTHOGONAL
  opt-in axis on any kind, mechanized as pack gate-rules SCOPED to the recipe's output — no new
  primitive, reuses the gate), DD-8 (the WAIVER is the dial between locked and free, making the
  kinds a spectrum; divergence = accountable waiver, not bespoke merge). Reconciled the OQ
  state: **OQ-6 RESOLVED** (kind first-class, CI is scaffolding-kind); **OQ-4 and OQ-8
  RESOLVED-via-waiver** (divergence/upgrade = regenerate + waiver, with a shared residual on
  apply-time regenerate/skip mechanics folded into the apply-mechanism seed); **OQ-1, OQ-3,
  OQ-7 INFORMED but still open** with new constraints recorded (declarative per-kind
  placeholder substitution; parameterized apply command; declarative-not-Turing-complete to
  preserve DD-3). Added **new OQ-10** (how a recipe declares + output-scopes its paired
  enforcement). Enlarged **OQ-2** (declaration must now carry kind + param schema + target
  paths + paired enforcement). **OQ-5 and OQ-9 unchanged.** Refined DD-4 (non-destructive
  protects USER-owned files; recipe-owned output regenerates with divergence tracked by
  waiver). Added a strategic note (recipes = a new USE of the substrate, generation the only new
  primitive — the compounding thesis proving itself) and prior-art references (Nx generators
  minus executable logic + paired enforcement; Terraform/Spring Boot starters ≈ implementing;
  degit/template repos ≈ templating). OQ count now 10 (6 open/informed-open, 3 resolved, 1 new);
  maturity unchanged (exploring) — founder drives remaining resolutions and promotion.

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
- **Waiver subsystem (BUNDLE-013)** — the existing `@waiver:<rule>:<reason>:<expiry>`
  machinery that DD-8 reuses as the dial between locked and free. Not new substrate.

### Prior art (external)

- **Nx generators** — the maturest "a plugin ships a parameterized scaffolder" model; the
  reference for the SCAFFOLDING kind (DD-6). **KEY DIVERGENCE to preserve:** an Nx generator
  is CODE (an executable function); a backstop recipe MUST be DECLARATIVE DATA (template +
  declared placeholders + declared target paths) processed by a GENERIC core applier — else the
  baked-knowledge door reopens (DD-3, OQ-7). The model is **"Nx generators minus the executable
  logic, plus a paired enforcement suite."**
- **Terraform modules / Spring Boot starters** — reference for the IMPLEMENTING kind (a
  canonical, config-surface-owned implementation you parameterize but don't hand-edit).
- **`degit` / GitHub template repos** — reference for the TEMPLATING kind (one-shot shell drop,
  then fully yours). Backstop's addition over these: an optional paired enforcement suite (DD-7)
  turns raw templating's silent drift into "guided freedom."
