---
title: "TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage blames all pkg/gate dirt on init and its own skip guard is unreachable"
schema_version: issue/v1

issue:
  id: ISSUE-139
  title: "TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage blames all pkg/gate dirt on init and its own skip guard is unreachable"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# init purity test misattributes shared pkg/gate dirt, and its own skip guard cannot fire when it would matter

## Problem

`pkg/initialize/sourceset_scan_test.go`, `TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage`
(the sole mandated test for SPEC-069 CLM-063, which asserts the REQ-013 denylist claim "this
spec's implementation changes no file under `pkg/gate`") has two distinct defects in its first
subtest, `"no file under pkg/gate is modified"` (lines 310-342).

### Part 1 — the purity assertion is now structurally overbroad

Lines 322-331:

```go
status := exec.Command("git", "status", "--porcelain", "--", "pkg/gate")
status.Dir = repo
changed, err := status.Output()
if err != nil {
    t.Skipf("git could not report the working tree: %v", err)
}
if strings.TrimSpace(string(changed)) != "" {
    t.Fatalf("this implementation changed files under pkg/gate:\n%s\nREQ-013 forbids it: ...",
        changed)
}
```

This treats ANY working-tree dirt under `pkg/gate` — from any source — as proof that "this
implementation" (init) touched gate machinery it should not have. The comment above it (lines
317-321) explicitly reasons about *why* working-tree-vs-HEAD was chosen over a merge-base diff:
to avoid blaming *other sessions'* landed commits on this change. But it does not extend that
reasoning to other sessions' **uncommitted, concurrent, in-scope** work in the same shared working
tree — which is exactly what a `git status --porcelain` snapshot cannot distinguish from init's
own leakage.

`pkg/gate` is legitimately, concurrently touched by multiple independent, in-scope lanes today.
Two live plan artifacts declare it in their own file scope:

- `plans/PLAN-ISSUE-118-gate-blind-spot-test-only-diffs.plan.yml` — touches
  `pkg/gate/result.go`, `pkg/gate/step_testverify.go`, `pkg/gate/test_verdict_join.go`,
  `pkg/gate/step_coverage.go`, and new `pkg/gate` test files, per its declared IN scope
  (around line 450-454).
- `plans/PLAN-ISSUE-113-zero-match-classification-refusal.plan.yml` — touches
  `pkg/gate/substantiveness_join.go` and adds `pkg/gate/substantiveness_zero_evidence_test.go`,
  `pkg/gate/substantiveness_join_nonregression_test.go` (task file-scope entries throughout).

Any one of those lanes running this test in the shared tree while the other is mid-flight will
see `git status --porcelain -- pkg/gate` non-empty for files it never touched, and fail fatally
with a message ("this implementation changed files under pkg/gate") that is simply false for that
lane. The test has no mechanism to attribute dirt to init specifically versus to a concurrent,
independently-scoped lane — it treats presence of ANY dirt as proof of init's own violation.

### Part 2 — the test's own non-vacuity/skip guard is ordered after the fatal check, making it unreachable in the one case it exists to protect

Lines 333-341, immediately following the fatal check above:

```go
// NON-VACUITY. Once this work is committed the working tree is clean and the
// assertion above would pass while checking nothing, so require the change set
// to actually contain this spec's own files.
own := exec.Command("git", "status", "--porcelain", "--", "pkg/initialize")
own.Dir = repo
mine, ownErr := own.Output()
if ownErr != nil || strings.TrimSpace(string(mine)) == "" {
    t.Skip("this spec's own files are no longer uncommitted, so a working-tree comparison can no longer attribute anything; re-run this before committing")
}
```

This guard's own stated purpose is to recognize when `pkg/initialize` itself has no uncommitted
changes — i.e. init's own work is fully committed, the steady state this repo is now in
(confirmed: `pkg/initialize/sourceset_scan_test.go` was added in a single commit, `48e6b85`,
2026-08-15, "feat(SPEC-069): backstop init implemented", and has not been touched since) — and
skip itself, because a clean `pkg/initialize` means there is nothing left for init to have
possibly leaked into `pkg/gate`.

But this guard sits AFTER the fatal `t.Fatalf` at line 329, not before it. Control flow in Go
`testing` does not continue past a `t.Fatalf` in the same goroutine — so once `pkg/initialize` is
committed and clean (today's steady state) and `pkg/gate` carries any concurrent lane's legitimate
dirt, execution reaches the fatal check, fails there, and never reaches the skip guard that was
written specifically to recognize this file has nothing left to attribute. The escape hatch exists
in the source but is dead code in the one scenario it was designed for.

## Why this matters

This is not a one-off false positive. This project now regularly runs multiple concurrent
implementer lanes against a shared working tree (the current overnight P0 batch — PLAN-ISSUE-118
and PLAN-ISSUE-113 running concurrently, both declaring `pkg/gate` in scope, is a live instance of
exactly this). Any lane that runs `go test ./pkg/initialize/...` (or a full-repo test/gate sweep)
while ANY other lane has uncommitted `pkg/gate` changes gets an unconditional, unfixable-by-that-
lane failure — the message accuses the wrong lane of a violation it did not commit, and the lane
has no way to make the assertion pass short of the other lane finishing and committing first.
Because the skip guard that would recognize "init's own files are clean, so this can't be init's
leak" is unreachable, there is no code path today that resolves the false positive short of
manual human judgment call each time it fires.

The underlying REQ-013 claim (init changes no file under `pkg/gate`) is legitimate and should stay
enforced — the defect is in how the test attributes dirt, not in the claim itself.

## Solution

Two independent fixes, not mutually exclusive:

1. **Fix the guard ordering (Part 2) regardless of Part 1's outcome.** Move the non-vacuity check
   (lines 333-341) so it runs and can skip BEFORE the fatal purity check (lines 322-331), not
   after. This alone at least restores the self-skip escape hatch for the "init's own files are
   already committed" steady state, though it does not by itself fix the misattribution in Part 1
   for the case where `pkg/initialize` genuinely does have uncommitted changes at the same time as
   an unrelated lane's `pkg/gate` dirt.

2. **Narrow or replace the Part 1 attribution mechanism.** Options to weigh once in the code
   (do not over-prescribe without reading the current shape at fix time):
   - Scope the purity check to specifically the files this spec's own commit history or diff
     actually touches, rather than "no dirt anywhere under `pkg/gate`" — e.g. diff against a
     tracked baseline point for this lane rather than a raw working-tree snapshot shared with
     every other concurrent process.
   - Accept that a raw shared-working-tree `git status` snapshot is fundamentally unable to
     distinguish "init leaked a change" from "an unrelated concurrent lane is mid-flight," and
     find a different enforcement point for REQ-013 that does not depend on working-tree state
     shared across sessions (e.g. a code-review/static check that init's own source files import
     or reference no `pkg/gate` symbol, rather than a git-status assertion).
   - If neither is tractable, at minimum document in the test itself that it is unsafe to run
     concurrently with any other in-scope `pkg/gate` lane, and gate CI/session orchestration
     accordingly — though this is a weaker fix since it pushes the burden onto human/orchestration
     discipline rather than the test being correct on its own.

Whichever direction is chosen for Part 1, the fix must preserve REQ-013's actual guarantee (init
changes no file under `pkg/gate`) without producing false failures attributable to other
legitimately concurrent, in-scope work.
