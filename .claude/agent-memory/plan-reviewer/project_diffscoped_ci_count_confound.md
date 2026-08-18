---
name: diffscoped-ci-count-confound
description: "CI-confirmation tasks that say 'expect a smaller violation count than the failed run' are confounded — backstop's CI gate is DIFF-SCOPED, so a small follow-up push shrinks the count regardless of the fix"
metadata:
  type: project
---

When a plan hands its real verification to CI (platform-gated defects: `//go:build linux`
mechanisms unverifiable on the darwin dev box), scrutinize the CONFIRMATION CRITERIA for a
count comparison across runs.

`.github/workflows/ci.yml`'s blocking job `gate` runs `./bin/backstop gate --base "$BASE"` —
DIFF-SCOPED, never `--all`. So "the failed release run had 62 violations; a smaller number
after the fix is the expected outcome" is trivially satisfied by pushing a two-file lane: the
in-scope set shrinks, so the count shrinks whether or not the defect was touched.

**Why:** the founder cannot re-run the original release diff, so the temptation is to compare
totals. Totals across differently-scoped runs are not comparable at all.

**How to apply:** demand SCOPE-ROBUST criteria instead, and check the plan actually has them:
(a) the changed package appears in `gate-report.json`'s `.scope.files` (the field exists —
`GateScope.Files`, `pkg/gate/scope.go:26`, serialized under `scope` in `GateResult`);
(b) the specific error SIGNATURE and the NAMED reproducing test are gone. If the plan also
carries a "smaller number expected" sentence, flag it — it invites a false confirmation.

Related: [[traceability-step-cannot-report-offcorpus]], [[stated-convention-vs-byte-arithmetic]].
