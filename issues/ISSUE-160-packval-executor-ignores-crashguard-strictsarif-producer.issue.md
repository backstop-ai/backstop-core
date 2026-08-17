---
title: "pkg/packval/executor.go's RunEngine ignores a binding's CrashGuard, StrictSarif, and Producer fields — all three honored by the real gate dispatch"
schema_version: issue/v1

issue:
  id: ISSUE-160
  title: "pkg/packval/executor.go's RunEngine ignores a binding's CrashGuard, StrictSarif, and Producer fields — all three honored by the real gate dispatch"
  type: bug
  status: closed
  created: "2026-08-17"
  closed: "2026-08-17"

delivered_by: PLAN-ISSUE-160

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# packval executor ignores CrashGuard, StrictSarif, and Producer

## Problem

`DefaultExecutor.RunEngine` (`pkg/packval/executor.go`, the dispatch behind `backstop pack test` /
`backstop pack check` phase3 fixture validation) never references three `engine.EngineBinding`
fields that the REAL gate dispatch path — `cmd/backstop/pack_gate.go`'s `runFindingsEngine` (and,
for `Producer`, `runCoverageEngine` too) — honors:

```
$ grep -rn "CrashGuard\|StrictSarif" pkg/packval/*.go   # zero matches
$ grep -rn "Producer" pkg/packval/*.go                  # zero matches
```

against the gate's real honoring sites:

- **`binding.CrashGuard`** (`cmd/backstop/pack_gate.go:869`) — for a CrashGuard engine (native
  build/test), a non-zero exit with zero parseable findings is a tool/infra crash, not a
  finding-free pass, and the gate fails loud instead of reading it as clean (SPEC-034
  REQ-003/CLM-010). `RunEngine` has no equivalent: a crashed CrashGuard-declaring engine that
  happens to produce zero findings reads as `Passed: false` with no error — a silent pass on a
  broken run, in the fixture-validation path meant to catch exactly this class of bug in a pack
  before it ships.
- **`binding.StrictSarif`** (`cmd/backstop/pack_gate_golint.go:44`, gated via
  `requireLintSarifShape` at `pack_gate.go`'s convert-guard site) — for a config-file engine that
  assumes a native-SARIF tool (e.g. golangci-lint v2), a too-old binary emitting non-SARIF JSON
  would otherwise parse as zero findings — vacuous green — so the gate fails loud on the shape
  mismatch instead (SPEC-034 REQ-005/CLM-019, Sharp Edge 5). `RunEngine` never calls this guard, so
  a pack fixture exercising a StrictSarif-declaring binding against malformed output cannot
  falsify the shape defect the gate is specifically built to catch.
- **`binding.Producer`** (`cmd/backstop/pack_gate.go:429` in `runCoverageEngine`, `:778` in
  `runFindingsEngine`) — an optional pack-relative script that runs UN-SANDBOXED via the runner IN
  PLACE OF the binding's plain command (ISSUE-045, ISSUE-067). `RunEngine` always builds and runs
  `binding.Command` via `buildEngineArgv`/`exec.Command` directly; it has no producer-substitution
  step at all, so a producer-declaring binding's fixture run invokes the WRONG COMMAND — the plain
  tool the pack author wrote around, not the producer that shapes its input for it.

## Why Producer is the most consequential of the three

The external `backstop-ai/go-toolchain` pack's `go-coverage` rule declares BOTH `stdout_artifact`
AND a `producer:` script together — the producer resolves the module + gofile list and folds them
into the invocation, while `stdout_artifact` is where the real coverage profile lands. `ISSUE-144`
(closed, delivered by `PLAN-ISSUE-144`) just fixed `RunEngine` to read `binding.StdoutArtifact`
correctly. That fix is real for `go-coverage`'s fixture validation only if `RunEngine` also invokes
the RIGHT COMMAND to produce that artifact in the first place — and today it does not: without
producer substitution, `RunEngine` runs the plain `binding.Command` against a fixture, which is not
the invocation the pack was written to require. The one real-world binding whose payload-selection
packval was just fixed to read correctly is also the one whose command packval invokes wrong.

## Impact

A pack fixture that exercises a `CrashGuard`, `StrictSarif`, or `Producer`-declaring binding cannot
reliably falsify the defect class each field exists to catch via `backstop pack test` / `backstop
pack check` phase3 — `RunEngine` silently runs a different (or unguarded) path than the real gate
would, so a pack author sees fixtures pass locally that the real gate dispatch would fail loud on,
or vice versa. This is the same drift family as `ISSUE-092` (manifest-model drift), `ISSUE-140`
(narrow never-started check), `ISSUE-141` (missing Convert stage), and `ISSUE-144`
(StdoutArtifact payload selection) — `pkg/packval`'s dispatch repeatedly diverging from the real
`cmd/backstop/pack_gate.go` dispatch it exists to mirror.

## Direction

Whoever plans the fix should treat this as (at least) three separate falsification mechanics, per
explicit instruction not to fold them into one undifferentiated change:

1. **CrashGuard** — a fixture where the engine exits non-zero with zero parseable findings and
   `binding.CrashGuard` is true; `RunEngine` should fail loud (mirroring
   `pack_gate.go:869`'s guard) instead of returning `Passed: false, err: nil`.
2. **StrictSarif** — a fixture where the engine's output is non-SARIF JSON and
   `binding.StrictSarif` is true; `RunEngine` should invoke the equivalent shape guard
   (`requireLintSarifShape` or an executor-side equivalent) before parsing, rather than silently
   reading zero findings.
3. **Producer** — a fixture on a `Producer`-declaring binding; `RunEngine` should resolve and
   invoke the producer script (mirroring the `packRoot`-relative resolution + un-sandboxed
   `runner.RunStdout` pattern at `pack_gate.go:429-450` and `:778-790`) in place of the plain
   command, with the same declared-but-missing-producer fail-loud behavior. This is the highest
   real-world-impact of the three (see above) and should probably be sequenced first.

Before scoping the fix, check whether `ISSUE-143`'s (`issues/ISSUE-143-packval-gate-convert-dual-implementation.issue.md`)
proposed shared-extraction of the Convert-application stage is underway or landed — if a shared
`pkg/packval` location for gate-mirroring dispatch logic already exists by the time this is picked
up, these three fields likely belong there rather than as a third/fourth/fifth independent copy
inside `RunEngine` directly.

## Notes

- **Origin:** discovered during `PLAN-ISSUE-144`'s implementation (`implementer-issue144`),
  independently confirmed the same night by `implementer-issue142`. Recorded as a residual in
  `ISSUE-144`'s own Resolution section rather than folded into that fix.
- **Sibling, not duplicate, of `ISSUE-143`:** `ISSUE-143` is the STRUCTURAL two-implementation
  problem for the Convert-application stage specifically (`binding.Convert`). This issue is about
  three DIFFERENT fields (`CrashGuard`, `StrictSarif`, `Producer`) that `RunEngine` never
  references at all — a missing-behavior gap, not (yet) a duplicated-implementation one. If
  `ISSUE-143`'s extraction lands first, these fields' fix should target the same shared location,
  but the two issues track distinct problems and neither absorbs the other.
- **Same drift family** as `ISSUE-092`, `ISSUE-140`, `ISSUE-141`, and `ISSUE-144` — repeated
  instances of `pkg/packval`'s dispatch drifting from `cmd/backstop/pack_gate.go`'s real dispatch.
  May fit `DIR-032` ("Gate Verdict Honesty") alongside those siblings; left unslotted here for
  backlog-pm/directive-author triage per this repo's artifact-authoring convention.
- **Existence-in-world check performed 2026-08-17 before filing:** searched `issues/` (`grep -l
  "Producer\|CrashGuard\|StrictSarif" issues/*.issue.md`) and `bundles/` for prior coverage of this
  specific gap. No open issue or bundle charter already owns it. `ISSUE-140` (never-started check),
  `ISSUE-143` (Convert dual-implementation), and `ISSUE-144` (StdoutArtifact) are related siblings
  in the same drift family but are mechanistically distinct — none covers `CrashGuard`,
  `StrictSarif`, or `Producer`.
- **Verified directly against current code 2026-08-17:** `grep -rn "CrashGuard\|StrictSarif"
  pkg/packval/*.go` and `grep -rn "Producer" pkg/packval/*.go` both return zero matches; the gate's
  honoring sites cited above (`pack_gate.go:429,778,869`, `pack_gate_golint.go:44`) were read
  directly to confirm they exist and do what this issue describes.

## Resolution

Delivered by `PLAN-ISSUE-160` (`status: completed`, committed at `c1e4ef4`/`a31d0ed`). This closes
the last remaining open member of `DIR-032`'s 21-item roster.

`pkg/packval/executor.go`'s `RunEngine` now honors all three previously-ignored `EngineBinding`
fields, landed as three sequential TDD cycles matching the gate's own stage order:

- **`Producer`** — a declared producer script is now substituted for the invoked command, with the
  full ordered argv preserved (findings-path semantics), avoiding the PLAN-ISSUE-067-class bug where
  a producer receives the wrong subcommand as `$1`. Sequenced first per this issue's "Direction"
  section, as the highest real-world-impact of the three (the `go-toolchain` `go-coverage` rule).
- **`CrashGuard`** — `RunEngine` now distinguishes a genuine clean pass from a non-zero-exit-with-
  zero-findings run, mirroring `pack_gate.go:869`'s guard, instead of silently returning
  `Passed: false, err: nil`.
- **`StrictSarif`** — non-SARIF payload noise now fails loud before parsing instead of silently
  reading as zero findings, mirroring `requireLintSarifShape`.

All three landed inline in `RunEngine` (no helper extraction), matching the gate's own stage order:
trust gate → buildEngineArgv → producer substitution → exec → never-started refusal →
stdout_artifact selection → strict-SARIF shape guard → convert → ParsePackFindings → crash guard →
return.

Falsified via mutation testing on the finished implementation: ten regressions were introduced one
at a time against the delivered code. One real gap was found — dropping the crash guard's
`runErr != nil` conjunct — and closed by adding a second subtest leg to an existing mandated test,
keeping the mandated set at exactly 16 rather than inventing a 17th.

Two judgment calls were made and independently re-verified across two review rounds:

1. **StrictSarif lands locally in `pkg/packval`**, not in a shared gate-mirroring location, because
   `ISSUE-143`'s proposed Convert-stage consolidation has no plan yet as of this close.
2. **Producer substitution uses findings-path argv-swap semantics** specifically (mirroring
   `runFindingsEngine`'s pattern at `pack_gate.go:778-790`, not `runCoverageEngine`'s bare producer
   invocation at `:429-450`).

**Residual not fixed here (R4):** the external `go-toolchain` pack still needs fixtures and a rule
source path for these three fields to actually fire in practice against a real pack. Tracked
separately, not blocking this close.
