---
title: "CAP-014 Deployed Journey Fails Because bindClaims Steals the Denial Paragraph Closer"
schema_version: issue/v1

issue:
  id: ISSUE-202
  title: "CAP-014 Deployed Journey Fails Because bindClaims Steals the Denial Paragraph Closer"
  type: bug
  status: ready
  created: "2026-09-09"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "./bin/backstop gate --all"

implementation:
  summary: >
    In bindClaims, when kramdown nested `<!-- /backstop-claim -->` inside the last
    paragraph and the closer needle was extended to include that `</p>`, replace the
    needle with `</p>` plus the semantic wrapper (`</aside>` or `</article>`) instead
    of the wrapper alone. That leaves the adjacent-guidance guarantee-denial paragraph
    a closed element so websitejourney firstElementText can read nonempty visible
    bytes. Do not change Pages deploy blocking, the extractor regex, sitecheck, or
    visitor copy.
  package: scripts/render-public-site-contracts

requirements:
  - id: REQ-001
    text: >
      When bindClaims wraps an adjacent-guidance claim whose kramdown-rendered closer
      is `<!-- /backstop-claim --></p>`, the last paragraph that it retags as
      `data-boundary-guarantee-denial` must remain a closed HTML element immediately
      before `</aside>`. Replacing that closer needle with bare `</aside>` is
      prohibited: it steals the denial paragraph's `</p>` and emits the live
      BOUNDARY-005 shape whose denial text is not followed by `<`.
  - id: REQ-002
    text: >
      The rendered adjacent-guidance denial must be extractable as nonempty visible
      bytes by the existing websitejourney contract
      `(?s)[^<]*data-boundary-guarantee-denial[^>]*>([^<]*)<`. On the existing
      render fixture, that extraction must yield `Denial bytes.` AssertCAP014DualIdentity
      / firstElementText must not see empty denial bytes on that closed markup. This
      issue does not loosen, rewrite, or relocate firstElementText.
  - id: REQ-003
    text: >
      The same closer restoration applies to every bindClaims wrapper that extended
      closingNeedle to include a following `</p>`: evidence cards must emit
      `</p></article>` rather than an unclosed `<p>` inside `<article>`, and
      non-continuation boundary asides must emit `</p></aside>`. Existing
      TestRenderPublicSiteContracts_FullFixturePasses must still pass and must still
      reject leftover claim markers, unresolved SITE-COMMIT, and the inverted
      `</article></p>` wrapper.
  - id: REQ-004
    text: >
      Scope is the renderer close-tag substitution only. Do not change
      `.github/workflows/pages.yml` to fail deploy on CAP-014, do not change
      `scripts/websitejourney/html.go` firstElementText to accept unclosed elements,
      do not change sitecheck's dual-identity regex, and do not edit `docs/status.md`
      or product-model denial copy. The product fix is valid HTML from bindClaims.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      Rendering the existing adjacent-guidance fixture emits a closed denial
      paragraph immediately before the callout closer:
      `<p data-boundary-guarantee-denial>Denial bytes.</p></aside>`.
    tests:
      - TestRenderPublicSiteContracts_ClosesAdjacentGuidanceDenialParagraph
  - id: CLM-002
    requirement: REQ-002
    text: >
      Applying the existing firstElementText pattern to the rendered BOUNDARY-005
      aside interior yields exactly `Denial bytes.` The unclosed live shape (denial
      text with no following `<` before `</aside>`) is not treated as success.
    tests:
      - TestRenderPublicSiteContracts_ClosesAdjacentGuidanceDenialParagraph
  - id: CLM-003
    requirement: REQ-003
    text: >
      Evidence-card and non-continuation boundary bindings that consumed a
      kramdown-wrapped `</p>` restore that closer inside the semantic wrapper, and
      TestRenderPublicSiteContracts_FullFixturePasses still forbids leftover markers
      and `</article></p>`.
    tests:
      - TestRenderPublicSiteContracts_PreservesKramdownParagraphCloserInsideSemanticWrapper
      - TestRenderPublicSiteContracts_FullFixturePasses
  - id: CLM-004
    requirement: REQ-004
    text: >
      The existing fixture's visible denial and explanation bytes stay
      `Denial bytes.` and `Adjacent bytes.`; the tests live in the renderer package
      and do not rewrite websitejourney, sitecheck, Pages, or status copy.
    tests:
      - TestRenderPublicSiteContracts_ClosesAdjacentGuidanceDenialParagraph
      - TestRenderPublicSiteContracts_PreservesKramdownParagraphCloserInsideSemanticWrapper

contracts:
  - file: scripts/render-public-site-contracts/main.go
    provides:
      - name: bindClaims
        kind: function
        signature: "func bindClaims(doc string, claims []evidenceClaim, boundaries map[string]boundary, route string) (string, error)"
    consumes:
      - source: scripts/render-public-site-contracts/main.go
        name: replaceOnce
        kind: function
---

# CAP-014 Deployed Journey Fails Because bindClaims Steals the Denial Paragraph Closer

## Problem

GitHub Pages job "Verify authoritative deployment identity" fails on every `main`
push since 2026-08-30 with:

```
websitejourney: CAP-014/@UJ-001: deployed journey failed: JLINK-024 BOUNDARY-005 visible bytes are empty
```

Build and Deploy still pass. The site publishes. SPEC-076 is `implemented` while this
production acceptance stays red. The live origin
`https://backstop.sh/status/` (fetched 2026-09-09) renders BOUNDARY-005 as:

```html
<aside data-boundary-callout data-boundary-id="BOUNDARY-005" data-boundary-state="adjacent-guidance">
<p data-boundary-explanation>Backstop stops at an inspectable verdict because external orchestration and organizational enforcement have different owners.</p>
<p><a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">Continue outside Backstop</a></p>
<p data-boundary-guarantee-denial>That continuation is guidance, not a guarantee provided by Backstop.
</aside>
```

The denial paragraph is unclosed. `FindRenderedBoundary` InnerHTML is the aside
interior (no `</aside>`). `firstElementText` in `scripts/websitejourney/html.go`
requires a following `<` after the text node:

```
(?s)[^<]*` + attr + `[^>]*>([^<]*)<
```

Unclosed denial has no following `<`, so `Denial == ""`. `AssertCAP014DualIdentity`
then errors "visible bytes are empty".

The product-model denial string is correct. The defect is the renderer. In
`scripts/render-public-site-contracts/main.go` `bindClaims`:

1. If `<!-- /backstop-claim -->` is followed by `</p>` (kramdown wraps the closer
   inside the last paragraph), it extends `closingNeedle` to include that `</p>`.
2. For continuation boundaries it retags the last `<p>` as
   `<p data-boundary-guarantee-denial>`.
3. It then `replaceOnce`s `closingNeedle` with `</aside>`, stealing the denial
   paragraph's closer.

The existing fixture in `scripts/render-public-site-contracts/main_test.go` already
has this kramdown shape (`<p>Denial bytes.\n<!-- /backstop-claim --></p>`) but does
not assert a closed denial tag, so the suite stays green. Synthetic CAP-014 trees
in `WriteAcceptedBuiltTree` emit well-formed `</p></aside>`, so built-journey tests
do not catch production HTML. Sitecheck's dual-identity regex does not require
`</p>`, so Seed 4 can pass while Seed 5 deployed CAP-014 is red.

## Solution

When `closingNeedle` was extended to include `</p>`, replace it with `</p>` plus
the semantic wrapper (`</p></aside>` or `</p></article>`), not the wrapper alone.
That keeps the last paragraph closed. Adjacent-guidance denial then has a following
`<` for `firstElementText`. Evidence cards and non-continuation asides get the same
closer restoration because they share the needle-extension path.

Do not fail Pages deploy on this check unless the founder rules it. Do not make
`firstElementText` accept unclosed elements as the product fix. Do not change
visitor copy.

## Verification

`TestRenderPublicSiteContracts_ClosesAdjacentGuidanceDenialParagraph` must fail on
the current renderer (unclosed denial) and pass after the closer is restored,
asserting `<p data-boundary-guarantee-denial>Denial bytes.</p></aside>` and that
the firstElementText pattern extracts `Denial bytes.` from the aside interior.

`TestRenderPublicSiteContracts_PreservesKramdownParagraphCloserInsideSemanticWrapper`
must assert evidence-card `</p></article>` and non-continuation `</p></aside>` on
the same fixture.

`TestRenderPublicSiteContracts_FullFixturePasses` remains green and still rejects
`</article></p>`.

Authoritative acceptance is `./bin/backstop gate --all`.

## References

- Live HTML: `https://backstop.sh/status/` BOUNDARY-005, 2026-09-09.
- `scripts/render-public-site-contracts/main.go` — `bindClaims` closer substitution.
- `scripts/websitejourney/html.go` — `firstElementText`.
- `scripts/websitejourney/traversal.go` — `AssertCAP014DualIdentity`.
- `specs/SPEC-076-end-to-end-website-capabilities.spec.md` — CAP-014/@UJ-001
  conjunctive identity; implemented while production deployed acceptance is red.
- `.github/workflows/pages.yml` — post-deploy
  `verify-website-capabilities.sh --deployed-origin https://backstop.sh`.
- DIR-035 — public-site home; this issue slots into `directive.source`.

## Existence-in-world Check

Searched `issues/` and `bundles/` for CAP-014, JLINK-024, BOUNDARY-005,
guarantee-denial, `data-boundary-guarantee-denial`, and bindClaims closer theft.
No open issue owns this renderer defect. SPEC-076 owns the capability contract
and is `implemented`; it does not own this newly observed production HTML bug.
BUNDLE-032 chartered the website expansion and does not name this closer-theft.
ISSUE-191 and ISSUE-192 are sitecheck-matrix / architecture-pack residuals, not
this binding. ISSUE-201 is a reserved tag with no file (burnt ID, not this work).
