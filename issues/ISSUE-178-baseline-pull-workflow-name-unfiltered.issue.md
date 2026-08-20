---
title: "Baseline Pull Workflow Name Unfiltered"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-178

issue:
  id: ISSUE-178
  title: "Baseline Pull Workflow Name Unfiltered"
  type: bug
  status: closed
  created: "2026-08-18"
  closed: "2026-08-20"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# Baseline Pull Workflow Name Unfiltered

## Problem

`resolveLatestSuccessfulMainRun` in `cmd/backstop/baseline.go:202-224` decodes each candidate
workflow run's `Name` field into its local `payload.WorkflowRuns[].Name` struct field but never
reads or filters on it. The function's actual selection loop is:

```go
for _, run := range payload.WorkflowRuns {
    if run.HeadBranch == "main" && run.Conclusion == "success" {
        return run.ID, nil
    }
}
```

It queries `repos/{repo}/actions/runs?branch=main&status=success&per_page=20` — which is already
scoped to `main` and to `status=success` server-side — decodes `Name`, and then filters only on
`HeadBranch == "main"` (redundant given the query parameter) and `Conclusion == "success"`
(likewise redundant given `status=success`). Nothing compares `run.Name` against the `CI`
workflow — the one whose `baseline` job publishes the `backstop-baseline-v1` artifact that
`runBaselinePull` (`cmd/backstop/baseline.go:114-148`) goes on to require via
`resolveBaselineArtifactID` (`cmd/backstop/baseline.go:226-246`). The function returns the newest
successful run on `main` from **any** workflow that reports there, not specifically `CI`.

## Why this is latent, not firing, today

This repository currently runs exactly one workflow against `main`: `CI`
(`.github/workflows/ci.yml`). Measured directly during `PLAN-ISSUE-176`'s evidence-gathering
(2026-08-18, `gh api "repos/backstop-ai/backstop-core/actions/runs?branch=main&status=success&per_page=20"`):
all 20 returned rows are workflow `CI`. So `resolveLatestSuccessfulMainRun`'s missing filter has
no visible effect today — the unfiltered "newest successful run on main" and the correctly-filtered
"newest successful CI run on main" happen to be the identical answer, for every one of the 20 most
recent successful runs.

## Impact — the dormant landmine

The moment a second workflow ever runs against `main` in this repository — a release workflow, a
docs build, a security-scan workflow, anything that reports a successful conclusion on the `main`
branch — `resolveLatestSuccessfulMainRun` can select THAT workflow's run instead of `CI`'s, purely
based on which one is newest. That other workflow will not have published a `backstop-baseline-v1`
artifact, so `resolveBaselineArtifactID` fails with `missing artifact: "backstop-baseline-v1" not
found in selected run`, and `backstop baseline pull` — including the self-healing pull that
`resolveBaselineCache` runs unconditionally inside every `backstop gate` invocation
(`cmd/backstop/gate.go:208,331`) — starts failing for a reason that has nothing to do with
authorization, retention, or the CI wiring `ISSUE-176`/`PLAN-ISSUE-176` fixed, and everything to do
with which workflow happened to run most recently. Anyone diagnosing that failure without knowing
this function ignores `Name` would have no reason to suspect a workflow-selection bug — the error
message names a missing artifact, not a wrong run.

## Root cause

The `Name` field is decoded into the anonymous struct at `cmd/backstop/baseline.go:207-213` but
the selection loop at `cmd/backstop/baseline.go:218-222` never reads it. This reads as an
incomplete implementation rather than a deliberate simplification — decoding a field purely to
discard it is itself a signal that a filter on it was intended and dropped.

## Solution

Add `run.Name == "CI"` (or, more robustly, a named constant shared with wherever else this
codebase might need to identify the workflow that publishes `backstop-baseline-v1` — currently
nowhere else does) to the selection loop's condition:

```go
for _, run := range payload.WorkflowRuns {
    if run.HeadBranch == "main" && run.Conclusion == "success" && run.Name == ciWorkflowName {
        return run.ID, nil
    }
}
```

Baking the literal `"CI"` into `cmd/backstop` would itself be a workflow-identity assumption worth
naming explicitly in review — this repo's own `.github/workflows/ci.yml` names its workflow `CI`
today, but a hardcoded match is coupling `backstop-core`'s own CLI to its own workflow's name
rather than to any general contract a pack or another repo could redeclare. Whoever picks this up
should weigh a literal-name match against a more general "the run whose artifacts include
`backstop-baseline-v1`" selection strategy (scanning candidate runs' artifacts in ID order until
one matches, rather than trusting `Name`) — the latter is more work but has no hidden workflow-name
coupling at all. Not prescribing the shape here; flagging the trade-off for whoever implements it.

Add a regression test asserting the selection loop rejects a synthetic run whose `Name` is not the
expected workflow even when it is the newest successful run on `main` in the fixture payload — the
current test suite (if any covers this function) should be checked for whether it already
constructs multi-workflow fixtures; if not, this is the first one to.

## Resolution

Fixed by `PLAN-ISSUE-178` (`status: completed`), landing in `cmd/backstop/baseline.go`:
`resolveLatestSuccessfulMainRun`'s selection loop now reads the `Name` field it was already
decoding and discarding, filtering on a package-level `const ciWorkflowName = "CI"`, pinned by a
test that reads `.github/workflows/ci.yml`'s own declared `name:` so a future rename reddens that
test rather than silently breaking `baseline pull`. The candidate window widened `per_page=20` →
`per_page=100`, and a selection miss now names the workflow it looked for and the foreign
workflows it rejected instead of surfacing downstream as a misleading `missing artifact` error.
Four pre-existing test fixtures across `cmd/backstop/baseline_test.go` and `cmd/backstop/
gate_test.go` were corrected from a fabricated lowercase `"ci"` to the real API's `"CI"` — they
had never been captured from real output.

**This issue's own "dormant" framing turned out to be exactly right, and briefly false.** Filed
2026-08-18 describing a landmine with no visible effect because this repo ran only one workflow
against `main`. It went live on 2026-08-20, the same day this plan was written and implemented,
when GitHub Pages was enabled on this repository — Pages runs a second, unfiled, always-fast
workflow (`pages build and deployment`) that started winning the "newest successful run on main"
race against `CI` within two days of filing.

Proven end-to-end, live, in both directions against the real repository the same night: the fixed
binary selected real `CI` run `32320833720` and pulled a valid 332KB `.backstop/baseline.json`
(exit 0); a pre-fix binary reproduced the live failure verbatim, selecting the Pages run
`32398480027` and failing with `missing artifact: "backstop-baseline-v1" not found in selected
run` (exit 2). Test substantiveness independently confirmed via 7 targeted mutations in a scratch
copy, each red exactly where predicted. A real `backstop gate` run found zero blocking violations
attributable to this fix; the one pre-existing failure encountered (`TestPackAuthoringLoop_
EndToEnd`) is `ISSUE-162`, confirmed unrelated and darwin-only.

The issue's own text raised a trade-off — a literal workflow-name match vs. a more general
artifact-scanning selection strategy — without prescribing a shape. The plan resolved it in favor
of the literal match (option a), rejecting the artifact-scan alternative (option b, an unbounded
per-gate `gh api` latency tax) and deliberately deferring a third, workflow-file-scoped endpoint
already used elsewhere in this repo's own `release.yml` (option c) as a plausible future
improvement, not a defect in what shipped.

## References

- `cmd/backstop/baseline.go:202-224` (`resolveLatestSuccessfulMainRun`) — the unfiltered selection
  loop; `Name` decoded at line 210, never referenced past that.
- `cmd/backstop/baseline.go:226-246` (`resolveBaselineArtifactID`) — the downstream call that
  fails with `missing artifact` when the selected run lacks `backstop-baseline-v1`.
- `cmd/backstop/gate.go:208,331` (`resolveBaselineCache`) — runs `resolveLatestSuccessfulMainRun`
  transitively via `runBaselinePull` on every `backstop gate` invocation, so a future misselection
  would surface here, not only via the explicit `backstop baseline pull` CLI command.
- `ISSUE-176` (`ci-gate-missing-baseline-json`) / `PLAN-ISSUE-176`
  (`ci-gate-baseline-pull-wiring`) — the lane whose evidence-gathering (2026-08-18) first surfaced
  this gap while confirming the artifact the self-healing pull would fetch, and recorded it as
  CLM-007(a): "a real latent defect in the pull, outside this lane's fix, filed as a follow-on."
  This issue is that follow-on. `PLAN-ISSUE-176`'s own sharp edge 14 required this be filed through
  the routed CLI scaffold rather than hand-written, per this repo's never-hand-edit-artifacts rule
  — this issue was scaffolded via `backstop artifact new issue`, as `ISSUE-176` itself was
  scaffolded from `PLAN-ISSUE-166`'s residual findings.

### Existence-in-world check

Performed 2026-08-18 before filing: `grep -rln "resolveLatestSuccessfulMainRun|resolveBaselineArtifactID|runBaselinePull"` over `issues/` and `bundles/` returned only `ISSUE-176` (which names this exact gap as CLM-007(a) and explicitly defers it as a follow-on, rather than owning a fix for it) and four bundle files whose hits were incidental matches on generic terms ("CI workflow", "workflow name") unrelated to this function. No open issue or bundle charter owns a fix for `resolveLatestSuccessfulMainRun`'s missing workflow-name filter.
