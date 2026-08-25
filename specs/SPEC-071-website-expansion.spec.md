---
title: "Website Expansion"
number: SPEC-071
created: "2026-08-23"
status: canceled
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    TOMBSTONE — canceled before implementation. This premature narrow docs-shell decomposition
    was abandoned after BUNDLE-032 evolved and rejected it as a completion path. The broader
    website work continues through BUNDLE-032's five non-overlapping future spec seeds, but the
    bundle is this spec's upstream source, not its successor. Do not plan or implement this
    contract; the historical scope is preserved below as evidence of the abandoned decomposition.
    Expand backstop-core's public GitHub Pages surface from one custom landing page plus
    stock-theme Markdown into one coherent Backstop site. Preserve the existing landing
    page as the reference surface; render the five canonical Markdown docs through a
    shared Backstop docs shell supplied by a released backstop-design-system pack; define
    one explicit docs information architecture; and add deterministic built-site checks
    for expected routes, navigation, internal links, custom-domain continuity, and the
    existing design-system rules. This spec owns Core consumption and verification. It
    does not mutate the design-system repository's artifact ledger; the reusable docs-shell
    capability must be released there first under that repository's own governance.
  subject: docs

verification:
  level: static
  test_command: ./scripts/verify-website.sh

contracts:
  - file: scripts/verify-website.sh
    provides:
      - name: verify_website
        kind: function
        signature: verify_website()

requirements:
  - id: REQ-001
    text: >
      `docs/index.html` remains the reference home surface. Refactoring may extract shared
      header/footer primitives, but the resulting home page must retain the current major
      content hierarchy, canonical wordmark, terminal proof, CTA destinations, responsive
      behavior, keyboard-visible focus, and reduced-motion behavior.
    supports:
      - website-expansion:REQ-001@1.0.0
  - id: REQ-002
    text: >
      The built docs site must expose exactly one primary documentation navigation model
      containing Getting Started, Concepts, Artifact Workflow, Pack Authoring, and CLI
      Reference, in that order, with an unambiguous current-page state on every docs page.
      The shell must also expose routes to Home and the Backstop GitHub repository.
    supports:
      - website-expansion:REQ-002@1.0.0
  - id: REQ-003
    text: >
      `docs/getting-started.md`, `docs/concepts.md`, `docs/artifact-workflow.md`,
      `docs/pack-authoring.md`, and `docs/cli-reference.md` remain the canonical technical
      content bodies. Presentation changes may add Jekyll frontmatter or metadata to those
      files but must not duplicate their substantive content into parallel HTML pages.
    supports:
      - website-expansion:REQ-003@1.0.0
  - id: REQ-004
    text: >
      Core must consume the docs-shell presentation from a released and explicitly pinned
      `backstop-ai/backstop-design-system` version in `backstop.yml`. Shared docs layout,
      navigation primitives, typography, tokens, code treatment, focus behavior, and
      reduced-motion behavior must not be independently re-authored in Core when the pack
      provides them.
    supports:
      - website-expansion:REQ-004@1.0.0
  - id: REQ-005
    text: >
      Every generated public page must pass the installed design-system pack with no raw
      color literal outside its token declaration surface, no inline style attribute,
      keyboard-visible focus treatment, reduced-motion handling for non-essential motion,
      and canonical Backstop wordmark usage.
    supports:
      - website-expansion:REQ-005@1.0.0
  - id: REQ-006
    text: >
      The docs navigation must remain usable with JavaScript disabled at both desktop and
      narrow mobile widths. It must not trap focus, hide the current page, require a
      pointer to reach a canonical docs destination, or create a second competing list of
      primary docs destinations in the same shell.
    supports:
      - website-expansion:REQ-006@1.0.0
  - id: REQ-007
    text: >
      The implementation must remain compatible with the existing Jekyll/GitHub Pages
      deployment path, keep `docs/CNAME` authoritative for `backstop.sh`, and introduce no
      application server, SPA router, or client-side framework as a requirement for basic
      page rendering or navigation.
    supports:
      - website-expansion:REQ-007@1.0.0
  - id: REQ-008
    text: >
      A deterministic website verification command must build or inspect the generated
      site and fail non-zero when any expected canonical page is absent, an internal link
      resolves to no generated target, a primary navigation destination is wrong or
      missing, the CNAME changes unexpectedly, or the design-system gate reports a rule
      violation. The verifier must print the failing route or rule rather than only a
      generic failure.
    supports:
      - website-expansion:REQ-008@1.0.0
  - id: REQ-009
    text: >
      Copy touched by this implementation must remove stale or contradictory product
      positioning, including the old Cayman site description if it conflicts with the
      current landing-page framing, while preserving the technical meaning of canonical
      docs. This spec must not add a generalized prose linter, prose LSP, or writing-style
      pack.
    supports:
      - website-expansion:REQ-009@1.0.0

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      The generated home page retains the existing Backstop home hierarchy and interaction
      contract after any shared-shell extraction.
    tests:
      - verify_home_reference_surface
  - id: CLM-002
    requirement: REQ-002
    text: >
      Every canonical docs page renders the same ordered five-item primary docs navigation
      and identifies exactly one item as current.
    tests:
      - verify_docs_navigation
  - id: CLM-003
    requirement: REQ-003
    text: >
      Each canonical technical page is sourced from its existing Markdown file and no
      duplicate hand-maintained HTML content page is introduced.
    tests:
      - verify_canonical_markdown_sources
  - id: CLM-004
    requirement: REQ-004
    text: >
      `backstop.yml` pins a released design-system version that supplies the docs-shell
      capability consumed by Core.
    tests:
      - verify_design_system_dependency
  - id: CLM-005
    requirement: REQ-005
    text: >
      The expanded generated site passes all installed Backstop design-system rules.
    tests:
      - verify_design_system_gate
  - id: CLM-006
    requirement: REQ-006
    text: >
      Canonical docs navigation remains present and semantically reachable without
      JavaScript, with current-page and focus behavior represented in generated markup and
      CSS rather than runtime-only state.
    tests:
      - verify_no_js_navigation_contract
  - id: CLM-007
    requirement: REQ-007
    text: >
      The built site remains a static Jekyll/GitHub Pages site and preserves `backstop.sh`
      through the checked-in CNAME contract.
    tests:
      - verify_static_deployment_contract
  - id: CLM-008
    requirement: REQ-008
    text: >
      One deterministic verification command detects missing canonical routes, broken
      internal links, wrong primary navigation targets, CNAME drift, and design-system
      violations with actionable diagnostics.
    tests:
      - verify_website_failure_modes
  - id: CLM-009
    requirement: REQ-009
    text: >
      Site metadata and touched positioning copy contain no known stale Cayman-era product
      description, while no generalized prose-enforcement subsystem is added to Core.
    tests:
      - verify_site_positioning_metadata
---

# SPEC-071: Website Expansion

## Overview

The prior website sprint proved the visual system on a single page. This spec removes the
remaining product seam: Home is custom Backstop, while the first Docs click still enters a
stock Cayman presentation. The target state is intentionally boring architecturally: one
static site, one design system, one docs information architecture, and the existing
Markdown bodies left in place.

The repository boundary is deliberate. `backstop-design-system` owns the reusable
presentation capability; `backstop-core` owns its public content, navigation ordering,
pinned dependency, and verification that the generated site is whole. Research and review
workers may inspect layouts or output, but they do not allocate, rename, promote, or mutate
Backstop ledger artifacts in either repository.

### In scope

- Apply a released design-system docs-shell capability to Core's five canonical docs.
- Remove the public Cayman visual seam while keeping Jekyll/GitHub Pages.
- Establish the five-page docs IA and current-page behavior.
- Reconcile shared header/footer behavior with the existing landing page where useful.
- Add deterministic generated-site route/link/navigation/CNAME/design-rule verification.
- Correct stale site metadata or contradictory positioning encountered in touched files.

### Explicitly out of scope

- A landing-page redesign.
- A docs CMS, search service, SPA framework, application server, or client-side router.
- Rewriting the technical docs merely to make them sound more marketed.
- Building a generalized Backstop prose/writing-style pack or prose LSP.
- Mutating the `backstop-design-system` artifact ledger from this Core plan.

## Requirements

The machine-readable frontmatter is the acceptance contract. The critical boundary is
REQ-003/REQ-004: content stays in Core Markdown; reusable presentation stays in the design
system. An implementation that creates beautiful docs by copying layout CSS into Core fails
the spec even if screenshots look correct.

The canonical reader path is:

1. **Getting Started** — get a real Backstop result quickly.
2. **Concepts** — learn the mental model and primitives.
3. **Artifact Workflow** — understand bundle/spec/plan/issue flow.
4. **Pack Authoring** — extend enforcement deliberately.
5. **CLI Reference** — look up exact command behavior.

`CODEBASE-MAP.md` remains excluded from the public site unless a later decision explicitly
promotes it; it is repository documentation rather than part of this reader path.

## Implementation

### Dependency boundary

Implementation begins only after `backstop-ai/backstop-design-system` releases a version
with the docs-shell recipe or equivalent deterministic generation surface. Core updates the
pinned version in `backstop.yml`, relocks/installs through normal Backstop pack workflow,
and applies the released capability rather than copying generated decisions by hand.

The previous landing-page dogfood exposed a literal-template escaping collision when a
recipe emitted Liquid-bearing Jekyll templates. The docs shell must use the resolved
ISSUE-182 mechanism or a generation shape that cannot reintroduce that collision; a fragile
nested-template workaround is not acceptable.

### Core site shape

Expected Core-owned content remains:

- `docs/index.html` — home/reference surface.
- `docs/getting-started.md` — Getting Started.
- `docs/concepts.md` — Concepts.
- `docs/artifact-workflow.md` — Artifact Workflow.
- `docs/pack-authoring.md` — Pack Authoring.
- `docs/cli-reference.md` — CLI Reference.

The design-system capability may generate Jekyll layout/include/style assets. Core may add
a small machine-readable site manifest. `scripts/verify-website.sh` is the single local/CI
verification entrypoint and must expose `verify_website()` as the contract named above.

### Copy posture

Navigation labels, metadata, short introductions, and stale positioning directly exposed by
the migration may change. The implementation must not perform an unbounded docs rewrite or
smuggle a generalized prose-enforcement system into this website scope.

## Verification

`./scripts/verify-website.sh` must fail independently when:

- a canonical docs source is omitted from generated output;
- an internal site link targets a missing generated page or anchor;
- a canonical docs page has missing/duplicate current-page state;
- a Home, GitHub, or canonical docs navigation destination is wrong;
- `docs/CNAME` no longer carries the expected `backstop.sh` value;
- the installed design-system gate reports a rule violation;
- stale Cayman-era positioning metadata is reintroduced on a touched public surface.

The plan creates this verification contract before presentation changes, so implementation
runs against red checks rather than screenshot judgment. Visual review remains useful for
typography and responsive polish, but deterministic verification plus Backstop gate owns
acceptance.

## Dependencies

- A released `backstop-ai/backstop-design-system` version with reusable docs-shell capability
  and passing pack fixtures.
- ISSUE-182 resolved or explicitly avoided by the generator shape.
- Existing Jekyll/GitHub Pages publishing remains operational.

## Sharp Edges

- **Cross-repo governance:** dependency does not grant this Core plan ownership of another
  repository's ledger.
- **Generated-vs-source validation:** source-link checks alone cannot prove Jekyll routes.
- **Landing regression:** extracting shared primitives must not become a redesign.
- **Literal template nesting:** ISSUE-182 is known evidence, not a hypothetical risk.
- **Copy scope creep:** fixing stale metadata is in scope; wholesale voice rewrite is not.

## References

- BUNDLE-032 — Website Expansion.
- ISSUE-182 — Recipe Literal Placeholder Escaping.
- `docs/index.html` — landing-page reference surface.
- `docs/_config.yml` — current Cayman seam.
- `backstop.yml` — current design-system pack pin.
- `backstop-ai/backstop-design-system` — reusable presentation owner.
