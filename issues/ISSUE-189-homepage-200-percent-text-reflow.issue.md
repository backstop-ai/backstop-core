---
title: "Designed Homepage Overflows at 200 Percent Text"
schema_version: issue/v1

issue:
  id: ISSUE-189
  title: "Designed Homepage Overflows at 200 Percent Text"
  type: bug
  status: open
  created: "2026-08-27"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# Designed Homepage Overflows at 200 Percent Text

## Problem

PR #28 on `fix/restore-designed-homepage` restores the designed homepage while
preserving the implemented BUNDLE-032 Seed 4 accessibility, browser, and design
contracts. At head `9ad0d36`, the full CI site verifier passes 39 of 40
Playwright tests. The only failure is
`200 percent text relayout › / reflows at actual 200 percent root text` at the
360px viewport, where the document overflows horizontally by 185px.

CI diagnostics identify `.checks` as the overflowing element, with width and
minimum width of 510px. `docs/assets/css/backstop.css:143` defines
`.checks, .verdict { min-width: 510px; }`. The responsive rules in
`docs/assets/css/site.css` reset individual `.check` and `.baseline-row`
elements, but do not reset the `.checks` container. The restored homepage
therefore violates SPEC-075's existing actual-200%-text relayout and
no-horizontal-document-overflow contract.

## Solution

Restore homepage reflow under the existing accessibility, browser, and design
contracts. Keep the correction focused on the responsive homepage integration,
likely in `docs/assets/css/site.css`; do not alter the shared design-system
source rule in `docs/assets/css/backstop.css`, page content, or Seed 4 contract
artifacts unless new evidence proves that necessary. Do not weaken, skip, or
replace the existing verifier assertions.

## Verification

Use the existing full public-site verifier and its current Playwright coverage.
The previously failing actual-200%-root-text case for `/` at 360px must pass
with document overflow within the existing SPEC-075 limit, while the other 39
browser tests and the restored homepage design remain passing.

## References

- PR #28, branch `fix/restore-designed-homepage`, head `9ad0d36`.
- `docs/assets/css/backstop.css:143` — shared `.checks, .verdict` minimum width.
- `docs/assets/css/site.css` — existing responsive integration overrides.
- `specs/SPEC-075-static-public-site-design-system.spec.md`, REQ-003.
- BUNDLE-032 Seed 4 owns the delivered site contracts; this issue records the
  concrete regression exposed while restoring the designed homepage rather
  than redefining that feature scope.
