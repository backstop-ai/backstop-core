---
title: "Recipe Provenance Ledger"
number: BUNDLE-017
created: "2026-07-17"
schema_version: bundle/v2

bundle:
  name: provenance-ledger
  version: "0.1.0"
  created: "2026-07-17"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Pack scaffolding recipes (BUNDLE-015) apply declarative operations into a consumer
    repo. A per-application PROVENANCE record of what each recipe application actually DID
    is a load-bearing primitive (BUNDLE-015 DD-15). In the 2026-07-17 BUNDLE-015 session
    the founder SPLIT that primitive in two. A MINIMAL, tracked ADOPTION RECORD —
    `{recipe ref, @version, adopted}`, the recipe analog of `backstop.lock` — STAYS in
    BUNDLE-015: it answers only "which recipes are applied here, at which version," gates
    enforcement activation (adoption-gated enforcement), and carries the applied `@version`
    for the recipe-moved-on drift signal. That thin piece is owned by BUNDLE-015 and is NOT
    this bundle's job. The RICH provenance ledger — everything heavier — spins out to THIS
    bundle as a genuine design surface of its own. It blocks the `implementing` / migration
    kind of recipe (which needs region-level detail), NOT the scaffolding kind. This bundle
    owns the design of that rich ledger: its durability model, entry schema, structure, its
    precise relationship to the thin adoption record, and the differing requirements its
    several readers impose.

  user_story: >
    As a consumer, pack author, and the founder (hosted capture), I want a rich, durable
    record of what each recipe application actually did — files AND regions touched, the op
    list, the applied `@version`, and the application mode — so that a dynamic `transform`'s
    enforcement can be scoped to exactly what the application WROTE, so that parallel-agent
    reconciliation (BUNDLE-015 DD-14) has an authoritative record to reconcile against, and
    so that fleet-wide drift and migration dashboards can prove which repos are on which
    recipe versions and whether a distributed migration actually completed — without the
    design prematurely committing to whether that record is tracked in-repo, gitignored, or
    split baseline-style.
---

# Recipe Provenance Ledger

## Current Thinking

This is a **spin-out**, carved from BUNDLE-015 (pack-scaffolding-recipes) exactly as
BUNDLE-015 was itself carved out of BUNDLE-003 (init). It is a founder-flagged dedicated
design session for the RICH provenance ledger. It starts at `exploring` with genuinely
open questions; the founder drives OQ resolution and promotion.

### The split that created this bundle (2026-07-17)

BUNDLE-015 DD-15 established that a per-application provenance record EXISTS and is
load-bearing, with three payoffs (dynamic-transform enforcement scoping, `@version`
traceability + the recipe-moved-on drift signal, concurrency reconciliation). In the
2026-07-17 session the founder split that one primitive into a thin part and a rich part:

- **The thin ADOPTION RECORD stays in BUNDLE-015.** `{recipe ref, @version, adopted}` — the
  recipe analog of `backstop.lock`. It answers exactly one question ("which recipes are
  applied in this repo, at which version"), gates enforcement activation
  (adoption-gated enforcement), and carries the applied `@version` that feeds the
  recipe-moved-on drift signal (DD-12). It is minimal and tracked. Not this bundle's job.

- **The rich LEDGER spins out here.** Everything heavier than the thin record — per-region
  detail, op lists, forensic history, the fleet/migration reader surface. It is a design
  surface of its own and blocks the `implementing` / migration kind of recipe (which needs
  the region-level record), NOT the scaffolding kind (served by the thin record + declared
  paths). This bundle owns that design.

### Settled / inherited context (NOT open questions)

These are decided upstream and recorded here as fixed frame, not as things to relitigate:

- **The ledger EXISTS.** BUNDLE-015 DD-15 decided that. This bundle designs its shape, not
  whether it should be.
- **The thin adoption record lives in BUNDLE-015, not here.** ref + `@version` + adopted;
  gates enforcement; carries the applied version for drift. This bundle designs the RICH
  superset that sits above / beside it.
- **This is a superset-or-sibling of the thin record, never a replacement for it.** The
  thin record's jobs (adoption-gating, the applied-version drift signal) are already served;
  the rich ledger exists for the readers the thin record CANNOT serve (region-level scoping,
  reconciliation record, fleet forensics). Delineating exactly which reader needs which is
  itself OQ-4.

### Why the durability model is the load-bearing question

The ledger's readers do not all want the same thing. Two of them — the drift signals and
the fleet/migration dashboards — are DURABILITY jobs, and durability argues for a tracked
in-repo record. But the bulkiest part (per-region `transform` output that isn't known until
apply) may be regenerable, and regenerable bulk argues for a gitignored derived cache. That
is the same tension the baseline faced (BUNDLE-007, tracked-vs-gitignored), which is why
OQ-1 deliberately mirrors it and belongs to the founder. It may eventually warrant an ADR —
once the model is DECIDED, not before (see Notes).

## Open Questions

These are the design session. They are genuinely open — NOT pre-resolved here. The founder
drives resolution and triggers promotion.

- **OQ-1 — Durability model.** Where does the ledger live and how durable is it? Options:
  (a) **tracked in-repo** — durable by construction, the drift/fleet readers get a committed
  record, but every apply churns a tracked file and per-region bulk lands in git history;
  (b) **gitignored `.backstop/`** — like `node_modules`/derived caches, no tracked churn, but
  the durable readers (drift, fleet dashboards) lose their record unless it's reconstructable;
  (c) **baseline-style split** — a small TRACKED durable core (enough for drift + fleet +
  reconciliation) plus a gitignored DERIVED cache for the bulky, regenerable per-region
  detail. This deliberately MIRRORS the baseline tracked-vs-gitignored debate (BUNDLE-007)
  and needs the founder. The tension to hold: the drift-signal and fleet-dashboard readers
  are durability jobs (→ tracked); the bulky per-region detail may be regenerable (→
  gitignored). This is the load-bearing OQ; no lean recorded — it belongs to the founder,
  and may be promoted to an ADR once resolved.

- **OQ-2 — Entry schema.** What does each ledger entry record? Candidates: the recipe
  reference `pack:recipe@version` (DD-12); the OP list that ran (`create`/`merge`/`transform`
  /`insert`, DD-9); the FILES touched AND — critically — the REGIONS touched, because a
  dynamic `transform`'s output isn't known until apply and region-level detail is what the
  enforcement-scoping reader (OQ-5a) needs; the application MODE (direct/deterministic vs
  SDLC-mediated, DD-11); a timestamp. Open sub-question: the GRANULARITY of "regions" — byte
  ranges? line ranges? AST-node spans? named anchors? — which trades scoping precision
  against how much survives the consumer later editing around the region.

- **OQ-3 — Granularity / structure.** How is the ledger organized? One ledger PER REPO vs one
  record PER APPLICATION. An append-only HISTORY LOG (every application ever, forensic) vs a
  current-state MANIFEST (what's live now, like a lock file). How does it represent MULTIPLE
  applications of the same recipe over time — re-applies, upgrades, a recipe applied then
  re-applied at a newer `@version`? History-log and current-manifest are not mutually
  exclusive (a log can be projected to current state), so this couples to OQ-1's split
  question.

- **OQ-4 — Relationship to the minimal adoption record (BUNDLE-015).** Is the rich ledger a
  SUPERSET that subsumes the adoption record (one structure, the thin record being a
  projection of it), or a SEPARATE structure that REFERENCES it? The delineation must be
  precise: the adoption record already carries `ref + @version + adopted` and already serves
  adoption-gating and the recipe-moved-on drift signal, so those readers need nothing from
  the rich ledger. Name exactly which reader is served by the thin record alone vs which
  genuinely requires the rich ledger — that mapping is the substance of this OQ and it
  interacts with OQ-1 (a superset is harder to split tracked-vs-gitignored than a sibling).

- **OQ-5 — The readers (each may impose different requirements).** The ledger has several
  distinct readers, and each may pull the design in a different direction — this OQ is about
  surfacing those requirements before OQ-1/OQ-2/OQ-3 are closed, because they constrain the
  answers:
  - **(a) Dynamic-transform ENFORCEMENT SCOPING.** BUNDLE-015 OQ-10's dynamic half was
    DEFERRED to here. Scoping a dynamic `transform`'s enforcement to what the application
    actually WROTE requires the ledger's region-level record (OQ-2). This is the reader that
    most drives entry granularity.
  - **(b) Concurrency RECONCILIATION.** BUNDLE-015 DD-14: parallel agents in worktrees
    reconcile via git merge, and enforcement is the semantic net; the ledger supplies the
    RECONCILIATION RECORD (what each agent's application did) that reconciliation reads.
  - **(c) FORENSIC REPLAY / FLEET DRIFT + MIGRATION DASHBOARDS.** The hosted-capture moat
    surface (see the agency business model). This is the strategically heaviest reader and
    likely the one that most pulls toward TRACKED + DURABLE — you cannot build a fleet drift
    or migration-completeness dashboard on a gitignored record that isn't reconstructable.
  Open: do these readers agree on a single structure, or do they force the OQ-1 split
  (durable core for b/c, derived cache for a)? No lean — the founder weighs them.

- **OQ-6 — Relationship to existing durable artifacts.** How should the ledger relate, for
  precedent and consistency, to the durable artifacts backstop already has — `backstop.lock`
  (the pack lock; the thin adoption record is explicitly modeled as its recipe analog) and
  the BASELINE (BUNDLE-007, the closest tracked-vs-gitignored precedent)? Should the ledger
  reuse their storage conventions, sit beside them, or borrow the baseline's exact split
  model? This is a consistency question that informs OQ-1 rather than a fully independent
  fork.

Maturity stays `exploring`. No OQs are pre-resolved; no design decisions or requirements
are recorded yet — those await founder-driven OQ resolution. Promotion is founder-triggered.

## Spec Seeds

Provisional only — this bundle is `exploring` and the decomposition firms up once OQ-1
(durability) and OQ-2/OQ-3 (schema/structure) resolve. Rough shape:

- **Ledger data structure + durability** — the on-disk form: entry schema (OQ-2), per-repo
  vs per-application and log-vs-manifest structure (OQ-3), and the tracked / gitignored /
  split durability model (OQ-1). This is the foundational seed everything else depends on.
- **Write path (apply mechanism emits entries)** — the recipe apply mechanism (BUNDLE-015's
  apply-mechanism seed) emits a ledger entry per application, recording ops + files/regions
  + `@version` + mode. Couples to the BUNDLE-015 executor.
- **Read paths** — the readers (OQ-5): enforcement scoping (dynamic-transform scope from the
  region record), drift (recipe-moved-on, feeding from `@version`), reconciliation (the
  concurrency record), and the fleet / migration dashboards (hosted capture). Likely
  decomposes further once the readers' distinct requirements (OQ-5) are pinned.

## Notes / Ideas

- **Sharpest hosted-capture surface.** The rich ledger — specifically its fleet-drift and
  migration-completeness dashboards — is the clearest hosted-capture surface backstop has. It
  ties directly to the business-model-agency thesis (backstop is OSS flag-planting; capture is
  via an agency using it). This is WHY the durability model matters beyond local correctness:
  a gitignored, non-reconstructable ledger cannot feed a fleet dashboard, so OQ-1 is a
  strategy question wearing a storage question's clothes.
- **Migration-as-a-distributable-artifact needs the ledger to be auditable.** BUNDLE-015's
  headline (compat matrix + version-keyed variants + `transform` + enforcement compose into a
  distributable, versioned, verified migration). The ledger records the PROVENANCE of a
  distributed migration's codemod application — what it rewrote, and (via enforcement +
  region detail) whether it completed. The ledger is what makes a FLEET migration auditable:
  "which repos re-applied, which are still on the old variant, which are red." Without it the
  migration is distributable but not observable at fleet scale.
- **The durability-model decision may be promoted to an ADR — once resolved, not now.** The
  founder mused that the tracked-vs-gitignored-vs-split decision (OQ-1) may eventually warrant
  an ADR, given it mirrors the baseline debate and has fleet/strategy implications. Recorded
  as a note only: there is NO decision yet, so there is NO ADR yet. Create it after OQ-1
  resolves, not before.

## Version History

- 0.1.0 (2026-07-17): Initial bundle at `exploring`, spun out of BUNDLE-015
  (pack-scaffolding-recipes) — carved from DD-15's provenance primitive after the 2026-07-17
  session split it into a thin tracked ADOPTION RECORD (stays in BUNDLE-015: ref + `@version`
  + adopted; gates enforcement; carries the applied version for drift) and this RICH ledger
  (the heavier superset, blocking the `implementing`/migration kind). Recorded the settled
  inherited context (the ledger exists per DD-15; the thin record lives in BUNDLE-015; this
  bundle designs the rich superset-or-sibling), six genuinely open questions (durability
  model mirroring the baseline debate; entry schema incl. region granularity; per-repo/
  per-application and log-vs-manifest structure; relationship to the thin adoption record;
  the several readers and their differing requirements — enforcement scoping, reconciliation,
  fleet/migration dashboards; relationship to `backstop.lock` and the baseline), three
  provisional spec seeds (data-structure+durability, write path, read paths), and the
  strategic notes (sharpest hosted-capture surface; ledger makes fleet migration auditable;
  the durability decision may become an ADR once resolved — not before). No OQs pre-resolved;
  no design decisions, requirements, or maturity self-promotion. Founder drives resolution
  and promotion.

## References

- **BUNDLE-015 (Pack Scaffolding Recipes)** — the parent. DD-15 (the provenance-ledger
  primitive this bundle inherits and designs), DD-14 (concurrency reconciliation — the ledger
  is the reconciliation record), DD-12 (recipe `@version` + the recipe-moved-on drift signal),
  OQ-10's DEFERRED dynamic half (dynamic-transform enforcement scoping needs the region-level
  ledger — the enforcement-scoping reader here), and OQ-11 (provenance-ledger shape — this
  bundle IS that open question, spun out). The thin adoption record stays in BUNDLE-015.
- **BUNDLE-007 (Baseline)** — the tracked-vs-gitignored precedent OQ-1 deliberately mirrors;
  reference for the split (tracked durable core + gitignored derived cache) model.
- **BUNDLE-013 (Waiver subsystem)** — the accountable-divergence machinery recipes reuse; the
  ledger records applications, waivers record deviations from them.
- **`backstop.lock` / pack distribution (BUNDLE-001 / BUNDLE-002)** — the durable-lock
  precedent; the thin adoption record is explicitly the recipe analog of `backstop.lock`, and
  OQ-6 asks how the rich ledger should relate to these existing durable artifacts.
