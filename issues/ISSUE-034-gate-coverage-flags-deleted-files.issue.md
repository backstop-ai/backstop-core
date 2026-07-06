---
title: "Gate Coverage Flags Deleted Files"
schema_version: issue/v1

issue:
  id: ISSUE-034
  title: "Gate Coverage Flags Deleted Files"
  type: bug
  status: closed
  created: "2026-07-05"
  closed: "2026-07-06"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/gate/..."

implementation:
  summary: >
    In coveragePathsInScope (pkg/gate/step_coverage.go), after the existing
    measurable-source glob filter, skip any in-scope path that does not exist
    on disk (os.Stat under scope.ProjectRoot). A git-deleted file matches the
    source glob but has nothing left to measure, so it no longer enters the
    coverage-required set and can no longer produce a coverage_unmeasured
    violation. The all-mode/nil-scope branch (paths sourced from actual
    coverage records) is unchanged. scope.go and GateScope are untouched —
    the fix is deliberately coverage-local, not a scope-wide change to what
    "in scope" means, to avoid disturbing contract-absence's use of
    scope.Contains for deletion-regression checks.
  package: pkg/gate

requirements:
  - id: REQ-001
    text: >
      The gate's coverage_threshold step must not treat a git-deleted
      in-scope measurable-source file as requiring a coverage record. A
      genuinely unmeasured added/modified measurable-source file must still
      produce a blocking coverage_unmeasured violation (no regression).

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      A deleted measurable-source file (in scope, not present on disk)
      produces zero coverage_unmeasured violations naming that file.
    tests:
      - TestCoverage_DeletedInScopeFile_NoUnmeasuredViolation
  - id: CLM-002
    requirement: REQ-001
    text: >
      An added/modified measurable-source file with no coverage record still
      produces a coverage_unmeasured violation — the existence filter narrows
      to deletions only and does not blind the genuine unmeasured-file check.
    tests:
      - TestCoverage_AddedUnmeasuredFile_StillFlagged
  - id: CLM-003
    requirement: REQ-001
    text: >
      A mixed scope containing both a deleted and an added measurable-source
      file produces exactly one coverage_unmeasured violation, naming only
      the added file — confirming the fix is precisely scoped to deletions
      and does not affect a co-occurring genuine gap.
    tests:
      - TestCoverage_DeletedAndAddedInScope_OnlyAddedFlagged

contracts:
  - file: pkg/gate/step_coverage.go
    provides:
      - name: coveragePathsInScope
        kind: function
        signature: "func coveragePathsInScope(coverage []check.CoverageRecord, scope *GateScope, classifier SourceClassifier) []string"
---

# ISSUE-034: Gate Coverage Flags Deleted Files

## Problem

The gate's `coverage_threshold` step treats git-**deleted** files as in-scope
"measurable source files that need coverage measurement," producing spurious
`coverage_unmeasured` violations for files that no longer exist.

### Repro (observed 2026-07-05, while implementing ISSUE-018)

ISSUE-018 deleted `cmd/backstop/code_check.go` and `pkg/check/registry.go` (both
`.go` files, matched by the Go toolchain pack's `classification.source` glob).
Running `./bin/backstop gate` against that change reported:

```
[coverage_unmeasured] no coverage measurement for in-scope changed measurable-source file cmd/backstop/code_check.go
[coverage_unmeasured] no coverage measurement for in-scope changed measurable-source file pkg/check/registry.go
```

Both files were **deleted** by the change, not modified or added. Neither
exists on disk. A deleted file cannot be measured for coverage, and nothing
should ever assert that it must be — but the step demanded a coverage record
for it anyway.

### Root cause (traced in `pkg/gate`)

1. `resolveGateScopeDiff` (`pkg/gate/scope.go:104-131`) builds the diff-scope
   file list from `git diff --name-only <base>` (plus `git ls-files --others
   --exclude-standard` for untracked files). `git diff --name-only` lists
   every path touched by the diff — added, modified, **and deleted** — with
   no `--diff-filter` applied and no per-file change-status captured.
2. `GateScope` (`pkg/gate/scope.go:24-30`) stores the result as a flat
   `Files []string`. There is no change-status field (no `A`/`M`/`D`) carried
   anywhere on the scope — by the time any step consumes `scope.Files`, the
   fact that a path was deleted has already been discarded.
3. `coveragePathsInScope` (`pkg/gate/step_coverage.go:361-386`) builds the
   coverage-required set by iterating `scope.Files` and keeping every path
   that `classifier.IsMeasurableSource(clean)` matches — a pack-declared glob
   check only. It never checks whether the path still exists on disk or was a
   deletion.
4. `StepCoverageThresholdScopedFunc` (`pkg/gate/step_coverage.go:120-137`)
   then walks that path set and, for any path with no coverage record at all,
   emits the blocking `coverage_unmeasured` violation (line 131-136) —
   correctly, for a real gap, but incorrectly here because there is no file
   left to measure.

So a deleted `.go` file survives every filter that exists today (it matches
the source glob; it has no coverage record) and lands in the same bucket as a
genuinely unmeasured new/changed file.

### Why it matters

This makes the coverage dimension go RED on **any** change that deletes a
measurable source file — which is exactly the shape of change the
thin-executor eradication backlog performs repeatedly (deleting baked-language
code; see ISSUE-018, ISSUE-030, and the broader eradication-backlog project
memory). It is a false positive that would force spurious waivers or block
otherwise-legitimate deletions, undermining the "loud on real gaps, not on
vacuous or wrong ones" enforcement philosophy the gate is built on (see
CLAUDE.md's "Loud ≠ blocking").

This is the mirror case of the prior diff-scope fix that made scope
**include** untracked files (`pkg/gate/scope.go`'s
`ls-files --others --exclude-standard` addition): that fix widened scope
correctly for additions; this issue is the missing narrowing for deletions.

### Scope of the leakage — coverage-specific, not (yet) confirmed elsewhere

`coverage_unmeasured` is unusual among the gate's per-file dimensions in that
it asserts a **positive obligation** — "this in-scope path must have a
coverage record" — derived purely from the diff-scope file list, independent
of whether any tool actually produced output for that path. The other
per-file dimension checked (`step_contract.go:54`) only *filters* violations
that a tool already produced down to in-scope files (`scope.Contains`); it
never asserts an obligation for a path with no violation, so a deleted file
simply produces no contract findings and is not flagged. The same appears
true for substantiveness (SARIF-finding-driven, not obligation-driven). The
planner should confirm this by grep/read of `pkg/gate/substantiveness_*.go`
and any other step iterating `scope.Files` directly (as `step_coverage.go`
does) before treating the fix as coverage-only, but the mechanism above is
believed specific to coverage's from-scratch obligation set.

## Solution

Exclude deleted files from the coverage-required set. Two viable directions
for the planner to weigh:

1. **Filter at the scope-resolution boundary.** Have `resolveGateScopeDiff`
   compute (or a sibling helper derive) git change-status per path — e.g. via
   `git diff --name-status <base>` — and drop paths whose status is `D`
   before they ever enter `GateScope.Files`. This is the more conservative
   surface: it fixes every current and future consumer of `scope.Files`
   uniformly, not just coverage, at the cost of changing what "in scope"
   means gate-wide (worth checking no other step relies on deleted files
   being visible, e.g. for reporting "this file was removed").
2. **Filter at the coverage step only.** Have `coveragePathsInScope` (or a
   helper it calls) check on-disk existence (`os.Stat`) before including a
   path in the coverage-required set — narrower, coverage-only, lower risk of
   behavior change elsewhere, but leaves the same defect latent for any
   future step that independently iterates `scope.Files` looking for
   measurable source.

Either direction must be proven with a real (non-stubbed) test: construct a
diff scope containing a deleted `.go` file (or fixture equivalent) and assert
`StepCoverageThresholdScopedFunc` does NOT emit `coverage_unmeasured` for it,
alongside a companion case proving a genuinely new/modified unmeasured file
still does (regression guard against silently swallowing the real check this
step exists for).

**Acceptance:** a change that deletes a measurable-source file produces zero
`coverage_unmeasured` violations for that file; a change that adds or modifies
a measurable-source file with no coverage record still produces the violation
(no regression); the planner has confirmed, and the fix documents, whether
contract/substantiveness needed the same treatment.

## Resolution

Shipped on `main` in squash commit `d5efd5b` (2026-07-06), as part of the
thin-executor eradication checkpoint that also delivered ISSUE-018,
ISSUE-035, ISSUE-036.

Direction 2 was chosen: **coverage-local fix**, not a scope-wide change.
`coveragePathsInScope` (`pkg/gate/step_coverage.go`) gained an on-disk
existence guard immediately after the existing measurable-source glob check.
When `scope.ProjectRoot != ""`, any in-scope path for which
`os.Stat(filepath.Join(scope.ProjectRoot, filepath.FromSlash(clean)))` errors
(i.e. does not exist — the git-deleted case) is skipped and never enters the
coverage-required set. The all-mode/nil-scope branch, which sources paths
from actual coverage records rather than `scope.Files`, is untouched.
`scope.go` / `GateScope` were deliberately left unchanged: the leakage
confirmation (recorded in `PLAN-ISSUE-034`) verified that `step_contract.go`
only filters already-produced findings via `scope.Contains` (no positive
per-path obligation) and substantiveness is SARIF-finding-driven, so neither
shares this defect — only coverage builds a from-scratch obligation set off
`scope.Files`, so only coverage needed the filter. A scope-wide deletion
filter was rejected because it would have silently turned
`step_contract.go`'s deletion-regression (`Absent`) check into a no-op for
any contract whose declared file had just been deleted.

Verified with a new, non-stubbed regression file,
`pkg/gate/step_coverage_deletion_test.go`, exercising the real
`StepCoverageThresholdScopedFunc` against a real `GateScope` and
`SourceClassifier` (no mocks):

- `TestCoverage_DeletedInScopeFile_NoUnmeasuredViolation` — a deleted
  in-scope `.go` file (absent on disk) produces no `coverage_unmeasured`
  violation naming it.
- `TestCoverage_AddedUnmeasuredFile_StillFlagged` — an added in-scope `.go`
  file present on disk with no coverage record still produces the blocking
  violation (regression guard against over-correction).
- `TestCoverage_DeletedAndAddedInScope_OnlyAddedFlagged` — a combined scope
  with both files present pins the two-sided behavior in one assertion:
  exactly one `coverage_unmeasured` violation, naming only the added file.

Confirmed via `backstop gate` after landing: the two deleted-file
`coverage_unmeasured` violations from ISSUE-018's deletion of
`cmd/backstop/code_check.go` and `pkg/check/registry.go` cleared, while the
unrelated genuine below-threshold gap on `pkg/check/manifest.go` (gutted by
the same ISSUE-018 change) correctly persisted — proving the fix narrowed
deletions only and did not blind or suppress a real gap.

## References

- `pkg/gate/scope.go:104-131` — `resolveGateScopeDiff`, uses
  `git diff --name-only` with no `--diff-filter`/change-status, so deletions
  are indistinguishable from additions/modifications in the resulting file
  list
- `pkg/gate/scope.go:24-30` — `GateScope` struct, `Files []string` carries no
  per-file change-status
- `pkg/gate/step_coverage.go:361-386` — `coveragePathsInScope`, builds the
  coverage-required set from `scope.Files` filtered only by
  `classifier.IsMeasurableSource`, no existence/deletion check
- `pkg/gate/step_coverage.go:120-137` — the `coverage_unmeasured` emission
  site, correct for a real gap but firing on paths that no longer exist
- `pkg/gate/step_contract.go:54` — contrast: contract only filters
  already-produced violations to in-scope files (`scope.Contains`), asserts
  no positive per-path obligation, so it does not appear to share this defect
  (confirm at plan time)
- ISSUE-018 — the deletion change (`cmd/backstop/code_check.go`,
  `pkg/check/registry.go`) that surfaced this defect while landing
- ISSUE-023 (gate-loud-on-nothing-measured) — sibling family: gate loudness
  firing on the wrong file set / wrong condition, not yet authored past stub
- ISSUE-035 (gate-substantiveness-flags-testmain-absence-tests) — sibling
  family filed the same day: gate loudness misfiring on a per-file dimension,
  not yet authored past stub
- Project memory `gate_scope_and_coverage` — the prior, opposite-direction
  scope fix (including untracked files in diff scope); this issue is the
  mirror case (excluding deleted files)
