---
name: project-issue154-plan-terminal-exemption
description: ISSUE-154 owns "plan validator lacks terminal-status exemption" (plan/phases-empty on canceled plans); check before filing a dup
metadata:
  type: project
---

ISSUE-154 (filed 2026-08-17, status open) tracks: `pkg/validate/plan.go` never calls the shared
`isTerminalStatus` predicate (`pkg/validate/terminal.go`) that `spec.go`/`bundle.go`/`issue.go`
each gate their completeness checks behind — so `validatePhases` (`plan.go:167`) runs
unconditionally even on `canceled`/`replaced`/`obsoleted` plans, producing a `plan/phases-empty`
violation on any terminal plan with an empty `phases: []` (e.g. `PLAN-ISSUE-066`, retired
2026-08-17 as never-authored/unnecessary work). Confirmed still-current 2026-08-19 (independent
grep of `plan.go` for `isTerminalStatus` — zero hits; `validatePhases` still called unconditionally
at line 167).

**Why:** on 2026-08-19, a byproduct-bug discovery during `PLAN-ISSUE-172` implementation
(gate-double-run-test-suite investigation) turned up this exact defect — same file
(`PLAN-ISSUE-066`), same root cause, same violation — one day after ISSUE-154 had already filed
it. Caught only because the existence-in-world check searched `issues/` for "phases-empty" before
authoring; the near-duplicate would otherwise have shipped.

**How to apply:** before filing any issue about a plan/spec/bundle/issue validator missing a
terminal-status exemption, or about `plan/phases-empty` on a canceled/retired plan, check
ISSUE-154 first — it likely already owns the surface. Annotate it (a note, not a status flip)
rather than filing anew. See [[feedback_stub_filename_extension]] for the adjacent
scaffold-mechanics gotcha in this same neighborhood of plan-retirement artifacts.
