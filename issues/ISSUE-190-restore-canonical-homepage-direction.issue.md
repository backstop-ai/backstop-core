---
title: "Restore Canonical Homepage Direction"
schema_version: issue/v1

issue:
  id: ISSUE-190
  title: "Restore Canonical Homepage Direction"
  type: bug
  status: closed
  created: "2026-08-28"
  closed: "2026-08-29"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
resolved-by: cd453e8c1c31aeb210979013345d4931d789b378

verification:
  level: integration
  coverage_threshold: 80
  test_command: ./scripts/verify-public-site.sh

implementation:
  summary: >
    Reconcile the public homepage with the user-approved canonical mockup and the
    083ae5d source baseline while retaining the delivered Pages, navigation,
    accessibility, source-ownership, and verification capabilities. Remove only the
    accidental field-guide homepage drift; do not revert the repository or redesign
    the remaining public site.
  package: docs

requirements:
  - id: REQ-001
    text: >
      The homepage must restore the user-approved visual rhythm and content sequence
      represented by `tmp/homepage-reconciliation/canonical.html`, using commit
      `083ae5d955ca4216c8abcd778ef8c5c1fd6987bb` as the canonical source baseline for
      `docs/index.html`, `docs/assets/css/backstop.css`, and
      `docs/assets/css/backstop-tokens.css`. The rendered header must show the complete
      `./b backstop.sh` wordmark, followed by the canonical hero and gate proof, the
      three ordered system sections Define the work, Enforce your standards, and Detect
      drift, and the composability section. The truncated top mark and extraneous
      field-guide question/scaffold headings must not remain.
  - id: REQ-002
    text: >
      The composability section must contain exactly the four canonical product modes,
      in order: Full framework, Artifact workflow, Standards enforcement, and
      Deterministic scaffolding. Their copy must preserve the distinctions expressed by
      the approved mockup. The accidental Evaluate, Understand, and Adopt replacement
      rows are not an alternate completion path.
  - id: REQ-003
    text: >
      Preserve the current visitor-journey navigation information architecture: seven
      ordered primary links (Evaluate, Model, Adopt, Use Cases, Packs, Extend,
      Reference) and two utility links (Status, Contributing), with real canonical
      destinations. At 360x800, 768x1024, and 1440x1000 with JavaScript disabled, all
      nine links and the Home wordmark must remain visible, keyboard reachable, and
      usable; responsive styling must not hide most navigation links behind clipping,
      display suppression, or a pointer-only control. Preserve the skip link and current
      accessibility behavior.
  - id: REQ-004
    text: >
      At each existing desktop and mobile acceptance viewport, the homepage must have
      no horizontal document overflow beyond the existing one-CSS-pixel tolerance.
      The existing actual-200%-text test must continue to prove that the computed root
      font size is exactly twice baseline and must rerun the full visibility, content,
      keyboard-order, focus-bounds, local-overflow, and document-overflow assertions.
      Text enlargement may reflow content but must not hide navigation or canonical
      homepage content.
  - id: REQ-005
    text: >
      Preserve legitimate modern site capabilities while restoring the homepage:
      GitHub Pages deployment and the static no-JavaScript posture; source-owned claim
      markers and journey links; canonical root-relative internal destinations;
      ownership and generated-region boundaries; and the established footer and
      next-action conventions where they do not recreate the accidental field-guide
      wrapper. The approved mockup supplies intent only: mockup-only placeholders and
      `href="#"` values must not enter production.
  - id: REQ-006
    text: >
      Keep the existing ownership, delivery-inventory, structural, browser, and
      installed-design-system enforcement substantive and passing. Homepage-specific
      expectations that currently encode the accidental three-row or field-guide drift
      may be reconciled narrowly with the approved direction, but checks must not be
      skipped, weakened, made vacuous, or broadened into locally owned visual policy.
      The complete `./scripts/verify-public-site.sh` site verifier must pass.
  - id: REQ-007
    text: >
      This is a focused homepage correction, not a wholesale Git reversion or a broad
      redesign. Changes must be limited to homepage markup/style composition and the
      smallest corresponding contract, inventory, and verification updates needed to
      prove this issue. Content, layout, and behavior of the remaining documentation
      pages are out of scope except where a shared shell must continue to satisfy its
      existing contract.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The rendered homepage has the complete wordmark and canonical hero, gate proof, and three-section sequence without the accidental field-guide headings.
    tests:
      - TestSiteCheck_HomepageCanonicalDirectionPasses
      - TestSiteCheck_HomepageCanonicalDirectionRejectsDriftMatrix
  - id: CLM-002
    requirement: REQ-002
    text: The rendered homepage exposes exactly the four ordered canonical composability modes and rejects the three accidental journey rows as a substitute.
    tests:
      - TestSiteCheck_HomepageCanonicalDirectionPasses
      - TestSiteCheck_HomepageCanonicalDirectionRejectsDriftMatrix
  - id: CLM-003
    requirement: REQ-003
    text: The full 7-primary plus 2-utility navigation and Home wordmark remain visible and keyboard reachable across the no-JavaScript viewport matrix.
    tests:
      - TestSiteCheck_ChromiumNoJSExactInteractionMatrixPasses
      - TestSiteCheck_NavigationMatrixPasses
  - id: CLM-004
    requirement: REQ-004
    text: The homepage stays within the document-overflow tolerance under every viewport and under actual 200 percent root text while retaining the complete interaction assertions.
    tests:
      - TestSiteCheck_LocalOverflowAndNavigationModesPass
      - TestSiteCheck_ActualRootFontRelayoutPasses
  - id: CLM-005
    requirement: REQ-005
    text: Restoring the canonical direction retains real journey destinations, source-owned contracts, static Pages delivery, footer, and next action without importing mockup placeholders.
    tests:
      - TestSiteCheck_RenderedJourneyLinkMatrixPasses
      - TestSiteCheck_PagesWorkflowPinnedContractPasses
      - TestSiteCheck_CanonicalRouteMatrixPasses
  - id: CLM-006
    requirement: REQ-006
    text: The complete public-site verifier remains green with substantive structural, browser, inventory, ownership, and installed design-system checks.
    tests:
      - TestSiteCheck_VerificationPipelinePasses
      - TestSiteCheck_Seed4DeliveryInventoryPasses
      - TestSiteCheck_UpstreamOwnershipAndGeneratedRegionsPass
  - id: CLM-007
    requirement: REQ-007
    text: The delivered diff is confined to the homepage correction and its minimum required contract and verification updates, without unrelated documentation-page redesign.
    tests:
      - TestSiteCheck_HomepageCanonicalChangeFence

contracts:
  - file: docs/index.html
    provides:
      - name: canonical_homepage_direction
        kind: variable
        signature: "complete ./b backstop.sh wordmark; canonical hero and gate proof; three ordered system sections; four ordered composability modes"
    consumes:
      - source: docs/_data/content-topology.yml
        name: canonical_navigation_and_journey_links
        kind: variable
      - source: docs/assets/css/site.css
        name: shared_public_site_shell
        kind: variable
      - source: docs/assets/css/backstop.css
        name: canonical_homepage_presentation
        kind: variable
---

# Restore Canonical Homepage Direction

## Problem

The homepage on current main, based on
`fb0bfa9e700aecc2816d2a0dd00be9fa55a22e7a`, has drifted from the direction the
user approved in `tmp/homepage-reconciliation/canonical.html`. The top wordmark
is truncated to `./b .sh`; field-guide scaffold headings interrupt the intended
hero-to-system rhythm; and the canonical four composability modes have been
replaced by three Evaluate, Understand, and Adopt journey rows. Responsive
behavior also hides much of the navigation instead of keeping the full 7+2 IA
reachable.

The approved direction is grounded in the homepage sources at
`083ae5d955ca4216c8abcd778ef8c5c1fd6987bb`, but the intervening work also
delivered capabilities that must not be discarded: Pages deployment, the current
navigation model, accessibility behavior, source-owned claims and journey links,
and deterministic site enforcement. A wholesale reversion would lose legitimate
work; preserving the current accidental wrapper would reject the approved
homepage. The defect is the unreconciled homepage composition between those two
states.

This blocks continuation of the remaining documentation pages because the
homepage is their approved visual and sequencing anchor.

## Solution

Reconcile the current homepage with the approved mockup's intent and the
`083ae5d` source baseline. Restore its visual rhythm, full wordmark, three system
sections, and four product composability modes while carrying forward the modern
site capabilities and real destinations listed above. Do not copy mockup-only
metadata, placeholder links, or `href="#"` values.

### Acceptance criteria

- Desktop and mobile render the complete canonical sequence and full
  `./b backstop.sh` wordmark without the accidental field-guide headings.
- The composability section contains exactly the four canonical modes in their
  approved order and meaning.
- All seven primary and two utility navigation links remain visible, reachable,
  and keyboard usable at 360px, 768px, and 1440px; none are responsively hidden.
- The document has no horizontal overflow beyond the existing 1px tolerance,
  including under the existing actual-200%-root-text behavior and assertions.
- Current source-owned claim markers, journey links, canonical routes, Pages
  delivery, accessibility, ownership boundaries, footer, and compatible
  next-action conventions remain intact.
- The full public-site verifier passes without weakening or bypassing any site
  check.
- No unrelated documentation-page redesign or broad rest-of-site work is
  included.

## Verification

Run `./scripts/verify-public-site.sh`. Its locked build, source-ownership and
journey-link checks, delivery inventory, site structure, installed design-system
matrix, no-JavaScript Chromium viewport matrix, and actual-200%-text matrix must
all pass. Add focused positive and negative structural coverage for the homepage
sequence, complete wordmark, and exact four-mode composition so the prior drift
cannot pass vacuously.

## References

- User-approved intent: `tmp/homepage-reconciliation/canonical.html`.
- Canonical source baseline: commit
  `083ae5d955ca4216c8abcd778ef8c5c1fd6987bb` (`docs/index.html`,
  `docs/assets/css/backstop.css`, `docs/assets/css/backstop-tokens.css`).
- Current-main baseline: `fb0bfa9e700aecc2816d2a0dd00be9fa55a22e7a`.
- Current integration surfaces: `docs/assets/css/site.css`,
  `tests/public-site-helpers.ts`, `tests/public-site.spec.ts`,
  `scripts/sitecheck/`, and `.backstop/seed4-delivery-inventory.yml`.
- BUNDLE-032 / SPEC-075 delivered the wider public-site capability and its
  enforcement. This issue records a reactive reconciliation defect in the
  delivered homepage, not a competing charter for the proactive website
  expansion.
- ISSUE-189 and PLAN-ISSUE-189 cover only the separate 200%-text overflow found
  on an existing restoration branch. They do not own the canonical direction,
  content sequence, wordmark, composability modes, or navigation-restoration
  outcome in this issue and remain untouched.

## Resolution

Landed on `main` in PR #29 (`cd453e8c1c31aeb210979013345d4931d789b378`, 2026-08-29),
"Restore canonical homepage direction". Follow-on PR #33 (`e412f1f`) retargeted
CAP-004 to the canonical homepage section. Closes via `resolved-by` on the
homepage restore merge. `PLAN-ISSUE-190` remains `status: draft` on disk.

### Existence-in-world check

Before filing, `issues/` and `bundles/` were searched for homepage, canonical,
field-guide, wordmark, composability, and navigation ownership. BUNDLE-032 and
its implemented Seed 4 spec own the broader public-site feature and existing
contracts; this issue is the focused reactive correction of drift within that
delivered surface. The only open homepage issue found was untracked ISSUE-189,
whose problem and plan are limited to a concrete actual-200%-text overflow and
explicitly do not own page content or direction. No existing issue owns this
reconciliation.
