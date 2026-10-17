---
title: "Pack Engines Evidence Omits Dispatched Engines On Pass"
schema_version: issue/v1

issue:
  id: ISSUE-230
  title: "Pack Engines Evidence Omits Dispatched Engines On Pass"
  type: enhancement
  status: open
  created: "2026-10-17"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Pack Engines Evidence Omits Dispatched Engines On Pass

## Problem

When the `pack_engines` step passes, its result does not say which engines ran or how many results each
returned. From the evidence alone you cannot show that a given pack engine was dispatched at all, so a
pass from an engine that ran and found nothing is indistinguishable from an engine that was never
dispatched.

Verified at origin/main:

- `gate.StepResult` (`pkg/gate/result.go:136-147`) carries `StepName`, `Status`, `Violations`, the
  baseline buckets, `Reason`, and `NativeSandboxApplied`. There is no per-engine field.
- The step is assembled in `cmd/backstop/gate.go` (~lines 1205-1212) as
  `StepResult{StepName: "pack_engines", Status: StepVerdict(violations), Violations: violations}`. Only
  violations survive; a passing run has an empty list.
- `pack_engines` is the step that enforces the pack rules, so "pass" is the main evidence a consumer has.

## Impact

A reviewer or an automated check cannot confirm from the gate output that, for example, a Node-based or
grep-based engine actually executed. Misconfiguration that silently drops an engine (an excluded rule,
an engine routed to a dedicated step) looks identical to a clean pass.

## Direction

(Direction, not a plan.) Record one row per dispatched engine in the step result: pack, engine, and the
number of results it returned (including zero), on pass as well as fail. Additive fields under the
existing gate schema would keep current consumers working.

## Notes

- Checked against the unmerged `fix/engine-evidence-integrity` branch. Its ISSUE-209 work (commit
  dacfd7c) makes the dispatcher continue past a failed engine and report each failure, ISSUE-207 makes an
  empty-output run with an undeclared exit status an engine failure, and ISSUE-217 makes the coverage
  floor configurable. All three make failure explicit; none adds a record of engines that succeeded. That
  branch leaves `pkg/gate/result.go` unchanged, so this issue is not covered by it. It does reshape
  `dispatchPackEnginesWithEvidence`, so the per-engine rows should be added after it merges or coordinated
  with it.
- Related: ISSUE-209 (per-engine failure accounting).
