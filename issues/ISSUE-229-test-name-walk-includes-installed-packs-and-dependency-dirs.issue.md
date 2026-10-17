---
title: "Test Name Walk Includes Installed Packs And Dependency Dirs"
schema_version: issue/v1

issue:
  id: ISSUE-229
  title: "Test Name Walk Includes Installed Packs And Dependency Dirs"
  type: bug
  status: open
  created: "2026-10-17"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Test Name Walk Includes Installed Packs And Dependency Dirs

## Problem

The walk that collects the project's test names descends into installed packs and dependency
directories, so test titles that belong to third-party content count as the project's own tests.

Verified at origin/main:

- `collectTestFuncNamesScoped` (`pkg/gate/step_testverify.go:634-668`) is a plain `filepath.Walk` of
  `codeDir`. It never returns `filepath.SkipDir` (the file has no `SkipDir`), and its only filter is
  `classifier.IsTestFile(rel)` against the pack-declared test globs. `.backstop/packs/` and any
  dependency directory are walked like project source.
- Two callers use it with the whole project root:
  - the test-verification step, `found := collectTestFuncNames(codeDir, ...)` (`step_testverify.go:511`);
  - `ResolveMandatedTestPaths` (`step_testverify.go:788-789`), which `computeDriftSurfaces`
    (`cmd/backstop/gate.go:~1300`) calls with `projectRoot`, and whose doc comment states it walks "all of
    projectRoot, NOT activeScope".
- `ClassifyStatusDrift` then reports an artifact as "looks delivered" (`pkg/gate/status_drift.go:164`)
  when all its mandated tests are present in that set.

### Reproduction (generic)

A project that installs a git pack whose repository carries its own `tests/` directory. The installer
makes a shallow clone and strips `.git`, so the pack cannot exclude those files. If a test title in that
directory matches a name a spec or plan mandates, the mandated test counts as present although the
project has no such test. Same shape for a test file inside a dependency directory such as `node_modules/`.

## Impact

- The drift advisory reports specs and plans as "looks delivered" on evidence the project does not own.
- Test verification can count a mandated test as found when it only exists in third-party content,
  which weakens the traceability guarantee.
- Not confirmed: whether the test verdict (pass/fail) path is affected; only name existence was traced.
  Also not confirmed: which toolchain packs' test globs already happen to exclude these directories.
  Whether a given project is hit depends on the globs its packs declare.

## Direction

(Direction, not a plan.) Skip `.backstop/packs/` and the union of installed packs' declared
`classification.dependency_dirs` during the walk. `mergeDependencyDirs` already computes that union for
artifact discovery (`cmd/backstop/pack_dependency_dirs.go`, per ISSUE-215 on the engine-evidence branch),
so the walk should take the same set rather than baking directory names, to keep core language-neutral.

## Notes

- Related, not the same: ISSUE-215 (on the unmerged `fix/engine-evidence-integrity` branch) is about the
  `--all` file list handed to engines (ARG_MAX). It touches the same `dependency_dirs` declaration but a
  different code path; its direction does not cover this walk. Whichever lands first should expose the
  dependency-dir set so the other can reuse it.
- Related: ISSUE-188 (closed) fixed which names are extracted per line, not which directories are walked.
