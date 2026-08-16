---
title: "Three SPEC-041 Claims Carry Dormant test_substantiveness Violations in Untouched Files"
schema_version: issue/v1

issue:
  id: ISSUE-138
  title: "Three SPEC-041 Claims Carry Dormant test_substantiveness Violations in Untouched Files"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

## Problem

Three SPEC-041-mandated tests, in two files, currently carry LATENT (dormant, not yet
firing) `test_substantiveness` violations that will red the moment either file next
enters a diff's scope — for reasons unrelated to whatever change touches them:

- `cmd/backstop/pack_gate_exempt_resolution_test.go` —
  `TestExemption_PerViolationResolutionNoGateTypeAggregation` (CLM-018)
- `cmd/backstop/shared_testrun_eradication_test.go` —
  `TestSharedRunner_Eradicated` (CLM-004) and
  `TestSharedRunner_NoRenamedWholeModuleGoTestRunner` (CLM-006)

This is the same underlying mechanism ISSUE-113 and the PLAN-ISSUE-129 investigation
encountered in a sibling file, `cmd/backstop/pack_gate_exempt_test.go` — that file's 5
tests already fired because an unrelated change happened to put it in diff scope. These
two files simply haven't been touched yet.

## Evidence (verified directly against current tree)

All three claims inherit target package `gate` — `specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md:66`
declares `implementation.package: pkg/gate` at the spec's top level, and none of
CLM-018/CLM-004/CLM-006 (`specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md:363`,
`:270`, `:280`) declares a `subject:` override, so `TargetPackageName` reduces the
target to `"gate"` (`pkg/gate/substantiveness_join.go` — `filepath.Base("pkg/gate")`).
None of the three claims is annotated `kind: absence`, so none pre-skips the join via
`NoTargetViolationForTest`'s absence check (`pkg/gate/substantiveness_join.go:89`).

`NoTargetViolation` (`pkg/gate/substantiveness_join.go:63`) is a pure
`referenced[targetPkg]` set-membership test, where the referenced set is assembled from
Q2 extraction pack findings scanning the test's own body for package-selector
references — a bare TYPE reference (e.g. `[]gate.Violation{}`) satisfies it exactly as
a real function call would; the join cannot distinguish the two.

Read directly, each test body contains **zero** `gate.` tokens of any kind:

```
$ awk '/^func TestExemption_PerViolationResolutionNoGateTypeAggregation/,/^}/' \
    cmd/backstop/pack_gate_exempt_resolution_test.go | grep -c 'gate\.'
0
$ awk '/^func TestSharedRunner_Eradicated/,/^}/' \
    cmd/backstop/shared_testrun_eradication_test.go | grep -c 'gate\.'
0
$ awk '/^func TestSharedRunner_NoRenamedWholeModuleGoTestRunner/,/^}/' \
    cmd/backstop/shared_testrun_eradication_test.go | grep -c 'gate\.'
0
```

`TestExemption_PerViolationResolutionNoGateTypeAggregation` asserts on `v.ProjectWide`
where `v` comes from a same-package helper (`dispatchOneEngine`, itself declared to
return `[]gate.Violation` — but that declaration lives in the helper, not in the
mandated test's own body). `TestSharedRunner_Eradicated` and
`TestSharedRunner_NoRenamedWholeModuleGoTestRunner` assert purely via `os.Stat`,
`grepNonTestSource`, and string-contains checks over `cmd/backstop`/`pkg/gate` source
text — no `gate` package symbol is referenced at all.

## Related finding: the fix direction to avoid

A neighboring claim in the same file, `TestExemption_TrueConflictExemptingValueWins`
(`cmd/backstop/pack_gate_exempt_resolution_test.go:87`, CLM-019), currently satisfies
the join only by accident: its body contains
`union := append(append([]gate.Violation{}, exemptViolations...), nonexemptViolations...)`
— a composite-literal TYPE reference, not a call. This is not itself flagged as
violating (the join is satisfied), but it demonstrates the trap: a cosmetic "drop an
unused `gate.Something` reference into the body" patch would satisfy the checker
without adding any real behavior-locking value. The honest fix for the dormant class
below is a real call into `gate` — e.g. driving `gate.StepCodeCheckScopedFunc` (or
equivalent) directly instead of delegating wholly to same-package helpers like
`filterThroughGate`/`dispatchOneEngine`/`grepNonTestSource` — not a decorative
reference chosen only to satisfy the set-membership check.

## Open question for the fix to weigh (not resolved here)

CLM-004 and CLM-006 are eradication/absence-shaped assertions — they prove something
was REMOVED or does not exist (no `shared_testrun.go`, no renamed whole-module
`go test` runner). This is structurally identical to CLM-021 through CLM-024 in the
same spec (`specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md:380-407`), which
ARE annotated `kind: absence` and therefore correctly pre-skip the join. CLM-004 and
CLM-006 lack that annotation despite the same shape.

Two ways to close this gap, and the fix should pick one deliberately rather than
defaulting:

1. **Annotation gap**: add `kind: absence` to CLM-004 and CLM-006 in
   `specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md`, matching the
   CLM-021..024 precedent, since both tests genuinely prove an absence and calling into
   `gate` would not make them more honest.
2. **Code-level fix**: leave the claims un-annotated and give each test a real in-body
   call into `pkg/gate` (matching whatever fix direction is chosen for the
   `pack_gate_exempt_test.go` sibling case under PLAN-ISSUE-129), on the theory that a
   real `gate.` call is deliberately meant to exist even in an absence-proving test.

This issue does not resolve which; it is the concrete open question the eventual fix
(or spec-author, if the annotation route is chosen) needs to settle for all three
claims plus CLM-019's accidental pass.

## Impact

Left as-is, the next unrelated change that happens to touch either
`pack_gate_exempt_resolution_test.go` or `shared_testrun_eradication_test.go` (for
example, adding an unrelated test to the same file) will red the gate's
`test_substantiveness` step with a "does not call package gate" violation on a test
that has nothing to do with the triggering change — the same false-attribution
surprise already hit once via the sibling `pack_gate_exempt_test.go` file, costing
PLAN-ISSUE-129 investigation time before the true cause (dormant-until-diff-scoped) was
identified.

## References

- `issues/ISSUE-113-zero-match-classification-refusal.issue.md` — a directly related
  sibling defect in the SAME substantiveness-join mechanism (SPEC-037/SPEC-041's
  `test_substantiveness` firing on files that only recently entered diff scope), but a
  DIFFERENT defect: ISSUE-113 is about the join's REFUSAL-condition design when
  classification matches zero test files project-wide; this issue is about specific
  dormant test bodies that will fail the join once diff-scoped. Not a duplicate of
  either.
- `specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md:66` (`implementation.package: pkg/gate`),
  `:270-289` (CLM-004/CLM-005/CLM-006), `:363-368` (CLM-018), `:380-407` (CLM-020..024,
  the `kind: absence` precedent).
- `pkg/gate/substantiveness_join.go:63` (`NoTargetViolation`), `:89`
  (`NoTargetViolationForTest`).
- `cmd/backstop/pack_gate_exempt_resolution_test.go:37-68` (CLM-018 test body),
  `:87-115` (CLM-019 test body, the accidental-pass composite literal).
- `cmd/backstop/shared_testrun_eradication_test.go:20-28` (CLM-004 test body),
  `:54-71` (CLM-006 test body).
