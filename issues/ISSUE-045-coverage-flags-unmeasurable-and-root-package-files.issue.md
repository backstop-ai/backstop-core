---
title: "Coverage Flags Unmeasurable And Root Package Files"
schema_version: issue/v1

issue:
  id: ISSUE-045
  title: "Coverage Flags Unmeasurable And Root Package Files"
  type: bug
  status: open
  created: "2026-07-06"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-045: Coverage Flags Unmeasurable And Root Package Files

## Problem

The gate's `coverage_threshold` step (`pkg/gate/step_coverage.go`) emits
`coverage_unmeasured` for two categories of file that are not genuine coverage
gaps — the same "gate flags the wrong file set" family as the closed
ISSUE-034 (deleted files) and the open ISSUE-040 (testdata scan). Both were
surfaced by ISSUE-027 (moving baked engine defaults into pack data).

Both are currently NEW/blocking (not merely benign residue) because coverage
is diff-scoped and `baseline generate` (full-scope) cannot grandfather a
diff-scoped coverage gap — the known baseline limitation (see
`project_baseline_design` / `project_baseline_ci_pull`). Neither defect is
"missing test coverage" — do not frame either as "add tests."

### 1. Files with zero measurable functions/statements

`pkg/pack/engine/fieldcontract.go` became types + consts only (0 functions,
0 executable statements) after ISSUE-027 moved the field-contract map to pack
data (`packs/base-engines/pack.yml`'s `engines:.<name>.field_contract` /
pack-declared `engines:` blocks). The file still matches the Go toolchain
pack's declared source glob (`**/*.go`, and it is not a `_test.go` file), so
`coveragePathsInScope` (`pkg/gate/step_coverage.go:363`) still treats it as an
"in-scope changed measurable-source file" requiring a coverage record.

But a file with no functions/blocks produces **zero lines** in a Go coverage
profile — confirmed directly:

```
$ go test -coverprofile=cov.out ./pkg/pack/engine/...
ok  	github.com/bmanson/backstop-core/pkg/pack/engine	(cached)	coverage: 95.2% of statements
$ grep -c fieldcontract.go cov.out
0
```

`fieldcontract.go` never enters the profile at all — not measured-and-passing,
not measured-and-failing, simply absent, because `go test -coverprofile`
never emits a block for a file with no statements. `coveragePathsInScope`
has no zero-measurable-content filter, so this genuinely-empty-of-executable-
code file lands in the same coverage-required set as a real gap, and
`resolveCoverageRecordsForPath` (line 298) correctly reports `hasRecords =
false` for it — correctly, given the inputs, but the step then emits
`coverage_unmeasured` for a file that has **nothing to measure**, which is a
false positive, not a real gap. (This mirrors the existing `Total == 0`
below-threshold guard already in this same step — "no executable lines ⇒ N/A,
never a 0%-fail" — but that guard only fires for a file that already HAS a
record; a file with no record at all skips straight past it into the
unconditional-obligation branch.)

### 2. Root-package (`ROOT`) files don't path-normalize

The module-root package (files like `embed.go` at repo root) is tested —
`ListSchemas` in `embed.go` is ~90% covered via `go test .` — yet the gate
reports it as unmeasured. Root cause is more specific than "doesn't
normalize": it's an ambiguous-match collision in the record-lookup fallback,
not a missing-record problem.

Traced end to end:

1. The go-toolchain pack's coverage producer
   (`scripts/coverage-to-records.sh` in the `backstop/go-toolchain` pack)
   reads the raw `go test -coverprofile` profile, which Go always emits with
   the file's **full module import path**, never a repo-relative path —
   confirmed directly:
   ```
   $ go test -coverprofile=cov.out .
   $ head -1 cov.out
   github.com/bmanson/backstop-core/embed.go:31.38,33.93 2 1
   ```
   The convert script does not strip the module prefix; the emitted
   coverage-record's `Path` is the full
   `github.com/bmanson/backstop-core/embed.go`.
2. `indexCoverageByPathMetric` (`pkg/gate/step_coverage.go:270`) keys records
   by `normalizeScopePath("", r.Path)` — which only does `filepath.Clean` +
   `ToSlash`, with **no module-prefix stripping** — so the map key stays the
   full import path for every file, not just root-package ones.
3. `resolveCoverageRecordsForPath` (`pkg/gate/step_coverage.go:298`) therefore
   NEVER gets an exact-match hit for any file (the repo-relative scope path,
   e.g. `pkg/gate/scope.go`, never equals the module-prefixed record key). It
   falls back to a `strings.HasSuffix(recPath, "/"+path)` scan, unique-match
   only. For a nested file like `pkg/gate/scope.go` this fallback is
   effectively safe: the suffix `"/pkg/gate/scope.go"` includes enough path
   segments that no other file in the module is likely to collide.
4. A **root-package** file's scope path is a bare filename with no directory
   component (`embed.go`), so its suffix collapses to `"/embed.go"` — i.e. a
   match by **basename alone**, with no directory qualifier at all. This repo
   has a second, unrelated `embed.go` at `cmd/backstop/embed.go`, so the
   suffix scan finds **two** candidates ending in `/embed.go`:
   ```
   github.com/bmanson/backstop-core/embed.go
   github.com/bmanson/backstop-core/cmd/backstop/embed.go
   ```
   `resolveCoverageRecordsForPath` requires `found == 1` to return a match
   (line 311); with `found == 2` it returns `(nil, false)` — ambiguous,
   discarded — even though the root `embed.go` record is real and present.
   `coveragePathsInScope` then reports `hasRecords = false` and the step
   emits `coverage_unmeasured` for a file that IS measured.

Root-package files are the only files whose gate-scope path has zero
directory segments, so they are the only files for which the suffix-fallback
degenerates all the way to a basename-only match — making them uniquely
exposed to any other identically-named file anywhere else in the module
(confirmed this repo alone has several duplicate basenames outside testdata:
`manifest.go` x3, `scope.go`/`scaffold.go`/`gate.go`/`embed.go`/`doc.go`/
`baseline.go` x2 each — any of these would collide the same way if it were a
root-package file). The real defect is one level up from "doesn't
normalize": the record index is never normalized to a true repo-relative
path at all (module prefix is never stripped), so exact matching never
engages and every single file's match depends on a suffix heuristic that is
merely *usually* unambiguous by accident of having enough path segments.

### Why it matters

Same enforcement-philosophy violation as ISSUE-034/040: the gate goes loud on
something that is not a defect (CLAUDE.md, "Loud ≠ blocking"), which — left
unfixed — forces spurious waivers or blocks otherwise-legitimate changes:
(1) any future pack-data migration that empties a `.go` file down to
types/consts (the exact shape of the thin-executor eradication work this repo
is mid-way through, e.g. ISSUE-018, ISSUE-027, ISSUE-030) will re-trigger
case 1; (2) any change touching root-package files (`embed.go`,
`root.go`-style CLI entry files common in Go modules) risks tripping case 2
whenever another package happens to reuse the same basename — a latent,
data-dependent flake, not a stable false positive, which makes it worse: it
can appear or disappear as unrelated files are added elsewhere in the tree.

## Solution (fix directions to evaluate — not committed)

**Case 1 — zero-measurable-content files:**
Exclude files with zero measurable functions/statements from the
coverage-required set built by `coveragePathsInScope`. A source file that
produces no coverage-profile entries for ANY reason tied to having nothing to
measure is not "unmeasured," it is un-measurable-by-construction, and should
be treated the same as the existing `Total == 0` N/A rule for a file that
already has a record — extended to cover the zero-record case up front. The
planner should weigh:
- Deriving "no measurable statements" from the coverage producer's silence
  (i.e., treat a source-classified file with zero records from ANY pack's
  coverage engine as N/A rather than unmeasured) — general, but must not
  swallow a genuinely-untested file that DOES have functions (the regression
  ISSUE-034's fix had to guard against for deletions applies here too).
- A pack-declared or gate-local static check (e.g. a lightweight
  function-count parse) as a second signal, only trusted in conjunction with
  the zero-records observation, to distinguish "empty file" from "untested
  file whose test run failed to produce a profile at all" (a real gap that
  must still be flagged).

**Case 2 — root-package path normalization:**
Normalize coverage-record paths to true repo-relative paths at ingestion,
so exact matching works for every file (root-package or not) and the
suffix-based fallback stops being the primary match path:
- Strip the Go module import-path prefix from each record's `Path` before or
  during `indexCoverageByPathMetric` / the all-mode branch of
  `coveragePathsInScope` — requires resolving the module path (e.g. from
  `go.mod` at `scope.ProjectRoot`) somewhere in the language-neutral gate, OR
  moving the stripping into the go-toolchain pack's
  `scripts/coverage-to-records.sh` convert step so the coverage-records
  channel is contractually repo-relative for every producer, not just Go's
  (this keeps the language-neutral consumer, `pkg/gate`, free of any
  Go-module-path knowledge — consistent with the thin-executor rule that
  language-specific normalization belongs in the pack, not the binary).
- Once records are truly repo-relative, decide whether the `HasSuffix`
  fallback in `resolveCoverageRecordsForPath` should be narrowed (e.g.
  require at least one path separator in the suffix, so a bare-basename
  match can never fire) or removed entirely as dead code if exact matching
  now always succeeds for every producer that emits repo-relative paths.

**Tests to add (both cases, non-stubbed, exercising the real
`StepCoverageThresholdScopedFunc`):**
- A changed measurable-source file with zero functions/statements (a
  `fieldcontract.go`-shaped fixture: package decl + type/const only) produces
  no `coverage_unmeasured` violation.
- A changed measurable-source file that DOES have functions but genuinely has
  no coverage record still produces the violation (regression guard,
  mirroring ISSUE-034's CLM-002 pattern).
- A root-package file (module-prefixed record path with zero directory
  segments after the module prefix) that IS measured produces no
  `coverage_unmeasured` violation, including in a fixture where a
  same-basename file exists elsewhere in the tree (reproducing the
  `embed.go` / `cmd/backstop/embed.go` collision directly, so the fix is
  proven against the exact ambiguity that causes the false positive, not
  just the simpler case with no collision).
- A root-package file that is genuinely unmeasured (no record at all) still
  produces the violation (regression guard).

**Acceptance:** a change touching only `fieldcontract.go`-shaped
(zero-statement) files produces zero `coverage_unmeasured` violations for
them; a change touching a measured root-package file (with or without a
same-basename collision elsewhere in the module) produces zero
`coverage_unmeasured` violations for it; both companion "still genuinely
unmeasured" cases keep firing (no blinding of the real check).

## References

- `pkg/gate/step_coverage.go:363` (`coveragePathsInScope`) — builds the
  coverage-required set from `scope.Files` filtered only by
  `classifier.IsMeasurableSource`; no zero-statement filter
- `pkg/gate/step_coverage.go:270` (`indexCoverageByPathMetric`) — keys
  records by `normalizeScopePath("", r.Path)`, which never strips a Go module
  import-path prefix, so the map key is never a true repo-relative path
- `pkg/gate/step_coverage.go:298` (`resolveCoverageRecordsForPath`) — exact
  match never hits (per above); the `strings.HasSuffix` fallback requires
  `found == 1`, and a root-package scope path's bare-basename suffix collides
  with `cmd/backstop/embed.go` for the `embed.go` case (`found == 2`, no
  match returned)
- `pkg/gate/scope.go:91` (`normalizeScopePath`) — `filepath.Clean` +
  `ToSlash` only; no module-path knowledge, by design (language-neutral gate)
- `pkg/pack/engine/fieldcontract.go` — the concrete zero-function fixture:
  post-ISSUE-027, types + consts only, 0 lines matched in a real
  `-coverprofile` run
- `embed.go` (repo root) / `cmd/backstop/embed.go` — the concrete
  same-basename collision that breaks case 2's suffix fallback
- go-toolchain pack (`backstop/go-toolchain`,
  `scripts/coverage-to-records.sh`) — the coverage producer; emits Go's raw
  module-prefixed profile paths unmodified, confirmed via direct
  `go test -coverprofile` runs against this repo's root and nested packages
- DIR-015 (Gate Checker Hardening) — parent directive covering gate-checker
  correctness hardening; this issue is scoped work under it
- ISSUE-034 (gate-coverage-flags-deleted-files, closed) — sibling in the same
  family: coverage asserting a positive per-path obligation off `scope.Files`
  without enough filtering, there for deletions, here for zero-content and
  root-package files; this issue's fixes should follow the same
  coverage-local, narrowly-scoped, non-stubbed-test pattern
- ISSUE-040 (gate-substantiveness-scans-testdata-fixtures, open) — sibling
  family: the gate scanning/measuring the wrong file set in a different
  dimension (`test_substantiveness` vs `coverage_threshold`)
- ISSUE-027 (eradicate-default-registry-into-packs, open) — the change that
  emptied `fieldcontract.go` down to types/consts and surfaced case 1; also
  the general shape of thin-executor migrations (pack-data extraction) that
  will keep re-triggering case 1 until fixed
- Project memory `gate_scope_and_coverage` — prior coverage/scope hardening
  precedent this issue continues
