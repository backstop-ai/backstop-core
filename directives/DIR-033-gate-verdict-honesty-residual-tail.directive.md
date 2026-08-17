---
title: "Gate Verdict Honesty Residual Tail"
number: DIR-033
created: "2026-08-17"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-149"
    - "ISSUE-150"
    - "ISSUE-152"
    - "ISSUE-154"
    - "ISSUE-155"
    - "ISSUE-156"
    - "ISSUE-157"
    - "ISSUE-159"
    - "ISSUE-161"
---

## Description

Homes nine issues that DIR-032's own member plans filed as mandated follow-ons
during DIR-032's delivery, discovered by a 2026-08-17 backlog-pm full sweep
after DIR-032 (`directives/DIR-032-gate-verdict-honesty.directive.md`) had
already reached `status: done` (completed 2026-08-17). This is a new
successor directive rather than a reopening of DIR-032 or a fold-in to
DIR-024 — a founder ruling made specifically to keep DIR-032's earned `done`
verdict intact and to give this tail its own priority rather than diluting
DIR-024's catch-all scope.

**The structural reason this directive exists, stated generally because it is
not unique to DIR-032:** a directive's roster is a closed set, fixed at
authoring/amendment time. A plan's mandated follow-on filings are an open set,
generated continuously during delivery — often after the directive that
sourced the plan has already closed. Nothing reconciles the two automatically;
a plan can file a follow-on issue the day after (or, as happened here, the same
day) its parent directive is marked `done`, and that issue has no home until
someone gives it one. DIR-032 itself narrated this exact pattern happening to
its OWN roster repeatedly while still open (see DIR-032's Description
corrections for items 12-21, each added mid-flight by the same mechanism).
This directive is that same pattern's tail, arriving after closure instead of
before it — future directive closures should expect the same and budget for a
similar tail-sweep rather than treating it as a one-off surprise.

All nine issues trace to plans written to implement DIR-032 roster members
(ISSUE-091, ISSUE-093, ISSUE-136, ISSUE-142, ISSUE-146, ISSUE-148 — each a
cited DIR-032 source item). **Two corrections to the provenance as it was
initially relayed, both verified directly against each issue's own text
before writing this directive:**

- **ISSUE-157 is filed by `PLAN-ISSUE-142` TASK-009** ("Source/parent:
  ISSUE-142 ... filed as PLAN-ISSUE-142 TASK-009"), not `PLAN-ISSUE-148` as
  initially relayed. It is a sibling in DEFECT SHAPE to `ISSUE-159` (both
  trace to the fixture-polarity family) but a different FILING PLAN and a
  different pack (`backstop-ai/go-contracts` vs `backstop-ai/go-substantiveness`).
- **ISSUE-154 is not a record-only prose correction.** It is a real code
  defect (`type: bug`) in `pkg/validate/plan.go`, motivated by (not filed as a
  TASK item of) `PLAN-ISSUE-066`'s retirement — the plan's cancellation is the
  concrete instance that exposed the validator gap, not the mechanism that
  filed the issue.

The nine issues are genuinely heterogeneous — grouping them as one
undifferentiated bucket would misrepresent what this directive actually
homes:

**Group 1 — completed-plan claim-prose retractions (no live defect, no gate
impact).** `ISSUE-149` and `ISSUE-150`, both filed by `PLAN-ISSUE-091`
TASK-006, record that PLAN-ISSUE-010's and PLAN-ISSUE-040's claim prose (about
`gate --all`'s pre-fix behavior) is now stale, per the "completed plans are
never rewritten — corrections go in a new issue" convention. Neither plan
declares a `test_names:` field, so neither retraction reds any gate dimension.
`ISSUE-150` additionally raises an unresolved founder-only open question
(whether a testdata-content-audit view is worth a separate flag now that
`--all` filters testdata) — not resolved here. `ISSUE-159`, filed by
`PLAN-ISSUE-148` TASK-005 item 2, is the same class applied to *open* artifact
prose rather than a completed plan: it corrects a falsified "purely a manifest
key swap" fix-menu claim in both `ISSUE-148` and DIR-032 item 19, replacing it
with the real combined-ruleset mechanism the actual fix used. It explicitly
does not hand-edit either target — that routing belongs to issue-author and
directive-author respectively, outside this directive's own scope.

**Group 2 — real code/pack gaps requiring implementation.** `ISSUE-154`
(`pkg/validate/plan.go` has no terminal-status exemption from
`validatePhases`, unlike the parallel exemption already present in
`spec.go`/`bundle.go`/`issue.go` — a canceled-but-unauthored plan cannot
validate clean without fabricating a task graph that never happened).
`ISSUE-155` (the installed `backstop-ai/go-toolchain` pack's `golangci` and
`go-coverage` engine bindings omit `exempt_from_scope_filter`, triggering
ISSUE-136's own audit advisory even though both are already behaviorally
correct — a pack-repo edit, version bump, and relock, not a backstop-core
code change). `ISSUE-157` (the published `backstop-ai/go-contracts` mirror
carries the same inverted fixture polarity that `PLAN-ISSUE-142` TASK-005
corrects in-repo; becomes an active install failure the moment
`pkg/packval` starts dispatching pattern-arg fixtures live, though CI/gate
are unaffected today since `pack install` never validates — publishing the
fix to that pack's own repo is founder-gated).

**Group 3 — a founder policy decision, both halves resolved.**
`ISSUE-152` (filed by `PLAN-ISSUE-091` TASK-006 item 4: whether ISSUE-091's
fix should lift the `--all` half of CI's blocking-job scope ban) and
`ISSUE-156` (the sibling `--file`-half question, filed by `PLAN-ISSUE-093`'s
own required-follow-on) are BOTH **`status: closed`**,
`resolved-by: 73eedb135b491685ea7251fc5fc4365ac9dbd5fa` — the same commit
resolves both halves of the one question they were deliberately split by
flag to ask. This directive's own authoring caught a transient race: an
earlier check on ISSUE-156 read it before its close-out landed and reported
it still open; re-verified directly against the committed artifact before
finalizing this text, both read `status: closed`.

**Group 4 — a founder-level product tradeoff, not a bug.** `ISSUE-161`
(`type: question`), observed during `PLAN-ISSUE-146` implementation: `backstop
pack new`'s scaffolded sample validator now genuinely discriminates at `pack
test`/`pack check` time (ISSUE-146's fix), but is structurally inert at real
`backstop gate` time by design (no `input_scope` declared, so the sandbox
validator receives the project root directory rather than per-file targets).
Whether to wire the sample live at gate time trades onboarding-friendliness
against a durable per-gate-run cost imposed on every future pack author — an
explicit tradeoff call, not something to decide inside an implementation lane.

## Notes

- **Origin directive.** `DIR-032` ("Gate Verdict Honesty",
  `directives/DIR-032-gate-verdict-honesty.directive.md`, `status: done`,
  completed 2026-08-17) is this directive's predecessor. DIR-032's own roster
  is unchanged by this directive's existence — DIR-033 homes only the
  follow-on tail, not a reopening of any DIR-032 item.
- **BACKLOG.yml position is deliberately not addressed here.** Per the
  founder's own stated preference, this directive's insertion point in
  BACKLOG.yml (and any resulting change to DIR-032's existing entry there) is
  a separate decision, deferred until after this directive exists.
- **Status verification snapshot, 2026-08-17 (the date this directive was
  authored, corrected after a transient race caught on re-check):** ISSUE-149
  open, ISSUE-150 open, ISSUE-152 closed
  (`resolved-by: 73eedb135b491685ea7251fc5fc4365ac9dbd5fa`), ISSUE-154 open,
  ISSUE-155 open, ISSUE-156 closed (same `resolved-by` commit as ISSUE-152),
  ISSUE-157 open, ISSUE-159 open, ISSUE-161 open. A directive's `source` list
  is a historical record of what it homes, not only a live-work list — closed
  members stay cited.
