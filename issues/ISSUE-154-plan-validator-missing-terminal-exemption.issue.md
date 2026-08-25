---
title: "Plan Validator Missing Terminal Exemption"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-154

issue:
  id: ISSUE-154
  title: "Plan Validator Missing Terminal Exemption"
  type: bug
  status: closed
  created: "2026-08-17"
  closed: "2026-08-25"
---

# Plan Validator Missing Terminal Exemption

## Resolution

Delivered by `PLAN-ISSUE-154` in commit `5ec5143`. Plan validation now applies phase/task
completeness only to live work while retaining structural and retirement-field validation for
terminal plans. The commit adds focused coverage for canceled, replaced, and obsoleted plans with
absent or empty phases, plus guards proving live plans still require phases and terminal plans do
not bypass structural checks. Closeout reran `go test ./pkg/validate/... -count=1` with Go 1.25.3
and validated the full 456-artifact corpus with the current source-built CLI; both passed with
zero violations.

## Problem

`pkg/validate/plan.go` has no terminal-status exemption from its phase-structure checks.
`validatePhases` is called unconditionally at `plan.go:167`, inside `Plan()`, regardless of the
plan's `status` field. Compare this to the other three artifact validators, which each gate their
completeness checks behind the shared `isTerminalStatus` predicate (`pkg/validate/terminal.go:23`,
which treats `replaced`/`canceled`/`deprecated`/`obsoleted` as retired work exempt from live-work
completeness rules — ISSUE-031's DQ-1):

- `pkg/validate/spec.go:175` — `if !isTerminalStatus(art.Metadata["status"]) { ... }` gates
  verification/implementation/requirements/claims/contracts/capabilities.
- `pkg/validate/bundle.go:94` — `if !isTerminalStatus(maturity) { ... }` gates maturity
  gates/epic/placeholder-ban/formal-requirements.
- `pkg/validate/issue.go:349` — `if isTerminalStatus(status) { return violations }` short-circuits
  `validateIssueTraceability` entirely.

`plan.go` never calls `isTerminalStatus` at all (confirmed by grep — zero hits in the file). It
does define its own local `planStatuses` map (`plan.go:16-21`) that lists `replaced`/`canceled`/
`obsoleted` as *valid enum values* for the `status` field, but that only means a terminal-status
plan passes the status-enum check — nothing exempts it from `validatePhases`, which still demands
a fully-formed, TDD-compliant task graph.

**Consequence.** A plan that is `canceled` because it was never authored and never needs to be
(e.g. `PLAN-ISSUE-066`, retired 2026-08-17 when the defect it would have fixed turned out to
already be resolved by other work) cannot validate clean without fabricating an unauthored
test → implementation → verification task triad: `validateFinalPhase` (`plan.go:603`) demands a
verification task in the last phase, and `validateVerificationDeps` (`plan.go:875`) demands that
verification task depend on an implementation/refactor task, which `validateTDD` (`plan.go:789`)
in turn requires to depend on a test task. The honest state — zero phases, because nothing was
ever implemented — is unrepresentable in a clean-validating plan artifact today. The only ways to
pass validation are either (a) inventing phases/tasks that describe work that never happened, or
(b) leaving the plan in a permanently-red validation state, which pollutes the gate/backlog signal
for a legitimately closed matter.

**Scope.** This is a real, systemic gap, not a one-off tied to one plan. Any future
empty-scaffold plan (`backstop artifact new plan` followed by a decision that the fix isn't
needed) hits this exact wall when retired without ever being authored. It was discovered during a
2026-08-17 sweep that scaffolded several empty draft plans under DIR-032, most of which went on to
be fully authored and implemented — but the wall is real for the retirement path regardless of how
many plans happen to hit it on any given night.

## Fix Direction (not fully designed — needs its own plan)

Add a terminal-status exemption to `pkg/validate/plan.go` mirroring the pattern already used in
`spec.go`/`bundle.go`/`issue.go`: gate `validatePhases` (`plan.go:167`) — and confirm whether any
other completeness check in the file implicitly assumes an authored body — behind
`!isTerminalStatus(status)`, using the shared predicate from `terminal.go` rather than a
local re-derivation, consistent with how the other three types already share it. `status` is
already extracted as a local variable at `plan.go:94` before the `validatePhases` call, so the
gate is a narrow, mechanical addition once the design is confirmed. The structural checks ahead of
it (filename, plan_id, spec_id, status enum, retirement-fields, created, coverage_threshold,
optional-field types) should keep applying to terminal plans exactly as they do for the other
three types — only the phase/task completeness block should be exempted.

Verify at pickup time whether `PLAN-ISSUE-066`'s validation state still reproduces this gap (it
may have been resolved by ad hoc handling since discovery), and re-confirm the comparison points
in `spec.go`/`bundle.go`/`issue.go` are still accurate before designing the fix.
