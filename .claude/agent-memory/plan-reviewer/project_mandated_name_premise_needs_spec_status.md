---
name: mandated-name-premise-needs-spec-status
description: A plan claiming "renaming this test reds artifact_status_drift" must be checked against the owning spec's STATUS — the gate is implemented-only and excludes retired-terminal specs
metadata:
  type: project
---

When a plan says "test name X is MANDATED by a shipped spec, so renaming it turns
artifact_status_drift RED as a broken promise", go read the owning spec's
`status:` field before accepting it. The enforcement is status-scoped twice over:

- `pkg/gate/status_drift.go:12` — a RETIRED-TERMINAL artifact (`replaced` /
  `canceled` / `deprecated`) is EXCLUDED, no violation. Its mandated names are free.
- `pkg/gate/step_testverify.go:348-353` (`filterDueMandatedTests`) — `test_verification`
  applies an implemented-only scope (ISSUE-054). A `draft` or
  `ready-for-implementation` spec's mandated tests are NOT live promises.

So only an `implemented` spec's mandated names are actually load-bearing.

**Why:** PLAN-ISSUE-112 (2026-08-16 review) built its self-described "MOST IMPORTANT
FACT" on four names it said were mandated by `implemented` specs. SPEC-034 is
`replaced` (a TOMBSTONE that says "DO NOT pick this spec up as live") and SPEC-031
is `draft`. Neither enforces. Three downstream harms followed from the unchecked
premise: docstrings the plan MANDATED would have baked a false claim into source; a
dispatcher task was queued to route spec-author at a terminal artifact; and CLM-007
existed only to serve the premise.

**How to apply:** grep the spec file's frontmatter `status:` (line ~5) for every
name a plan calls mandated. `replaced`/`canceled`/`deprecated` → excluded outright.
`draft`/`ready-for-implementation` → not due. Only `implemented` → real promise.
Keeping a name is still usually the safe call, but the plan must not state a false
enforcement mechanism as its justification, and must not send spec-author to amend
a retired spec. Related: [[repurposed-test-claim-text-drift]],
[[coverage-rewrite-predating-spec-drift]].
