---
name: grandfathering-decides-the-fix-direction
description: When an issue offers two fix directions, read backstop.yml's `applies-to` for the dimension each would fire under — a dimension without `applies-to: new-code` does NOT grandfather, so a new error-severity rule there reds the whole corpus at once
metadata:
  type: project
---

When an issue names two candidate fix directions ("make field X required" vs "derive X
from data that already exists"), the deciding evidence is usually not elegance — it is
**which gate dimension each direction's findings land under, and whether that dimension
grandfathers**. Read `backstop.yml`'s policy block before choosing.

- `applies-to: new-code` on a dimension ⇒ pre-existing findings grandfather against
  `.backstop/baseline.json`; only net-new ones block. A corpus-wide new rule is survivable.
- **No `applies-to` key** ⇒ every finding blocks unconditionally, forever. Observed on
  `artifact_validation` (2026-08-17), while its sibling `artifact_status_drift` DOES carry
  `applies-to: new-code`. Two dimensions in the same file, opposite survivability.

**Why:** ISSUE-114 (drift advisory structurally unable to fire for plans) offered exactly
this pair. Direction (a), requiring `test_names` in `pkg/validate/plan.go`, fires under
`artifact_validation` — which does not grandfather — and 52 of 54 non-terminal plans lack
the field, so it would have taken `gate --all` hard red mid-launch with no baseline relief,
plus 23 `completed` plans that convention forbids rewriting. Direction (b), deriving a
plan's mandated tests from its tasks' already-required `claims[]` against the source
artifact's `claims[].tests`, needed zero backfill. A probe over all 127 plans measured the
adopted direction at **0 new blocking findings and 15 newly-firing advisories** — the
number that made the choice defensible rather than tasteful.

**How to apply:** before committing to a direction, (1) find the dimension its violations
would carry, (2) grep `backstop.yml` for that dimension's `applies-to`, (3) count the
existing corpus violations the rule would create, and (4) check whether any land on
artifact classes the repo treats as immutable (completed plans are never rewritten — see
ISSUE-149). Put the measured counts in the plan's `notes` and record the REJECTED direction
with its numbers, so a reviewer can check the reasoning instead of re-litigating it. Also
prefer a direction that reads fields the corpus already has: "fully retroactive, no backfill"
is a stronger property than any amount of enforcement rigor.

Related: [[measure-fp-surface-before-designing-a-detector]] (measure before designing),
[[verify-issue-premises]] (the issue's own corpus numbers are claims — ISSUE-114's were
2026-08-02 stale and had partly moved), [[structural-dimensions-nonwaivable]].
