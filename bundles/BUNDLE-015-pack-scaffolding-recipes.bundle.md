---
title: "Pack Scaffolding Recipes"
number: BUNDLE-015
created: "2026-07-14"
schema_version: bundle/v2

bundle:
  name: pack-scaffolding-recipes
  version: "0.9.0"
  created: "2026-07-14"
  updated: "2026-07-20"
  category: feature

status:
  maturity: defined

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
    engine, never in core (DD-3). The 2026-07-17 founder session resolved the last open mechanisms:
    a recipe is declared as its OWN DIRECTORY indexed from `pack.yml` (DD-18), enforcement is
    ADOPTION-GATED with static path scope (DD-19), DD-15's ledger SPLITS into a thin tracked
    ADOPTION RECORD here and a rich provenance ledger spun out to BUNDLE-017 (DD-20), and a
    publish-time REV-GUARD keeps recipe versions trustworthy (DD-21). The dynamic-`transform`
    output scoping and the rich forensic/fleet ledger are the downstream dependency owned by
    BUNDLE-017. The CI recipe pack is the first canonical consumer and the invariant's acceptance
    test — and ships on the thin adoption record alone. The 2026-07-20 founder session, driven by
    the arrival of the real-world standup capture (the bclabs-portal go-live fixture), extended the
    model to full-system STANDUP: recipes decompose into PROVIDER packs (supabase / vercel / nextjs /
    the CI pack), each on its own version clock (DD-22); a recipe may declare an EXECUTABLE RUNBOOK
    of ordered platform-state STEPS with paired verify-checks ("a checklist with receipts"), each
    step carrying an execution MODE (auto / assisted / consent / decision) whose EFFECTIVE value is
    COMPUTED against current standing state via precondition receipts — the bootstrap ladder (DD-23 /
    DD-24). The surface is DIAGNOSE-FIRST: a silent probe pass renders only unmet steps with their
    fix attached, escalating automation API > CLI > deep-link > click-instruction, where receipts are
    law and click-instructions are best-effort vibes (DD-25). A standup recipe treats the
    COMPOSITION/framework-wiring step as first-class — provisioned infra without it is a vacuous
    standup (DD-26). Core still only RENDERS/sequences declared steps and runs receipts; it NEVER
    performs a platform operation itself (thin executor holds). CARVE-OUT (2026-07-20 v0.8.0): the
    founder then GENERALIZED runbooks beyond standup (ops + maintenance genres), so the runbook
    CAPABILITY — DD-23..DD-26 — was carved out to BUNDLE-019 (Runbooks), recorded there as
    DD-1..DD-4; lineage stubs remain here. BUNDLE-015 retains provider-pack decomposition (DD-22),
    the op schema + apply sequencing the `step` op rides on, and the provider packs' FILE recipes,
    and RETURNS to its core job: the file-op recipe capability `backstop init` is blocked on.

requirements:
  # --- Seed 1: Recipe apply mechanism (core) ---
  - id: REQ-001
    version: "1.0.0"
    text: >
      Core must provide a GENERIC applier that resolves a recipe by
      `<pack>:<recipe>@<recipe_version>` and executes its manifest's declared, ORDERED
      operations in sequence. The applier carries no path or target knowledge — the recipe
      declares its own target(s); the applier only runs what the recipe declares.
      (DD-1, DD-2, DD-9.)
  - id: REQ-002
    version: "1.0.0"
    text: >
      The applier must support four language-neutral OP families: `create` (drop a templated
      file/tree), `merge` (deep-merge a fragment into a structured file — json/yaml/toml/.env),
      `transform` (a declarative AST rewrite dispatched to an allowlisted GENERAL engine running
      a declared rule), and `insert` (a snippet at a declared anchor). Value substitution is a
      declarative `{{ }}`-style convention that is explicitly NOT Turing-complete. (DD-9,
      DD-10; OQ-1/OQ-7.)
  - id: REQ-003
    version: "1.0.0"
    text: >
      The applier must support two application MODES from the SAME recipe artifact: DIRECT
      (self-applies its ops identically every run from recipe-declared defaults/params) and
      SDLC-MEDIATED (applied through a spec/plan that supplies only the WHERE for bespoke
      injection while the recipe supplies the WHAT + the enforcement GUARANTEE). (DD-11; OQ-3.)
  - id: REQ-004
    version: "1.0.0"
    text: >
      Application must be non-destructive toward USER-OWNED files (never clobber a consumer's
      own file). For RECIPE-OWNED output the model is regenerate-by-default, with any divergence
      recorded as an accountable WAIVER via the existing `@waiver:<rule>:<reason>:<expiry>`
      subsystem — never a bespoke merge. (DD-4, DD-8; OQ-4/OQ-8.)
  - id: REQ-005
    version: "1.0.0"
    text: >
      Each application must write a MINIMAL, tracked ADOPTION RECORD entry —
      `{recipe ref, @version, adopted}`, the recipe analog of `backstop.lock` — which is the
      enforcement-activation primitive and carries the applied `@version` for the
      recipe-moved-on drift signal. It must NOT emit the rich per-op/per-region provenance
      ledger (owned by BUNDLE-017). (DD-15, DD-20.)
  - id: REQ-006
    version: "1.0.0"
    text: >
      The applier and its `transform`-engine dispatch must contain ZERO language/platform/CI
      literals — they resolve and run declared DATA only, with language-awareness living in the
      allowlisted engine, never in core. `backstop/self` guards this seam. (DD-3, DD-10.)
  - id: REQ-007
    version: "1.0.0"
    text: >
      The apply sequencing must RESERVE a `step` op slot as a fifth op family so file ops and
      runbook steps can interleave in one ordered apply. This bundle owns only the op-schema /
      sequencing SEAM the step op rides on; the step EXECUTOR and probe-receipt/precondition
      engine are out of scope here (owned by BUNDLE-019). (DD-22; DD-23 lineage stub.)

  # --- Seed 2: Manifest recipe declaration (schema) ---
  - id: REQ-008
    version: "1.0.0"
    text: >
      A recipe must be declared as its OWN DIRECTORY colocating a `recipe.yml` manifest, its
      template payload, and its `transform`-rule files. `pack.yml` must carry a lightweight
      `recipes:` INDEX mapping a stable recipe-id → directory. Multiple distinct recipes in one
      pack are multiple directories, each addressed by a stable id in the pack namespace. The
      `recipes:` index must not collide with `content.scaffolds` or with `pack scaffold` /
      `artifact new`. (DD-18; OQ-2/OQ-5.)
  - id: REQ-009
    version: "1.0.0"
    text: >
      The `recipe.yml` manifest must declare the ordered ops, the KIND
      (scaffolding / implementing / templating), the param schema, the target paths, the
      `transform` rules, the paired enforcement suite, the recipe VERSION, and — optionally —
      the compat matrix and version-keyed variants. Pack-manifest validation must validate the
      manifest's structure. (DD-6, DD-18.)

  # --- Seed 3: Recipe versioning + reference resolution ---
  - id: REQ-010
    version: "1.0.0"
    text: >
      A recipe must carry its OWN semver version, distinct from and not redundant with the pack
      version (distribution snapshot vs capability/injection contract). References must resolve
      `<pack>:<recipe>@<recipe_version>` at apply time to the pinned recipe version. (DD-12.)
  - id: REQ-011
    version: "1.0.0"
    text: >
      A publish-time REV-GUARD in the pack-authoring tooling (OUTSIDE core) must FAIL when a
      recipe's content changed but its `@version` did not, forcing a rev so the recipe-moved-on
      drift signal stays trustworthy. The guard enforces THAT a rev happens; the semver LEVEL is
      the author's judgment and is never policed. (DD-21; OQ-12.)
  - id: REQ-012
    version: "1.0.0"
    text: >
      The capability must surface THREE independent drift signals: code-diverged-from-recipe
      (the paired enforcement suite), recipe-moved-on (the adoption record's applied `@version`
      vs the current recipe version), and world-moved (the compat matrix). (DD-12, DD-16, DD-20.)

  # --- Seed 4: Recipe enforcement scoping ---
  - id: REQ-013
    version: "1.0.0"
    text: >
      A recipe's paired enforcement must be ADOPTION-GATED — active only for recipes present in
      the adoption record — which, combined with enforcement being opt-in per recipe, is a DOUBLE
      opt-in before anything can go red. A recipe an installed pack merely SHIPS but the consumer
      never applied must be INERT (never red). (DD-7, DD-19.)
  - id: REQ-014
    version: "1.0.0"
    text: >
      For an adopted recipe, enforcement STATIC scope must be the recipe's declared target paths
      (read from the ADOPTED `@version`'s manifest) expressed via the gate's EXISTING per-rule
      path-scoping — no new gate primitive. Gate outcomes: declared output present/valid → GREEN;
      partial or misplaced output → RED (broken promise, naming the absent paths); total absence
      of all declared files → WARN + un-adopt guidance. (Dynamic-`transform` output scoping is
      deferred to BUNDLE-017.) (DD-19.)

  # --- Seed 5: Compat + version-keyed variants + migration ---
  - id: REQ-015
    version: "1.0.0"
    text: >
      A recipe MAY declare an optional COMPAT MATRIX as a set of `{file, path, range}` selectors.
      Core must read the consumer's actual installed version at the declared path and
      semver-compare it to the declared range, FAILING LOUD when out of range. Core does only
      generic read-value-at-path + generic semver-compare; the ecosystem-specific "where the
      version lives" is declared data. (DD-16.)
  - id: REQ-016
    version: "1.0.0"
    text: >
      A recipe MAY carry VERSION-KEYED VARIANTS — internal `{compat range → ops}` pairings of ONE
      logical recipe. Apply must RESOLVE the variant whose compat range matches the consumer's
      actual environment and FAIL LOUD when no variant matches. Adding a variant REVS the recipe
      version (versions the whole variant set). Resolution is generic — pick the variant whose
      declared range matches the read version. (DD-17.)
  - id: REQ-017
    version: "1.0.0"
    text: >
      Compat matrix (REQ-015) + version-keyed variants (REQ-016) + `transform` (REQ-002) +
      enforcement (REQ-013/REQ-014) must compose into MIGRATION-AS-A-DISTRIBUTABLE-ARTIFACT: a
      dependency bump drives compat RED → re-apply resolves the new variant whose `transform`
      carries the codemod → enforcement guarantees completeness (stays red until fully migrated).
      Authored once by the pack author, distributed, and verified for completeness. (DD-16, DD-17;
      migration note.)

  # --- Seed 6: CI recipe pack + provider standup packs (first consumers / acceptance test) ---
  - id: REQ-018
    version: "1.0.0"
    text: >
      A CI recipe pack (in backstop-packs, NOT core) must ship per-platform gate-workflow recipes
      (github / gitlab / bitbucket / jenkins) as DATA of the scaffolding kind, and must gate a
      project packs-only — with NO baked language/platform branch in core. It is the invariant's
      acceptance test and ships on the thin adoption record alone. (DD-5, DD-20.)
  - id: REQ-019
    version: "1.0.0"
    text: >
      The provider standup packs — `supabase`, `vercel`, `nextjs` — must ship their FILE recipes
      (e.g. `vercel.json`, shell scripts, workflows, migrations, app shells) as data, each pack on
      its own independent version clock so per-provider compat + variants apply. Their RUNBOOK
      fragments are out of scope here (owned by BUNDLE-019). Sourced from the bclabs-portal
      capture; no finer splits until a second consumer demands them. (DD-22.)
---

# Pack Scaffolding Recipes

## Current Thinking

### Provenance: spun out of the init bundle, then deepened over three sessions

This bundle was carved out of **BUNDLE-003 (onboarding / `backstop init`)** on 2026-07-14 as
the BLOCKING pack-recipe dependency init consumes but cannot be built without (BUNDLE-003
DD-12). Its model was then deepened across founder-driven design sessions: the 2026-07-14
"two axes + waiver" brain-dump (DD-6/7/8), the 2026-07-16 session that reframed recipes as
declarative OPERATIONS, versioned artifacts, two application modes, and a provenance-backed
concurrency story (DD-9..DD-15), a later 2026-07-16 session adding the compat matrix,
version-keyed variants, and migration-as-a-distributable-artifact (DD-16/DD-17), the
2026-07-17 session that RESOLVED the last four open questions — the per-recipe-directory
declaration shape, adoption-gated enforcement, the adoption-record/provenance-ledger split,
and the version rev-guard (DD-18..DD-21) — and the 2026-07-20 session (triggered by the arrival of
the bclabs-portal standup capture) that extended the model to full-system STANDUP: provider-pack
decomposition, the executable runbook, the bootstrap ladder, the diagnose-first UX, and the
composition step (DD-22..DD-26). The direction of travel: the model got MORE concrete
and the thin-core thesis got STRONGER, not vaguer — generation is the only new primitive;
everything else is existing substrate.

### Carve-out: the runbook capability → BUNDLE-019 (2026-07-20)

Shortly after the standup session below recorded DD-23..DD-26, the founder GENERALIZED: runbooks
are not a standup-only device — standup is just the first and rarest genre, "often less frequent
than ops runbooks" (credential rotation, cert renewal, incident diagnostics, backup verification,
access reviews) and scheduled maintenance sweeps. So the runbook CAPABILITY was CARVED OUT to
**BUNDLE-019 (Runbooks)** — the same carve pattern by which this bundle was spun from BUNDLE-003
and BUNDLE-017 was spun from here. DD-23..DD-26 are MIGRATED there (lineage stubs remain below,
so the record that they were decided in this session survives); DD-22 (provider-pack
decomposition) STAYS, as it governs recipe/pack decomposition generally. With runbooks carved
away, BUNDLE-015 returns to its core job: the FILE-op recipe capability `backstop init` is
blocked on (create / merge / transform / insert + variants + enforcement). The step op is still
declared in this bundle's manifest and rides this bundle's apply-sequencing seam (DD-22, the
apply-mechanism seed); only the step EXECUTOR / probe-receipt engine and the runbook surface move
to BUNDLE-019.

### THE CAPTURE ARRIVED (2026-07-20)

The capture-first decision (2026-07-18) delivered. **"Portal Go-Live: Physical Scaffolding &
Admin/Setup Reference"** — a full standup capture of bclabs-portal on Supabase + Vercel + GitHub +
PostHog, explicitly authored as the real-life recipe fixture (its §6 splits fixture-data vs
generalizable patterns). It lives at
`~/.claude/uploads/3826af56-28a4-484f-975c-492a561591c4/eaeae70d-golivescaffolding.md` (persistent;
slices move into the born packs' `fixtures/captured/` per the captured-fixtures convention). It was
CROSS-CHECKED against the portal repo, with corrections:

- **(a) The code gap is the DEPLOYMENT SUBSTRATE, not just a composition root.** `package.json` has
  no `next` / `react` / `@supabase/supabase-js` at all — route handlers exist in Next App Router
  shape but the framework is uninstalled, with no app shell / build scripts.
- **(b) The 4 `POSTHOG_*` env vars are ASPIRATIONAL** — nothing reads them (config for the missing
  composition layer), so env surfaces must be DERIVED POST-COMPOSITION, not from a static scan.
- **(c) `PORTAL_OIDC_SIGNING_KEY` is dead** (the removed HMAC path) — drop it.
- **(d) `PORTAL_REPO_PATH` is actively wired** (cwd fallback); its vestigiality is unresolved.

Empirics from the capture: ~**9/11** standup steps are scriptable via the `supabase` / `vercel` /
`gh` CLIs + PostHog REST; the irreducible human core is **token supply + naming decisions +
explicit-go on real-money provisioning**; the §5 gotchas (Supavisor transaction-pooler + `prepare=false`,
`storage.objects` ownership, OIDC audience parity, prod/preview split) are verify-check candidates
nearly verbatim. **Platform-state dominance CONFIRMED:** a standup is almost entirely external
platform state, with few file ops — which is what forces the runbook/step-op model below (DD-23) and
the provider-pack decomposition (DD-22). This capture is the concrete forcing function for DD-22..DD-26.

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
Notes).

As of the 2026-07-17 founder session, ALL open questions are now resolved: the MANIFEST
DECLARATION shape is a per-recipe DIRECTORY indexed from `pack.yml` (OQ-2 → DD-18), the residual
distinct-recipes multiplicity is those directories addressed by stable id (OQ-5 → DD-17 variants +
DD-18), enforcement is ADOPTION-GATED with static path scope (OQ-10 → DD-19), DD-15's ledger
SPLITS into a thin tracked ADOPTION RECORD kept here and a rich provenance ledger (OQ-11 →
DD-20), and a publish-time REV-GUARD keeps recipe versions trustworthy (OQ-12 → DD-21). TWO
downstream pieces are DELEGATED to **BUNDLE-017 (recipe provenance ledger)**, not left open here:
the dynamic-`transform` output scoping (the OQ-10 dynamic half) and the rich forensic/fleet
provenance ledger (the OQ-11 rich half). Maturity stays `exploring` — the founder triggers
promotion separately.

The 2026-07-20 founder session (triggered by the arrived capture) EXTENDED the model to full-system
standup and recorded DD-22..DD-26 as DECIDED — provider-pack decomposition (DD-22), the executable
runbook of step-ops with modes and receipts (DD-23), the bootstrap ladder of preconditions and
computed effective mode (DD-24), the diagnose-first UX + automation escalation ladder (DD-25), and
the composition/framework step as first-class (DD-26). These do NOT reopen any resolved OQ; the open
mechanism details they surface (step-op as a fifth op family, exact step shape, precondition receipt
form) are recorded as SPEC-TIME RESIDUALS, not new OQs. Maturity still stays `exploring` — the
founder's walkthrough of the standup model precedes promotion.

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

## Draft Requirements

The authoritative, machine-readable requirements live in the `requirements:` frontmatter array
(REQ-001…REQ-019). They FORMALIZE the slimmed 6-seed file-op scope resolved across the
2026-07-14 → 2026-07-20 founder sessions — every REQ is derived from a decided DD / resolved OQ,
none invents new scope. They map one-to-one onto the Spec Seeds and feed the
`requirement_traceability` gate (delivered ⇒ every REQ covered by an implemented spec). The
runbook-capability requirements (step executor, probe/receipt engine, bootstrap ladder,
diagnose-first surface) are deliberately ABSENT — they were carved out to BUNDLE-019; this bundle
reserves only the `step` op sequencing seam (REQ-007).

Grouped by spec seed:

- **Recipe apply mechanism (core)** — REQ-001 (generic resolve-and-run of ordered ops),
  REQ-002 (the four op families + declarative non-Turing-complete substitution), REQ-003 (two
  application modes from one artifact), REQ-004 (non-destructive + regenerate/waiver for
  recipe-owned output), REQ-005 (write the thin adoption record, not the rich ledger), REQ-006
  (zero language/platform literals — the `backstop/self`-guarded seam), REQ-007 (reserve the
  `step` op slot; executor is BUNDLE-019's). This is the piece `backstop init` (BUNDLE-003) is
  blocked on.
- **Manifest recipe declaration (schema)** — REQ-008 (per-recipe directory + `pack.yml`
  `recipes:` index), REQ-009 (the `recipe.yml` manifest fields + pack-manifest validation).
- **Recipe versioning + reference resolution** — REQ-010 (recipe-own version + `pack:recipe@version`
  resolution), REQ-011 (publish-time rev-guard), REQ-012 (the three drift signals).
- **Recipe enforcement scoping** — REQ-013 (adoption-gated double opt-in), REQ-014 (static path
  scope + partial-red / total-warn semantics).
- **Compat + version-keyed variants + migration** — REQ-015 (compat matrix, generic read +
  semver-compare, fail-loud), REQ-016 (version-keyed variant resolution, fail-loud on no match),
  REQ-017 (composition into migration-as-a-distributable-artifact).
- **CI recipe pack + provider standup packs** — REQ-018 (the CI recipe pack as the packs-only
  acceptance test), REQ-019 (the supabase / vercel / nextjs provider packs' FILE recipes).

## Draft Design Decisions

DD-1..DD-5 are the frame inherited from the BUNDLE-003 init session. DD-6..DD-8 are the
2026-07-14 "two axes + waiver" model. DD-9..DD-15 are the 2026-07-16 deepening; DD-16/DD-17 the
later 2026-07-16 compat/variants session. DD-18..DD-21 are the 2026-07-17 session resolving the
last four open questions. DD-22..DD-26 are the 2026-07-20 standup session, driven by the arrived
capture. All are founder-driven and recorded as decided.

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

- **DD-15: An application emits an ADOPTION RECORD; the rich provenance ledger is BUNDLE-017.**
  (Narrowed by DD-20 on 2026-07-17 — originally "the per-application PROVENANCE LEDGER primitive.")
  The thin, load-bearing contract that STAYS in this bundle: a recipe application emits a MINIMAL
  tracked ADOPTION RECORD — `{recipe ref, @version, adopted}` (DD-20) — which is the
  enforcement-activation primitive (DD-19) and carries the applied `@version` for the
  recipe-moved-on drift signal (DD-12). The RICH per-op/per-region record — the thing that scopes a
  dynamic `transform`'s enforcement (OQ-10 dynamic half), backs forensic replay, fleet/migration
  dashboards, and the concurrency reconciliation record (DD-14) — is the downstream **BUNDLE-017**
  provenance ledger. Its exact form is owned there, not here.

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

### Declaration shape, adoption-gating, the ledger split, and the rev-guard (2026-07-17)

- **DD-18: A recipe is declared as its OWN DIRECTORY; `pack.yml` carries a lightweight INDEX.**
  (Resolves OQ-2 and the OQ-5 residual.) A recipe is a directory colocating a `recipe.yml`
  manifest + its template payload + its `transform`-rule files. `pack.yml` carries a lightweight
  `recipes:` INDEX mapping stable recipe-id → directory. Multiple DISTINCT recipes in one pack =
  multiple directories, each addressed by a stable id in the pack namespace, each internally
  variant-resolved (DD-17) — which is also the resolution of OQ-5's residual multiplicity.
  Rationale: chosen OVER a single inline `recipes:` block in `pack.yml`. The two share the fact
  that ops always point at on-disk payloads; the only thing that moves is where recipe METADATA
  lives. Inline wins only for a handful of light `scaffolding` recipes; it TIPS OVER on the heavy
  `implementing`/migration kind — DD-16/DD-17 variants + codemods + enforcement bloat a shared
  `pack.yml` into a monolith and couple unrelated recipes into one review/diff surface. Per-recipe
  dirs make recipe-own versioning (DD-12) physically true: a recipe that hasn't changed touches no
  files. The founder's call: the heavy packs are where the real power is, so optimize the shape for
  them. The `recipes:` index does NOT collide with `content.scaffolds` (a distinct top-level key)
  or with `pack scaffold` / `artifact new` — the naming-collision hazard is respected.

- **DD-19: Enforcement is ADOPTION-GATED, with static path scope from declared targets.**
  (Resolves OQ-10, static scope.) A recipe's paired enforcement (DD-7) is ACTIVE only for ADOPTED
  recipes — those present in the adoption record (DD-20). A recipe an installed pack merely SHIPS
  but the consumer never applied is INERT (never red). Combined with enforcement being opt-in per
  recipe (DD-7), this is a DOUBLE opt-in before anything can go red — the deliberate guard against
  overly-eager enforcement. Static scope = the recipe's declared target paths, expressed via the
  gate's EXISTING per-rule path-scoping — NO new gate primitive (confirm at spec time that the gate
  can express per-rule include-globs over a declared path set; believed yes since pack rules
  already scope this way — "verify, don't assert"). At gate time, for an ADOPTED recipe:
  - declared output present/valid → GREEN.
  - PARTIAL or misplaced output (e.g. 3/5 declared files present, or one moved so its declared path
    is empty) → RED (broken promise), naming the absent paths — half-present is drift no one chose.
  - TOTAL absence of ALL declared files → WARN + guide ("recipe X looks removed; run un-adopt to
    clear it"), NOT red — the cautious default, because total absence reads as intent while partial
    reads as breakage.
  Deliberate divergence/removal is made accountable via a WAIVER (DD-8) or an explicit UN-ADOPT
  (drop the adoption entry; files stay as the consumer's own code per the removal non-fork). The
  declared file set is read from the ADOPTED @version's manifest, not the latest.
  DEFERRED: dynamic-`transform` output scoping (injection site unknown until apply) requires
  region-level provenance and is MOVED to BUNDLE-017 (provenance ledger). Only the static half is
  resolved here.

- **DD-20: The DD-15 ledger SPLITS — a thin ADOPTION RECORD here, the rich ledger → BUNDLE-017.**
  (Resolves OQ-11 for this bundle.) DD-15's "per-application provenance ledger" is SPLIT. A
  MINIMAL, tracked ADOPTION RECORD — `{recipe ref, @version, adopted}`, the recipe analog of
  `backstop.lock` — STAYS in this bundle. It is the enforcement-activation primitive (DD-19) and
  carries the applied @version for the recipe-moved-on drift signal (DD-12). The RICH provenance
  ledger — per-op/per-region detail, dynamic-`transform` output, forensic replay, fleet/migration
  dashboards, concurrency reconciliation record — SPINS OUT to **BUNDLE-017** as a downstream
  dependency. DD-15's language narrows to the thin contract: "an application emits an adoption
  record; the rich provenance ledger is BUNDLE-017." Consequence: the scaffolding kind (the CI-pack
  acceptance test, init's first consumer) ships on the thin adoption record alone; the
  `implementing`/migration kind and the dynamic-scope + fleet-dashboard payoffs depend on
  BUNDLE-017.

- **DD-21: A publish-time REV-GUARD forces a recipe version bump when its content changes.**
  (Resolves OQ-12.) A publish-time GUARD in the pack-authoring tooling (OUTSIDE core) fails when a
  recipe's content changed but its `@version` did not — forcing a rev, keeping the recipe-moved-on
  drift signal (DD-12) trustworthy. The guard enforces THAT you rev; the SEMVER LEVEL
  (patch/minor/major) is the author's judgment — the guard never polices magnitude (no tool
  reliably infers breaking-ness). DEFERRED (not a promotion blocker): auto-ROLLING the containing
  pack's version from recipe deltas (changesets-style) waits until a real multi-recipe pack informs
  the diff-baseline (git tag / last-publish / snapshot) and the recipe-delta→pack-delta mapping
  policy — building that policy now would be guessing in a vacuum. Spec-time knob: hash only
  semantically-meaningful files so a whitespace-only edit doesn't force a rev.

### Full-system standup — decomposition, runbook, bootstrap ladder, diagnose-first UX (2026-07-20)

The arrived capture (see Current Thinking) is the forcing function for DD-22..DD-26. All were
founder-decided in this session. DD-23..DD-26 (the runbook capability itself) have since been
MIGRATED to **BUNDLE-019 (Runbooks)** — see their lineage stubs below; only DD-22 (provider-pack
decomposition) stays in 015, as it governs recipe/pack decomposition generally. Open mechanism
details are recorded as SPEC-TIME RESIDUALS, not new OQs.

- **DD-22: Recipes decompose into PROVIDER packs, not a monolithic standup pack.** The standup
  capability splits along providers: `supabase` (migrations/RLS patterns, private bucket + path-RLS,
  Supavisor pooler-posture check, local-stack script), `vercel` (provision / env / cron / deploy —
  its RUNBOOK fragment is BUNDLE-019-scoped — plus the `vercel.json` file op, which stays here),
  `nextjs` (framework adoption + app-shell scaffolding), and the existing **github-actions CI pack**
  (= DD-5's canonical pack; the OIDC emitter workflow slots in). **PostHog is DEFERRED** — too thin
  to stand alone; wait for a second observability consumer. Rationale: providers evolve on
  INDEPENDENT version clocks → DD-16 compat + DD-17 variants apply PER PROVIDER; a monolith couples
  unrelated churn (the same tipping-over logic as DD-18, one level up); composable-by-default (a
  consumer installs only what it uses). **vercel vs nextjs split specifically** because they are
  different KINDS (DD-6): provisioning-runbook-dominant vs scaffolding-dominant. Cross-pack concerns:
  (a) ORDERING — the consumer's declared recipe sequence IS the order (the OQ-9 sequential-apply
  dissolution; its extension to runbook STEPS is BUNDLE-019's, which reuses this sequencing seam);
  (b) PARITY CHECKS spanning packs (e.g. OIDC audience must match between Vercel env and a GitHub
  repo var) — owned by the pack shipping the recipe for ONE side, with the OTHER side's location as
  a declared PARAMETER (the check stays declarative: read-at-A / read-at-B / assert-equal,
  DD-16-shaped); (c) shared config surfaces (`.env.example`) compose via `merge` ops (already
  covered). **RESTRAINT:** exactly one fixture exists — no finer splits until a SECOND consumer
  demands them (decomposition derives from consumers, not from symmetry). The runbook FRAGMENTS of
  these provider packs are scoped to **BUNDLE-019**; their FILE recipes (`vercel.json`, shells,
  workflows, migrations, app shells) remain here.

- **DD-23: The EXECUTABLE RUNBOOK — step ops with modes and receipts. → MIGRATED to BUNDLE-019.**
  Decided in this 2026-07-20 session: a recipe MAY declare ordered RUNBOOK STEPS with paired
  verify-checks ("a checklist with receipts"), each step carrying an execution MODE
  (auto / assisted / consent / decision), receipts reusing the DD-16 selectors + check-command
  engines, core rendering/sequencing/verifying but NEVER executing a platform op, step state
  derived not stored, and steps likely a fifth `step` op family. The runbook CAPABILITY is now
  owned by **BUNDLE-019 (Runbooks)** — generalized beyond standup to the ops and maintenance genres
  — where this is recorded as DD-1. BUNDLE-015 retains only the op-schema / apply-sequencing SEAM
  the step op rides on (see DD-22 and the apply-mechanism seed).

- **DD-24: The BOOTSTRAP LADDER — preconditions and computed effective mode. → MIGRATED to
  BUNDLE-019.** Decided in this 2026-07-20 session: scriptability is a function of standing state,
  not a static step property — Tier 0 (account existence, never scriptable) / Tier 1 (auth
  bootstrap, human-does machine-verifies) / Tier 2 (provisioning, reachable when T0/T1 receipts are
  green); steps declare preconditions and the effective mode is computed against current state; the
  human core is explicit / minimal / verifiable / one-time (the one-time-ness IS the agency
  amortization); Tier-1 tokens land in **Stash (BUNDLE-018)**. Now owned by **BUNDLE-019
  (Runbooks)** as DD-2 — the bootstrap ladder is a general runbook concern, not standup-specific.

- **DD-25: DIAGNOSE-FIRST UX + the AUTOMATION ESCALATION LADDER. → MIGRATED to BUNDLE-019.**
  Decided in this 2026-07-20 session: the runbook opens with a silent probe pass and renders ONLY
  unmet steps with their fix attached; probe law (receipts read-only / idempotent / cheap /
  fail-soft); the assisted-rung device-flow mechanics; the automation escalation ladder
  API > CLI > deep-link > click-instruction; click-instructions declared BEST-EFFORT
  ("instructions are vibes, receipts are law" — warn-and-rev, never red); invisible ≠ uninspectable
  via an `--explain` action log. Now owned by **BUNDLE-019 (Runbooks)** as DD-3 — the diagnose-first
  surface is the general runbook UX. (Its click-instruction promotion sweep is itself a
  maintenance-genre runbook there.)

- **DD-26: The COMPOSITION/FRAMEWORK step is first-class. → MIGRATED to BUNDLE-019.** Decided in
  this 2026-07-20 session: a standup recipe MUST include the code-wiring step as first-class — Mode
  B (DD-11: recipe supplies the adapter template WHAT + an acceptance-check GUARANTEE like "app
  boots; ingest route returns non-500"; the plan supplies the WHERE), the acceptance check being the
  step's receipt — because provisioned infra without composition is a VACUOUS standup (the portal's
  deployment-substrate + composition-root gap proves it). Now owned by **BUNDLE-019 (Runbooks)** as
  DD-4, scoped to the standup genre. (The Mode-B / composition-root mechanics remain BUNDLE-015's
  via DD-11; DD-4 makes the step first-class within a standup runbook.)

## Open Questions

Status index (numbers held stable across versions for traceability; resolved/dissolved OQs kept
with their decision so the reasoning survives):

- OQ-1 Recipe format — **RESOLVED** (DD-9, operation-set)
- OQ-2 Manifest declaration — **RESOLVED** (DD-18, per-recipe directory + `recipes:` index)
- OQ-3 Invocation — **RESOLVED** (DD-11, two modes; small ordering residual)
- OQ-4 Conflict (recipe-vs-user-file) — **RESOLVED-via-waiver** (residual: apply-time mechanics)
- OQ-5 Multiplicity & addressing — **RESOLVED** (DD-17 variants = primary form; DD-18 distinct-recipe directories = the residual)
- OQ-6 Is CI a distinct kind? — **RESOLVED** (DD-6, CI is scaffolding-kind)
- OQ-7 Templating engine — **RESOLVED** (DD-9/DD-10; residual: exact substitution syntax)
- OQ-8 Lifecycle on upgrade — **RESOLVED-via-waiver**
- OQ-9 Cross-pack collision — **DISSOLVED** (sequential ordered apply + non-destructive)
- OQ-10 Enforcement scoping — **RESOLVED** (DD-19, static half; dynamic-transform output scoping → BUNDLE-017)
- OQ-11 Provenance-ledger shape — **RESOLVED-via-split** (DD-20, thin adoption record here; rich ledger → BUNDLE-017)
- OQ-12 Recipe→pack version derivation — **RESOLVED** (DD-21, rev-guard; auto-roll deferred)
- Citation / traceability — **RESOLVED** (DD-12, `pack:recipe@version`)

ALL open questions are now resolved (with the OQ-10 dynamic half and the OQ-11 rich ledger
delegated to **BUNDLE-017**). Maturity stays `exploring` — the founder triggers promotion
separately.

### Open

None. As of the 2026-07-17 founder session all open questions are resolved (see below); the
OQ-10 dynamic-transform output scoping and the OQ-11 rich provenance ledger are DELEGATED to
**BUNDLE-017**, not left open here.

### Resolved / dissolved (kept for the reasoning)

- **OQ-2 — Manifest DECLARATION. (RESOLVED → DD-18.)** A recipe is declared as its OWN DIRECTORY
  (`recipe.yml` manifest + template payload + `transform`-rule files, colocated); `pack.yml`
  carries a lightweight `recipes:` INDEX mapping stable recipe-id → directory. The declaration
  carries the ordered OPS (DD-9), the KIND (DD-6), the PARAM schema, the TARGET paths, the
  `transform` RULES, the paired ENFORCEMENT suite (DD-7 / DD-19), the recipe VERSION (DD-12), and
  — optionally — the COMPAT MATRIX (`{file, path, range}` selectors, DD-16) and VERSION-KEYED
  VARIANTS (`{compat range → ops}` sets, DD-17). **Why the directory over an inline `pack.yml`
  block:** the two share the fact that ops always point at on-disk payloads; the only thing that
  moves is where recipe METADATA lives. Inline wins only for a handful of light `scaffolding`
  recipes; it TIPS OVER on the heavy `implementing`/migration kind (variants + codemods +
  enforcement bloat a shared `pack.yml` and couple unrelated recipes into one review/diff
  surface), and per-recipe dirs make recipe-own versioning physically true (a recipe that hasn't
  changed touches no files). Does NOT collide with `content.scaffolds` or `pack scaffold` /
  `artifact new` (naming-collision hazard respected).

- **OQ-5 — MULTIPLICITY & addressing. (RESOLVED → DD-17 + DD-18.)** DD-17 answers the PRIMARY
  multiplicity form: version-keyed VARIANTS of ONE logical recipe (`{compat range → ops}` sets),
  addressed by one name and resolved by the consumer's environment (an SDK-integration recipe
  spanning stripe@12 / @15 / @19 without N separately-addressed recipes). DD-18 answers the
  RESIDUAL: a single pack shipping several DISTINCT logical recipes is N per-recipe DIRECTORIES,
  each addressed by a stable recipe id in the pack namespace, each internally variant-resolved per
  DD-17. Both multiplicities are now resolved.

- **OQ-10 — Enforcement DECLARATION + SCOPING. (RESOLVED → DD-19, static half; dynamic →
  BUNDLE-017.)** A recipe's paired enforcement (DD-7) is ADOPTION-GATED — active only for recipes
  present in the adoption record (DD-20), a DOUBLE opt-in with DD-7's per-recipe opt-in. STATIC
  scope = the recipe's declared target paths, expressed via the gate's EXISTING per-rule
  path-scoping (no new gate primitive; confirm at spec time). Gate outcomes for an adopted recipe:
  declared output present/valid → green; PARTIAL/misplaced → RED (broken promise, names absent
  paths); TOTAL absence → WARN + guide (reads as intent, not breakage). Divergence is accountable
  via a WAIVER (DD-8) or explicit UN-ADOPT. **DEFERRED to BUNDLE-017:** dynamic-`transform` output
  scoping (injection site unknown until apply) needs region-level provenance.

- **OQ-11 — Provenance-ledger SHAPE. (RESOLVED-via-split → DD-20.)** DD-15's ledger SPLITS. A
  MINIMAL, tracked ADOPTION RECORD — `{recipe ref, @version, adopted}`, the recipe analog of
  `backstop.lock` — STAYS here; it is the enforcement-activation primitive (DD-19) and carries the
  applied @version for the recipe-moved-on drift signal (DD-12). The RICH provenance ledger
  (per-op/per-region detail, dynamic-`transform` output, forensic replay, fleet/migration
  dashboards, concurrency reconciliation) SPINS OUT to **BUNDLE-017**. Consequence: the scaffolding
  kind (CI-pack acceptance test, init's first consumer) ships on the thin adoption record alone.

- **OQ-12 — Recipe→pack version DERIVATION. (RESOLVED → DD-21; auto-roll deferred.)** A
  publish-time REV-GUARD in the pack-authoring tooling (outside core) fails when a recipe's content
  changed but its `@version` did not — forcing a rev to keep the recipe-moved-on drift signal
  (DD-12) trustworthy. The guard enforces THAT you rev; the SEMVER LEVEL is author's judgment (no
  tool reliably infers breaking-ness). **DEFERRED (not a promotion blocker):** auto-ROLLING the
  containing pack's version from recipe deltas (changesets-style) waits for a real multi-recipe
  pack to inform the diff-baseline and the recipe-delta→pack-delta mapping policy. Spec-time knob:
  hash only semantically-meaningful files so a whitespace-only edit doesn't force a rev.

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

The decomposition is now clear; all OQs are resolved, with the dynamic-transform scoping and rich
provenance ledger delegated to BUNDLE-017. The 2026-07-20 standup session adds two seeds (the
runbook/step-op + probe/receipt engine, and the provider standup packs) and extends three existing
ones.

- **Recipe apply mechanism (core)** — the generic executor: resolve a recipe by
  `pack:recipe@version`, run its declarative OPS (`create` / `merge` / `transform` / `insert`,
  DD-9) with declarative substitution (DD-9/OQ-7), across the two application MODES (direct +
  SDLC-mediated, DD-11), non-destructive toward user files with the regenerate/waiver model for
  recipe-owned output (DD-4/DD-8), writing an ADOPTION RECORD entry — `{recipe ref, @version,
  adopted}` (DD-20), NOT the full ledger. `transform` dispatches to allowlisted general engines
  running declared rules (DD-10). Contains ZERO language/platform literals (DD-3) — this seed is
  what `backstop/self` guards. The piece BUNDLE-003 `backstop init` is blocked on. Dynamic-
  `transform` enforcement scoping (region-level provenance) depends on BUNDLE-017. **Extended
  (2026-07-20):** RESERVES the `step` op slot in sequencing (interleaving file ops and runbook steps
  in one ordered apply) as the fifth op family — but the step EXECUTOR / probe-receipt engine and
  precondition evaluation are **BUNDLE-019 (Runbooks)**'; this seed owns only the op-schema /
  sequencing SEAM the step op rides on.

- **Manifest recipe declaration (schema)** — the per-recipe DIRECTORY shape (DD-18): a
  `recipe.yml` manifest + template payload + `transform`-rule files, colocated, indexed from
  `pack.yml`'s lightweight `recipes:` block (stable recipe-id → directory). The manifest declares
  ops, kind (DD-6), param schema, target paths, transform rules, paired enforcement (DD-7/DD-19),
  recipe version (DD-12), and optional compat (DD-16) + variants (DD-17). Validated by pack-manifest
  validation; distinct from `content.scaffolds`.

- **Recipe versioning + reference resolution** — the `<pack>:<recipe>@<recipe_version>` scheme
  (DD-12): recipe-own versioning distinct from pack version, reference resolution at apply time,
  the publish-time REV-GUARD forcing a rev when recipe content changes (DD-21; auto-roll of the
  pack version deferred), and the THREE drift signals — enforcement (code-diverged), recipe-moved-on
  (the adoption record carries the applied @version, DD-20), and compat (world-moved, DD-16). The
  fuller drift/forensic surface depends on BUNDLE-017.

- **Recipe enforcement scoping** — adoption-gated activation (DD-19): enforcement is active only
  for adopted recipes (DD-20) — a double opt-in with DD-7. STATIC scope from the recipe's declared
  target paths via the gate's existing per-rule path-scoping (no new gate primitive; confirm at
  spec time). Gate outcomes: present/valid → green; partial/misplaced → red (broken promise);
  total absence → warn + un-adopt guidance. Dynamic-`transform` output scoping depends on
  BUNDLE-017.

- **Compat + version-keyed variants + migration** — the optional compat matrix (DD-16:
  generic read-value-at-path + semver-compare over declared `{file, path, range}` selectors),
  version-keyed variant resolution (DD-17: pick the variant whose range matches the environment,
  fail loud on no match), and the migration-as-artifact flow they compose into (see the strategic
  note): bump SDK → compat red → re-apply → new variant's `transform` carries the codemod →
  enforcement guarantees completeness. Most relevant to the `implementing` kind.

- **CI recipe pack (backstop-packs, first consumer)** — the cross-cutting pack holding
  per-platform gate-workflow recipes as data (DD-5); the invariant's acceptance test. Lives in
  backstop-packs, not core. Ships on the thin adoption record alone (DD-20). **Extended (2026-07-20):**
  now also carries the OIDC EMITTER WORKFLOW pattern (`.github/workflows/backstop-ingest.yml` — the
  captured portal's CI-ingest side) and one side of the cross-pack OIDC-audience parity check (DD-22).

- **Provider standup packs (backstop-packs)** — the provider-pack decomposition (DD-22): `supabase`
  (migrations / RLS patterns, private bucket + path-folder RLS, Supavisor transaction-pooler posture
  check, scripted local stack), `vercel` (the `vercel.json` file op + provision / env / cron / deploy),
  and `nextjs` (framework adoption + app-shell scaffolding). Each is its own pack on an independent
  version clock (per-provider compat + variants); vercel vs nextjs split because they are different
  KINDS (provisioning-runbook-dominant vs scaffolding-dominant, DD-6). **Scope split:** this seed owns
  each pack's FILE recipes (`vercel.json`, shells, workflows, migrations, app shells); each pack's
  RUNBOOK fragments — the provision / env / cron / deploy STEPS, the §5-gotcha verify-check receipts,
  and the composition step — are **BUNDLE-019 (Runbooks)**-scoped (see its standup-genre-fragments
  seed). Sourced from the bclabs-portal capture (the standing fixture; slices land in each pack's
  `fixtures/captured/`). PostHog DEFERRED until a second observability consumer (DD-22). RESTRAINT: no
  finer splits until a second consumer demands them.

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
- 0.6.0 (2026-07-17): **Founder session — declaration shape, adoption-gating, the ledger split,
  and the rev-guard; ALL open questions resolved.** Added DD-18 (a recipe is declared as its OWN
  DIRECTORY — `recipe.yml` + template payload + transform-rule files — indexed from a lightweight
  `pack.yml` `recipes:` block; per-recipe dirs over an inline block because the heavy
  `implementing`/migration kind bloats and couples a shared `pack.yml`, and dirs make recipe-own
  versioning physically true — RESOLVING OQ-2 and the OQ-5 residual), DD-19 (enforcement is
  ADOPTION-GATED — active only for recipes in the adoption record, a double opt-in with DD-7 —
  with STATIC scope from declared target paths via the gate's existing per-rule path-scoping;
  present/valid → green, partial/misplaced → red, total-absence → warn+un-adopt — RESOLVING OQ-10's
  static half), DD-20 (SPLITS DD-15's ledger: a thin tracked ADOPTION RECORD `{recipe ref, @version,
  adopted}` stays here as the enforcement-activation primitive, the rich provenance ledger spins out
  to BUNDLE-017 — RESOLVING-via-split OQ-11), and DD-21 (a publish-time REV-GUARD in the pack-
  authoring tooling forces a recipe version bump when content changes, keeping the recipe-moved-on
  signal trustworthy; guard enforces THAT you rev, not the semver level — RESOLVING OQ-12).
  Narrowed DD-15's language to the thin adoption-record contract. Reconciled OQs: **OQ-2, OQ-5,
  OQ-10, OQ-11, OQ-12 all RESOLVED** — ALL open questions now closed, with the OQ-10
  dynamic-`transform` output scoping and the OQ-11 rich forensic/fleet ledger DEFERRED to
  **BUNDLE-017 (recipe provenance ledger)**, added as a downstream dependency reference; auto-roll
  of the pack version from recipe deltas (DD-21) also deferred. Refreshed the "settled vs open"
  summary, the solution approach, the provenance/DD-header notes, and the spec seeds (apply
  mechanism now writes an adoption record not the full ledger; manifest seed = per-recipe directory +
  `recipes:` index; versioning seed adds the rev-guard; added a recipe-enforcement-scoping seed).
  No OQs pre-resolved by the author (these are the founder's own decisions, recorded); maturity
  unchanged (exploring) — founder triggers promotion separately.
- 0.7.0 (2026-07-20): **Founder standup session — the capture arrived; full-system standup model.**
  The 2026-07-18 capture-first bet paid off: the bclabs-portal go-live capture (Supabase + Vercel +
  GitHub + PostHog), explicitly authored as the real-life recipe fixture, arrived and was
  CROSS-CHECKED against the portal repo (deployment-substrate gap, aspirational `POSTHOG_*`, dead
  `PORTAL_OIDC_SIGNING_KEY`, live `PORTAL_REPO_PATH`). Recorded DD-22 (recipes decompose into PROVIDER
  packs — supabase / vercel / nextjs / the CI pack, each on its own version clock; PostHog deferred;
  vercel vs nextjs split on KIND; cross-pack ordering / parity-checks / merge-composed config;
  one-fixture restraint), DD-23 (the EXECUTABLE RUNBOOK — ordered step-ops with modes
  auto / assisted / consent / decision + paired receipts, "checklist with receipts," step state
  derived not stored, likely a fifth `step` op family — promoting the 2026-07-18 candidate runbook DD),
  DD-24 (the BOOTSTRAP LADDER — Tier-0 account / Tier-1 auth / Tier-2 provisioning, preconditions +
  computed effective mode; one-time-ness IS the agency economics; Stash tie), DD-25 (DIAGNOSE-FIRST
  UX + the automation escalation ladder API > CLI > deep-link > click-instruction; probe law;
  receipts-are-law / instructions-are-vibes; `--explain` action-log), and DD-26 (the
  COMPOSITION/framework step is first-class — infra without it is a vacuous standup). Added two spec
  seeds (runbook / step-op + probe/receipt engine; provider standup packs) and extended three (apply
  mechanism gains step-op sequencing + precondition eval; CI-pack gains the OIDC emitter workflow;
  the standup packs carry the §5 gotchas as verify-checks). Wove the standup model into the solution
  approach, Current Thinking ("THE CAPTURE ARRIVED"), and the settled-vs-open summary. Added the
  capture doc + portal cross-check and BUNDLE-018 (Stash) to References. NO resolved OQ reopened;
  open mechanism details recorded as SPEC-TIME RESIDUALS, not new OQs. Maturity unchanged
  (exploring) — the founder's walkthrough of the standup model precedes promotion.
- 0.8.0 (2026-07-20): **The slim — the runbook capability carved out to BUNDLE-019.** Shortly after
  0.7.0 recorded the standup runbook model, the founder GENERALIZED: runbooks are a general
  declared, receipt-verified operational-procedure capability of which STANDUP is only the first and
  rarest genre (the frequent ones are OPS — rotation / cert renewal / incident diagnostics / backup
  verification / access reviews — and scheduled MAINTENANCE sweeps). So the runbook capability was
  CARVED OUT to **BUNDLE-019 (Runbooks)** — the same carve pattern as this bundle from BUNDLE-003 and
  BUNDLE-017 from here. MIGRATED DD-23 (executable runbook), DD-24 (bootstrap ladder), DD-25
  (diagnose-first UX + automation escalation ladder), and DD-26 (first-class composition step) to
  BUNDLE-019 (recorded there as DD-1..DD-4) — replacing each body here with a brief lineage stub
  (what it decided + "→ MIGRATED to BUNDLE-019") so the record that they were decided in this session
  is NOT stranded. DD-22 (provider-pack decomposition) STAYS — it governs recipe/pack decomposition
  generally — with its runbook-fragment mentions trimmed to cite BUNDLE-019 (the provider packs'
  FILE recipes remain here; their RUNBOOK fragments are 019-scoped). Spec seeds: REMOVED the
  "runbook / step-op + probe/receipt engine" seed (migrated to 019); the apply-mechanism seed now
  RESERVES the `step` op slot in sequencing with the executor noted as 019's; the provider-standup-
  packs seed gains the file/runbook scope split; the CI-pack seed is unchanged. Added the
  "Carve-out" note to Current Thinking (runbooks generalized beyond standup → 019; 015 returns to
  the file-op recipe capability init is blocked on) and BUNDLE-019 to References. No OQs reopened or
  resolved; maturity unchanged (exploring) — promotion is the founder's next step, imminent.
- 0.9.0 (2026-07-20): **Promotion to `defined` — requirements formalized.** Founder-triggered
  promotion after the full standup/file-op-scope walkthrough. Advanced `status.maturity`
  `exploring` → `defined` and added the required `requirements:` frontmatter array (REQ-001…
  REQ-019) plus a `Draft Requirements` section indexing it. The 19 requirements FORMALIZE the
  slimmed 6-seed file-op scope — each derived from a decided DD / resolved OQ (no new scope
  invented): the generic apply mechanism (REQ-001…007, incl. the reserved `step` op seam),
  per-recipe-directory manifest declaration + `recipes:` index (REQ-008/009), recipe-own
  versioning + `pack:recipe@version` resolution + rev-guard + three drift signals (REQ-010…012),
  adoption-gated enforcement with static path scope and partial-red / total-warn semantics
  (REQ-013/014), the optional compat matrix + version-keyed variants composing into
  migration-as-a-distributable-artifact (REQ-015…017), and the CI recipe pack + provider standup
  packs as the packs-only acceptance test / first consumers (REQ-018/019). No DDs or OQs changed;
  the runbook-capability requirements stay OUT (carved to BUNDLE-019, with only the `step` op
  sequencing seam reserved here). No advance past `defined`.

## References

- **The standup capture — bclabs-portal go-live (THE fixture, 2026-07-20)** —
  `~/.claude/uploads/3826af56-28a4-484f-975c-492a561591c4/eaeae70d-golivescaffolding.md`. A full
  standup capture of bclabs-portal on Supabase + Vercel + GitHub + PostHog, explicitly authored as the
  real-life recipe fixture (§6 splits fixture-data vs generalizable patterns). The forcing function
  for DD-22..DD-26; persistent, with slices moving into each born pack's `fixtures/captured/`.
  **Portal cross-check findings (corrections to the capture):** (a) the code gap is the DEPLOYMENT
  SUBSTRATE, not just a composition root (`package.json` has no `next` / `react` /
  `@supabase/supabase-js`; framework uninstalled); (b) the 4 `POSTHOG_*` vars are ASPIRATIONAL
  (nothing reads them → derive env surfaces post-composition); (c) `PORTAL_OIDC_SIGNING_KEY` is dead
  (drop); (d) `PORTAL_REPO_PATH` is actively wired (cwd fallback), vestigiality unresolved. Empirics:
  ~9/11 steps CLI/API-scriptable; irreducible human core = token supply + naming + explicit-go on
  real-money provisioning; §5 gotchas are near-verbatim verify-checks; platform-state dominance
  confirmed.
- **BUNDLE-019 (Runbooks)** — the CARVE-OUT (2026-07-20) that now OWNS the runbook capability. DD-23
  (executable runbook), DD-24 (bootstrap ladder), DD-25 (diagnose-first UX + automation escalation
  ladder), and DD-26 (first-class composition step) MIGRATED there as DD-1..DD-4, generalized beyond
  standup to the ops and maintenance genres; lineage stubs remain here. BUNDLE-015 retains the op
  schema + apply sequencing (the `step` op rides this seam) and the provider packs' FILE recipes;
  BUNDLE-019 owns the step executor / probe-receipt engine, the runbook surface, and the provider
  packs' runbook fragments.
- **BUNDLE-018 (Stash — credential custody)** — the credential-custody tie for the bootstrap ladder
  (now BUNDLE-019 DD-2, formerly DD-24): Tier-1 auth outputs (provider tokens) land in Stash; future
  precondition receipts read presence-in-stash, later `use`-brokered so tokens never surface.
- **BUNDLE-003 (onboarding / `backstop init`)** — the consumer and origin. DD-12 (packs carry
  scaffolding recipes), DD-13 (hard thin-executor invariant, inherited here as DD-3), OQ-6
  (converge-never-clobber), DD-14 (ecosystem-scaffolder composition), OQ-7 (CI wired via a recipe
  pack). Records this capability as a BLOCKING dependency.
- **BUNDLE-017 (recipe provenance ledger)** — the DOWNSTREAM dependency spun out of DD-20. Owns
  the RICH provenance ledger (per-op/per-region detail, forensic replay, fleet/migration
  dashboards, concurrency reconciliation record), the dynamic-`transform` output scoping (the
  OQ-10 dynamic half — region-level provenance so an injection-site transform can scope its
  enforcement to what it actually wrote), and the fleet/migration visibility payoffs. This bundle
  ships on the thin tracked ADOPTION RECORD alone (DD-20); the `implementing`/migration kind and the
  dynamic-scope + dashboard payoffs depend on BUNDLE-017.
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
