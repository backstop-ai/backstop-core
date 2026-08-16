---
title: "backstop pack new ships a sample validator that always exits 0 — no fresh pack ever discriminates"
schema_version: issue/v1

issue:
  id: ISSUE-146
  title: "backstop pack new ships a sample validator that always exits 0 — no fresh pack ever discriminates"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# pack new ships a sample validator that always exits 0

## Problem

`backstop pack new` (all three scaffold types — `engine`, `mechanism`, `toolchain`; they share one
code path) writes a sample sandbox validator that is knowingly, permanently vacuous.
`pkg/pack/scaffold.go:184`:

```go
validator := "#!/bin/sh\n# Sample sandbox validator: always passes. Replace with real enforcement —\n# exit non-zero and print a message to flag a violation.\nexit 0\n"
```

The comment states the defect outright: it always passes. The generated rule
(`pkg/pack/scaffold.go:135-182`) declares `validator: validators/<slug>.sh` with no `layer:` field
— an engine-model rule, dispatched through the validator-only guard `if rule.Validator != ""` in
`pkg/packval/phase3.go`. The paired negative fixture (`scaffold.go:186`,
`fixtures/invalid/example.txt`) carries no marker or property the validator could ever detect —
its content is structurally identical to the positive fixture (both are `package sample\n` plus a
comment), so even a validator that tried to discriminate would have nothing to check for. Neither
half of the pair — validator or fixture — is wired to genuinely distinguish pass from fail.

### Why this was previously invisible

Before `PLAN-ISSUE-092`'s fix to `pkg/packval/phase3.go` landed, the validator-dispatch guards
checked `rule.Layer == 3` (the retired layer model), which was `false` for this engine-model rule —
so the vacuous validator's dispatch was dead code and never actually ran. `pack test` on a
freshly-scaffolded pack reported `PASS`, but not because the sample validator correctly
discriminated pass/fail — because it never dispatched at all.

### Confirmed as a true positive, not a test artifact

Verified directly against the current tree (2026-08-16): `PLAN-ISSUE-092`'s fix is landed —
`pkg/packval/phase3.go`'s validator branches (lines 112-117, 135-142) are gated only on
`rule.Validator != ""`, with no `Layer` check anywhere in the file. With that guard now correctly
dispatching the sample validator, it exits 0 on the negative fixture (which should fail), and
phase 3 correctly reports: `[phase3-fixtures/validator-negative] layer3 negative unexpectedly passed`.

Two tests catch this now:

- `TestPackNew_ScaffoldPassesCheckAndTest` (`cmd/backstop/pack_new_test.go`) — all three pack types
- `TestPackAuthoringLoop_EndToEnd` (`cmd/backstop/pack_authoring_loop_test.go`)

Per `PLAN-ISSUE-092`'s own ruling, these two tests were deliberately NOT weakened to paper over
this — the red is correct, and belongs to a fix in `pkg/pack/scaffold.go`, outside that plan's file
scope.

## Impact

Every pack scaffolded via `backstop pack new` ships a sample rule whose validator can never
red — not "rarely," structurally never, because the validator script and the negative fixture are
both fixed literals with no discriminating relationship between them. A pack author who runs
`pack test` on a fresh scaffold and sees green has verified nothing about their own enforcement
logic; the green is unconditional from the moment the files are written. This is exactly the class
of vacuous-green defect the wider `PLAN-ISSUE-092`/`DIR-032` effort exists to kill, now surfaced on
the one command whose entire job is to hand a new pack author a working example to build from.

The command's own help text overstates what it delivers: `cmd/backstop/pack_new.go:30` promises "a
sample rule that passes pack check, pack test, and the gate" — true today only in the sense that it
passes trivially, not that it demonstrates real enforcement. Once fixed, this framing should hold in
the stronger sense (a rule that genuinely discriminates and passes because both fixtures are
correctly classified) — worth a final check once the fix lands, not a re-scope now.

## Direction

Not decided, just the shape: the generated validator and its paired negative fixture need to
actually discriminate — e.g. the validator greps its input file for a marker string (or checks some
other genuine property) and exits non-zero when found; the negative fixture carries that marker
and the positive fixture does not. `pack test` on a fresh scaffold should then exercise a real
pass/fail signal instead of an always-pass placeholder. The fix is scoped to `pkg/pack/scaffold.go`
(the `validator`, `positiveFixture`, `negativeFixture` string literals and/or the generated
`pack.yml`'s rule declaration) — it does not touch `pkg/packval` at all, since that machinery is now
correctly dispatching; the defect is in what gets scaffolded, not in how it's checked.

## Notes

- Discovered during `PLAN-ISSUE-092` implementation (2026-08-16) as an unanticipated second blocker
  — in scope for `PLAN-ISSUE-092` to surface (its own fix is what makes this rule's dispatch live
  for the first time), out of scope for that plan to fix (its file scope does not include
  `pkg/pack/scaffold.go`).
- Same "vacuous green" defect family as `ISSUE-092` (phase3 fixture dispatch dead code),
  `ISSUE-140`/`ISSUE-142`/`ISSUE-144` (packval dispatch gaps), all homed in `DIR-032` ("Gate Verdict
  Honesty"). This issue is a different layer than all of them: those are dispatch-machinery gaps
  (the check never runs); this is an authoring-content gap (the check runs but was authored to
  never discriminate). Left unslotted here for backlog-pm/directive-author triage rather than
  hand-edited, per this repo's artifact-authoring convention — likely fit is `DIR-032` given the
  vacuous-pass shape, but the charter call belongs to that triage step, not to this filing.
- Existence-in-world check performed 2026-08-16 before filing: searched `issues/` for
  `sample-check`/`sample validator`/`pack new` and `bundles/` for the same. Two closed issues
  matched textually — `ISSUE-032` (pack-CLI authoring-loop reboot; delivered the `pack.yml`-emitting
  scaffold this issue's code lives in, but did not address validator/fixture content) and
  `ISSUE-049` (pack check/test ignoring a positional path argument; unrelated defect on the same
  command). Neither is a duplicate; neither owns this defect. No open issue or bundle charter
  already covers it.
