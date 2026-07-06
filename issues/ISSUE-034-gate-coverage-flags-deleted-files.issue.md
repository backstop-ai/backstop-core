---
title: "Gate Coverage Flags Deleted Files"
schema_version: issue/v1

issue:
  id: ISSUE-034
  title: "Gate Coverage Flags Deleted Files"
  type: bug
  status: open
  created: "2026-07-05"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
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
