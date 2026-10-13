---
title: "Go Test Engine Cmd Backstop Exceeds Default Timeout"
schema_version: issue/v1

issue:
  id: ISSUE-220
  title: "Go Test Engine Cmd Backstop Exceeds Default Timeout"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Go Test Engine Cmd Backstop Exceeds Default Timeout

## Problem

The `cmd/backstop` test package now takes longer than Go's default 10 minute per-package test timeout
when run through the gate, so the go-toolchain `go-test` engine reports a crash and no test results at
all.

Mechanism, verified 2026-10-12 in the installed `backstop-ai/go-toolchain` pack
(`.backstop/packs/backstop-ai/go-toolchain/pack.yml`):

- The `go-test` engine runs `go test -coverprofile=cover.out` against `./...`. The `go-coverage` engine
  (~line 47) runs the same suite under the single-run convention (ISSUE-172). Neither command passes
  `-timeout`, so Go's default of 10 minutes per package applies.
- Neither the pack manifest nor `pkg/config` exposes a timeout setting (no `timeout` key in either), so a
  consumer cannot raise it from `backstop.yml`.
- Observed: `panic: test timed out after 10m0s` and
  `FAIL github.com/backstop-ai/backstop-core/cmd/backstop 603.707s`. An earlier full non-race run took
  about 14.5 minutes under load.
- A timeout panic is not parseable as test findings, so the engine reports
  `crashed: non-zero exit with no parseable findings: exit status 1` and `pack_engines` fails.

### Reproduction

On a developer Mac, run `backstop gate --all` (or any run that executes the `go-test` engine over
`./...`) in backstop-core. The `cmd/backstop` package exceeds 10 minutes and the engine crashes.

## Impact

- The gate cannot go green on a developer machine: `pack_engines` fails every time the package is slow.
- Failure is closed, not a vacuous green, but it is uninformative. Not a single test result is reported,
  so real test failures in other packages and in `cmd/backstop` itself are masked by the crash.
- The failure reads as "crashed", which sends a reader hunting for a defect rather than a slow suite.

## Root cause

1. The package is large and still growing. Recent issues each added tests to `cmd/backstop`, and CPU
   contention on a loaded machine stretches wall time further.
2. The pack hardcodes a `go test` command with no `-timeout`, and no configuration surface exists to
   change that.
3. The engine does not distinguish a timeout panic from a crash, so the cause is not named.

## Direction

(Direction, not a plan. Options are not exclusive.)

1. Pack (`backstop-ai/go-toolchain`, a separate repo that owns this change): add `-timeout` to the
   `go-test` and `go-coverage` commands, either configurable through pack config or with a sensible higher
   default. Keep the two commands consistent, since the single-run convention reuses the profile.
2. Suite: identify the slowest `cmd/backstop` tests with `go test -json` timing, then speed them up or
   split the package so no single package approaches the limit.
3. Core (optional): expose a per-engine timeout, and surface a timeout distinctly from a crash in the
   engine error (`pkg/packval/executor.go`, `cmd/backstop/pack_gate.go`), for example
   `timed out after 10m0s` instead of `crashed: non-zero exit with no parseable findings`.

Keep each change small and contained.

## Notes

- Related: ISSUE-172 (single-run convention shared by `go-test` and `go-coverage`); ISSUE-209 (pack
  engine dispatch error discards findings), the same class of evidence loss where the real cause is
  hidden.
