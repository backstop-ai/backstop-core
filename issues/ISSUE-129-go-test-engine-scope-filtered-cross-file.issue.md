---
title: "go-test pack engine's failures are diff-scope-filtered: a diff-scoped gate PASS can coexist with a genuinely failing, unrelated test"
schema_version: issue/v1

issue:
  id: ISSUE-129
  title: "go-test pack engine's failures are diff-scope-filtered: a diff-scoped gate PASS can coexist with a genuinely failing, unrelated test"
  type: bug
  status: open
  created: "2026-08-15"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: critical
---

# go-test pack engine's failures are diff-scope-filtered: a diff-scoped gate PASS can coexist with a genuinely failing, unrelated test

## Problem

`backstop gate` (diff-scoped — the default invocation, and the only blocking check CI runs on a
pull request) can report a full PASS while a real, currently-failing Go test sits in the tree,
as long as the failing test's file is outside the current diff's scope. This holds whether the
test was JUST broken by an unrelated in-scope change, or was ALREADY failing on `main` before the
diff existed — in either case the failure is computed and then silently discarded.

## Reproduction

Surfaced 2026-08-15 during `PLAN-SPEC-070` (backstop doctor) implementation. The plan's
implementer registered a new `doctor` command in `cmd/backstop/root.go` — a legitimate,
plan-mandated change. `cmd/backstop/ci_recipes_mechanism_test.go`'s
`TestCIRecipes_RegisteredCommandSurfaceUnchanged` (SPEC-067's CLM-052 anti-regression pin, which
enumerates the ENTIRE top-level CLI command set by name) went red the instant the new command was
registered — by design: any spec that adds a top-level command must update the enumerated list
(SPEC-069 already did this for `init`). But `ci_recipes_mechanism_test.go` itself was never part
of PLAN-SPEC-070's diff.

The implementer ran every gate check throughout implementation, including a final diff-scoped run
against a freshly rebuilt binary — every one reported PASS. The red test was caught only by
running `go test ./...` unfiltered by hand as a final sanity check, not by any gate dimension.

## Root cause

Two engines actually execute `go test` against a diff-scoped gate run:

- `pkg/gate/step_coverage.go`'s built-in `coverage_threshold` step, which runs
  `go test -coverprofile=...` — but this measures coverage percentage only. A failing test still
  emits a usable coverage profile (execution and pass/fail are independent of whether coverage
  data is produced), so this step is structurally blind to failure.
- The `go-toolchain` pack's `go-test` findings engine (declared in
  `.backstop/packs/backstop-ai/go-toolchain/pack.yml:77-85`, `gate_type: test`,
  `scope_kind: project-wide`, `project_target: "./..."`), dispatched through
  `cmd/backstop/pack_gate.go`. This IS the engine that runs `go test ./...` project-wide and
  converts real pass/fail output to SARIF findings via `scripts/test-to-sarif.sh` — it correctly
  computes the failure.

The finding is computed, then discarded. `cmd/backstop/pack_gate.go:730` stamps every bridged
violation's `ProjectWide` field from `binding.ExemptFromScopeFilter` (declared per-engine in the
pack manifest, per SPEC-041's declared build-exemption mechanism). The `go-build` engine
(`pack.yml:64-73`) declares `exempt_from_scope_filter: true`, so an out-of-scope build break still
REDs a diff-scoped gate. The `go-test` engine (`pack.yml:77-85`) declares no such flag, so it
defaults false. `pkg/gate/scope.go:302-326` (`filterViolations`) keeps a violation only when
`ProjectWide` is true OR `scope.Contains(violation.File)` — for a `go-test` violation whose file
sits outside the diff, neither holds, and the violation is silently dropped before status
computation.

Confirmed both invocation paths:

- **Diff-scoped (default, and what CI runs on every PR via `.github/workflows/ci.yml:155`,
  `./bin/backstop gate --base "$BASE"`):** affected. `filterViolations` only short-circuits when
  `scope.Mode == GateScopeModeAll` (`pkg/gate/scope.go:303`); a diff/base scope always filters.
- **`backstop gate --all`:** NOT affected — `filterViolations` returns all violations unchanged
  when `scope.Mode == GateScopeModeAll`, so a full sweep sees every `go-test` failure regardless
  of the exemption flag.

The scope of the blind spot is broader than "a change breaks an unrelated test": because the
filter is purely file-membership-based and blind to WHY a violation exists, it equally masks a
test that was ALREADY red on `main` before the current diff, as long as no diff ever happens to
touch that test's file. And no CI job runs `go test ./...` unfiltered as a blocking check —
`.github/workflows/ci.yml` has exactly two jobs: `gate` (diff-scoped, blocking, on every push/PR)
and `baseline` (`--all`-scoped, but gated `if: github.event_name == 'push' && github.ref ==
'refs/heads/main'`, i.e. post-merge only, and `backstop baseline generate`
(`cmd/backstop/baseline.go:51`) is a snapshot generator with no pass/fail exit contract — it does
not fail the workflow on a RED result). So a cross-file test break can be merged to `main` via a
green PR gate with no later blocking check to catch it either.

This is the same structural mechanism ISSUE-070 fixed for the `pack_engines` step's lint findings
in general (`packValidatorStep` skipping the scope filter entirely) — but ISSUE-070's fix made the
filter apply correctly, keyed on `ProjectWide`/`ExemptFromScopeFilter`, which is exactly working
as declared here. This issue is about the DECLARATION: `go-test` should arguably behave like
`go-build`, not like `golangci` lint findings, because unlike a lint severity gradient, a failing
test is a binary "the code is broken" signal with no soft failure mode — and Go compilation-level
whole-module reasoning (why `go-build` is exempt) applies just as much to a whole-module test
suite where cross-package references are common. Whether the fix is "declare `go-test`
`exempt_from_scope_filter: true`" or something more surgical (e.g. only project-wide-scope tests
that regressed relative to a baseline) is not resolved here.

## Related work (not a duplicate — different mechanism)

- **ISSUE-118** (open) names a similar-sounding symptom ("gate reports full PASS while a mandated
  test genuinely fails") but a different root cause: in ISSUE-118's scenario the Go suite never
  gets EXECUTED to a verdict at all, specific to diffs whose changed files are entirely
  `_test.go` (the built-in `coverage_threshold` step's early skip when there are no in-scope
  production files, plus `test_verification`/`test_substantiveness` only checking test
  presence/substantiveness, never pass/fail). This issue's mechanism is the opposite: the suite
  DOES execute, DOES fail, and DOES produce a real finding — the finding is computed correctly and
  then discarded post-hoc by diff-scope filtering, and it applies regardless of what kind of files
  are in the diff (the repro here is a diff that changed only a production file, `root.go`).
  ISSUE-118's own root-cause analysis does not mention the `go-toolchain` pack's `go-test` engine
  at all, suggesting these were independently-discovered gaps in the same "gate says green, isn't"
  territory. Fixing one does not fix the other — both should stay open until each has its own
  proof-of-fix regression fixture.
- **ISSUE-070** (closed, delivered by PLAN-ISSUE-070) fixed the `pack_engines` step failing to
  apply the diff-scope filter AT ALL to dispatched violations. That fix is what makes THIS issue's
  mechanism ("the filter runs, correctly, on the flag as declared") the residual gap — before
  ISSUE-070, every pack finding leaked past scope; after it, only non-exempt engines' out-of-scope
  findings are (correctly, by design) filtered, and `go-test` is one of them.
- **BUNDLE-003**'s "trustworthy-green guards" seed (delivered as SPEC-068) was checked and does
  NOT own this — that seed's requirements (REQ-021, REQ-022, REQ-026–030, REQ-034) are about
  artifact/schema-cohort validation trustworthiness (content-derived schema identity, artifact
  root resolution, provenance stamping), not test-execution scope-filtering. No open bundle or
  directive obviously owns this; left unhomed for PM triage.

## Why this matters

This directly contradicts the gate's core promise — that PASS means the codebase is genuinely in
a good state, not just that the touched files look fine in isolation. A cross-file breakage
(change to file A silently breaking a test in unrelated, unchanged file B) evading detection under
the DEFAULT and CI-blocking gate invocation is a trust violation in the same family BUNDLE-003's
trustworthy-green work exists to close, and arguably more serious than the artifact-layout
congruence gaps SPEC-068 already fixed, because a failing test is unambiguous ground truth about
code correctness with no interpretation required.

## Direction (not scoped here)

1. Decide whether `go-test` should declare `exempt_from_scope_filter: true` like `go-build`
   (whole-module reasoning: Go test failures, like build failures, can legitimately originate from
   a change to a file the failing test doesn't live in), or whether a narrower mechanism is needed
   (e.g. distinguishing "this test was already red before the diff" from "this diff caused it" via
   a baseline compare, so pre-existing red state doesn't retroactively block unrelated PRs).
2. Decide whether the same audit should be run against every OTHER findings engine that lacks
   `exempt_from_scope_filter` — the flag is declared per-binding, not derived from `gate_type`, so
   any future pack engine could reintroduce this gap silently unless something asserts intent
   engine-by-engine.
3. A regression fixture proving the fix: a tree with a genuinely failing test in a file OUTSIDE
   the current diff scope must turn a diff-scoped `backstop gate` red. Absent that proof, any fix
   risks being another vacuous-green claim, per the fixtures-from-real-output/must-falsify
   convention.

## Notes / references

- Reported during `PLAN-SPEC-070` implementation (2026-08-15), surfaced by running the full
  `go test ./...` suite by hand as a final check, not by any gate dimension.
- Sibling to the gate-verdict-honesty cluster (ISSUE-066, ISSUE-067, ISSUE-091, ISSUE-118): a
  validation signal that reads as complete/authoritative and silently isn't.
