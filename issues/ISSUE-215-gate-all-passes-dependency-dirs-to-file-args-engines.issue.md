---
title: "Gate All Passes Dependency Dirs To File Args Engines"
schema_version: issue/v1

issue:
  id: ISSUE-215
  title: "Gate All Passes Dependency Dirs To File Args Engines"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Gate All Passes Dependency Dirs To File Args Engines

## Problem

On a TypeScript consumer repository (`backstop-ai/typescript-toolchain@1.3.1` plus
`typescript-standards@1.1.1`), `backstop gate --all` fails in the pack engine step:

```
engine "semgrep --sarif --quiet" never started: ... (fork/exec .../semgrep: argument list too long)
```

The `--all` file set handed to the engine contained `node_modules/` (about 16,500 files) against about
141 project files, even though `typescript-toolchain` declares
`classification.dependency_dirs: [node_modules]`. The process image for `semgrep` plus that many path
arguments exceeds the OS `ARG_MAX`, so `exec` fails before the tool runs and the engine produces no
evidence at all.

### What the code does (verified on this branch)

- `classification.dependency_dirs` is consumed only by artifact handling.
  `mergeDependencyDirs` (`cmd/backstop/pack_dependency_dirs.go`) unions the declared names into
  `artifact.NonCorpusDirs` (`pkg/artifact/layout.go`), which is passed to artifact discovery,
  `artifact validate`, requirement-traceability ref collection (`cmd/backstop/gate.go`) and
  `doctor`. Nothing outside the artifact layer reads it.
- The `--all` file set is built by `resolveGateScopeAll` (`pkg/gate/scope.go`). It is a plain
  `filepath.Walk` of the project root that skips only directories whose name begins with `.`. It
  applies neither `dependency_dirs` nor `.gitignore` (no `git ls-files` or `check-ignore`). So
  `node_modules`, `vendor`, build output and any other ignored directory are all in `scope.Files`.
- The scope is constructed in `cmd/backstop/gate.go` (`ComputeGateScopeWithBase`, mode `all`) before
  any pack is consulted, so the scope builder has no access to pack manifests today.
- The rule-fed file-args branch in `cmd/backstop/pack_gate.go` (the `else` branch near the
  "DIRECTORY TARGET AND THE FILE LIST ... NOT EQUIVALENT INPUTS" comment, ISSUE-091) appends
  `excludeTestdataPaths(scope.Files)` straight onto `cmdArgs`. The only filter is the `testdata`
  segment drop (ISSUE-040). There is no cap, chunking, or file-list/stdin/`@file` transport, so
  argument length grows linearly with repository size.

Stated precisely: `dependency_dirs` is NOT applied to the `--all` engine file list, and `.gitignore`
is NOT applied to it either.

### A second, independent hazard

Excluding dependency directories treats the symptom here. Any file-args engine on a sufficiently
large repository can exceed `ARG_MAX` with no dependency directory involved at all (a large monorepo
of project files, long paths, or a small `ARG_MAX` such as macOS). That is a separate defect in how
the file list is transported, and it would remain after dependency directories are excluded.

## Impact

- `gate --all` cannot dispatch any file-args engine on a repository with a large dependency tree.
  The failure is a hard "never started", so the engine yields no findings and no evidence; this is a
  broken run, not a clean pass.
- Even where `ARG_MAX` is not hit, scanning `node_modules` makes `--all` slow and reports findings in
  third-party code the consumer does not own, contradicting the pack's own declaration that this
  directory is dependency content.
- Related, not the same defect:
  - ISSUE-205 (`gate --all` scans gitignored artifacts) concerns the same scope builder ignoring
    `.gitignore`. It is not in this branch's `issues/` directory; it exists only in commit a9ee787 on
    the `review-ledger` branch. Cite it as such until that branch merges. A fix there and a fix here
    would both change `resolveGateScopeAll` and must be coordinated.
  - ISSUE-209 (one engine error discards the rest of the dispatch) is on this branch. The engine
    dispatcher now records each failure and continues (see "CONTINUE PAST AN ENGINE ERROR" in
    `pack_gate.go`), so semgrep's never-start surfaces as an engine-error violation alongside the
    other engines' evidence rather than stopping the dispatch. That makes this defect visible
    instead of masking others, but the semgrep evidence is still lost.

## Root cause

Two gaps, one visible symptom:

1. The `--all` scope is repository-wide and pack-blind. The `dependency_dirs` declaration lives in
   the pack manifest but is only wired into artifact discovery, never into the gate scope that feeds
   engines.
2. The file-args dispatch path transports an unbounded file list as argv, with no length awareness.

## Direction

(Options, not a decision; this is not a plan.)

1. Apply the union of installed packs' `classification.dependency_dirs` to the `--all` file set, so
   those directories are pruned (as the walk already prunes dot-directories). Open choice: filter in
   `resolveGateScopeAll` by threading `mergeDependencyDirs(packs)` into scope construction, or filter
   at engine dispatch. Scope-level filtering also affects non-engine consumers of `scope.Files`
   (coverage, substantiveness, test verdict) and needs checking for unintended narrowing.
2. Optionally apply `.gitignore` to the `--all` set (for example by listing with `git ls-files` plus
   untracked-not-ignored, or `git check-ignore`). This overlaps ISSUE-205 and should be decided once,
   with that issue, not twice.
3. Independently of 1 and 2, make file-args dispatch robust to `ARG_MAX`: batch invocations into
   chunks sized under a conservative argv budget (xargs-style), or pass a file list through a
   pack-declared transport where the tool supports one. Chunking changes engine semantics for
   engines whose findings are not file-independent (cross-file rules), so it needs a per-engine
   declaration or a safe default; merging per-chunk SARIF into one evidence record must preserve the
   ISSUE-209 per-engine failure accounting.

## Notes

- Complexity is `exploratory`: moving `dependency_dirs` into scope touches the scope builder, every
  `scope.Files` consumer, and the pack-manifest contract; chunking touches evidence merging and has
  semantic edge cases. Risk is `moderate` because narrowing the `--all` set can silently reduce
  coverage if the filter is wrong, so it needs a mandated test that a path under a declared
  dependency directory is absent from the engine argv while a project file is present.
- A scale test should exercise argv length directly (a synthetic file list above `ARG_MAX`), not
  only the dependency-directory case, so the two causes stay separately covered.
