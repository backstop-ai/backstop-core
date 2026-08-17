---
name: plan-validator-no-terminal-exemption
description: pkg/validate/plan.go has NO isTerminalStatus exemption, so a never-authored (phases-empty) plan cannot be retired validator-green — surface the gap, never fabricate phases
metadata:
  type: project
---

`pkg/validate/plan.go` runs `validatePhases` unconditionally. Unlike `spec.go:175`,
`bundle.go:94` and `issue.go:349` — each of which gates its live-work completeness checks
behind `isTerminalStatus` (`pkg/validate/terminal.go`) — plans have NO terminal exemption.
A plan retired at `status: canceled`/`replaced`/`obsoleted` is still held to full phase
structure: at least one phase, every task with id/title/description/files/claims/depends_on,
TDD ordering, gate cadence, and a verification task in the final phase.

Consequence: an EMPTY SCAFFOLD plan (`phases: []`, `artifact new` output never filled in)
**cannot be retired validator-green**. The minimum passing body is a fabricated
test→implementation→verification triad, which would be plan content that was never authored
and must never be implemented. The `.claude/hooks/backstop-validate-artifact.sh` PostToolUse
hook also blocks on the write (the file still lands; the hook errors after).

**Why:** confirmed live 2026-08-17 retiring PLAN-ISSUE-066 (ISSUE-066 no longer reproduced,
so no plan was ever authored). Retiring it as `canceled` left exactly one violation,
`plan/phases-empty` — an improvement over the pre-edit `plan/phases-required`, but not green.

**How to apply:** when asked to retire a never-authored plan, do the retirement
(`status: canceled`, no successor pointer needed — `canceled` requires no `replaced-by`;
`reason`/`notes` optional per `validateRetirementFields`), then REPORT the residual
violation and let the requester choose the disposition. Do not fabricate phases to get
green, and do not silently delete the file — other artifacts cite empty scaffolds by name
(PLAN-ISSUE-067 cites PLAN-ISSUE-066; DIR-032 censuses them). The systematic fix is an
issue→plan against `pkg/validate/plan.go` adding the terminal exemption; DIR-032 lists
eleven `status: draft` PLAN-ISSUE scaffolds that will need the same shape.
Precedent for retiring empty scaffolds exists for DIRECTIVES (DIR-012, DIR-013) — those
validate because directives have no phases requirement. See [[project_plan_closeout_convention]].
