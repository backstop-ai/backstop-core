---
title: "Gate Double Runs Test Suite"
schema_version: issue/v1

issue:
  id: ISSUE-068
  title: "Gate Double Runs Test Suite"
  type: technical-debt
  status: open
  created: "2026-07-18"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Gate Double Runs Test Suite

## Problem

A single `backstop gate --all` runs the project's full test suite TWICE, concurrently:

- the TEST path (`pack_engines` → the `ts-test` engine, `vitest run`) runs the whole suite for
  test results/verification, and
- the COVERAGE path (`coverage_threshold` → the `ts-coverage` engine's producer, `vitest run
  --coverage`) runs the whole suite again for coverage records.

They run concurrently. For a TS suite with heavy pglite (WASM Postgres) DB tests, two
overlapping full runs double the concurrent WASM boots and thrash CPU/memory.

Discovered 2026-07-18 while delivering bclabs-portal — the first non-Go, TypeScript consumer of
backstop. Source detail: bclabs-portal ISSUE-001 investigation.

**Evidence (bclabs-portal):**

- In `gate --all`: `pack_engines` ≈ 174s and `coverage_threshold` ≈ 174s — near-identical and
  overlapping.
- A single standalone `vitest run --coverage` over the same suite: ≈ 86s.
- So the gate pays ~2x the single-run cost, concurrently. The concurrency ALSO forced the
  consumer to cap vitest parallelism (`poolOptions.forks.maxForks = 2` in `vitest.config.ts`)
  purely to stop the two overlapping runs from timing out (`beforeAll` boots exceeding 30s). With
  a single run, no cap would be needed.

This is a genuine defect, not a nice-to-have: it wastes roughly 2x wall-clock on the gate's test
step AND forces consumers to cap test parallelism to avoid timeouts, which is itself a workaround
for the gate's own redundant concurrency.

**Framing:** this is pure de-duplication — the same tests run, once; no change to which tests
run or to the pass/fail/coverage verdicts. It does NOT reduce test/coverage scope and must not be
mistaken for check-filtering (see [[feedback_no_check_filtering]]).

## Root cause

`vitest run --coverage` already produces BOTH per-test results AND coverage in one pass. Running
a separate `vitest run` (the `ts-test` engine) for test results is redundant. The `ts-test` and
`ts-coverage` engines (in the `backstop/typescript-toolchain` pack) don't share a run, and the
gate dispatches them independently/concurrently.

## Generality

NOT a single "one run feeds both" principle — that framing over-generalizes a TS-specific fact
into core. Backstop shells engines out to packs; core sees `ts-test` and `ts-coverage` (or any
other toolchain's equivalents) as two independently-declared commands. Core CANNOT know they are
"the same tool" without encoding that relationship, which is exactly the baked language/tool
knowledge the thin-executor rule forbids (see [[feedback_zero_baked_checks]]).

Whether test and coverage can share one invocation depends on the toolchain, and splits into three
buckets:

- **Same tool, one invocation** (Go `go test -coverprofile`, TS/JS `vitest --coverage`, .NET
  `dotnet test --collect`) — consolidation is possible; where a pack declares it, that pack gets
  the full ~2x win.
- **Separate tool composed into one run** (Python pytest + coverage.py, Java + a JaCoCo agent,
  Ruby + SimpleCov) — often declared as two engines; consolidation only happens if the pack wires
  one invocation that emits both.
- **Genuinely separate invocation or build** (Rust `cargo llvm-cov` re-runs instrumented; C/C++
  needs an instrumented BUILD then a run) — coverage cannot be folded into the plain test run at
  all.

Coverage may also be ABSENT entirely — the gate already models coverage as a dimension that can be
`capability_absent` when a pack ships a test engine and no coverage engine. "Both exist and share a
run" is never a safe assumption core can make.

## Fix direction

The fix SPLITS across two layers because only one of them is safe to bake into core:

1. **Core-universal, assumes nothing — SERIALIZE.** Core controls step concurrency in the gate, so
   it can run the test and coverage engines sequentially instead of overlapping, for EVERY
   toolchain, regardless of whether test and coverage share a tool. This removes the concurrency
   thrash (and the need for a consumer-side workaround like a vitest fork-cap) universally. It does
   NOT remove the redundant work — it's the safe floor, not the full win, but it is the part that
   is genuinely general.
2. **Pack-declared, opt-in, tool-specific — ONE COMBINED ENGINE FEEDS BOTH.** Where a toolchain
   genuinely produces test-results AND coverage from a single invocation (Go, TS, .NET — see
   Generality above), the PACK declares a single engine whose one run's output splits into (a) the
   test-results SARIF/findings that `test_verification`/`pack_engines` consumes and (b) the
   coverage-records `coverage_threshold` consumes. This is the real ~2x win — but it is a pack
   capability available only where genuinely true, never a core assumption. Backstop-core stays
   ignorant of which toolchains can do this; it only runs what the pack declares.

**Locus (partly resolved):** SERIALIZE is core — gate step-concurrency orchestration between the
`pack_engines` and `coverage_threshold` steps. CONSOLIDATION is pack — the toolchain pack declaring
a combined engine (e.g. `backstop/typescript-toolchain`'s `ts-test`/`ts-coverage`, as a candidate
first adopter once the core serialize fix exists). The remaining open diagnostic is HOW the
double-run/concurrency is produced today — core step-orchestration dispatching two engines
concurrently, vs. the pack declaring two independent engines with no shared-run option — resolve
this at plan time so the serialize fix lands on the actual concurrency control point. Look at:

- backstop-core gate orchestration: how the `pack_engines` and `coverage_threshold` steps dispatch
  their engines, and whether/where that dispatch is concurrent.
- `backstop/typescript-toolchain` pack: the `ts-test` and `ts-coverage` engine definitions
  (commands, `producer`/`convert` scripts, `coverage-produce.sh`) — the candidate for a
  pack-declared combined engine, once core's serialize fix exists.

## Acceptance

- Core serialization removes the concurrency thrash for ALL toolchains, packs-only: consumers can
  drop test-parallelism caps (e.g. a vitest fork-cap) without hitting timeouts. This is the
  universal, always-delivered outcome — it ships once in core and applies to every pack.
- For toolchains whose pack declares a combined test+coverage engine, the suite runs ONCE per gate
  (gate test-time ≈ a single combined invocation, not two) — the ~2x wall-clock win, delivered
  per-pack as packs adopt it, not globally guaranteed by core.
- No change to which tests run or to pass/fail/coverage verdicts in either layer — pure
  de-duplication/de-thrash, never check-filtering (see [[feedback_no_check_filtering]]).
- Coverage-absent (a pack with a test engine and no coverage engine) stays a graceful
  `capability_absent` WARN, not an error — core must not assume coverage exists.

## Impact

Roughly halves gate wall-clock on test-heavy TypeScript projects and removes the concurrency
thrash that currently forces consumers to cap test parallelism.

## Notes / references

- Related (context only, already handled portal-side): bclabs-portal ISSUE-001 shared one pglite
  per test file instead of per-test (standalone suite 242s→86s, ~2.8x; auth ~5x). The portal
  RETAINED its fork-cap (decision 2026-07-18) precisely BECAUSE of this core double-run — so this
  core fix is what would let the portal drop its cap.
- Ties to the recipe/portal capture-first work (BUNDLE-015/016) as the first non-Go consumer
  surfacing a defect that's general across toolchains, not TS-specific.
