---
title: "Fixture Path Filter Diagnostics"
schema_version: issue/v1

issue:
  id: ISSUE-184
  title: "Fixture Path Filter Diagnostics"
  type: bug
  status: open
  created: "2026-08-22"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Fixture Path Filter Diagnostics

## Problem

`backstop pack test` reports `negative fixture not triggered` when a fixture is
silently excluded by the rule's own path filters. The result is technically
correct but does not distinguish a dead rule pattern from a fixture that never
entered the rule's scope.

Observed while authoring `backstop-design-system`:

- Production rules correctly scoped themselves to `backstop.css` and
  `index.html`.
- Negative fixtures used descriptive names such as
  `backstop-raw-color.css` and `index-inline-style.html`.
- All five claims failed only as `negative fixture not triggered` because the
  basenames did not match the rules' `paths.include` filters.
- Widening the filters to make the fixtures execute caused the generated site's
  token source, `backstop-tokens.css`, to enter scope.
- `pack test` then passed, but the real gate caught 28 integration violations
  because the rule was now enforcing against the one file intentionally
  allowed to declare raw colors.

The gate did its job. The authoring feedback did not make the causal distinction
legible early enough. A false-negative pattern and a path-filtered fixture need
different diagnostics because their remedies are opposites: fix the pattern
versus fix fixture path fidelity or production scoping.

## Expected Behavior

Phase 3 records whether each fixture was dispatched after include/exclude
evaluation. If a fixture is outside the rule's effective path scope, pack test
fails with a diagnostic naming the fixture, rule, and filter responsible. A
recipe author should also have a deterministic way to give a fixture its
production basename/path while retaining polarity organization.

## Acceptance Evidence

- A negative fixture excluded by `paths.include` reports `fixture filtered from
  rule scope`, not `negative fixture not triggered`.
- A fixture excluded by `paths.exclude` identifies the matching exclusion.
- A dispatched negative fixture whose pattern finds nothing retains the current
  not-triggered diagnostic.
- Positive fixtures receive the same path-fidelity accounting.
- Tests cover basename-only filters under both explicit-file and directory
  dispatch shapes.
- Documentation shows a fixture layout that preserves production-relative paths
  without requiring unique basenames to widen production policy.

## Existence-in-world Check

Searched the current Backstop Core issue/artifact corpus and repository source
for fixture path filtering and `negative fixture not triggered` before filing.
No existing issue owns the diagnostic distinction described here.
