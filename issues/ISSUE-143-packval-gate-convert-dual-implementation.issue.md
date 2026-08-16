---
title: "Two independent Convert-application implementations — pack_gate.go and packval/executor.go apply the same pipeline stage separately"
schema_version: issue/v1

issue:
  id: ISSUE-143
  title: "Two independent Convert-application implementations — pack_gate.go and packval/executor.go apply the same pipeline stage separately"
  type: technical-debt
  status: open
  created: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Two independent Convert-application implementations

## Problem

`PLAN-ISSUE-141` (in flight the night this issue was filed) fixes `pkg/packval/executor.go`'s
`RunEngine` to apply a pack engine binding's declared `Convert` script before parsing engine
output, matching the real gate dispatch path. Once that fix lands, there will be TWO
independently-maintained implementations of the same pipeline stage — apply `binding.Convert` to
raw engine output before handing it to `check.ParsePackFindings`:

- `cmd/backstop/pack_gate.go`'s `runFindingsEngine` (the real gate dispatch path, `backstop gate`)
- `pkg/packval/executor.go`'s `RunEngine` (the pack-validation/fixture-testing path, `backstop pack
  test` / `backstop pack check` phase3)

This is the same class of defect as `ISSUE-092` (packval vs. the real pack manifest model
drifting apart) and as `ISSUE-141` itself (packval's dispatch drifting from the gate's dispatch by
omitting a step entirely). Two hand-maintained copies of "apply Convert, then parse SARIF" will
drift again the moment either call site changes — a new edge case handled in one copy (error
wrapping, empty-output handling, sandboxing behavior) and not the other reproduces exactly the
bug class ISSUE-141 was filed to fix, just for a different case.

`PLAN-ISSUE-141` deliberately does not attempt consolidation itself: doing so would require
editing `pack_gate.go`, which was the active file scope of two other in-flight implementers
(`PLAN-ISSUE-067`, `PLAN-ISSUE-140`) the same night. Instead, PLAN-ISSUE-141 installs a mechanical
content-scan drift guard (its CLM-006) as an interim tripwire — not a fix for the duplication
itself, only a check that the two copies haven't silently diverged.

## Direction

Extract a single shared Convert-application step (e.g. `packval.ApplyConvert`, name TBD by
whoever plans this) and have `cmd/backstop/pack_gate.go`'s `runFindingsEngine` delegate to it
instead of maintaining its own copy, rather than the reverse — `cmd/backstop` already has a
dependency edge onto `pkg/packval` per
`.backstop/packs/backstop-ai/backstop-core-architecture/architecture/backstop-core.yml`, and
already imports `pkg/packval` for `packval.SandboxedRunStdout`, so this direction requires no new
architectural edge. `pkg/packval` cannot depend back on `cmd/backstop` (packages don't import
`main` packages), which is why the extraction has to live in `pkg/packval`, not `cmd/backstop`.

Before scoping the fix:

1. Confirm the current shape of both call sites — `cmd/backstop/pack_gate.go`'s
   `runFindingsEngine` and `pkg/packval/executor.go`'s `RunEngine` — since PLAN-ISSUE-141 may have
   landed with a different structure than the plan described, and other issues (ISSUE-140,
   ISSUE-142) touching the same file may have shifted it further.
2. Confirm whether PLAN-ISSUE-141's CLM-006 content-scan drift guard is still in place, and read
   exactly what it checks. This issue's fix will likely retire that guard once real consolidation
   removes the duplication it was tripwiring — leaving the interim guard in place after
   consolidation would itself be dead weight.
3. Check whether the two implementations have already diverged in some way that produces wrong
   output on one path but not the other (not just structural duplication) — if so, that divergence
   is a live defect, not just tech debt, and should be flagged/re-scoped accordingly.

## Notes

- Sibling/prerequisite: `ISSUE-141`
  (`issues/ISSUE-141-packval-executor-missing-convert-application.issue.md`) is the bug this
  issue's duplication is a residual of — ISSUE-141 fixes packval's dispatch to apply Convert at
  all; this issue is about the two-implementation shape that fix leaves behind, not the original
  missing-step bug.
- Same drift family: `ISSUE-092` (manifest-model drift between packval and the real pack model)
  and `ISSUE-140` (a narrower `RunEngine`-only drift, the never-started check) are prior instances
  of packval's dispatch diverging from the gate's real dispatch. This issue is the same underlying
  pattern — two independently maintained implementations of one pipeline stage — applied to the
  Convert step specifically.
- Existence-in-world check performed 2026-08-16 before filing: searched `issues/` and `bundles/`
  for `Convert`/`RunEngine`/`packval` and for duplication/consolidation language. No open issue or
  bundle charter already owns this specific dual-implementation concern; ISSUE-141's own
  "Direction" section flags the drift risk for its OWN fix to avoid, but does not file a follow-on
  for the duplication the fix leaves behind — that gap is what this issue covers.
