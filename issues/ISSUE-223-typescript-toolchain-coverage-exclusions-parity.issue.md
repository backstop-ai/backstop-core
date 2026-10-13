---
title: "Typescript Toolchain Coverage Exclusions Parity"
schema_version: issue/v1

issue:
  id: ISSUE-223
  title: "Typescript Toolchain Coverage Exclusions Parity"
  type: bug
  status: open
  created: "2026-10-13"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Typescript Toolchain Coverage Exclusions Parity

## Problem

`backstop-ai/typescript-toolchain@1.3.2` has no way for a project to exclude a file that coverage
cannot measure. Every coverage record it emits has `excluded: false`. A file that v8 coverage cannot
see therefore always fails the gate as a 0% gap, however well it is tested.

A TypeScript consumer repository using Temporal workflows hit this first. Temporal bundles workflow
modules and runs them inside its isolated workflow VM, not in the Vitest process, so v8 coverage
never sees the statements a workflow test executes. Since 1.3.2 the `--coverage.include` globs
(ISSUE-221) give each such module a `0/N` summary entry. Core then reports it as a
`coverage_threshold` failure. The consumer's only options are a waiver or test-only contortions, and
the backstop rules forbid using either to turn a gate green.

`backstop-ai/go-toolchain` already solves this, with tracked, justified exclusions (verified in the
installed pack's `scripts/coverage-produce.sh` and `scripts/coverage-to-records.sh`, and in
`cmd/backstop/go_toolchain_coverage_exclusions_test.go`):

- The consumer declares exclusions in a tracked `.backstop/coverage-exclusions` file. Each line is
  `<repo-relative-path><TAB><justification>`. Blank lines and `#` comments are ignored.
- The un-sandboxed producer validates each line. A line with no justification, a malformed path
  (absolute, trailing `/`, or an empty, `.` or `..` segment) or a forbidden control character is
  **dropped**: it is not honoured and it is not fatal, so the path stays measured. A stderr notice
  reports each dropped line.
- The sandboxed convert marks a listed path's record `excluded: true` and puts the justification in
  `justification`. A declared path with no inventory record gets one synthetic excluded record
  (`covered: 0`, `total: 0`, `measured: false`; ISSUE-186). A duplicate declaration keeps its first
  position and its last justification.
- Core (`pkg/gate/step_coverage.go`, ~lines 200-215) skips the threshold for an excluded record.
  For an in-scope changed file it reports a non-blocking `coverage_exclusion` warning, and
  `coverageExclusionReason` appends the record's `Justification` (`pkg/check/coverage.go`,
  `json:"justification,omitempty"`) to that warning.

The TypeScript pack has none of this. Its coverage records come from the ts-test engine's
json-summary, through `scripts/coverage-produce.sh` (it reuses the summary when fresh and otherwise
runs Vitest itself) and then `scripts/coverage-to-records.sh`. The convert hardcodes
`excluded: false` and nothing reads `.backstop/coverage-exclusions`.

## Impact

- A TypeScript consumer cannot ship a structurally unmeasurable module, such as a Temporal workflow,
  without a red gate, a waiver, or tests written only to satisfy coverage.
- The two toolchain packs give consumers different coverage semantics for the same problem. A
  polyglot consumer learns the exclusion file in Go and finds it silently ignored in TypeScript.

## Root cause

The exclusion mechanism shipped only in go-toolchain. The TypeScript pack's producer/convert split
was never extended to carry consumer declarations to the sandboxed convert, so the convert cannot
know about them.

## Direction

(Direction, not a plan.) All changes belong in the `backstop-ai/typescript-toolchain` pack. Core
needs no change, because it already reads `excluded` and `justification` from every pack's records.

1. **Producer.** In `scripts/coverage-produce.sh`, read `.backstop/coverage-exclusions` with the
   same grammar and drop rules as go-toolchain, and fold the accepted declarations into the
   relativized artifact. Do this after both the reuse path and the fallback path, so both carry the
   declarations. A reserved key holding `{path, justification}` entries works, written only when at
   least one line is accepted so the artifact is unchanged for projects without the file.
2. **Convert.** In `scripts/coverage-to-records.sh`, re-validate each declaration. Stamp listed
   summary records `excluded: true` with `justification`, and append a synthetic excluded record for
   a declared path with no summary entry, but only when the path has one of this pack's source
   extensions (`.ts`, `.tsx`, `.mts`, `.cts`, `.js`, `.jsx`, `.mjs`, `.cjs`). Skip other paths
   silently (see Notes). Keep the go-toolchain ordering and duplicate rules. Leave unlisted records
   byte-identical.
3. **Proof.** Extend `selftest/run.sh` with these cases, through both the reuse path and the
   fallback path:
   - a justified path yields an excluded record with its reason, and `backstop gate --file` on it
     passes with a `coverage_exclusion` warning that shows the reason;
   - a line with no justification, or with a control character in its path or justification, leaves
     the path measured and enforced.

   Document the file in the README.
4. **Release.** Bump `pack.yml` to `1.3.3` and tag `v1.3.3`. Consumers run `backstop pack update`.

## Notes

- An implementation of this direction is in the pack working copy as 1.3.3 (uncommitted, not
  tagged). `backstop pack check`, `backstop pack test` and `selftest/run.sh` (including its gate
  phase) pass on it.
- Repositories that use both toolchain packs share one `.backstop/coverage-exclusions` file. If
  each pack synthesized a `statement` record for every accepted declaration missing from its own
  inventory, go-toolchain would emit one for a `.ts` path and this pack one for a `.go` path. Either
  synthetic record plus the other pack's real record is a duplicate `(path, metric)`, which core
  reports as `coverage_metric_collision`. (Found by reading the code, not reproduced.) The
  TypeScript side avoids this: it synthesizes only for its own source extensions and skips other
  paths silently, and the selftest checks that a declared `.go` path yields no TypeScript record.
  **Follow-on needed:** go-toolchain still synthesizes for every accepted declaration (ISSUE-186),
  so it needs the mirror change: synthesize only for `.go` paths. Until it ships, a declared `.ts`
  path in a repository using both packs can still produce a Go synthetic record that collides.
- Related: ISSUE-186 (go-toolchain synthetic records for absent declared paths), ISSUE-221
  (`--coverage.include` gives unloaded files `0/N` entries, which is what made this failure
  reachable), ISSUE-222 (same pack, same release line).
