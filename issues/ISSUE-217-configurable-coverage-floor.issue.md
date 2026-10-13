---
title: "Configurable Coverage Floor"
schema_version: issue/v1

issue:
  id: ISSUE-217
  title: "Configurable Coverage Floor"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Configurable Coverage Floor

## Problem

The fallback per-file coverage threshold is a constant compiled into core, and nothing in
`backstop.yml` or any pack can change it.

`pkg/gate/step_coverage.go:29`:

```go
const defaultCodeScopeCoverageFloor = 90
```

(introduced in commit e675e8a, 2026-05-27). It is the threshold applied whenever no in-scope spec
with status `implemented` declares `verification.coverage_threshold`.

### Verified behavior

Threshold resolution lives in `coverageThresholdsForScope` (`pkg/gate/step_coverage.go`, line 521).
In bare/diff mode every changed source file is measured, and the governing threshold is resolved in
this order:

1. Specs whose own file is in the diff (`scope.Contains(spec.File)`) govern; their declared
   thresholds are used (strictest wins). The floor is not consulted on this path.
2. Otherwise the scope collapses to a single code-scope threshold: the max `CoverageThreshold` over
   specs whose `implementation` package matches a changed file's directory; if none match, the max
   over specs with no specific package (root fallback).
3. If that max is still 0, the hard-coded `defaultCodeScopeCoverageFloor` (90) is used
   (line 555-557).

Asymmetry in `--all` mode: when `scope == nil` or `scope.Mode == GateScopeModeAll`, the function
returns all specs as-is (lines 522-524) and never reaches the floor. `coverageThresholdForMetric`
then returns the max declared threshold, which is 0 when no spec declares one, and the caller skips
the below-bar check for that metric (`if threshold <= 0`, line 202). So the same repository with no
qualifying spec is held to 90 on a diff-scoped gate but to nothing on `gate --all`. This is itself
surprising: the full sweep is more lenient than the incremental one.

Not changeable by configuration: `pkg/config/config.go` has no coverage key (the `Enforcement`
struct carries `waiver_warning_days`, `baseline_ttl`, `test_command`, `toolchain`, `policy`,
`security`), and no pack setting reaches this constant.

## Impact

- A TypeScript consumer repository adopting packs on existing code cannot express a project-level
  coverage bar. Any legacy file touched by a diff is held to 90 regardless of the project's reality,
  so adoption either blocks on unrelated legacy debt or forces a spec per package just to lower the
  number.
- This is policy baked into core, contrary to the thin-executor principle (core runs; config and
  packs carry policy).
- The diff vs `--all` asymmetry makes results hard to predict for adopters.

## Root cause

A policy value was introduced as a core constant instead of a config-carried setting.

## Direction

(Direction, not a plan.)

1. Add a project-level key in `backstop.yml`, `enforcement.coverage_floor: <int 0-100>`, following
   the existing `Enforcement` struct conventions (snake_case yaml key, `omitempty`, defaulting in
   the config loader near `WaiverWarningDays`). The exact name is to be confirmed against
   `pkg/config/config.go` conventions in the plan.
2. Default 90 when unset, so there is no behavior change for existing consumers. Distinguish unset
   from an explicit 0 (for example a pointer or presence check), because 0 is a meaningful value
   ("no floor").
3. Thread the configured value to every place the constant is used today (a single site, line 556,
   via the `StepCoverageThresholdScopedFunc` call at `cmd/backstop/gate.go:1807`), replacing the
   constant with the resolved value.
4. Validate the range: reject values outside 0-100 and non-integers with a clear config error.
5. Spec-declared thresholds keep precedence per the existing resolution order above.

### Open points for the plan

- Whether a spec may declare a threshold below the project floor. Recommendation: yes, the spec
  wins, as today (the floor is only a fallback when no qualifying spec declares a threshold).
- Whether `gate --all` should use the project floor instead of 0 when no spec qualifies. Related
  but a behavior change for existing consumers (the full sweep would start failing files it
  currently skips), so it is likely better as a separate issue; the plan should decide and, if
  deferred, record the asymmetry as known.

## Requirements

- REQ-001: `enforcement.coverage_floor` is an optional integer in `backstop.yml`; when unset the
  fallback threshold is 90.
- REQ-002: When set, the configured value replaces the hard-coded fallback for changed files with
  no qualifying spec threshold.
- REQ-003: A spec-declared `coverage_threshold` that qualifies under the existing resolution order
  takes precedence over the configured floor.
- REQ-004: Values outside 0-100, or non-integer values, are rejected at config load with an error
  naming the key.

## Acceptance tests (mandated names)

- `TestCoverageFloorUnsetDefaultsTo90`
- `TestCoverageFloorConfiguredUsedForFilesWithoutSpec`
- `TestCoverageFloorSpecThresholdStillWins`
- `TestCoverageFloorRejectsInvalidValues`

## Notes

- Evidence for the consumer scenario is intentionally stated generically; this repo is public.
- Scope is small and contained: one config field, one validation, one call-site substitution.
