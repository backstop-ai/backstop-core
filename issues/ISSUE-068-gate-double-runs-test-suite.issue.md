---
title: "Gate Double Runs Test Suite"
schema_version: issue/v1

issue:
  id: ISSUE-068
  title: "Gate Double Runs Test Suite"
  type: technical-debt
  status: closed
  created: "2026-07-18"
  closed: "2026-07-18"

resolved-by: 60a1316

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

**Corrected 2026-07-18 by code-grounded investigation — the original "serialize / concurrency"
framing below was WRONG and is replaced, not extended.**

The gate runs FULLY SEQUENTIALLY. There is ZERO concurrency anywhere in `pkg/` — no goroutines,
no `errgroup`, no `WaitGroup`; the gate step loop (`pkg/gate/gate.go:146`) is a plain sequential
`for`. So there is no concurrent double-run and nothing to "serialize" — that framing was an
incorrect inference and is dropped entirely.

The portal's vitest fork-cap (`maxForks = 2`) addresses vitest's OWN within-run parallelism (many
workers each booting pglite), which is independent of this issue. It is not evidence of a backstop
concurrency bug.

The double-run is redundant WORK forced by the engine MODEL, not by orchestration. Engines are
declared as a name-keyed map (`Engines map[string]EngineSpec`, `pkg/pack/manifest.go:29`), each
carrying a scalar `gate_type` (`pkg/pack/engine/binding.go`). The `test` dimension
(`pack_engines`/test-verification) and the `coverage` dimension (`coverage_threshold`) are
therefore filled by TWO DISTINCT engines — e.g. `ts-test` runs `vitest run`, `ts-coverage`'s
producer runs `vitest run --coverage` — and EACH runs the full suite. One engine = one command =
one run; two engines = two runs. That is the whole cause.

## Generality

Language-agnostic and OPT-IN. Any toolchain whose test results and coverage come from ONE tool
invocation (Go `go test -coverprofile`, TS vitest, .NET `dotnet test --collect`) can declare the
shared run-key described in Fix direction below. Toolchains where coverage needs a genuinely
separate build/run (Rust `cargo llvm-cov` re-runs instrumented; C/C++ instrumented build) simply
don't declare it and keep two runs — no regression. Coverage-absent packs are unaffected (graceful
WARN); the gate already models coverage as a dimension that can be `capability_absent` when a pack
ships a test engine and no coverage engine.

## Fix direction

**Option C — declared shared run-key (founder-chosen 2026-07-18).**

Add a small DECLARED field to the engine binding letting two engines state they SHARE ONE run (a
`run_group`/shared-run key). At dispatch, core MEMOIZES the run by that declared key: run the
command ONCE, then feed that single run's output to EACH participating engine's own convert
script — the test engine's convert extracts test findings, the coverage engine's convert extracts
coverage records, both from the one output. Core dedupes by the OPAQUE declared key and NEVER
inspects or understands the commands (thin-executor / DD-3 — no command-string sniffing, no tool
comprehension). A pack opts in by declaring the shared key on its test + coverage engines and
pointing them at one superset command (e.g. `vitest run --coverage --reporter=json`).

Explicitly NOT:

- NOT serialize — nothing is concurrent, so there's nothing to serialize.
- NOT widening `gate_type` to a set — the engines stay DISTINCT, each keeps its own `gate_type`
  and convert; they merely share a RUN. Chosen over command-string equality (fragile,
  whitespace-sensitive, and the commands genuinely differ) because engines already have declared
  identity, so dedupe by DECLARED identity, not by sniffing commands — explicit, whitespace-proof,
  can't silently rot.
- Still pure de-duplication: same tests run once, same pass/fail/coverage verdicts;
  coverage-absent stays a graceful WARN. Never check-filtering.

**Scope split:**

- **backstop-core** (this issue's plannable slice): the new binding field + run-memoization at the
  dispatch layer + fan-out of one run's output to multiple converts + manifest validation + tests.
- **backstop-packs** (separate follow-on in that repo): `backstop/typescript-toolchain` (and
  analogously `go-toolchain`) declares the shared run-key on its test + coverage engines and
  unifies their command. This is a dependent follow-on, not part of the core plan.
- **Timing note:** doing this schema change now is deliberately cheap — the pack surface is still
  small, so few packs need migrating.

## Acceptance

- With the shared run-key declared, a toolchain's suite runs ONCE per gate (gate test-time ≈ a
  single `vitest run --coverage`, not two), output fanned to both the test-verification and
  coverage paths.
- No shared key declared ⇒ unchanged two-run behavior (safe default; no regression for
  separate-build toolchains).
- Unchanged invariants: same tests run, same pass/fail/coverage verdicts (pure de-dup, not
  check-filtering); coverage-absent stays a graceful WARN.
- Core dedupes only by the opaque declared key — no command inspection (thin-executor preserved).
- Consuming projects (e.g. bclabs-portal) can drop their vitest fork-cap once the redundant run is
  gone.

## Impact

Roughly halves gate wall-clock on test-heavy TypeScript projects and removes the concurrency
thrash that currently forces consumers to cap test parallelism.

## Resolution

The double-run defect is FIXED, but NOT by this issue's own fix-direction (Option C, the declared
`run_group` shared-run-key mechanism). That core mechanism was built, shipped green, then REMOVED
on 2026-07-18. The actual fix lives entirely pack-side.

**Core mechanism (Option C) — built, then removed:**

The `run_group` shared-run-key + memoized-fan-out mechanism described in Fix direction was
implemented in backstop-core and its own tests passed. It was validated only against a FABRICATED
fixture shaped to fit the design: two engines with byte-identical commands sharing one stdout
payload. That fixture never resembled a real toolchain.

The real consumer — `backstop/typescript-toolchain`'s `ts-test` and `ts-coverage` engines —
exposed the gap immediately: the two engines have DIFFERENT commands (`vitest run` vs `vitest run
--coverage`), a producer asymmetry (only the coverage side runs a producer step), and emit TWO
DISTINCT FILE artifacts (a test report and a coverage summary), not one shared stdout payload. The
mechanism's own coherence check rejected the real pairing on the FIRST field it inspected.

A survey across other real toolchains (Go `go test -coverprofile`, pytest, JaCoCo, .NET `dotnet
test --collect`) found NONE with the shared-stdout shape the mechanism was built for — every one
of them emits distinct FILE artifacts from a single run, not a shared stdout stream. The fixture
that validated the mechanism was fictional; no real consumer would ever satisfy it.

Root lesson: WHICH flags combine into one command and WHICH files a run produces is TOOLCHAIN
knowledge. A tool-blind core structurally cannot recognize "same tool, combinable flags,
extractable from these two files" — that is exactly the class of fact the thin-executor rule
(DD-3, [[feedback_zero_baked_checks]]) reserves for packs. Baking a generalized shared-run
mechanism into core, even one that dedupes only by an opaque declared key, still required core to
model a *shape* of tool relationship (shared stdout) that doesn't hold for any real toolchain. The
mechanism was removed from backstop-core on 2026-07-18 rather than reshaped to fit files, because
the fix that actually fits belongs entirely in the pack.

**Actual fix — pack-side, `backstop/typescript-toolchain` v1.1.0 (commit 60a1316, that pack's own
repo):**

- The `ts-test` engine's command now carries the coverage flags directly, so a SINGLE `vitest run`
  invocation emits BOTH the test report and the coverage summary as its two file artifacts — no
  second full-suite run for coverage.
- The `ts-coverage` engine's producer gained a reuse-if-fresh check: if the coverage summary file
  is not older than the test report file, it reuses the existing summary instead of re-running
  vitest. If the freshness check fails (e.g. the coverage step runs standalone, out of gate order),
  it falls back to running vitest itself — a degraded-but-correct path, never a silent gap.
- Proven on bclabs-portal's real gate: `coverage_threshold` step time collapsed from ~174s to
  ~0.75s, the gate PASSES, and there is zero change in violations/verdicts. The portal's
  `vitest.config.ts` `poolOptions.forks.maxForks = 2` fork-cap — the workaround this issue names as
  removable — is eliminated as a result.

**Follow-on (not scoped here):** the same combined-run convention (one native run emits both
artifacts + a freshness-reuse check on the second consumer) can be adopted by `go-toolchain`
(`go test -json -coverprofile=...` feeding both `go-test`'s test results and the existing
`coverage-produce.sh` reuse check). Not filed as a separate issue; noted for whoever next touches
go-toolchain's coverage path.

**Acceptance — met, via the pack convention, not the core mechanism originally proposed:**

- Suite runs ONCE per gate on the real consumer (bclabs-portal): confirmed, ~0.75s vs ~174s.
- Same tests run, same pass/fail/coverage verdicts: confirmed — zero change in gate violations.
- No core assumption about tool relationships: confirmed — core carries NO run_group/shared-run
  mechanism; the convention (one combined command + freshness-reuse on the second consumer) lives
  entirely in the pack's own engine bindings and producer script.
- Separate-build toolchains unaffected: confirmed by construction — the convention is a per-pack
  opt-in (a pack chooses to combine its own commands and add its own reuse check); no unrelated
  pack changed shape.

**Backing plan:** `PLAN-ISSUE-068-engine-shared-run-key-dedup.plan.yml` is `obsoleted`
(delivered-then-removed) — it built and proved the core Option C mechanism this issue originally
specified, which was then removed per the post-mortem above. No backing core plan delivers this
close; the real fix has no backstop-core commit at all — see `resolved-by`, which points at the
pack-side commit instead.

**Accepted residual:** the go-toolchain follow-on noted above is real but small and unscoped;
tracked here as a note, not filed as a separate issue, since it is optional per-pack adoption with
no forcing deadline.

## Verification

- bclabs-portal real gate run: `coverage_threshold` step ~174s → ~0.75s, gate PASS, zero change in
  violations.
- `backstop/typescript-toolchain` v1.1.0 (commit 60a1316 in that pack's own repo) is the resolving
  change; its verification lives in that pack's repo/tests, not backstop-core's.
- `./bin/backstop artifact validate issues/ISSUE-068-gate-double-runs-test-suite.issue.md` —
  schema-valid.

## Notes / references

- Related (context only, already handled portal-side): bclabs-portal ISSUE-001 shared one pglite
  per test file instead of per-test (standalone suite 242s→86s, ~2.8x; auth ~5x). The portal
  RETAINED its fork-cap (decision 2026-07-18) precisely BECAUSE of this core double-run — so this
  core fix is what would let the portal drop its cap.
- Ties to the recipe/portal capture-first work (BUNDLE-015/016) as the first non-Go consumer
  surfacing a defect that's general across toolchains, not TS-specific.
- Surfaced ISSUE-069 (ast-grep null-ruleId jq crash) incidentally while verifying this issue on
  jq 1.8.1 — unrelated defect, fixed separately.
- Resolved by `backstop/typescript-toolchain` v1.1.0, commit 60a1316 (that pack's own repo, not
  backstop-core). Backing core plan `PLAN-ISSUE-068-engine-shared-run-key-dedup.plan.yml` is
  `obsoleted` — see Resolution.
