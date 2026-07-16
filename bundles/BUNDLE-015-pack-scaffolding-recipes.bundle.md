---
title: "Pack Scaffolding Recipes"
number: BUNDLE-015
created: "2026-07-14"
schema_version: bundle/v2

bundle:
  name: pack-scaffolding-recipes
  version: "0.5.0"
  created: "2026-07-14"
  updated: "2026-07-16"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Backstop is a thin executor with a hard invariant: detection, framework/language,
    and CI-platform knowledge live in PACKS as data, NEVER in core/CLI (backstop/self
    enforces it). But onboarding needs to SCAFFOLD things that are inherently
    language- and platform-specific — a starter TS or Go project skeleton, a
    `.github/workflows/backstop.yml` gate workflow, a canonical HTTP client, and so on.
    The only way to do that without baking knowledge into core is for PACKS to carry the
    recipes (data) and for a GENERIC core mechanism to apply them. That capability does
    not exist yet, and it is a BLOCKING dependency for `backstop init` (BUNDLE-003 DD-12):
    init consumes recipes but cannot be built until they exist. This bundle defines what
    a recipe IS, how a pack declares and versions one, and how init / `pack add` apply
    one — a generic, declarative mechanism whose content, target, behavior, and paired
    enforcement are owned entirely by the pack.

  user_story: >
    As a pack author, I want to ship a starter project skeleton, a CI gate-workflow, and
    canonical implementations inside my pack so that a consumer running `backstop init`
    (or `backstop pack add <me>`, or a spec/plan that references my recipe) gets those
    materialized in their repo — without backstop core containing a single line that knows
    my language or CI platform, and with a paired enforcement suite that keeps the output
    in bounds afterward. As the consumer, I want application to be deterministic where the
    target is conventional, judgment-mediated where it is bespoke, non-destructive toward
    my own files, and fully traceable (which recipe at which version wrote what).

solution:
  approach: >
    Define a pack-recipe capability whose ONLY genuinely new primitive is GENERATION;
    everything that makes a recipe *backstop* reuses existing substrate (the engines, the
    gate, the ratchet, the waiver, the spec/plan pipeline, worktrees, a provenance ledger).
    A recipe is a first-class VERSIONED artifact shipped by a pack and referenced as
    `<pack>:<recipe>@<recipe_version>`. Its body is a manifest of declarative OPERATIONS
    from a small language-neutral set — `create` (templated file/tree), `merge` (deep-merge
    a fragment into a structured file), `transform` (declarative AST rewrite run by an
    allowlisted general engine), `insert` (snippet at a declared anchor) — with declarative
    `{{ }}`-style substitution that is explicitly NOT Turing-complete. A recipe has a KIND
    (scaffolding / implementing / templating — a spectrum of post-generation ownership) and
    MAY ship a paired enforcement suite (gate rules scoped to its output). It applies in one
    of two modes: DIRECT/deterministic (self-applies its ops identically every run,
    enforcement locks it) or SDLC-MEDIATED (applied through a spec/plan that supplies only
    the WHERE for bespoke injection; the recipe supplies the WHAT + the GUARANTEE). Where
    surgical injection can't reach, enforcement stays red until the manual wiring is correct.
    Divergence is an accountable WAIVER, not a bespoke merge. A recipe may optionally declare a
    COMPAT MATRIX (declared `{file, path, range}` selectors, generic read + semver-compare) — the
    third drift axis — and carry VERSION-KEYED VARIANTS resolved against the consumer's actual
    environment; together with `transform` + enforcement these compose into
    MIGRATION-AS-A-DISTRIBUTABLE-ARTIFACT (author once, distribute, verify completeness). Core
    stays thin: a general engine runs a declared rule (data); language-awareness lives in the
    engine, never in core (DD-3). The remaining open mechanism is chiefly how a pack DECLARES all of this in
    its manifest (OQ-2) and the exact shape of the provenance ledger (OQ-11); the CI recipe
    pack is the first canonical consumer and the invariant's acceptance test.
---

# Pack Scaffolding Recipes

## Current Thinking

### Provenance: spun out of the init bundle, then deepened over three sessions

This bundle was carved out of **BUNDLE-003 (onboarding / `backstop init`)** on 2026-07-14 as
the BLOCKING pack-recipe dependency init consumes but cannot be built without (BUNDLE-003
DD-12). Its model was then deepened across founder-driven design sessions: the 2026-07-14
"two axes + waiver" brain-dump (DD-6/7/8), the 2026-07-16 session that reframed recipes as
declarative OPERATIONS, versioned artifacts, two application modes, and a provenance-backed
concurrency story (DD-9..DD-15), and a later 2026-07-16 session adding the compat matrix,
version-keyed variants, and migration-as-a-distributable-artifact (DD-16/DD-17). The direction
of travel: the model got MORE concrete and the thin-core thesis got STRONGER, not vaguer —
generation is the only new primitive; everything else is existing substrate.

### The thin-executor invariant (still the north star)

All language/platform knowledge lives in recipe DATA, never in core/CLI. A general engine runs
a DECLARED rule; the language-awareness is in the engine, not core. The line is
general-engine-runs-declared-rule (allowed) vs bespoke-procedural-code-encoding-language-
structure (forbidden — that is what an Nx generator IS). `backstop/self` enforces it. This is
the invariant every decision below is measured against (DD-3).

### What is now SETTLED vs OPEN

Settled across the sessions: what a recipe IS (declarative operation-set, DD-9), that `transform`
does not bake language knowledge (enforcement/rewrite symmetry, DD-10), how it's invoked (two
modes, DD-11), how it's cited (`pack:recipe@version`, DD-12), the honest injection limit
(DD-13), that concurrency needs no new machinery (DD-14), that a per-application provenance
ledger is a load-bearing primitive (DD-15), the optional COMPAT MATRIX as the third drift axis
(DD-16), and VERSION-KEYED VARIANTS resolved by environment (DD-17). These last two compose with
`transform` + enforcement into MIGRATION-AS-A-DISTRIBUTABLE-ARTIFACT — the headline strength (see
Notes). Still open: the MANIFEST DECLARATION shape (OQ-2, now the foundational open question, and
now also carrying optional compat + variants), the residual distinct-recipes multiplicity (OQ-5,
largely answered by DD-17 variants), enforcement scoping (OQ-10, anchored to the ledger), the
provenance-ledger's exact form (OQ-11), and the recipe→pack version derivation mechanism (OQ-12).

### Naming-collision hazard (load-bearing for OQ-2)

The word "scaffold" is already overloaded in this codebase and MUST NOT be conflated with this
capability:

- `pkg/pack/manifest.go` `Content.Scaffolds []Scaffold` — a rule's paired TEST scaffold (tier,
  `test_command`, `pairs_with.rules`). Authoring metadata about a rule's fixtures, not a
  consumer-repo template.
- `pkg/pack/scaffold.go` / `backstop artifact new` (`ValidPackTypes = engine / mechanism /
  toolchain`) — scaffolds a NEW PACK's own `pack.yml`. About authoring a pack, not applying a
  pack's contents to a consumer.

Neither is the "pack ships a recipe that init materializes into a consumer repo" concept here.
OQ-2 must pick a name/shape that does not collide. The working term is **recipe**, deliberately
distinct from **scaffold**.

## Draft Design Decisions

DD-1..DD-5 are the frame inherited from the BUNDLE-003 init session. DD-6..DD-8 are the
2026-07-14 "two axes + waiver" model. DD-9..DD-15 are the 2026-07-16 deepening. All are
founder-driven and recorded as decided.

### Inherited frame (BUNDLE-003)

- **DD-1: A recipe declares its OWN target(s).** The recipe names where it installs (e.g.
  `.github/workflows/backstop.yml`); the applier contributes NO path knowledge. Source:
  BUNDLE-003 DD-12.

- **DD-2: The applier is a GENERIC executor with no language/platform branch.** Core resolves a
  recipe and runs its declared operations (DD-9); it never contains a language- or
  platform-aware branch. Selectors index into a pack's recipes; the recipe carries the
  behavior. Source: BUNDLE-003 DD-12 / OQ-7, refined by DD-9.

- **DD-3: HARD INVARIANT — all language/platform knowledge lives in DATA, never in core.** A
  language/framework/platform name in core CLI code is the bug; `backstop/self` enforces it.
  Inherited verbatim from BUNDLE-003 DD-13 — the invariant that makes this capability safe.

- **DD-4: Application is non-destructive toward USER-OWNED files.** A recipe must not clobber a
  consumer's own file (init's converge-never-clobber, BUNDLE-003 OQ-6). For RECIPE-OWNED output
  (scaffolding/implementing kinds) the model is regenerate-by-default with divergence recorded
  as a WAIVER (DD-8) — "never clobber" protects files the recipe does not own; recipe-owned
  output is regenerable and your edits to it are accountable deviations, not silently preserved.

- **DD-5: The CI recipe pack is the first and canonical consumer.** A cross-cutting pack holding
  per-platform gate-workflow recipes (github / gitlab / bitbucket / jenkins) as data. It is a
  spec seed / first consumer here, NOT a separate artifact, and the forcing function that keeps
  the mechanism honest — if wiring CI needs a language/platform branch in core, the design has
  failed. Source: BUNDLE-003 OQ-7.

### Two axes + the waiver (2026-07-14)

- **DD-6: Recipe KIND is first-class — THREE kinds, defined by ownership AFTER generation.**
  (Resolves OQ-6.) **Scaffolding** — writes exact files, recipe-OWNED ("regenerate, don't
  hand-edit"); CI recipes are this kind. **Implementing** — a canonical, blessed implementation
  (HTTP client, event-bus listener); you pass config, the implementation is fixed;
  config-surface-owned. **Templating** — a starter shell / SDK-boilerplate shortcut; one-shot
  drop, then fully yours. A spectrum of post-generation ownership: recipe-owned → config-owned →
  yours. Kind is manifest METADATA, not a different on-disk shape (DD-9).

- **DD-7: Enforcement is an ORTHOGONAL, opt-in axis on ANY kind.** A recipe MAY ship a paired
  enforcement suite — gate rules SCOPED to its generated output. Scaffolding+enforce = locked
  structure; Implementing+enforce = canonical stays canonical; Templating+enforce = "guided
  freedom." Mechanism: pack gate-rules scoped to the recipe's output — no new enforcement
  primitive, it reuses the existing gate. (Declaration/scoping = OQ-10.)

- **DD-8: The WAIVER is the dial between locked and free.** To customize an enforced recipe's
  output, apply a WAIVER via the existing `@waiver:<rule>:<reason>:<expiry>` subsystem. This
  makes the kinds a SPECTRUM, not silos, and is the answer to "your version diverges from the
  recipe": a waiver (accountable deviation with reason + expiry), NOT a bespoke merge/upgrade
  mechanism. Strictly better than raw templating's silent drift.

### The operation-set model + traceability + concurrency (2026-07-16)

- **DD-9: A recipe is a manifest of declarative OPERATIONS.** (Resolves OQ-1.) Not "a template
  directory" — an ordered set of ops from a small, language-neutral set:
  - `create` — drop a templated file (a template DIR is the `create` payload; scales down to
    one file).
  - `merge` — deep-merge a fragment into a STRUCTURED file (package.json / .env / tsconfig /
    json / yaml / toml). Language-neutral because the format is universal.
  - `transform` — a declarative AST rewrite executed by an allowlisted GENERAL engine (ast-grep
    `fix`/rewrite, comby, semgrep autofix).
  - `insert` — a snippet at a declared anchor/marker (the dumb string-op fallback).
  Substitution is DECLARATIVE (a generic `{{ }}`-style convention, explicitly NOT a
  Turing-complete engine). The format is UNIFORM across kinds; kind is manifest metadata (DD-6),
  not a different on-disk shape.

- **DD-10: `transform` does NOT bake language knowledge** (corrects an earlier "no AST edits"
  caveat). AST modification is the IDENTICAL shape as backstop's existing AST *enforcement*: a
  general engine executes a *declared rule* (data); the language-awareness lives in the engine,
  not core. The real line is general-engine-runs-declared-rule (allowed) vs
  bespoke-procedural-code-encoding-language-structure (forbidden — what an Nx generator is).
  Symmetry to state explicitly: **enforcement reads the AST ("match → finding"); a recipe
  rewrites it ("match → fix") — same engines, same model, same thin core.** Recipes therefore
  reuse the ENGINE substrate, not just the gate/waiver.

- **DD-11: Two application MODES.** (Reshapes/resolves OQ-3.) **Mode A — DIRECT/deterministic
  (the grail):** the recipe self-applies its ops, identical every run, enforcement locks it.
  Used wherever the target is conventional — "don't make me think, just do it the same way and
  keep it from drifting." **Mode B — SDLC-MEDIATED:** for bespoke surgical injection, the recipe
  is applied THROUGH a spec/plan. The spec says "implement capability X in file Y using recipe
  Z"; the plan (agent-authored, codebase-aware) pins the exact injection site + params,
  REFERENCING the recipe's template; the implementer applies; the recipe's enforcement verifies.
  Division of labor: **the recipe supplies the WHAT (template) + the GUARANTEE (enforcement);
  the plan supplies only the WHERE (judgment about this codebase).** Same recipe artifact in
  both modes (one source of truth); ceremony matches difficulty, bridging the fast/naive and
  rigorous surfaces onto one recipe.

- **DD-12: Recipes are first-class VERSIONED artifacts, referenced as
  `<pack>:<recipe>@<recipe_version>`.** (Resolves the citation question.) Full-qualification +
  version-pin = complete traceability of the intended injection. The version is the RECIPE'S OWN
  version, not the pack's, because recipe versions are STABLE across unrelated pack churn
  ("checkout hasn't changed in 9 pack releases" is information a pack version can't carry). NOT
  redundant with pack version: pack version = distribution/lock snapshot; recipe version =
  capability/injection contract (precedent: Helm chart `version` vs `appVersion`). Authoring
  cost (two bumps) is tractable via auto-derivation — a recipe rev auto-rolls the pack version,
  changesets-style (mechanism = OQ-12). Enables TWO drift signals: code-diverged-from-recipe
  (enforcement) AND recipe-moved-on (applied @1.2.0 vs current @1.4.0 → "stale, re-apply").
  Feeds the ledger / forensic-replay moat.

- **DD-13: The injection LIMIT, and enforcement backstops the residue.** Surgical `transform`
  works WHERE convention exists to target; where composition is bespoke, NO tool reliably
  auto-injects (Nx / Rails only work by ASSUMING their conventions). The honest three-tier
  answer: (1) conventional → `transform`; (2) bespoke → fail LOUD with the exact one-line manual
  instruction; (3) the recipe's ENFORCEMENT stays RED until the manual wiring is correct. "When
  generation can't reach, enforcement does."

- **DD-14: Concurrency needs NO new machinery.** The sequential guarantee holds whenever
  something sequences (a plan, or the direct-apply order). It breaks only in vibe-coding: loose
  direction + parallel background agents sharing one tree = a real race. Resolved by LAYERING
  EXISTING substrate: ISOLATE parallel agents (worktrees) → no textual corruption, reconciliation
  = git merge; but git resolves TEXTUAL not SEMANTIC conflicts (two agents both register a route
  → merged duplicate) → ENFORCEMENT is the semantic net (the recipe's gate catches the duplicate
  → red → can't ship); the PROVENANCE LEDGER (DD-15) gives the reconciliation record; an optional
  apply-lock is belt-and-suspenders. This is the runtime thesis: the chaotic top is safe because
  the deterministic enforcement floor doesn't move.

- **DD-15: The per-application PROVENANCE LEDGER (primitive).** A record of what a recipe
  application actually DID (files/regions touched, `@version` applied). One primitive, THREE
  payoffs: (a) scoping a dynamic `transform`'s enforcement (OQ-10 — scope = what the application
  actually wrote); (b) `@version` traceability and the recipe-moved-on drift signal (DD-12); (c)
  concurrency reconciliation (DD-14). Its exact form is OQ-11.

### Compat, variants, and migration-as-artifact (2026-07-16)

- **DD-16: A recipe may declare an optional COMPAT MATRIX (the third drift axis).** A recipe MAY
  declare compatibility with external deps as a set of `{file, path, range}` selectors — e.g.
  `{file: package.json, path: dependencies.stripe, range: ">=12 <15"}`. Backstop reads the
  consumer's actual installed version at the declared path and semver-compares to the range;
  out-of-range → FAIL LOUD. Thin-executor-clean: the ecosystem-specific "where the version
  lives" is DECLARED DATA (the selector); core only does generic read-value-at-path + generic
  semver-compare (semver is a universal convention, not baked knowledge). This completes
  ROT-DETECTION across THREE axes: (1) **code diverged from recipe** → the enforcement suite
  (DD-7); (2) **recipe moved on** (applied @version vs current) → version comparison (DD-12);
  (3) **the world moved** (deps drifted past compat) → this. Most relevant to the `implementing`
  kind (SDK integrations); optional/absent for CI, structural, and templating recipes.

- **DD-17: VERSION-KEYED VARIANTS + environment resolution.** A recipe MAY carry multiple
  internal variants, each a `{compat range → ops}` pairing — ONE recipe, internally
  multi-variant. Applying RESOLVES the variant whose compat range matches the consumer's actual
  environment; NO matching variant → FAIL LOUD ("no variant covers stripe@19 — pack hasn't added
  support"). Adding a variant REVS the recipe: `@recipe_version` versions the whole variant SET
  (consistent with DD-12). Resolution is generic — "pick the variant whose declared range matches
  the read version" — no baked knowledge. This is the concrete answer to a chunk of OQ-5: the key
  form of multiplicity is version-keyed variants of ONE logical recipe, addressed by one name,
  resolved by environment. It extends the grail from "same way every time" (DD-11 Mode A) to
  "the RIGHT way for MY environment, automatically."

## Open Questions

Status index (numbers held stable across versions for traceability; resolved/dissolved OQs kept
with their decision so the reasoning survives):

- OQ-1 Recipe format — **RESOLVED** (DD-9, operation-set)
- OQ-2 Manifest declaration — **OPEN** (foundational; now also optional compat + variants)
- OQ-3 Invocation — **RESOLVED** (DD-11, two modes; small ordering residual)
- OQ-4 Conflict (recipe-vs-user-file) — **RESOLVED-via-waiver** (residual: apply-time mechanics)
- OQ-5 Multiplicity & addressing — **RESOLVED-IN-PART** (DD-17 variants = primary form; distinct-recipes shape residual)
- OQ-6 Is CI a distinct kind? — **RESOLVED** (DD-6, CI is scaffolding-kind)
- OQ-7 Templating engine — **RESOLVED** (DD-9/DD-10; residual: exact substitution syntax)
- OQ-8 Lifecycle on upgrade — **RESOLVED-via-waiver**
- OQ-9 Cross-pack collision — **DISSOLVED** (sequential ordered apply + non-destructive)
- OQ-10 Enforcement scoping — **OPEN** (now anchored to the provenance ledger)
- OQ-11 Provenance-ledger shape — **NEW / OPEN**
- OQ-12 Recipe→pack version derivation — **NEW / OPEN**
- Citation / traceability — **RESOLVED** (DD-12, `pack:recipe@version`)

Maturity stays `exploring` — the founder drives the remaining resolutions and triggers
promotion.

### Open

- **OQ-2 — Manifest DECLARATION. (OPEN — now the foundational mechanism question.)** How does a
  pack declare a recipe? The declaration must now carry: the ordered OPS (DD-9), the KIND (DD-6),
  the PARAM schema (placeholders + types/defaults), the TARGET paths (for `create`/`merge`), the
  `transform` RULES, the paired ENFORCEMENT suite (DD-7 / OQ-10), the recipe VERSION (DD-12), and
  — optionally — the COMPAT MATRIX (`{file, path, range}` selectors, DD-16) and VERSION-KEYED
  VARIANTS (`{compat range → ops}` sets, DD-17).
  Options: an explicit `recipes:` block in `pack.yml` vs a convention-based directory the applier
  discovers. Must NOT collide with `content.scaffolds` or `pack scaffold`/`artifact new` (see the
  naming-collision hazard). Lean: explicit block — every one of those facets is data the manifest
  has to validate — but the exact shape is the open work everything else now rests on.

- **OQ-5 — MULTIPLICITY & addressing. (RESOLVED-IN-PART, residual open.)** DD-17 answers the
  PRIMARY multiplicity form: version-keyed VARIANTS of ONE logical recipe (`{compat range → ops}`
  sets), addressed by one name and resolved by the consumer's environment. That handles the
  most-important case (an SDK-integration recipe spanning stripe@12 / @15 / @19) without N
  separately-addressed recipes. **Residual (still open):** the OTHER multiplicity — a single pack
  shipping several DISTINCT logical recipes (a starter recipe AND N per-platform CI recipes AND an
  HTTP-client implementing recipe). Their in-pack SHAPE (one block with N entries? N directories?
  how selectors map to recipe ids) is open and couples to OQ-2. Lean: yes, multiple distinct
  recipes; addressed by stable recipe id within the pack namespace, each internally
  variant-resolved per DD-17.

- **OQ-10 — Enforcement DECLARATION + SCOPING. (OPEN — anchored to the ledger.)** DD-7 says a
  recipe may ship a paired enforcement suite reusing the gate; how is it declared and SCOPED to
  the recipe's OUTPUT rather than repo-wide? For static targets, scope = the recipe's declared
  paths. For a dynamic `transform` (injection site not known until apply), scope = what the
  application actually WROTE — i.e. the provenance ledger (DD-15). Open: (a) manifest declaration
  of which rules pair with the recipe (folds into OQ-2); (b) how scope survives the consumer
  moving/renaming output; (c) whether output-scoped enforcement needs a gate change or is
  expressible with existing path-scoping + the ledger. Lean: declared alongside the recipe;
  static scope from declared paths, dynamic scope from the ledger; confirm the gate can express
  output-scoped rules.

- **OQ-11 — Provenance-ledger SHAPE. (NEW.)** DD-15 decides the ledger EXISTS; its form is open.
  Where does it live (in-repo tracked? `.backstop/` gitignored? both, like baseline)? What does
  each entry record (recipe `pack:recipe@version`, op list, files/regions touched, timestamp,
  applying mode)? How is it READ — for enforcement-scoping (OQ-10), for the recipe-moved-on drift
  signal (DD-12), and for concurrency reconciliation (DD-14)? Is it one ledger per repo or per
  application? Lean: a tracked per-repo ledger of applications (durable traceability), but the
  tracked-vs-gitignored question mirrors the baseline debate and needs the founder.

- **OQ-12 — Recipe→pack version DERIVATION. (NEW, small.)** DD-12 wants a recipe rev to
  auto-roll the containing pack's version (changesets-style) so authors don't hand-bump twice.
  What's the mechanism — a tool/hook at pack-publish time that scans recipe version changes and
  computes the pack bump? Where does it run (pack authoring CLI, CI)? Lean: a publish-time
  derivation in the pack authoring tooling; small and mechanical, but unspecified.

### Resolved / dissolved (kept for the reasoning)

- **OQ-1 — Recipe FORMAT. (RESOLVED → DD-9.)** A recipe is a manifest of declarative operations
  (`create` / `merge` / `transform` / `insert`) with declarative `{{ }}` substitution, uniform
  across kinds. Supersedes the earlier "template directory vs per-file" framing — a template dir
  is simply the `create` op's payload.

- **OQ-3 — INVOCATION. (RESOLVED → DD-11.)** Two application modes: DIRECT/deterministic and
  SDLC-mediated. The omakase-prompt-free tension dissolves — direct mode self-applies from
  recipe-declared defaults/params; bespoke cases go through a spec/plan that supplies the WHERE.
  **Small residual (note, not a fork):** the operation ORDER in direct-apply mode (declaration
  order vs a declared dependency order between ops) — a detail for the apply-mechanism spec.

- **OQ-4 — CONFLICT (recipe-vs-user-file). (RESOLVED-via-waiver.)** Divergence between a
  consumer's file and the recipe is a WAIVER (DD-8), not a bespoke merge; user-owned files are
  never clobbered (DD-4). **Residual:** apply-time mechanics when a recipe-owned target already
  exists on re-apply (overwrite/regenerate/skip/surface) — folds into the apply-mechanism spec.

- **OQ-6 — Is "CI recipe" a distinct KIND? (RESOLVED → DD-6.)** Kind is first-class (three
  kinds); a CI recipe is the SCAFFOLDING kind whose targets land under `.github/` (or the
  platform equivalent). CI is not its own axis.

- **OQ-7 — Templating engine + baking risk. (RESOLVED → DD-9/DD-10.)** Substitution is
  declarative `{{ }}` (not Turing-complete); AST rewrites are `transform` via allowlisted general
  engines running declared rules — the same substrate as AST enforcement, so no baked knowledge
  (DD-10). **Residual (note):** the exact declarative substitution syntax.

- **OQ-8 — Lifecycle on pack upgrade. (RESOLVED-via-waiver.)** No bespoke merge/upgrade
  mechanism: a pack upgrade re-applies recipe-owned output and divergences carry waivers; the
  recipe-moved-on drift signal (applied @1.2.0 vs current @1.4.0, DD-12) surfaces staleness.

- **OQ-9 — Cross-pack target COLLISION. (DISSOLVED.)** Application is SEQUENTIAL and ordered, and
  non-destructive toward user files; two packs touching the same structured file is NORMAL
  composition (that is exactly what `merge` is for), not a conflict to arbitrate. The earlier
  "fail loud on same-path" framing over-indexed on `create`-style whole-file ownership.
  **Correction:** the provenance ledger (DD-15) serves OQ-10 / traceability / concurrency — it is
  NOT the resolution to OQ-9 (an earlier draft conflated them).

- **Citation / traceability. (RESOLVED → DD-12.)** `<pack>:<recipe>@<recipe_version>` — recipe's
  own version, distinct from and not redundant with the pack version.

### Non-forks (recorded, not open)

- **Recipe removal / uninstall.** When a pack is removed, its generated files STAY — they became
  the consumer's own code the moment they landed. Removal does not touch them. (Enforcement tied
  to a removed recipe simply stops applying.)

## Spec Seeds

Provisional until OQ-2 (declaration) and OQ-11 (ledger) firm up, but the decomposition is now
clear.

- **Recipe apply mechanism (core)** — the generic executor: resolve a recipe by
  `pack:recipe@version`, run its declarative OPS (`create` / `merge` / `transform` / `insert`,
  DD-9) with declarative substitution (DD-9/OQ-7), across the two application MODES (direct +
  SDLC-mediated, DD-11), non-destructive toward user files with the regenerate/waiver model for
  recipe-owned output (DD-4/DD-8), writing a per-application PROVENANCE LEDGER entry (DD-15).
  `transform` dispatches to allowlisted general engines running declared rules (DD-10). Contains
  ZERO language/platform literals (DD-3) — this seed is what `backstop/self` guards. The piece
  BUNDLE-003 `backstop init` is blocked on.

- **Manifest recipe declaration (schema)** — how a pack declares a recipe in `pack.yml` (OQ-2):
  ops, kind (DD-6), param schema, target paths, transform rules, paired enforcement (DD-7/OQ-10),
  and recipe version (DD-12). Validated by pack-manifest validation; distinct from
  `content.scaffolds`.

- **Recipe versioning + reference resolution** — the `<pack>:<recipe>@<recipe_version>` scheme
  (DD-12): recipe-own versioning distinct from pack version, reference resolution at apply time,
  the recipe→pack version auto-derivation (OQ-12), and the THREE drift signals — enforcement
  (code-diverged), recipe-moved-on, and compat (world-moved, DD-16) — read from the ledger.

- **Compat + version-keyed variants + migration** — the optional compat matrix (DD-16:
  generic read-value-at-path + semver-compare over declared `{file, path, range}` selectors),
  version-keyed variant resolution (DD-17: pick the variant whose range matches the environment,
  fail loud on no match), and the migration-as-artifact flow they compose into (see the strategic
  note): bump SDK → compat red → re-apply → new variant's `transform` carries the codemod →
  enforcement guarantees completeness. Most relevant to the `implementing` kind.

- **CI recipe pack (backstop-packs, first consumer)** — the cross-cutting pack holding
  per-platform gate-workflow recipes as data (DD-5); the invariant's acceptance test. Lives in
  backstop-packs, not core. Unchanged.

## Notes / Ideas

- **Headline strength — MIGRATIONS become a distributable, versioned, verified ARTIFACT.** The
  compat matrix (DD-16) + version-keyed variants (DD-17) + `transform` (DD-9/DD-10) + enforcement
  (DD-7) compose into a migration mechanism that turns migrations inside out: bump the SDK →
  compat goes RED → re-apply the recipe → its new variant's `transform` ops carry the API codemod
  (old→new) that rewrites existing call sites → enforcement guarantees COMPLETENESS (the gate
  stays red until fully migrated) → provenance records it (DD-15). The leverage: the migration is
  authored ONCE by the pack author (who knows the SDK best) as a variant + transform + enforcement,
  and DISTRIBUTED — every consumer migrates by re-applying, instead of N teams each reinventing the
  same migration. Migration know-how becomes a versioned, reusable, verified artifact
  ("integrate-don't-build / the bundle is the product," applied to the most-duplicated work in
  software). Opt-in/staged per consumer (stay green on the old variant until ready) + fleet
  visibility (who's on old variants = who's red). Honest edge, same three tiers as DD-13:
  `transform` migrates where patternable; bespoke call sites flow through Mode B (spec/plan pins
  them, recipe supplies the codemod, enforcement verifies); enforcement guarantees the whole either
  way. This COMPLETES the rot grail — a recipe doesn't just DETECT drift across three axes, it
  carries the deterministic PATH to fix it; the integrated capability maintains itself across the
  SDK's lifetime with effort centralized once.
- **Three drift axes = complete rot detection (DD-16).** (1) code diverged from recipe →
  enforcement; (2) recipe moved on → `@version` comparison; (3) the world moved (deps drifted) →
  compat matrix. All three read from / feed the provenance ledger.
- **Strengthened thesis — recipes are a new USE of the substrate, not new substrate.**
  GENERATION is the ONLY genuinely new primitive. Everything that makes a recipe *backstop*
  already exists: the ENGINES (`transform` reuses the AST engine substrate — DD-10), the
  gate/ENFORCEMENT (DD-7), the RATCHET, the WAIVER (accountable divergence — DD-8), the
  SPEC/PLAN pipeline (Mode B judgment — DD-11), WORKTREES (concurrency isolation — DD-14), and
  the PROVENANCE LEDGER (DD-15). This got TRUER across the design sessions, not vaguer — the
  "integrate-don't-build / the bundle is the product / the pieces compound" thesis proving
  itself on a concrete feature.
- **Enforcement/rewrite symmetry is the cleanest framing of the whole idea.** Enforcement reads
  the AST (match → finding); a recipe rewrites it (match → fix). Same engines, same declared-rule
  model, same thin core (DD-10). Worth leading with when this bundle is pitched or speced.
- **The CI pack is the acceptance test for the invariant.** If wiring GitHub-vs-GitLab CI needs
  a branch in core, DD-3 is violated. The CI recipe pack working packs-only IS the evidence the
  mechanism is genuinely thin.
- **Relationship to `content.scaffolds` and `pack scaffold`** — see the naming-collision hazard.
  OQ-2 should state explicitly how the new declaration coexists without overloading "scaffold."

## Version History

- 0.1.0 (2026-07-14): Initial bundle at `exploring`, spun out of BUNDLE-003 (init) as the
  BLOCKING pack-recipe dependency (BUNDLE-003 DD-12). Five inherited design decisions, seven open
  questions on the mechanism, the "scaffold" naming-collision hazard, three spec seeds. No OQs
  pre-resolved; no self-promotion.
- 0.2.0 (2026-07-14): Bundle-reviewer pass (nits only). Added OQ-8 (lifecycle on upgrade) and
  OQ-9 (cross-pack collision) as genuine forks; recorded removal/uninstall as a non-fork; swept
  two cosmetic nits (BUNDLE-003 OQ-6 citation; de-overloaded "scaffold" in OQ-3). OQ count 9.
- 0.3.0 (2026-07-14): **Founder brain-dump — two axes + waiver.** DD-6 (recipe KIND first-class,
  three kinds; CI is scaffolding-kind), DD-7 (enforcement orthogonal opt-in axis, reuses the
  gate), DD-8 (waiver is the dial). Resolved OQ-6; resolved-via-waiver OQ-4 and OQ-8; informed
  OQ-1/OQ-3/OQ-7; added OQ-10 (enforcement scoping); enlarged OQ-2; refined DD-4. Added the
  strategic "new use of the substrate" note and prior-art references. OQ count 10.
- 0.4.0 (2026-07-16): **Founder design session — the operation-set model, versioning, modes,
  concurrency.** Added DD-9 (a recipe is a manifest of declarative OPERATIONS —
  `create`/`merge`/`transform`/`insert` with declarative `{{ }}` substitution, uniform across
  kinds — RESOLVING OQ-1), DD-10 (`transform` does NOT bake language knowledge; the
  enforcement/rewrite AST symmetry; recipes reuse the ENGINE substrate — RESOLVING OQ-7 with the
  earlier "no AST edits" caveat corrected), DD-11 (two application MODES: direct/deterministic +
  SDLC-mediated; recipe supplies WHAT+GUARANTEE, plan supplies WHERE — RESOLVING OQ-3), DD-12
  (recipes are first-class VERSIONED artifacts referenced `<pack>:<recipe>@<recipe_version>`;
  recipe-own version distinct from pack version, Helm `version`/`appVersion` precedent, two drift
  signals — RESOLVING the citation question), DD-13 (the injection limit + enforcement backstops
  the residue: conventional→transform, bespoke→fail loud, enforcement stays red till wired),
  DD-14 (concurrency needs no new machinery — worktrees isolate, enforcement is the semantic net,
  ledger reconciles, optional apply-lock), DD-15 (the per-application PROVENANCE LEDGER primitive,
  three payoffs). Refined DD-2 (generic OP executor) to reference DD-9. Reconciled OQs: OQ-1/OQ-3/
  OQ-7 and the citation question RESOLVED; **OQ-9 DISSOLVED** (sequential ordered + non-destructive
  application; co-editing a structured file is normal composition = what `merge` is for) and
  CORRECTED the earlier ledger↔OQ-9 conflation (the ledger serves OQ-10/traceability/concurrency,
  not collision); OQ-2 STILL OPEN and now the foundational mechanism question (must declare ops +
  kind + params + paths + transform rules + enforcement + version); OQ-5 STILL OPEN; OQ-10 STILL
  OPEN and re-anchored to the ledger; added **OQ-11** (provenance-ledger shape) and **OQ-12**
  (recipe→pack version derivation). Rewrote the solution approach and strategic notes around the
  strengthened "generation is the only new primitive; everything else is existing substrate"
  thesis and the enforcement/rewrite symmetry. Updated spec seeds (apply mechanism now covers the
  op-set + two modes + ledger; added a recipe-versioning/reference-resolution seed). Renumbered
  cleanly (DD-1..DD-15; OQ numbers held stable with status labels). Maturity unchanged
  (exploring) — founder drives remaining resolutions and promotion.
- 0.5.0 (2026-07-16): **Founder session — compat matrix, version-keyed variants, migration as a
  distributable artifact.** Added DD-16 (optional COMPAT MATRIX — declared `{file, path, range}`
  selectors; core does generic read-value-at-path + semver-compare; the THIRD drift axis
  completing rot-detection: code-diverged / recipe-moved-on / world-moved) and DD-17 (VERSION-KEYED
  VARIANTS — one recipe carrying `{compat range → ops}` sets, resolved by the consumer's
  environment, fail-loud on no match, adding a variant revs `@recipe_version`). Added the headline
  strategic note: compat + variants + `transform` + enforcement compose into
  MIGRATION-AS-A-DISTRIBUTABLE-ARTIFACT (author the migration once as variant+codemod+enforcement,
  distribute it; every consumer migrates by re-applying; enforcement guarantees completeness;
  opt-in/staged + fleet visibility) — completing the rot grail (detect across three axes AND carry
  the deterministic fix path). Reconciled OQs: **OQ-5 now RESOLVED-IN-PART** — DD-17 version-keyed
  variants are the PRIMARY multiplicity form (residual: the distinct-recipes-per-pack shape);
  **OQ-2 extended** to optionally declare compat + variants. Updated spec seeds (versioning seed
  now carries three drift signals; added a compat+variants+migration seed) and the "settled vs
  open" summary. No OQs pre-resolved by the author; maturity unchanged (exploring) — founder drives
  promotion.

## References

- **BUNDLE-003 (onboarding / `backstop init`)** — the consumer and origin. DD-12 (packs carry
  scaffolding recipes), DD-13 (hard thin-executor invariant, inherited here as DD-3), OQ-6
  (converge-never-clobber), DD-14 (ecosystem-scaffolder composition), OQ-7 (CI wired via a recipe
  pack). Records this capability as a BLOCKING dependency.
- **backstop/self pack** — enforces the zero-baked-language boundary the apply mechanism and
  `transform` engines must respect (DD-3 / DD-10).
- **Gate + AST engines** (`pkg/pack`, semgrep / ast-grep) — the existing engine substrate
  `transform` reuses (DD-10); enforcement (match→finding) and recipe rewrite (match→fix) run the
  same declared-rule model.
- **Waiver subsystem (BUNDLE-013)** — the `@waiver:<rule>:<reason>:<expiry>` machinery DD-8
  reuses as the dial between locked and free. Not new substrate.
- **Pack manifest** (`pkg/pack/manifest.go`) — existing `Content.Scaffolds` (rule test scaffolds)
  and the shape a `recipes:` declaration (OQ-2) would sit beside; naming-collision hazard.
- **`pkg/pack/scaffold.go` / `backstop artifact new`** — the pack-authoring scaffolder; the other
  meaning of "scaffold" to stay clear of.
- **BUNDLE-001 / BUNDLE-002 (pack distribution / publishing)** — how packs (including the CI recipe
  pack) are distributed, installed, and versioned; the recipe→pack version derivation (OQ-12)
  touches publishing.

### Prior art (external)

- **Nx generators** — the maturest "a plugin ships a parameterized scaffolder" model; reference
  for the scaffolding kind. **KEY DIVERGENCE:** an Nx generator is CODE (an executable function);
  a backstop recipe MUST be DECLARATIVE DATA (ops + declared placeholders + declared paths + a
  declared `transform` rule) run by a generic core applier — else the baked-knowledge door
  reopens (DD-3/DD-10). Model = **"Nx generators minus the executable logic, plus a paired
  enforcement suite."**
- **Helm chart `version` vs `appVersion`** — precedent for the recipe-own version being distinct
  from the pack/distribution version (DD-12).
- **Changesets** — precedent for auto-deriving a package (pack) version bump from component
  (recipe) changes (DD-12 / OQ-12).
- **Terraform modules / Spring Boot starters** — reference for the IMPLEMENTING kind (canonical,
  config-surface-owned, parameterized-not-hand-edited).
- **`degit` / GitHub template repos** — reference for the TEMPLATING kind (one-shot shell, then
  yours); backstop's addition is the optional paired enforcement suite (DD-7) turning silent
  drift into "guided freedom."
- **ast-grep `fix` / comby / semgrep autofix** — the allowlisted general engines a `transform`
  op dispatches to (DD-9/DD-10).
- **Codemods / jscodeshift, `nx migrate`, Go `go fix` / `gofmt -r`, OpenRewrite** — prior art for
  distributed, tool-driven API migrations. Backstop's divergence (DD-16/DD-17 + the migration
  note): the migration is a DECLARED variant+transform+enforcement carried by a versioned recipe,
  applied by a generic core and VERIFIED for completeness by the gate — not a bespoke executable
  the consumer runs once with no completeness guarantee.
- **npm/Cargo semver ranges** — the universal convention DD-16 compares against; the range lives
  in recipe data, the comparison is generic in core.
