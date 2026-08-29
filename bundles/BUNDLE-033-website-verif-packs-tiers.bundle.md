---
title: "Website Verification Packs and Tiered Flags"
number: BUNDLE-033
created: "2026-08-29"
schema_version: bundle/v2

bundle:
  name: website-verif-packs-tiers
  version: "0.1.0"
  created: "2026-08-29"
  updated: "2026-08-29"
  category: infrastructure

status:
  maturity: exploring

problem:
  summary: >
    Website release verification is baked into backstop-core as a monolithic consumer
    pipeline. `scripts/verify-public-site.sh` always runs sitecheck (≈3k LOC of Go),
    documentation-semantics integration, product-truth check, Jekyll production build,
    design-system asset install and contract render, structural/ownership/Pages/design-matrix
    checks, and Playwright Chromium no-JS / 200% matrices. Both
    `.github/workflows/site-verification.yml` (every PR) and `.github/workflows/pages.yml`
    (every main push) invoke that full entrypoint with no path or tier discrimination.
    A CSS-only or token-only change therefore pays the same ~20-minute cost as a
    capability-journey or Pages-identity change. That placement also violates the standing
    thin-executor invariant: Jekyll, Playwright, closed Seed-4 inventory role matrices, and
    design-system presentation policy live as first-party core code and workflows rather than
    as pack-declared checks. BUNDLE-032 deliberately put consumer acceptance in core; that
    choice now collides with packs-external / zero-baked-tool-knowledge and with day-to-day
    latency. Adjacent evidence: ISSUE-191 (`.cursor/` files fail the core-owned closed
    sitecheck inventory) and ISSUE-190 (homepage style work must green the full verifier).
  user_story: >
    As a maintainer shipping a presentation or content change to the public site, I want
    website release verification to live in the pack(s) that own each concern, and I want
    tiered invocation so a style-only change runs only the checks that can fail from that
    change — not the full Jekyll + Playwright + journey + deploy-identity pipeline — so that
    core stays a thin executor, pack ownership stays honest, and ordinary site edits do not
    burn twenty minutes of CI on every push.
  success_criteria: []

solution:
  approach: >
    UNDECIDED on ownership boundaries and tier mechanics — this bundle holds the question.
    Direction of travel (not yet design): (1) extract website release verification out of
    core into the appropriate external pack(s), following the BUNDLE-011 collapse-into-packs
    precedent rather than extending core sitecheck; (2) add tiered flags / selective
    invocation so style-class diffs do not always run the full pipeline. Pack ownership,
    what "extract" means (full relocation vs thin core orchestration of pack-declared
    engines), the tier substrate (Actions paths vs pack gate_type/profiles vs CLI flags),
    the tier cuts themselves, the relationship to BUNDLE-032's consumer-in-core DDs, and
    whether product-truth / Pages deploy identity move with the rest are all open (OQ-1
    through OQ-6). Requirements, design decisions, and spec seeds stay provisional sketches
    until those OQs resolve. Maturity stays exploring; the founder drives OQ resolution and
    promotion.
  assumptions: []

requirements: []
---

# Website Verification Packs and Tiered Flags

## Current Thinking

### Two coupled concerns, one charter

This bundle owns two founder-stated goals that share a design surface:

1. **Pack extraction** — move website release verification out of core into the appropriate
   pack(s).
2. **Tiered flags** — so a simple style change does not kick off a ~20-minute pipeline run.

They are coupled because the unit of selective invocation (a "tier") is only as honest as the
unit of ownership (which pack declares which check). Splitting CI with path filters while
leaving a 3k-LOC sitecheck monolith in core would paper over latency without fixing the
baked-tool defect. Extracting into packs without a tier model would relocate the same
always-on monolith. Both land here; neither is pre-solved.

### What is baked in core today

Verified against the tree on 2026-08-29:

| Surface | Role |
| --- | --- |
| `scripts/verify-public-site.sh` | Monolithic acceptance entrypoint |
| `scripts/sitecheck/**` | Structural, inventory, Pages, design-matrix, presentation checks |
| `scripts/verify-documentation-semantics-integration.sh` | Seed-2 consumer of documentation-semantics |
| `scripts/verify-product-truth.sh` / `scripts/generate-product-truth.sh` | Derived product-truth pipeline |
| `scripts/verify-pages-deployment.sh` / `scripts/stamp-pages-artifact.sh` | Deploy identity |
| `scripts/install-design-assets.sh` / `scripts/render-public-site-contracts/` | Design-system consumer glue |
| `tests/**` Playwright public-site specs | Browser / no-JS / 200% matrix |
| `.backstop/seed4-delivery-inventory.yml` | Closed path/role matrix for sitecheck |
| `.backstop/website-pack-releases.yml` | Pinned owner-pack release evidence |
| `.github/workflows/site-verification.yml` | PR: full verifier, every time |
| `.github/workflows/pages.yml` | Main: full verifier + deploy + identity |
| `.github/workflows/ci.yml` | Product-truth + documentation-semantics before gate |

`verify-public-site.sh` phases, in order: sitecheck race+coverage (≥80%) → documentation-semantics
integration → product-truth `--check` → Jekyll production build → design-system asset install →
contract render → sitecheck structure/ownership/Pages/design-matrix → Playwright.

### What already lives in packs

Pinned via `.backstop/website-pack-releases.yml` / lock:

- `backstop-ai/documentation-semantics` v0.1.1 — semantic rules + fixtures
- `backstop-ai/backstop-design-system` v0.1.5 — tokens, presentation rules, exports
  `contracts/public-site-acceptance.yml`

BUNDLE-032's settled split was: product truth in core, presentation in design-system,
reusable documentation semantics in its own pack, **consumer acceptance evidence in core**.
This bundle asks whether that last clause still holds.

### Why not an issue → plan

Issue→plan fits a constrained reactive repair with settled ownership. This work revisits a
BUNDLE-032 design decision, invents pack-boundary and tier-taxonomy answers that do not yet
exist, and touches workflows plus possibly two external pack repos. That is exploring-bundle
territory (same shape as BUNDLE-011's "collapse baked X into packs"), not a single ISSUE
with ready requirements. ISSUE-191 remains a separate open symptom of the closed core
inventory; this bundle cites it and does not absorb or close it.

### Standing invariants that constrain every answer

- **Thin executor / zero baked tool knowledge** — core runs what packs declare; it does not
  bake Jekyll, Playwright, or presentation policy. (CLAUDE.md first principle; BUNDLE-011
  precedent.)
- **Packs stay external** — lock file is the durability boundary; do not vendor pack
  verification into core.
- **Loud ≠ blocking** — absent capability warns; broken promises block.
- **BUNDLE-032 content design is not reopened** — visitor journey, neighborhoods, evidence
  corpus, and Seed 4/5 acceptance *substance* stay; only *where verification lives* and
  *how selectively it runs* are in scope.

## Draft Requirements

Provisional sketches only — not settled. Promote to real `requirements[]` only after OQ
resolution.

- **(sketch) R1** — Website release verification that encodes presentation, documentation
  semantics, or site-journey policy must be declared and released by the owning pack, not
  compiled into core as bespoke scripts that bake those tools.
- **(sketch) R2** — Core may retain only thin orchestration that invokes pack-declared
  contracts/engines and first-party product-truth / deploy-identity checks if OQ-2/OQ-6 keep
  those in core; it must not retain a parallel policy implementation.
- **(sketch) R3** — A style-class change (exact path set per OQ-4) must be able to run a
  declared narrow verification tier without invoking full browser/journey/deploy tiers.
- **(sketch) R4** — Wider tiers remain available and mandatory for changes that can break
  them; narrowing must never produce silent vacuous green over skipped applicable checks.
- **(sketch) R5** — Extraction must not weaken BUNDLE-032's installed-released-pack
  acceptance bar (local/stale/unreleased bytes cannot satisfy consumer acceptance).

## Draft Design Decisions

None settled. Candidate DDs after OQ resolution will likely cover: pack ownership map,
extraction shape, tier substrate, tier cut taxonomy, BUNDLE-032 supersession-or-follow-on
posture, and product-truth / Pages-identity home.

## Spec Seeds

Provisional decomposition only — seed list will rewrite when OQs resolve.

1. **Pack ownership + extraction cutover** — move or re-declare verification surfaces into
   the chosen pack(s); delete or shrink core sitecheck/entrypoints accordingly.
2. **Tiered invocation** — implement the chosen tier substrate and wire workflows / CLI so
   style-class diffs take the narrow path and full diffs still hit the wide path.
3. **Consumer residual (if any)** — whatever thin core glue OQ-2/OQ-6 leave behind, with
   tests that prove it cannot substitute for pack-owned policy.

## Open Questions

Work these one at a time with the founder. Leans are starting positions, not resolutions.

### OQ-1 — Pack ownership map

Which pack owns which verification concern?

- **(a)** Extend existing packs only: design-system owns presentation/browser/token
  acceptance; documentation-semantics owns semantic/corpus consumer checks; no new pack.
- **(b)** Add a new `website` / `public-site` pack that owns journey, inventory, Pages
  wiring, and orchestration; existing packs keep rule/fixture ownership only.
- **(c)** Hybrid: presentation → design-system; semantics → documentation-semantics;
  journey/inventory/Pages → new pack; core keeps only product-truth (see OQ-6).

**Lean:** **(c)** — matches BUNDLE-032's "presentation vs semantics vs product truth"
  split while giving the cross-cutting consumer pipeline a pack home instead of core.
  Cost: a third pack to govern and pin.

### OQ-2 — What "move out of core" means

- **(a)** Full relocation: sitecheck, Playwright harness, and verify entrypoints live in
  pack repo(s); core workflows only invoke pack-published commands/engines.
- **(b)** Thin core orchestration: core keeps a small dispatcher script/workflow that only
  calls pack-declared engines/contracts; policy, fixtures, matrices, and tool knowledge
  live in packs.
- **(c)** Contracts-only: packs export acceptance contracts; core keeps implementing the
  checkers against those contracts (minimal move — likely insufficient vs thin-executor).

**Lean:** **(b)** — preserves a single consumer-facing CI entry while deleting baked policy
  from core. (a) is cleaner ownership but splits CI UX across pack repos; (c) fails the
  zero-baked-tool test if checkers stay core-owned.

### OQ-3 — Tier substrate

How do tiered flags actually select work?

- **(a)** GitHub Actions `paths:` / path-filter jobs (and matching local script flags).
- **(b)** Pack-declared `gate_type` / profiles consumed by `backstop gate` diff-scope.
- **(c)** Explicit CLI flags on a retained entrypoint
  (e.g. `verify-public-site.sh --tier=style|content|full`).
- **(d)** Compose (a)+(c): Actions paths pick a tier; the entrypoint enforces the same
  tier locally so agents cannot diverge from CI.

**Lean:** **(d)** — Actions path filters alone drift from local runs; gate profiles alone
  do not cover Jekyll/Playwright workflow jobs that are not gate steps today. A named tier
  shared by CI and local entrypoint keeps one vocabulary.

### OQ-4 — Tier cuts (what is "style-only"?)

What change sets map to which tiers, and which checks run?

- **(a)** Two tiers: `style` (CSS/tokens/design assets + design-system pack rules +
  minimal render smoke) vs `full` (everything today).
- **(b)** Three tiers: `style` → `content` (Markdown/claims/topology, no Playwright
  journey) → `full` (browser + journey + Pages identity).
- **(c)** Fine-grained path→check matrix (inventory-driven); no named tiers.

**Lean:** **(b)** — matches the founder's "style change ≠ 20 minutes" pain while still
  distinguishing content edits from deploy/journey risk. (a) may still over-run content
  PRs; (c) is flexible but harder to explain and easier to get silently wrong.

Concrete path sets for each tier are **not** invented here — they are an output of
resolving this OQ (and must be tested so an out-of-tier path cannot vacuous-green).

### OQ-5 — Relationship to BUNDLE-032

- **(a)** Follow-on: preserve BUNDLE-032's acceptance bar; relocate the consumer
  implementation; record an additive DD that consumer evidence may live in packs.
- **(b)** Formal supersession: a BUNDLE-032 DD that required consumer acceptance *in core*
  is superseded; cite and replace.
- **(c)** Narrow carve-out: only presentation verification moves; semantics/journey
  consumer evidence stays core per BUNDLE-032.

**Lean:** **(a)** unless audit of BUNDLE-032 DDs finds a hard "must be core source" clause
  that cannot be read as "must be proven against the actual site with released packs" — in
  which case **(b)**. Do not silently contradict delivered specs; supersede explicitly if
  needed.

### OQ-6 — Product-truth and Pages deploy identity

Do these move with website verification?

- **(a)** Stay first-party core: product truth is core-owned project truth; Pages stamp /
  deploy identity is hosting/ops owned by this repo's release path.
- **(b)** Move both into the website/public-site pack (if OQ-1 creates one).
- **(c)** Split: product-truth stays core; Pages deploy identity moves with site
  verification pack; documentation-semantics integration moves with that pack.

**Lean:** **(c)** if OQ-1 picks a website pack; else **(a)**. Product-truth's authoritative
  sources are core artifacts (CLI, schemas, packs lock, release history) — packing it under
  "website" would invert ownership. Deploy identity is closer to site release verification.

## Version History

| Version | Date | Maturity | Notes |
| --- | --- | --- | --- |
| 0.1.0 | 2026-08-29 | exploring | Initial bundle. Captures founder intent to extract website release verification into packs and add tiered invocation. Six OQs filed with leans; no OQ resolved; no requirements promoted. |

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md` — website expansion; consumer-in-core placement under revisit
- `bundles/BUNDLE-011-collapse-legacy-codecheck-into-packs.bundle.md` — collapse-baked-into-packs precedent
- `specs/SPEC-075-static-public-site-design-system.spec.md` — sitecheck / verify-public-site consumer contract
- `specs/SPEC-076-end-to-end-website-capabilities.spec.md` — journey / capability acceptance
- `specs/SPEC-073-documentation-semantics-integration.spec.md` — semantics consumer integration
- `specs/SPEC-074-derived-product-truth-pipeline.spec.md` — product-truth pipeline
- `.backstop/website-pack-releases.yml` — pinned documentation-semantics + design-system releases
- `issues/ISSUE-191-cursor-env-files-outside-seed4-matrix.issue.md` — closed core inventory symptom
- `issues/ISSUE-190-restore-canonical-homepage-direction.issue.md` — full-verifier latency on style work
- `.github/workflows/site-verification.yml`, `.github/workflows/pages.yml`, `.github/workflows/ci.yml`
- `scripts/verify-public-site.sh`, `scripts/sitecheck/`
