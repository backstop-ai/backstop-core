---
name: baseline-pull-family
description: baseline-pull defects home DIR-003 (not DIR-024) on defect-site; three workflows exist but two are tag-triggered; the run-selection test fixture says "ci" while the real workflow is "CI"
metadata:
  type: project
---

`backstop baseline pull` defects in **production** `cmd/backstop/baseline.go` home under
`DIR-003 "Baseline Implementation"` — its Description names the command and "GitHub Actions
artifact publishing" verbatim, and it already owns ISSUE-120 (the open founder ruling on
GitHub-Actions knowledge baked into that exact file). Defects in the baseline **test harness**
(e.g. `bun_ratchet_flip_test.go`) are the DIR-024 side of the same line — see
[[project_baseline_committed_premise]]. Applying one defect-site test to both gives opposite,
consistent answers (ISSUE-176 → DIR-024 recommended; ISSUE-178 → DIR-003 slotted 2026-08-19).

**Why:** DIR-024's charter test is a loud red or a repo test-harness false verdict. A latent
production-code defect is neither, so DIR-024 fails its own test and DIR-003 is the sole
plausible home — clear fit, not ambiguous, even when provenance points at a DIR-024 lane.
Provenance was unusable here anyway: ISSUE-176's own home is still unruled.

**How to apply:** three measured facts that keep getting mis-stated in filings —

1. `.github/workflows/` holds **three** workflows: `CI` (ci.yml), `release`, `tag-integrity`.
   Any filing claiming "only one workflow runs against main" is wrong as written. The true
   reason `?branch=main` returns only CI rows is that `release` and `tag-integrity` are
   `push: tags: ['v*']`, so their runs carry the TAG as `head_branch`. The thing to watch for
   is the first **branch**-triggered second workflow, not the first second workflow.
2. `cmd/backstop/baseline_test.go:248`'s fake-`gh` fixture emits `"name":"ci"` **lowercase**
   while `ci.yml` declares `name: CI`. Any workflow-name filter using the real literal goes
   RED at `baseline_test.go:267` on arrival. The fixture also violates
   fixtures-from-real-output by that same mismatch.
3. `resolveBaselineCache` runs the self-healing pull inside **every** `backstop gate`
   (`gate.go:208,331`), so pull defects surface as generic gate failures, not as
   `baseline pull` failures — a misdiagnosis trap, which is why they're worth filing while
   still latent.

DIR-003's founder hold (reaffirmed 2026-08-10) is scoped to the **coverage-baseline refresh**
only, and the ISSUE-086 gate is on directive **completion** — neither blocks citing a new
source into it. Don't over-read the hold as "DIR-003 is frozen."

Any fix that adds a workflow-name literal enlarges ISSUE-120's surface (a fourth
GitHub-Actions-specific literal in one file); the alternative "pick the run whose artifacts
contain `backstop-baseline-v1`" scan has no such coupling. Recommend planning ISSUE-120 and
ISSUE-178 as one lane.
