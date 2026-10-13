---
title: "Types Only Files Fail Coverage No Measurement"
schema_version: issue/v1

issue:
  id: ISSUE-221
  title: "Types Only Files Fail Coverage No Measurement"
  type: bug
  status: open
  created: "2026-10-13"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Types Only Files Fail Coverage No Measurement

## Problem

A source file that holds only TypeScript types and interfaces has no runtime statements. When a diff
touches such a file, the gate fails `coverage_threshold` with a blocking `coverage_unmeasured`
violation. A file with zero executable statements has nothing to cover, so this is a false failure.

Observed 2026-10-13 in a TypeScript consumer repository's bare (diff-scoped) gate. The repository
adopts `backstop-ai/typescript-toolchain@1.3.1`, and the gate was built from backstop-core `dacfd7c`.
A types-only `src/contracts.ts` was in the diff and produced:

```
coverage_unmeasured: no coverage measurement for in-scope changed measurable-source file
src/contracts.ts (any metric) and it is not pack-declared excluded — refusing to pass with
nothing measured
```

### Mechanism (verified against the source)

1. **Core already treats `Total==0` as N/A.** The failure comes from a missing record.
   `pkg/gate/step_coverage.go` has two separate paths:
   - Record present with `Total == 0`: skipped as N/A, never a 0% failure (~line 222, `if
     record.Total == 0 { continue }`, CLM-026).
   - No record for the path at all: blocking `coverage_unmeasured` (~lines 158-172). This is the path
     that fires here.
   - `CoverageRecord.Measured` is never read by the verdict. The pack's convert always sets it `true`.
2. **There is no record because the producer never emits one.** The consumer's `coverage-summary.json`
   (the v8 json-summary written to `.backstop/ts-coverage/`) has no entry for `src/contracts.ts`. The
   record is absent, not present with 0/0. `scripts/coverage-to-records.sh` maps entries one to one,
   so no input entry means no record.
3. **`--coverage.all` no longer does anything.** The pack passes `--coverage.all` in the `ts-test`
   command (`pack.yml`) and in the `scripts/coverage-produce.sh` fallback. The producer's comment
   says the flag makes untested files "surface as 0%-covered". The consumer runs Vitest 5.0.3 and
   `@vitest/coverage-v8` 5.0.3. Vitest 4 removed `coverage.all`, and the CLI ignores the unknown
   flag. In `@vitest/coverage-v8/dist/provider.js` (~line 76), untested files are merged into the
   coverage map only when `coverage.include` is set. The pack does not set it and the consumer has no
   vitest config. So the report holds only modules that tests actually load.
4. **A types-only module is never loaded.** Type-only imports are erased at transpile time, so no test
   ever loads `src/contracts.ts`. V8 has no script coverage for it, so it has no entry and no record.

### The same gap hides real untested code

The same mechanism hit a non-type file in the same run. A runtime stub, `src/intake.ts`, is imported by
no test yet, and it also showed up as `coverage_unmeasured` instead of as a 0% coverage gap. The gate
still fails closed, so this is not a vacuous green. But the pack's stated guarantee, that an untested
file shows as 0%, is silently gone on Vitest 4 and later. Two different cases now produce the same
message: "trivially covered (no statements)" and "real code with no tests".

## Impact

- Any diff that touches a types-only module (contracts, DTOs, shared type definitions) blocks the gate,
  even though the gate has nothing to measure in it. The only workarounds are a pack-declared
  exclusion or a waiver, and both are ceremony for a non-finding.
- On Vitest 4 and later, untested runtime files report as "unmeasured" instead of "0% below
  threshold", so the cause is misnamed.
- Fixtures executed as child processes (ISSUE-222) hit this same no-record path.

## Root cause

1. The pack depends on `--coverage.all`, which Vitest 4 removed. On current Vitest it is a silent
   no-op, so files no test loads get no coverage entry.
2. Core cannot tell "no executable statements" from "never loaded" when no record exists. It knows no
   languages (thin executor), so it cannot parse the file to find out. Failing closed on a missing
   record is correct core behavior (SPEC-043 REQ-003). The defect is that the producer drops the
   record.

## Direction

(Direction, not a plan.)

1. **Pack (`backstop-ai/typescript-toolchain`, primary fix).** Replace `--coverage.all` with explicit
   `--coverage.include` globs that mirror the pack's `classification.source` globs, plus
   `--coverage.exclude` globs that mirror `classification.test`. Make the same change in the `ts-test`
   command and in the `coverage-produce.sh` fallback, so the single-run convention (ISSUE-172) stays
   consistent. With `include` set, v8 generates entries for files no test loaded:
   - A types-only module gets an entry with 0 statements. The convert emits a record with
     `total: 0`, which core already treats as N/A and passes.
   - An untested runtime file gets a 0/N entry, which restores the "0% below threshold" verdict.
   - **To verify empirically before relying on this:** that Vitest 5's json-summary really emits a
     `statements.total: 0` entry for a types-only module under `include`, and does not omit it. Add a
     pack fixture (`backstop pack test`) with a types-only `.ts` file and assert that it produces a
     `total: 0` record.
2. **Sequencing.** Land ISSUE-222's classification fix first, or in the same release. Once `include`
   is set, fixture `.cjs` files classified as source turn from `coverage_unmeasured` into 0%
   `coverage_threshold` failures.
3. **Core (secondary, optional).** Keep failing closed on a missing record. The `coverage_unmeasured`
   message could add a hint that a module with no runtime statements, or one no test loads, may be
   missing because the coverage producer only reports loaded files. This must not become a pass.
   Confirm that a regression test pins "record present, `Total==0`, in-scope changed ⇒ pass" through
   the full scoped step (existing CLM-026 tests in `pkg/gate/coverage_unmeasurable_test.go` /
   `step_coverage_test.go`), so the pack-side fix can rely on it.

## Notes

- The pack fix lives in the `backstop-ai/typescript-toolchain` repository (currently 1.3.1, tag
  `v1.3.1`). It should ship as a release with a matching `v<version>` tag, and consumers then run
  `backstop pack update`. Consumers on Vitest 3 or earlier still accept `--coverage.all`, so check
  whether `--coverage.include` behaves the same on the oldest Vitest the pack supports.
- Related: ISSUE-222 (fixtures classified as source, same `coverage_unmeasured` path); ISSUE-045
  (earlier unmeasurable-file false positives); ISSUE-172 (single-run coverage convention); ISSUE-179
  (coverage reuse freshness check).
