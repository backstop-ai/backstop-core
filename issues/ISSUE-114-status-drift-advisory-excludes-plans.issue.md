---
title: "Status Drift Advisory Never Fires For Plans — test_names Is Optional, Manual, and Decoupled From The Prose Mandated-Test List Plan Authors Already Write"
schema_version: issue/v1

issue:
  id: ISSUE-114
  title: "Status Drift Advisory Never Fires For Plans — test_names Is Optional, Manual, and Decoupled From The Prose Mandated-Test List Plan Authors Already Write"
  type: bug
  status: closed
  created: "2026-08-02"
  closed: "2026-08-17"

delivered_by: PLAN-ISSUE-114
---

# Status Drift Advisory Never Fires For Plans

## Resolution

Delivered by `PLAN-ISSUE-114` (`status: completed`). The fix gives plans a second,
claims-derived mandated-test channel instead of relying solely on the optional, manually-populated
`test_names:` field this issue diagnosed as empirically dead for every non-terminal plan.

**Fix:** `pkg/gate/artifact_status.go` now builds a `buildSourceClaimIndex` over each plan's source
spec's `claims[].tests` mapping, and `planClaimDerivedMandatedTests` folds that index in via each
task's `Claims` refs — before the plan walk populates `MandatedTests`. `unionPlanMandatedTests`
merges the derived names with any explicit `task_test_names` without duplicates, so a plan whose
tasks cite spec claims now surfaces the same mandated-test vocabulary the spec itself already
enforces, without requiring authors to hand-copy names into `test_names:`. `artifacts/plan/v1/
schema.json`'s `task_test_names_description` was updated to document both channels.

**Verification:** falsified via targeted mutation (not solely the plan's own predicted reds, since
2 of the 6 predicted mutations were structurally non-reddable pre-fix) and via a real
control-vs-treatment measurement on this repo's own tree: `artifact_status_drift_advisory`
findings went from 3 to 18 (net +15, matching the plan's own prediction — plans with claim-derived
mandated tests are now visible to the dimension this issue's Problem section described as
structurally blind to them), while BLOCK-severity violations stayed 0 to 0 (the stop-and-report
condition never triggered on this tree).

## Problem

`./bin/backstop gate --all --json` flags the `artifact_status_drift_advisory` step on 5 specs
(SPEC-018, SPEC-019, SPEC-030, SPEC-036, SPEC-037 — each non-terminal with all mandated tests
PRESENT, "it looks delivered") but **zero plans**, even though `PLAN-ISSUE-048` is independently
known to be `status: draft` while its code demonstrably shipped in the same commit that authored
the plan — exactly the shape this advisory exists to catch.

**This is not a categorical exclusion of plans from the classifier.** `KindPlan` is a first-class
type throughout the pipeline: `ResolveArtifactStatus` walks `plans/*.plan.yml`
(`pkg/gate/artifact_status.go:241-257`), `ClassifyArtifactStatus` resolves plan statuses
type-aware (`draft`/`ready`/`implementing` -> `ClassNonTerminal`, `completed` ->
`ClassSuccessTerminal`, `artifact_status.go:103-106`), and `ClassifyStatusDrift`
(`pkg/gate/status_drift.go:31-71`) iterates whatever records it's handed with no kind filter at
all. A plan record reaches the classifier exactly like a spec or issue record does.

**The actual root cause is `looksDelivered`'s empty-set gate combined with an empirically-dead
data channel.** `looksDelivered` (`pkg/gate/status_drift.go:75-85`) requires
`len(rec.MandatedTests) > 0` before it will ever warn:

```go
func looksDelivered(rec ArtifactStatusRecord, present map[string]bool) bool {
    if len(rec.MandatedTests) == 0 {
        return false
    }
    ...
}
```

For specs and issues, `MandatedTests` comes from the claim->tests walk over a structured
`claims[].tests` array (`ExtractMandatedTests`, `claimsToMandatedTests`,
`artifact_status.go:262-278`) — a field that is load-bearing from `ready` onward per this repo's
own status-gated requirements, so it is populated for essentially every live spec/issue.

For plans, `MandatedTests` comes from exactly one source: the OPTIONAL, per-task `test_names`
field (`planTaskMandatedTests`, `artifact_status.go:345-363`), introduced by `PLAN-ISSUE-048`
itself (`task_optional_keys: ["test_names"]`, `artifacts/plan/v1/schema.json:44`). Nothing in
`pkg/validate/plan.go` reads, requires, or even acknowledges `test_names` — it is documentary at
the schema level and mechanically optional. Plans have no structured top-level `claims[]` array
the way specs/issues do; a plan's mandated-test intent lives in task `description:` prose
(conventionally "mandated test names (exact): ...") and is only machine-visible if an author
separately, manually, redundantly copies those same names into the `test_names:` YAML key.

**Measured against the real corpus (2026-08-02):** of 98 plans, 48 are non-terminal
(`draft`/`ready`). **Zero** of those 48 have any task-level `test_names` populated. 28 plans do
have `test_names` populated somewhere in the file — but every single one of those is already
`status: completed`, meaning it already routes through `ClassSuccessTerminal`'s BLOCK check
(broken-promise-if-absent), never through the `ClassNonTerminal` WARN/advisory path this issue is
about. In other words: **the only lever that can make `looksDelivered` true for a plan is a field
that, in practice, gets populated only after (or as part of) marking the plan complete** — a
chicken-and-egg gap that makes the advisory structurally unable to ever fire for a plan still
in `draft`/`ready`, regardless of how obviously delivered it looks.

**`PLAN-ISSUE-048` is the concrete, almost ironic instance.** It is `status: draft`
(confirmed: `plans/PLAN-ISSUE-048-obsoleted-resolvedby-vocab.plan.yml:6`), yet its own tasks'
`description:` prose explicitly enumerates "mandated test names (exact)" — e.g.
`TestObsoleted_RequiresObsoletedBy`, `TestResolvedBy_ValidCloseWithoutOwnClaimsOrPlan`,
`TestClassifyArtifactStatus_ObsoletedIsRetiredTerminal`,
`TestResolveArtifactStatus_ParsesPlanTaskTestNames` — and every one of those functions exists in
the codebase today (`pkg/validate/obsoleted_test.go:63`, `pkg/validate/resolved_by_test.go:36`,
`pkg/gate/artifact_status_obsoleted_test.go:41`, `pkg/gate/status_drift_plantests_test.go:16`),
alongside the shipped `resolved-by`/`obsoleted` machinery those tests exercise
(`pkg/validate/resolved_by.go`, `pkg/validate/terminal.go:18-43`). None of TASK-001 through
TASK-012's tasks in the plan populate the structured `test_names:` field for any task — the exact
mandated names are present only as unstructured prose. The plan itself even predicted this in its
own Phase 6 verification note: *"no real plan carries test_names yet, so this is additive"*
(`PLAN-ISSUE-048-obsoleted-resolvedby-vocab.plan.yml:715-718`) — a prediction that has held true
for every non-terminal plan since, including this one.

## Impact

The delivered-but-not-marked-complete advisory — the exact drift class `PLAN-ISSUE-048` was built
to surface for `completed` plans and that `ISSUE-042`/`ISSUE-048` extended the resolver to cover —
has never once fired for a plan in this repo's history, because the one signal it depends on
(`test_names`) is optional, unenforced, and in practice only ever gets written once a plan is
already terminal. Every non-terminal plan is invisible to this dimension no matter how complete
its shipped code is. This is a design gap wearing a bug's clothes: the mechanism exists and is
wired correctly, but the data it depends on is never populated at the point the check would be
useful.

## Notes / references

- Motivating instance: `PLAN-ISSUE-048` (`status: draft`), delivered in the same commit that
  authored the plan; its shipped tests are named in this issue's Problem section.
- `ISSUE-098` (drift-resolver-blind-to-pack-claim-ids, closed) is a sibling in the same step
  family — a different facet (pack-declared claim presence vs. Go-test-function presence) of
  `computeDriftSurfaces`'s existence resolution, not a duplicate of this gap.
- Likely home: `DIR-024` (gate-engine-quality) per this step family's existing territory —
  not self-homed here per convention.
- Fix direction (not prescribed): either make `test_names` load-bearing at plan-authoring time
  (e.g. the planner agent/plan schema requires it alongside the existing prose convention, so the
  two representations can't drift apart), or give plans a task-claims-derived mandated-test
  concept structurally parallel to spec/issue `claims[].tests` instead of a free-floating optional
  field. Either direction should close the gap that a plan's prose-declared mandated tests and its
  machine-readable `MandatedTests` are two independent, unsynchronized representations today.
