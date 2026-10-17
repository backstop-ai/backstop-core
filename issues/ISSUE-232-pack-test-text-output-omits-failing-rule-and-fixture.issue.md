---
title: "Pack Test Text Output Omits Failing Rule And Fixture"
schema_version: issue/v1

issue:
  id: ISSUE-232
  title: "Pack Test Text Output Omits Failing Rule And Fixture"
  type: bug
  status: open
  created: "2026-10-17"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Pack Test Text Output Omits Failing Rule And Fixture

## Problem

`backstop pack test` in text mode does not say which rule or fixture failed, so a pack author with many
rules has to bisect.

Verified at origin/main:

- `ValidationError` (`pkg/packval/result.go:22-30`) carries `Phase`, `Check`, `Rule`, `Claim`, `Message`,
  `FixHint` and `ManifestPath`. Phase 3 populates `Rule` and `Claim` on its fixture errors
  (`pkg/packval/phase3.go`, e.g. the `validator-positive` and `semgrep-negative` errors).
- The text formatter (`pkg/packval/format.go:27-29`) prints only
  `ERROR [<phase>/<check>] <message>`. `Rule`, `Claim`, `FixHint` and `ManifestPath` are dropped, so two
  failing rules produce indistinguishable lines such as
  `ERROR [phase3-fixtures/validator-positive] layer3 positive failed`.
- The JSON output (`--json` or `--format json`) includes the rule and claim fields, so the information
  exists and only the text path loses it.
- Separately, the validator-seam messages in phase 3 do not name the fixture path at all (the fixture
  `f.Path` is passed to the executor but not placed on the `ValidationError`), so even JSON cannot name
  the fixture for those checks. Engine-seam errors built by `engineError` were not inspected.

## Impact

Slow authoring loop; the failing rule is known to the tool and withheld from the default output.

## Direction

(Direction, not a plan.) Print rule and claim in the text line (and the fix hint when present); add the
fixture path to phase-3 `ValidationError`s so both formats can name it.

## Notes

- Related: ISSUE-216 and ISSUE-218 (phase 3 reports a sandbox denial as a bare failure); this issue is
  about naming what failed, those are about explaining why.
